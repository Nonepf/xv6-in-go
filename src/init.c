#include <stdint.h>
#include <stddef.h>

typedef long int ssize_t;

// constants in kernel.ld
extern char end[];
uintptr_t get_end(void) { return (uintptr_t)end; }

extern char etext[];
uintptr_t get_etext(void) { return (uintptr_t)etext; }

// basic IO
// basic output
void uart_putc(char c) {
    volatile uint8_t *uart = (uint8_t *)0x10000000;
    *uart = c;
}

// runtime support
// allocate a 64KB heap for go runtime, make sure it can work
static uint8_t go_heap[64 * 1024];
uintptr_t runtime_heapMaxSize = (uintptr_t)sizeof(go_heap);

// simulate mmap
void* mmap(void* addr, size_t length, int prot, int flags, int fd, int64_t offset) {
    return (void*)go_heap;
}

// just a infinite loop
void runtime_exit(int code) { 
    for(;;);
}

// just a infinite loop, cator for runtime's interface demand
void runtime_abort() {
    for(;;);
}

// transmission layer
void c_start() {
    extern void KMain();
    
    uart_putc('C'); uart_putc(' '); uart_putc('O'); uart_putc('K'); uart_putc('\n');
    
    KMain();
}

// abort
void abort() { 
    while(1); 
}

ssize_t write(int fd, const void *buf, size_t count) {
    for(size_t i = 0; i < count; i++) 
        uart_putc(((char*)buf)[i]);
    return count;
}

//vm support
unsigned long kernel_pagetable;

void kvminithart() {
    // Sv39 mode + Physical Page Number
    uint64_t x = (8L << 60) | (((uint64_t)kernel_pagetable) >> 12);
    asm volatile("csrw satp, %0" : : "r" (x));
    asm volatile("sfence.vma zero, zero");
}