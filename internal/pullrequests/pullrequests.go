package pullrequests

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/cli/go-gh/pkg/api"
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
		var response *http.Response
		var err error

		// Retry logic for handling 502 errors
		for retries := 0; retries < 5; retries++ {
			response, err = client.Request("GET", url, nil)
			if err == nil || !strings.Contains(err.Error(), "502") {
				break
			}
			log.Printf("Received 502 error. Retrying in %d seconds...", (1 << retries))
			time.Sleep(time.Duration(1<<retries) * time.Second)
		}
		limiter.ReleaseConcurrentLimiter()

		if err != nil {
			if strings.Contains(err.Error(), "Git Repository is empty.") {
				log.Printf("Repository %s is empty", repo)
				break
			} else {
				log.Printf("Failed to fetch pull request comments: %v", err)
				return nil
			}
		}

		var pullRequestComments PullRequestComments

		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&pullRequestComments)
		if err != nil {
			log.Fatalf("Failed to decode pull request comments: %v", err)
		}

		allPullRequestComments = append(allPullRequestComments, pullRequestComments...)

		// Check for the 'Link' header to see if there are more pages
		linkHeader := response.Header.Get("Link")
		if linkHeader == "" {
			break
		}

		nextURL := header.GetNextPageURL(linkHeader)
		if nextURL == "" {
			log.Printf("No next page URL found")
			break
		}

		url = nextURL
	}

	return allPullRequestComments
}
