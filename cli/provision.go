package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"encoding/csv"
	"strings"

	"github.com/compiuta-origin/connhex-cli/internal/sdk"
	"github.com/spf13/cobra"
)

const (
	defManufacturingDeviceSchema            = "devices"
	defManufacturingDeviceSerialNumberField = "serial_number"
	defManufacturingDeviceConnhexIdField    = "connhex_id"
	jsonExt                                 = ".json"
	csvExt                                  = ".csv"
)

var (
	errNoDeviceToProvision     = errors.New("no device to provision")
	errCSVEmpty                = errors.New("CSV file must contain a header row and at least one data row")
	errCSVMissingRequiredField = errors.New("missing required field")
	errCSVReadError            = errors.New("error reading CSV file")
)

type CLIProvisionData struct {
	Name              string `json:"name,omitempty"`
	InitId            string `json:"init_id"`
	InitKey           string `json:"init_key"`
	MigrationKey      string `json:"migration_key,omitempty"`
	MigrationKeyQuota int    `json:"migration_key_quota,omitempty"`
	Model             string `json:"model,omitempty"`
}

type Device struct {
	// Manufacturing fields depends on the service schema
	Manufacturing map[string]interface{} `json:"manufacturing"`
	Provision     CLIProvisionData       `json:"provision"`
	Tenant        string                 `json:"tenant"`
}

type ManufacturingDeviceConfig struct {
	schema            string
	SerialNumberField string
	ConnhexIdField    string
}

func (device *Device) toProvisionData() sdk.ProvisionData {
	return sdk.ProvisionData{
		Name:              device.Provision.Name,
		InitId:            device.Provision.InitId,
		InitKey:           device.Provision.InitKey,
		MigrationKey:      device.Provision.MigrationKey,
		MigrationKeyQuota: device.Provision.MigrationKeyQuota,
		Model:             device.Provision.Model,
		Tenant:            device.Tenant,
	}
}

func (device *Device) toManufacturingData() sdk.Resource {
	resource := sdk.Resource{}

	for key, value := range device.Manufacturing {
		resource[key] = value
	}
	resource["tenant"] = device.Tenant

	return resource
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
			Manufacturing: make(map[string]interface{}),
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
			if strings.HasPrefix(colName, "provision.") {
				fieldName := strings.TrimPrefix(colName, "provision.")
				switch fieldName {
				case "name":
					device.Provision.Name = value
				case "init_id":
					device.Provision.InitId = value
				case "init_key":
					device.Provision.InitKey = value
				case "migration_key":
					device.Provision.MigrationKey = value
				case "migration_key_quota":
					quota, err := strconv.Atoi(value)
					if err == nil {
						device.Provision.MigrationKeyQuota = quota
					}
				case "model":
					device.Provision.Model = value
				}
				continue
			}
			if strings.HasPrefix(colName, "manufacturing.") {
				fieldName := strings.TrimPrefix(colName, "manufacturing.")
				device.Manufacturing[fieldName] = value
			}
		}

		// Validate required fields
		requiredFields := []struct {
			column string
			field  string
		}{
			{"provision.init_id", "InitId"},
			{"provision.init_key", "InitKey"},
			{"tenant", "Tenant"},
		}

		for _, req := range requiredFields {
			var fieldValue string

			switch req.field {
			case "InitId":
				fieldValue = device.Provision.InitId
			case "InitKey":
				fieldValue = device.Provision.InitKey
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

func getProvisionData(devices []Device) []sdk.ProvisionData {
	data := []sdk.ProvisionData{}

	for _, device := range devices {
		data = append(data, device.toProvisionData())
	}
	return data
}

func getManufacturingData(devices []Device, connectables []sdk.Connectable, mdc ManufacturingDeviceConfig) []sdk.Resource {
	data := []sdk.Resource{}

	serialNumberField := mdc.SerialNumberField
	connhexIdField := mdc.ConnhexIdField

	for _, device := range devices {
		manufacturing := device.toManufacturingData()
		for _, connectable := range connectables {
			if connectable.Metadata["init_id"] == manufacturing[serialNumberField] {
				manufacturing[connhexIdField] = connectable.ID
			}
		}
		data = append(data, manufacturing)
	}

	return data
}

func provisionDevices(devices []Device, mdc ManufacturingDeviceConfig) error {
	config, err := LoadConfig()
	if err != nil {
		return err
	}

	provisionData := getProvisionData(devices)
	provisionResult, err := chxsdk.BulkProvision(provisionData, config.Token)
	if err != nil {
		return err
	}

	manufacturingData := getManufacturingData(devices, provisionResult.Things, mdc)

	if _, err := chxsdk.CreateResources(manufacturingData, "manufacturing", mdc.schema, config.Token); err != nil {
		ids := make([]string, len(provisionResult.Things))
		for i := range ids {
			ids[i] = provisionResult.Things[i].ID
		}

		// Ignoring unprovisioning errors
		chxsdk.BulkUnprovision(ids, config.Token)

		return err
	}

	return nil
}

func provisionDevicesFromFile(path string, mdc ManufacturingDeviceConfig) error {
	devices, err := parseDeviceProvisioningFile(path)
	if err != nil {
		return err
	}
	if len(devices) == 0 {
		return errNoDeviceToProvision
	}

	if err := provisionDevices(devices, mdc); err != nil {
		return err
	}

	logOK("Devices successfully provisioned!")

	return nil
}

func NewProvisionCmd() *cobra.Command {
	var manufacturingDeviceSchema string
	var manufacturingDeviceSerialNumberField string
	var manufacturingDeviceConnhexIdField string

	cmd := &cobra.Command{
		Use:   "provision [--schema=<schema-name>] <filename>",
		Short: "Provision devices in bulk",
		Long: `Provision new devices from file (.csv, .json)

CSV File Format:
- The CSV file must have a header row defining the columns
- Required columns:
  * provision.init_id - Device initialization ID
  * provision.init_key - Device initialization key

- Optional "tenant" column to specify the target tenant. If not provided, devices will be provisioned without any assigned tenant.

- Optional provision columns use the prefix "provision." followed by:
  * name, model, migration_key, migration_key_quota

- Custom manufacturing data columns use the prefix "manufacturing." followed by any field name
  Example: manufacturing.serial_number, manufacturing.hw_version`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsage(cmd.Use)
				return
			}

			mdc := ManufacturingDeviceConfig{
				schema:            manufacturingDeviceSchema,
				SerialNumberField: manufacturingDeviceSerialNumberField,
				ConnhexIdField:    manufacturingDeviceConnhexIdField,
			}

			if err := provisionDevicesFromFile(args[0], mdc); err != nil {
				logError(err)
				return
			}
		},
	}

	cmd.Flags().StringVar(&manufacturingDeviceSchema, "schema",
		defManufacturingDeviceSchema, "Manufacturing device schema")
	cmd.Flags().StringVar(&manufacturingDeviceSerialNumberField, "serial-number-field",
		defManufacturingDeviceSerialNumberField, "Manufacturing device serial number field")
	cmd.Flags().StringVar(&manufacturingDeviceConnhexIdField, "connhex-id-field",
		defManufacturingDeviceConnhexIdField, "Manufacturing device Connhex ID field")

	return cmd
}
