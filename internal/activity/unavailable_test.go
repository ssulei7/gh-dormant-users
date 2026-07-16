package activity

import (
	"net/http"
	"testing"
	"time"

	"github.com/cli/go-gh/pkg/api"
	"github.com/ssulei7/gh-dormant-users/internal/repository"
	"github.com/ssulei7/gh-dormant-users/internal/users"
)

func TestCheckActivityContinuesWhenRepositoryEndpointIsUnavailable(t *testing.T) {
	client := &countingRESTClient{
		err: api.HTTPError{StatusCode: http.StatusNotFound},
	}
	checker := NewActivityChecker()
	err := checker.CheckActivity(
		users.Users{{Login: "octocat"}},
		"example",
		repository.Repositories{{Name: "removed-repository", Size: 1}},
		time.Now().UTC().Format(time.RFC3339),
		client,
		[]string{"issue-comments"},
	)
	if err != nil {
		t.Fatalf("CheckActivity returned error: %v", err)
	}
	if client.requests.Load() != 1 {
		t.Fatalf("expected one attempted request, got %d", client.requests.Load())
	}
}
