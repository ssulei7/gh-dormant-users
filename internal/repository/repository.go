package repository

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/cli/go-gh/pkg/api"
	"github.com/ssulei7/gh-dormant-users/internal/header"
	"github.com/ssulei7/gh-dormant-users/internal/limiter"
	"github.com/ssulei7/gh-dormant-users/internal/ui"
)

type Repository struct {
	Name string `json:"name"`
}

type Repositories []Repository

func GetOrgRepositories(organization string, client api.RESTClient) (Repositories, error) {
	// Start the spinner
	spinner := ui.NewSimpleSpinner("Fetching repositories...")
	spinner.Start()

	// Fetch first page
	url := fmt.Sprintf("orgs/%s/repos?per_page=100", organization)
	limiter.AcquireConcurrentLimiter()
	response, err := client.Request("GET", url, nil)
	if err != nil {
		limiter.ReleaseConcurrentLimiter()
		spinner.StopFail("Failed to fetch repositories")
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}

	var repositories Repositories
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&repositories)
	linkHeader := response.Header.Get("Link")
	response.Body.Close()
	limiter.ReleaseConcurrentLimiter()
	limiter.CheckAndHandleRateLimit(response)

	if err != nil {
		spinner.StopFail("Failed to decode repositories")
		return nil, fmt.Errorf("failed to decode repositories: %w", err)
	}

	allRepositories := make(Repositories, len(repositories))
	copy(allRepositories, repositories)

	// Get all page URLs from Link header
	var pageURLs []string
	for linkHeader != "" {
		nextURL := header.GetNextPageURL(linkHeader)
		if nextURL == "" {
			break
		}
		pageURLs = append(pageURLs, nextURL)

		// Fetch next page to get updated Link header
		limiter.AcquireConcurrentLimiter()
		response, err := client.Request("GET", nextURL, nil)
		if err != nil {
			limiter.ReleaseConcurrentLimiter()
			ui.Warning("Failed to fetch page %s: %v", nextURL, err)
			continue
		}
		linkHeader = response.Header.Get("Link")
		response.Body.Close()
		limiter.ReleaseConcurrentLimiter()
		limiter.CheckAndHandleRateLimit(response)
	}

	// Fetch remaining pages concurrently
	if len(pageURLs) > 0 {
		pageChan := make(chan string, len(pageURLs))
		resultChan := make(chan Repositories, len(pageURLs))
		var wg sync.WaitGroup
		numWorkers := 5

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for pageURL := range pageChan {
					limiter.AcquireConcurrentLimiter()
					response, err := client.Request("GET", pageURL, nil)
					if err != nil {
						limiter.ReleaseConcurrentLimiter()
						ui.Warning("Failed to fetch page %s: %v", pageURL, err)
						continue
					}

					var pageRepos Repositories
					decoder := json.NewDecoder(response.Body)
					err = decoder.Decode(&pageRepos)
					response.Body.Close()
					limiter.ReleaseConcurrentLimiter()
					limiter.CheckAndHandleRateLimit(response)

					if err != nil {
						ui.Warning("Failed to decode page %s: %v", pageURL, err)
						continue
					}
					resultChan <- pageRepos
				}
			}()
		}

		for _, pageURL := range pageURLs {
			pageChan <- pageURL
		}
		close(pageChan)
		wg.Wait()
		close(resultChan)

		for pageRepos := range resultChan {
			allRepositories = append(allRepositories, pageRepos...)
		}
	}

	spinner.Stop("Fetched repositories successfully")
	ui.Info("Fetched %d repositories", len(allRepositories))
	return allRepositories, nil
}
