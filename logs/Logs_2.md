# 实验记录
## Log 06: 权限过渡
### 问题描述
这是在实现时钟中断前发现的一个严重问题。之前的代码一直运行在`Machine-mode`之下。默认情况下，M-mode 下的内存访问不经过 MMU 地址翻译。我们在`vm.go`的`kvminit`中注释掉一行关键映射代码，发现系统可以正常运行，因此情况属实。

### 处理方法
我们还是回到xv6，看看它的`start.c`中是怎么处理的。

首先，我们先将xv6中封装好的寄存器读写函数直接迁移过来（没办法，只能用C实现内联汇编）。然后观察`start`函数，其涉及到以下几个寄存器的修改。

| 名称  | 作用  |
| --- | --- |
| `mstatus` | 机器模式状态寄存器，控制全局中断使能、特权级等 |  
| `mepc` | 机器异常程序计数器，保存异常/中断发生的指令地址 | 
| `satp` | 页表 |
| `medeleg` | 机器异常委托寄存器，将特定异常委托给低特权级处理 | 
| `mideleg` | 机器中断委托寄存器，将特定中断委托给低特权级处理 | 
| `sie` | 机器模式中断使能寄存器|
| `tp` | 线程指针，通常用于存储每线程数据地址  |
| `mhartid` | 硬件线程ID，标识当前执行的硬件线程 |

我们将`start`函数迁移到我们的`c_start`代码中。

补充，此处我们改为使用`mret`，而不是使用函数调用的原因：`mret` 是 RISC-V 架构中用于在特权模式间进行正式、受控切换的唯一标准机制。这种从固件（M模式）到内核（S模式）的“交接仪式”必须用它来完成。

### 问题与解决
初次修改，我忽略后面两个寄存器，然后添加了前面的。运行发现，系统在`mret`后无反应了。

使用GDB调试发现，`mret`前，寄存器的值确实指向`KMain`，但是执行`mret`这一步时卡住了。中断并查看寄存器，如下：

    (gdb) p/x $pc
    $3 = 0x0
    (gdb) p/x $mepc
    $4 = 0x0
    (gdb) p/x $mcause
    $5 = 0x1
    (gdb) p/x (($mstatus >> 11) & 3)
    $11 = 0x3

可见，`pc`，`mepc`被异常清零，`mstatus`显示为M-Mode，`mcause`提示指令访问异常。

经过长时间Debug，唯一的可能性是设置为S-Mode后系统没有权限读取内存数据。一直没有发现的原因在于xv6 2020年的版本也没有设置，不过查找了2024年版本，确实有相关设置。加上这两行，并补充相关定义后可以进入系统了。至于为什么2020版没有这部分设置也可以正常进入系统，我们不得而知。

```c
  w_pmpaddr0(0x3fffffffffffffull);
  w_pmpcfg0(0xf);
```

### 效果

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
    kvminithart...

看来页表映射出问题了，我们在下一个日志解决...

### 附录：调试方法
两个窗口分别启用qemu与gdb：
```
qemu-system-riscv64 -machine virt -bios none -kernel kernel-qemu   -nographic -serial mon:stdio   -s -S

gdb-multiarch kernel-qemu
```

然后在gdb远程连接：
```
target remote localhost:1234
```

然后就可以了。

---
## Log 07: 页表问题修复
### 问题描述
- 系统启动卡在了开启页表映射一环，说明页表映射存在问题。

### 问题解决

用gdb定位，发现问题在这里：

    (gdb) stepi
    sfence_vma () at src/riscv.h:339
    339       asm volatile("sfence.vma zero, zero");
    (gdb) stepi
    Cannot access memory at address 0x80000b5e

但这个地址确实在我的映射范围之内啊

    mappages 0x80000000, 0x80000000
    mappages 0x80001000, 0x80001000

我们进一步追溯到问题根源：

    Breakpoint 1, kvminithart () at src/riscv.h:214
    214       asm volatile("csrw satp, %0" : : "r" (x));
    (gdb) p/x x
    $1 = 0x8000000000000000
    (gdb) p/x kernel_pagetable
    $2 = 0x0

Go中定义的kernel_pagetable的值似乎并没有同步到C中！

既然external声明不靠谱，我们用显式传参来传入kernel_pagetable值。之后，我们成功进入了系统：

    C OK
    ......
    allocate 130660 KB memory

---
## Log 08: C 代码架构优化
### 整体内容
- 将init.c分为小文件，提高可读性

### 细节
分为了三个部分：
- goc.c: Go无法调用底层的地方，由C代替
- runtime.c: 欺骗Go的Runtime，让它以为自己运行在操作系统之上
- start.c: 启动入口，之后跳到KMain

---
## Log 09: 时钟中断
### 实验目标
- 实现时钟中断的处理，使得每个一段时间打印一个`tick`（这是后面多任务调度的基础）

### 实验思路
首先我们来看看整体时钟中断过程的全貌：

系统启动阶段，`timerinit`函数设置`CLINT_MTIMECMP`, `interval`等中断信息，设置`mtevc`存储 M-Mode 中断处理代码的位置，然后启用中断。

当时钟到点了（`CLINT_MTIMECMP == CLINT_MTIME`），启动M-Mode中断，CPU升至M-Mode，读取`mtevc`指向位置（`timervec`），然后跳转。

`timervec`负责重置下一个中断的时间点，并设置`sip`位，提示有S-Mode的中断待处理，然后`mret`返回。

