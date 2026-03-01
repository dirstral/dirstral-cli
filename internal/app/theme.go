package app

import "fmt"

const (
	ansiReset = "\033[0m"
	ansiClear = "\033[2J\033[H"

	colorBrandStrong = "\033[38;5;214m"
	colorBrand       = "\033[38;5;208m"
	colorMuted       = "\033[38;5;245m"
	colorSubtle      = "\033[38;5;242m"
	colorError       = "\033[38;5;203m"
	colorBold        = "\033[1m"

	spacingSection = "\n"
)

func paint(text string, styles ...string) string {
	if text == "" {
		return ""
	}
	styled := ""
	for _, style := range styles {
		styled += style
	}
	return styled + text + ansiReset
}

func printSpacer() {
	fmt.Print(spacingSection)
}

func statusLine(label, details string) string {
	return fmt.Sprintf("%s %s", paint(label, colorBrandStrong, colorBold), paint(details, colorMuted))
}

func errorLine(err error) string {
	return paint(fmt.Sprintf("error: %v", err), colorError, colorBold)
}
