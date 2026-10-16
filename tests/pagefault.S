.section .user_code, "ax", @progbits
.global user_proc
user_proc:
    # 用户态不能访问内核地址。
    li t0, 0x80000000
    ld a0, 0(t0)
    unimp
