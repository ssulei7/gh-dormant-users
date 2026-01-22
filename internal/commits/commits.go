package commits

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh/pkg/api"
	"github.com/ssulei7/gh-dormant-users/internal/header"
	"github.com/ssulei7/gh-dormant-users/internal/limiter"
	"github.com/ssulei7/gh-dormant-users/internal/ui"
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
		response, err := client.Request("GET", url, nil)
		if err != nil {
			limiter.ReleaseConcurrentLimiter()
			if strings.Contains(err.Error(), "Git Repository is empty.") {
				break
			} else {
				return nil
			}
		}

		var commits Commits
		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&commits)
		linkHeader := response.Header.Get("Link")
		response.Body.Close()
		limiter.ReleaseConcurrentLimiter()
		limiter.CheckAndHandleRateLimit(response)

		if err != nil {
			ui.Error("Failed to decode commits: %v", err)
			os.Exit(1)
		}

		allCommits = append(allCommits, commits...)

		if linkHeader == "" {
			break
		}

		nextURL := header.GetNextPageURL(linkHeader)
		if nextURL == "" {
			break
		}

		url = nextURL
	}

	return allCommits
}
