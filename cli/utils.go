package cli

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/fatih/color"
)

func sanitizeConnhexInstance(instance string) (string, error) {
	instance = strings.TrimSpace(instance)

	if strings.HasPrefix(instance, "http://") {
		return "", errors.New("http protocol not supported")
	}

	if strings.HasPrefix(instance, "https://") {
		parsedURL, err := url.Parse(instance)
		if err != nil {
			return "", fmt.Errorf("invalid URL: %w", err)
		}
		instance = parsedURL.Host
	}

	return instance, nil
}

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
