## Log 14 - 1: 用户态 准备
### 整体实验目标
- 初步实现从内核态跳到用户态这个过程，然后陷入循环。（还没实现系统调用，无法打印结果，只能先用 `gdb` 来看看成果了）

本部分则负责进行最基本的准备工作，不涉及进程的启动，调度等，因此只保证编译正确.
（参考 xv6 `kernel/proc.c`，`kernel/trap.c`，`kernel/trapoline.S`）
### 全过程模拟

#### 主函数
首先，系统启动，将有关内核的东西准备好后，然后进入 `userinit`。`userinit` 准备好第一个进程的相关事宜，然后返回一个进程指针 `p`. 

#### 详细流程1-`proc`层面
`userinit`:
- 调用 `allocproc`
- 使用`kalloc`，分配一页物理内存，然后用`memset`归零.
- 设置 `p.trapframe.epc = 0`，之后`sret`后，从虚拟地址 `0` 开始运行.
- 将陷阱帧的`sp`栈指针设置为 `PGSIZE`[^1]，让代码段与栈凑合着用一个页.
- 将代码复制到该物理页（`memmove`）[^2]
- 释放锁[^3]

[^1]: 为何不是`PGSIZE-1`，`PGSIZE`不是无效地址吗？首先，当我们要往栈里压入一个 8 字节的数据时，CPU 会先执行 `sp = sp - 8`，然后再把数据存入 `sp` 指向的内存，并不会访问到无效区域。其次，RISC-V 的要求 `sp` 必须保持 16 字节对齐.
[^2]: 此位置在内核页表中已被映射，无需担心MMU翻译问题；不过需要把那个 `TRAPOLINE` 的内核页表映射补上.
[^3]: 锁在 `allocproc` 中被持有，直到一切准备就绪后可以释放.

`allocproc`:
- 在可用进程数组中遍历，找到一个 `UNUSED` 的进程.
- 为 `trapframe` 分配物理空间[^4]
- 调用`proc_pagetable`，创建一个新的用户页表
- 将 `p.context` 清空，然后设置 `context` 的返回地址`ra`为 `forkret`（目前没有复杂功能，就是调用`usertrapret`，不过给个函数定义，为后期扩充提供空间），`sp` 为内核栈顶部（`p.kstack + PGSIZE`）.[^5]

[^4]: `p->trapframe` 是一个指针，而非内嵌在 `proc` 中的结构体，目的是为了方便汇编码的操控，也是为了良好的隔离性（内核与用户都需要进行操作）。
[^5]: `context.sp` 与 `trapframe.sp` 对比：前者是内核态的栈指针。它决定了当该进程在内核中运行时，它的函数调用和局部变量存在哪里；后者是用户态的栈指针。它决定了用户程序运行时的栈在哪里。

`proc_pagetable`:
- 调用 `uvmcreate` （补充在 `vm.go`中，就是分配一块内存然后清零）
- 映射 `TRAPOLINE` 与 `TRAPFRAME` （注意，用户无权限进行读写，这是给内核用的）

#### 详细流程2- `trap`层面
`usertrapret`
- 首先关闭中断（下面的步骤不能被打断！）
- 调用`w_stvec`，修改中断处理代码位置为 `TRAMPOLINE + (uservec - trampoline)` [^6]
- 保存 `satp`（内核页表），`sp`（进程的内核栈），`trap`，`hartid`(暂时用不着)到陷阱帧.
- 配置用户模式的功能（允许中断）
- 配置 `sepc` 为 `trapframe.epc`（恢复后，读取`sepc`，然后从那里开始跑）
- 编辑好用户的 `satp`，等会儿传参进去.
- 进入`trampoline.S`.（关键之处在于，不能直接使用其物理地址的恒等映射，否则切换页表后，CPU会迷失）

[^6]: 不能直接弄成 `uservec`，那是内核态的地址/物理地址，用户空间中找不到！

#### 详细流程3- `trampoline` 层面
`trapoline.S` 有两个函数：
- `uservec`
- `userret`
具体不多说，负责不同时段的封存寄存器，切换页表等.

