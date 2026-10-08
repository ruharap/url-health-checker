// URL Health Checker - CSP3341 Technical Report
// Demonstrates: goroutines, channels, sync.WaitGroup, sync.Mutex, context,
// implicit interfaces, struct-based ADTs, explicit error handling, defer.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"time"
)

// Result is a record type (struct) holding the outcome of one URL check.
// Exported fields (capitalised) are visible outside the package.
type Result struct {
	URL      string
	Status   int
	Duration time.Duration
	Err      error
}

// Checker is an interface. Any type with a matching Check method satisfies it
// implicitly - there is no "implements" keyword (structural typing).
type Checker interface {
	Check(ctx context.Context, url string) Result
}

// HTTPChecker is the concrete type. Its client field is unexported, so it is
// encapsulated at package level (Go's form of information hiding).
type HTTPChecker struct {
	client *http.Client
}

// NewHTTPChecker is a constructor function (Go has no constructors).
func NewHTTPChecker(timeout time.Duration) *HTTPChecker {
	return &HTTPChecker{client: &http.Client{Timeout: timeout}}
}

// Check performs one HTTP GET. Errors are returned as values and wrapped
// with %w, instead of being thrown as exceptions.
func (c *HTTPChecker) Check(ctx context.Context, url string) Result {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{URL: url, Err: fmt.Errorf("build request: %w", err)}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return Result{URL: url, Duration: time.Since(start), Err: fmt.Errorf("request failed: %w", err)}
	}
	// defer guarantees the body is closed on every return path
	// (Go's equivalent of Java's finally). Prevents connection leaks.
	defer resp.Body.Close()

	return Result{URL: url, Status: resp.StatusCode, Duration: time.Since(start)}
}

// Stats is shared between goroutines, so access is guarded by a mutex
// (competition synchronisation, Module 10).
type Stats struct {
	mu     sync.Mutex
	ok     int
	failed int
	total  time.Duration
}

// Record updates the counters. The lock makes the update atomic.
func (s *Stats) Record(r Result) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.Err != nil || r.Status >= 400 {
		s.failed++
	} else {
		s.ok++
	}
	s.total += r.Duration
}

// worker pulls URLs from the jobs channel until it is closed.
// Channel directions (<-chan / chan<-) are enforced by the compiler.
func worker(ctx context.Context, checker Checker, jobs <-chan string, results chan<- Result, wg *sync.WaitGroup) {
	defer wg.Done()
	for url := range jobs {
		// Stop early if the user pressed Ctrl+C (event-style cancellation).
		select {
		case <-ctx.Done():
			return
		default:
		}
		results <- checker.Check(ctx, url)
	}
}

func main() {
	// Command-line flags (event/input handling at program start).
	workers := flag.Int("workers", 3, "number of concurrent workers")
	timeout := flag.Duration("timeout", 5*time.Second, "per-request timeout")
	flag.Parse()

	urls := flag.Args()
	if len(urls) == 0 {
		urls = []string{"https://go.dev", "https://google.com", "https://ecu.edu.au", "https://doesnotexist.invalid"}
	}

	// signal.NotifyContext cancels ctx when Ctrl+C is received.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var checker Checker = NewHTTPChecker(*timeout)

	// Buffered channels sized to len(urls) so neither side blocks.
	jobs := make(chan string, len(urls))
	results := make(chan Result, len(urls))

	// Fixed-size worker pool.
	var wg sync.WaitGroup
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go worker(ctx, checker, jobs, results, &wg)
	}

	// Send all jobs, then close so workers' range loops end.
	for _, u := range urls {
		jobs <- u
	}
	close(jobs)

	// Close results once every worker has finished, so the loop below ends.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Aggregator: collect results in the main goroutine.
	var stats Stats
	var all []Result
	for r := range results {
		stats.Record(r)
		all = append(all, r)
	}

	// Results arrive in completion order, so sort for a stable report.
	sort.Slice(all, func(i, j int) bool { return all[i].URL < all[j].URL })

	fmt.Println(strings.Repeat("-", 64))
	fmt.Printf("%-34s %-8s %s\n", "URL", "STATUS", "TIME")
	fmt.Println(strings.Repeat("-", 64))
	for _, r := range all {
		if r.Err != nil {
			fmt.Printf("%-34s %-8s %v\n", r.URL, "FAILED", r.Duration.Round(time.Millisecond))
			continue
		}
		fmt.Printf("%-34s %-8d %v\n", r.URL, r.Status, r.Duration.Round(time.Millisecond))
	}
	fmt.Println(strings.Repeat("-", 64))

	checked := stats.ok + stats.failed
	fmt.Printf("Checked: %d | OK: %d | Failed: %d\n", checked, stats.ok, stats.failed)
	if checked > 0 {
		fmt.Printf("Average response time: %v\n", (stats.total / time.Duration(checked)).Round(time.Millisecond))
	}
	if ctx.Err() != nil {
		fmt.Println("Run was interrupted before all URLs were checked.")
	}
}
