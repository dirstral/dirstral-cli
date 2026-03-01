package app

import (
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	revealDelay = 30 * time.Millisecond
)

// revealTickMsg triggers showing the next menu item.
type revealTickMsg struct{}

// animationsEnabled returns false when NO_COLOR is set or TERM=dumb.
func animationsEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if strings.ToLower(os.Getenv("TERM")) == "dumb" {
		return false
	}
	return true
}

// tickReveal returns a tea.Cmd that fires a revealTickMsg after the delay.
func tickReveal() tea.Cmd {
	return tea.Tick(revealDelay, func(t time.Time) tea.Msg {
		return revealTickMsg{}
	})
}
