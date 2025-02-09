# OPC-UA Device Service

> [!NOTE]
> This service is designed for EdgeX Foundry v4.0

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

Define devices for device-sdk to auto upload device profile and create device instance. Please modify `devices.yml` file found under the `./cmd/res/devices` folder.

```yaml
deviceList:
  - name: SimulationServer
    profileName: OPCUA-Server
    description: OPCUA device is created for test purpose
    labels:
      - test
    protocols:
      opcua:
        Endpoint: "opc.tcp://127.0.0.1:53530/OPCUA/SimulationServer"
        # Security policy: None, Basic128Rsa15, Basic256, Basic256Sha256, Aes128Sha256RsaOaep, Aes256Sha256RsaPss. Default: None
        Policy: None
        # Security mode: None, Sign, SignAndEncrypt. Default: None
        Mode: None
        # Path to cert.pem. Required for security mode/policy != None
        CertFile: ""
        # Path to private key.pem. Required for security mode/policy != None
        KeyFile: ""
        Resources: [Counter, Random]
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

A REST endpoint is available at `POST /api/v3/call` to handle method calls. The request body is defined as follows:

```json
{
  "device": "Device_Name",
  "method": "Device_Resource_Name",
  "parameters": [""]
}
```

Both `device` and `method` properties are required, and `parameters` is optional.

## Build and Run Binary

```bash
make build
EDGEX_SECURITY_SECRET_STORE=false make run
```

## Build and Run a Container Image

```bash
make docker
```

### Running with EdgeX Foundry in No-security Mode

Update [docker-compose-no-secty.yml](https://github.com/edgexfoundry/edgex-compose/blob/v3.1/docker-compose-no-secty.yml) with the following service configuration:

```yml
services:
  device-opcua:
    container_name: edgex-device-opcua
    depends_on:
      consul:
        condition: service_started
      core-data:
        condition: service_started
      core-metadata:
        condition: service_started
    environment:
      EDGEX_SECURITY_SECRET_STORE: "false"
      SERVICE_HOST: edgex-device-opcua
    hostname: edgex-device-opcua
    # Update image ref if necessary
    image: edgexfoundry/device-opcua-go:0.0.0-dev 
    networks:
      edgex-network: null
    ports:
      - mode: ingress
        host_ip: 127.0.0.1
        target: 59997
        published: "59997"
        protocol: tcp
    read_only: true
    restart: always
    security_opt:
      - no-new-privileges:true
    user: 2002:2001
    volumes:
      - type: bind
        source: /etc/localtime
        target: /etc/localtime
        read_only: true
        bind:
          create_host_path: true
```

### Running with EdgeX Foundry in Secure Mode

Update [docker-compose.yml](https://github.com/edgexfoundry/edgex-compose/blob/v3.1/docker-compose.yml) with the following configuration changes:

```yml
services:
  # Update custom proxy route - New for EdgeX Foundry v3.1
  security-proxy-setup:
    environment:
      EDGEX_ADD_PROXY_ROUTE: device-opcua.http://edgex-device-opcua:59997
  # Update known secrets and secretstore tokens
  security-secretstore-setup:
    environment:
      EDGEX_ADD_KNOWN_SECRETS: redisdb[device-opcua],message-bus[device-opcua]
      EDGEX_ADD_SECRETSTORE_TOKENS: "device-opcua"
  # Update custom Consul ACL roles
  consul:
    environment:
      EDGEX_ADD_REGISTRY_ACL_ROLES: "device-opcua"
  # Device service configuration
  device-opcua:
    command:
      - /device-opcua
      - -cp=consul.http://edgex-core-consul:8500
      - --registry
    container_name: edgex-device-opcua
    depends_on:
      consul:
        condition: service_started
      core-data:
        condition: service_started
      core-metadata:
        condition: service_started
      security-bootstrapper:
        condition: service_started
    entrypoint:
      - /edgex-init/ready_to_run_wait_install.sh
    environment:
      EDGEX_SECURITY_SECRET_STORE: "true"
      PROXY_SETUP_HOST: edgex-security-proxy-setup
      SECRETSTORE_HOST: edgex-vault
      SERVICE_HOST: edgex-device-opcua
      STAGEGATE_BOOTSTRAPPER_HOST: edgex-security-bootstrapper
      STAGEGATE_BOOTSTRAPPER_STARTPORT: "54321"
      STAGEGATE_DATABASE_HOST: edgex-redis
      STAGEGATE_DATABASE_PORT: "6379"
      STAGEGATE_DATABASE_READYPORT: "6379"
      STAGEGATE_PROXYSETUP_READYPORT: "54325"
      STAGEGATE_READY_TORUNPORT: "54329"
      STAGEGATE_REGISTRY_HOST: edgex-core-consul
      STAGEGATE_REGISTRY_PORT: "8500"
      STAGEGATE_REGISTRY_READYPORT: "54324"
      STAGEGATE_SECRETSTORESETUP_HOST: edgex-security-secretstore-setup
      STAGEGATE_SECRETSTORESETUP_TOKENS_READYPORT: "54322"
      STAGEGATE_WAITFOR_TIMEOUT: 60s
    hostname: edgex-device-opcua
    # Update image ref if necessary
    image: edgexfoundry/device-opcua-go:0.0.0-dev
    networks:
      edgex-network: null
    ports:
      - mode: ingress
        host_ip: 127.0.0.1
        target: 59997
        published: "59997"
        protocol: tcp
    read_only: true
    restart: always
    security_opt:
      - no-new-privileges:true
    user: 2002:2001
    volumes:
      - type: volume
        source: edgex-init
        target: /edgex-init
        read_only: true
        volume: {}
      - type: bind
        source: /etc/localtime
        target: /etc/localtime
        read_only: true
        bind:
          create_host_path: true
      - type: bind
        source: /tmp/edgex/secrets/device-opcua
        target: /tmp/edgex/secrets/device-opcua
        read_only: true
        bind:
          selinux: z
          create_host_path: true
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
