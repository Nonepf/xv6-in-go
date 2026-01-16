package main

import _ "unsafe"

type Counter struct {
	lock spinlock
	num  int
}
var count Counter
var addLimit int 

//export KMain
func KMain() {
	printf("kmeminit... ")
	kinit()
	printf("OK\n")

	printf("kvminit...  ")
	kvminit()
	printf("OK\n")

	printf("kvminithart...  ")
	kvminithart(kernel_pagetable)
	printf("OK\n")

	addLimit = 0
	printf("initlock...")
	initlock(&count.lock)
	printf("OK\n")

	printf("trapinithart...  ")
	trapinithart()
	printf("OK\n")

	//printfTest()
	//kallocTest()
	spinlockTest()
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

func spinlockTest() {
	printf("--- spinlock test ---\n")
	for i := 0; i < 1000; i++ {
		for i := 0; i < 100000; i++ {
			printf("")
		}
		acquire(&count.lock)
		count.num++
		release(&count.lock)
	}
	printf("Current Count: %d\n", count.num)
	for addLimit < 1000 {
		for i := 0; i < 100000; i++ {
			printf("")
		}
	}
	printf("Expected Count: 2000, Real Count: %d\n", count.num)
}

func main() {}