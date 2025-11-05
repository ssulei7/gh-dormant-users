package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

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
	// Start the spinner
	spinner, _ := pterm.DefaultSpinner.Start("Fetching repositories...")

	// Fetch first page
	url := fmt.Sprintf("orgs/%s/repos?per_page=100", organization)
	if err := limiter.WaitForTokenAndAcquire(context.Background()); err != nil {
		spinner.Fail("Failed to acquire rate limit token")
		pterm.Error.Printf("Failed to acquire rate limit token: %v\n", err)
		os.Exit(1)
	}
	
	response, err := client.Request("GET", url, nil)
	if err != nil {
		limiter.ReleaseConcurrentLimiter()
		spinner.Fail("Failed to fetch repositories")
		pterm.Error.Printf("Failed to fetch repositories: %v\n", err)
		os.Exit(1)
	}

	var repositories Repositories
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&repositories)
	linkHeader := response.Header.Get("Link")
	response.Body.Close()

	limiter.ReleaseAndHandleRateLimit(response)

	if err != nil {
		spinner.Fail("Failed to decode repositories")
		pterm.Error.Printf("Failed to decode repositories: %v\n", err)
		os.Exit(1)
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
		if err := limiter.WaitForTokenAndAcquire(context.Background()); err != nil {
			continue
		}
		
		response, err := client.Request("GET", nextURL, nil)
		if err != nil {
			limiter.ReleaseConcurrentLimiter()
			continue
		}
		linkHeader = response.Header.Get("Link")
		response.Body.Close()
		limiter.ReleaseAndHandleRateLimit(response)
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
					if err := limiter.WaitForTokenAndAcquire(context.Background()); err != nil {
						continue
					}
					
					response, err := client.Request("GET", pageURL, nil)
					if err != nil {
						limiter.ReleaseConcurrentLimiter()
						continue
					}
					limiter.CheckAndHandleRateLimit(response)
					limiter.ReleaseConcurrentLimiter()

					var pageRepos Repositories
					decoder := json.NewDecoder(response.Body)
					err = decoder.Decode(&pageRepos)
					response.Body.Close()

					limiter.ReleaseAndHandleRateLimit(response)

					if err != nil {
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

	spinner.Success("Fetched repositories successfully")
	pterm.Info.Printf("Fetched %d repositories\n", len(allRepositories))
	return allRepositories
}
