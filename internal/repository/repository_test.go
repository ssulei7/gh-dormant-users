package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"testing"
)

type mockRESTClient struct {
	responders map[string]func() (*http.Response, error)
}

func (m *mockRESTClient) Request(method string, path string, body io.Reader) (*http.Response, error) {
	if responder, ok := m.responders[path]; ok {
		return responder()
	}
	return nil, fmt.Errorf("unexpected request: %s", path)
}

func (m *mockRESTClient) RequestWithContext(ctx context.Context, method string, path string, body io.Reader) (*http.Response, error) {
	return m.Request(method, path, body)
}

func (m *mockRESTClient) Do(method string, path string, body io.Reader, resp interface{}) error {
	response, err := m.Request(method, path, body)
	if err != nil {
		return err
	}
	if response == nil || resp == nil {
		return nil
	}
	defer response.Body.Close()
	return json.NewDecoder(response.Body).Decode(resp)
}

func (m *mockRESTClient) DoWithContext(ctx context.Context, method string, path string, body io.Reader, resp interface{}) error {
	return m.Do(method, path, body, resp)
}

func (m *mockRESTClient) Delete(path string, resp interface{}) error {
	_, err := m.Request("DELETE", path, nil)
	return err
}

func (m *mockRESTClient) Get(path string, resp interface{}) error {
	_, err := m.Request("GET", path, nil)
	return err
}

func (m *mockRESTClient) Patch(path string, body io.Reader, resp interface{}) error {
	_, err := m.Request("PATCH", path, body)
	return err
}

func (m *mockRESTClient) Post(path string, body io.Reader, resp interface{}) error {
	_, err := m.Request("POST", path, body)
	return err
}

func (m *mockRESTClient) Put(path string, body io.Reader, resp interface{}) error {
	_, err := m.Request("PUT", path, body)
	return err
}

func (m *mockRESTClient) RESTPrefix() string {
	return ""
}

func responseWithJSON(body string, linkHeader string) *http.Response {
	headers := http.Header{}
	if linkHeader != "" {
		headers.Set("Link", linkHeader)
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     headers,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

func sortedRepoNames(repos Repositories) []string {
	names := make([]string, 0, len(repos))
	for _, repo := range repos {
		names = append(names, repo.Name)
	}
	sort.Strings(names)
	return names
}

func TestGetOrgRepositories_SinglePage(t *testing.T) {
	org := "test-org"
	url := fmt.Sprintf("orgs/%s/repos?per_page=100", org)
	jsonBody := `[{"name":"repo-one"},{"name":"repo-two"}]`

	mockClient := &mockRESTClient{
		responders: map[string]func() (*http.Response, error){
			url: func() (*http.Response, error) {
				return responseWithJSON(jsonBody, ""), nil
			},
		},
	}

	repos, err := GetOrgRepositories(org, mockClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	names := sortedRepoNames(repos)

	expected := []string{"repo-one", "repo-two"}
	if fmt.Sprint(names) != fmt.Sprint(expected) {
		t.Fatalf("expected repos %v, got %v", expected, names)
	}
}

func TestGetOrgRepositories_MultiplePages(t *testing.T) {
	org := "test-org"
	firstURL := fmt.Sprintf("orgs/%s/repos?per_page=100", org)
	page2URL := "https://api.github.com/organizations/1/repos?page=2"
	page3URL := "https://api.github.com/organizations/1/repos?page=3"
	linkHeader := fmt.Sprintf("<%s>; rel=\"next\", <%s>; rel=\"last\"", page2URL, page3URL)
	page2LinkHeader := fmt.Sprintf("<%s>; rel=\"next\", <%s>; rel=\"last\"", page3URL, page3URL)

	mockClient := &mockRESTClient{
		responders: map[string]func() (*http.Response, error){
			firstURL: func() (*http.Response, error) {
				return responseWithJSON(`[{"name":"repo-one"}]`, linkHeader), nil
			},
			page2URL: func() (*http.Response, error) {
				return responseWithJSON(`[{"name":"repo-two"}]`, page2LinkHeader), nil
			},
			page3URL: func() (*http.Response, error) {
				return responseWithJSON(`[{"name":"repo-three"}]`, ""), nil
			},
		},
	}

	repos, err := GetOrgRepositories(org, mockClient)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	names := sortedRepoNames(repos)

	expected := []string{"repo-one", "repo-three", "repo-two"}
	if fmt.Sprint(names) != fmt.Sprint(expected) {
		t.Fatalf("expected repos %v, got %v", expected, names)
	}
}
