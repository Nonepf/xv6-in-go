#include <stdint.h>
#include <stddef.h>
#include "riscv.h"

// constants in kernel.ld
extern char end[];
uintptr_t get_end(void) { return (uintptr_t)end; }

extern char etext[];
uintptr_t get_etext(void) { return (uintptr_t)etext; }

// vm support
unsigned long kernel_pagetable;

void kvminithart(uint64_t kernel_pagetable) {
    // Sv39 mode + Physical Page Number
    uint64_t x = SATP_SV39 | (((uint64_t)kernel_pagetable) >> 12);
    w_satp(x);
    sfence_vma();
}

// trap support

void trapinithart() {
    extern void kernelvec();
    w_stvec((uint64)kernelvec);
}

// spinlock support
int sync_test_and_set(volatile int *addr) {
    return __sync_lock_test_and_set(addr, 1);
}

void sync_barrier() {
    __sync_synchronize();
}

void sync_release(volatile int *addr) {
    __sync_lock_release(addr);
}