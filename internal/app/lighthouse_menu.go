package app

const (
	lighthouseActionStart  = "Start Server"
	lighthouseActionStatus = "Server Status"
	lighthouseActionStop   = "Stop Server"
	lighthouseActionBack   = "Back"
)

func chooseLighthouseAction() (string, error) {
	items := LighthouseMenuItems()
	options := make([]menuOption, 0, len(items))
	for _, item := range items {
		options = append(options, menuOption{Label: item, Value: item})
	}

	return runInteractiveMenu(menuSpec{
		Title:        "Lighthouse",
		Intro:        []string{"Manage the local dir2mcp host process."},
		Options:      options,
		QuitValue:    lighthouseActionBack,
		ControlsText: "Controls: arrows navigate, Enter select, q/esc back",
	})
}

func LighthouseMenuItems() []string {
	return []string{lighthouseActionStart, lighthouseActionStatus, lighthouseActionStop, lighthouseActionBack}
}
