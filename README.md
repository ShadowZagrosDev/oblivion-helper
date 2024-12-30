# Oblivion-Helper

Oblivion-Helper is a high-performance utility designed for managing **Sing-Box** directly within the [Oblivion-Desktop](https://github.com/bepass-org/oblivion-desktop) application. It provides seamless cross-platform compatibility for Windows, macOS, and Linux.

<br>
<div align="center">

[![Version](https://img.shields.io/github/v/release/ShadowZagrosDev/oblivion-helper?label=Version&color=blue)](https://github.com/ShadowZagrosDev/oblivion-helper/releases/latest)
[![Downloads](https://img.shields.io/github/downloads/ShadowZagrosDev/oblivion-helper/total?label=Downloads&color=success)](https://github.com/ShadowZagrosDev/oblivion-helper/releases/latest)
[![Stars](https://img.shields.io/github/stars/ShadowZagrosDev/oblivion-helper?style=flat&label=Stars&color=ff69b4)](https://github.com/ShadowZagrosDev/oblivion-helper)

[![Go](https://img.shields.io/badge/Go-1.23.2-00ADD8.svg)](https://go.dev/)
[![gRPC](https://img.shields.io/badge/gRPC-Protocol-009688)](https://grpc.io/)
[![Go Report Card](https://goreportcard.com/badge/github.com/ShadowZagrosDev/oblivion-helper)](https://goreportcard.com/report/github.com/ShadowZagrosDev/oblivion-helper)
[![Code Size](https://img.shields.io/github/languages/code-size/ShadowZagrosDev/oblivion-helper?color=lightgrey)](https://github.com/ShadowZagrosDev/oblivion-helper)
[![Top Language](https://img.shields.io/github/languages/top/ShadowZagrosDev/oblivion-helper?color=yellowgreen)](https://github.com/ShadowZagrosDev/oblivion-helper)

</div>
<br>


## Overview

Oblivion-Helper is a lightweight yet powerful utility built with Go and gRPC. It directly integrates **Sing-Box**, allowing for seamless core functionality and flexible configuration. The application is distributed under the **GNU General Public License version 3 (GPLv3)** to comply with the usage of Sing-Box. *This means that any derivative works, including modifications to this software and any projects that directly include this software, must also be licensed under GPLv3.*


## Features

- **Cross-Platform Compatibility**: Windows, macOS, and Linux
- **gRPC Interface**: Robust, high-performance inter-process communication
- **Direct Sing-Box Integration**: Embeds the Sing-Box library for enhanced performance
- **Automated Ruleset Updates**: Download and update rulesets from remote URLs
- **Configurable Options**: Easily configurable through JSON files


## Installation

### Download Binaries

Download the latest release for your platform from the [Releases Page](https://github.com/ShadowZagrosDev/oblivion-helper/releases/latest).

### Build from Source

#### Prerequisites

- **Go**: https://go.dev
- **Protocol Buffers**: `protoc` compiler installed. https://grpc.io/docs/protoc-installation
- **Protobuf Plugins**: `protoc-gen-go` and `protoc-gen-go-grpc` installed. https://grpc.io/docs/languages/go/quickstart

1. Clone the repository:
   ```bash
   git clone https://github.com/ShadowZagrosDev/oblivion-helper.git
   cd oblivion-helper
   ```

2. Initialize Go modules:
   ```bash
   go mod init oblivion-helper
   go mod tidy
   ```

3. Generate Go files from the gRPC definitions:
   ```bash
   protoc --go_out=./ --go-grpc_out=./ ./proto/oblivion.proto
   ```

4. Build the project:
   ```bash
   go build -tags "with_gvisor" -ldflags "-X 'main.Version=<version>'" -o oblivion-helper ./cmd
   ```

## Configuration

### Sing-Box Configuration

Create a `sbConfig.json` file in the same directory as the Oblivion-Helper binary. Example:

```json
{
    "log": { "level": "info" },
    "inbounds": [...],
    "outbounds": [...]
}
```

- **`sbConfig.json`**: Configuration file for Sing-Box functionality.


### Export Configuration (Optional)

Create an `sbExportList.json` file in the same directory to enable dynamic ruleset updates. Example:

```json
{
    "interval": 7,
    "urls": {
        "ruleset1.srs": "https://example.com/ruleset1.srs",
        "ruleset2.srs": "https://example.com/ruleset2.srs"
    }
}
```

- `interval`: Update interval in days.
- `urls`: Rulesets to download and manage.


## Usage

Run the helper with administrative/root privileges:
```bash
sudo ./oblivion-helper
```

Command-line options:
- `version`: Display the current version and environment details.
  ```bash
  ./oblivion-helper version
  ```


### gRPC Client Interaction

The helper exposes a gRPC service with these methods:
- `Start()`: Starts the Sing-Box process using the provided configuration.
- `Stop()`: Terminates the currently running Sing-Box process.
- `StreamStatus()`: Streams real-time status updates to the client.
- `Exit()`: Shuts down the helper gracefully.


## License

This software is distributed under the **GNU General Public License version 3 (GPLv3)**. This is required because Oblivion-Helper directly embeds the **Sing-Box** library, which is also licensed under GPLv3.

You can find the full text of the GPLv3 in the [**LICENSE**](LICENSE) file in the source code repository. *Because of this license, you have the freedom to use, modify, and distribute this software, including any modifications you make, under the terms of GPLv3.*

[**View Source Code**](https://github.com/ShadowZagrosDev/oblivion-helper)


## Acknowledgments

Oblivion-Helper utilizes several open-source libraries and tools to deliver its functionality. We extend our gratitude to the developers and maintainers of these projects:

- **[Sing-Box](https://github.com/SagerNet/sing-box)**: A comprehensive library for network proxy functionalities, directly embedded in Oblivion-Helper.
- **[atomicgo/isadmin](https://github.com/atomicgo/isadmin)**: For providing a simple way to check administrative privileges in Go.
- **[fatih/color](https://github.com/fatih/color)**: For enabling colorful terminal outputs, making logs more readable.
- **[gRPC](https://grpc.io/)**: A high-performance framework for building RPC communication between processes.
  - **Submodules**:
    - `google.golang.org/grpc/codes`: For handling gRPC error codes.
    - `google.golang.org/grpc/status`: For working with gRPC status messages.

We appreciate the open-source community for providing these invaluable tools.


---

**Made with ❤️ by ShadowZagrosDev**
