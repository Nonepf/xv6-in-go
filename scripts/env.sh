#!/usr/bin/env bash
# 把 .toolchain/ 里的 Go / TinyGo / QEMU 加入当前 shell 环境。
# 用法（必须 source，不能直接执行）：
#     source scripts/env.sh
#     make && make qemu
#
# 之后 `make`、`make qemu`、`make debug` 都能直接用。

_xv6go_root="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")/.." && pwd)"
TOOLCHAIN_DIR="${TOOLCHAIN_DIR:-$_xv6go_root/.toolchain}"

if [ ! -d "$TOOLCHAIN_DIR" ]; then
    echo "未找到 $TOOLCHAIN_DIR，请先运行 scripts/bootstrap-toolchain.sh" >&2
    return 1 2>/dev/null || exit 1
fi

export TOOLCHAIN_DIR
export PATH="$TOOLCHAIN_DIR/go/bin:$TOOLCHAIN_DIR/tinygo/bin:$TOOLCHAIN_DIR/qemu/usr/bin:$PATH"

# 把 Go/TinyGo 缓存也放在仓库内，避免污染 ~/.cache（沙箱/只读 HOME 下也能用）
export XDG_CACHE_HOME="$TOOLCHAIN_DIR/cache"
export GOCACHE="$TOOLCHAIN_DIR/cache/go-build"
export GOPATH="$TOOLCHAIN_DIR/cache/gopath"

# 交叉编译器：优先 bare-metal 工具链，否则退回发行版的 linux-gnu 工具链
if command -v riscv64-unknown-elf-gcc >/dev/null 2>&1; then
    export CC=riscv64-unknown-elf-gcc
    export LD=riscv64-unknown-elf-ld
elif command -v riscv64-linux-gnu-gcc >/dev/null 2>&1; then
    export CC=riscv64-linux-gnu-gcc
    export LD=riscv64-linux-gnu-ld
fi
export TINYGO=tinygo
export QEMU="${QEMU:-qemu-system-riscv64}"

echo "xv6-in-go 环境就绪:"
echo "  go      : $(go version 2>/dev/null || echo 缺失)"
echo "  tinygo  : $(tinygo version 2>/dev/null | head -1 || echo 缺失)"
echo "  cc      : ${CC:-缺失}"
echo "  qemu    : $QEMU"
