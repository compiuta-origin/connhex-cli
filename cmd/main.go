package main

import (
	"log"
	"slices"

	"github.com/compiuta-origin/connhex-cli/cli"
	"github.com/compiuta-origin/connhex-cli/internal/config"
	"github.com/compiuta-origin/connhex-cli/internal/sdk"
	"github.com/spf13/cobra"
)

func main() {
	if err := config.Setup(); err != nil {
		log.Fatal(err)
	}

	rootCmd := &cobra.Command{
		Use:   "connhex-cli",
		Short: "Connhex CLI",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			skipCommands := []string{"login"}
			if slices.Contains(skipCommands, cmd.Name()) || cmd.CalledAs() == "help" {
				return
			}

			cfg, err := config.Load()
			if err != nil {
				log.Fatal(err)
			}

			if !config.IsUserAuthenticated() {
				log.Println("You're not authenticated or you token has expired. Logging in...")

				loginCmd := cli.NewLoginCmd()
				loginCmd.SetArgs([]string{})
				if err := loginCmd.Execute(); err != nil {
					log.Fatal(err)
				}

				// We need to reload the config since it has been updated during the login
				cfg, err = config.Load()
				if err != nil {
					log.Fatal(err)
				}
			}

			s := sdk.NewSDK(cfg.ConnhexInstance)
			cli.SetSDK(s)
		},
	}

	rootCmd.AddCommand(cli.NewLoginCmd())
	rootCmd.AddCommand(cli.NewConfigCmd())
	rootCmd.AddCommand(cli.NewProvisionCmd())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}

}
