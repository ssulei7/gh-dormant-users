package githubapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	defaultMaxRetries = 3
	cacheVersion      = 1
	cacheMarker       = ".gh-dormant-users-cache"
)

type Config struct {
	Transport          http.RoundTripper
	CacheDir           string
	CacheEnabled       bool
	MaxConcurrency     int
	InitialConcurrency int
	RequestsPerSecond  float64
	RateLimitReserve   float64
	MaxRetries         int
	Now                func() time.Time
	Sleep              func(context.Context, time.Duration) error
	Jitter             func(time.Duration) time.Duration
}

func isSecondaryLimitResponse(response *http.Response) bool {
	if response.StatusCode != http.StatusForbidden && response.StatusCode != http.StatusTooManyRequests {
		return false
	}
	return response.Header.Get("X-RateLimit-Remaining") != "0"
}

func retryableRead(request *http.Request) bool {
	if request.Method == http.MethodGet || request.Method == http.MethodHead {
		return true
	}
	if request.Method != http.MethodPost || !strings.HasSuffix(request.URL.Path, "/graphql") || request.GetBody == nil {
		return false
	}
	body, err := request.GetBody()
	if err != nil {
		return false
	}
	defer body.Close()
	data, err := io.ReadAll(body)
	if err != nil {
		return false
	}
	var payload struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return false
	}
	query := strings.TrimSpace(payload.Query)
	return query != "" && !strings.HasPrefix(strings.ToLower(query), "mutation")
}

func resetRequestBody(request *http.Request) (*http.Request, error) {
	if request.GetBody == nil {
		return request, nil
	}
	body, err := request.GetBody()
	if err != nil {
		return nil, fmt.Errorf("reset request body: %w", err)
	}
	copy := request.Clone(request.Context())
	copy.Body = body
	return copy, nil
}

type Stats struct {
	Requests                  int
	Responses                 int
	CacheHits                 int
	CacheErrors               int
	Retries                   int
	Waits                     int
	WaitDuration              time.Duration
	RateLimit                 int
	RateRemaining             int
	RateReset                 time.Time
	EndpointCounts            map[string]int
	InitialConcurrency        int
	CurrentConcurrency        int
	PeakConcurrency           int
	ConcurrencyReductions     int
	P50Latency                time.Duration
	P95Latency                time.Duration
	AchievedRequestsPerSecond float64
	ConcurrencyWaitDuration   time.Duration
}

type Coordinator struct {
	transport        http.RoundTripper
	cacheDir         string
	cacheEnabled     bool
	rateLimitReserve float64
	minInterval      time.Duration
	maxRetries       int
	now              func() time.Time
	sleep            func(context.Context, time.Duration) error
	jitter           func(time.Duration) time.Duration
	gate             *concurrencyGate
	controller       *adaptiveController

	scheduleMu   sync.Mutex
	nextRequest  time.Time
	blockedUntil time.Time

	statsMu sync.Mutex
	stats   Stats
	rates   map[string]rateState
}

type rateState struct {
	limit     int
	remaining int
	reset     time.Time
}

