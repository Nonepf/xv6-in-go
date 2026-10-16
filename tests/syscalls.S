# 独立用户程序：检查系统调用返回值、错误参数和用户寄存器恢复。
.section .user_code, "ax", @progbits
.option norelax
.option nopic
.global user_proc
user_proc:
    li s0, -1
    li s1, 0x123
    li t3, 0x456

    # 未知系统调用。
    li a7, 99
    ecall
    bne a0, s0, fail

    # 不支持的 fd。
    li a0, 0
    la a1, ok
    li a2, 1
    li a7, 1
    ecall
    bne a0, s0, fail

    # 内核地址、trapframe、超出 MAXVA 的地址。
    li a0, 1
    li a1, 0x80000000
    li a2, 1
    ecall
    bne a0, s0, fail
    li a0, 1
    li a1, 0x3fffffe000
    ecall
    bne a0, s0, fail
    li a0, 1
    li a1, 0x4000000000
    ecall
    bne a0, s0, fail

    # 跨越用户页末尾，以及可能发生整数溢出的范围。
    li a0, 1
    li a1, 4095
    li a2, 2
    ecall
    bne a0, s0, fail
    li a0, 1
    li a1, -8
    li a2, 16
    ecall
    bne a0, s0, fail
    li a0, 1
    li a1, 0
    li a2, -1
    ecall
    bne a0, s0, fail

    # 零长度写不访问缓冲区。
    li a0, 1
    li a1, -1
    li a2, 0
    ecall
    bnez a0, fail

    li t0, 0x123
    bne s1, t0, fail
    li t0, 0x456
    bne t3, t0, fail

    # stderr 可写；返回值应等于实际写入长度。
    li a0, 2
    la a1, ok
    la a2, ok_end
    sub a2, a2, a1
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
    .ascii "syscalls: OK\n"
ok_end:
bad:
    .ascii "FAIL\n"
