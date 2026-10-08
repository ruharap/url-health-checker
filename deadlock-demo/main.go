package main

import "fmt"

func main() {
	ch := make(chan int) // unbuffered
	ch <- 1              // BUG: blocks forever, no receiver
	fmt.Println(<-ch)
}
