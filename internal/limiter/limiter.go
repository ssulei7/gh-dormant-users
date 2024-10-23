package limiter

const (
	PrimaryRateLimit      = 5000
	MaxConcurrentRequests = 100
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
