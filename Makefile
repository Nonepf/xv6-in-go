# tool chains
TINYGO = tinygo
CC = riscv64-unknown-elf-gcc
LD = riscv64-unknown-elf-ld

# target
TARGET = kernel-qemu
BUILD_DIR = build

# settings
CFLAGS = -march=rv64imac -mabi=lp64 -mcmodel=medany -ffreestanding -nostdlib -O2
TINYGO_FLAGS = -target=./kernel/riscv64-bare.json

# object files
OBJS = $(BUILD_DIR)/entry.o \
       $(BUILD_DIR)/main.o	\
       $(BUILD_DIR)/init.o

# Rules for Building
all: $(TARGET)

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

# entry
$(BUILD_DIR)/entry.o: kernel/entry.S | $(BUILD_DIR)
	$(CC) $(CFLAGS) -c $< -o $@

# compile init.c
$(BUILD_DIR)/init.o: kernel/init.c | $(BUILD_DIR)
	$(CC) $(CFLAGS) -c $< -o $@

# compile Go kernel
$(BUILD_DIR)/main.o: kernel/main.go | $(BUILD_DIR)
	$(TINYGO) build $(TINYGO_FLAGS) -o $@ ./kernel/main.go

# link
$(TARGET): $(OBJS) kernel/kernel.ld
	$(LD) -T kernel/kernel.ld -o $(TARGET) $(OBJS) \
		--gc-sections \
		--allow-multiple-definition

# simulate
qemu: $(TARGET)
	qemu-system-riscv64 -machine virt -bios none -kernel $(TARGET) -m 128M -nographic

# clean
clean:
	rm -rf $(BUILD_DIR) $(TARGET)

.PHONY: all clean qemu