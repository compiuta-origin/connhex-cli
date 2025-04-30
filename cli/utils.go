package cli

import (
	"fmt"

	"github.com/fatih/color"
)

func logUsage(u string) {
	fmt.Printf(color.GreenString("\nusage: %s\n\n"), u)
}

func logError(err error) {
	boldRed := color.New(color.FgRed, color.Bold)
	boldRed.Print("\nerror: ")

	fmt.Printf("%s\n\n", color.RedString(err.Error()))
}

func logOK(s string) {
	fmt.Printf("\n%s\n\n", color.YellowString(s))
}
