package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const authService = "auth"

func (sdk chxSDK) Login(user string, password string) (string, error) {
	authServiceURL := sdk.getServiceURL(authService)

	// Get the login action URL
	initialReq, err := http.NewRequest("GET", fmt.Sprintf("%s/self-service/login/api", authServiceURL), nil)
	if err != nil {
		return "", err
	}

	initialResp, err := sdk.sendRequest(initialReq, "", CTJSON)
	if err != nil {
		return "", err
	}
	defer initialResp.Body.Close()

	if initialResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%v: %w", initialResp.Status, ErrFailedLogin)
	}

	var initialRespBody map[string]interface{}
	if err := json.NewDecoder(initialResp.Body).Decode(&initialRespBody); err != nil {
		return "", err
	}

	// Extract action URL
	ui, ok := initialRespBody["ui"].(map[string]interface{})
	if !ok {
		return "", err
	}

	actionURL, ok := ui["action"].(string)
	if !ok || actionURL == "" {
		return "", err
	}

	// Send credentials to the action URL
	loginData := map[string]string{
		"identifier": user,
		"password":   password,
		"method":     "password",
	}

	loginDataJSON, err := json.Marshal(loginData)
	if err != nil {
		return "", err
	}

	loginReq, err := http.NewRequest("POST", actionURL, bytes.NewBuffer(loginDataJSON))
	if err != nil {
		return "", err
	}
	loginReq.Header.Set("Accept", "application/json")

	loginResp, err := sdk.sendRequest(loginReq, "", CTJSON)
	if err != nil {
		return "", err
	}
	defer loginResp.Body.Close()

	if loginResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%v: %w", loginResp.Status, ErrFailedLogin)
	}

	var sessionData map[string]interface{}
	if err := json.NewDecoder(loginResp.Body).Decode(&sessionData); err != nil {
		return "", err
	}

	// Extract token
	sessionToken, ok := sessionData["session_token"].(string)
	if !ok || sessionToken == "" {
		return "", err
	}

	return sessionToken, nil
}
