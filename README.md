# OPC-UA Device Service

## Overview

This repository is a Go-based EdgeX Foundry Device Service which uses OPC-UA protocol to interact with the devices or IoT objects.

## Features

1. Subscribe/Unsubscribe one or more variables
2. Execute read command
3. Execute write command
4. Execute method

## Prerequisites

- Edgex-go: core data, core command, core metadata
- OPCUA Server (Prosys Simulation Server, for example)

## Predefined configuration

### Simulation Server

Download the Prosys OPC UA Simulation Server from [here](https://www.prosysopc.com/products/opc-ua-simulation-server/). Install and run it to have access to the default configured resources.

### Pre-defined Devices

Define devices for device-sdk to auto upload device profile and create device instance. Please modify `devices.toml` file found under the `./cmd/res/devices` folder.

```yaml
DeviceList:
  - Name: SimulationServer
    ProfileName: OPCUA-Server
    Description: OPCUA device is created for test purpose
    Labels:
      - test
    Protocols:
      opcua:
        Endpoint: "opc.tcp://127.0.0.1:53530/OPCUA/SimulationServer"
        # Security policy: None, Basic128Rsa15, Basic256, Basic256Sha256. Default: None
        Policy: None
        # Security mode: None, Sign, SignAndEncrypt. Default: None
        Mode: None
        # Path to cert.pem. Required for security mode/policy != None
        CertFile: ""
        # Path to private key.pem. Required for security mode/policy != None
        KeyFile: ""
        Resources: "Counter,Random"
```

## Device Profile

A Device Profile can be thought of as a template of a type or classification of a Device.

Write a device profile for your own devices; define `deviceResources` and `deviceCommands`. Please refer to `cmd/res/profiles/OpcuaServer.yaml`.

### Using Methods

OPC UA methods can be referenced in the device profile and called with a read command. An example of a method instance might look something like this:

```yaml
deviceResources:
  - name: "SetDefaultsMethod"
    description: "Set all variables to their default values"
    isHidden: "false" # Specifies if the method can be called
    properties:
      valueType: "String" # Response type will always be String
      readWrite: "R"
    attributes: { methodId: "ns=5;s=Defaults", objectId: "ns=5;i=1111" }
```

Notice that method calls require specifying the Node ID of both the method and its parent object.

A REST endpoint is available at `POST /api/v2/call` to handle method calls. The request body is defined as follows:

```json
{
  "device": "Device_Name",
  "method": "Device_Resource_Name",
  "parameters": [""]
}
```

Both `device` and `method` properties are required, and `parameters` is optional.

## Build and Run

```bash
make build
EDGEX_SECURITY_SECRET_STORE=false make run
```

## Build a Container Image

```bash
make docker
```

## Build with NATS Messaging

Currently, the NATS Messaging capability (NATS MessageBus) is opt-in at build time. This means that the published Docker image and Snaps do not include the NATS messaging capability.

The following make commands will build the local binary or local Docker image with NATS messaging capability included.

```bash
make build-nats
make docker-nats
```

The locally built Docker image can then be used in place of the published Docker image in your compose file.
See [Compose Builder](https://github.com/edgexfoundry/edgex-compose/tree/main/compose-builder#gen) `nat-bus` option to generate compose file for NATS and local dev images.

## Testing

Running unit tests starts a mock OPCUA server on port `48408`.

The mock server defines the following attributes:

| Variable Name | Type      | Default Value          | Writable |
| ------------- | --------- | ---------------------- | -------- |
| `ro_bool`     | `Boolean` | `True`                 |          |
| `rw_bool`     | `Boolean` | `True`                 | ✅       |
| `ro_int32`    | `Int32`   | `5`                    |          |
| `rw_int32`    | `Int32`   | `5`                    | ✅       |
| `square`      | `Method`  | `Int64` (return value) |          |

All attributes are defined in `ns=2`.

```bash
# Install requirements (if necessary)
python3 -m pip install opcua
# Run tests
make test
```

## Reference

- [EdgeX Foundry Services](https://github.com/edgexfoundry/edgex-go)
- [Go OPCUA library](https://github.com/gopcua/opcua)
- [OPCUA Server](https://www.prosysopc.com/products/opc-ua-simulation-server)
