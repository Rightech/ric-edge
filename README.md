# ric-edge

[![Build Status](https://github.com/Rightech/ric-edge/workflows/Build%20and%20release/badge.svg)](https://github.com/Rightech/ric-edge/actions?query=workflow%3A%22Build+and+release%22)
[![GitHub release](https://img.shields.io/github/v/release/Rightech/ric-edge?include_prereleases)](https://github.com/Rightech/ric-edge/releases/tag/v0.6.11)

## config

You can use `config.toml` file to configure core and connectors or specify path via `-config` option.

You can generate minimal required config with

```bash
$ ./core-<os>-<arch> -min-config
```

Minimal configuration

```toml
[core]
    id = "" # id of edge

    [core.cloud]
    # cloud jwt access token
    # minimal scope should be
    #
    # GET models/:id
    # GET objects/:id
    token = ""

    [core.mqtt]
    cert_file = "" # mqtt certificate file path
    key_path = "" # mqtt key file path
```

Also you can generate configuration with default values

```bash
$ ./core-<os>-<arch> -default-config
```

Default configuration

```toml
log_level = "info"
log_format = "text" # output log format (you can use text or json)
ws_port = 9000
check_updates = true
auto_download_updates = false  # if true service will download update and exit

[core]
    id = "" # id of edge
    rpc_timeout = "1m" # how long core should wait response from connector before return timeout error

    [core.db]
    path = "storage.db"
    clean_state = false # should internal state be cleaned on start or not

    [core.cloud]
    url = "https://sandbox.rightech.io/api/v1"
    # cloud jwt access token
    # minimal scope should be
    #
    # GET models/:id
    # GET objects/:id
    token = ""

    [core.mqtt]
    # if cert_file and key_path provided core will be use tls connection
    # in this case make sure your url start with tls://
    url = "tls://sandbox.rightech.io:8883"
    cert_file = "" # mqtt certificate file path
    key_path = "" # mqtt key file path

[modbus]
    mode = "tcp" # rtu and ascii also supported
    addr = "localhost:8000"  # if mode = rtu or ascii there is should be path

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
$ make build_all
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

See [init](/init) folder for systemd services.

## dev

### dev env

You can find some development helpers at tools dir.

To run helper docker containers execute

```bash
$ make dev_env
```

To stop

```bash
$ make stop_env
```

And to remove

```bash
$ make rm_env
```

### git hooks

It is useful to place validate-license script to pre-commit hook. To do this run

```bash
$ ln -f scripts/validate-license.sh .git/hooks/pre-commit
```

Also you can run it manual by

```bash
$ make validate
```

### bumpversion

To prepare releases we use [bumpversion](https://pypi.org/project/bumpversion).
