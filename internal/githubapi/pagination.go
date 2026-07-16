package githubapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/cli/go-gh/pkg/api"
	"github.com/ssulei7/gh-dormant-users/internal/header"
)

func GetAll[T any](client api.RESTClient, url string) ([]T, error) {
	var all []T
	for url != "" {
		response, err := client.Request(http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("request %s: %w", url, err)
		}

		var page []T
		decodeErr := json.NewDecoder(response.Body).Decode(&page)
		closeErr := response.Body.Close()
		if decodeErr != nil {
			return nil, fmt.Errorf("decode %s: %w", url, decodeErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close %s response: %w", url, closeErr)
		}

		all = append(all, page...)
		url = header.GetNextPageURL(response.Header.Get("Link"))
	}
	return all, nil
}
