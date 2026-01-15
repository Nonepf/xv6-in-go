# tool chains
TINYGO = tinygo
CC = riscv64-unknown-elf-gcc
LD = riscv64-unknown-elf-ld

# target
TARGET = kernel-qemu
BUILD_DIR = build

# settings
# -march=rv64imac_zicsr: 包含原子指令(a)和控制状态寄存器(zicsr)，这对时钟中断至关重要
CFLAGS = -march=rv64imac_zicsr -mabi=lp64 -mcmodel=medany -ffreestanding -nostdlib -O2 -g
TINYGO_FLAGS = -target=./scripts/riscv64-bare.json

# --- 核心改进区 ---

# 1. 自动寻找 src 目录下所有的 .c 文件 (用于后续时钟中断等 trap.c 文件)
C_SOURCES = $(wildcard src/*.c)

# 2. 将 src/*.c 转换为 build/*.o
DYNAMIC_C_OBJS = $(patsubst src/%.c, $(BUILD_DIR)/%.o, $(C_SOURCES))

# 3. 汇总所有 Object 文件：
# 保留原有的 entry.o 和 main.o，并自动加入后续新增的所有 C 模块
OBJS = $(BUILD_DIR)/entry.o \
       $(BUILD_DIR)/main.o  \
       $(DYNAMIC_C_OBJS)

# --- 编译规则 ---

all: $(TARGET)

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# entry: 引导汇编，通常位置比较固定
$(BUILD_DIR)/entry.o: kernel/entry.S | $(BUILD_DIR)
	$(CC) $(CFLAGS) -c $< -o $@

# compile c files: 统一处理 src/ 下的所有 C 源文件
$(BUILD_DIR)/%.o: src/%.c | $(BUILD_DIR)
	$(CC) $(CFLAGS) -c $< -o $@

# compile Go kernel: FORCE 确保 TinyGo 内部增量编译逻辑生效
$(BUILD_DIR)/main.o: FORCE | $(BUILD_DIR)
	$(TINYGO) build $(TINYGO_FLAGS) -o $@ ./kernel/

.PHONY: FORCE

# link: 使用汇总后的 OBJS，并保留你的链接器参数
$(TARGET): $(OBJS) scripts/kernel.ld
	$(LD) -T scripts/kernel.ld -o $(TARGET) $(OBJS) \
		--gc-sections \
		--allow-multiple-definition

# --- 其他辅助命令 ---

# simulate
qemu: $(TARGET)
	qemu-system-riscv64 -machine virt -bios none -kernel $(TARGET) -m 128M -nographic

# clean
clean:
	rm -rf $(BUILD_DIR) $(TARGET)

.PHONY: all clean qemu