package commits

import (
	"fmt"
	"strings"

	"github.com/cli/go-gh/pkg/api"
	"github.com/ssulei7/gh-dormant-users/internal/githubapi"
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

func GetCommitsSinceDate(organization string, repository string, date string, client api.RESTClient) (Commits, error) {
	url := fmt.Sprintf("repos/%s/%s/commits?per_page=100&since=%s", organization, repository, date)
	commitList, err := githubapi.GetAll[Commit](client, url)
	if err != nil {
		if strings.Contains(err.Error(), "Git Repository is empty.") {
			return nil, nil
		}
		return nil, fmt.Errorf("fetch commits for %s/%s: %w", organization, repository, err)
	}
	return Commits(commitList), nil
}
