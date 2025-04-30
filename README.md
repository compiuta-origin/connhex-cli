# Connhex CLI

A command-line interface tool for interacting with [Connhex](https://connhex.com?utm_campaign=content&utm_medium=web&utm_source=github&utm_content=connhex+cli) Cloud.

## Overview

Connhex CLI allows you to interact with your Connhex Cloud instance directly from your terminal. It provides commands for authentication, configuration management, device provisioning, etc.

## Installation

### Prerequisites

- Go 1.20 or later

### Building from source

1. Clone the repository:

   ```bash
   git clone https://github.com/compiuta-origin/connhex-cli.git
   cd connhex-cli
   ```

2. Build the CLI:
   ```bash
   make cli
   ```

## Getting Started

> [!NOTE]  
> This guide assumes the `connhex-cli` binary is globally available (i.e. stored in a directory that has been added to your `PATH`. Common choices are `/usr/local/bin` or a custom directory like `~/bin`.)

If this is not your case, navigate to the folder containing the `connhex-cli` binary. You will then be able to execute all of the commands below by replacing `connhex-cli` with `./connhex-cli`.

### Authentication

Before using Connhex CLI, you need to authenticate with your Connhex instance:

```bash
connhex-cli login
```

You will be prompted to enter:

- Your Connhex instance domain
- Email address
- Password

Alternatively, you can provide all parameters in a single command:

```bash
connhex-cli login <connhex-instance> <email> <password>
```

Authentication tokens are stored in `~/.connhex/config.json` and are valid for 24 hours.

### View Configuration

To check your current configuration:

```bash
connhex-cli config
```

This will display your Connhex instance, authentication token, and other settings.

## Device Provisioning

### Bulk Device Provisioning

You can [provision](https://connhex.com/docs/api/core/provision/provision-a-connectable?utm_campaign=content&utm_medium=web&utm_source=github&utm_content=connhex+cli) multiple devices at once using a CSV or JSON file:

```bash
connhex-cli provision [--schema=<schema-name>] <filename>
```

#### CSV Format

CSV files must have a header row with the following required columns:

- `tenant` - Target tenant for the device
- `connectable.init_id` - Device initialization ID (typically the serial number)
- `connectable.init_key` - Device initialization key

Optional connectable columns are:

- `connectable.name` - Device name
- `connectable.model` - Device model
- `connectable.migration_key` - Alternative key used by the device to download its configuration if the main key is not available.
- `connectable.migration_key_quota` - The number of times the migration key can be used.

All custom [manufacturing](https://connhex.com/docs/manufacturing/intro?utm_campaign=content&utm_medium=web&utm_source=github&utm_content=connhex+cli) data columns use the prefix `manufacturing.` followed by the field name. These will depend on your Connhex Manufacturing [configuration](https://connhex.com/docs/manufacturing/default-configuration?utm_campaign=content&utm_medium=web&utm_source=github&utm_content=connhex+cli):

- `manufacturing.serial_number`
- `manufacturing.hw_version`
- etc.

Example CSV file:

```csv
tenant,connectable.init_id,connectable.init_key,connectable.name,manufacturing.serial_number
connhex,device123,SN123456789,Sensor 1,SN123456789
connhex,device124,SN123456790,Sensor 2,SN123456790
```

#### JSON Format

Alternatively, you can use a JSON file with the following structure:

```json
[
  {
    "tenant": "connhex",
    "connectable": {
      "name": "Sensor 1",
      "init_id": "SN123456789",
      "init_key": "8f8b7361-3380-4dbe-9cfd-7d5a4d6eb167",
      "model": "sensor-v1"
    },
    "manufacturing": {
      "serial_number": "SN123456789",
      "hw_version": "1.0"
    }
  }
]
```

#### Manufacturing Schema

By default, the CLI uses the `devices` schema for manufacturing data. You can specify a different schema using the `--schema` flag:

```bash
connhex-cli provision --schema=custom-schema devices.csv
```

## Configuration

The Connhex CLI stores its configuration in `~/.connhex/config.json`. This includes:

- `connhex_instance`: Your Connhex instance domain
- `token`: Authentication token
- `user`: User email
- `expires_at`: Token expiration timestamp

## Troubleshooting

_I'm getting the following error: `error: failed to provision connectables`_

Make sure all fields in your `devices` CSV file are valid: common errors include specifying tenants or models that don't exist.

_I'm getting the following error: `cmd/main.go:5:2: package slices is not in GOROOT`_

Make sure your Go version is 1.20+.

## License

The project is licensed under the MIT License.
