package commits

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type mockRESTClient struct {
	path string
	body string
	err  error
}

func (m *mockRESTClient) Request(_ string, path string, _ io.Reader) (*http.Response, error) {
	m.path = path
	if m.err != nil {
		return nil, m.err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(m.body)),
	}, nil
}

func (m *mockRESTClient) RequestWithContext(_ context.Context, method, path string, body io.Reader) (*http.Response, error) {
	return m.Request(method, path, body)
}

func (m *mockRESTClient) Do(method, path string, body io.Reader, result interface{}) error {
	response, err := m.Request(method, path, body)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return json.NewDecoder(response.Body).Decode(result)
}

func (m *mockRESTClient) DoWithContext(_ context.Context, method, path string, body io.Reader, result interface{}) error {
	return m.Do(method, path, body, result)
}

func (m *mockRESTClient) Delete(path string, result interface{}) error {
	return m.Do(http.MethodDelete, path, nil, result)
}

func (m *mockRESTClient) Get(path string, result interface{}) error {
	return m.Do(http.MethodGet, path, nil, result)
}

func (m *mockRESTClient) Patch(path string, body io.Reader, result interface{}) error {
	return m.Do(http.MethodPatch, path, body, result)
}

func (m *mockRESTClient) Post(path string, body io.Reader, result interface{}) error {
	return m.Do(http.MethodPost, path, body, result)
}

func (m *mockRESTClient) Put(path string, body io.Reader, result interface{}) error {
	return m.Do(http.MethodPut, path, body, result)
}

func (m *mockRESTClient) RESTPrefix() string { return "" }

func TestGetCommitsSinceDate(t *testing.T) {
	t.Parallel()

	client := &mockRESTClient{body: `[{"sha":"abc123","author":{"login":"octocat"}}]`}
	commits, err := GetCommitsSinceDate("example", "widgets", "2026-07-01T00:00:00Z", client)
	if err != nil {
		t.Fatalf("GetCommitsSinceDate returned error: %v", err)
	}
	if client.path != "repos/example/widgets/commits?per_page=100&since=2026-07-01T00:00:00Z" {
		t.Fatalf("request path = %q", client.path)
	}
	if len(commits) != 1 || commits[0].Sha != "abc123" || commits[0].Author.Login != "octocat" {
		t.Fatalf("commits = %#v", commits)
	}
}

func TestGetCommitsSinceDateEmptyRepository(t *testing.T) {
	t.Parallel()

	client := &mockRESTClient{err: errors.New("Git Repository is empty.")}
	commits, err := GetCommitsSinceDate("example", "empty", "2026-07-01T00:00:00Z", client)
	if err != nil {
		t.Fatalf("GetCommitsSinceDate returned error: %v", err)
	}
	if commits != nil {
		t.Fatalf("commits = %#v, want nil", commits)
	}
}

func TestGetCommitsSinceDateWrapsError(t *testing.T) {
	t.Parallel()

	client := &mockRESTClient{err: errors.New("boom")}
	_, err := GetCommitsSinceDate("example", "widgets", "2026-07-01T00:00:00Z", client)
	if err == nil || !strings.Contains(err.Error(), "fetch commits for example/widgets") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %v", err)
	}
}
