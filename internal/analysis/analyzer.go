package analysis

import (
	"fmt"
	"time"

	copilot "github.com/github/copilot-sdk/go"
	"github.com/ssulei7/gh-dormant-users/internal/ui"
)

// DefaultTimeout is the maximum time to wait for a Copilot response
const DefaultTimeout = 2 * time.Minute

// Analyzer handles CSV analysis using Copilot SDK
type Analyzer struct {
	copilotAvailable bool
	client           *copilot.Client
}

// NewAnalyzer creates a new Analyzer instance
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		copilotAvailable: IsCopilotCLIAvailable(),
	}
}

// IsCopilotAvailable returns whether Copilot CLI is available
func (a *Analyzer) IsCopilotAvailable() bool {
	return a.copilotAvailable
}

// AnalyzeCSV analyzes a CSV file using the specified template and returns the AI response
func (a *Analyzer) AnalyzeCSV(csvPath string, templateName string, customPrompt string) (string, error) {
	if !a.copilotAvailable {
		return "", fmt.Errorf("GitHub Copilot CLI is not available. Install it with: gh extension install github/gh-copilot")
	}

	template := GetTemplate(templateName)
	if template == nil {
		return "", fmt.Errorf("unknown template: %s. Available templates: %v", templateName, GetTemplateNames())
	}

	// Parse CSV and aggregate statistics (much smaller than raw data)
	stats, err := ParseCSVStats(csvPath)
	if err != nil {
		return "", fmt.Errorf("failed to parse CSV file: %w", err)
	}

	// Format stats for prompt
	statsText := stats.FormatForPrompt()

	// Build the prompt with aggregated stats instead of raw CSV
	var prompt string
	if templateName == "custom" {
		if customPrompt == "" {
			return "", fmt.Errorf("custom template requires a custom prompt")
		}
		prompt = fmt.Sprintf(template.Prompt, customPrompt, statsText)
	} else {
		prompt = fmt.Sprintf(template.Prompt, statsText)
	}

	// Use Copilot SDK to analyze
	return a.sendToCopilot(prompt)
}

// sendToCopilot sends the prompt to Copilot and returns the response with timeout
func (a *Analyzer) sendToCopilot(prompt string) (string, error) {
	client := copilot.NewClient(&copilot.ClientOptions{
		LogLevel: "error",
	})

	if err := client.Start(); err != nil {
		return "", fmt.Errorf("failed to start Copilot client: %w", err)
	}
	defer client.Stop()

	session, err := client.CreateSession(&copilot.SessionConfig{
		Model: "gpt-4o",
	})
	if err != nil {
		return "", fmt.Errorf("failed to create Copilot session: %w", err)
	}
	defer session.Destroy()

	response, err := session.SendAndWait(copilot.MessageOptions{
		Prompt: prompt,
	}, DefaultTimeout)
	if err != nil {
		return "", fmt.Errorf("copilot error: %w", err)
	}

	if response != nil && response.Data.Content != nil {
		return *response.Data.Content, nil
	}
	return "", nil
}

// BuildPrompt builds a prompt from a template and CSV data without sending to Copilot
func (a *Analyzer) BuildPrompt(csvPath string, templateName string, customPrompt string) (string, error) {
	template := GetTemplate(templateName)
	if template == nil {
		return "", fmt.Errorf("unknown template: %s. Available templates: %v", templateName, GetTemplateNames())
	}

	// Parse CSV and aggregate statistics
	stats, err := ParseCSVStats(csvPath)
	if err != nil {
		return "", fmt.Errorf("failed to parse CSV file: %w", err)
	}

	statsText := stats.FormatForPrompt()

	var prompt string
	if templateName == "custom" {
		if customPrompt == "" {
			return "", fmt.Errorf("custom template requires a custom prompt")
		}
		prompt = fmt.Sprintf(template.Prompt, customPrompt, statsText)
	} else {
		prompt = fmt.Sprintf(template.Prompt, statsText)
	}

	return prompt, nil
}

// ListTemplates displays available templates
func (a *Analyzer) ListTemplates() {
	ui.Header("Available Analysis Templates")
	ui.Println()

	for name, template := range PredefinedTemplates {
		ui.Printf("  %s - %s\n", name, template.Description)
	}
	ui.Println()
}

// CheckCopilotStatus displays the Copilot CLI availability status
func (a *Analyzer) CheckCopilotStatus() {
	if a.copilotAvailable {
		ui.Success("GitHub Copilot CLI is available")
	} else {
		ui.Warning("GitHub Copilot CLI is not available")
		ui.Info("Install it with: gh extension install github/gh-copilot")
	}
}
