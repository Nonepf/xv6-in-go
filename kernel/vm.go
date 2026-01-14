package main

import "unsafe"

//go:external kernel_pagetable
var kernel_pagetable pagetable_t

//go:linkname get_etext get_etext
func get_etext() uintptr

func kvminit() {
	kernel_pagetable = pagetable_t(kalloc())
	printf("kernel_pagetable at %x\n", uintptr(kernel_pagetable))
	memset(uintptr(kernel_pagetable), 0, uint(PGSIZE))

	kvmmap(UART0, UART0, PGSIZE, PTE_R | PTE_W)
	kvmmap(VIRTIO0, VIRTIO0, PGSIZE, PTE_R | PTE_W)
	kvmmap(PLIC, PLIC, 0x400000, PTE_R | PTE_W)
	kvmmap(KERNBASE, KERNBASE, get_etext() - KERNBASE, PTE_R | PTE_X)
	kvmmap(get_etext(), get_etext(), PHYSTOP - get_etext(), PTE_R | PTE_W)
	//kvmmap(TRAMPOLINE)
}

//go:linkname kvminithart kvminithart
func kvminithart()

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
	a := PGGROUNDDOWN(va)
	last := PGGROUNDDOWN(va + size - 1)
	for {
		pte := walk(pagetable, a, true)
		if pte == nil {
			return -1
		}
		if *pte & PTE_V != 0 {
			panic("remap")
		}
		*pte = PA2PTE(pa) | pte_t(perm | PTE_V)
		if a == last {
			break
		}
		a += PGSIZE
		pa += PGSIZE
	}
	return 0
}