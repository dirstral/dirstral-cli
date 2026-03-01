package app

// LighthouseMenuConfig returns the menu configuration for the lighthouse submenu.
func LighthouseMenuConfig() MenuConfig {
	return MenuConfig{
		Title: "Lighthouse",
		Intro: []string{"Manage the local dir2mcp host process."},
		Items: []MenuItem{
			{Label: lighthouseActionStart, Description: "Launch dir2mcp and stream logs", Value: lighthouseActionStart},
			{Label: lighthouseActionStatus, Description: "Check process and endpoint health", Value: lighthouseActionStatus},
			{Label: lighthouseActionStop, Description: "Terminate managed dir2mcp", Value: lighthouseActionStop},
			{Label: lighthouseActionBack, Description: "Return to main menu", Value: lighthouseActionBack},
		},
		ShowLogo: false,
		Controls: "up/down or j/k move · enter select · esc/q back",
	}
}
