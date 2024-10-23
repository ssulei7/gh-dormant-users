package issues

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
		defer limiter.ReleaseConcurrentLimiter()
		response, err := client.Request("GET", url, nil)
		if err != nil {
			if strings.Contains(err.Error(), "Git Repository is empty.") {
				log.Printf("Repository %s is empty", repo)
				break
			} else {
				log.Printf("Failed to fetch issues: %v", err)
				return nil
			}
		}

		var issues Issues

		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&issues)
		if err != nil {
			log.Fatalf("Failed to decode issues: %v", err)
		}

		allIssues = append(allIssues, issues...)

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

	return allIssues
}

func GetIssueCommentsSinceDate(organization string, repo string, date string, client api.RESTClient) IssueComments {
	var allIssueComments IssueComments
	url := fmt.Sprintf("repos/%s/%s/issues/comments?per_page=100&since=%s", organization, repo, date)
	for {
		limiter.AcquireConcurrentLimiter()
		defer limiter.ReleaseConcurrentLimiter()
		response, err := client.Request("GET", url, nil)
		if err != nil {
			if strings.Contains(err.Error(), "Git Repository is empty.") {
				log.Printf("Repository %s is empty", repo)
				break
			} else {
				log.Printf("Failed to fetch issues: %v", err)
				return nil
			}
		}

		var issueComments IssueComments

		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&issueComments)
		if err != nil {
			log.Fatalf("Failed to decode issues: %v", err)
		}

		allIssueComments = append(allIssueComments, issueComments...)

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

	return allIssueComments
}