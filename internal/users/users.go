package users

import (
	"encoding/json"
	"fmt"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"github.com/pterm/pterm"
	"github.com/ssulei7/gh-dormant-users/internal/header"
	"github.com/ssulei7/gh-dormant-users/internal/limiter"
)

type User struct {
	Login         string `json:"login"`
	ID            int    `json:"id"`
	Email         string `json:"email"`
	Active        bool
	ActivityTypes map[string]bool
}

type Users []User

func GetOrganizationUsers(organization string, email bool, client api.RESTClient) Users {
	pterm.Info.Printf("Starting to fetch members for organization: %s\n", organization)
    return GetUsers(organization, "orgs/%s/members?per_page=100", email, client)
}

func GetOrganizationOutsideCollaborators(organization string, email bool, client api.RESTClient) Users {
	pterm.Info.Printf("Starting to fetch outside collaborators for organization: %s\n", organization)
    return GetUsers(organization, "orgs/%s/outside_collaborators?per_page=100", email, client)
}

func GetUsers(organization string, endpoint string, email bool, client api.RESTClient) Users {
	var allUsers Users

	// Start the spinner
	spinner, _ := pterm.DefaultSpinner.Start("Fetching users...")

	url := fmt.Sprintf(endpoint, organization)
	for {
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

		// get user emails sequentially
		if email {
			getUserEmails(users)
		}
		allUsers = append(allUsers, users...)

		// Check for the 'Link' header to see if there are more pages
		linkHeader := response.Header.Get("Link")
		if linkHeader == "" {
			break
		}

		nextURL := header.GetNextPageURL(linkHeader)
		if nextURL == "" {
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

func (u *User) AddActivityType(t string) {
	if u.ActivityTypes == nil {
		u.ActivityTypes = make(map[string]bool)
	}
	u.ActivityTypes[t] = true
}

func (u *User) GetActivityTypes() []string {
	if u.ActivityTypes == nil {
		return nil
	}
	var atSlice []string
	for t := range u.ActivityTypes {
		atSlice = append(atSlice, t)
	}
	return atSlice
}

func getUserEmails(users Users) {
	pterm.Info.Println("Getting user emails, if present")
	client, err := gh.RESTClient(nil)
	if err != nil {
		pterm.Fatal.PrintOnErrorf("Failed to create REST client: %v", err)
	}

	for index := range users {
		limiter.AcquireConcurrentLimiter()
		url := fmt.Sprintf("users/%s", users[index].Login)
		response, err := client.Request("GET", url, nil)
		limiter.ReleaseConcurrentLimiter()
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
