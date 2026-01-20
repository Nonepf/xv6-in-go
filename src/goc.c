#include <stdint.h>
#include <stddef.h>
#include "riscv.h"

// constants in kernel.ld
extern char end[];
uintptr_t get_end(void) { return (uintptr_t)end; }

extern char etext[];
uintptr_t get_etext(void) { return (uintptr_t)etext; }

extern char trampoline[];
uintptr_t get_trampoline(void) { return (uintptr_t)trampoline; }

extern char user_proc[];
uintptr_t get_initcode(void) { return (uintptr_t)user_proc; }

extern char user_proc_start[];
extern char user_proc_end[];
uintptr_t get_user_proc_size() { return (uintptr_t)user_proc_end - (uintptr_t)user_proc_start; }

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

// proc support
extern char uservec[];
uintptr_t get_uservec() {
    return (uintptr_t)uservec;
}

extern char userret[];
uintptr_t get_userret() {
    return (uintptr_t)userret;
}