# 实验记录
## Log 11: 内核态多进程
### 实验目标
- 实现在内核态中进行多个进程的调度

### 纵向比较
与一般的函数跳转不同，每个内核态进程有自己独立的栈空间，并感觉是自己在独占CPU（寄存器）；但与用户态的多进程不同，内核态进程共用内核页表。

另外，内核态进程不需要处理 `trapframe`，此处回顾一下 `trapframe` 的概念：

|**特性**|**Trapframe (陷阱帧)**|**Context (上下文)**|
|---|---|---|
|**保存时机**|**用户态进入内核**时 (Trap)|**内核线程 A 切到 B** 时 (Yield)|
|**保存内容**|**全部** 32 个通用寄存器|只保存 **Callee-saved** 寄存器 (`ra`, `sp`, `s0-s11`)|
|**存放位置**|进程页表中的一个**物理页**|进程 PCB 结构体中的一个**字段**|
|**目的**|保证从内核返回用户态时，用户程序能**原地复活**|保证内核在多个任务之间**来回跳转**|

### 实现细节-数据结构
下面给出具体的实现步骤：
- [ ] 定义 `kproc` 结构体，其主体内容如下（由 xv6 的 `proc.h` 简化，只保留必需的，后面根据需求添加）
```go
type KProc struct {
    lock Spinlock

    // p->lock must be held when using these:
    state procstate         // Process state
    pid int                 // Process ID

    // p->lock needn't be held since they are private to kproc
    kstack uintptr
    context Context
    char name[16]           // Process name (debugging)
}
```

当然需要补充 `procstate` 的定义。

- [ ] 补全 `Context` 的定义。这里可以完全仿照 xv6.
```go
// Saved registers for kernel context switches.
type Context struct {
    ra uintptr
    sp uintptr

    // callee-saved
    s0 uintptr
    s1 uintptr
    s2 uintptr
    s3 uintptr
    s4 uintptr
    s5 uintptr
    s6 uintptr
    s7 uintptr
    s8 uintptr
    s9 uintptr
    s10 uintptr
    s11 uintptr
}
```

现在，我们拥有了一个完整的内核态进程的定义，可以开始为其设计函数了。

### 实现细节-函数
- [ ] `procinit`: 初始化进程，为它们提供 `pid`，`name`，分配并映射`kstack`，并初始化它们的 `spinlock`, `state`。

- [ ] `allocProc`: 分配某个函数给某个进程

- [ ] `scheduler`: 负责遍历进程数组，将下一个可运行的进程标为可运行状态

- [ ] `Kerneltrap`: 修改此中断处理程序，如果为时钟中断，则 `yeild` 当前进程，然后执行语境切换汇编，接着调用 `scheduler`

- [ ] `swtch.S`: 负责上下文切换.

此处补充上下文切换的完整过程：

`KMain`:
- `procinit`: 初始化
- `scheduler`: 进入...（此时中断应当是关闭的）

`scheduler`:
- 进入无限循环，遍历进程数组
- 发现一个`RUNNABLE`的进程,`aquire(&p.lock)`
- 设置其状态为`RUNNING`，然后调用`swtch(&cpu_context, &p.context)`
- 释放锁

`swtch.S`:
- 交换当前`ra`与`p.context`中的`ra`
- 交换寄存器的值
- `ret`（此时会`ret`到`ra`中的位置，即`p`的位置）

`p`:
- 先释放锁
- 运行直到发生时钟中断

`kernelvec.S`:
- 管理寄存器，调用`Kerneltrap`

`Kerneltrap`:
- 判断中断类型，发现是时钟中断
- 判断当前语境，发现是从进程跳出
- 使其`yeild`

`yeild`:
- 申请锁
- 将状态改为`RUNNABLE`
- `swtch`交换语境（进入scheduler）
- 释放锁（但并不是马上释放，此时该进程已经`sleep`了）

### 细节补充
- 注意 `kstack` 的映射

### 主要问题
#### 函数传参
在 Go 中，函数传参后，不能直接解引用得到函数的地址。这个问题困扰了我很久，最后还是选择用硬编码妥协。不过为了普适性，我新增了一个中介函数，并将调用函数存储在`proc`结构体之中，只需要记住该函数的硬编码地址即可。
```go
p.context.ra = uintptr(0x80000d6a)
```

#### swtch.S
Go 有一些寄存器需要另外存储

