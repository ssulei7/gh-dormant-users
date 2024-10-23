package commits

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/cli/go-gh/pkg/api"
	"github.com/ssulei7/gh-dormant-users/config"
	"github.com/ssulei7/gh-dormant-users/internal/header"
	"github.com/ssulei7/gh-dormant-users/internal/limiter"
)

type Commit struct {
	Sha    string `json:"sha"`
	Commit struct {
		Author struct {
			Name  string `json:"name"`
			Email string `json:"email"`
			Date  string `json:"date"`
		} `json:"author"`
	} `json:"commit"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
}

type Commits []Commit

func GetCommitsSinceDate(organization string, repository string, date string, client api.RESTClient) Commits {
	var allCommits Commits
	url := fmt.Sprintf("repos/%s/%s/commits?per_page=100&since=%s", organization, repository, date)
	for {
		limiter.AcquireConcurrentLimiter()
		defer limiter.ReleaseConcurrentLimiter()
		response, err := client.Request("GET", url, nil)
		if err != nil {
			if strings.Contains(err.Error(), "Git Repository is empty.") {
				log.Printf("Repository %s is empty", repository)
				break
			} else {
				log.Printf("Failed to fetch commits: %v", err)
				return nil
			}
		}

		var commits Commits
		decoder := json.NewDecoder(response.Body)

		err = decoder.Decode(&commits)
		if err != nil {
			log.Fatalf("Failed to decode commits: %v", err)
		}

		allCommits = append(allCommits, commits...)

		// Check for the 'Link' header to see if there are more pages
		linkHeader := response.Header.Get("Link")
		if linkHeader == "" {
			if config.Verbose {
				log.Printf("No more pages to fetch")
			}
			break
		}

		nextURL := header.GetNextPageURL(linkHeader)
		if nextURL == "" {
			log.Printf("No next page URL found")
			break
		}

		url = nextURL
	}

	return allCommits
}
