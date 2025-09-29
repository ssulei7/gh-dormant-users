package repository

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cli/go-gh/pkg/api"
	"github.com/pterm/pterm"
	"github.com/ssulei7/gh-dormant-users/internal/header"
	"github.com/ssulei7/gh-dormant-users/internal/limiter"
)

type Repository struct {
	Name string `json:"name"`
}

type Repositories []Repository

func GetOrgRepositories(organization string, client api.RESTClient) Repositories {
	var allRepositories Repositories

	// Start the spinner
	spinner, _ := pterm.DefaultSpinner.Start("Fetching repositories...")

	url := fmt.Sprintf("orgs/%s/repos?per_page=100", organization)
	for {
		var response *http.Response
		var err error
		for retries := 0; retries < 5; retries++ {
			response, err = client.Request("GET", url, nil)
			if err != nil {
				pterm.Warning.Printf("Failed to fetch repositories: %v. Retrying in %d seconds...\n", err, (1 << retries))
				time.Sleep(time.Duration(1<<retries) * time.Second)
				continue
			}
			// Check and handle rate limits from headers
			limiter.CheckAndHandleRateLimit(response)
			break
		}
		if err != nil {
			spinner.Fail("Failed to fetch repositories after retries")
			pterm.Fatal.Printf("Failed to fetch repositories after retries: %v\n", err)
		}

		var repositories Repositories
		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&repositories)
		if err != nil {
			spinner.Fail("Failed to decode repositories")
			pterm.Fatal.Printf("Failed to decode repositories: %v\n", err)
		}

		allRepositories = append(allRepositories, repositories...)

		// Check for the 'Link' header to see if there are more pages
		linkHeader := response.Header.Get("Link")
		if linkHeader == "" {
			break
		}

		nextURL := header.GetNextPageURL(linkHeader)
		if nextURL == "" {
			break
		}

		url = nextURL
	}

	spinner.Success("Fetched repositories successfully")
	pterm.Info.Printf("Fetched %d repositories\n", len(allRepositories))
	return allRepositories
}
