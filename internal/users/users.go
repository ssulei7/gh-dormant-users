package users

import (
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"github.com/pterm/pterm"
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
	pterm.Info.Printf("Starting to fetch users for organization: %s\n", organization)
	var allUsers Users

	// Start the spinner
	spinner, _ := pterm.DefaultSpinner.Start("Fetching users...")

	url := fmt.Sprintf("orgs/%s/members?per_page=100", organization)
	for {
		if config.Verbose {
			pterm.Debug.Printf("Fetching users from URL: %s", url)
		}
		response, err := client.Request("GET", url, nil)
		if err != nil {
			spinner.Fail("Failed to fetch users")
			pterm.Fatal.PrintOnErrorf("Failed to fetch users: %v", err)
		}

		var users Users
		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&users)
		if err != nil {
			spinner.Fail("Failed to decode users")
			pterm.PrintOnErrorf("Failed to decode users: %v\n", err)
		}

		if config.Verbose {
			pterm.Info.Printf("Fetched %d users\n", len(users))
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
				pterm.Info.Printf("No more pages to fetch")
			}
			break
		}

		nextURL := header.GetNextPageURL(linkHeader)
		if nextURL == "" {
			if config.Verbose {
				pterm.Info.Printf("No next page URL found\n")
			}

			break
		}

		url = nextURL
	}

	spinner.Success("Fetched users successfully")

	return allUsers
}

func (u *User) MakeActive() {
	u.Active = true
}

func (u *User) MakeInactive() {
	u.Active = false
}

func getUserEmails(users Users) {
	pterm.Info.Println("Getting user emails, if present")
	client, err := gh.RESTClient(nil)
	if err != nil {
		pterm.Fatal.PrintOnErrorf("Failed to create REST client: %v", err)
	}

	for index := range users {
		limiter.AcquireConcurrentLimiter()
		defer limiter.ReleaseConcurrentLimiter()
		url := fmt.Sprintf("users/%s", users[index].Login)
		response, err := client.Request("GET", url, nil)
		if err != nil {
			pterm.Info.Printf("Failed to fetch user details: %v\n", err)
			continue
		}

		var userDetails User
		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&userDetails)
		if err != nil {
			pterm.Fatal.PrintOnErrorf("Failed to decode user details: %v\n", err)
			continue
		}

		users[index].Email = userDetails.Email
	}
}
