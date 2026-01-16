package main

import _ "unsafe"

//go:linkname trapinithart trapinithart
func trapinithart()

//go:nosplit
//export Kerneltrap
func Kerneltrap() {
	w_sip(r_sip() & ^uintptr(2))
	if addLimit < 1000 {
		//acquire(&count.lock)
		count.num++
		addLimit++
		//release(&count.lock)
	}
}