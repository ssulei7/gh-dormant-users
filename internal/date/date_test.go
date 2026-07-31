package date

import (
	"strings"
	"testing"
	"time"
)

func TestValidateDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		date    string
		wantErr string
	}{
		{name: "current date", date: time.Now().Format("Jan 2 2006")},
		{name: "future date", date: time.Now().AddDate(0, 1, 0).Format("Jan 2 2006")},
		{
			name:    "date older than three months",
			date:    time.Now().AddDate(0, -4, 0).Format("Jan 2 2006"),
			wantErr: "date must be within the last 3 months",
		},
		{name: "invalid date", date: "2026-07-31", wantErr: "failed to parse date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := ValidateDate(tt.date)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("ValidateDate(%q) returned error: %v", tt.date, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ValidateDate(%q) error = %v, want error containing %q", tt.date, err, tt.wantErr)
			}
		})
	}
}

func TestGetISODate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		date    string
		want    string
		wantErr string
	}{
		{name: "valid date", date: "Jul 31 2026", want: "2026-07-31T00:00:00Z"},
		{name: "invalid date", date: "July 31, 2026", wantErr: "failed to parse date"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := GetISODate(tt.date)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("GetISODate(%q) error = %v, want error containing %q", tt.date, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetISODate(%q) returned error: %v", tt.date, err)
			}
			if got != tt.want {
				t.Fatalf("GetISODate(%q) = %q, want %q", tt.date, got, tt.want)
			}
		})
	}
}
