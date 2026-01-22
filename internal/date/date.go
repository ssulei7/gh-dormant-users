package date

import (
	"os"
	"time"

	"github.com/ssulei7/gh-dormant-users/internal/ui"
)

func ValidateDate(date string) bool {
	// Validate date is no longer than 3 months, and turn into an ISO string
	parsedDate, err := time.Parse("Jan 2 2006", date)
	if err != nil {
		ui.Error("Failed to parse date: %v", err)
		os.Exit(1)
	}
	// Check if date is within the last 3 months
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	if parsedDate.Before(threeMonthsAgo) {
		ui.Error("Date must be within the last 3 months")
		os.Exit(1)
	}
	return true
}

func GetISODate(date string) string {
	parsedDate, err := time.Parse("Jan 2 2006", date)
	if err != nil {
		ui.Error("Failed to parse date: %v", err)
		os.Exit(1)
	}
	// Convert to iso 8601 format
	return parsedDate.Format("2006-01-02T15:04:05Z")
}
