package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

const provisionService = "provision"

type ProvisionData struct {
	Name              string `json:"name,omitempty"`
	InitId            string `json:"init_id"`
	InitKey           string `json:"init_key"`
	MigrationKey      string `json:"migration_key,omitempty"`
	MigrationKeyQuota int    `json:"migration_key_quota,omitempty"`
	Model             string `json:"model,omitempty"`
	Tenant            string `json:"tenant,omitempty"`
}

type Connectable struct {
	ID       string                 `json:"id,omitempty"`
	Name     string                 `json:"name,omitempty"`
	Key      string                 `json:"key,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Tenant   string                 `json:"tenant,omitempty"`
	Model    string                 `json:"model,omitempty"`
}

type BulkProvisionResult struct {
	Things    []Connectable `json:"things"`
	Processed int           `json:"processed"`
	Failed    int           `json:"failed"`
	Errors    []string      `json:"errors,omitempty"`
}

func (sdk chxSDK) BulkProvision(data []ProvisionData, token string) (BulkProvisionResult, error) {
	result := BulkProvisionResult{}

	body, err := json.Marshal(data)
	if err != nil {
		return result, err
	}

	reqURL := fmt.Sprintf("%s/things/bulk", sdk.getServiceURL(provisionService))
	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(body))
	if err != nil {
		return result, err

	}

	resp, err := sdk.sendRequest(req, token, CTJSON)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	if statusCode < http.StatusOK || statusCode >= http.StatusBadRequest {
		return result, fmt.Errorf("%v: %w", resp.Status, ErrFailedCreation)
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return result, err
	}

	if result.Failed > 0 {
		return result, ErrFailedCreation
	}

	return result, nil
}

func (sdk chxSDK) BulkUnprovision(ids []string, token string) error {
	body, err := json.Marshal(ids)
	if err != nil {
		return err
	}

	reqURL := fmt.Sprintf("%s/things/bulk", sdk.getServiceURL(provisionService))
	req, err := http.NewRequest("DELETE", reqURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	resp, err := sdk.sendRequest(req, token, CTJSON)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	statusCode := resp.StatusCode
	if statusCode < http.StatusOK || statusCode >= http.StatusBadRequest {
		return fmt.Errorf("%v: %w", resp.Status, ErrFailedRemoval)
	}

	return nil
}
