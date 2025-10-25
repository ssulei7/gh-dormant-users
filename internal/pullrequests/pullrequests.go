package pullrequests

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cli/go-gh/pkg/api"
	"github.com/pterm/pterm"
	"github.com/ssulei7/gh-dormant-users/internal/header"
	"github.com/ssulei7/gh-dormant-users/internal/limiter"
)

type PullRequestComment struct {
	ID        int    `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

type PullRequestComments []PullRequestComment

func GetPullRequestCommentsSinceDate(organization string, repo string, date string, client api.RESTClient) PullRequestComments {
	var allPullRequestComments PullRequestComments
	url := fmt.Sprintf("repos/%s/%s/pulls/comments?per_page=100&since=%s", organization, repo, date)
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

		limiter.CheckAndHandleRateLimit(response)
		limiter.ReleaseConcurrentLimiter()

		var pullRequestComments PullRequestComments

		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&pullRequestComments)
		linkHeader := response.Header.Get("Link")
		response.Body.Close()

		if err != nil {
			pterm.Error.Printf("Failed to decode pull request comments: %v\n", err)
			os.Exit(1)
		}

		allPullRequestComments = append(allPullRequestComments, pullRequestComments...)

		if linkHeader == "" {
			break
		}

		nextURL := header.GetNextPageURL(linkHeader)
		if nextURL == "" {
			break
		}

		url = nextURL
	}

	return allPullRequestComments
}
