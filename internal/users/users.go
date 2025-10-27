package users

import (
	"context"
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

	// Start the spinner
	spinner, _ := pterm.DefaultSpinner.Start("Fetching users...")

	if email {
		pterm.Info.Println("Getting user emails, if present")
	}

	// Fetch first page to get total count
	url := fmt.Sprintf("orgs/%s/members?per_page=100", organization)
	if err := limiter.WaitForTokenAndAcquire(context.Background()); err != nil {
		spinner.Fail("Failed to acquire rate limit token")
		pterm.Error.Printf("Failed to acquire rate limit token: %v\n", err)
		os.Exit(1)
	}
	
	response, err := client.Request("GET", url, nil)
	if err != nil {
		limiter.ReleaseConcurrentLimiter()
		spinner.Fail("Failed to fetch users")
		pterm.Error.Printf("Failed to fetch users: %v\n", err)
		os.Exit(1)
	}

	var users Users
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&users)
	linkHeader := response.Header.Get("Link")
	response.Body.Close()

	limiter.ReleaseAndHandleRateLimit(response)

	if err != nil {
		spinner.Fail("Failed to decode users")
		pterm.PrintOnErrorf("Failed to decode users: %v\n", err)
	}

	allUsers := make(Users, len(users))
	copy(allUsers, users)

	// Get all page URLs from Link header
	var pageURLs []string
	for linkHeader != "" {
		nextURL := header.GetNextPageURL(linkHeader)
		if nextURL == "" {
			break
		}
		pageURLs = append(pageURLs, nextURL)

		// Fetch next page to get updated Link header
		if err := limiter.WaitForTokenAndAcquire(context.Background()); err != nil {
			continue
		}
		
		response, err := client.Request("GET", nextURL, nil)
		if err != nil {
			limiter.ReleaseConcurrentLimiter()
			continue
		}
		linkHeader = response.Header.Get("Link")
		response.Body.Close()
		limiter.ReleaseAndHandleRateLimit(response)
	}

	// Fetch remaining pages concurrently
	if len(pageURLs) > 0 {
		pageChan := make(chan string, len(pageURLs))
		resultChan := make(chan Users, len(pageURLs))
		var wg sync.WaitGroup
		numWorkers := 5

		for i := 0; i < numWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for pageURL := range pageChan {
					if err := limiter.WaitForTokenAndAcquire(context.Background()); err != nil {
						continue
					}
					
					response, err := client.Request("GET", pageURL, nil)
					if err != nil {
						limiter.ReleaseConcurrentLimiter()
						continue
					}

					var pageUsers Users
					decoder := json.NewDecoder(response.Body)
					err = decoder.Decode(&pageUsers)
					response.Body.Close()

					limiter.ReleaseAndHandleRateLimit(response)

					if err != nil {
						continue
					}
					resultChan <- pageUsers
				}
			}()
		}

		for _, pageURL := range pageURLs {
			pageChan <- pageURL
		}
		close(pageChan)
		wg.Wait()
		close(resultChan)

		for pageUsers := range resultChan {
			allUsers = append(allUsers, pageUsers...)
		}
	}

	// get user emails
	if email {
		getUserEmails(allUsers)
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
	numWorkers := 5

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range userChan {
				if err := limiter.WaitForTokenAndAcquire(context.Background()); err != nil {
					pterm.Info.Printf("Failed to acquire rate limit token: %v\n", err)
					continue
				}
				
				url := fmt.Sprintf("users/%s", users[index].Login)
				response, err := client.Request("GET", url, nil)
				if err != nil {
					limiter.ReleaseConcurrentLimiter()
					pterm.Info.Printf("Failed to fetch user details: %v\n", err)
					continue
				}

				var userDetails User
				decoder := json.NewDecoder(response.Body)
				err = decoder.Decode(&userDetails)
				response.Body.Close()

				limiter.ReleaseAndHandleRateLimit(response)

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
