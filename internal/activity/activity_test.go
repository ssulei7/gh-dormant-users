package activity

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ssulei7/gh-dormant-users/internal/repository"
	"github.com/ssulei7/gh-dormant-users/internal/users"
)

type countingRESTClient struct {
	requests atomic.Int32
	err      error
}

type routeRESTClient struct {
	mu       sync.Mutex
	routes   map[string]string
	errs     map[string]error
	requests []string
}

func (c *routeRESTClient) Request(_ string, path string, _ io.Reader) (*http.Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requests = append(c.requests, path)
	if err := c.errs[path]; err != nil {
		return nil, err
	}
	body, ok := c.routes[path]
	if !ok {
		return nil, fmt.Errorf("unexpected request: %s", path)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}, nil
}

func (c *routeRESTClient) RequestWithContext(_ context.Context, method, path string, body io.Reader) (*http.Response, error) {
	return c.Request(method, path, body)
}

func (c *routeRESTClient) Do(method, path string, body io.Reader, result interface{}) error {
	response, err := c.Request(method, path, body)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return json.NewDecoder(response.Body).Decode(result)
}

func (c *routeRESTClient) DoWithContext(_ context.Context, method, path string, body io.Reader, result interface{}) error {
	return c.Do(method, path, body, result)
}

func (c *routeRESTClient) Delete(path string, result interface{}) error {
	return c.Do(http.MethodDelete, path, nil, result)
}

func (c *routeRESTClient) Get(path string, result interface{}) error {
	return c.Do(http.MethodGet, path, nil, result)
}

func (c *routeRESTClient) Patch(path string, body io.Reader, result interface{}) error {
	return c.Do(http.MethodPatch, path, body, result)
}

func (c *routeRESTClient) Post(path string, body io.Reader, result interface{}) error {
	return c.Do(http.MethodPost, path, body, result)
}

func (c *routeRESTClient) Put(path string, body io.Reader, result interface{}) error {
	return c.Do(http.MethodPut, path, body, result)
}

func (c *routeRESTClient) RESTPrefix() string { return "" }

func (c *countingRESTClient) Request(_ string, path string, _ io.Reader) (*http.Response, error) {
	c.requests.Add(1)
	if c.err != nil {
		return nil, c.err
	}
	return nil, fmt.Errorf("unexpected request: %s", path)
}

func (c *countingRESTClient) RequestWithContext(_ context.Context, method, path string, body io.Reader) (*http.Response, error) {
	return c.Request(method, path, body)
}

func (c *countingRESTClient) Do(method, path string, body io.Reader, result interface{}) error {
	response, err := c.Request(method, path, body)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return json.NewDecoder(response.Body).Decode(result)
}

func (c *countingRESTClient) DoWithContext(_ context.Context, method, path string, body io.Reader, result interface{}) error {
	return c.Do(method, path, body, result)
}

func (c *countingRESTClient) Delete(path string, result interface{}) error {
	return c.Do(http.MethodDelete, path, nil, result)
}

func (c *countingRESTClient) Get(path string, result interface{}) error {
	return c.Do(http.MethodGet, path, nil, result)
}

func (c *countingRESTClient) Patch(path string, body io.Reader, result interface{}) error {
	return c.Do(http.MethodPatch, path, body, result)
}

func (c *countingRESTClient) Post(path string, body io.Reader, result interface{}) error {
	return c.Do(http.MethodPost, path, body, result)
}

func (c *countingRESTClient) Put(path string, body io.Reader, result interface{}) error {
	return c.Do(http.MethodPut, path, body, result)
}

func (c *countingRESTClient) RESTPrefix() string {
	return ""
}

func TestCheckActivitySkipsRepositoriesWithoutRecentPushes(t *testing.T) {
	oldPush := time.Now().AddDate(0, -2, 0)
	repositories := repository.Repositories{
		{Name: "empty", Size: 0},
		{Name: "old", Size: 100, PushedAt: &oldPush},
	}

	client := &countingRESTClient{}
	checker := NewActivityChecker()

	err := checker.CheckActivity(
		users.Users{{Login: "octocat"}},
		"example",
		repositories,
		time.Now().AddDate(0, -1, 0).UTC().Format(time.RFC3339),
		client,
		[]string{"commits"},
	)
	if err != nil {
		t.Fatalf("CheckActivity returned error: %v", err)
	}
	if client.requests.Load() != 0 {
		t.Fatalf("expected no commit requests, got %d", client.requests.Load())
	}
}

