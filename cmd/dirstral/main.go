package main

import (
	"fmt"
	"os"

	"github.com/alibilge/dirstral-cli/internal/app"
)

func main() {
	if err := app.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