#### 函数设计
之前的测试函数：
```go
func printA() {
	for {
		printf("A\n")
		for i := 0; i < 1000000; i++ {
            print("")
		}
	}
}
```
这样会导致栈空间很容易就耗尽了。

本来遇到的问题挺多的，跑通代码后懒得写了，直接AI总结一下（很多时候还得靠AI Debug，自己的知识储备太少了，经验不足，唉）：

| **问题现象**             | **核心原因描述**                                         | **最终解决方案**                                              |
| -------------------- | -------------------------------------------------- | ------------------------------------------------------- |
| `panic: remap`       | 内核栈使用物理地址等值映射，导致与 `kalloc` 分配的页表内存冲突。              | 使用 `KSTACK(i)` 函数将内核栈映射到虚拟地址空间顶端（$MAXVA$ 附近）。           |
| 始终只打印 `A`            | 进程启动后未开启 `sstatus.SIE`，导致时钟中断无法抢占当前循环。             | 在 `TaskStub` 中显式调用 `intr_on()`，并释放 `scheduler` 传过来的进程锁。 |
| `Kerneltrap 0x1` 未调度 | M-Mode 或模拟器将时钟中断以软件中断（SSI）形式转发给 Supervisor。        | 在 `trap.go` 中同时处理 `0x1` 和 `0x5` 两种 `scause`，并正确清除中断位。   |
| `A B C` 后死锁/崩溃       | `yield` 中 `swtch` 返回后未释放进程锁，导致下次调度时重复申请锁。          | 确保 `yield` 函数在 `swtch` 指令之后的下一行立即执行 `release(&p.lock)`。 |
| `A B C A` 后 `0xd`    | `swtch` 仅保存了通用寄存器，未恢复 Go 运行依赖的 `gp` (全局指针) 或 `tp`。 | 扩展 `Context` 结构体和 `swtch.S`，在切换中强制保存和恢复 `gp` 与 `tp`。    |
| `printf` 随机位置报错      | 进程栈（4KB）在多次中断嵌套和 `printf` 递归调用下发生溢出。               | 增加 `KSTACK` 的映射范围（如 8KB），并确保 `sp` 指针始终 16 字节对齐。         |

### 效果
测试函数：
```go
func schedTest() {
	printf("--- scheduler test ---\n")
	allocProc(printA)
	allocProc(printB)
	allocProc(printC)
}
```

效果：
（不好复制粘贴，大致如下）

    A
    A
    A
    ...
    A
    A
    B
    B
    ...
    B
    C
    ...
    C
    C
    A
    ...
    ...

## Log 12: 多线程代码锁的优化
### 问题
- 之前只是为了跑起来，对锁的占有，释放没有太多斟酌，`TaskStub`里面释放锁的操作使得整体并不是一个`acquire`，一个`release`这么对应的，应当想一下一下怎么改。

### 实际
没有进行任何修改...

要求`acquire`与`release`总体数量相同仅在有限情形下如此，在无限循环中则不然。我们重新回顾一下整个锁的占有/释放流程。

初次阶段：
`scheduler`申请锁，发现一个`RUNNABLE`的进程(记为`A`)，进行`swtch`，转到该进程的`TaskStub`。`TaskStub`释放锁，然后进入`A`。

进程中，发生时钟中断，判断当前为`A`，于是调用`yield`，其申请锁，转换上下文，进入`scheduler`。`scheduler`恢复到之前切换上下文的下一步，释放锁（释放`yield`申请的锁）

第二次：
`scheduler`申请锁，发现`A`，于是切换上下文，此时进入`yield`上次的位置，释放锁（释放`scheduler`申请的锁），然后继续，从中断处理程序返回`A`。

整体而言，锁的申请与释放是合理的。

## Log 13: `TaskStub`地址传参优化
### 问题描述
- 之前采用硬编码的形式让程序跑起来了，这次动态获取`TaskStub`函数的地址，然后调用。

### 解决方案
之前尝试过在 Go 中直接取地址，失败。后面发现了一种行之有效的办法：

在`swtch.S`中加一段：
```asm
.global GetTaskStubAddr
GetTaskStubAddr:
    la a0, TaskStub 
    ret
```

在`proc.c`中：
```go
//go:linkname GetTaskStubAddr GetTaskStubAddr
func GetTaskStubAddr() uintptr
```
后面直接根据这个函数接口获取TaskStub地址即可