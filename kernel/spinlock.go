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

func initlock(lk *spinlock) {
	lk.locked = 0
}

func acquire(lk *spinlock) {
	intr_off()
	for sync_test_and_set(&lk.locked) == 1 {}
	sync_barrier()
}

func release(lk *spinlock) {
	sync_release(&lk.locked)
	intr_on()
}