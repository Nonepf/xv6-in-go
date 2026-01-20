.section .user_code
.global user_proc
user_proc:
loop:
    li a0, 0x123
    j loop