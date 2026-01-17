package analysis

import (
	"fmt"
	"strings"
	"time"

	copilot "github.com/github/copilot-sdk/go"
	"github.com/pterm/pterm"
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

	// Collect response with timeout protection
	var response strings.Builder
	done := make(chan bool)
	var responseErr error

	session.On(func(event copilot.SessionEvent) {
		if event.Type == "assistant.message" {
			if event.Data.Content != nil {
				response.WriteString(*event.Data.Content)
			}
		}
		if event.Type == "session.idle" {
			select {
			case <-done:
				// Already closed
			default:
				close(done)
			}
		}
		if event.Type == "error" {
			if event.Data.Content != nil {
				responseErr = fmt.Errorf("copilot error: %s", *event.Data.Content)
			}
			select {
			case <-done:
				// Already closed
			default:
				close(done)
			}
		}
	})

	_, err = session.Send(copilot.MessageOptions{
		Prompt: prompt,
	})
	if err != nil {
		return "", fmt.Errorf("failed to send message to Copilot: %w", err)
	}

	// Wait for response with timeout
	select {
	case <-done:
		// Response received
	case <-time.After(DefaultTimeout):
		return "", fmt.Errorf("timeout waiting for Copilot response (exceeded %v)", DefaultTimeout)
	}

	if responseErr != nil {
		return "", responseErr
	}

	return response.String(), nil
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
	pterm.DefaultHeader.WithFullWidth().Println("Available Analysis Templates")
	pterm.Println()

	tableData := pterm.TableData{
		{"Template", "Description"},
	}

	for name, template := range PredefinedTemplates {
		tableData = append(tableData, []string{name, template.Description})
	}

	pterm.DefaultTable.WithHasHeader().WithData(tableData).Render()
}

// CheckCopilotStatus displays the Copilot CLI availability status
func (a *Analyzer) CheckCopilotStatus() {
	if a.copilotAvailable {
		pterm.Success.Println("GitHub Copilot CLI is available")
	} else {
		pterm.Warning.Println("GitHub Copilot CLI is not available")
		pterm.Info.Println("Install it with: gh extension install github/gh-copilot")
	}
}
