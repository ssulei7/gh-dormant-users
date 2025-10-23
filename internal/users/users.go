package users

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

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
	pterm.Info.Printf("Starting to fetch users for organization: %s\n", organization)
	var allUsers Users

	// Start the spinner
	spinner, _ := pterm.DefaultSpinner.Start("Fetching users...")

	if email {
		pterm.Info.Println("Getting user emails, if present")
	}

	url := fmt.Sprintf("orgs/%s/members?per_page=100", organization)
	for {
		response, err := client.Request("GET", url, nil)
		if err != nil {
			spinner.Fail("Failed to fetch users")
			pterm.Error.Printf("Failed to fetch users: %v\n", err)
			os.Exit(1)
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
	client, err := gh.RESTClient(nil)
	if err != nil {
		pterm.Error.Printf("Failed to create REST client: %v\n", err)
		os.Exit(1)
	}

	var wg sync.WaitGroup
	userChan := make(chan int, len(users))
	numWorkers := 10

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range userChan {
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
					pterm.Error.Printf("Failed to decode user details for %s: %v\n", users[index].Login, err)
					continue
				}

				users[index].Email = userDetails.Email
			}
		}()
	}

	for index := range users {
		userChan <- index
	}
	close(userChan)
	wg.Wait()
}
