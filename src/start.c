#include <stdint.h>
#include <stddef.h>
#include "riscv.h"
#include "output.h"
#include "memlayout.h"

extern void timervec();
void timerinit();

uint64 timer_scratch[5];

// transmission layer
void c_start() {
    // set to s-mode
    unsigned long x = r_mstatus();
    x &= ~MSTATUS_MPP_MASK;
    x |= MSTATUS_MPP_S;
    w_mstatus(x);

    // M Exception Counter
    extern void KMain();
    w_mepc((uint64)KMain);

    // disable paging
    w_satp(0);

    // delegate all to s-mode
    w_medeleg(0xffff);
    w_mideleg(0xffff);
    w_sie(r_sie() | SIE_SEIE | SIE_STIE | SIE_SSIE);

    w_pmpaddr0(0x3fffffffffffffull);
    w_pmpcfg0(0xf);

    timerinit();

    uart_putc('C'); uart_putc(' '); uart_putc('O'); uart_putc('K'); uart_putc('\n');
    asm volatile("mret");
}

// initialize timer interrupts
void timerinit() {
    int interval = 100000; // cycles
    
    *(uint64*)CLINT_MTIMECMP = *(uint64*)CLINT_MTIME + interval;

    uint64 *scratch = timer_scratch;
    scratch[3] = CLINT_MTIMECMP;
    scratch[4] = interval;
    w_mscratch((uint64)scratch);

    w_mtvec((uint64)timervec);
    w_mstatus(r_mstatus() | MSTATUS_MIE);
    w_mie(r_mie() | MIE_MTIE);
}