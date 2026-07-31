package pullrequests

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

func TestGetPullRequestCommentsSinceDate(t *testing.T) {
	t.Parallel()

	client := &mockRESTClient{body: `[{"id":3,"user":{"login":"octocat"}}]`}
	comments, err := GetPullRequestCommentsSinceDate("example", "widgets", "2026-07-01T00:00:00Z", client)
	if err != nil {
		t.Fatalf("GetPullRequestCommentsSinceDate returned error: %v", err)
	}
	if client.path != "repos/example/widgets/pulls/comments?per_page=100&since=2026-07-01T00:00:00Z" {
		t.Fatalf("request path = %q", client.path)
	}
	if len(comments) != 1 || comments[0].ID != 3 || comments[0].User.Login != "octocat" {
		t.Fatalf("comments = %#v", comments)
	}
}

func TestGetPullRequestCommentsSinceDateEmptyRepository(t *testing.T) {
	t.Parallel()

	client := &mockRESTClient{err: errors.New("Git Repository is empty.")}
	comments, err := GetPullRequestCommentsSinceDate("example", "empty", "2026-07-01T00:00:00Z", client)
	if err != nil {
		t.Fatalf("GetPullRequestCommentsSinceDate returned error: %v", err)
	}
	if comments != nil {
		t.Fatalf("comments = %#v, want nil", comments)
	}
}

func TestGetPullRequestCommentsSinceDateWrapsError(t *testing.T) {
	t.Parallel()

	client := &mockRESTClient{err: errors.New("boom")}
	_, err := GetPullRequestCommentsSinceDate("example", "widgets", "2026-07-01T00:00:00Z", client)
	if err == nil || !strings.Contains(err.Error(), "fetch pull request comments for example/widgets") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %v", err)
	}
}
