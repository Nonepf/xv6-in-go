package main

import "unsafe"

func memset(dst uintptr, c int, n uint) {
	for i := uint(0); i < n; i++ {
		*(*byte)(unsafe.Pointer(dst + uintptr(i))) = byte(c)
	}
}

func memmove(dst uintptr, src uintptr, n uintptr) {
	s := src
	d := dst

	if (s < d && s + n > d) {
		s += n
		d += n
		for ; n > 0; n-- {
			d--
			s--
			*(*byte)(unsafe.Pointer(d)) = *(*byte)(unsafe.Pointer(s))
		}
	} else {
		for ; n > 0; n-- {
			*(*byte)(unsafe.Pointer(d)) = *(*byte)(unsafe.Pointer(s))
			d++
			s++
		}
	}
}