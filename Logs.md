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

## Log 02: printf
### 实验目标
- 实现一个符合直觉的printf函数

### 设计方案与难点
我们暂且只实现 `%d`, `%s`, `%c`三种格式化输出。

为此，我们需要先实现 `printInt`, `printString`两个函数，接着实现`printf`，综合上述结果。

而`printf`涉及到不定长的参数传入，这里需要导入`runtime`库，使用`interface{}`. 在实际操作中，使用这些东西会默认调用操作系统的`abort`, `write`接口，我们又需要在`init.c`中进行伪造。

### 实现细节
`init.c` 新增接口如下：
```c
void abort();
ssize_t write(int fd, const void *buf, size_t count);
```

我们另外创建了一个`printf.go`文件，实现了以下函数（实现方法很简单，就是一些简单的字符处理）。注意到，在打印字符时，如果单纯使用`args[argIdx].(int)`来提取，会造成断言错误。
```go
func printInt(num int)
func printString(str string)
func printf(format string, args ...interface{}) 
```

### 实验效果
我们的主函数：
```go
func KMain() {
	msg := "RPF: Hello, world!"
	for _, c := range msg {
		uart_putc(byte(c))
	}
	uart_putc('\n')
	printInt(2147483647)
	uart_putc('\n')
	printString("Hello there")
	uart_putc('\n')
	printf("Today is %s \n, %c %d %d %x", "Monday", 'M', 9, 100)
    for {}
}
```

对应输出：

	C OK
	RPF: Hello, world!
	2147483647
	Hello there
	Today is Monday
	, M 9 100 %x

## Log 03: kalloc
### 实验目标
- 使用分页管理物理内存

### 设计思路与细节
这次新增功能高度参考了xv6系统的 `kalloc.c` 文件。

在具体实施之前，我考虑过能否直接用Go内部的内存管理来实现。但这是一个我并不熟悉的道路，而且与xv6的设计相距有点远，因此还是选择了我最熟悉的这条道路。

具体实施时，除去Go语言语法上的一些问题，最麻烦的是如何让Go读取到`kernel.ld`中的变量。

曾经尝试过这样的方法：
```go
//go:linkname bottom _bss_end
var bottom uintptr
```
不过似乎并不能正确地读取到该字段，并在反复修改`kernel.ld`后无果，遂采用临时方案：
```go
const (
    BSS_END   = uintptr(0x80021000)
    PHYSTOP   = uintptr(0x88000000)
)
```
现在暂时先这样，后面会考虑替换掉。

### 其他细节

- 之前我们的`printfInt`函数有误，需要判断数是否为`0`.

- `kalloc.go`实现的函数如下：
```go
func kinit()
func freerange(pa_start uintptr, pa_end uintptr)
func kalloc() uintptr
func kfree(pa uintptr)
```

### 实验效果
我将 `main.go` 文件分成了几个测试板块，方便模块化测试，如下：
```go
func KMain() {
	printfTest()
	kallocTest()
    for {}
}

func printfTest() {
	printf("--- printf test ---\n")
	printInt(2147483647)
	uart_putc('\n')
	printString("Hello there")
	uart_putc('\n')
	t := 1
	printf("Today is %s \n, %c %d %d %x\n", "Monday", 'M', t, 2)
}

func kallocTest() {
	printf("--- kalloc test ---\n")

	printf("init kmem... ")
	kinit()
	printf("OK\n")

	printf("test kalloc\n")
	count := 0
	for kalloc() != 0 {
		count++
	}
	printf("allocate %d KB memory\n", int(count * 4))

}
```

输出如下：

	C OK
	--- printf test ---
	2147483647
	Hello there
	Today is Monday
	, M 1 2 %x
	--- kalloc test ---
	init kmem... kinit: [2147618816, 2281701376)
	freerange: [2147618816, 2281701376)
	OK
	test kalloc
	allocate 130940 KB memory

### 后续完善
- 完成自旋锁机制
- 替换当前的`bottom`, `top`设计，改用更加结构化的设计