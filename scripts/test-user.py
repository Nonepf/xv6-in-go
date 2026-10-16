#!/usr/bin/env python3
"""在 QEMU 中运行真正的用户程序，并检查退出和物理页回收。只用 Python 标准库。"""

import json
import os
from pathlib import Path
import re
import shlex
import subprocess
import time


ROOT = Path(__file__).resolve().parent.parent
CASES = (
    ("hello", "kernel/initcode.S", "hello from user\n", None),
    ("syscalls", "tests/syscalls.S", "syscalls: OK\n", None),
    ("timer", "tests/timer.S", "timer: OK\n", None),
    ("illegal", "tests/illegal.S", "Usertrap: pid 1, scause 0x2,", 2),
    ("pagefault", "tests/pagefault.S", "Usertrap: pid 1, scause 0xd,", 13),
)


class QMP:
    def __init__(self, proc):
        self.proc = proc
        greeting = json.loads(proc.stdout.readline())
        if "QMP" not in greeting:
            raise RuntimeError(f"QEMU 未返回 QMP greeting: {greeting}")
        self.command("qmp_capabilities")

    def command(self, name, arguments=None):
        request = {"execute": name}
        if arguments is not None:
            request["arguments"] = arguments
        self.proc.stdin.write(json.dumps(request) + "\n")
        self.proc.stdin.flush()
        while True:
            line = self.proc.stdout.readline()
            if not line:
                raise RuntimeError("QEMU 提前退出")
            response = json.loads(line)
            if "error" in response:
                raise RuntimeError(str(response["error"]))
            if "return" in response:
                return response["return"]

    def read64(self, address):
        result = self.command("human-monitor-command", {
            "command-line": f"xp /1gx 0x{address:x}"
        })
        match = re.search(r":\s*0x([0-9a-fA-F]+)", result)
        if match is None:
            raise RuntimeError(f"无法读取 guest 内存: {result}")
        return int(match.group(1), 16)


def symbols(kernel):
    output = subprocess.check_output(
        shlex.split(os.environ.get("NM", "nm")) + ["-n", str(kernel)], text=True
    )
    return {parts[2]: int(parts[0], 16) for line in output.splitlines()
            if len(parts := line.split()) == 3}


def check_exit(qmp, names, output):
    # Proc.lock 占 4 字节，RV64 的 state 对齐到偏移 8。
    assert qmp.read64(names["main.current_proc"]) == 0, "退出后仍有当前进程"
    assert qmp.read64(names["main.proc"] + 8) == 0, "进程槽没有回到 UNUSED"

    # trapframe 是 userinit 分配的第一张页。回收后，空闲链表前 32 页应
    # 与分配前相同（顺序可以不同）；这样能发现页泄漏和重复释放。
    trapframe = int(re.search(r"mappages 0x3fffffe000, (0x[0-9a-f]+)", output)[1], 16)
    pa = qmp.read64(names["main.kmem"])
    pages = []
    for _ in range(32):
        pages.append(pa)
        pa = qmp.read64(pa)
    expected = {trapframe - i * 4096 for i in range(32)}
    assert set(pages) == expected and len(set(pages)) == 32, "物理页未完整回收"
    assert pa == trapframe - 32 * 4096, "空闲链表被破坏"


def run_case(name, source, marker, fault):
    directory = ROOT / "build" / "tests" / name
    directory.mkdir(parents=True, exist_ok=True)
    kernel = directory / "kernel-qemu"
    build = subprocess.run([
        "make", "--no-print-directory", f"BUILD_DIR={directory}",
        f"TARGET={kernel}", f"INITCODE={source}", "all"
    ], cwd=ROOT, text=True, stdout=subprocess.PIPE, stderr=subprocess.STDOUT)
    if build.returncode:
        raise RuntimeError(build.stdout)

    serial = directory / "serial.log"
    trace = directory / "traps.log"
    serial.write_text("")
    command = shlex.split(os.environ.get("QEMU", "qemu-system-riscv64")) + [
        "-machine", "virt", "-bios", "none", "-kernel", str(kernel),
        "-m", "128M", "-display", "none", "-monitor", "none",
        "-serial", f"file:{serial}", "-qmp", "stdio",
        "-d", "int", "-D", str(trace)
    ]
    with (directory / "qemu.log").open("w") as errors:
        proc = subprocess.Popen(command, cwd=ROOT, stdin=subprocess.PIPE,
                                stdout=subprocess.PIPE, stderr=errors, text=True)
        try:
            qmp = QMP(proc)
            deadline = time.monotonic() + 10
            while marker not in (output := serial.read_text(errors="replace")):
                if proc.poll() is not None or time.monotonic() > deadline:
                    raise AssertionError(f"{name}: 未看到预期输出\n{output}")
                if "panic" in output or "FAIL" in output or "Kerneltrap " in output:
                    raise AssertionError(f"{name}: 内核或用户程序失败\n{output}")
                time.sleep(0.02)
            # 留出多个时钟周期，确认 exit 后不会重新执行用户程序。
            time.sleep(0.2)
            qmp.command("stop")
            output = serial.read_text(errors="replace")
            assert output.count(marker) == 1, f"{name}: 用户程序被重复执行"
            assert "panic" not in output and "FAIL" not in output and "Kerneltrap " not in output, output
            assert ("Usertrap:" in output) == (fault is not None), output
            check_exit(qmp, symbols(kernel), output)

            interrupts = trace.read_text()
            events = re.findall(
                r"async:(\d), cause:([0-9a-f]+), epc:0x([0-9a-f]+)", interrupts
            )
            exceptions = [int(cause, 16) for asynchronous, cause, _ in events
                          if asynchronous == "0"]
            if fault is not None:
                assert exceptions == [fault], f"{name}: 异常链路不正确: {exceptions}"
            else:
                assert exceptions and all(cause == 8 for cause in exceptions), exceptions
                if name in ("hello", "timer"):
                    assert len(exceptions) == 2, "应只有 write 和 exit 两次 ecall"
            user_ticks = sum(asynchronous == "1" and int(cause, 16) == 1
                             and int(epc, 16) < 4096
                             for asynchronous, cause, epc in events)
            if name == "timer":
                assert user_ticks > 0, "没有真正发生用户态时钟抢占"
            qmp.command("quit")
            proc.wait(timeout=5)
            suffix = f" ({user_ticks} user timer interrupts)" if name == "timer" else ""
            print(f"PASS {name}: user trap, exit and page reclamation{suffix}", flush=True)
        finally:
            if proc.poll() is None:
                proc.kill()
                proc.wait(timeout=5)


if __name__ == "__main__":
    for case in CASES:
        run_case(*case)
    print("All 5 QEMU tests passed.")
