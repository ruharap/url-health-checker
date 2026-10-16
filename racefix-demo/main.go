// Race condition fix: the same 1000 goroutines, but the update is guarded by a mutex.
package main

import (
	"fmt"
	"sync"
)

func main() {
	var counter int
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			counter++ // protected: only one goroutine updates at a time
			mu.Unlock()
		}()
	}
	wg.Wait()

	fmt.Println("expected 1000, got", counter)
}
