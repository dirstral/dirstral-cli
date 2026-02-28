package app

import (
	"errors"
	"fmt"

	"github.com/manifoldco/promptui"
)

const logo = "\033[38;5;214m\n         ▄██████████▄                         ██████╗ ██╗██████╗ ███████╗████████╗██████╗  █████╗ ██╗\n      ▄████████████████▄                      ██╔══██╗██║██╔══██╗██╔════╝╚══██╔══╝██╔══██╗██╔══██╗██║\n    ▄████████████████████▄                    ██║  ██║██║██████╔╝███████╗   ██║   ██████╔╝███████║██║\n    ██████████████████████                     ██║  ██║██║██╔══██╗╚════██║   ██║   ██╔══██╗██╔══██║██║\n\033[38;5;208m\n    ██████████████████████                     ██████╔╝██║██║  ██║███████║   ██║   ██║  ██║██║  ██║███████╗\n    ██████████████████████                     ╚═════╝ ╚═╝╚═╝  ╚═╝╚══════╝   ╚═╝   ╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝\n\033[0m"

type StartChoice string

const (
	ChoiceBreeze     StartChoice = "Breeze (Text Chat)"
	ChoiceTempest    StartChoice = "Tempest (Voice Chat)"
	ChoiceLighthouse StartChoice = "Lighthouse (Host MCP)"
	ChoiceQuit       StartChoice = "Quit"
)

func ShowStartScreen() (StartChoice, error) {
	fmt.Println(logo)
	fmt.Println("Welcome to dirstral")
	fmt.Println("Tip: choose Lighthouse first to start dir2mcp quickly.")
	fmt.Println()

	items := []string{string(ChoiceBreeze), string(ChoiceTempest), string(ChoiceLighthouse), string(ChoiceQuit)}
	prompt := promptui.Select{
		Label: "Select mode",
		Items: items,
		Size:  len(items),
	}
	_, result, err := prompt.Run()
	if err != nil {
		if errors.Is(err, promptui.ErrInterrupt) {
			return ChoiceQuit, nil
		}
		return ChoiceQuit, err
	}
	return StartChoice(result), nil
}
