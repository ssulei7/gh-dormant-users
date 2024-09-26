package users

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/cli/go-gh"
)

type User struct {
	Login string `json:"login"`
	ID    int    `json:"id"`
	Email string `json:"email"`
}

type Users []User

func GetOrganizationUsers(organization string) Users {
	log.Printf("Starting to fetch users for organization: %s", organization)
	client, err := gh.RESTClient(nil)
	if err != nil {
		log.Fatalf("Failed to create REST client: %v", err)
	}

	var allUsers Users
	url := "orgs/" + organization + "/members"

	for {
		log.Printf("Fetching users from URL: %s", url)
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

		log.Printf("Fetched %d users", len(users))
		// get user emails
		getUserEmails(users)
		allUsers = append(allUsers, users...)

		// Check for the 'Link' header to see if there are more pages
		linkHeader := response.Header.Get("Link")
		if linkHeader == "" {
			log.Printf("No more pages to fetch")
			break
		}

		nextURL := getNextPageURL(linkHeader)
		if nextURL == "" {
			log.Printf("No next page URL found")
			break
		}

		url = nextURL
	}

	log.Printf("Found %d users in organization %s", len(allUsers), organization)

	return allUsers
}

func getUserEmails(users Users) {
	log.Println("Getting user emails, if present")
	client, err := gh.RESTClient(nil)
	if err != nil {
		log.Fatalf("Failed to create REST client: %v", err)
	}

	for i := range users {
		url := fmt.Sprintf("users/%s", users[i].Login)
		response, err := client.Request("GET", url, nil)
		if err != nil {
			log.Fatalf("Failed to fetch user details: %v", err)
		}

		var userDetails User
		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&userDetails)
		if err != nil {
			log.Fatalf("Failed to decode user details: %v", err)
		}

		users[i].Email = userDetails.Email

	}

}

func getNextPageURL(linkHeader string) string {
	links := strings.Split(linkHeader, ",")
	for _, link := range links {
		parts := strings.Split(strings.TrimSpace(link), ";")
		if len(parts) < 2 {
			continue
		}
		urlPart := strings.Trim(parts[0], "<>")
		relPart := strings.TrimSpace(parts[1])
		if relPart == `rel="next"` {
			return urlPart
		}
	}
	return ""
}
