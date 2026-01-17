package cmd

import (
	"os"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/ssulei7/gh-dormant-users/internal/analysis"
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
		pterm.Error.Println("CSV file path is required. Use --file or -f flag.")
		os.Exit(1)
	}

	// Check if file exists
	if _, err := os.Stat(csvFile); os.IsNotExist(err) {
		pterm.Error.Printf("CSV file not found: %s\n", csvFile)
		os.Exit(1)
	}

	templateName, _ := cmd.Flags().GetString("template")
	customPrompt, _ := cmd.Flags().GetString("prompt")
	promptOnly, _ := cmd.Flags().GetBool("prompt-only")

	// If prompt-only mode, just build and display the prompt
	if promptOnly {
		prompt, err := analyzer.BuildPrompt(csvFile, templateName, customPrompt)
		if err != nil {
			pterm.Error.Printf("Failed to build prompt: %v\n", err)
			os.Exit(1)
		}
		pterm.DefaultHeader.WithFullWidth().Println("Generated Analysis Prompt")
		pterm.Println()
		pterm.Println(prompt)
		return
	}

	// Check Copilot availability for actual analysis
	if !analyzer.IsCopilotAvailable() {
		pterm.Error.Println("GitHub Copilot CLI is not available.")
		pterm.Info.Println("Install it with: gh extension install github/gh-copilot")
		os.Exit(1)
	}

	// Perform the analysis using Copilot SDK
	pterm.Info.Println("Analyzing CSV with Copilot...")
	spinner, _ := pterm.DefaultSpinner.Start("Sending to Copilot for analysis...")

	response, err := analyzer.AnalyzeCSV(csvFile, templateName, customPrompt)
	if err != nil {
		spinner.Fail("Analysis failed")
		pterm.Error.Printf("Analysis failed: %v\n", err)
		os.Exit(1)
	}

	spinner.Success("Analysis complete")
	pterm.Println()
	pterm.DefaultHeader.WithFullWidth().Println("Analysis Results")
	pterm.Println()
	pterm.Println(response)
}
