package githubapi

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestCoordinatorRevalidatesCachedResponse(t *testing.T) {
	var mu sync.Mutex
	requests := 0
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		mu.Lock()
		defer mu.Unlock()
		requests++
		if requests == 1 {
			return response(http.StatusOK, `[{"name":"repo"}]`, map[string]string{
				"ETag": `"one"`,
				"Link": `<https://api.github.com/repos?page=2>; rel="next"`,
			}), nil
		}

		if request.Header.Get("If-None-Match") != `"one"` {
			t.Fatalf("expected If-None-Match header, got %q", request.Header.Get("If-None-Match"))
		}
		return response(http.StatusNotModified, "", nil), nil
	})

	coordinator, err := NewCoordinator(Config{
		Transport:        transport,
		CacheDir:         t.TempDir(),
		CacheEnabled:     true,
		MaxConcurrency:   1,
		RateLimitReserve: 0.1,
		Sleep:            noSleep,
		Jitter:           noJitter,
	})
	if err != nil {
		t.Fatalf("NewCoordinator returned error: %v", err)
	}

	request, _ := http.NewRequest(http.MethodGet, "https://api.github.com/repos", nil)
	request.Header.Set("Authorization", "token secret")
	first, err := coordinator.RoundTrip(request)
	if err != nil {
		t.Fatalf("first request returned error: %v", err)
	}
	_, _ = io.ReadAll(first.Body)
	_ = first.Body.Close()

	second, err := coordinator.RoundTrip(request)
	if err != nil {
		t.Fatalf("second request returned error: %v", err)
	}
	body, _ := io.ReadAll(second.Body)
	_ = second.Body.Close()
	if string(body) != `[{"name":"repo"}]` {
		t.Fatalf("unexpected cached body: %s", body)
	}
	if second.StatusCode != http.StatusOK {
		t.Fatalf("expected synthesized 200, got %d", second.StatusCode)
	}
	if second.Header.Get("Link") == "" {
		t.Fatal("expected cached Link header")
	}
	if stats := coordinator.Stats(); stats.Requests != 2 || stats.CacheHits != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
}

func TestCoordinatorRetriesSecondaryLimit(t *testing.T) {
	requests := 0
	var delays []time.Duration
	coordinator, err := NewCoordinator(Config{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			requests++
			if requests == 1 {
				return response(http.StatusTooManyRequests, `{"message":"secondary rate limit"}`, map[string]string{
					"Retry-After": "2",
				}), nil
			}
			return response(http.StatusOK, `{}`, nil), nil
		}),
		MaxConcurrency:   1,
		RateLimitReserve: 0.1,
		Sleep: func(_ context.Context, delay time.Duration) error {
			delays = append(delays, delay)
			return nil
		},
		Jitter: noJitter,
	})
	if err != nil {
		t.Fatalf("NewCoordinator returned error: %v", err)
	}

	request, _ := http.NewRequest(http.MethodGet, "https://api.github.com/repos", nil)
	result, err := coordinator.RoundTrip(request)
	if err != nil {
		t.Fatalf("RoundTrip returned error: %v", err)
	}
	_ = result.Body.Close()
	if requests != 2 {
		t.Fatalf("expected 2 requests, got %d", requests)
	}
	if len(delays) != 1 || delays[0] < 1900*time.Millisecond {
		t.Fatalf("expected a two-second wait, got %v", delays)
	}
}

func TestCoordinatorAppliesRequestRateCap(t *testing.T) {
	now := time.Unix(1000, 0)
	var delays []time.Duration
	coordinator, err := NewCoordinator(Config{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return response(http.StatusOK, `{}`, map[string]string{
				"X-RateLimit-Limit":     "100",
				"X-RateLimit-Remaining": "90",
				"X-RateLimit-Reset":     "1100",
				"X-RateLimit-Resource":  "core",
			}), nil
		}),
		MaxConcurrency:    1,
		RequestsPerSecond: 10,
		RateLimitReserve:  0.1,
		Now:               func() time.Time { return now },
		Sleep: func(_ context.Context, delay time.Duration) error {
			delays = append(delays, delay)
			return nil
		},
		Jitter: noJitter,
	})
	if err != nil {
		t.Fatalf("NewCoordinator returned error: %v", err)
	}

	request, _ := http.NewRequest(http.MethodGet, "https://api.github.com/repos", nil)
	for range 2 {
		result, requestErr := coordinator.RoundTrip(request)
		if requestErr != nil {
			t.Fatalf("RoundTrip returned error: %v", requestErr)
		}
		_ = result.Body.Close()
	}
	if len(delays) != 1 || delays[0] != 100*time.Millisecond {
		t.Fatalf("expected a 100 millisecond rate-cap delay, got %v", delays)
	}
}