`mret`读取`mepc`中的数据，即CPU被打断前的pc位置，然后返回至该位置。但由于`sip`不为`0`，又有新的S-Mode中断需要处理，此时CPU再跳转到`stevc`所处的位置。（即`kernelvec`，非直接`trap`处理函数，因为需要处理寄存器相关内容）

跳转后，`kernelvec`负责备份寄存器的值，然后调用我们的`kerneltrap`处理程序。程序返回后，再恢复，之后`sret`回到之前的状态，即操作系统的正常工作流程状态。

这就是整个时钟中断处理的全貌。

### 坑
整个板块实现了，但是似乎并没有触发中断。

后面找了半天，才发现没有打开S-Mode的中断。（xv6是在scheduler中打开的，所以我没有注意到）

---
## Log 10: Spinlock
### 实验目标
- 实现自旋锁（内容不必过多展开，和xv6的自旋锁一样）

### 实现与测试思路
在这里复现一下锁的简化运行流程。注意，我们此时没有多进程的概念，因此测试锁时只能用中断来制造一种“伪并发”（虽然这个测试并不能完全保证锁的正确性）。我们创建一个含锁的`count`结构体，分别在`KMain`与中断处理程序中执行`count.num++`。

`KMain`在中断处理初始化前执行`initlock(count.lock)`。随即中断打开。

`KMain`进入一个固定次数（如`10000`）的循环，每次`aquire(count.lock)`，然后`count.num++`，之后`release(count.lock)`。中断执行一样的`aquire`，`release`，不过总的次数受某个独立值约束（如`addLimit := 1000`）。等待足够长的时间后，输出总值。

`acquire`的设计：首先需要关闭中断，防止死锁，接着进入循环等待锁被腾出，锁被腾出后，立即设置锁的状态被锁定，然后返回。

`release`的设计：将锁的状态设定为未锁定，然后返回。

另外注意，C 中`static` `inline`的关键字可能会使将定义的函数优化掉，使Go识别不到定义的函数。（原`riscv.h`中的`intr_on`与`intr_off`）

### 初次实现
这是一个非常朴素的想法，但存在严重的漏洞：
```go
type spinlock struct {
	locked bool
}

func initlock(lk *spinlock) {
	lk.locked = false
}

func acquire(lk *spinlock) {
	intr_off()
	for lk.locked {}
	lk.locked = true;
}

func release(lk *spinlock) {
	lk.locked = false
	intr_on()
}
```

加入打印语句调试，发现这样的情况：

    acquire.acquire... OK
    release... acquire... OK

非常有趣，一个进程正在执行`acquire`，但连打印语句都还没执行完，CPU就被另一个进程抢去了，它执行到`release`，`OK`都还没打印，又有另一个进程来`acquire`了。

### 修改方案
我们不得不摒弃之前的朴素想法，必须将整个`acquire`/`release`过程原子化地执行下去，而不能有打断。我们转向xv6的`spinlock.c`中的巧妙设计。

`spinlock.c`中使用了C编译器特有的`__sync_lock_test_and_set`等指令保证原子化，我们在Go中通过`goc.c`进行复用。如下：

```go
func acquire(lk *spinlock) {
	intr_off()
	printf("acquire... ")
	for sync_test_and_set(&lk.locked) == 1 {}
	sync_barrier()
	printf("OK\n")
}

func release(lk *spinlock) {
	printf("release... ")
	sync_release(&lk.locked)
	printf("OK\n")
	intr_on()
}
```

现在似乎可以整齐打印`acquire`与`release`了，但是突然发现`KMain`在打印了几十行后就再也不见了。

完善调试信息，发现，这种原因并不是卡在`KMain`的无限循环了。当`KMain`执行完`count.num++`后，反复地发生中断，根本没有`KMain`执行的机会。

难道是`trap`异常，不能返回到原函数？确实很可能，我们没有清除`sip`位，可能会导致一直中断，完成下面一行代码来补救：
```go
w_sip(r_sip() & ^uintptr(2))
```

下面的测试表明修正成功了：
```go
for {
	for i := 0; i < 1000000; i++ {
		printf("")
	}
	printf("back\n")
}
```

    ...
    tick
    back
    tick
    back
    back
    tick
    back
    tick
    back
    back
    tick
    ....

另外，调试输出的打印时间过长也很可能导致一个中断结束，另一个中断又开始了，因此先去除调试输出。

反复修改无果后，最后发现了一个关键之处：
```go
	w_sip(r_sip() & ^uintptr(2))
```
这个语句的位置很重要！将它放在`Kerneltrap`函数开头，就解决了之前的问题。

### 效果
测试程序：
```go
func spinlockTest() {
	printf("--- spinlock test ---\n")
	for i := 0; i < 1000; i++ {
		for i := 0; i < 100000; i++ {
			printf("")
		}
		acquire(&count.lock)
		count.num++
		release(&count.lock)
	}
	printf("Current Count: %d\n", count.num)
	for addLimit < 1000 {
		for i := 0; i < 1000000; i++ {
			printf("")
		}
		printf("Waiting... current addLimit: %d\n", addLimit)
	}
	printf("Expected Count: 2000, Real Count: %d\n", count.num)
}
```

`trap.go`中相应部分：
```go
func Kerneltrap() {
	w_sip(r_sip() & ^uintptr(2))
	if addLimit < 1000 {
		acquire(&count.lock)
		count.num++
		addLimit++
		release(&count.lock)
	}
}
```

    Expected Count: 2000, Real Count: 2000

（不过关闭锁的情况下，也没有出错的情况...）
