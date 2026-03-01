package app

var startupTips = []string{
	"Tip: Lighthouse first is the fastest demo path.",
	"Tip: Press q any time to return or quit.",
	"Tip: Use j/k if arrow keys are awkward in your terminal.",
	"Tip: Breeze is best for quick tool-driven lookups.",
}

// StartupTips returns startup hints shown on the home screen.
func StartupTips() []string {
	out := make([]string, len(startupTips))
	copy(out, startupTips)
	return out
}

// StartupTip returns the tip for a specific index.
func StartupTip(index int) string {
	if len(startupTips) == 0 {
		return ""
	}
	if index < 0 {
		index = 0
	}
	return startupTips[index%len(startupTips)]
}

// StartMenuConfig returns the menu configuration for the start screen.
func StartMenuConfig() MenuConfig {
	return MenuConfig{
		Title: "Welcome to Dirstral",
		Intro: []string{
			"Launch mode: Breeze (chat), Tempest (voice), Lighthouse (host MCP)",
			StartupTip(0),
		},
		Items: []MenuItem{
			{Label: string(ChoiceBreeze), Description: "Interactive text chat with MCP tools", Value: string(ChoiceBreeze)},
			{Label: string(ChoiceTempest), Description: "Voice-powered agent loop", Value: string(ChoiceTempest)},
			{Label: string(ChoiceLighthouse), Description: "Host and monitor dir2mcp server", Value: string(ChoiceLighthouse)},
			{Label: "Exit", Description: "Exit Dirstral", Value: string(ChoiceQuit)},
		},
		ShowLogo: true,
		Controls: "up/down or j/k move · enter select · esc/q quit",
	}
}
