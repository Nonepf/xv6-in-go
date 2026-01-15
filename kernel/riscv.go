package main

import _ "unsafe"

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