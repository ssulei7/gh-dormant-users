package issues

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

type Issue struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	User  struct {
		Login string `json:"login"`
	} `json:"user"`
	CreatedAt string `json:"created_at"`
}

type IssueComment struct {
	ID        int    `json:"id"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
}

type IssueComments []IssueComment

type Issues []Issue

func GetIssuesSinceDate(organization string, repo string, date string, client api.RESTClient) Issues {
	var allIssues Issues
	url := fmt.Sprintf("repos/%s/%s/issues?per_page=100&since=%s", organization, repo, date)
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

		var issues Issues

		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&issues)
		linkHeader := response.Header.Get("Link")
		response.Body.Close()

		if err != nil {
			pterm.Error.Printf("Failed to decode issues: %v\n", err)
			os.Exit(1)
		}

		allIssues = append(allIssues, issues...)

		if linkHeader == "" {
			break
		}

		nextURL := header.GetNextPageURL(linkHeader)
		if nextURL == "" {
			break
		}

		url = nextURL
	}

	return allIssues
}

func GetIssueCommentsSinceDate(organization string, repo string, date string, client api.RESTClient) IssueComments {
	var allIssueComments IssueComments
	url := fmt.Sprintf("repos/%s/%s/issues/comments?per_page=100&since=%s", organization, repo, date)
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

		var issueComments IssueComments

		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&issueComments)
		linkHeader := response.Header.Get("Link")
		response.Body.Close()

		if err != nil {
			pterm.Error.Printf("Failed to decode issue comments: %v\n", err)
			os.Exit(1)
		}

		allIssueComments = append(allIssueComments, issueComments...)

		if linkHeader == "" {
			break
		}

		nextURL := header.GetNextPageURL(linkHeader)
		if nextURL == "" {
			break
		}

		url = nextURL
	}

	return allIssueComments
}
