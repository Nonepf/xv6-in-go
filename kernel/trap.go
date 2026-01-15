package main

import _ "unsafe"

//go:linkname trapinithart trapinithart
func trapinithart()

//go:nosplit
//export Kerneltrap
func Kerneltrap() {
	printf("tick\n")
}