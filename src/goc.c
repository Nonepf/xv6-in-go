#include <stdint.h>
#include <stddef.h>
#include "riscv.h"

// constants in kernel.ld
extern char end[];
uintptr_t get_end(void) { return (uintptr_t)end; }

extern char etext[];
uintptr_t get_etext(void) { return (uintptr_t)etext; }

//vm support
unsigned long kernel_pagetable;

void kvminithart(uint64_t kernel_pagetable) {
    // Sv39 mode + Physical Page Number
    uint64_t x = SATP_SV39 | (((uint64_t)kernel_pagetable) >> 12);
    w_satp(x);
    sfence_vma();
}