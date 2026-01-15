#ifndef MEMLAYOUT_H
#define MEMLAYOUT_H

// core local interruptor (CLINT), which contains the timer.
#define CLINT 0x2000000
#define CLINT_MTIME (CLINT + 0xBFF8)
#define CLINT_MTIMECMP (CLINT + 0x4000)

#endif