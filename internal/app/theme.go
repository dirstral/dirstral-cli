package app

import (
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Legacy ANSI constants kept for logo rendering which pre-dates lipgloss.
const (
	ansiReset = "\033[0m"

	colorBrandStrong = "\033[38;5;214m"
	colorBrand       = "\033[38;5;208m"
	colorMuted       = "\033[38;5;245m"
	colorSubtle      = "\033[38;5;242m"
	colorError       = "\033[38;5;203m"
	colorBold        = "\033[1m"
)

// Lipgloss color palette.
var (
	clrBrandStrong = lipgloss.Color("214")
	clrMuted       = lipgloss.Color("245")
	clrSubtle      = lipgloss.Color("242")
	clrGreen       = lipgloss.Color("114")
)

// Reusable lipgloss styles.
var (
	styleBrandStrong  = lipgloss.NewStyle().Foreground(clrBrandStrong).Bold(true)
	styleMuted        = lipgloss.NewStyle().Foreground(clrMuted)
	styleSubtle       = lipgloss.NewStyle().Foreground(clrSubtle)
	styleSelected     = lipgloss.NewStyle().Foreground(clrBrandStrong).Bold(true)
	styleDescription  = lipgloss.NewStyle().Foreground(clrSubtle).Italic(true)
	styleSelectedDesc = lipgloss.NewStyle().Foreground(clrMuted).Italic(true)
	styleGreen        = lipgloss.NewStyle().Foreground(clrGreen)
)

// paint wraps text in raw ANSI codes. Used by logo rendering.
// Respects NO_COLOR environment variable.
func paint(text string, styles ...string) string {
	if text == "" {
		return ""
	}
	if os.Getenv("NO_COLOR") != "" {
		return text
	}
	styled := ""
	for _, style := range styles {
		styled += style
	}
	return styled + text + ansiReset
}

func statusLine(label, details string) string {
	return fmt.Sprintf("%s %s", paint(label, colorBrandStrong, colorBold), paint(details, colorMuted))
}

func errorLine(err error) string {
	return paint(fmt.Sprintf("error: %v", err), colorError, colorBold)
}
