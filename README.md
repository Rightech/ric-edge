# ric-edge

[![Build Status](https://cloud.drone.io/api/badges/Rightech/ric-edge/status.svg)](https://cloud.drone.io/Rightech/ric-edge)

## config

You can use `config.toml` file to configure core and connectors or specify path via `-config` option.

Configuration with default values

```toml
log_level = "info"
log_format = "text" # output log format (you can use text or json)
ws_port = 9000

[core]
    id = "" # id of edge
    rpc_timeout = "1m" # how long core should wait response from connector before return timeout error

    [core.db]
    path = "storage.db"
    clean_state = false # should internal state be cleaned on start or not

    [core.cloud]
    url = "https://sandbox.rightech.io/api/v1"
    token = ""  # cloud jwt access token

    [core.mqtt]
    # if cert_file and key_path provided core will be use tls connection
    # in this case make sure your url start with tls://
    url = "tcp://localhost:1883"
    cert_file = "" # mqtt certificate file path
    key_path = "" # mqtt key file path

[modbus]
    tcp = true
    rtu = false
    addr = "localhost:8000"

[opcua]
    endpoint = "opc.tcp://localhost:4840"

[snmp]
    host_port="localhost:161"
    version="2c"
    community= "public"
```

## build

To build all services run

```bash
$ make buildall
```

also you can build specific service

```bash
$ make build_core
```

build results will be placed at `build` folder

## run

To run core service use

```bash
$ make run_core
```

To run connectors (modbus connector example)

```bash
$ make run_modbus
```
