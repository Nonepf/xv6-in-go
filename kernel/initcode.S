.section .user_code, "ax", @progbits
# 代码会从内核地址复制到 VA 0，地址必须保持 PC-relative。
.option norelax
.option nopic
.global user_proc
user_proc:
    # write(1, msg, msg_end - msg)
    li a0, 1
    la a1, msg
    la a2, msg_end
    sub a2, a2, a1
    li a7, 1                 # SYS_write
    ecall

    # write 返回用户态后，再执行 exit(0)。
    li a0, 0
    li a7, 2                 # SYS_exit
    ecall

    # exit 不应返回。
    unimp

msg:
    .ascii "hello from user\n"
msg_end:
