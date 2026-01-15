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

---
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

---
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

---
## Log 04: memorylayout
### 主要内容
- 将 xv6 中的`memlayout.h` 迁移到了Go
- 改写了对END的硬编码

### 实现细节
- 所有地址用 `uintptr` 类型，但注意Go没有隐式类型转换，在定义一些运算时需小心
- `const`定义的数值不占用实际空间，和C的宏定义很像
- 尽量不要在全局就给变量赋值（如本次修改`kalloc.go`时发现在全局给`BSS_END`赋值会导致其值为`0`）

---
## Log 05: page tables
### 实验目标
- 简单实现操作系统的三级页表机制

### 实验设计
这个阶段我们只负责页表的初始化与创建，因此只实现下面这几个函数

| 函数            | 作用                  |
| ------------- | ------------------- |
| `walk`        | 模拟MMU遍历页表，为后面操作的基石  |
| `mappages`    | 将一段连续的虚拟地址映射到一段物理地址 |
| `kvmmap`      | 内核页表的初始化            |
| `kvminit`     | 全局初始化               |
| `kvminithart` | 开启MMU               |

### 实现细节
为了方便调试，我为`printf`函数设计了`%x`格式的输出，如下：
```go
func printHex(val uintptr) {
	if val == 0 {
		uart_putc('0')
		return
	}

	var buf [16]byte
	i := 15
	chars := "0123456789abcdef"

	for val > 0 && i >= 0 {
		buf[i] = chars[val%16]
		val /= 16
		i--
	}

	uart_putc('0')
	uart_putc('x')
	for j := i + 1; j < 16; j++ {
		uart_putc(buf[j])
	}
}
```

在实际操作了，我还定义了大量的类似xv6中宏的函数：
```go
func PX(level int, va uintptr) uintptr { return (va >> (12 + uintptr(level)*9)) & 0x1FF }
func PTE2PA(pte pte_t) uintptr { return (uintptr(pte) >> 10) << 12 }
func PA2PTE(pa uintptr) pte_t { return pte_t((pa >> 12) << 10) }

func PGGROUNDDOWN(a uintptr) uintptr { return a & ^(PGSIZE - 1) }
```

以及内存管理的辅助函数：
```go
func memset(dst uintptr, c int, n uint)
```

至于关键函数`kvminithart`与`walk`的实现，见下一小节。

### 实现难点
主要的一个难点（或者说需要斟酌的点）是如何权衡硬件，C与Go。

在实现`kvminithart`时，发现Go不能直接使用内联汇编，需要在`init.c`中代替其实现。但`init.c`又需要Go代码中`kernel_pagetable`的具体值，我尝试过在`vm.go`中使用`\\export`，不过没有效果，最后还是在C中定义了内核页表，在`vm.go`中使用它。

另外，Go语言严格的类型检查也常常导致编译错误（尤其是Go没有C中的隐式`bool`类型转换），以后应多加注意。

对于`walk`，其实实现原理与xv6中相同，如下：
```go
func walk(pagetable pagetable_t, va uintptr, alloc bool) *pte_t {
	if va >= MAXVA {
		panic("walk")
	}

	for level := 2; level > 0; level-- {
		idx := PX(level, va)
		pte_ptr := (*pte_t)(unsafe.Pointer(uintptr(pagetable) + idx*8))
	
		if (*pte_ptr & PTE_V) != 0 {
			pagetable = pagetable_t(PTE2PA(*pte_ptr))
		} else {
			if !alloc {
				return nil
			}

			new_page := kalloc()
			if new_page == 0 {
				return nil
			}

			memset(new_page, 0, uint(PGSIZE))

			*pte_ptr = PA2PTE(new_page) | PTE_V
			pagetable = pagetable_t(new_page)
		}
	}

	idx0 := PX(0, va)
	return (*pte_t)(unsafe.Pointer(uintptr(pagetable) + idx0*8))
}
```

### 实验效果

	C OK
	kmeminit... kinit: [2147618816, 2281701376)
	freerange: [2147618816, 2281701376)
	OK
	kvminit...  kernel_pagetable at 0x87fff000
	mappages 0x10000000, 0x10000000
	mappages 0x10001000, 0x10001000
	mappages 0xc000000, 0xc000000
	mappages 0x80000000, 0x80000000
	mappages 0x80001000, 0x80001000
	OK
	kvminithart...  OK
	--- printf test ---
	2147483647
	Hello there
	Today is Monday
	, M 1 2
	--- kalloc test ---
	test kalloc
	allocate 130660 KB memory