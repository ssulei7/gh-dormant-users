package limiter

import (
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
