# 实验记录
## Log 01:  Hello World
### 实验目标
- 实现Kernel从启动到输出 `"hello, world"` 的整个基本流程
### 设计方案与难点
原初的大致方案是先禁用TinyGo的GC与调度器，期望能暂时与C语言一样编译。但无论怎么设置，Go的runtime还是需要一些基本的调用接口支撑，导致报错不断。因此，后面改为下面的设计：

- `entry.S`：设置基本的栈空间，然后进入`init.c` 
- `init.c`：实现基本的runtime调用以及基本输出接口`uart_putc`，并过渡到`main.go`
- `main.go`：主程序，输出结果
- `kernel.ld`：指定链接的布局

### 实现细节

`entry.S`
```asm
.section .text
.global _entry
_entry:
    la sp, stack0       # sp pointer
    li a0, 4096
    add sp, sp, a0      # set sp to its starting point
    call c_start

.section .bss
.align 16
stack0: .space 4096     # spare a 4096-byte room
```
`entry.S` 首先空出`4096b` 大小的空间作为栈，然后将栈指针设置成开始值，接着调用`init.c` 中的函数.

而 `c_start` 本身很简单，再次调用了`main.go` 中的`KMain`.
```c
void c_start() {
    extern void KMain();
    uart_putc('C'); uart_putc(' '); uart_putc('O'); uart_putc('K'); uart_putc('\n');
    KMain();
}
```

不过，`init.c`还实现了以下这些函数，让Go的运行时误以为自己运行在一个真的操作系统之上
```c
void uart_putc(char c);

static uint8_t go_heap[64 * 1024];
uintptr_t runtime_heapMaxSize = (uintptr_t)sizeof(go_heap);

void* mmap(void* addr, size_t length, int prot, int flags, int fd, int64_t offset);
void runtime_exit(int code);
void runtime_abort();
```

紧接着便是`Go`部分的主代码，通过循环输出字符串，此处不展开.

### 实验结果
效果如下：

	rpfLAPTOP-50JMISE7:~/xv6-in-go$ make qemu
	tinygo build -target=./riscv64-bare.json -o build/main.o ./kernel/main.go
	riscv64-unknown-elf-ld -T kernel/kernel.ld -o kernel-qemu build/entry.o build/main.o build/init.o \
	        --gc-sections \
	        --allow-multiple-definition
	qemu-system-riscv64 -machine virt -bios none -kernel kernel-qemu -m 128M -nographic
	C OK
	RPF: Hello, world!
