# Oblivion-Helper 🚀

Oblivion-Helper is a robust process management tool designed to manage the Sing-Box core for the [Oblivion-Desktop](https://github.com/bepass-org/oblivion-desktop) application, providing seamless cross-platform support for Windows, macOS, and Linux.

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

## 🌟 Overview

Oblivion-Helper is a lightweight, high-performance process management utility built with Go and gRPC. It provides a reliable interface for controlling the Sing-Box core across multiple operating systems, ensuring smooth and efficient core management for the Oblivion-Desktop application.

## ✨ Features

- **Cross-Platform Support**: Works seamlessly on Windows, macOS, and Linux
- **gRPC-Powered**: Utilizes gRPC for robust, high-performance inter-process communication
- **Secure Process Management**: Provides start, stop, and status streaming capabilities
- **Configurable**: Easy configuration through a simple JSON configuration file

## 🚀 Installation

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
   go build -o oblivion-helper ./cmd/main.go
   ```

## 📝 Configuration

Create a `config.obv` file in the same directory as the Oblivion-Helper binary.

```json
{
    "sbConfig": "singbox-config.json",
    "sbBin": "sing-box"
}
```

- `sbConfig`: Name of the Sing-Box configuration file
- `sbBin`: Name of the Sing-Box binary

**Important**: 
- The `singbox-config.json` file
- The `sing-box` binary
- The `oblivion-helper` binary
- The `config.obv` file

All these files must be in the **same directory** for the helper to function correctly.

## 🔧 Usage

Run the helper with root privileges:
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
- `Start()`: Initiate the Sing-Box process
- `Stop()`: Terminate the Sing-Box process
- `StreamStatus()`: Receive real-time status updates
- `Exit()`: Gracefully exit the helper

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 License

Distributed under the MIT License. See the [LICENSE](LICENSE) for more information.

---

**Made with ❤️ by ShadowZagrosDev**
