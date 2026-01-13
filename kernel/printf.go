package main

import (
	_ "runtime"
	_ "unsafe"
)

//go:linkname uart_putc uart_putc
func uart_putc(c byte)

func printInt(num int) {
	// Int in Go ranges from -9,223,372,036,854,775,808
	//					 to   9,223,372,036,854,775,807.
	// We need roughly 20 bytes to store it.
	var buf	[20]byte
	i := 0
	
	if num < 0 {
		uart_putc('-')
		num = -num
	}

	for num > 0 {
		buf[i] = byte(num % 10) + '0'
		i++
		num = num / 10
	}

	for i = i - 1; i >= 0; i-- {
		uart_putc(buf[i])
	}
}

func printString(str string) {
	for _, c := range str {
		uart_putc(byte(c))
	}
}

func printf(format string, args ...interface{}) {
	argIdx := 0
	for i := 0; i < len(format); i++ {
		if (format[i] == '%' && i+1 < len(format)) {
			i++
			switch format[i] {
			case 'd':
				printInt(args[argIdx].(int))
				argIdx++
			case 's':
				printString(args[argIdx].(string))
				argIdx++
			case 'c':
				switch v := args[argIdx].(type) {
				case int:
					uart_putc(byte(v))
				case int32:
					uart_putc(byte(v))
				default:
					uart_putc('?')
				}
				argIdx++
			default:
				uart_putc('%')
				uart_putc(byte(format[i]))
			}
		} else {
			uart_putc(byte(format[i]))
		}
	}
}