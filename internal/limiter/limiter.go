package limiter

import (
	"net/http"
	"strconv"
	"time"

	"github.com/pterm/pterm"
)

const (
	PrimaryRateLimit      = 5000
	MaxConcurrentRequests = 15
)

var (
	ConcurrentLimiter = make(chan struct{}, MaxConcurrentRequests)
)

// AcquireConcurrentLimiter acquires a token from the concurrency limiter.
func AcquireConcurrentLimiter() {
	ConcurrentLimiter <- struct{}{}
}

// ReleaseConcurrentLimiter releases a token back to the concurrency limiter.
func ReleaseConcurrentLimiter() {
	<-ConcurrentLimiter
}

// CheckAndHandleRateLimit checks for rate limit headers and waits if necessary.
// Returns true if a rate limit was hit and handled, false otherwise.
func CheckAndHandleRateLimit(response *http.Response) bool {
	if response == nil {
		return false
	}

	// Check for secondary rate limit (Retry-After header)
	if retryAfter := response.Header.Get("Retry-After"); retryAfter != "" {
		seconds, err := strconv.Atoi(retryAfter)
		if err == nil && seconds > 0 {
			pterm.Warning.Printf("Secondary rate limit hit. Waiting %d seconds...\n", seconds)
			time.Sleep(time.Duration(seconds) * time.Second)
			return true
		}
	}

	// Check for primary rate limit
	remaining := response.Header.Get("X-RateLimit-Remaining")
	if remaining != "" {
		remainingCount, err := strconv.Atoi(remaining)
		if err == nil && remainingCount <= 10 {
			reset := response.Header.Get("X-RateLimit-Reset")
			if reset != "" {
				resetTime, err := strconv.ParseInt(reset, 10, 64)
				if err == nil {
					resetTimestamp := time.Unix(resetTime, 0)
					waitDuration := time.Until(resetTimestamp)
					if waitDuration > 0 {
						pterm.Warning.Printf("Rate limit approaching (%d remaining). Waiting %v until reset...\n", remainingCount, waitDuration.Round(time.Second))
						time.Sleep(waitDuration + time.Second)
						return true
					}
				}
			}
		}
	}

	return false
}