type cacheEntry struct {
	Version      int       `json:"version"`
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"last_modified,omitempty"`
	Link         string    `json:"link,omitempty"`
	ContentType  string    `json:"content_type,omitempty"`
	Body         []byte    `json:"body"`
	StoredAt     time.Time `json:"stored_at"`
}

func NewCoordinator(config Config) (*Coordinator, error) {
	if config.Transport == nil {
		config.Transport = http.DefaultTransport
	}
	if config.MaxConcurrency < 1 || config.MaxConcurrency > 15 {
		return nil, fmt.Errorf("max concurrency must be between 1 and 15")
	}
	if config.InitialConcurrency == 0 {
		config.InitialConcurrency = config.MaxConcurrency
	}
	if config.InitialConcurrency < 1 || config.InitialConcurrency > config.MaxConcurrency {
		return nil, fmt.Errorf("initial concurrency must be between 1 and max concurrency")
	}
	if config.RequestsPerSecond < 0 || config.RequestsPerSecond > 15 {
		return nil, fmt.Errorf("requests per second must be between 0 and 15")
	}
	if config.RateLimitReserve < 0 || config.RateLimitReserve >= 1 {
		return nil, fmt.Errorf("rate limit reserve must be between 0 and 1")
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = defaultMaxRetries
	}
	if config.MaxRetries < 0 {
		return nil, fmt.Errorf("max retries cannot be negative")
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	if config.Sleep == nil {
		config.Sleep = sleepContext
	}
	if config.Jitter == nil {
		config.Jitter = func(max time.Duration) time.Duration {
			if max <= 0 {
				return 0
			}
			return time.Duration(rand.Int64N(int64(max)))
		}
	}
	if config.CacheEnabled {
		if config.CacheDir == "" {
			return nil, errors.New("cache directory is required when caching is enabled")
		}
		if err := os.MkdirAll(config.CacheDir, 0o700); err != nil {
			return nil, fmt.Errorf("create cache directory: %w", err)
		}
		if err := ensureCacheMarker(config.CacheDir); err != nil {
			return nil, err
		}
	}

	gate := newConcurrencyGate(config.InitialConcurrency, config.MaxConcurrency)
	coordinator := &Coordinator{
		transport:        config.Transport,
		cacheDir:         config.CacheDir,
		cacheEnabled:     config.CacheEnabled,
		rateLimitReserve: config.RateLimitReserve,
		minInterval:      requestInterval(config.RequestsPerSecond),
		maxRetries:       config.MaxRetries,
		now:              config.Now,
		sleep:            config.Sleep,
		jitter:           config.Jitter,
		gate:             gate,
		stats: Stats{
			EndpointCounts: make(map[string]int),
		},
		rates: make(map[string]rateState),
	}
	coordinator.controller = newAdaptiveController(
		gate,
		config.InitialConcurrency,
		config.MaxConcurrency,
		config.RequestsPerSecond,
		config.Now,
	)
	return coordinator, nil
}

func (c *Coordinator) RoundTrip(request *http.Request) (*http.Response, error) {
	if err := c.acquire(request.Context()); err != nil {
		return nil, err
	}
	defer c.release()

	var entry *cacheEntry
	var cachePath string
	if c.cacheEnabled && request.Method == http.MethodGet {
		cachePath = c.cachePath(request)
		var cacheErr error
		entry, cacheErr = c.readCache(cachePath)
		if cacheErr != nil {
			c.recordCacheError()
		}
		if entry != nil {
			request = request.Clone(request.Context())
			if entry.ETag != "" {
				request.Header.Set("If-None-Match", entry.ETag)
			} else if entry.LastModified != "" {
				request.Header.Set("If-Modified-Since", entry.LastModified)
			}
		}
	}

	for attempt := 0; ; attempt++ {
		primaryCandidate := entry == nil
		if err := c.waitForTurn(request.Context(), request, primaryCandidate); err != nil {
			return nil, err
		}
		if attempt > 0 {
			var err error
			request, err = resetRequestBody(request)
			if err != nil {
				return nil, err
			}
		}

		c.recordRequest(request)
		requestStarted := c.now()
		response, err := c.transport.RoundTrip(request)
		if err != nil {
			c.observeRequest(requestStarted)
			if retryableRead(request) && attempt < c.maxRetries {
				c.recordRetry()
				if err := c.wait(request.Context(), c.retryDelay(attempt)); err != nil {
					return nil, err
				}
				continue
			}
			return nil, err
		}

		c.recordResponse(response)
		c.updateRateState(response)

		if entry != nil && response.StatusCode == http.StatusNotModified {
			_ = response.Body.Close()
			c.recordCacheHit()
			c.observeRequest(requestStarted)
			return cachedResponse(request, response, entry), nil
		}

		shouldRetry := c.shouldRetry(request, response)
		if shouldRetry && isSecondaryLimitResponse(response) {
			c.controller.onSecondaryLimit()
		}
		if shouldRetry && attempt < c.maxRetries {
			delay := c.rateLimitDelay(response, attempt)
			_ = response.Body.Close()
			c.observeRequest(requestStarted)
			c.blockFor(delay)
			c.recordRetry()
			continue
		}

		if c.cacheEnabled && request.Method == http.MethodGet && response.StatusCode == http.StatusOK {
			body, readErr := io.ReadAll(response.Body)
			_ = response.Body.Close()
			if readErr != nil {
				c.observeRequest(requestStarted)
				return nil, fmt.Errorf("read response for cache: %w", readErr)
			}
			response.Body = io.NopCloser(bytes.NewReader(body))
			response.ContentLength = int64(len(body))
			if validatorPresent(response.Header) {
				if err := c.writeCache(cachePath, response, body); err != nil {
					c.recordCacheError()
				}
			}
		}

		c.observeRequest(requestStarted)
		return response, nil
	}
}

func (c *Coordinator) observeRequest(started time.Time) {
	gate := c.gate.snapshot()
	c.controller.observe(c.now().Sub(started), gate.inFlight >= gate.limit)
}

func (c *Coordinator) Stats() Stats {
	c.statsMu.Lock()
	stats := c.stats
	stats.EndpointCounts = make(map[string]int, len(c.stats.EndpointCounts))
	for endpoint, count := range c.stats.EndpointCounts {
		stats.EndpointCounts[endpoint] = count
	}
	c.statsMu.Unlock()

	adaptive := c.controller.snapshot()
	stats.InitialConcurrency = adaptive.initial
	stats.CurrentConcurrency = adaptive.current
	stats.PeakConcurrency = adaptive.peak
	stats.ConcurrencyReductions = adaptive.reductions
	stats.P50Latency = adaptive.p50
	stats.P95Latency = adaptive.p95
	stats.AchievedRequestsPerSecond = adaptive.achievedRate
	return stats
}

func ClearCache(cacheDir string) error {
	if cacheDir == "" {
		return errors.New("cache directory is required")
	}
	_, err := os.Stat(cacheDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect cache directory: %w", err)
	}
	if _, err := os.Stat(filepath.Join(cacheDir, cacheMarker)); err != nil {
		return fmt.Errorf("refusing to clear unrecognized cache directory %s", cacheDir)
	}
	if err := os.RemoveAll(cacheDir); err != nil {
		return fmt.Errorf("clear cache: %w", err)
	}
	return nil
}

func DefaultCacheDir() (string, error) {
	cacheRoot, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("resolve user cache directory: %w", err)
	}
	return filepath.Join(cacheRoot, "gh-dormant-users"), nil
}

func (c *Coordinator) acquire(ctx context.Context) error {
	started := c.now()
	if err := c.gate.acquire(ctx); err != nil {
		return err
	}
	waited := c.now().Sub(started)
	if waited > 0 {
		c.statsMu.Lock()
		c.stats.ConcurrencyWaitDuration += waited
		c.statsMu.Unlock()
	}
	return nil
}

func (c *Coordinator) release() {
	c.gate.release()
}

func (c *Coordinator) waitForTurn(ctx context.Context, request *http.Request, primaryCandidate bool) error {
	c.scheduleMu.Lock()
	now := c.now()
	start := now
	if c.nextRequest.After(start) {
		start = c.nextRequest
	}
	if c.blockedUntil.After(start) {
		start = c.blockedUntil
	}
	if primaryCandidate {
		if primaryBlockedUntil := c.primaryBlockedUntil(request, now); primaryBlockedUntil.After(start) {
			start = primaryBlockedUntil
		}
	}
	delay := start.Sub(now)
	c.nextRequest = start.Add(c.minInterval)
	c.scheduleMu.Unlock()

	return c.wait(ctx, delay)
}

func (c *Coordinator) primaryBlockedUntil(request *http.Request, now time.Time) time.Time {
	c.statsMu.Lock()
	defer c.statsMu.Unlock()

	rate := c.rates[rateResource(request)]
	if rate.limit <= 0 || !rate.reset.After(now) {
		return time.Time{}
	}
	reserve := int(float64(rate.limit) * c.rateLimitReserve)
	if rate.remaining > reserve {
		return time.Time{}
	}
	return rate.reset.Add(time.Second)
}

func (c *Coordinator) wait(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	c.statsMu.Lock()
	c.stats.Waits++
	c.stats.WaitDuration += delay
	c.statsMu.Unlock()
	return c.sleep(ctx, delay)
}

func (c *Coordinator) retryDelay(attempt int) time.Duration {
	base := time.Second << attempt
	return base + c.jitter(base/2)
}

func (c *Coordinator) shouldRetry(request *http.Request, response *http.Response) bool {
	if !retryableRead(request) {
		return false
	}
	if response.StatusCode >= 500 {
		return true
	}
	if response.StatusCode == http.StatusTooManyRequests {
		return true
	}
	if response.StatusCode != http.StatusForbidden {
		return false
	}
	if response.Header.Get("Retry-After") != "" || response.Header.Get("X-RateLimit-Remaining") == "0" {
		return true
	}

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return false
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	message := strings.ToLower(string(body))
	return strings.Contains(message, "secondary rate limit") || strings.Contains(message, "abuse detection")
}

func (c *Coordinator) rateLimitDelay(response *http.Response, attempt int) time.Duration {
	if retryAfter, err := strconv.Atoi(response.Header.Get("Retry-After")); err == nil && retryAfter > 0 {
		return time.Duration(retryAfter) * time.Second
	}
	if response.Header.Get("X-RateLimit-Remaining") == "0" {
		if reset, err := strconv.ParseInt(response.Header.Get("X-RateLimit-Reset"), 10, 64); err == nil {
			delay := time.Unix(reset, 0).Sub(c.now()) + time.Second
			if delay > 0 {
				return delay
			}
		}
	}
	base := time.Minute << attempt
	return base + c.jitter(base/4)
}

func (c *Coordinator) blockFor(delay time.Duration) {
	c.scheduleMu.Lock()
	until := c.now().Add(delay)
	if until.After(c.blockedUntil) {
		c.blockedUntil = until
	}
	c.scheduleMu.Unlock()
}

func (c *Coordinator) updateRateState(response *http.Response) {
	limit, limitErr := strconv.Atoi(response.Header.Get("X-RateLimit-Limit"))
	remaining, remainingErr := strconv.Atoi(response.Header.Get("X-RateLimit-Remaining"))
	reset, resetErr := strconv.ParseInt(response.Header.Get("X-RateLimit-Reset"), 10, 64)
	if limitErr != nil || remainingErr != nil || resetErr != nil {
		return
	}

	resource := response.Header.Get("X-RateLimit-Resource")
	if resource == "" {
		resource = "core"
	}
	state := rateState{limit: limit, remaining: remaining, reset: time.Unix(reset, 0)}

	c.statsMu.Lock()
	c.rates[resource] = state
	if resource == "core" {
		c.stats.RateLimit = limit
		c.stats.RateRemaining = remaining
		c.stats.RateReset = state.reset
	}
	c.statsMu.Unlock()
}

func requestInterval(requestsPerSecond float64) time.Duration {
	if requestsPerSecond == 0 {
		return 0
	}
	return time.Duration(float64(time.Second) / requestsPerSecond)
}

func (c *Coordinator) recordRequest(request *http.Request) {
	c.statsMu.Lock()
	c.stats.Requests++
	c.stats.EndpointCounts[endpointFamily(request)]++
	c.statsMu.Unlock()
}

func (c *Coordinator) recordResponse(_ *http.Response) {
	c.statsMu.Lock()
	c.stats.Responses++
	c.statsMu.Unlock()
}

func (c *Coordinator) recordRetry() {
	c.statsMu.Lock()
	c.stats.Retries++
	c.statsMu.Unlock()
}

func (c *Coordinator) recordCacheHit() {
	c.statsMu.Lock()
	c.stats.CacheHits++
	c.statsMu.Unlock()
}

func (c *Coordinator) recordCacheError() {
	c.statsMu.Lock()
	c.stats.CacheErrors++
	c.statsMu.Unlock()
}

func (c *Coordinator) cachePath(request *http.Request) string {
	authHash := sha256.Sum256([]byte(request.Header.Get("Authorization")))
	requestHash := sha256.Sum256([]byte(request.Method + "\n" + request.URL.String()))
	return filepath.Join(
		c.cacheDir,
		hex.EncodeToString(authHash[:]),
		hex.EncodeToString(requestHash[:])+".json",
	)
}

func ensureCacheMarker(cacheDir string) error {
	path := filepath.Join(cacheDir, cacheMarker)
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect cache marker: %w", err)
	}
	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		return fmt.Errorf("inspect cache directory: %w", err)
	}
	if len(entries) != 0 {
		return fmt.Errorf("refusing to use non-empty unrecognized cache directory %s", cacheDir)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("create cache marker: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close cache marker: %w", err)
	}
	return nil
}

func (c *Coordinator) readCache(path string) (*cacheEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read cache entry: %w", err)
	}
	var entry cacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, fmt.Errorf("decode cache entry: %w", err)
	}
	if entry.Version != cacheVersion {
		return nil, fmt.Errorf("unsupported cache version %d", entry.Version)
	}
	return &entry, nil
}

func (c *Coordinator) writeCache(path string, response *http.Response, body []byte) error {
	entry := cacheEntry{
		Version:      cacheVersion,
		ETag:         response.Header.Get("ETag"),
		LastModified: response.Header.Get("Last-Modified"),
		Link:         response.Header.Get("Link"),
		ContentType:  response.Header.Get("Content-Type"),
		Body:         body,
		StoredAt:     c.now().UTC(),
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encode cache entry: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create cache namespace: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(path), ".cache-*")
	if err != nil {
		return fmt.Errorf("create temporary cache entry: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return fmt.Errorf("set cache entry permissions: %w", err)
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write cache entry: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close cache entry: %w", err)
	}
	if err := os.Rename(tempName, path); err != nil {
		return fmt.Errorf("replace cache entry: %w", err)
	}
	return nil
}

func cachedResponse(request *http.Request, response *http.Response, entry *cacheEntry) *http.Response {
	headers := response.Header.Clone()
	if entry.Link != "" {
		headers.Set("Link", entry.Link)
	}
	if entry.ETag != "" {
		headers.Set("ETag", entry.ETag)
	}
	if entry.LastModified != "" {
		headers.Set("Last-Modified", entry.LastModified)
	}
	if entry.ContentType != "" {
		headers.Set("Content-Type", entry.ContentType)
	}
	headers.Set("X-Gh-Dormant-Cache", "revalidated")

	return &http.Response{
		Status:        "200 OK",
		StatusCode:    http.StatusOK,
		Header:        headers,
		Body:          io.NopCloser(bytes.NewReader(entry.Body)),
		ContentLength: int64(len(entry.Body)),
		Request:       request,
	}
}

func validatorPresent(header http.Header) bool {
	return header.Get("ETag") != "" || header.Get("Last-Modified") != ""
}

func endpointFamily(request *http.Request) string {
	if strings.HasSuffix(request.URL.Path, "/graphql") {
		return "graphql"
	}
	parts := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
	for index, part := range parts {
		if part == "members" || part == "repos" || part == "commits" || part == "issues" || part == "comments" {
			if part == "comments" && index > 0 {
				return parts[index-1] + "-comments"
			}
			return part
		}
	}
	return "other"
}

func rateResource(request *http.Request) string {
	if strings.HasSuffix(request.URL.Path, "/graphql") {
		return "graphql"
	}
	return "core"
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
