package cli

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/compiuta-origin/connhex-cli/internal/config"
	"github.com/compiuta-origin/connhex-cli/internal/sdk"
	"github.com/google/uuid"
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
	errInvalidFieldValue       = errors.New("invalid field value")
)

func isValidUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func applyDefaults(devices []Device) {
	for i := range devices {
		if devices[i].Provision.InitKey == "" {
			devices[i].Provision.InitKey = uuid.New().String()
		}
	}
}

func validateDevices(devices []Device, serialNumberField string) error {
	seenInitId := make(map[string]int, len(devices))
	seenInitKey := make(map[string]int, len(devices))
	for i, device := range devices {
		if device.Provision.InitId == "" {
			return fmt.Errorf("row %d: %w 'provision.init_id'", i+1, errCSVMissingRequiredField)
		}
		if device.Provision.Model != "" && !isValidUUID(device.Provision.Model) {
			return fmt.Errorf("row %d: %w: 'provision.model' must be a valid UUID", i+1, errInvalidFieldValue)
		}
		if _, ok := device.Manufacturing[serialNumberField]; ok {
			return fmt.Errorf("row %d: %w: 'manufacturing.%s' must not be specified — it is automatically set from 'provision.init_id'", i+1, errInvalidFieldValue, serialNumberField)
		}
		if firstRow, dup := seenInitId[device.Provision.InitId]; dup {
			return fmt.Errorf("row %d: %w: 'provision.init_id' %q already used at row %d", i+1, errInvalidFieldValue, device.Provision.InitId, firstRow)
		}
		seenInitId[device.Provision.InitId] = i + 1
		if firstRow, dup := seenInitKey[device.Provision.InitKey]; dup {
			return fmt.Errorf("row %d: %w: 'provision.init_key' %q already used at row %d", i+1, errInvalidFieldValue, device.Provision.InitKey, firstRow)
		}
		seenInitKey[device.Provision.InitKey] = i + 1
	}
	return nil
}

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
	serialNumberField string
	connhexIdField    string
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
	if device.Tenant != "" {
		resource["tenants"] = []string{device.Tenant}
	}

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

	for _, record := range records[1:] {
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

		devices = append(devices, device)
	}

	return devices, nil
}

func parseDeviceProvisioningFile(path string, serialNumberField string) ([]Device, error) {
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
		devices, err = parseCSVDevices(file)
	case jsonExt:
		err = json.NewDecoder(file).Decode(&devices)
	default:
		return devices, nil
	}

	if err != nil {
		return devices, err
	}
	applyDefaults(devices)
	return devices, validateDevices(devices, serialNumberField)
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

	serialNumberField := mdc.serialNumberField
	connhexIdField := mdc.connhexIdField

	for _, device := range devices {
		manufacturing := device.toManufacturingData()
		manufacturing[serialNumberField] = device.Provision.InitId
		for _, connectable := range connectables {
			if connectable.Metadata["init_id"] == device.Provision.InitId {
				manufacturing[connhexIdField] = connectable.ID
			}
		}
		data = append(data, manufacturing)
	}

	return data
}

func provisionDevices(devices []Device, mdc ManufacturingDeviceConfig) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	provisionData := getProvisionData(devices)
	provisionResult, err := chxsdk.BulkProvision(provisionData, cfg.Token)
	if err != nil {
		return err
	}

	manufacturingData := getManufacturingData(devices, provisionResult.Things, mdc)

	if _, err := chxsdk.CreateResources(manufacturingData, "manufacturing", mdc.schema, cfg.Token); err != nil {
		ids := make([]string, len(provisionResult.Things))
		for i := range ids {
			ids[i] = provisionResult.Things[i].ID
		}
		chxsdk.BulkUnprovision(ids, cfg.Token)

		return err
	}

	return nil
}

func provisionDevicesFromFile(path string, mdc ManufacturingDeviceConfig) error {
	devices, err := parseDeviceProvisioningFile(path, mdc.serialNumberField)
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
  * provision.init_id - Device initialization ID (typically the device serial number)

- Optional columns:
  * provision.init_key - Device initialization key (auto-generated UUID v4 if not provided)
  * tenant            - Target tenant. If not provided, devices are provisioned without an assigned tenant.

- Optional provision columns use the prefix "provision." followed by:
  * name, model (must be a valid model ID), migration_key, migration_key_quota

- Custom manufacturing data columns use the prefix "manufacturing." followed by any field name
  Example: manufacturing.hw_version

Note: the serial number field in the manufacturing record is always populated from provision.init_id.`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				logUsage(cmd.Use)
				return
			}

			mdc := ManufacturingDeviceConfig{
				schema:            manufacturingDeviceSchema,
				serialNumberField: manufacturingDeviceSerialNumberField,
				connhexIdField:    manufacturingDeviceConnhexIdField,
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