### 实操
首先完善`kvminit` 页表映射，修改链接文件，添加`trampoline` 位置，设置 `trampoline` 结构体，完善 `Proc` 结构体等，同时将 xv6 的 `trapoline.S` 搬运过来。

完善 `memmove`，`uvminit` 等基本函数. 如下：
```go
func memmove(dst uintptr, src uintptr, n uintptr) {
	s := src
	d := dst

	if (s < d && s + n > d) {
		s += n
		d += n
		for ; n > 0; n-- {
			d--
			s--
			*(*byte)(unsafe.Pointer(d)) = *(*byte)(unsafe.Pointer(s))
		}
	} else {
		for ; n > 0; n-- {
			*(*byte)(unsafe.Pointer(d)) = *(*byte)(unsafe.Pointer(s))
			d++
			s++
		}
	}
}
```

另外，我创建了一个小程序（死循环），然后在链接文件中对其所属的段进行规划。
```asm
.section .user_code
.global user_proc
user_proc:
loop:
    j loop
```

做好基础准备工作后，进入具体实操环节.

在 `userinit` 中，对于代码长度的获取还是需要 `C` 来帮忙。如下：
```go
// first user process
func userinit() {
    var p *Proc
    p = allocproc()

    uvminit(p.pagetable, get_initcode(), get_user_proc_size())

    p.trapframe.epc = 0
    p.trapframe.sp = PGSIZE
    p.state = RUNNABLE

    release(&p.lock)
}
```

其他地方按照上述流程依葫芦画瓢就行了。（~~懒得写了~~）

### 烦人的细节
之前已经解决过Go中函数地址的获取问题了，此处系统化地说明一下：
1. 使用`export`标签使函数全局可见（注意首字母大写）
2. 在汇编中编写获取地址函数，示例：
```asm
.global GetUsertrapAddr
GetUsertrapAddr:
    la a0, Usertrap 
    ret
```
3. 在 Go 中声明此函数
4. 通过函数调用获取地址

还有一点，我们在 `Usertrapret` 函数中，计算到了一个函数的地址，还是需要用汇编写一个跳板.

另外，Go 的强类型检查还是有点烦，需要多加注意。

---
## Log 14 - 2: 用户态 进入
### 实验目标
- 我们利用之前搭建好的工具，首次进入用户态系统（然后暂且卡在无限循环之中）

### 具体实现
突然发现似乎没有什么要特别做的...... 只需要在`KMain` 加上 `userinit` 就可以了。后面全是漫长的 Debug 环节.

首先，修复掉加上 `userinit` 后带来的编译错误，然后分析以下错误的原因：

	trapinithart...  OK
	userinit...  mappages 0x3ffffff000, 0x80002000
	mappages 0x3fffffe000, 0x87faf000
	Kerneltrap 0xc at 0x80002112

使用 `printf` 溯源，最后发现，`GetForkretAddr` 跟在 `trampoline` 后面，根本没有执行权限，放在其他地方就好了（不过很奇怪，`trampoline` 那个段在映射时不是有权限吗？）

还有这里条件苛刻了，本来就会对齐为一个 `PGSIZE`:
```go
if (sz > PGSIZE) {
		panic("uvminit: more than a page")
	}
```

搞掉了这个bug，继续，发现卡住了？使用gdb不断追踪，追踪到这个位置：

	(gdb) x/5i 0x3ffffff090
	   0x3ffffff090:        unimp
	   0x3ffffff092:        unimp
	   0x3ffffff094:        unimp
	   0x3ffffff096:        unimp
	   0x3ffffff098:        unimp

