package main

import "unsafe"

// from cgo
//
//go:linkname get_trampoline get_trampoline
func get_trampoline() uintptr

//go:linkname get_initcode get_initcode
func get_initcode() uintptr

//go:linkname get_user_proc_size get_user_proc_size
func get_user_proc_size() uintptr

//go:linkname GetForkretAddr GetForkretAddr
func GetForkretAddr() uintptr

const NPROC = 8

type procstate int

const (
	UNUSED   procstate = iota // 0
	USED                      // 1
	SLEEPING                  // 2
	RUNNABLE                  // 3
	RUNNING                   // 4
	ZOMBIE                    // 5
)

type Context struct {
	ra uintptr
	sp uintptr

	// callee-saved
	s0  uintptr
	s1  uintptr
	s2  uintptr
	s3  uintptr
	s4  uintptr
	s5  uintptr
	s6  uintptr
	s7  uintptr
	s8  uintptr
	s9  uintptr
	s10 uintptr
	s11 uintptr

	// go
	gp uintptr
	tp uintptr
}

type Trapframe struct {
	/*   0 */ kernel_satp uintptr // kernel page table
	/*   8 */ kernel_sp uintptr // top of process's kernel stack
	/*  16 */ kernel_trap uintptr // usertrap()
	/*  24 */ epc uintptr // saved user program counter
	/*  32 */ kernel_hartid uintptr // saved kernel tp
	/*  40 */ ra uintptr
	/*  48 */ sp uintptr
	/*  56 */ gp uintptr
	/*  64 */ tp uintptr
	/*  72 */ t0 uintptr
	/*  80 */ t1 uintptr
	/*  88 */ t2 uintptr
	/*  96 */ s0 uintptr
	/* 104 */ s1 uintptr
	/* 112 */ a0 uintptr
	/* 120 */ a1 uintptr
	/* 128 */ a2 uintptr
	/* 136 */ a3 uintptr
	/* 144 */ a4 uintptr
	/* 152 */ a5 uintptr
	/* 160 */ a6 uintptr
	/* 168 */ a7 uintptr
	/* 176 */ s2 uintptr
	/* 184 */ s3 uintptr
	/* 192 */ s4 uintptr
	/* 200 */ s5 uintptr
	/* 208 */ s6 uintptr
	/* 216 */ s7 uintptr
	/* 224 */ s8 uintptr
	/* 232 */ s9 uintptr
	/* 240 */ s10 uintptr
	/* 248 */ s11 uintptr
	/* 256 */ t3 uintptr
	/* 264 */ t4 uintptr
	/* 272 */ t5 uintptr
	/* 280 */ t6 uintptr
}

type Proc struct {
	lock spinlock

	// p->lock must be held when using these:
	state  procstate // Process state
	pid    int       // Process ID
	xstate int       // Exit status (no wait syscall yet)

	// p->lock needn't be held since they are private to Proc
	kstack    uintptr
	context   Context
	trapframe *Trapframe
	pagetable pagetable_t
	sz        uintptr  // Size of user memory
	name      [16]byte // Process name (debugging)
	task      func()
}

var proc [NPROC]Proc
var cpu_context Context
var current_proc *Proc
var nextpid int

// before turn on time interrupt
func procinit() {
	current_proc = nil
	nextpid = 1
	for i := 0; i < NPROC; i++ {
		p := &proc[i]
		initlock(&p.lock)

		kstack := kalloc()
		if kstack == 0 {
			panic("procinit: kalloc failed")
		}

		kvmmap(KSTACK(i), kstack, PGSIZE, PTE_R|PTE_W)
		p.kstack = KSTACK(i)

		p.state = UNUSED
	}
}

func scheduler() {
	for {
		intr_on()
		for i := 0; i < NPROC; i++ {
			p := &proc[i]
			acquire(&p.lock)
			if p.state == RUNNABLE {
				p.state = RUNNING
				current_proc = p

				swtch(&cpu_context, &p.context)

				current_proc = nil
				// 已切回调度器栈，现在才能回收退出进程的用户内存。
				if p.state == ZOMBIE {
					freeproc(p)
				}
			}
			release(&p.lock)
		}
	}
}

