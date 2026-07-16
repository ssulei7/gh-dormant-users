package pullrequests

import (
	"fmt"
	"strings"

	"github.com/cli/go-gh/pkg/api"
	"github.com/ssulei7/gh-dormant-users/internal/githubapi"
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

func GetPullRequestCommentsSinceDate(organization string, repo string, date string, client api.RESTClient) (PullRequestComments, error) {
	url := fmt.Sprintf("repos/%s/%s/pulls/comments?per_page=100&since=%s", organization, repo, date)
	comments, err := githubapi.GetAll[PullRequestComment](client, url)
	if err != nil {
		if strings.Contains(err.Error(), "Git Repository is empty.") {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch pull request comments for %s/%s: %w", organization, repo, err)
	}
	return PullRequestComments(comments), nil
}
