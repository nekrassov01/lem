# lem

[![CI](https://github.com/nekrassov01/lem/actions/workflows/ci.yml/badge.svg)](https://github.com/nekrassov01/lem/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/nekrassov01/lem)](https://goreportcard.com/report/github.com/nekrassov01/lem)
![GitHub](https://img.shields.io/github/license/nekrassov01/lem)
![GitHub](https://img.shields.io/github/v/release/nekrassov01/lem)

## Overview

LEM stands for Local Environments Manager. This tool is intended for configurations where, for example, back-end APIs, front-end UIs, infrastructure resource definitions, etc. are managed in a single repository, and provides utilities for managing .env files that should be in separate directory roots in one central .env based on configuration files. based on a configuration file.

## Features

The functionality is very small and supports only the following:

- Configuration file initialization
- Configuration file validation
- Splitting, prefix substitution, and distribution of the central .env
- Monitoring of central .env and reflection of changes

## Optional features

The following features are available by enabling your settings:

- Detects environment variables with empty values and exits with an error
- Automatically generate `.envrc` and use `watch_file` to have direnv monitor `.env`

## Commands

```text
NAME:
   lem - The local env manager for monorepo

USAGE:
   lem [global options] [command [command options]]

VERSION:
   0.0.0 (revision: b821f30)

COMMANDS:
   init      Initialize the configuration file to current directory
   validate  Validate that the configuration file is executable
   run       Deliver env files to the specified directories based on configuration
   watch     Watch changes in the central env and run continuously

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```

### Init

```text
NAME:
   lem init - Initialize the configuration file to current directory

USAGE:
   lem init [command [command options]]

DESCRIPTION:
   Init generates a lem.example.toml in the current directory.
   You can customize this file for your use.

OPTIONS:
   --help, -h  show help
```

### Validate

```text
NAME:
   lem validate - Validate that the configuration file is executable

USAGE:
   lem validate [command [command options]]

DESCRIPTION:
   Validate validates whether the configuration file in the current directory is executable.
   In addition to syntax checks, it also checks whether the path exists.

OPTIONS:
   --config string, -c string  set configuration file path (default: "lem.toml")
   --help, -h                  show help
```

### Run

```text
NAME:
   lem run - Deliver env files to the specified directories based on configuration

USAGE:
   lem run [command [command options]]

DESCRIPTION:
   Run splits the central env based on configuration and distributes it to each directory.
   It also checks for empty values based on configuration.

OPTIONS:
   --config string, -c string  set configuration file path (default: "lem.toml")
   --stage string, -s string   set stage context to run (default: "default")
   --help, -h                  show help
```

### Watch

```text
NAME:
   lem watch - Watch changes in the central env and run continuously

USAGE:
   lem watch [command [command options]]

DESCRIPTION:
   Watch continuously monitors changes in the central env and synchronizes changes to each directory.

OPTIONS:
   --config string, -c string  set configuration file path (default: "lem.toml")
   --stage string, -s string   set stage context to run (default: "default")
   --help, -h                  show help
```

## Configuration

Set up with TOML format as follows:

```toml
[stage]
default = "<central-env-dir>/.env"
dev = "<central-env-dir>/.env.development"
stg = "<central-env-dir>/.env.staging"
prod = "<central-env-dir>/.env.production"

[group.api]
prefix = "API"
dir = "./backend"
replace = ["REPLACEABLE1"]
check = true
direnv = ["api", "ui"]

[group.ui]
prefix = "UI"
dir = "./frontend"
replace = ["REPLACEABLE2"]
check = true
direnv = ["ui"]
```

| Table        | Key        | Value           | Description                                                                                                         |
| ------------ | ---------- | --------------- | ------------------------------------------------------------------------------------------------------------------- |
| `stage`      | `<string>` | string          | The pairs of stage name and .env file path. If not specified, `default` is used.                                    |
| `group.<id>` | `prefix`   | string          | The prefixes environment variables to be delivered by the group.                                                    |
| `group.<id>` | `dir`      | string          | The destination for the group to be delivered.                                                                      |
| `group.<id>` | `replace`  | array\<string\> | The Prefixes of the environment variable to be delivered after being replaced by the `prefix` defined by the group. |
| `group.<id>` | `check`    | bool            | Whether the group performs an empty value check or not.                                                             |
| `group.<id>` | `direnv`   | array\<id\>     | Automatically generate `.envrc` in each directory, write `watch_file` to track changes.                             |

## Installation

Install with homebrew

```sh
brew install nekrassov01/tap/lem
```

Install with go

```sh
go install github.com/nekrassov01/lem@latest
```

Or download binary from [releases](https://github.com/nekrassov01/lem/releases)

## Shell completion

Supported Shells are as follows:

- bash
- zsh
- fish
- pwsh

```sh
lem completion bash|zsh|fish|pwsh

# In the case of bash
source <(lem completion bash)
```

## Todo

- [x] Support direnv Integration
- [ ] Logging

## Author

[nekrassov01](https://github.com/nekrassov01)

## License

[MIT](https://github.com/nekrassov01/lem/blob/main/LICENSE)
