package analysis

import (
	"testing"
)

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"empty string", "", "test", false},
		{"empty substr", "test", "", true},
		{"exact match", "copilot", "copilot", true},
		{"contains substr", "github/gh-copilot", "copilot", true},
		{"no match", "github/gh-actions", "copilot", false},
		{"partial match at start", "copilot-cli", "copilot", true},
		{"partial match at end", "gh-copilot", "copilot", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("contains(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}

func TestContainsCopilotExtension(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected bool
	}{
		{"empty output", "", false},
		{"contains gh-copilot", "github/gh-copilot\tCopilot in the CLI\tv1.0.0", true},
		{"contains copilot only", "some copilot extension", true},
		{"no copilot", "github/gh-actions\tActions\tv1.0.0", false},
		{"multiple extensions with copilot", "github/gh-actions\ncopilot\ngithub/gh-pr", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsCopilotExtension(tt.output)
			if result != tt.expected {
				t.Errorf("containsCopilotExtension(%q) = %v, want %v", tt.output, result, tt.expected)
			}
		})
	}
}

func TestSearchSubstring(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		substr   string
		expected bool
	}{
		{"found at start", "hello world", "hello", true},
		{"found at end", "hello world", "world", true},
		{"found in middle", "hello world", "lo wo", true},
		{"not found", "hello world", "xyz", false},
		{"empty substr", "hello", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := searchSubstring(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("searchSubstring(%q, %q) = %v, want %v", tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}
