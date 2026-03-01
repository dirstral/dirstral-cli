package app

import (
	"os"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/term"
)

const (
	DefaultTerminalWidth = 120
	logoPadding          = 2
	maxLeftPad           = 20
	compactLeftPad       = 2
)

const CompactLogoText = "DIRSTRAL"

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)

var fullLogoLines = []string{
	"    ▄████████▄                                 ██████╗ ██╗██████╗ ███████╗████████╗██████╗  █████╗ ██╗",
	"    █████████████████████████▄                 ██╔══██╗██║██╔══██╗██╔════╝╚══██╔══╝██╔══██╗██╔══██╗██║",
	"    █░░░░░░░░░░░░░░░░░░░░░░░░█                 ██║  ██║██║██████╔╝███████╗   ██║   ██████╔╝███████║██║",
	"    █░░░░░░░░░░░░░░░░░░░░░░░░█                 ██║  ██║██║██╔══██╗╚════██║   ██║   ██╔══██╗██╔══██║██║",
	"    █░░░░░░░░░░░░░░░░░░░░░░░░█                 ██████╔╝██║██║  ██║███████║   ██║   ██║  ██║██║  ██║███████╗",
	"    ▀████████████████████████▀                 ╚═════╝ ╚═╝╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝",
}

var mediumLogoLines = []string{
	" ✦ ▄███▄        ██████╗ ██╗██████╗ ███████╗████████╗██████╗  █████╗ ██╗",
	"   ██████████   ██╔══██╗██║██╔══██╗██╔════╝╚══██╔══╝██╔══██╗██╔══██╗██║",
	"   █░░░░░░░░█   ██║  ██║██║██████╔╝███████╗   ██║   ██████╔╝███████║██║",
	"   █░░░░░░░░█   ██║  ██║██║██╔══██╗╚════██║   ██║   ██╔══██╗██╔══██║██║",
	"   █░░░░░░░░█   ██████╔╝██║██║  ██║███████║   ██║   ██║  ██║██║  ██║███████╗",
	"   ▀████████▀   ╚═════╝ ╚═╝╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝",
}

type StartChoice string

const (
	ChoiceBreeze     StartChoice = "Breeze"
	ChoiceTempest    StartChoice = "Tempest"
	ChoiceLighthouse StartChoice = "Lighthouse"
	ChoiceQuit       StartChoice = "Quit"
)

type LogoTier int

const (
	LogoFull    LogoTier = iota // wide terminals: full logo with large folder
	LogoMedium                  // medium terminals: compact folder + block text
	LogoCompact                 // narrow terminals: plain styled text
)

func StartMenuItems() []string {
	return []string{string(ChoiceBreeze), string(ChoiceTempest), string(ChoiceLighthouse), string(ChoiceQuit)}
}

func ChooseTier(width int) LogoTier {
	fullWidth := maxVisibleWidth(NormalizeLeftSpacing(fullLogoLines))
	if width >= fullWidth+logoPadding {
		return LogoFull
	}
	medWidth := maxVisibleWidth(mediumLogoLines)
	if width >= medWidth+logoPadding {
		return LogoMedium
	}
	return LogoCompact
}

func maxVisibleWidth(lines []string) int {
	max := 0
	for _, line := range lines {
		if w := visibleWidth(line); w > max {
			max = w
		}
	}
	return max
}

func RenderLogo(width int) string {
	tier := ChooseTier(width)
	switch tier {
	case LogoCompact:
		return padLine(paint(CompactLogoText, colorBrandStrong, colorBold), compactLeftPad)
	case LogoMedium:
		styled := make([]string, 0, len(mediumLogoLines))
		for i, line := range mediumLogoLines {
			style := colorBrandStrong
			if i >= len(mediumLogoLines)-2 {
				style = colorBrand
			}
			styled = append(styled, paint(line, style))
		}
		return strings.Join(centerBlockLines(styled, width), "\n")
	default:
		lines := NormalizeLeftSpacing(fullLogoLines)
		styled := make([]string, 0, len(lines))
		for i, line := range lines {
			style := colorBrandStrong
			if i >= len(lines)-2 {
				style = colorBrand
			}
			styled = append(styled, paint(line, style))
		}
		return strings.Join(centerBlockLines(styled, width), "\n")
	}
}

func padLine(line string, pad int) string {
	if pad <= 0 {
		return line
	}
	return strings.Repeat(" ", pad) + line
}

func centerBlockLines(lines []string, width int) []string {
	if len(lines) == 0 {
		return nil
	}
	maxWidth := 0
	for _, line := range lines {
		if w := visibleWidth(line); w > maxWidth {
			maxWidth = w
		}
	}
	pad := 0
	if width > maxWidth {
		pad = clampLeftPad((width - maxWidth) / 2)
	}
	out := make([]string, len(lines))
	left := strings.Repeat(" ", pad)
	for i, line := range lines {
		out[i] = left + line
	}
	return out
}

func clampLeftPad(n int) int {
	if n < 0 {
		return 0
	}
	if n > maxLeftPad {
		return maxLeftPad
	}
	return n
}

func visibleWidth(s string) int {
	plain := ansiPattern.ReplaceAllString(s, "")
	return len([]rune(plain))
}

func NormalizeLeftSpacing(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	minIndent := -1
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if trimmed == "" {
			continue
		}
		indent := len(line) - len(trimmed)
		if minIndent == -1 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		dup := make([]string, len(lines))
		copy(dup, lines)
		return dup
	}
	out := make([]string, len(lines))
	for i, line := range lines {
		if len(line) >= minIndent {
			out[i] = line[minIndent:]
		} else {
			out[i] = line
		}
	}
	return out
}

func TerminalWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	if raw := strings.TrimSpace(os.Getenv("COLUMNS")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	return DefaultTerminalWidth
}
