package limiter

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/cli/go-gh/pkg/api"
	"github.com/pterm/pterm"
	"golang.org/x/time/rate"
)

const (
	MaxConcurrentRequests = 15
	DefaultRateLimit      = 5000 // Fallback if detection fails
)

var (
	ConcurrentLimiter     = make(chan struct{}, MaxConcurrentRequests)
	RateLimiter           *rate.Limiter
	limiterMux            sync.Mutex
	detectedRateLimit     int
	rateLimitDetected     bool
	rateLimitDetectionMux sync.Mutex
)

func init() {
	// Start with conservative default rate limiter
	// Will be updated after first API call
	RateLimiter = rate.NewLimiter(rate.Limit(DefaultRateLimit/3600.0), MaxConcurrentRequests)
}

// DetectRateLimit queries GitHub's rate_limit endpoint to determine the actual rate limit.
// This is called once at startup to configure the token bucket correctly.
func DetectRateLimit(client api.RESTClient) int {
	rateLimitDetectionMux.Lock()
	defer rateLimitDetectionMux.Unlock()

	// Return cached value if already detected
	if rateLimitDetected {
		return detectedRateLimit
	}

	response, err := client.Request("GET", "rate_limit", nil)
	if err != nil {
		pterm.Warning.Printf("Failed to detect rate limit, using default %d/hour: %v\n", DefaultRateLimit, err)
		detectedRateLimit = DefaultRateLimit
		rateLimitDetected = true
		return DefaultRateLimit
	}
	defer response.Body.Close()

	var rateLimitResponse struct {
		Resources struct {
			Core struct {
				Limit int `json:"limit"`
			} `json:"core"`
		} `json:"resources"`
	}

	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&rateLimitResponse); err != nil {
		pterm.Warning.Printf("Failed to parse rate limit, using default %d/hour: %v\n", DefaultRateLimit, err)
		detectedRateLimit = DefaultRateLimit
		rateLimitDetected = true
		return DefaultRateLimit
	}

	limit := rateLimitResponse.Resources.Core.Limit
	if limit <= 0 {
		pterm.Warning.Printf("Invalid rate limit detected, using default %d/hour\n", DefaultRateLimit)
		limit = DefaultRateLimit
	}

	detectedRateLimit = limit
	rateLimitDetected = true

	// Update the rate limiter with detected limit
	requestsPerSecond := float64(limit) / 3600.0
	limiterMux.Lock()
	RateLimiter.SetLimit(rate.Limit(requestsPerSecond))
	RateLimiter.SetBurst(MaxConcurrentRequests)
	limiterMux.Unlock()

	pterm.Info.Printf("Detected GitHub API rate limit: %d requests/hour (%.2f req/sec)\n", limit, requestsPerSecond)
	return limit
}

// WaitForTokenAndAcquire waits for a rate limit token and acquires the concurrency semaphore.
// This should be called before making any API request.
func WaitForTokenAndAcquire(ctx context.Context) error {
	if err := RateLimiter.Wait(ctx); err != nil {
		return err
	}
	ConcurrentLimiter <- struct{}{}
	return nil
}

// AcquireConcurrentLimiter acquires a token from the concurrency limiter.
// Deprecated: Use WaitForTokenAndAcquire for new code.
func AcquireConcurrentLimiter() {
	ConcurrentLimiter <- struct{}{}
}

// ReleaseConcurrentLimiter releases a token back to the concurrency limiter.
func ReleaseConcurrentLimiter() {
	<-ConcurrentLimiter
}

// UpdateRateLimitFromResponse dynamically adjusts the rate limiter based on GitHub's response headers.
func UpdateRateLimitFromResponse(response *http.Response) {
	if response == nil {
		return
	}

	remaining := response.Header.Get("X-RateLimit-Remaining")
	reset := response.Header.Get("X-RateLimit-Reset")

	if remaining != "" && reset != "" {
		remainingCount, err1 := strconv.Atoi(remaining)
		resetTime, err2 := strconv.ParseInt(reset, 10, 64)

		if err1 == nil && err2 == nil {
			resetTimestamp := time.Unix(resetTime, 0)
			timeUntilReset := time.Until(resetTimestamp)

			if timeUntilReset > 0 && remainingCount > 0 {
				// Dynamically adjust rate: spread remaining requests over remaining time
				newRate := float64(remainingCount) / timeUntilReset.Seconds()

				// Keep a small buffer - use 90% of calculated rate to be safe
				newRate = newRate * 0.9

				limiterMux.Lock()
				RateLimiter.SetLimit(rate.Limit(newRate))
				RateLimiter.SetBurst(MaxConcurrentRequests)
				limiterMux.Unlock()
			}
		}
	}
}

// ReleaseAndHandleRateLimit releases the semaphore and handles any rate limit issues.
// This releases the semaphore BEFORE sleeping, allowing other workers to continue.
func ReleaseAndHandleRateLimit(response *http.Response) {
	// Update rate limiter based on actual response
	UpdateRateLimitFromResponse(response)

	// Release semaphore BEFORE any potential sleep
	ReleaseConcurrentLimiter()

	if response == nil {
		return
	}

	// Handle secondary rate limit (Retry-After header)
	if retryAfter := response.Header.Get("Retry-After"); retryAfter != "" {
		seconds, err := strconv.Atoi(retryAfter)
		if err == nil && seconds > 0 {
			pterm.Warning.Printf("Secondary rate limit hit. Waiting %d seconds...\n", seconds)
			time.Sleep(time.Duration(seconds) * time.Second)
		}
	}

	// Handle primary rate limit exhaustion (rare with token bucket)
	remaining := response.Header.Get("X-RateLimit-Remaining")
	if remaining == "0" {
		reset := response.Header.Get("X-RateLimit-Reset")
		if reset != "" {
			resetTime, err := strconv.ParseInt(reset, 10, 64)
			if err == nil {
				resetTimestamp := time.Unix(resetTime, 0)
				waitDuration := time.Until(resetTimestamp)
				if waitDuration > 0 {
					pterm.Warning.Printf("Rate limit exhausted (%s remaining). Waiting %v until reset...\n",
						remaining, waitDuration.Round(time.Second))
					time.Sleep(waitDuration + time.Second)
				}
			}
		}
	}
}

// CheckAndHandleRateLimit checks for rate limit headers and waits if necessary.
// Deprecated: Use ReleaseAndHandleRateLimit for new code which properly releases semaphore before sleeping.
func CheckAndHandleRateLimit(response *http.Response) bool {
	UpdateRateLimitFromResponse(response)

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
	if remaining == "0" {
		reset := response.Header.Get("X-RateLimit-Reset")
		if reset != "" {
			resetTime, err := strconv.ParseInt(reset, 10, 64)
			if err == nil {
				resetTimestamp := time.Unix(resetTime, 0)
				waitDuration := time.Until(resetTimestamp)
				if waitDuration > 0 {
					pterm.Warning.Printf("Rate limit exhausted. Waiting %v until reset...\n", waitDuration.Round(time.Second))
					time.Sleep(waitDuration + time.Second)
					return true
				}
			}
		}
	}

	return false
}
