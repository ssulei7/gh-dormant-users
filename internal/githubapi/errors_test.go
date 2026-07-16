package githubapi

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/cli/go-gh/pkg/api"
)

func TestIsRepositoryUnavailable(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusConflict} {
		err := fmt.Errorf("wrapped: %w", api.HTTPError{StatusCode: status})
		if !IsRepositoryUnavailable(err) {
			t.Fatalf("expected status %d to be repository-unavailable", status)
		}
	}
	if IsRepositoryUnavailable(api.HTTPError{StatusCode: http.StatusForbidden}) {
		t.Fatal("403 must not be treated as repository-unavailable")
	}
}
