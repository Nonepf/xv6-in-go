OUTPUT_ARCH(riscv) /* target ISA */
ENTRY(_entry) /* the entry of the whole system */

/* define the arrangement of each section */
SECTIONS { 
    /* location counter, every thing starts here */
    . = 0x80000000; 

    /* a section for code, including entry.S */
    .text : {
        *(.text .text.*)
    }

    .bss : {
        /* we define _bss_start and _bss_end to record the location of
            bss sections so that we can initialize its contents later */
        _bss_start = .;
        *(.bss .bss.*)
        _bss_end = .;
    }
}