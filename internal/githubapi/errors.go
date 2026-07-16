package githubapi

import (
	"errors"
	"net/http"

	"github.com/cli/go-gh/pkg/api"
)

func IsRepositoryUnavailable(err error) bool {
	var httpError api.HTTPError
	if !errors.As(err, &httpError) {
		return false
	}
	return httpError.StatusCode == http.StatusNotFound || httpError.StatusCode == http.StatusConflict
}
