package users

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"github.com/ssulei7/gh-dormant-users/config"
	"github.com/ssulei7/gh-dormant-users/internal/header"
	"github.com/ssulei7/gh-dormant-users/internal/limiter"
)

type User struct {
	Login  string `json:"login"`
	ID     int    `json:"id"`
	Email  string `json:"email"`
	Active bool
}

type Users []User

func GetOrganizationUsers(organization string, email bool, client api.RESTClient) Users {
	log.Printf("Starting to fetch users for organization: %s", organization)
	var allUsers Users
	url := fmt.Sprintf("orgs/%s/members?per_page=100", organization)

	for {
		if config.Verbose {
			log.Printf("Fetching users from URL: %s", url)
		}
		response, err := client.Request("GET", url, nil)
		if err != nil {
			log.Fatalf("Failed to fetch users: %v", err)
		}

		var users Users
		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&users)
		if err != nil {
			log.Fatalf("Failed to decode users: %v", err)
		}

		if config.Verbose {
			log.Printf("Fetched %d users", len(users))
		}

		// get user emails sequentially
		if email {
			getUserEmails(users)
		}
		allUsers = append(allUsers, users...)

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
				log.Printf("No next page URL found")
			}

			break
		}

		url = nextURL
	}

	log.Printf("Found %d users in organization %s", len(allUsers), organization)

	return allUsers
}

func (u *User) MakeActive() {
	u.Active = true
}

func (u *User) MakeInactive() {
	u.Active = false
}

func getUserEmails(users Users) {
	log.Println("Getting user emails, if present")
	client, err := gh.RESTClient(nil)
	if err != nil {
		log.Fatalf("Failed to create REST client: %v", err)
	}

	for index := range users {
		limiter.AcquireConcurrentLimiter()
		defer limiter.ReleaseConcurrentLimiter()
		url := fmt.Sprintf("users/%s", users[index].Login)
		response, err := client.Request("GET", url, nil)
		if err != nil {
			log.Printf("Failed to fetch user details: %v", err)
			continue
		}

		var userDetails User
		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&userDetails)
		if err != nil {
			log.Printf("Failed to decode user details: %v", err)
			continue
		}

		users[index].Email = userDetails.Email
	}
}
