OUTPUT_ARCH(riscv)
ENTRY(_entry)

SECTIONS { 
    . = 0x80000000; 

   .text : {
        *(.text .text.*)
        . = ALIGN(4096);
        etext = .;
    }

    .bss : {
        *(.bss .bss.*)
        . = ALIGN(4096);
        end = .;
    }
}