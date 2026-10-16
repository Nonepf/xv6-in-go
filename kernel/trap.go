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
	scause := r_scause()
	sepc := r_sepc()
	sstatus := r_sstatus()
	if sstatus&SSTATUS_SPP == 0 || sstatus&SSTATUS_SIE != 0 {
		panic("Kerneltrap: bad status")
	}

	// timer interrupt
	if timerintr(scause) {
		if current_proc != nil && current_proc.state == RUNNING {
			yield(current_proc)
		}
	} else {
		printf("Kerneltrap %x at %x, stval %x\n", scause, sepc, r_stval())
		for {
		}
	}
	// yield 期间其他进程可能改过这些 CPU 寄存器。
	intr_off()
	w_sepc(sepc)
	w_sstatus(sstatus)
}

func timerintr(scause uintptr) bool {
	// timervec 将 M-mode 时钟中断转发为 S-mode 软件中断。
	if scause == 0x8000000000000001 {
		w_sip(r_sip() & ^uintptr(2))
		return true
	}
	return false
}

//go:nosplit
//export Usertrap
func Usertrap() {
	if r_sstatus()&SSTATUS_SPP != 0 {
		panic("Usertrap: not from user")
	}
	// 此后在内核中发生的中断，必须走 kernelvec。
	trapinithart()
	p := current_proc
	p.trapframe.epc = r_sepc()
	scause := r_scause()
	if scause == 8 {
		// ecall 是 4 字节指令，返回时应执行下一条指令。
		p.trapframe.epc += 4
		intr_on()
		syscall(p)
	} else if timerintr(scause) {
		yield(p)
	} else {
		printf("Usertrap: pid %d, scause %x, sepc %x, stval %x\n",
			p.pid, scause, p.trapframe.epc, r_stval())
		exitproc(-1)
	}
	Usertrapret()
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
