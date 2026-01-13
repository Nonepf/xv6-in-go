package main

import _ "unsafe"

//export KMain
func KMain() {
	msg := "RPF: Hello, world!"
	for _, c := range msg {
		uart_putc(byte(c))
	}
	uart_putc('\n')
	printInt(2147483647)
	uart_putc('\n')
	printString("Hello there")
	uart_putc('\n')
	printf("Today is %s \n, %c %d %d %x", "Monday", 'M', 9, 100)
    for {}
}

func main() {}