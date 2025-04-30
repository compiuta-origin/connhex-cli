package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const (
	configDirname  = ".connhex"
	configFilename = "config.json"
	configType     = "json"
)

type Config struct {
	ConnhexInstance string    `mapstructure:"connhex_instance"`
	Token           string    `mapstructure:"token"`
	User            string    `mapstructure:"user"`
	ExpiresAt       time.Time `mapstructure:"expires_at"`
}

func Setup() error {
	viper.SetConfigName(configFilename)
	viper.SetConfigType(configType)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, configDirname)
	viper.AddConfigPath(configDir)
	viper.AddConfigPath(".")

	viper.SetDefault("connhex_instance", "")
	viper.SetDefault("token", "")
	viper.SetDefault("user", "")
	viper.SetDefault("expires_at", time.Time{})

	if err := viper.ReadInConfig(); err != nil {
		// It's okay if the config file doesn't exist - we'll create it when needed
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read config: %w", err)
		}
	}

	return nil
}

func Load() (Config, error) {
	var cfg Config

	decodeHook := mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeHookFunc(time.RFC3339),
	)

	if err := viper.Unmarshal(&cfg, viper.DecodeHook(decodeHook)); err != nil {
		return Config{}, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return cfg, nil
}

func Save(cfg Config) error {
	var err error
	cfg.ConnhexInstance, err = SanitizeConnhexInstance(cfg.ConnhexInstance)
	if err != nil {
		return err
	}

	viper.Set("connhex_instance", cfg.ConnhexInstance)
	viper.Set("token", cfg.Token)
	viper.Set("user", cfg.User)
	// Explicitly set expires_at as RFC3339 string
	viper.Set("expires_at", cfg.ExpiresAt.Format(time.RFC3339))

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, configDirname)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, configFilename)
	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
