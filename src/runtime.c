#include <stdint.h>
#include <stddef.h>
#include "output.h"

typedef long int ssize_t;

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

// abort
void abort() { 
    while(1); 
}

ssize_t write(int fd, const void *buf, size_t count) {
    for(size_t i = 0; i < count; i++) 
        uart_putc(((char*)buf)[i]);
    return count;
}