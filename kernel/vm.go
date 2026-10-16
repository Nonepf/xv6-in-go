package main

import "unsafe"

//go:external kernel_pagetable
var kernel_pagetable pagetable_t

//go:linkname get_etext get_etext
func get_etext() uintptr

func kvminit() {
	kernel_pagetable = pagetable_t(kalloc())
	if kernel_pagetable == 0 {
		panic("kvminit: kalloc failed")
	}
	printf("kernel_pagetable at %x\n", uintptr(kernel_pagetable))
	memset(uintptr(kernel_pagetable), 0, uint(PGSIZE))

	kvmmap(UART0, UART0, PGSIZE, PTE_R|PTE_W)
	kvmmap(VIRTIO0, VIRTIO0, PGSIZE, PTE_R|PTE_W)
	kvmmap(PLIC, PLIC, 0x400000, PTE_R|PTE_W)
	kvmmap(KERNBASE, KERNBASE, get_etext()-KERNBASE, PTE_R|PTE_X)
	kvmmap(get_etext(), get_etext(), PHYSTOP-get_etext(), PTE_R|PTE_W)
	kvmmap(TRAMPOLINE, get_trampoline(), PGSIZE, PTE_R|PTE_X)
}

//go:linkname kvminithart kvminithart
func kvminithart(kernel_pagetable pagetable_t)

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

func kvmmap(va uintptr, pa uintptr, sz uintptr, perm int) {
	if mappages(kernel_pagetable, va, sz, pa, perm) != 0 {
		panic("kvmmap")
	}
}

func mappages(pagetable pagetable_t, va uintptr, size uintptr, pa uintptr, perm int) int {
	printf("mappages %x, %x \n", va, pa)
	if size == 0 || va >= MAXVA || size > MAXVA-va {
		return -1
	}
	a := PGGROUNDDOWN(va)
	last := PGGROUNDDOWN(va + size - 1)
	for {
		pte := walk(pagetable, a, true)
		if pte == nil {
			return -1
		}
		if *pte&PTE_V != 0 {
			panic("remap")
		}
		*pte = PA2PTE(pa) | pte_t(perm|PTE_V|PTE_A|PTE_D)
		if a == last {
			break
		}
		a += PGSIZE
		pa += PGSIZE
	}
	return 0
}

func uvminit(pagetable pagetable_t, src uintptr, sz uintptr) bool {
	if sz > PGSIZE {
		panic("uvminit: more than a page")
	}

	mem := kalloc()
	if mem == 0 {
		return false
	}
	memset(mem, 0, uint(PGSIZE))
	if mappages(pagetable, 0, PGSIZE, mem, PTE_W|PTE_R|PTE_X|PTE_U) != 0 {
		kfree(mem)
		return false
	}
	memmove(mem, src, sz)
	return true
}

func uvmcreate() pagetable_t {
	pagetable := pagetable_t(kalloc())
	if pagetable == 0 {
		return 0
	}
	memset(uintptr(pagetable), 0, uint(PGSIZE))
	return pagetable
}

// 用户 VA 不能在内核页表中直接解引用，先找到可读用户页的物理地址。
func walkaddr(pagetable pagetable_t, va uintptr) uintptr {
	if va >= MAXVA {
		return 0
	}
	pte := walk(pagetable, va, false)
	if pte == nil || *pte&(PTE_V|PTE_U|PTE_R) != (PTE_V|PTE_U|PTE_R) {
		return 0
	}
	return PTE2PA(*pte)
}

// 也用于创建失败后的回滚，因此允许遇到尚未建立的映射。
func uvmunmap(pagetable pagetable_t, va uintptr, npages uintptr, do_free bool) {
	if va%PGSIZE != 0 {
		panic("uvmunmap: not aligned")
	}
	for i := uintptr(0); i < npages; i++ {
		pte := walk(pagetable, va+i*PGSIZE, false)
		if pte == nil || *pte&PTE_V == 0 {
			continue
		}
		if *pte&(PTE_R|PTE_W|PTE_X) == 0 {
			panic("uvmunmap: not a leaf")
		}
		if do_free {
			kfree(PTE2PA(*pte))
		}
		*pte = 0
	}
}

// 所有叶子映射移除后，递归释放三级页表自身。
func freewalk(pagetable pagetable_t) {
	for i := uintptr(0); i < 512; i++ {
		pte := (*pte_t)(unsafe.Pointer(uintptr(pagetable) + i*8))
		if *pte&PTE_V == 0 {
			continue
		}
		if *pte&(PTE_R|PTE_W|PTE_X) != 0 {
			panic("freewalk: leaf")
		}
		freewalk(pagetable_t(PTE2PA(*pte)))
		*pte = 0
	}
	kfree(uintptr(pagetable))
}

func proc_freepagetable(pagetable pagetable_t, sz uintptr) {
	// trampoline 属于内核，trapframe 由 freeproc 另外释放。
	uvmunmap(pagetable, TRAMPOLINE, 1, false)
	uvmunmap(pagetable, TRAPFRAME, 1, false)
	uvmunmap(pagetable, 0, (sz+PGSIZE-1)/PGSIZE, true)
	freewalk(pagetable)
}
