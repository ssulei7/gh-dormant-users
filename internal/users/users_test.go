package users

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/cli/go-gh/pkg/api"
)

type mockGQLClient struct {
	calls int
}

type mockRESTClient struct {
	body string
	err  error
	path string
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

func (m *mockRESTClient) Do(method, path string, body io.Reader, response interface{}) error {
	httpResponse, err := m.Request(method, path, body)
	if err != nil {
		return err
	}
	defer httpResponse.Body.Close()
	return json.NewDecoder(httpResponse.Body).Decode(response)
}

func (m *mockRESTClient) DoWithContext(_ context.Context, method, path string, body io.Reader, response interface{}) error {
	return m.Do(method, path, body, response)
}

func (m *mockRESTClient) Delete(path string, response interface{}) error {
	return m.Do(http.MethodDelete, path, nil, response)
}

func (m *mockRESTClient) Get(path string, response interface{}) error {
	return m.Do(http.MethodGet, path, nil, response)
}

func (m *mockRESTClient) Patch(path string, body io.Reader, response interface{}) error {
	return m.Do(http.MethodPatch, path, body, response)
}

func (m *mockRESTClient) Post(path string, body io.Reader, response interface{}) error {
	return m.Do(http.MethodPost, path, body, response)
}

func (m *mockRESTClient) Put(path string, body io.Reader, response interface{}) error {
	return m.Do(http.MethodPut, path, body, response)
}

func (m *mockRESTClient) RESTPrefix() string { return "" }

func (m *mockGQLClient) Do(_ string, variables map[string]interface{}, response interface{}) error {
	m.calls++
	result := make(map[string]map[string]string, len(variables))
	for index := 0; index < len(variables); index++ {
		login := fmt.Sprint(variables[fmt.Sprintf("login%d", index)])
		result[fmt.Sprintf("user%d", index)] = map[string]string{
			"login": login,
			"email": login + "@example.com",
		}
	}
	data, _ := json.Marshal(result)
	return json.Unmarshal(data, response)
}

func (m *mockGQLClient) DoWithContext(_ context.Context, query string, variables map[string]interface{}, response interface{}) error {
	return m.Do(query, variables, response)
}

func (m *mockGQLClient) Mutate(_ string, _ interface{}, _ map[string]interface{}) error {
	return nil
}

func (m *mockGQLClient) MutateWithContext(_ context.Context, _ string, _ interface{}, _ map[string]interface{}) error {
	return nil
}

func (m *mockGQLClient) Query(_ string, _ interface{}, _ map[string]interface{}) error {
	return nil
}

func (m *mockGQLClient) QueryWithContext(_ context.Context, _ string, _ interface{}, _ map[string]interface{}) error {
	return nil
}

func TestGetUserEmailsBatchesGraphQLRequests(t *testing.T) {
	userList := make(Users, 3000)
	for index := range userList {
		userList[index].Login = fmt.Sprintf("user-%d", index)
	}
	client := &mockGQLClient{}

	if err := getUserEmails(userList, client); err != nil {
		t.Fatalf("getUserEmails returned error: %v", err)
	}
	if client.calls != 60 {
		t.Fatalf("expected 60 GraphQL requests, got %d", client.calls)
	}
	for index := range userList {
		user := &userList[index]
		expected := user.Login + "@example.com"
		if user.Email != expected {
			t.Fatalf("expected %s email %s, got %s", user.Login, expected, user.Email)
		}
	}
}

func TestGetUserEmailsKeepsPartialGraphQLData(t *testing.T) {
	userList := Users{{Login: "available"}, {Login: "removed"}}
	client := &partialErrorGQLClient{}

	if err := getUserEmails(userList, client); err != nil {
		t.Fatalf("getUserEmails returned error: %v", err)
	}
	if userList[0].Email != "available@example.com" {
		t.Fatalf("expected available user's email, got %q", userList[0].Email)
	}
	if userList[1].Email != "" {
		t.Fatalf("expected removed user's email to remain empty, got %q", userList[1].Email)
	}
}

func TestEmailQueryUsesVariables(t *testing.T) {
	client := &capturingGQLClient{}
	userList := Users{{Login: `quote"login`}}
	if err := getUserEmails(userList, client); err != nil {
		t.Fatalf("getUserEmails returned error: %v", err)
	}
	if regexp.MustCompile(`quote`).MatchString(client.query) {
		t.Fatalf("query embedded a login instead of using variables: %s", client.query)
	}
}

func TestUserActivityState(t *testing.T) {
	t.Parallel()

	user := User{Login: "octocat"}
	if user.IsActive() {
		t.Fatal("new user should be inactive")
	}

	user.MakeActive()
	if !user.IsActive() {
		t.Fatal("MakeActive did not activate user")
	}

	user.AddActivityType("issues")
	user.AddActivityType("issues")
	user.MarkActiveWithType("commits")
	got := user.GetActivityTypes()
	sort.Strings(got)
	if fmt.Sprint(got) != fmt.Sprint([]string{"commits", "issues"}) {
		t.Fatalf("activity types = %v", got)
	}

	user.MakeInactive()
	if user.IsActive() {
		t.Fatal("MakeInactive did not deactivate user")
	}
}

func TestGetOrganizationUsersWithoutEmail(t *testing.T) {
	t.Parallel()

	restClient := &mockRESTClient{body: `[{"login":"octocat","id":1}]`}
	userList, err := GetOrganizationUsers("example", false, restClient, nil)
	if err != nil {
		t.Fatalf("GetOrganizationUsers returned error: %v", err)
	}
	if restClient.path != "orgs/example/members?per_page=100" {
		t.Fatalf("request path = %q", restClient.path)
	}
	if len(userList) != 1 || userList[0].Login != "octocat" {
		t.Fatalf("users = %#v", userList)
	}
}

func TestGetOrganizationUsersWithEmail(t *testing.T) {
	t.Parallel()

	restClient := &mockRESTClient{body: `[{"login":"octocat","id":1}]`}
	gqlClient := &mockGQLClient{}
	userList, err := GetOrganizationUsers("example", true, restClient, gqlClient)
	if err != nil {
		t.Fatalf("GetOrganizationUsers returned error: %v", err)
	}
	if userList[0].Email != "octocat@example.com" {
		t.Fatalf("email = %q", userList[0].Email)
	}
}

func TestGetOrganizationUsersRequiresGraphQLForEmail(t *testing.T) {
	t.Parallel()

	restClient := &mockRESTClient{body: `[{"login":"octocat","id":1}]`}
	_, err := GetOrganizationUsers("example", true, restClient, nil)
	if err == nil || !strings.Contains(err.Error(), "GraphQL client is required") {
		t.Fatalf("error = %v", err)
	}
}

func TestGetOrganizationUsersWrapsRESTError(t *testing.T) {
	t.Parallel()

	restClient := &mockRESTClient{err: errors.New("boom")}
	_, err := GetOrganizationUsers("example", false, restClient, nil)
	if err == nil || !strings.Contains(err.Error(), "fetch organization users") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("error = %v", err)
	}
}

