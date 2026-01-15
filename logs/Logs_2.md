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
