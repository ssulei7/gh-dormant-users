package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ssulei7/gh-dormant-users/internal/analysis"
	"github.com/ssulei7/gh-dormant-users/internal/ui"
)

type csvAnalyzer interface {
	ListTemplates()
	CheckCopilotStatus()
	BuildPrompt(csvPath string, templateName string, customPrompt string) (string, error)
	IsCopilotAvailable() bool
	AnalyzeCSV(csvPath string, templateName string, customPrompt string) (string, error)
}

type analysisSpinner interface {
	Start()
	Stop(message string)
	StopFail(message string)
}

var (
	newCSVAnalyzer = func() csvAnalyzer { return analysis.NewAnalyzer() }
	statCSVFile    = os.Stat
	newSpinner     = func(message string) analysisSpinner { return ui.NewSimpleSpinner(message) }
)

var analyzeCmd = newAnalyzeCommand()

func newAnalyzeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze dormant user CSV files using AI",
		Long: `Analyze dormant user CSV files using predefined analysis templates with GitHub Copilot.

Available templates:
  - summary:         High-level summary of dormant user data
  - trends:          Identify patterns in user activity
  - risk:            Security and compliance risk assessment
  - recommendations: Actionable recommendations for managing users
  - custom:          Custom analysis with user-provided prompt`,
		RunE: runAnalyze,
	}
	cmd.Flags().StringP("file", "f", "", "Path to the CSV file to analyze")
	cmd.Flags().StringP("template", "t", "summary", "Analysis template to use")
	cmd.Flags().StringP("prompt", "p", "", "Custom prompt (only used with 'custom' template)")
	cmd.Flags().Bool("list-templates", false, "List available analysis templates")
	cmd.Flags().Bool("check-copilot", false, "Check if Copilot CLI is available")
	cmd.Flags().Bool("prompt-only", false, "Only generate the prompt without sending to Copilot")
	return cmd
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	analyzer := newCSVAnalyzer()

	// Check for --list-templates flag
	listTemplates, _ := cmd.Flags().GetBool("list-templates")
	if listTemplates {
		analyzer.ListTemplates()
		return nil
	}

	// Check for --check-copilot flag
	checkCopilot, _ := cmd.Flags().GetBool("check-copilot")
	if checkCopilot {
		analyzer.CheckCopilotStatus()
		return nil
	}

	// Validate required flags
	csvFile, _ := cmd.Flags().GetString("file")
	if csvFile == "" {
		return fmt.Errorf("CSV file path is required. Use --file or -f flag")
	}

	// Check if file exists
	if _, err := statCSVFile(csvFile); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("CSV file not found: %s", csvFile)
		}
		return fmt.Errorf("check CSV file %s: %w", csvFile, err)
	}

	templateName, _ := cmd.Flags().GetString("template")
	customPrompt, _ := cmd.Flags().GetString("prompt")
	promptOnly, _ := cmd.Flags().GetBool("prompt-only")

	// If prompt-only mode, just build and display the prompt
	if promptOnly {
		prompt, err := analyzer.BuildPrompt(csvFile, templateName, customPrompt)
		if err != nil {
			return fmt.Errorf("failed to build prompt: %w", err)
		}
		ui.Header("Generated Analysis Prompt")
		ui.Println()
		ui.Println(prompt)
		return nil
	}

	// Check Copilot availability for actual analysis
	if !analyzer.IsCopilotAvailable() {
		ui.Info("Install it with: gh extension install github/gh-copilot")
		return fmt.Errorf("GitHub Copilot CLI is not available")
	}

	// Perform the analysis using Copilot SDK
	ui.Info("Analyzing CSV with Copilot...")
	spinner := newSpinner("Sending to Copilot for analysis...")
	spinner.Start()

	response, err := analyzer.AnalyzeCSV(csvFile, templateName, customPrompt)
	if err != nil {
		spinner.StopFail("Analysis failed")
		return fmt.Errorf("analysis failed: %w", err)
	}

	spinner.Stop("Analysis complete")
	ui.Println()
	ui.Header("Analysis Results")
	ui.Println()
	ui.Println(response)
	return nil
}
