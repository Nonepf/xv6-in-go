package main

import "unsafe"

const (
	SYS_write = 1
	SYS_exit  = 2
)

func syscall(p *Proc) {
	switch p.trapframe.a7 {
	case SYS_write:
		p.trapframe.a0 = sys_write(p)
	case SYS_exit:
		exitproc(int(p.trapframe.a0))
	default:
		p.trapframe.a0 = ^uintptr(0) // -1
	}
}

func sys_write(p *Proc) uintptr {
	fd := p.trapframe.a0
	addr := p.trapframe.a1
	n := p.trapframe.a2
	if fd != 1 && fd != 2 {
		return ^uintptr(0)
	}
	if n == 0 {
		return 0
	}
	// 当前用户程序只有 [0, sz)；先检查完整范围，避免溢出和半截输出。
	if n > p.sz || addr > p.sz-n {
		return ^uintptr(0)
	}

	// 先检查每一页的权限。不能把用户指针交给内核直接解引用。
	end := addr + n
	for va := PGGROUNDDOWN(addr); va < end; va += PGSIZE {
		if walkaddr(p.pagetable, va) == 0 {
			return ^uintptr(0)
		}
	}
	for src := addr; src < end; {
		va := PGGROUNDDOWN(src)
		pa := walkaddr(p.pagetable, va)
		count := PGSIZE - (src - va)
		if count > end-src {
			count = end - src
		}
		// 物理页在内核页表中恒等映射，不需要额外的 Go 堆缓冲区。
		for i := uintptr(0); i < count; i++ {
			uart_putc(*(*byte)(unsafe.Pointer(pa + src - va + i)))
		}
		src += count
	}
	return n
}
