OUTPUT_ARCH(riscv)
ENTRY(_entry)

SECTIONS { 
    . = 0x80000000; 

    .text : {
        *(.text .text.*)
        . = ALIGN(4096);
        PROVIDE(etext = .);
    }

    . = ALIGN(4096);
    .trampsec : {
        *(.trampsec)
    }
    . = ALIGN(4096);
    PROVIDE(trampoline_end = .);

    .rodata : {
        *(.rodata .rodata.*)
    }

    .data : {
        *(.data .data.*)
    }

    . = ALIGN(4096);
    .user_init : {
        PROVIDE(user_proc_start = .);
        *(.user_code)        
        . = ALIGN(4096);
        PROVIDE(user_proc_end = .);
    }

    .bss : {
        *(.bss .bss.*)
        . = ALIGN(4096);
        PROVIDE(end = .);
    }
}