package main

// Physical memory layout
// a go version of memlayout.h

// qemu -machine virt is set up like this,
// based on qemu's hw/riscv/virt.c:
//
// 00001000 -- boot ROM, provided by qemu
// 02000000 -- CLINT
// 0C000000 -- PLIC
// 10000000 -- uart0 
// 10001000 -- virtio disk 
// 80000000 -- boot ROM jumps here in machine mode
//             -kernel loads the kernel here
// unused RAM after 80000000.

// the kernel uses physical memory thus:
// 80000000 -- entry.S, then kernel text and data
// end -- start of kernel page allocation area
// PHYSTOP -- end RAM used by the kernel

// qemu puts UART registers here in physical memory.
const (
	UART0 = uintptr(0x10000000)
	UART0_IRQ = 10
)

// virtio mmio interface
const (
	VIRTIO0 = uintptr(0x10001000)
	VIRTIO0_IRQ = 1
)

// core local interruptor (CLINT), which contains the timer.
const (
	CLINT = uintptr(0x2000000)
	CLINT_MTIME = CLINT + 0xBFF8
)
func CLINT_MTIMECMP(hartid int) uintptr { return CLINT + 0x4000 + 8*uintptr(hartid) }

// qemu puts platform-level interrupt controller (PLIC) here.
const (
	PLIC = uintptr(0x0c000000)
	PLIC_PRIORITY = PLIC + 0x0
	PLIC_PENDING = PLIC + 0x1000
) 
func PLIC_MENABLE(hart int) uintptr { return PLIC + 0x2000 + uintptr(hart)*0x100 }
func PLIC_SENABLE(hart int) uintptr { return PLIC + 0x2080 + uintptr(hart)*0x100 }
func PLIC_MPRIORITY(hart int) uintptr { return PLIC + 0x200000 + uintptr(hart)*0x2000 }
func PLIC_SPRIORITY(hart int) uintptr { return PLIC + 0x201000 + uintptr(hart)*0x2000 }
func PLIC_MCLAIM(hart int) uintptr { return PLIC + 0x200004 + uintptr(hart)*0x2000 }
func PLIC_SCLAIM(hart int) uintptr { return PLIC + 0x201004 + uintptr(hart)*0x2000 }

// the kernel expects there to be RAM
// for use by the kernel and user pages
// from physical address 0x80000000 to PHYSTOP.
const (
	KERNBASE = uintptr(0x80000000)
	PHYSTOP = KERNBASE + 128*1024*1024
)

// map the trampoline page to the highest address,
// in both user and kernel space.
//const TRAMPOLINE = MAXVA - PGSIZE

// map kernel stacks beneath the trampoline,
// each surrounded by invalid guard pages.
//func KSTACK(p int) uintptr { return TRAMPOLINE - ((p)+1)* 2*PGSIZE }

// User memory layout.
// Address zero first:
//   text
//   original data and bss
//   fixed-size stack
//   expandable heap
//   ...
//   TRAPFRAME (p->trapframe, used by the trampoline)
//   TRAMPOLINE (the same page as in the kernel)
//const TRAPFRAME = TRAMPOLINE - PGSIZE
