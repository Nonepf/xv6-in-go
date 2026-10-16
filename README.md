# xv6 in Go

## 项目背景

在学完 MIT 6.S081 的课程后，总想自己从头到尾实现一个操作系统，加深自己对系统的理解。本来最初打算在 x64-86 平台上复刻 xv6，不过似乎有点缺乏挑战性。又由于受到 [Biscuit](https://pdos.csail.mit.edu/6.828/2020/readings/biscuit.pdf) 这个系统设计想法的启发，于是打算使用Go语言来复现一下。

期末周期间，我简单地学习了一下Go语言的语法与特色功能，打算在寒假期间把一个基本的系统框架弄出来。受制于寒假较短的时间跨度，加之还有很多其他事务需要处理，因此我的实现比xv6还要简化，目前只考虑运行在一个单核CPU上，不涉及多核管理。

(2026-10-16 补充) 最近考虑用 AI 把缺漏的地方补上。

## 整体规划
我打算分成多个步骤完成此系统

- [x] 基础I/O: 实现基本的字符输出
- [x] 物理内存管理: 使用分页来管理物理内存区域
- [x] 虚拟内存: 实现页表，开启CPU的MMU功能，让内核运行在虚拟地址中
- [x] 中断与异常
- [x] 锁机制
- [x] 内核多进程管理
- [x] 用户态：进入、陷入返回与时钟抢占
- [x] 最小系统调用：串口 `write`、`exit`，以及退出后的用户内存回收
- [ ] 用户多进程管理
- [ ] 简单用户程序（`sh`，`echo`）
- [ ] 文件系统

这次以 `hello from user` 作为阶段性的收尾：用户程序通过 `ecall` 请求内核打印，再回到用户态执行 `exit`，由调度器回收资源。用户多进程、shell 和文件系统留作后续方向。

## 快速开始

工具链：Go 1.22.10 + TinyGo 0.31.2 + `riscv64-unknown-elf-gcc` + `qemu-system-riscv64`。

```bash
scripts/bootstrap-toolchain.sh   # 首次：把工具链装到 .toolchain/（无需 root）
source scripts/env.sh            # 每个 shell 一次
make && make qemu                # 构建并运行
make debug                       # QEMU 等待 gdb 连接
make test                        # 运行 5 项 QEMU 用户态检查（需要 Python 3 和 nm）
```

启动输出的最后几行：

```text
OK
jump to scheduler...
hello from user
```

用户程序退出后，内核留在调度器中等待；QEMU 不会自动关闭，按 `Ctrl-A X` 退出。

## 补充

整个系统的实现记录在以下这些文件之中。
- [Logs_1.md](logs/Logs_1.md) 
- [Logs_2.md](logs/Logs_2.md)
- [Logs_3.md](logs/Logs_3.md)
- [Logs_4.md](logs/Logs_4.md)
- [Logs_5.md — hello from user](logs/Logs_5.md)
