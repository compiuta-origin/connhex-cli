package cli

import (
	"encoding/json"
	"fmt"

	"github.com/compiuta-origin/connhex-cli/internal/config"
	"github.com/spf13/cobra"
)

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

			cfg, err := config.Load()
			if err != nil {
				logError(fmt.Errorf("failed to load configuration: %w", err))
				return
			}

			output, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				logError(fmt.Errorf("failed to format configuration: %w", err))
				return
			}

			logOK(string(output))
		},
	}

	return cmd
}
