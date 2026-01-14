package main

import "unsafe"

func memset(dst uintptr, c int, n uint) {
	for i := uint(0); i < n; i++ {
		*(*byte)(unsafe.Pointer(dst + uintptr(i))) = byte(c)
	}
}