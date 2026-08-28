package main

import (
	"fmt"
	"net/http"
	"sync"
)

func worker(urls <-chan string, results chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	for url := range urls {
		resp, err := http.Get(url)
		if err != nil {
			results <- fmt.Sprintf("%s -> ERROR", url)
			continue
		}
		results <- fmt.Sprintf("%s -> %d", url, resp.StatusCode)
	}
}

func main() {
	urls := []string{"https://go.dev", "https://google.com", "https://ecu.edu.au"}
	urlChan := make(chan string, len(urls))
	results := make(chan string, len(urls))
	var wg sync.WaitGroup
	for i := 0; i < 3; i++ { // fixed-size worker pool
		wg.Add(1)
		go worker(urlChan, results, &wg)
	}
	for _, u := range urls {
		urlChan <- u
	}
	close(urlChan)
	go func() { wg.Wait(); close(results) }()
	for r := range results {
		fmt.Println(r)
	}
}
