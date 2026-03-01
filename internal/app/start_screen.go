package app

// StartMenuConfig returns the menu configuration for the start screen.
func StartMenuConfig() MenuConfig {
	return MenuConfig{
		Title: "Welcome to dirstral",
		Intro: []string{
			"Launch mode: Breeze (chat), Tempest (voice), Lighthouse (host MCP)",
			"Tip: Lighthouse first is the fastest demo path.",
		},
		Items: []MenuItem{
			{Label: string(ChoiceBreeze), Description: "Interactive text chat with MCP tools", Value: string(ChoiceBreeze)},
			{Label: string(ChoiceTempest), Description: "Voice-powered agent loop", Value: string(ChoiceTempest)},
			{Label: string(ChoiceLighthouse), Description: "Host and monitor dir2mcp server", Value: string(ChoiceLighthouse)},
			{Label: string(ChoiceQuit), Description: "Exit dirstral", Value: string(ChoiceQuit)},
		},
		ShowLogo: true,
		Controls: "arrows navigate · enter select · q/esc quit",
	}
}
