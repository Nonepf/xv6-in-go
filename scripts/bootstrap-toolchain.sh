#!/usr/bin/env bash
# 无 root 下载并安装本项目所需的工具链到 <repo>/.toolchain
# 版本与 2026-01 最初开发时一致：Go 1.22.10 + TinyGo 0.31.2
# 用法： scripts/bootstrap-toolchain.sh
set -euo pipefail

GO_VERSION=1.22.10
TINYGO_VERSION=0.31.2
TINYGO_ARCH="linux-amd64"

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TOOLCHAIN_DIR="${TOOLCHAIN_DIR:-$ROOT/.toolchain}"
mkdir -p "$TOOLCHAIN_DIR"
cd "$TOOLCHAIN_DIR"

echo "==> 工具链目录: $TOOLCHAIN_DIR"

# 1. Go 1.22.10（TinyGo 需要 go 命令做包解析）
if [ ! -x go/bin/go ] && [ ! -x "$TOOLCHAIN_DIR/go/bin/go" ]; then
    echo "==> 下载 Go ${GO_VERSION}"
    curl -fL -o go.tar.gz "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
    tar xzf go.tar.gz
    rm -f go.tar.gz
else
    echo "==> Go 已存在，跳过"
fi
"$TOOLCHAIN_DIR/go/bin/go" version

# 2. TinyGo 0.31.2（真正编译 kernel/*.go 的编译器）
if [ ! -x tinygo/bin/tinygo ]; then
    echo "==> 下载 TinyGo ${TINYGO_VERSION}"
    curl -fL -o tinygo.tar.gz \
        "https://github.com/tinygo-org/tinygo/releases/download/v${TINYGO_VERSION}/tinygo${TINYGO_VERSION}.${TINYGO_ARCH}.tar.gz"
    tar xzf tinygo.tar.gz
    rm -f tinygo.tar.gz
else
    echo "==> TinyGo 已存在，跳过"
fi
PATH="$TOOLCHAIN_DIR/go/bin:$TOOLCHAIN_DIR/tinygo/bin:$PATH" \
    XDG_CACHE_HOME="$TOOLCHAIN_DIR/cache" tinygo version

# 3. QEMU（只需要 riscv 系统模拟器）
if [ -x "$TOOLCHAIN_DIR/qemu/usr/bin/qemu-system-riscv64" ]; then
    echo "==> QEMU 已存在，跳过"
elif command -v qemu-system-riscv64 >/dev/null 2>&1; then
    echo "==> 系统已安装 qemu-system-riscv64，跳过"
else
    echo "==> 通过 apt 下载 QEMU（不需要 root）"
    # 注意：较新的 Ubuntu 把 riscv 模拟器单独放在 qemu-system-riscv，
    # 旧版本则包含在 qemu-system-misc 中。
    apt-get download qemu-system-riscv 2>/dev/null || apt-get download qemu-system-misc
    mkdir -p qemu
    for deb in qemu-system-*.deb; do
        dpkg -x "$deb" qemu
        rm -f "$deb"
    done
fi
QEMU_BIN="$TOOLCHAIN_DIR/qemu/usr/bin/qemu-system-riscv64"
if [ -x "$QEMU_BIN" ]; then
    "$QEMU_BIN" --version | head -1
else
    command -v qemu-system-riscv64 >/dev/null 2>&1 && qemu-system-riscv64 --version | head -1
fi

# 4. RISC-V 交叉编译器（只用来编译 .c/.S）
if command -v riscv64-unknown-elf-gcc >/dev/null 2>&1; then
    echo "==> 交叉编译器: riscv64-unknown-elf-gcc（与原始开发环境一致）"
elif command -v riscv64-linux-gnu-gcc >/dev/null 2>&1; then
    echo "==> 未找到 riscv64-unknown-elf-gcc，将退回 riscv64-linux-gnu-gcc"
    echo "    （本项目只用它编译 -c，链接由 riscv64-linux-gnu-ld 直接完成，实测可用）"
    echo "    如需原版： sudo apt install gcc-riscv64-unknown-elf"
else
    echo "!! 缺少 RISC-V 交叉编译器，请安装： sudo apt install gcc-riscv64-unknown-elf" >&2
fi

# 5. 调试器（可选）
if command -v gdb-multiarch >/dev/null 2>&1; then
    echo "==> gdb-multiarch 可用"
else
    echo "!! 未找到 gdb-multiarch（可选）： sudo apt install gdb-multiarch"
fi

echo
echo "完成。接下来："
echo "    source scripts/env.sh"
echo "    make && make qemu"
