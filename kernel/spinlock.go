package main

import _ "unsafe"

//go:linkname sync_barrier sync_barrier
func sync_barrier()

//go:linkname sync_test_and_set sync_test_and_set
func sync_test_and_set(addr *uint32) uint32

//go:linkname sync_release sync_release
func sync_release(addr *uint32)

type spinlock struct {
	locked uint32
}

// 单核 CPU 的关中断层数，以及最外层关中断前的状态。
var noff int
var intena bool

func push_off() {
	enabled := r_sstatus()&SSTATUS_SIE != 0
	intr_off()
	if noff == 0 {
		intena = enabled
	}
	noff++
}

func pop_off() {
	if r_sstatus()&SSTATUS_SIE != 0 || noff < 1 {
		panic("pop_off")
	}
	noff--
	if noff == 0 && intena {
		intr_on()
	}
}

func initlock(lk *spinlock) {
	lk.locked = 0
}

func acquire(lk *spinlock) {
	push_off()
	for sync_test_and_set(&lk.locked) == 1 {
	}
	sync_barrier()
}

func release(lk *spinlock) {
	sync_release(&lk.locked)
	pop_off()
}
