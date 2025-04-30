package cli

import (
	"errors"
	"fmt"
	"syscall"
	"time"

	"github.com/compiuta-origin/connhex-cli/internal/config"
	"github.com/compiuta-origin/connhex-cli/internal/sdk"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	tokenExpirationTime = 24 * time.Hour
)

var (
	ErrInvalidInput = errors.New("invalid input: instance, user and password cannot be empty")
)

func NewLoginCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "login [connhex-instance user password]",
		Short: "Login to Connhex instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			var connhexInstance, user, password string

			if len(args) > 0 && len(args) != 3 {
				logUsage(cmd.Use)
				return nil
			}

			if len(args) == 3 {
				connhexInstance = args[0]
				user = args[1]
				password = args[2]
			}
			if len(args) == 0 {
				fmt.Print("Enter Connhex instance: ")
				fmt.Scanln(&connhexInstance)

				fmt.Print("Enter user: ")
				fmt.Scanln(&user)

				fmt.Print("Enter password: ")
				passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
				if err != nil {
					return err
				}
				// Add a newline after password entry
				fmt.Println()
				password = string(passwordBytes)
			}

			// Validate inputs are not empty
			if connhexInstance == "" || user == "" || password == "" {
				return ErrInvalidInput
			}

			connhexInstance, err := config.SanitizeConnhexInstance(connhexInstance)
			if err != nil {
				return err
			}

			// Initialize the SDK if needed, since the login command
			// doesn't require authentication when first executed
			if chxsdk == nil {
				SetSDK(sdk.NewSDK(connhexInstance))
			}

			token, err := chxsdk.Login(user, password)
			if err != nil {
				return err
			}

			cfg := config.Config{
				ConnhexInstance: connhexInstance,
				Token:           token,
				User:            user,
				ExpiresAt:       time.Now().Add(tokenExpirationTime),
			}

			if err := config.Save(cfg); err != nil {
				return err
			}

			logOK(fmt.Sprintf("Login successful! Your token is: %s", token))

			return nil
		},
	}

	return &cmd
}
