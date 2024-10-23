package repository

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/cli/go-gh/pkg/api"
	"github.com/ssulei7/gh-dormant-users/config"
	"github.com/ssulei7/gh-dormant-users/internal/header"
)

type Repository struct {
	Name string `json:"name"`
}

type Repositories []Repository

func GetOrgRepositories(organization string, client api.RESTClient) Repositories {
	var allRepositories Repositories
	log.Default().Println("Fetching repositories for organization: " + organization)
	url := fmt.Sprintf("orgs/%s/repos?per_page=100", organization)
	for {
		if config.Verbose {
			log.Printf("Fetching repositories from URL: %s", url)
		}
		var response *http.Response
		var err error
		for retries := 0; retries < 5; retries++ {
			response, err = client.Request("GET", url, nil)
			if err != nil {
				log.Printf("Failed to fetch repositories: %v. Retrying in %d seconds...", err, (1 << retries))
				time.Sleep(time.Duration(1<<retries) * time.Second)
				continue
			}
			break
		}
		if err != nil {
			log.Fatalf("Failed to fetch repositories after retries: %v", err)
		}

		var repositories Repositories
		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&repositories)
		if err != nil {
			log.Fatalf("Failed to decode repositories: %v", err)
		}

		allRepositories = append(allRepositories, repositories...)

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
			if config.Verbose {
				log.Printf("No more pages to fetch")
			}
			break
		}

		url = nextURL
	}

	log.Printf("Fetched %d repositories", len(allRepositories))
	return allRepositories
}
