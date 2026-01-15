.section .text
.global timervec
.align 4
timervec:
    csrrw a0, mscratch, a0

    # save value temporarily
    sd a1, 0(a0)
    sd a2, 8(a0)
    sd a3, 16(a0)

    ld a1, 24(a0)   # CLINT_MTIMECMP, mscratch[3]
    ld a2, 32(a0)   # interval, mscratch[4]
    ld a3, 0(a1)    # a3 = CLINT_MTIMECMP
    add a3, a3, a2  # a3 = CLINT_MTIMECMP + interval
    sd a3, 0(a1)

    # raise a S-Mode interrupt 
    li a1, 2
    csrw sip, a1

    # recover
    ld a3, 16(a0)
    ld a2, 8(a0)
    ld a1, 0(a0)
    csrrw a0, mscratch, a0

    mret

# a copy from kernelvec.S in xv6
.global kernelvec
.align 4
kernelvec:
        # make room to save registers.
        addi sp, sp, -256

        # save caller-saved registers.
        sd ra, 0(sp)
        sd sp, 8(sp)
        sd gp, 16(sp)
        sd tp, 24(sp)
        sd t0, 32(sp)
        sd t1, 40(sp)
        sd t2, 48(sp)
        sd a0, 72(sp)
        sd a1, 80(sp)
        sd a2, 88(sp)
        sd a3, 96(sp)
        sd a4, 104(sp)
        sd a5, 112(sp)
        sd a6, 120(sp)
        sd a7, 128(sp)
        sd t3, 216(sp)
        sd t4, 224(sp)
        sd t5, 232(sp)
        sd t6, 240(sp)

        call Kerneltrap

        # restore registers.
        ld ra, 0(sp)
        ld sp, 8(sp)
        ld gp, 16(sp)
        # not tp (contains hartid), in case we moved CPUs
        ld t0, 32(sp)
        ld t1, 40(sp)
        ld t2, 48(sp)
        ld a0, 72(sp)
        ld a1, 80(sp)
        ld a2, 88(sp)
        ld a3, 96(sp)
        ld a4, 104(sp)
        ld a5, 112(sp)
        ld a6, 120(sp)
        ld a7, 128(sp)
        ld t3, 216(sp)
        ld t4, 224(sp)
        ld t5, 232(sp)
        ld t6, 240(sp)

        addi sp, sp, 256

        # return to whatever we were doing in the kernel.
        sret
