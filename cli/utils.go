package cli

import (
	"fmt"
	"net/url"
	"strings"
	"unicode"

	"github.com/fatih/color"
)

func sanitizeConnhexInstance(instance string) (string, error) {
	instance = strings.TrimSpace(instance)

	if strings.HasPrefix(instance, "http://") {
		return "", fmt.Errorf("http protocol not supported")
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

func ToCamelCase(s string) string {
	if s == "" {
		return s
	}

	words := strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	})

	if len(words) == 0 {
		return ""
	}

	result := strings.ToLower(words[0])

	// Convert the rest of the words to title case and join them
	for _, word := range words[1:] {
		if word != "" {
			result += strings.Title(strings.ToLower(word))
		}
	}

	return result
}

func convertSliceItemsToCamelCase(slice []interface{}) []interface{} {
	result := make([]interface{}, len(slice))

	for i, item := range slice {
		if nestedMap, ok := item.(map[string]interface{}); ok {
			// If the slice item is a map, convert its keys
			result[i] = ConvertMapKeysToCamelCase(nestedMap)
		} else if nestedSlice, ok := item.([]interface{}); ok {
			// If the slice item is another slice, recursively convert
			result[i] = convertSliceItemsToCamelCase(nestedSlice)
		} else {
			// For simple values, just copy
			result[i] = item
		}
	}

	return result
}

func ConvertMapKeysToCamelCase(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	for k, v := range m {
		camelKey := ToCamelCase(k)

		// Handle nested maps
		if nestedMap, ok := v.(map[string]interface{}); ok {
			result[camelKey] = ConvertMapKeysToCamelCase(nestedMap)
		} else if nestedSlice, ok := v.([]interface{}); ok {
			// Handle slices that might contain maps
			result[camelKey] = convertSliceItemsToCamelCase(nestedSlice)
		} else {
			// For simple values, just copy with the new key
			result[camelKey] = v
		}
	}

	return result
}
