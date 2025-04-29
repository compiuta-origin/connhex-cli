package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"encoding/csv"
	"strings"

	"github.com/spf13/cobra"
)

const (
	defDeviceManufacturingSchema = "devices"
	jsonExt                      = ".json"
	csvExt                       = ".csv"
)

var (
	errNoDeviceToProvision     = errors.New("no device to provision")
	errProvisionFailed         = errors.New("failed to provision connectables")
	errCSVEmpty                = errors.New("CSV file must contain a header row and at least one data row")
	errCSVMissingRequiredField = errors.New("missing required field")
	errCSVReadError            = errors.New("error reading CSV file")
)

// DeviceManufacturing's properties depends on the user-defined schema
type DeviceManufacturing map[string]interface{}

type Connectable struct {
	Name              string `json:"name,omitempty"`
	InitId            string `json:"init_id"`
	InitKey           string `json:"init_key"`
	MigrationKey      string `json:"migration_key,omitempty"`
	MigrationKeyQuota *int   `json:"migration_key_quota,omitempty"`
	Model             string `json:"model,omitempty"`
}

type ConnectableWithTenant struct {
	Connectable
	Tenant string `json:"tenant"`
}

type Device struct {
	DeviceManufacturing map[string]interface{} `json:"manufacturing"`
	Connectable         Connectable            `json:"connectable"`
	Tenant              string                 `json:"tenant"`
}

func parseCSVDevices(file *os.File) ([]Device, error) {
	devices := []Device{}

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return devices, fmt.Errorf("%w: %v", errCSVReadError, err)
	}

	// Need at least one row (header) and one data row
	if len(records) < 2 {
		return devices, errCSVEmpty
	}

	header := records[0]
	for i := range header {
		header[i] = strings.TrimSpace(header[i])
	}

	for i, record := range records[1:] {
		device := Device{
			DeviceManufacturing: make(map[string]interface{}),
		}

		for j, value := range record {
			if j >= len(header) {
				continue
			}

			value = strings.TrimSpace(value)

			if value == "" {
				continue
			}

			colName := header[j]

			// Handle all field types
			if colName == "tenant" {
				device.Tenant = value
				continue
			}
			if strings.HasPrefix(colName, "connectable.") {
				fieldName := strings.TrimPrefix(colName, "connectable.")
				switch fieldName {
				case "name":
					device.Connectable.Name = value
				case "init_id":
					device.Connectable.InitId = value
				case "init_key":
					device.Connectable.InitKey = value
				case "migration_key":
					device.Connectable.MigrationKey = value
				case "migration_key_quota":
					quota, err := strconv.Atoi(value)
					if err == nil {
						device.Connectable.MigrationKeyQuota = &quota
					}
				case "model":
					device.Connectable.Model = value
				}
				continue
			}
			if strings.HasPrefix(colName, "manufacturing.") {
				fieldName := strings.TrimPrefix(colName, "manufacturing.")
				device.DeviceManufacturing[fieldName] = value
			}
		}

		// Validate required fields
		requiredFields := []struct {
			column string
			field  string
		}{
			{"connectable.init_id", "InitId"},
			{"connectable.init_key", "InitKey"},
			{"tenant", "Tenant"},
		}

		for _, req := range requiredFields {
			var fieldValue string

			switch req.field {
			case "InitId":
				fieldValue = device.Connectable.InitId
			case "InitKey":
				fieldValue = device.Connectable.InitKey
			case "Tenant":
				fieldValue = device.Tenant
			}

			if fieldValue == "" {
				return devices, fmt.Errorf("row %d: %w '%s'", i+2, errCSVMissingRequiredField, req.column)
			}
		}

		devices = append(devices, device)
	}

	return devices, nil
}

func parseDeviceProvisioningFile(path string) ([]Device, error) {
	devices := []Device{}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return devices, err
	}

	file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
	if err != nil {
		return devices, err
	}
	defer file.Close()

	switch filepath.Ext(path) {
	case csvExt:
		return parseCSVDevices(file)
	case jsonExt:
		if err := json.NewDecoder(file).Decode(&devices); err != nil {
			return devices, err
		}
	default:
		return devices, nil
	}

	return devices, nil
}

func getConnectablesData(devices []Device) []ConnectableWithTenant {
	cwts := []ConnectableWithTenant{}

	for _, device := range devices {
		cwt := ConnectableWithTenant{
			Connectable: device.Connectable,
			Tenant:      device.Tenant,
		}
		cwts = append(cwts, cwt)
	}

	return cwts
}