func TestCoordinatorWaitsWhenPrimaryReserveReached(t *testing.T) {
	now := time.Unix(1000, 0)
	var delays []time.Duration
	coordinator, err := NewCoordinator(Config{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return response(http.StatusOK, `{}`, map[string]string{
				"X-RateLimit-Limit":     "100",
				"X-RateLimit-Remaining": "10",
				"X-RateLimit-Reset":     "1100",
				"X-RateLimit-Resource":  "core",
			}), nil
		}),
		MaxConcurrency:    1,
		RequestsPerSecond: 10,
		RateLimitReserve:  0.1,
		Now:               func() time.Time { return now },
		Sleep: func(_ context.Context, delay time.Duration) error {
			delays = append(delays, delay)
			return nil
		},
		Jitter: noJitter,
	})
	if err != nil {
		t.Fatalf("NewCoordinator returned error: %v", err)
	}

	request, _ := http.NewRequest(http.MethodGet, "https://api.github.com/repos", nil)
	for range 2 {
		result, requestErr := coordinator.RoundTrip(request)
		if requestErr != nil {
			t.Fatalf("RoundTrip returned error: %v", requestErr)
		}
		_ = result.Body.Close()
	}
	if len(delays) != 1 || delays[0] != 101*time.Second {
		t.Fatalf("expected wait through reset, got %v", delays)
	}
}

func TestCoordinatorRetriesGraphQLQueriesButNotMutations(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		requests int
	}{
		{name: "query", query: "query { viewer { login } }", requests: 2},
		{name: "mutation", query: "mutation { changeUserStatus(input: {}) { clientMutationId } }", requests: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			coordinator, err := NewCoordinator(Config{
				Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
					requests++
					body, _ := io.ReadAll(request.Body)
					if len(body) == 0 {
						t.Fatal("expected GraphQL body on every attempt")
					}
					return response(http.StatusInternalServerError, `{}`, nil), nil
				}),
				MaxConcurrency:   1,
				RateLimitReserve: 0.1,
				MaxRetries:       1,
				Sleep:            noSleep,
				Jitter:           noJitter,
			})
			if err != nil {
				t.Fatalf("NewCoordinator returned error: %v", err)
			}

			body := bytes.NewBufferString(`{"query":"` + test.query + `"}`)
			request, _ := http.NewRequest(http.MethodPost, "https://api.github.com/graphql", body)
			result, requestErr := coordinator.RoundTrip(request)
			if requestErr != nil {
				t.Fatalf("RoundTrip returned error: %v", requestErr)
			}
			_ = result.Body.Close()
			if requests != test.requests {
				t.Fatalf("expected %d requests, got %d", test.requests, requests)
			}
		})
	}
}

func TestCoordinatorBoundsConcurrency(t *testing.T) {
	var active atomic.Int32
	var maximum atomic.Int32
	coordinator, err := NewCoordinator(Config{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			current := active.Add(1)
			for {
				seen := maximum.Load()
				if current <= seen || maximum.CompareAndSwap(seen, current) {
					break
				}
			}
			time.Sleep(10 * time.Millisecond)
			active.Add(-1)
			return response(http.StatusOK, `{}`, nil), nil
		}),
		MaxConcurrency:   2,
		RateLimitReserve: 0.1,
		Sleep:            noSleep,
		Jitter:           noJitter,
	})
	if err != nil {
		t.Fatalf("NewCoordinator returned error: %v", err)
	}

	var wg sync.WaitGroup
	for index := 0; index < 10; index++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			request, _ := http.NewRequest(http.MethodGet, "https://api.github.com/repos", nil)
			result, requestErr := coordinator.RoundTrip(request)
			if requestErr != nil {
				t.Errorf("RoundTrip returned error: %v", requestErr)
				return
			}
			_ = result.Body.Close()
		}()
	}
	wg.Wait()
	if maximum.Load() != 2 {
		t.Fatalf("expected maximum concurrency 2, got %d", maximum.Load())
	}
}

