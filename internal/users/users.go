package users

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/api"
	"github.com/ssulei7/gh-dormant-users/internal/header"
	"github.com/ssulei7/gh-dormant-users/internal/limiter"
	"github.com/ssulei7/gh-dormant-users/internal/ui"
)

type User struct {
	Login         string `json:"login"`
	ID            int    `json:"id"`
	Email         string `json:"email"`
	Active        bool
	ActivityTypes map[string]bool
	mu            sync.Mutex // protects Active and ActivityTypes
}

type Users []User

func GetOrganizationUsers(organization string, email bool, client api.RESTClient) (Users, error) {
	ui.Info("Starting to fetch users for organization: %s", organization)

	// Start the spinner
	spinner := ui.NewSimpleSpinner("Fetching users...")
	spinner.Start()

	if email {
		ui.Info("Getting user emails, if present")
	}

	// Fetch first page to get total count
	url := fmt.Sprintf("orgs/%s/members?per_page=100", organization)
	limiter.AcquireConcurrentLimiter()
	response, err := client.Request("GET", url, nil)
	if err != nil {
		limiter.ReleaseConcurrentLimiter()
		spinner.StopFail("Failed to fetch users")
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	var users Users
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&users)
	linkHeader := response.Header.Get("Link")
	response.Body.Close()
	limiter.ReleaseConcurrentLimiter()
	limiter.CheckAndHandleRateLimit(response)

	if err != nil {
		spinner.StopFail("Failed to decode users")
		return nil, fmt.Errorf("failed to decode users: %w", err)
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
		limiter.AcquireConcurrentLimiter()
		response, err := client.Request("GET", nextURL, nil)
		if err != nil {
			limiter.ReleaseConcurrentLimiter()
			ui.Warning("Failed to fetch page %s: %v", nextURL, err)
			continue
		}
		linkHeader = response.Header.Get("Link")
		response.Body.Close()
		limiter.ReleaseConcurrentLimiter()
		limiter.CheckAndHandleRateLimit(response)
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
					limiter.AcquireConcurrentLimiter()
					response, err := client.Request("GET", pageURL, nil)
					if err != nil {
						limiter.ReleaseConcurrentLimiter()
						ui.Warning("Failed to fetch page %s: %v", pageURL, err)
						continue
					}

					var pageUsers Users
					decoder := json.NewDecoder(response.Body)
					err = decoder.Decode(&pageUsers)
					response.Body.Close()
					limiter.ReleaseConcurrentLimiter()
					limiter.CheckAndHandleRateLimit(response)

					if err != nil {
						ui.Warning("Failed to decode page %s: %v", pageURL, err)
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
		if err := getUserEmails(allUsers); err != nil {
			spinner.StopFail("Failed to get user emails")
			return nil, err
		}
	}

	spinner.Stop("Fetched users successfully")

	return allUsers, nil
}

func (u *User) MakeActive() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Active = true
}

func (u *User) MakeInactive() {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.Active = false
}

func (u *User) IsActive() bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	return u.Active
}

func (u *User) AddActivityType(t string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.ActivityTypes == nil {
		u.ActivityTypes = make(map[string]bool)
	}
	u.ActivityTypes[t] = true
}

// MarkActiveWithType atomically adds activity type and marks user active
func (u *User) MarkActiveWithType(t string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.ActivityTypes == nil {
		u.ActivityTypes = make(map[string]bool)
	}
	u.ActivityTypes[t] = true
	u.Active = true
}

func (u *User) GetActivityTypes() []string {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.ActivityTypes == nil {
		return nil
	}
	var atSlice []string
	for t := range u.ActivityTypes {
		atSlice = append(atSlice, t)
	}
	return atSlice
}

func getUserEmails(users Users) error {
	client, err := gh.RESTClient(nil)
	if err != nil {
		return fmt.Errorf("failed to create REST client: %w", err)
	}

	var wg sync.WaitGroup
	userChan := make(chan int, len(users))
	numWorkers := 5

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range userChan {
				limiter.AcquireConcurrentLimiter()
				url := fmt.Sprintf("users/%s", users[index].Login)
				response, err := client.Request("GET", url, nil)
				if err != nil {
					limiter.ReleaseConcurrentLimiter()
					ui.Warning("Failed to fetch user details for %s: %v", users[index].Login, err)
					continue
				}

				var userDetails User
				decoder := json.NewDecoder(response.Body)
				err = decoder.Decode(&userDetails)
				response.Body.Close()
				limiter.ReleaseConcurrentLimiter()
				limiter.CheckAndHandleRateLimit(response)

				if err != nil {
					ui.Warning("Failed to decode user details for %s: %v", users[index].Login, err)
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
	return nil
}
