package main

import _ "unsafe"

//go:linkname trapinithart trapinithart
func trapinithart()

//go:nosplit
//export Kerneltrap
func Kerneltrap() {
	w_sip(r_sip() & ^uintptr(2))

    scause := r_scause()
	sepc := r_sepc()

	// timer interrupt
    if scause == 0x8000000000000005 || scause == 0x8000000000000001 {
        if current_proc != nil && current_proc.state == RUNNING {
            yield(current_proc)
        }
    } else {
        printf("Kerneltrap %x at %x\n", scause, sepc)
        for {}
    }
}