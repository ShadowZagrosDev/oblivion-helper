# Oblivion-Helper

Oblivion-Helper is a process management tool designed to manage and monitor the Sing-Box service in the **Oblivion-Desktop** application. It helps ensure that Sing-Box, along with Warp-Plus and Oblivion processes, runs reliably by monitoring and restarting them when necessary.

## Features

- **Start and stop Sing-Box**: Easily control the Sing-Box service.
- **Monitor processes**: Automatically check if Sing-Box, Warp-Plus, and Oblivion are running and restart them if needed.
- **Command-based control**: Accepts commands like `start`, `stop`, and `exit` from a command file to manage the services.
- **Process logging**: Logs events such as process starts, stops, and errors for easy debugging and monitoring.
  
## Installation

1. Ensure that Go is installed on your system. If not, install it by following the instructions at [https://golang.org/doc/install](https://golang.org/doc/install).

2. Clone the repository:
   ```bash
   git clone https://github.com/ShadowZagrosDev/oblivion-helper.git
   ```
   
3. Navigate to the project directory:
   ```bash
   cd oblivion-helper
   ```

4. Build the project:
   ```bash
   go build -o oblivion-helper
   ```

5. Ensure that you have the proper configuration files (`config.obv`, `cmd.obv`) in the same directory as the executable.

## Usage

- **Start the helper**:
  ```bash
  ./oblivion-helper
  ```

- **Command File**: Use the `cmd.obv` file to send commands to the Oblivion-Helper. Supported commands include:
  - `start`: Start Sing-Box and related processes.
  - `stop`: Stop Sing-Box.
  - `exit`: Safely shut down Oblivion-Helper.

## Configuration

Oblivion-Helper reads its configuration from a `config.obv` file, which should be a JSON file structured like this:

```json
{
  "sbConfig": "sing-box_config.json",
  "sbBin": "sing-box_binary",
  "wpBin": "warp-plus_binary",
  "obBin": "oblivion-desktop_binary",
  "monitorWp": true,
  "monitorOb": true
}
```

- `sbConfig`: Name of Sing-Box's configuration file.
- `sbBin`: Name of the Sing-Box binary.
- `wpBin`: Name of the Warp-Plus binary.
- `obBin`: Name of the Oblivion-Desktop binary.
- `monitorWp`: Enable or disable monitoring for Warp-Plus.
- `monitorOb`: Enable or disable monitoring for Oblivion-Desktop.

Place the `config.obv` and `cmd.obv` files in the same directory as the `oblivion-helper` binary.

## Logs

The helper writes logs to `oblivion-helper.log` to capture key events and errors. You can check this file to troubleshoot or verify the status of processes.

## License

This project is licensed under the MIT License.
