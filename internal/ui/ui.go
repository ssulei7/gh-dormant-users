// Package ui provides styled terminal output using the Charm stack.
// This replaces pterm with charmbracelet/lipgloss for styling,
// charmbracelet/bubbletea for interactive components, and
// charmbracelet/bubbles for reusable UI components.
package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Colors used throughout the UI
var (
	cyan   = lipgloss.Color("6")
	green  = lipgloss.Color("2")
	yellow = lipgloss.Color("3")
	red    = lipgloss.Color("1")
	gray   = lipgloss.Color("8")
	white  = lipgloss.Color("15")
)

// Styles for different message types
var (
	infoStyle = lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(green).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(yellow).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(red).
			Bold(true)

	labelStyle = lipgloss.NewStyle().
			Foreground(gray)

	valueStyle = lipgloss.NewStyle().
			Foreground(white).
			Bold(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true).
			Underline(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cyan).
			Padding(1, 2)

	titleBoxStyle = lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true)
)

// Prefixes for message types
const (
	infoPrefix    = "ℹ "
	successPrefix = "✓ "
	warningPrefix = "⚠ "
	errorPrefix   = "✗ "
)

// Info prints an info message with cyan styling
func Info(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(infoStyle.Render(infoPrefix) + msg)
}

// Success prints a success message with green styling
func Success(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(successStyle.Render(successPrefix) + msg)
}

// Warning prints a warning message with yellow styling
func Warning(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Println(warningStyle.Render(warningPrefix) + msg)
}

// Error prints an error message with red styling
func Error(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Fprintln(os.Stderr, errorStyle.Render(errorPrefix)+msg)
}

// Header prints a header with underline styling
func Header(title string) {
	fmt.Println(headerStyle.Render(title))
}

// Box prints content in a styled box
func Box(content string) {
	fmt.Println(boxStyle.Render(content))
}

// BoxWithTitle prints content in a styled box with a title
func BoxWithTitle(title, content string) {
	titleLine := titleBoxStyle.Render(title)
	box := boxStyle.Render(content)
	fmt.Println(titleLine)
	fmt.Println(box)
}

// Print outputs text without any styling
func Print(format string, a ...interface{}) {
	fmt.Printf(format, a...)
}

// Println outputs text on a new line without styling
func Println(a ...interface{}) {
	fmt.Println(a...)
}

// Printf outputs formatted text without styling
func Printf(format string, a ...interface{}) {
	fmt.Printf(format, a...)
}

// Bar represents a bar in a bar chart
type Bar struct {
	Label string
	Value int
}

// BarChart prints a simple horizontal bar chart using lipgloss styling
func BarChart(bars []Bar) {
	if len(bars) == 0 {
		return
	}

	// Find max value for scaling
	maxValue := 0
	maxLabelLen := 0
	for _, bar := range bars {
		if bar.Value > maxValue {
			maxValue = bar.Value
		}
		if len(bar.Label) > maxLabelLen {
			maxLabelLen = len(bar.Label)
		}
	}

	// Bar width
	maxBarWidth := 40

	// Styles for the chart
	barActiveStyle := lipgloss.NewStyle().Foreground(green)
	barInactiveStyle := lipgloss.NewStyle().Foreground(red)
	labelBarStyle := lipgloss.NewStyle().Foreground(white)
	valueBarStyle := lipgloss.NewStyle().Foreground(gray)

	fmt.Println()
	for _, bar := range bars {
		// Calculate bar width
		width := 0
		if maxValue > 0 {
			width = (bar.Value * maxBarWidth) / maxValue
		}
		if width < 1 && bar.Value > 0 {
			width = 1
		}

		// Create the bar
		barStr := strings.Repeat("█", width)

		// Color based on label
		var styledBar string
		if strings.ToLower(bar.Label) == "active" {
			styledBar = barActiveStyle.Render(barStr)
		} else {
			styledBar = barInactiveStyle.Render(barStr)
		}

		// Format label with padding
		paddedLabel := fmt.Sprintf("%-*s", maxLabelLen, bar.Label)

		// Print the bar
		fmt.Printf("  %s %s %s\n",
			labelBarStyle.Render(paddedLabel),
			styledBar,
			valueBarStyle.Render(fmt.Sprintf("(%d)", bar.Value)))
	}
	fmt.Println()
}

// CyanBox prints cyan text in a padded box (like the welcome screen)
func CyanBox(text string) {
	style := lipgloss.NewStyle().
		Foreground(cyan).
		Bold(true).
		Padding(3, 5).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(cyan)

	fmt.Println(style.Render(text))
}
