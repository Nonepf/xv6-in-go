#ifndef OUTPUT_H
#define OUTPUT_H

#include <stdint.h>
#include <stddef.h>

// basic IO
void uart_putc(char c) {
    volatile uint8_t *uart = (uint8_t *)0x10000000;
    *uart = c;
}

#endif