package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
)

func IsUserAuthenticated() bool {
	token := viper.GetString("token")
	expiresAt := viper.GetTime("expires_at")
	return token != "" && time.Now().Before(expiresAt)
}

func SanitizeConnhexInstance(instance string) (string, error) {
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
