package main

import (
	"fmt"
	"sync"
)

func main() {
	var counter int
	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter++ // BUG: non-atomic fetch-operate-store
		}()
	}
	wg.Wait()
	fmt.Println("expected 1000, got", counter)
}
