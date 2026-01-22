package cmd

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/ssulei7/gh-dormant-users/internal/analysis"
	"github.com/ssulei7/gh-dormant-users/internal/ui"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Analyze dormant user CSV files using AI",
	Long: `Analyze dormant user CSV files using predefined analysis templates with GitHub Copilot.

Available templates:
  - summary:         High-level summary of dormant user data
  - trends:          Identify patterns in user activity
  - risk:            Security and compliance risk assessment
  - recommendations: Actionable recommendations for managing users
  - custom:          Custom analysis with user-provided prompt`,
	Run: runAnalyze,
}

func init() {
	analyzeCmd.Flags().StringP("file", "f", "", "Path to the CSV file to analyze")
	analyzeCmd.Flags().StringP("template", "t", "summary", "Analysis template to use")
	analyzeCmd.Flags().StringP("prompt", "p", "", "Custom prompt (only used with 'custom' template)")
	analyzeCmd.Flags().Bool("list-templates", false, "List available analysis templates")
	analyzeCmd.Flags().Bool("check-copilot", false, "Check if Copilot CLI is available")
	analyzeCmd.Flags().Bool("prompt-only", false, "Only generate the prompt without sending to Copilot")
}

func runAnalyze(cmd *cobra.Command, args []string) {
	analyzer := analysis.NewAnalyzer()

	// Check for --list-templates flag
	listTemplates, _ := cmd.Flags().GetBool("list-templates")
	if listTemplates {
		analyzer.ListTemplates()
		return
	}

	// Check for --check-copilot flag
	checkCopilot, _ := cmd.Flags().GetBool("check-copilot")
	if checkCopilot {
		analyzer.CheckCopilotStatus()
		return
	}

	// Validate required flags
	csvFile, _ := cmd.Flags().GetString("file")
	if csvFile == "" {
		ui.Error("CSV file path is required. Use --file or -f flag.")
		os.Exit(1)
	}

	// Check if file exists
	if _, err := os.Stat(csvFile); os.IsNotExist(err) {
		ui.Error("CSV file not found: %s", csvFile)
		os.Exit(1)
	}

	templateName, _ := cmd.Flags().GetString("template")
	customPrompt, _ := cmd.Flags().GetString("prompt")
	promptOnly, _ := cmd.Flags().GetBool("prompt-only")

	// If prompt-only mode, just build and display the prompt
	if promptOnly {
		prompt, err := analyzer.BuildPrompt(csvFile, templateName, customPrompt)
		if err != nil {
			ui.Error("Failed to build prompt: %v", err)
			os.Exit(1)
		}
		ui.Header("Generated Analysis Prompt")
		ui.Println()
		ui.Println(prompt)
		return
	}

	// Check Copilot availability for actual analysis
	if !analyzer.IsCopilotAvailable() {
		ui.Error("GitHub Copilot CLI is not available.")
		ui.Info("Install it with: gh extension install github/gh-copilot")
		os.Exit(1)
	}

	// Perform the analysis using Copilot SDK
	ui.Info("Analyzing CSV with Copilot...")
	spinner := ui.NewSimpleSpinner("Sending to Copilot for analysis...")
	spinner.Start()

	response, err := analyzer.AnalyzeCSV(csvFile, templateName, customPrompt)
	if err != nil {
		spinner.StopFail("Analysis failed")
		ui.Error("Analysis failed: %v", err)
		os.Exit(1)
	}

	spinner.Stop("Analysis complete")
	ui.Println()
	ui.Header("Analysis Results")
	ui.Println()
	ui.Println(response)
}