//go:linkname swtch swtch
func swtch(old *Context, new *Context)

func yield(p *Proc) {
	acquire(&p.lock)
	p.state = RUNNABLE
	sched()
	release(&p.lock)
}

// 调用者持有 p.lock，且已经将进程改为非 RUNNING 状态。
func sched() {
	p := current_proc
	if p.lock.locked == 0 || noff != 1 || p.state == RUNNING || r_sstatus()&SSTATUS_SIE != 0 {
		panic("sched: bad state")
	}
	// 关中断前的状态属于当前执行流，不能被其他进程覆盖。
	enabled := intena
	swtch(&p.context, &cpu_context)
	intena = enabled
}

func exitproc(status int) {
	p := current_proc
	acquire(&p.lock)
	p.xstate = status
	p.state = ZOMBIE
	sched()
	panic("exitproc: returned")
}

// p.lock 必须持有；内核栈预先分配给进程槽，留待下次复用。
func freeproc(p *Proc) {
	if p.pagetable != 0 {
		proc_freepagetable(p.pagetable, p.sz)
	}
	if p.trapframe != nil {
		kfree(uintptr(unsafe.Pointer(p.trapframe)))
	}
	p.trapframe = nil
	p.pagetable = 0
	p.sz = 0
	p.pid = 0
	p.xstate = 0
	p.state = UNUSED
}

func allocproc() *Proc {
	var p *Proc
	for i := 0; i < NPROC; i++ {
		p = &proc[i]
		acquire(&p.lock)
		if p.state == UNUSED {
			goto found
		} else {
			release(&p.lock)
		}
	}
	return nil

found:
	p.state = USED
	p.pid = nextpid
	nextpid++

	ptr := kalloc()
	if ptr == 0 {
		freeproc(p)
		release(&p.lock)
		return nil
	}
	memset(ptr, 0, uint(PGSIZE))
	p.trapframe = (*Trapframe)(unsafe.Pointer(ptr))

	p.pagetable = proc_pagetable(p)
	if p.pagetable == 0 {
		freeproc(p)
		release(&p.lock)
		return nil
	}

	//p.context = Context{} // seem to be better way in Go
	memset(uintptr(unsafe.Pointer(&p.context)), 0, uint(unsafe.Sizeof(p.context)))

	p.context.ra = GetForkretAddr()
	p.context.sp = p.kstack + PGSIZE
	return p
}

func proc_pagetable(p *Proc) pagetable_t {
	var pagetable pagetable_t
	pagetable = uvmcreate()
	if pagetable == 0 {
		return 0
	}

	if mappages(pagetable, TRAMPOLINE, PGSIZE, get_trampoline(), PTE_R|PTE_X) != 0 ||
		mappages(pagetable, TRAPFRAME, PGSIZE, uintptr(unsafe.Pointer(p.trapframe)), PTE_R|PTE_W) != 0 {
		proc_freepagetable(pagetable, 0)
		return 0
	}
	return pagetable
}

//export Forkret
func Forkret() {
	release(&current_proc.lock)
	Usertrapret()
}

func KSTACK(i int) uintptr {
	// TRAMPOLINE
	// Process 0 Stack
	// Process 0 Guard Page
	// Process 1 Stack
	// Process 1 Guard Page
	// ...
	return TRAMPOLINE - uintptr(i+1)*2*PGSIZE
}

// first user process
func userinit() {
	var p *Proc
	p = allocproc()
	if p == nil {
		panic("userinit: allocproc failed")
	}

	printf("uvminit %x %x\n", get_initcode(), get_user_proc_size())
	if !uvminit(p.pagetable, get_initcode(), get_user_proc_size()) {
		freeproc(p)
		release(&p.lock)
		panic("userinit: uvminit failed")
	}
	p.sz = PGSIZE

	p.trapframe.epc = 0
	p.trapframe.sp = PGSIZE
	p.state = RUNNABLE

	release(&p.lock)
}
