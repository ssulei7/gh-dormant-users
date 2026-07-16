package issues

import (
	"fmt"
	"strings"

	"github.com/cli/go-gh/pkg/api"
	"github.com/ssulei7/gh-dormant-users/internal/githubapi"
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

func GetIssuesSinceDate(organization string, repo string, date string, client api.RESTClient) (Issues, error) {
	url := fmt.Sprintf("repos/%s/%s/issues?per_page=100&since=%s", organization, repo, date)
	issueList, err := githubapi.GetAll[Issue](client, url)
	if err != nil {
		if strings.Contains(err.Error(), "Git Repository is empty.") {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch issues for %s/%s: %w", organization, repo, err)
	}
	return Issues(issueList), nil
}

func GetIssueCommentsSinceDate(organization string, repo string, date string, client api.RESTClient) (IssueComments, error) {
	url := fmt.Sprintf("repos/%s/%s/issues/comments?per_page=100&since=%s", organization, repo, date)
	comments, err := githubapi.GetAll[IssueComment](client, url)
	if err != nil {
		if strings.Contains(err.Error(), "Git Repository is empty.") {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch issue comments for %s/%s: %w", organization, repo, err)
	}
	return IssueComments(comments), nil
}
