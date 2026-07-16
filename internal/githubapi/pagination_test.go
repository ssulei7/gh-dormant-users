package githubapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"
)

type mockRESTClient struct {
	responses map[string]*http.Response
	requests  map[string]int
}

func (m *mockRESTClient) Request(_ string, path string, _ io.Reader) (*http.Response, error) {
	m.requests[path]++
	response, ok := m.responses[path]
	if !ok {
		return nil, fmt.Errorf("unexpected request: %s", path)
	}
	copy := *response
	copy.Header = response.Header.Clone()
	body, _ := io.ReadAll(response.Body)
	response.Body = io.NopCloser(bytes.NewReader(body))
	copy.Body = io.NopCloser(bytes.NewReader(body))
	return &copy, nil
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

func (m *mockRESTClient) RESTPrefix() string {
	return ""
}

func TestGetAllRequestsEachPageOnce(t *testing.T) {
	first := "items?per_page=100"
	second := "https://api.github.com/items?page=2"
	third := "https://api.github.com/items?page=3"
	client := &mockRESTClient{
		requests: make(map[string]int),
		responses: map[string]*http.Response{
			first:  jsonResponse(`[{"name":"one"}]`, fmt.Sprintf("<%s>; rel=\"next\", <%s>; rel=\"last\"", second, third)),
			second: jsonResponse(`[{"name":"two"}]`, fmt.Sprintf("<%s>; rel=\"next\", <%s>; rel=\"last\"", third, third)),
			third:  jsonResponse(`[{"name":"three"}]`, ""),
		},
	}

	type item struct {
		Name string `json:"name"`
	}
	items, err := GetAll[item](client, first)
	if err != nil {
		t.Fatalf("GetAll returned error: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	for _, url := range []string{first, second, third} {
		if client.requests[url] != 1 {
			t.Fatalf("expected %s to be requested once, got %d", url, client.requests[url])
		}
	}

}

func jsonResponse(body, link string) *http.Response {
	header := make(http.Header)
	if link != "" {
		header.Set("Link", link)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     header,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}