func TestCoordinatorRecoversFromCorruptCache(t *testing.T) {
	cacheDir := t.TempDir()
	requests := 0
	coordinator, err := NewCoordinator(Config{
		Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			requests++
			if request.Header.Get("If-None-Match") != "" {
				t.Fatal("corrupt cache validator should not be used")
			}
			return response(http.StatusOK, `{}`, map[string]string{"ETag": `"value"`}), nil
		}),
		CacheDir:         cacheDir,
		CacheEnabled:     true,
		MaxConcurrency:   1,
		RateLimitReserve: 0.1,
		Sleep:            noSleep,
		Jitter:           noJitter,
	})
	if err != nil {
		t.Fatalf("NewCoordinator returned error: %v", err)
	}

	request, _ := http.NewRequest(http.MethodGet, "https://api.github.com/repos", nil)
	request.Header.Set("Authorization", "token secret")
	result, err := coordinator.RoundTrip(request)
	if err != nil {
		t.Fatalf("first RoundTrip returned error: %v", err)
	}
	_ = result.Body.Close()

	var cacheFile string
	_ = filepath.Walk(cacheDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr == nil && !info.IsDir() && filepath.Ext(path) == ".json" {
			cacheFile = path
		}
		return nil
	})
	if cacheFile == "" {
		t.Fatal("expected cache file")
	}
	if err := os.WriteFile(cacheFile, []byte("not json"), 0o600); err != nil {
		t.Fatalf("corrupt cache file: %v", err)
	}

	result, err = coordinator.RoundTrip(request)
	if err != nil {
		t.Fatalf("second RoundTrip returned error: %v", err)
	}
	_ = result.Body.Close()
	if requests != 2 {
		t.Fatalf("expected corrupt entry to be refetched, got %d requests", requests)
	}
	if coordinator.Stats().CacheErrors != 1 {
		t.Fatalf("expected one cache error, got %+v", coordinator.Stats())
	}
}

func TestClearCacheRefusesUnrecognizedDirectory(t *testing.T) {
	cacheDir := t.TempDir()
	sentinel := filepath.Join(cacheDir, "keep")
	if err := os.WriteFile(sentinel, []byte("important"), 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	if err := ClearCache(cacheDir); err == nil {
		t.Fatal("expected ClearCache to reject an unrecognized directory")
	}
	if _, err := os.Stat(sentinel); err != nil {
		t.Fatalf("sentinel should not be removed: %v", err)
	}
}

func TestCoordinatorIsolatesCacheByAuthentication(t *testing.T) {
	requests := 0
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests++
		switch request.Header.Get("Authorization") {
		case "token one":
			if request.Header.Get("If-None-Match") == `"one"` {
				return response(http.StatusNotModified, "", nil), nil
			}
			return response(http.StatusOK, `{"user":"one"}`, map[string]string{"ETag": `"one"`}), nil
		case "token two":
			if request.Header.Get("If-None-Match") != "" {
				t.Fatal("second identity received first identity's validator")
			}
			return response(http.StatusOK, `{"user":"two"}`, map[string]string{"ETag": `"two"`}), nil
		default:
			t.Fatalf("unexpected authorization header %q", request.Header.Get("Authorization"))
			return nil, nil
		}
	})
	coordinator, err := NewCoordinator(Config{
		Transport:        transport,
		CacheDir:         t.TempDir(),
		CacheEnabled:     true,
		MaxConcurrency:   1,
		RateLimitReserve: 0.1,
		Sleep:            noSleep,
		Jitter:           noJitter,
	})
	if err != nil {
		t.Fatalf("NewCoordinator returned error: %v", err)
	}

	for _, token := range []string{"token one", "token two", "token one"} {
		request, _ := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
		request.Header.Set("Authorization", token)
		result, requestErr := coordinator.RoundTrip(request)
		if requestErr != nil {
			t.Fatalf("RoundTrip returned error: %v", requestErr)
		}
		_ = result.Body.Close()
	}
	if requests != 3 || coordinator.Stats().CacheHits != 1 {
		t.Fatalf("unexpected cache behavior: requests=%d stats=%+v", requests, coordinator.Stats())
	}
}

func TestCoordinatorRefusesNonEmptyUnrecognizedCacheDirectory(t *testing.T) {
	cacheDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(cacheDir, "keep"), []byte("important"), 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}
	_, err := NewCoordinator(Config{
		CacheDir:         cacheDir,
		CacheEnabled:     true,
		MaxConcurrency:   1,
		RateLimitReserve: 0.1,
	})
	if err == nil {
		t.Fatal("expected NewCoordinator to reject an unrecognized directory")
	}
}

func response(status int, body string, headers map[string]string) *http.Response {
	header := make(http.Header)
	for key, value := range headers {
		header.Set(key, value)
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

func noSleep(_ context.Context, _ time.Duration) error {
	return nil
}

func noJitter(_ time.Duration) time.Duration {
	return 0
}
