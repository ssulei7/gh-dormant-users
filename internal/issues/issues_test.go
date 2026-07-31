package issues

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

func TestGetIssuesSinceDate(t *testing.T) {
	t.Parallel()

	client := &mockRESTClient{body: `[{"id":1,"title":"Bug","user":{"login":"octocat"}}]`}
	issues, err := GetIssuesSinceDate("example", "widgets", "2026-07-01T00:00:00Z", client)
	if err != nil {
		t.Fatalf("GetIssuesSinceDate returned error: %v", err)
	}
	if client.path != "repos/example/widgets/issues?per_page=100&since=2026-07-01T00:00:00Z" {
		t.Fatalf("request path = %q", client.path)
	}
	if len(issues) != 1 || issues[0].ID != 1 || issues[0].User.Login != "octocat" {
		t.Fatalf("issues = %#v", issues)
	}
}

func TestGetIssueCommentsSinceDate(t *testing.T) {
	t.Parallel()

	client := &mockRESTClient{body: `[{"id":2,"user":{"login":"hubot"}}]`}
	comments, err := GetIssueCommentsSinceDate("example", "widgets", "2026-07-01T00:00:00Z", client)
	if err != nil {
		t.Fatalf("GetIssueCommentsSinceDate returned error: %v", err)
	}
	if client.path != "repos/example/widgets/issues/comments?per_page=100&since=2026-07-01T00:00:00Z" {
		t.Fatalf("request path = %q", client.path)
	}
	if len(comments) != 1 || comments[0].ID != 2 || comments[0].User.Login != "hubot" {
		t.Fatalf("comments = %#v", comments)
	}
}

func TestIssueEndpointsHandleEmptyRepository(t *testing.T) {
	t.Parallel()

	client := &mockRESTClient{err: errors.New("Git Repository is empty.")}
	issues, err := GetIssuesSinceDate("example", "empty", "2026-07-01T00:00:00Z", client)
	if err != nil || issues != nil {
		t.Fatalf("issues = %#v, error = %v", issues, err)
	}
	comments, err := GetIssueCommentsSinceDate("example", "empty", "2026-07-01T00:00:00Z", client)
	if err != nil || comments != nil {
		t.Fatalf("comments = %#v, error = %v", comments, err)
	}
}

func TestIssueEndpointsWrapErrors(t *testing.T) {
	t.Parallel()

	client := &mockRESTClient{err: errors.New("boom")}
	_, issueErr := GetIssuesSinceDate("example", "widgets", "2026-07-01T00:00:00Z", client)
	if issueErr == nil || !strings.Contains(issueErr.Error(), "fetch issues for example/widgets") {
		t.Fatalf("issue error = %v", issueErr)
	}
	_, commentErr := GetIssueCommentsSinceDate("example", "widgets", "2026-07-01T00:00:00Z", client)
	if commentErr == nil || !strings.Contains(commentErr.Error(), "fetch issue comments for example/widgets") {
		t.Fatalf("comment error = %v", commentErr)
	}
}
