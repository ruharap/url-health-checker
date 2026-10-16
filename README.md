# URL Health Checker

Concurrent URL health checker written in Go: worker pool, channels, mutex, context cancellation.

Requirements: Go 1.27 (any Go 1.22+ should work).

Run:
  go run . https://go.dev https://google.com
  go run . -workers 5 -timeout 5s https://go.dev https://github.com
  (Ctrl+C stops the run and reports unfinished checks as failed)

Demonstrations:
  go run ./race-demo       (unprotected counter, usually prints less than 1000)
  go run ./deadlock-demo   (ends with "fatal error: all goroutines are asleep - deadlock!")

Repository: https://github.com/ruharap/url-health-checker
