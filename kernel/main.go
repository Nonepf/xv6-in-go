package main

import _ "unsafe"

//export KMain
func KMain() {
	printfTest()
	kallocTest()
    for {}
}

func printfTest() {
	printf("--- printf test ---\n")
	printInt(2147483647)
	uart_putc('\n')
	printString("Hello there")
	uart_putc('\n')
	t := 1
	printf("Today is %s \n, %c %d %d %x\n", "Monday", 'M', t, 2)
}

func kallocTest() {
	printf("--- kalloc test ---\n")

	printf("init kmem... ")
	kinit()
	printf("OK\n")

	printf("test kalloc\n")
	count := 0
	for kalloc() != 0 {
		count++
	}
	printf("allocate %d KB memory\n", int(count * 4))

}

func main() {}