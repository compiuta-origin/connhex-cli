package sdk

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Resource fields depends on the service schema
type Resource map[string]interface{}

type IncludedResource struct {
	Type       string                 `json:"type"`
	ID         string                 `json:"id"`
	Attributes map[string]interface{} `json:"attributes"`
	Links      struct {
		Self string `json:"self"`
	} `json:"links"`
}

type JsonApiResponseData struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Links struct {
		Self string `json:"self"`
	} `json:"links"`
	Relationships map[string]interface{} `json:"relationships,omitempty"`
	Attributes    map[string]interface{} `json:"attributes"`
}

type JsonApiResponse struct {
	JsonAPI struct {
		Version string `json:"version"`
	} `json:"jsonapi"`
	Meta struct {
		Count int `json:"count"`
	} `json:"meta"`
	Links struct {
		Self  string `json:"self"`
		First string `json:"first"`
		Last  string `json:"last"`
	} `json:"links"`
	Data     interface{}        `json:"data"` // Can be a single `JsonApiResponseData` object or array
	Included []IncludedResource `json:"included,omitempty"`
}

func (sdk chxSDK) CreateResources(resources []Resource, service string, schema string, token string) (JsonApiResponse, error) {
	result := JsonApiResponse{}

	// Convert map keys to camelcase: it is required from the manufacturing service when posting requests with JSON serializer
	camelized := make([]Resource, len(resources))
	for i := range camelized {
		camelized[i] = ConvertMapKeysToCamelCase(resources[i])
	}

	body, err := json.Marshal(camelized)
	if err != nil {
		return result, err
	}

	reqURL := fmt.Sprintf("%s/%s", sdk.getServiceURL(service), schema)
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

	return result, nil
}