func provisionConnectables(connhexInstance string, cwts []ConnectableWithTenant) ([]string, error) {
	body, err := json.Marshal(cwts)
	if err != nil {
		return nil, err
	}

	reqURL := fmt.Sprintf("https://apis.%s/iot/provision/things/bulk", connhexInstance)
	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(body))
	if err != nil {
		return nil, err

	}
	req.Header.Add("Content-Type", "application/json")
	if err := setAuthHeader(req); err != nil {
		return nil, err
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()

	statusCode := res.StatusCode
	if statusCode < http.StatusOK || statusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("error code %d from Connhex Provision service", statusCode)
	}

	provisionedConnectables := struct {
		Things []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
			Key  string `json:"key"`
		} `json:"things"`
		Processed int `json:"processed"`
		Failed    int `json:"failed"`
	}{}
	if err := json.NewDecoder(res.Body).Decode(&provisionedConnectables); err != nil {
		return nil, err
	}
	if provisionedConnectables.Failed > 0 {
		return nil, errProvisionFailed
	}

	ids := make([]string, len(provisionedConnectables.Things))
	for i, thing := range provisionedConnectables.Things {
		ids[i] = thing.ID
	}

	return ids, nil
}

func unprovisionConnectables(connhexInstance string, ids []string) error {
	body, err := json.Marshal(ids)
	if err != nil {
		return err
	}

	reqURL := fmt.Sprintf("https://apis.%s/iot/provision/things/bulk", connhexInstance)
	req, err := http.NewRequest("DELETE", reqURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	if err = setAuthHeader(req); err != nil {
		return err
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer req.Body.Close()

	statusCode := res.StatusCode
	if statusCode < http.StatusOK || statusCode >= http.StatusBadRequest {
		return fmt.Errorf("error code %d from Connhex Provision service", statusCode)
	}
	return nil
}

func getManufacturingData(devices []Device) []DeviceManufacturing {
	dms := []DeviceManufacturing{}

	for _, device := range devices {
		dm := DeviceManufacturing{}

		for k, v := range device.DeviceManufacturing {
			dm[k] = v
		}

		dm["tenant"] = device.Tenant
		dms = append(dms, dm)
	}

	return dms
}

func createManufacturingData(connhexInstance string, dwts []DeviceManufacturing, deviceManufacturingSchema string) error {
	if deviceManufacturingSchema == "" {
		deviceManufacturingSchema = defDeviceManufacturingSchema
	}

	// Convert map keys to camelcase: it is required from the manufacturing service when posting bulk requests
	camelized := make([]map[string]interface{}, len(dwts))
	for i := range camelized {
		camelized[i] = ConvertMapKeysToCamelCase(dwts[i])
	}

	body, err := json.Marshal(camelized)
	if err != nil {
		return err
	}

	reqURL := fmt.Sprintf("https://apis.%s/manufacturing/%s", connhexInstance, deviceManufacturingSchema)
	req, err := http.NewRequest("POST", reqURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	if err = setAuthHeader(req); err != nil {
		return err
	}

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer req.Body.Close()

	statusCode := res.StatusCode
	if statusCode < http.StatusOK || statusCode >= http.StatusBadRequest {
		return fmt.Errorf("error code %d from Connhex Manufacturing service", statusCode)
	}

	return nil
}

func provisionDevices(devices []Device, manufacturingSchema string) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}
	connhexInstance := config.ConnhexInstance

	cwts := getConnectablesData(devices)
	ids, err := provisionConnectables(connhexInstance, cwts)
	if err != nil {
		return err
	}

	dms := getManufacturingData(devices)
	if err = createManufacturingData(connhexInstance, dms, manufacturingSchema); err != nil {
		unprovisionConnectables(connhexInstance, ids)
		return err
	}

	return nil
}

func provisionDevicesFromFile(path string, manufacturingSchema string) error {
	devices, err := parseDeviceProvisioningFile(path)
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		return errNoDeviceToProvision
	}

	if err = provisionDevices(devices, manufacturingSchema); err != nil {
		return err
	}

	logOK("Devices successfully provisioned!")

	return nil
}

func NewProvisionCmd() *cobra.Command {
	var deviceManufacturingSchema string

	cmd := &cobra.Command{
		Use:   "provision [--schema=<schema-name>] <filename>",
		Short: "Provision devices in bulk",
		Long: `Provision new devices from file (.csv, .json)

CSV File Format:
- The CSV file must have a header row defining the columns
- Required columns:
  * connectable.init_id - Device initialization ID
  * connectable.init_key - Device initialization key

- Optional "tenant" column to specify the target tenant. If not provided, the device will be provisioned without any assigned tenant.

- Optional connectable columns use the prefix "connectable." followed by:
  * name, model, migration_key, migration_key_quota

- Custom manufacturing data columns use the prefix "manufacturing." followed by any field name
  Example: manufacturing.serial_number, manufacturing.hw_version`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsage(cmd.Use)
				return
			}

			if err := provisionDevicesFromFile(args[0], deviceManufacturingSchema); err != nil {
				logError(err)
				return
			}
		},
	}

	cmd.Flags().StringVarP(&deviceManufacturingSchema, "schema", "s", defDeviceManufacturingSchema,
		"Device manufacturing schema")

	return cmd
}
