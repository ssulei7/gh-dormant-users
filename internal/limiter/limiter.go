package limiter

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ssulei7/gh-dormant-users/internal/ui"
)

// MaxConcurrentRequests limits concurrent API calls to avoid secondary rate limits.
// GitHub recommends avoiding concurrent requests; 50 is a conservative limit.
const MaxConcurrentRequests = 50

// ConcurrentLimiter is a semaphore to limit concurrent API requests.
var ConcurrentLimiter = make(chan struct{}, MaxConcurrentRequests)

// ETag cache for conditional requests
// Key: URL, Value: ETag header value
var (
	etagCache    = make(map[string]string)
	etagCacheMux sync.RWMutex
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
// This blocks if the maximum number of concurrent requests is already in flight.
func AcquireConcurrentLimiter() {
	ConcurrentLimiter <- struct{}{}
}

// ReleaseConcurrentLimiter releases a token back to the concurrency limiter.
func ReleaseConcurrentLimiter() {
	<-ConcurrentLimiter
}

// GetCachedETag returns the cached ETag for a URL, if any.
// Use this to set the If-None-Match header on requests.
func GetCachedETag(url string) string {
	etagCacheMux.RLock()
	defer etagCacheMux.RUnlock()
	return etagCache[url]
}

// CacheETag stores the ETag from a response for future conditional requests.
// Call this after successful (200) responses.
func CacheETag(url string, response *http.Response) {
	if response == nil {
		return
	}
	etag := response.Header.Get("ETag")
	if etag != "" {
		etagCacheMux.Lock()
		etagCache[url] = etag
		etagCacheMux.Unlock()
	}
}

// IsNotModified checks if response is 304 Not Modified.
// When true, use cached data - this response doesn't count against rate limit.
func IsNotModified(response *http.Response) bool {
	return response != nil && response.StatusCode == http.StatusNotModified
}

// CheckAndHandleRateLimit checks GitHub API response headers for rate limit status.
// If rate limit is exhausted or secondary rate limit is hit, it waits appropriately.
// Returns true if we had to wait due to rate limiting.
func CheckAndHandleRateLimit(response *http.Response) bool {
	UpdateRateLimitFromResponse(response)

	if response == nil {
		return false
	}

	// Check for secondary rate limit (Retry-After header) - highest priority
	if retryAfter := response.Header.Get("Retry-After"); retryAfter != "" {
		seconds, err := strconv.Atoi(retryAfter)
		if err == nil && seconds > 0 {
			ui.Warning("Secondary rate limit hit. Waiting %d seconds...", seconds)
			time.Sleep(time.Duration(seconds) * time.Second)
			return true
		}
	}

	// Check for primary rate limit exhaustion
	remaining := response.Header.Get("X-RateLimit-Remaining")
	if remaining == "0" {
		reset := response.Header.Get("X-RateLimit-Reset")
		if reset != "" {
			resetTime, err := strconv.ParseInt(reset, 10, 64)
			if err == nil {
				resetTimestamp := time.Unix(resetTime, 0)
				waitDuration := time.Until(resetTimestamp)
				if waitDuration > 0 {
					ui.Warning("Rate limit exhausted. Waiting %v until reset...", waitDuration.Round(time.Second))
					time.Sleep(waitDuration + time.Second)
					return true
				}
			}
		}
	}

	return false
}
