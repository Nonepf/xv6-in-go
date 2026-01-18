package main

import _ "unsafe"

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
    s0 uintptr
    s1 uintptr
    s2 uintptr
    s3 uintptr
    s4 uintptr
    s5 uintptr
    s6 uintptr
    s7 uintptr
    s8 uintptr
    s9 uintptr
    s10 uintptr
    s11 uintptr

    // go
    gp uintptr
    tp uintptr
}

type KProc struct {
    lock spinlock

    // p->lock must be held when using these:
    state procstate         // Process state
    pid int                 // Process ID

    // p->lock needn't be held since they are private to kproc
    kstack uintptr
    context Context
    name [16]byte           // Process name (debugging)
    task func()
}

var proc [NPROC]KProc
var cpu_context Context
var current_proc *KProc

// before turn on time interrupt
func procinit() {
    current_proc = nil
	for i := 0; i < NPROC; i++ {
        p := &proc[i]
		initlock(&p.lock)
        
        kstack := kalloc() 
        if kstack == 0 {
            panic("procinit: kalloc failed")
        }

        kvmmap(KSTACK(i), kstack, PGSIZE, PTE_R | PTE_W)
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
            }
            release(&p.lock)
        }
    }
}

//go:linkname swtch swtch
func swtch(old *Context, new *Context)

func yield(p *KProc) {
    acquire(&p.lock)
    p.state = RUNNABLE
    swtch(&p.context, &cpu_context)
    release(&p.lock)
}

//go:linkname GetTaskStubAddr GetTaskStubAddr
func GetTaskStubAddr() uintptr

func allocProc(fun func()) *KProc {
    var p *KProc
    for i := 0; i < NPROC; i++ {
        p = &proc[i]
        acquire(&p.lock)
        if p.state == UNUSED {
            goto found
        }
        release(&p.lock)
    }
    return nil

found:
    p.pid = 0   // Not implemented yet
    p.state = RUNNABLE
    p.task = fun
    p.context.ra = GetTaskStubAddr()
    p.context.sp = p.kstack + PGSIZE

    release(&p.lock)
    return p
}

//export TaskStub
func TaskStub() {
    release(&current_proc.lock)
    printf("STUB START\n")
    intr_on()
    if current_proc.task != nil {
        current_proc.task()
    }
    
    panic("task returned")
}

func KSTACK(i int) uintptr {
    // TRAMPOLINE
	// Process 0 Stack
	// Process 0 Guard Page
	// Process 1 Stack
	// Process 1 Guard Page
	// ...
    return TRAMPOLINE - uintptr(i+1) * 2 * PGSIZE
}