程序先跳到 `0x3ffffff090`,后面直接卡住. 查看字节编码，的确是有问题的：

	(gdb) x/100xb 0x3ffffff000
	0x3ffffff000:   0x00    0x00    0x00    0x00    0x00    0x00    0x00    0x00
	0x3ffffff008:   0x00    0x00    0x00    0x00    0x00    0x00    0x00    0x00
	0x3ffffff010:   0x00    0x00    0x00    0x00    0x00    0x00    0x00    0x00
	0x3ffffff018:   0x00    0x00    0x00    0x00    0x00    0x00    0x00    0x00
	0x3ffffff020:   0x00    0x00    0x00    0x00    0x00    0x00    0x00    0x00
	0x3ffffff028:   0x00    0x00    0x00    0x00    0x00    0x00    0x00    0x00
	0x3ffffff030:   0x00    0x00    0x00    0x00    0x00    0x00    0x00    0x00
	0x3ffffff038:   0x00    0x00    0x00    0x00    0x00    0x00    0x00    0x00
	0x3ffffff040:   0x00    0x00    0x00    0x00    0x00    0x00    0x00    0x00
	0x3ffffff048:   0x00    0x00    0x00    0x00    0x00    0x00    0x00    0x00
	0x3ffffff050:   0x00    0x00    0x00    0x00    0x00    0x00    0x00    0x00
	0x3ffffff058:   0x00    0x00    0x00    0x00    0x00    0x00    0x00    0x00
	0x3ffffff060:   0x00    0x00    0x00    0x00

很有可能是没有正确映射，后检测原地址 `0x80002090`，同样也是全零。因此问题不在于这里。

后面检测编译后的文件，发现：

	  1 .trampsec     00000112  0000000080002000  0000000080002000  00005000  2**4
	                  CONTENTS, READONLY
推测由于是 `READONLY`，因而会在加载时被优化掉，不加载。后在 `trampoline` 中修改：
```asm
.section .trampsec, "ax", %progbits
```
让编译器不要优化。

修改后，确实有成效：

	(gdb) x/5i 0x0000003ffffff000
	=> 0x3ffffff000:        csrrw   a0,sscratch,a0
	   0x3ffffff004:        sd      ra,40(a0)
	   0x3ffffff008:        sd      sp,48(a0)
	   0x3ffffff00c:        sd      gp,56(a0)
	   0x3ffffff010:        sd      tp,64(a0)

但还是卡住了。反复定位到了这个位置：

	(gdb) x/20i 0x3ffffff10e
	=> 0x3ffffff10e:        sret

执行后，出现故障，跳回到 `0x3ffffff000` 这个地方。

我们在 `sret` 前执行检查：

	(gdb) p/x $sstatus
	$1 = 0x200000020
	(gdb) set $sstatus |= 0x40000
	(gdb) x/5i 0x0
	   0x0: j       0x0
	   0x2: unimp
	   0x4: unimp
	   0x6: unimp
	   0x8: unimp

。。。

不知过了多久，终于发现风险点——中断开关！我还没有实现用户态的中断处理，这里应该关掉！

修改后，终于可以运行了......

### 效果

	(gdb) target remote localhost:1234
	Remote debugging using localhost:1234
	0x0000000000001000 in ?? ()
	(gdb) c
	Continuing.
	^C
	Program received signal SIGINT, Interrupt.
	0x0000000000000000 in ?? ()
	(gdb) si
	0x0000000000000004 in ?? ()
	(gdb) si
	0x0000000000000000 in ?? ()
	(gdb) x/5i $pc
	=> 0x0: li      a0,291
	   0x4: j       0x0
	   0x6: unimp
	   0x8: unimp
	   0xa: unimp
	(gdb)

### 补充
（Gemini生成）再次明晰 `sie`，`sip`，`sstatus`的职责

- **`sie` (Supervisor Interrupt Enable)**：**“准入清单”**。 它决定了**哪些种类**的中断是被允许的（比如时钟中断、外部设备中断）。即使有人敲门，如果它不在 `sie` 的清单上，安保理都不理。
- **`sip` (Supervisor Interrupt Pending)**：**“待处理信封”**。 它是一个状态显示。如果某个中断发生了（比如闹钟响了），但 CPU 还没来得及处理，`sip` 对应的位就会亮起。它告诉你：“有一个中断正在门口等着呢。”
- **`sstatus` (Supervisor Status)**：**“总闸 & 状态记录仪”**。 它里面的 `SIE` 位是**全局开关**。就算 `sie` 清单里允许了时钟中断，如果 `sstatus.SIE` 总闸是关的，所有的中断都进不来。