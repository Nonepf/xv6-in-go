# 长时间留在用户态，检查时钟抢占后寄存器和执行位置能恢复。
.section .user_code, "ax", @progbits
.option norelax
.option nopic
.global user_proc
user_proc:
    li a0, 0x123
    li t3, 0x456
    li s1, 0x789
    li s0, 50000000
loop:
    addi s0, s0, -1
    bnez s0, loop
    li t0, 0x123
    bne a0, t0, fail
    li t0, 0x456
    bne t3, t0, fail
    li t0, 0x789
    bne s1, t0, fail

    li a0, 1
    la a1, ok
    la a2, ok_end
    sub a2, a2, a1
    li a7, 1
    ecall
    bne a0, a2, fail
    li a0, 0
    li a7, 2
    ecall
    j fail
fail:
    li a0, 1
    la a1, bad
    li a2, 5
    li a7, 1
    ecall
    li a0, 1
    li a7, 2
    ecall
    unimp
ok:
    .ascii "timer: OK\n"
ok_end:
bad:
    .ascii "FAIL\n"
