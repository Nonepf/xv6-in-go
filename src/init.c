#include <stdint.h>
#include <stddef.h>

typedef long int ssize_t;

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
    uart_putc('p'); uart_putc('a'); uart_putc('n'); uart_putc('i'); uart_putc('c'); uart_putc('\n');
    while(1); 
}

ssize_t write(int fd, const void *buf, size_t count) {
    for(size_t i = 0; i < count; i++) 
        uart_putc(((char*)buf)[i]);
    return count;
}