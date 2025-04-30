package main

import (
	"log"
	"slices"

	"github.com/compiuta-origin/connhex-cli/cli"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "connhex-cli",
		Short: "Connhex CLI",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// TODO: add more commands if needed
			skipCommands := []string{"login"}
			if slices.Contains(skipCommands, cmd.Name()) || cmd.CalledAs() == "help" {
				return
			}

			config, err := cli.LoadConfig()
			if err != nil {
				log.Fatal(err)
			}

			if !cli.IsAuthenticated(config) {
				log.Println("You're not authenticated or you token has expired. Logging in...")
				loginCmd := cli.NewLoginCmd()
				loginCmd.SetArgs([]string{})
				if err := loginCmd.Execute(); err != nil {
					log.Fatal(err)
				}
			}
		},
	}

	rootCmd.AddCommand(cli.NewLoginCmd())
	rootCmd.AddCommand(cli.NewConfigCmd())
	rootCmd.AddCommand(cli.NewProvisionCmd())

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}

}
