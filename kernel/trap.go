package main

import _ "unsafe"

//go:linkname trapinithart trapinithart
func trapinithart()

//go:linkname get_uservec get_uservec
func get_uservec() uintptr

//go:linkname get_userret get_userret
func get_userret() uintptr

//go:linkname GetUsertrapAddr GetUsertrapAddr
func GetUsertrapAddr() uintptr

//go:linkname trampoline_call trampoline_call
func trampoline_call(fn, trapframe, satp uintptr)

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

//go:nosplit
//export Usertrap
func Usertrap() {

}

//go:nosplit
//export Usertrapret
func Usertrapret() {
    intr_off()

    w_stvec(TRAMPOLINE + (get_uservec() - get_trampoline()))

    p := current_proc
    p.trapframe.kernel_satp = r_satp()
    p.trapframe.kernel_sp = p.kstack + PGSIZE
    p.trapframe.kernel_trap = GetUsertrapAddr()
    p.trapframe.kernel_hartid = 0 // not implemented yet

    x := r_sstatus()
    x &= ^SSTATUS_SPP
    x |= SSTATUS_SPIE
    w_sstatus(x)

    w_sepc(p.trapframe.epc)

    satp := MAKE_SATP(p.pagetable)
    
    fn := TRAMPOLINE + (get_userret() - get_trampoline())
    trampoline_call(fn, TRAPFRAME, satp)
}