func TestGetUserEmailsWrapsBatchError(t *testing.T) {
	t.Parallel()

	userList := Users{{Login: "octocat"}}
	err := getUserEmails(userList, &errorGQLClient{err: errors.New("boom")})
	if err == nil || !strings.Contains(err.Error(), "fetch user email batch starting at 0") {
		t.Fatalf("error = %v", err)
	}
}

func TestPartialUserLookupErrorValidation(t *testing.T) {
	t.Parallel()

	aliases := map[string]struct{}{"user0": {}}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "non GraphQL error", err: errors.New("boom")},
		{name: "empty GraphQL errors", err: api.GQLError{}},
		{name: "missing path", err: api.GQLError{Errors: []api.GQLErrorItem{{Message: "missing"}}}},
		{name: "non string alias", err: api.GQLError{Errors: []api.GQLErrorItem{{Path: []interface{}{1}}}}},
		{name: "unknown alias", err: api.GQLError{Errors: []api.GQLErrorItem{{Path: []interface{}{"user1"}}}}},
		{name: "known alias", err: api.GQLError{Errors: []api.GQLErrorItem{{Path: []interface{}{"user0"}}}}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isPartialUserLookupError(tt.err, aliases); got != tt.want {
				t.Fatalf("isPartialUserLookupError() = %v, want %v", got, tt.want)
			}
		})
	}
}

type capturingGQLClient struct {
	mockGQLClient
	query string
}

func (c *capturingGQLClient) Do(query string, variables map[string]interface{}, response interface{}) error {
	c.query = query
	return c.mockGQLClient.Do(query, variables, response)
}

type partialErrorGQLClient struct {
	mockGQLClient
}

func (c *partialErrorGQLClient) Do(query string, variables map[string]interface{}, response interface{}) error {
	data := []byte(`{"user0":{"login":"available","email":"available@example.com"},"user1":null}`)
	if err := json.Unmarshal(data, response); err != nil {
		return err
	}
	return api.GQLError{Errors: []api.GQLErrorItem{{
		Message: "Could not resolve to a User",
		Path:    []interface{}{"user1"},
		Type:    "NOT_FOUND",
	}}}
}

type errorGQLClient struct {
	mockGQLClient
	err error
}

func (c *errorGQLClient) Do(_ string, _ map[string]interface{}, _ interface{}) error {
	return c.err
}