func TestNewActivityCheckerWorkerSelection(t *testing.T) {
	if got := NewActivityChecker().workers; got != 5 {
		t.Fatalf("default workers = %d, want 5", got)
	}
	if got := NewActivityChecker(0).workers; got != 5 {
		t.Fatalf("zero workers = %d, want 5", got)
	}
	if got := NewActivityChecker(3).workers; got != 3 {
		t.Fatalf("configured workers = %d, want 3", got)
	}
}

func TestCheckActivityMarksUsersForAllSelectedActivity(t *testing.T) {
	date := "2026-07-01T00:00:00Z"
	client := &routeRESTClient{routes: map[string]string{
		"repos/example/widgets/commits?per_page=100&since=" + date:         `[{"author":{"login":"commit-user"}},{"author":{"login":"outsider"}}]`,
		"repos/example/widgets/issues?per_page=100&since=" + date:          `[{"user":{"login":"issue-user"}}]`,
		"repos/example/widgets/issues/comments?per_page=100&since=" + date: `[{"user":{"login":"comment-user"}}]`,
		"repos/example/widgets/pulls/comments?per_page=100&since=" + date:  `[{"user":{"login":"pr-user"}}]`,
	}}
	userList := users.Users{
		{Login: "commit-user"},
		{Login: "issue-user"},
		{Login: "comment-user"},
		{Login: "pr-user"},
		{Login: "inactive-user"},
	}

	checker := NewActivityChecker(1)
	err := checker.CheckActivity(
		userList,
		"example",
		repository.Repositories{{Name: "widgets", Size: 1}},
		date,
		client,
		[]string{"commits", "issues", "issue-comments", "pr-comments"},
	)
	if err != nil {
		t.Fatalf("CheckActivity returned error: %v", err)
	}

	for i := range userList[:4] {
		if !userList[i].IsActive() {
			t.Fatalf("%s was not marked active", userList[i].Login)
		}
		types := userList[i].GetActivityTypes()
		sort.Strings(types)
		if len(types) != 1 {
			t.Fatalf("%s activity types = %v", userList[i].Login, types)
		}
	}
	if userList[4].IsActive() {
		t.Fatal("inactive user was marked active")
	}
}

func TestCheckActivityOnlyRequestsSelectedTypes(t *testing.T) {
	date := "2026-07-01T00:00:00Z"
	path := "repos/example/widgets/issues?per_page=100&since=" + date
	client := &routeRESTClient{routes: map[string]string{path: `[]`}}

	err := NewActivityChecker(1).CheckActivity(
		users.Users{{Login: "octocat"}},
		"example",
		repository.Repositories{{Name: "widgets", Size: 1}},
		date,
		client,
		[]string{"issues"},
	)
	if err != nil {
		t.Fatalf("CheckActivity returned error: %v", err)
	}
	if fmt.Sprint(client.requests) != fmt.Sprint([]string{path}) {
		t.Fatalf("requests = %v", client.requests)
	}
}

func TestCheckActivityRejectsInvalidDate(t *testing.T) {
	client := &countingRESTClient{}
	err := NewActivityChecker().CheckActivity(
		users.Users{{Login: "octocat"}},
		"example",
		repository.Repositories{{Name: "widgets", Size: 1}},
		"not-a-date",
		client,
		[]string{"issues"},
	)
	if err == nil {
		t.Fatal("CheckActivity returned nil error")
	}
	if client.requests.Load() != 0 {
		t.Fatalf("requests = %d, want 0", client.requests.Load())
	}
}

func TestCheckActivityReturnsEndpointError(t *testing.T) {
	date := "2026-07-01T00:00:00Z"
	path := "repos/example/widgets/issues?per_page=100&since=" + date
	client := &routeRESTClient{
		routes: map[string]string{},
		errs:   map[string]error{path: fmt.Errorf("boom")},
	}

	err := NewActivityChecker(1).CheckActivity(
		users.Users{{Login: "octocat"}},
		"example",
		repository.Repositories{{Name: "widgets", Size: 1}},
		date,
		client,
		[]string{"issues"},
	)
	if err == nil || !strings.Contains(err.Error(), "fetch issues for example/widgets") {
		t.Fatalf("error = %v", err)
	}
}

func TestMarkUserActiveIgnoresUnknownLogin(t *testing.T) {
	checker := NewActivityChecker()
	checker.userIndex["octocat"] = &users.User{Login: "octocat"}
	checker.activeUsers["octocat"] = false

	checker.markUserActive("outsider", "issues")
	if checker.activeUsers["octocat"] {
		t.Fatal("known user changed when marking unknown login")
	}
}
