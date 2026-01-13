OUTPUT_ARCH(riscv)
ENTRY(_entry)

SECTIONS { 
    . = 0x80000000; 

   .text : {
        *(.text .text.*)
        . = ALIGN(4096);
    }

    .bss : {
        _bss_start = .;
        *(.bss .bss.*)
        . = ALIGN(4096);
        _bss_end = .;
    }
}