package app

import "github.com/manifoldco/promptui"

func chooseLighthouseAction() (string, error) {
	items := []string{"Start Server", "Server Status", "Stop Server", "Back"}
	prompt := promptui.Select{Label: "Lighthouse", Items: items, Size: len(items)}
	_, result, err := prompt.Run()
	if err != nil {
		return "Back", err
	}
	return result, nil
}
