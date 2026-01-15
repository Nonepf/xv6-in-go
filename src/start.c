#include <stdint.h>
#include <stddef.h>
#include "riscv.h"
#include "output.h"

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

    uart_putc('C'); uart_putc(' '); uart_putc('O'); uart_putc('K'); uart_putc('\n');
    asm volatile("mret");
}


