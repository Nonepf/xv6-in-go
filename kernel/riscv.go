package main

import _ "unsafe"

// Supervisor Status Register, sstatus
const (
    SSTATUS_SPP  uintptr = 1 << 8    // Previous mode, 1=Supervisor, 0=User
    SSTATUS_SPIE uintptr = 1 << 5    // Supervisor Previous Interrupt Enable
    SSTATUS_UPIE uintptr = 1 << 4    // User Previous Interrupt Enable
    SSTATUS_SIE  uintptr = 1 << 1    // Supervisor Interrupt Enable
    SSTATUS_UIE  uintptr = 1 << 0    // User Interrupt Enable
)

const PGSIZE = uintptr(4096)
const MAXVA = uintptr(1) << 38

const (
    PTE_V = 1 << 0 // Valid
    PTE_R = 1 << 1 // Readable
    PTE_W = 1 << 2 // Writable
    PTE_X = 1 << 3 // Executable
    PTE_U = 1 << 4 // User
    PTE_G = 1 << 5 // Global
    PTE_A = 1 << 6 // Accessed
    PTE_D = 1 << 7 // Dirty
)

type pte_t uintptr
type pagetable_t uintptr

func PX(level int, va uintptr) uintptr { return (va >> (12 + uintptr(level)*9)) & 0x1FF }
func PTE2PA(pte pte_t) uintptr { return (uintptr(pte) >> 10) << 12 }
func PA2PTE(pa uintptr) pte_t { return pte_t((pa >> 12) << 10) }

//func PGGROUNDDOWN(a uintptr) uintptr { return a - a % PGSIZE }
func PGGROUNDDOWN(a uintptr) uintptr { return a & ^(PGSIZE - 1) }

const (
    SATP_SV39 uintptr = 8 << 60
)

func MAKE_SATP(pagetable pagetable_t) uintptr {
    ppn := uintptr(pagetable) >> 12
    return SATP_SV39 | ppn
}

//go:linkname intr_on intr_on
func intr_on()

//go:linkname intr_off intr_off
func intr_off()

//go:linkname r_sip r_sip
func r_sip() uintptr

//go:linkname w_sip w_sip
func w_sip(x uintptr)

//go:linkname r_scause r_scause
func r_scause() uintptr

//go:linkname r_sepc r_sepc
func r_sepc() uintptr

//go:linkname w_sepc w_sepc
func w_sepc(x uintptr)

//go:linkname r_sstatus r_sstatus
func r_sstatus() uintptr

//go:linkname w_sstatus w_sstatus
func w_sstatus(x uintptr)

//go:linkname w_stvec w_stvec
func w_stvec(x uintptr)

//go:linkname r_satp r_satp
func r_satp() uintptr

//go:linkname w_sie w_sie
func w_sie(x uintptr) 