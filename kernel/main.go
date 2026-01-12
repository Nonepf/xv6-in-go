package main

import _ "unsafe"

//go:linkname uart_putc uart_putc
func uart_putc(c byte)

//export KMain
func KMain() {
	msg := "RPF: Hello, world!"
	for _, c := range msg {
		uart_putc(byte(c))
	}
    for {}
}

func main() {}