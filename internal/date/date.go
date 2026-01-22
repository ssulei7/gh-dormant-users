package date

import (
	"fmt"
	"time"
)

func ValidateDate(date string) error {
	parsedDate, err := time.Parse("Jan 2 2006", date)
	if err != nil {
		return fmt.Errorf("failed to parse date: %w", err)
	}
	// Check if date is within the last 3 months
	threeMonthsAgo := time.Now().AddDate(0, -3, 0)
	if parsedDate.Before(threeMonthsAgo) {
		return fmt.Errorf("date must be within the last 3 months")
	}
	return nil
}

func GetISODate(date string) (string, error) {
	parsedDate, err := time.Parse("Jan 2 2006", date)
	if err != nil {
		return "", fmt.Errorf("failed to parse date: %w", err)
	}
	return parsedDate.Format("2006-01-02T15:04:05Z"), nil
}
