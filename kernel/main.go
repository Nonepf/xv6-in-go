package main

import _ "unsafe"

//export KMain
func KMain() {
	printf("kmeminit... ")
	kinit()
	printf("OK\n")

	printf("kvminit...  ")
	kvminit()
	printf("OK\n")

	printf("kvminithart...  ")
	kvminithart()
	printf("OK\n")

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
	printf("Today is %s \n, %c %d %d\n", "Monday", 'M', t, 2)
}

func kallocTest() {
	printf("--- kalloc test ---\n")

	printf("test kalloc\n")
	count := 0
	for kalloc() != 0 {
		count++
	}
	printf("allocate %d KB memory\n", int(count * 4))

}

func main() {}