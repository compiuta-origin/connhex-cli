package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
)

var (
	configDirname  = ".connhex"
	configFilename = "config.json"
)

type Config struct {
	ConnhexInstance string    `json:"connhex_instance"`
	Token           string    `json:"token"`
	User            string    `json:"user"`
	ExpiresAt       time.Time `json:"expires_at"`
}

func (c *Config) sanitize() error {
	sanitizedInstance, err := sanitizeConnhexInstance(c.ConnhexInstance)
	if err != nil {
		return err
	}
	c.ConnhexInstance = sanitizedInstance

	return nil
}

func (c *Config) Save() error {
	if err := c.sanitize(); err != nil {
		return err
	}

	configFile, err := getConfigFilePath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configFile)

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	data, err := json.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0600)
}

func (c *Config) IsUserAuthenticated() bool {
	return c.Token != "" && time.Now().Before(c.ExpiresAt)
}

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configDir := filepath.Join(homeDir, configDirname)
	return filepath.Join(configDir, configFilename), nil
}

func LoadConfig() (Config, error) {
	config := Config{}

	configFile, err := getConfigFilePath()
	if err != nil {
		return config, err
	}

	// Check if config file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return config, nil
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(data, &config)
	return config, err
}

func NewConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Display the current configuration",
		Long:  "Display the current Connhex CLI configuration including the instance URL and authentication details",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 0 {
				logUsage(cmd.Use)
				return
			}

			config, err := LoadConfig()
			if err != nil {
				logError(fmt.Errorf("failed to load configuration: %w", err))
				return
			}

			output, err := json.MarshalIndent(config, "", "  ")
			if err != nil {
				logError(fmt.Errorf("failed to format configuration: %w", err))
				return
			}

			logOK(string(output))
		},
	}

	return cmd
}
