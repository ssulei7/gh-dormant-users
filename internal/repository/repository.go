package repository

import (
	"fmt"
	"time"

	"github.com/cli/go-gh/pkg/api"
	"github.com/ssulei7/gh-dormant-users/internal/githubapi"
	"github.com/ssulei7/gh-dormant-users/internal/ui"
)

type Repository struct {
	Name     string     `json:"name"`
	Size     int        `json:"size"`
	PushedAt *time.Time `json:"pushed_at"`
}

type Repositories []Repository

func GetOrgRepositories(organization string, client api.RESTClient) (Repositories, error) {
	spinner := ui.NewSimpleSpinner("Fetching repositories...")
	spinner.Start()

	url := fmt.Sprintf("orgs/%s/repos?per_page=100", organization)
	repositories, err := githubapi.GetAll[Repository](client, url)
	if err != nil {
		spinner.StopFail("Failed to fetch repositories")
		return nil, fmt.Errorf("fetch organization repositories: %w", err)
	}

	spinner.Stop("Fetched repositories successfully")
	ui.Info("Fetched %d repositories", len(repositories))
	return Repositories(repositories), nil
}
