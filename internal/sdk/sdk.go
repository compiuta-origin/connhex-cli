package sdk

import (
	"errors"
	"fmt"
	"net/http"
)

const (
	CTJSON    ContentType = "application/json"
	CTJSONAPI ContentType = "application/vnd.api+json"
)

// Shared errors across services
var (
	ErrFailedLogin    = errors.New("failed to login")
	ErrFailedCreation = errors.New("failed to create entity")
	ErrFailedRemoval  = errors.New("failed to remove entity")
)

// Type assertion for interface implementation
var _ SDK = (*chxSDK)(nil)

type ContentType string

type chxSDK struct {
	// Domain of the Connhex Cloud instance
	connhexInstance string
	client          *http.Client
}

type SDK interface {
	// Return user session token
	Login(user string, password string) (string, error)

	// Connectables bulk provision
	BulkProvision(data []ProvisionData, token string) (BulkProvisionResult, error)

	// Connectables bulk unprovision
	BulkUnprovision(ids []string, token string) error

	// Create resources
	CreateResources(resources []Resource, service string, schema string, token string) (JsonApiResponse, error)
}

func NewSDK(connhexInstance string) *chxSDK {
	return &chxSDK{
		connhexInstance: connhexInstance,
		client:          &http.Client{},
	}
}

func (sdk chxSDK) sendRequest(req *http.Request, token string, contentType ContentType) (*http.Response, error) {

	if contentType == "" {
		contentType = CTJSON
	}

	req.Header.Add("Content-Type", string(contentType))

	if token != "" {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	}

	return sdk.client.Do(req)
}

func (sdk chxSDK) getServiceURL(service string) string {
	switch service {
	case "auth":
		return fmt.Sprintf("https://accounts.%s/auth", sdk.connhexInstance)
	case "provision":
		return fmt.Sprintf("https://apis.%s/iot/provision", sdk.connhexInstance)
	case "resources":
		return fmt.Sprintf("https://apis.%s/resources", sdk.connhexInstance)
	case "manufacturing":
		return fmt.Sprintf("https://apis.%s/manufacturing", sdk.connhexInstance)
	default:
		return ""
	}
}
