# void trampoline_call(uintptr_t fn, uintptr_t trapframe_va, uintptr_t satp);
.section .text
.global trampoline_call
trampoline_call:
    # a0: fn
    # a1: trapframe_va
    # a2: satp
    
    mv t0, a0  
    mv a0, a1   
    mv a1, a2
    
    jr t0    