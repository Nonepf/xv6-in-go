.section .text
.global _entry
_entry:
    la sp, stack0       # sp pointer
    li a0, 4096
    add sp, sp, a0      # set sp to its starting point
    call c_start

.section .bss
.align 16
stack0: .space 4096     # spare a 4096-byte room