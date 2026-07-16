package activity

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
