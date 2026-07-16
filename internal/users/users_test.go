package users

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"testing"

	"github.com/cli/go-gh/pkg/api"
)

type mockGQLClient struct {
	calls int
}

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
