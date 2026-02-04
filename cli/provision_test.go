package cli

import (
	"os"
	"strings"
	"testing"
)

func writeTempCSV(t *testing.T, content string) *os.File {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "*.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatal(err)
	}
	if _, err := f.Seek(0, 0); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestParseCSVDevices_OnlyInitIdRequired(t *testing.T) {
	f := writeTempCSV(t, "provision.init_id\ndev-001\n")
	devices, err := parseCSVDevices(f)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if len(devices) != 1 {
		t.Fatalf("expected 1 device, got %d", len(devices))
	}
	if devices[0].Provision.InitId != "dev-001" {
		t.Errorf("unexpected InitId: %s", devices[0].Provision.InitId)
	}
}

func TestParseCSVDevices_MissingInitId(t *testing.T) {
	f := writeTempCSV(t, "provision.init_key,tenant\nkey-001,acme\n")
	devices, err := parseCSVDevices(f)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if err := validateDevices(devices, defManufacturingDeviceSerialNumberField); err == nil {
		t.Fatal("expected error for missing provision.init_id")
	}
}

func TestParseCSVDevices_TenantOptional(t *testing.T) {
	f := writeTempCSV(t, "provision.init_id,provision.init_key\ndev-001,key-001\n")
	devices, err := parseCSVDevices(f)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if devices[0].Tenant != "" {
		t.Errorf("expected empty tenant, got: %s", devices[0].Tenant)
	}
}

func TestParseCSVDevices_InitKeyOptional(t *testing.T) {
	f := writeTempCSV(t, "provision.init_id,tenant\ndev-001,acme\n")
	devices, err := parseCSVDevices(f)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if devices[0].Provision.InitKey != "" {
		t.Errorf("expected empty InitKey before defaults, got: %s", devices[0].Provision.InitKey)
	}
}

func TestApplyDefaults_GeneratesInitKey(t *testing.T) {
	devices := []Device{{}}
	devices[0].Provision.InitId = "dev-001"
	applyDefaults(devices)
	if !isValidUUID(devices[0].Provision.InitKey) {
		t.Errorf("expected a valid UUID for InitKey, got: %s", devices[0].Provision.InitKey)
	}
}

func TestApplyDefaults_PreservesExistingInitKey(t *testing.T) {
	devices := []Device{{}}
	devices[0].Provision.InitId = "dev-001"
	devices[0].Provision.InitKey = "my-custom-key"
	applyDefaults(devices)
	if devices[0].Provision.InitKey != "my-custom-key" {
		t.Errorf("expected InitKey to be preserved, got: %s", devices[0].Provision.InitKey)
	}
}

func TestParseCSVDevices_ModelMustBeUUID(t *testing.T) {
	f := writeTempCSV(t, "provision.init_id,provision.model\ndev-001,not-a-uuid\n")
	devices, err := parseCSVDevices(f)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if err := validateDevices(devices, defManufacturingDeviceSerialNumberField); err == nil {
		t.Fatal("expected error for non-UUID model")
	}
}

func TestParseCSVDevices_ValidUUIDModel(t *testing.T) {
	f := writeTempCSV(t, "provision.init_id,provision.model\ndev-001,550e8400-e29b-41d4-a716-446655440000\n")
	devices, err := parseCSVDevices(f)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if devices[0].Provision.Model != "550e8400-e29b-41d4-a716-446655440000" {
		t.Errorf("unexpected Model: %s", devices[0].Provision.Model)
	}
}

func TestParseCSVDevices_ManufacturingTenants(t *testing.T) {
	f := writeTempCSV(t, "provision.init_id,tenant\ndev-001,acme\n")
	devices, err := parseCSVDevices(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resource := devices[0].toManufacturingData()
	if _, ok := resource["tenant"]; ok {
		t.Error("manufacturing data should not have 'tenant' key")
	}
	tenants, ok := resource["tenants"]
	if !ok {
		t.Fatal("manufacturing data should have 'tenants' key")
	}
	list, ok := tenants.([]string)
	if !ok || len(list) != 1 || list[0] != "acme" {
		t.Errorf("unexpected tenants value: %v", tenants)
	}
}

func TestParseCSVDevices_NoTenantNoTenantsKey(t *testing.T) {
	f := writeTempCSV(t, "provision.init_id\ndev-001\n")
	devices, err := parseCSVDevices(f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resource := devices[0].toManufacturingData()
	if _, ok := resource["tenants"]; ok {
		t.Error("manufacturing data should not have 'tenants' key when tenant is empty")
	}
}

func TestValidateDevices_SerialNumberFieldRejected(t *testing.T) {
	f := writeTempCSV(t, "provision.init_id,manufacturing.serial_number\ndev-001,SN-001\n")
	devices, err := parseCSVDevices(f)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if err := validateDevices(devices, defManufacturingDeviceSerialNumberField); err == nil {
		t.Fatal("expected error when manufacturing.serial_number is specified")
	}
}

func TestValidateDevices_DuplicateInitKey(t *testing.T) {
	f := writeTempCSV(t, "provision.init_id,provision.init_key\ndev-001,same-key\ndev-002,same-key\n")
	devices, err := parseCSVDevices(f)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if err := validateDevices(devices, defManufacturingDeviceSerialNumberField); err == nil {
		t.Fatal("expected error for duplicate provision.init_key")
	}
}

func TestValidateDevices_DuplicateInitId(t *testing.T) {
	f := writeTempCSV(t, "provision.init_id\ndev-001\ndev-001\n")
	devices, err := parseCSVDevices(f)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if err := validateDevices(devices, defManufacturingDeviceSerialNumberField); err == nil {
		t.Fatal("expected error for duplicate provision.init_id")
	}
}

func TestValidateDevices_MissingInitId(t *testing.T) {
	err := validateDevices([]Device{{}}, defManufacturingDeviceSerialNumberField)
	if err == nil {
		t.Fatal("expected error for missing init_id")
	}
	if !strings.Contains(err.Error(), "provision.init_id") {
		t.Errorf("error should mention provision.init_id, got: %v", err)
	}
}

func TestValidateDevices_InvalidUUIDModel(t *testing.T) {
	d := Device{}
	d.Provision.InitId = "dev-001"
	d.Provision.Model = "not-a-uuid"
	err := validateDevices([]Device{d}, defManufacturingDeviceSerialNumberField)
	if err == nil {
		t.Fatal("expected error for invalid UUID model")
	}
}
