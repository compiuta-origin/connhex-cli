package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	tokenExpirationTime = 24 * time.Hour
)

var errNotAuthenticated = errors.New("You are not authenticated. Please login...")

// FIXME: pass token as params? We're opening a file for every time
func setAuthHeader(request *http.Request) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}
	token := config.Token

	request.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))

	return nil
}

func login(connhexInstance string, email string, password string) (string, error) {
	// Get the login action URL
	initialReq, err := http.NewRequest("GET", fmt.Sprintf("https://accounts.%s/auth/self-service/login/api", connhexInstance), nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}
	initialReq.Header.Set("Accept", "application/json")

	client := &http.Client{}
	initialResp, err := client.Do(initialReq)
	if err != nil {
		return "", fmt.Errorf("error getting login action URL: %w", err)
	}
	defer initialResp.Body.Close()

	if initialResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", initialResp.StatusCode)
	}

	var initialRespBody map[string]interface{}
	if err := json.NewDecoder(initialResp.Body).Decode(&initialRespBody); err != nil {
		return "", fmt.Errorf("error parsing response: %w", err)
	}

	// Extract action URL
	ui, ok := initialRespBody["ui"].(map[string]interface{})
	if !ok {
		return "", errors.New("missing ui field in response")
	}

	actionURL, ok := ui["action"].(string)
	if !ok || actionURL == "" {
		return "", errors.New("unable to get login action URL")
	}

	// Send credentials to the action URL
	loginData := map[string]string{
		"identifier": email,
		"password":   password,
		"method":     "password",
	}

	loginDataJSON, err := json.Marshal(loginData)
	if err != nil {
		return "", fmt.Errorf("error marshaling login data: %w", err)
	}

	loginReq, err := http.NewRequest("POST", actionURL, bytes.NewBuffer(loginDataJSON))
	if err != nil {
		return "", fmt.Errorf("error creating login request: %w", err)
	}

	loginReq.Header.Set("Accept", "application/json")
	loginReq.Header.Set("Content-Type", "application/json")

	loginResp, err := client.Do(loginReq)
	if err != nil {
		return "", fmt.Errorf("error logging in: %w", err)
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("login failed with status code: %d", loginResp.StatusCode)
	}

	var sessionData map[string]interface{}
	if err := json.NewDecoder(loginResp.Body).Decode(&sessionData); err != nil {
		return "", fmt.Errorf("error parsing session response: %w", err)
	}

	// Extract token
	sessionToken, ok := sessionData["session_token"].(string)
	if !ok || sessionToken == "" {
		return "", errors.New("unable to extract session token")
	}

	return sessionToken, nil
}

func IsAuthenticated(config Config) bool {
	return config.Token != "" && time.Now().Before(config.ExpiresAt)
}

func NewLoginCmd() *cobra.Command {
	cmd := cobra.Command{
		Use:   "login [connhex-instance email password]",
		Short: "Login to Connhex instance",
		// We use `RunE` beacause this command is used in the main PersistentPreRun
		RunE: func(cmd *cobra.Command, args []string) error {
			var connhexInstance, email, password string

			if len(args) > 0 && len(args) != 3 {
				logUsage(cmd.Use)
				return nil
			}

			if len(args) == 3 {
				connhexInstance = args[0]
				email = args[1]
				password = args[2]
			}
			if len(args) == 0 {
				fmt.Print("Enter Connhex instance: ")
				fmt.Scanln(&connhexInstance)

				fmt.Print("Enter email: ")
				fmt.Scanln(&email)

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
			if connhexInstance == "" || email == "" || password == "" {
				return errors.New("instance, email, and password cannot be empty")
			}

			connhexInstance, err := sanitizeConnhexInstance(connhexInstance)
			if err != nil {
				return err
			}

			token, err := login(connhexInstance, email, password)
			if err != nil {
				return err
			}

			config := Config{
				ConnhexInstance: connhexInstance,
				Token:           token,
				User:            email,
				ExpiresAt:       time.Now().Add(tokenExpirationTime),
			}

			if err := SaveConfig(&config); err != nil {
				return err
			}

			logOK(fmt.Sprintf("Login successful! You token is: %s", token))

			return nil
		},
	}

	return &cmd
}
