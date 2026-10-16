.section .user_code, "ax", @progbits
.global user_proc
user_proc:
    .word 0xffffffff
