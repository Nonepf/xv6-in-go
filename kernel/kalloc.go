package main

import "unsafe"

//go:linkname get_end get_end
func get_end() uintptr

const PGSIZE = 4096

type run struct {
	next *run
}

type Kmem struct {
	freelist *run
	//lock	spinlock	
}

var kmem Kmem

func kinit() {
	// initlock(&kmem.lock, "kmem")
	var BSS_END uintptr = get_end()
	printf("kinit: [%d, %d)\n", int(BSS_END), int(PHYSTOP))
	freerange(BSS_END, PHYSTOP)
}

func freerange(pa_start uintptr, pa_end uintptr) {
	printf("freerange: [%d, %d)\n", int(pa_start), int(pa_end))
	for p := pa_start; p + PGSIZE <= pa_end; p += PGSIZE {
		kfree(p)
	}
}

func kfree(pa uintptr) {
	// kmem.lock.acquire()
	// defer kmem.lock.acquire()
	var BSS_END uintptr = get_end()

	if pa % PGSIZE != 0 || pa < BSS_END || pa >= PHYSTOP {
		printf("panic: kfree")
		for {}
	}

	r := (*run)(unsafe.Pointer(pa))
	r.next = kmem.freelist;
	kmem.freelist = r;
}

func kalloc() uintptr {
	// kmem.lock.acquire()
	// defer kmem.lock.acquire()

	r := kmem.freelist
	if r != nil {
		kmem.freelist = r.next
		return uintptr(unsafe.Pointer(r))
	}
	return 0
}