package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

// SingBoxManager handles Sing-Box process management
type SingBoxManager struct {
	commandFile   string
	configFile    string
	binFile       string
	singBoxProcess *exec.Cmd
}

// getExecutableDir returns the directory of the current executable
func getExecutableDir() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("Error getting executable path: %w", err)
	}
	return filepath.Dir(executable), nil
}

// startSingBox starts the Sing-Box process
func (m *SingBoxManager) startSingBox() {
	if m.singBoxProcess == nil {
		m.singBoxProcess = exec.Command(m.binFile, "run", "-c", m.configFile)
		err := m.singBoxProcess.Start()
		if err != nil {
			fmt.Printf("Failed to start Sing-Box: %v\n", err)
			return
		}
		fmt.Println("Sing-Box started.")
	} else {
		fmt.Println("Sing-Box is already running.")
	}
}

// stopSingBox stops the Sing-Box process gracefully, with a forced timeout fallback
func (m *SingBoxManager) stopSingBox() {
	if m.singBoxProcess != nil {
		var err error
		if runtime.GOOS == "windows" {
			err = m.singBoxProcess.Process.Kill()
			if err != nil {
				fmt.Printf("Failed to kill process on Windows: %v\n", err)
			} else {
				fmt.Println("Sing-Box was stopped on Windows.")
			}
		} else {
			err = m.singBoxProcess.Process.Signal(syscall.SIGTERM)
			if err != nil {
				fmt.Printf("Failed to send termination signal: %v\n", err)
				if killErr := m.singBoxProcess.Process.Kill(); killErr != nil {
					fmt.Printf("Failed to kill process: %v\n", killErr)
				} else {
					fmt.Println("Sing-Box was forcefully stopped.")
				}
			} else {
				done := make(chan error, 1)
				go func() {
					done <- m.singBoxProcess.Wait()
				}()

				select {
				case <-time.After(5 * time.Second):
					if killErr := m.singBoxProcess.Process.Kill(); killErr != nil {
						fmt.Printf("Failed to kill process after timeout: %v\n", killErr)
					} else {
						fmt.Println("Sing-Box was forcefully stopped after timeout.")
					}
				case err := <-done:
					if err != nil {
						fmt.Printf("Process exited with error: %v\n", err)
					} else {
						fmt.Println("Sing-Box stopped gracefully.")
					}
				}
			}
		}
		m.singBoxProcess = nil
	} else {
		fmt.Println("No Sing-Box process to stop.")
	}
}

// handleExit stops the Sing-Box and exits the program
func handleExit(m *SingBoxManager) {
	m.stopSingBox()
	fmt.Println("Exiting helper.")
	os.Exit(0)
}

// readCommandFromFile reads the command from the provided file
func (m *SingBoxManager) readCommandFromFile() (string, error) {
	file, err := os.Open(m.commandFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		return strings.ToLower(strings.TrimSpace(scanner.Text())), nil
	}
	return "", scanner.Err()
}

// processCommand processes the received command and takes appropriate actions
func (m *SingBoxManager) processCommand(command string) {
	switch command {
	case "start":
		m.startSingBox()
	case "stop":
		m.stopSingBox()
	case "exit":
		handleExit(m)
	default:
		fmt.Printf("Unknown command: %s\n", command)
	}
}

// watchCommandFile watches the command file for changes and sends new commands to a channel
func (m *SingBoxManager) watchCommandFile(commandChan chan string) {
	for {
		command, err := m.readCommandFromFile()
		if err != nil {
			fmt.Printf("Error reading command: %v\n", err)
			time.Sleep(time.Second) // Avoid busy-waiting
			continue
		}
		if command != "" {
			commandChan <- command
		}
		time.Sleep(time.Second)
	}
}

func main() {
	// Get the directory of the executable
	execDir, err := getExecutableDir()
	if err != nil {
		fmt.Println(err)
		return
	}

	// Initialize SingBoxManager with relevant file paths
	manager := &SingBoxManager{
		commandFile: filepath.Join(execDir, "cmd.obv"),
		configFile:  filepath.Join(execDir, "sbConfig.json"),
	}

	// Set the correct binary path based on the OS
	if runtime.GOOS == "windows" {
		manager.binFile = filepath.Join(execDir, "oblivion-sing-box.exe")
	} else {
		manager.binFile = filepath.Join(execDir, "oblivion-sing-box")
	}

	fmt.Println("Oblivion helper started. Waiting for commands...")

	// Signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nExiting by user request.")
		handleExit(manager)
	}()

	// Command file watcher
	commandChan := make(chan string)
	go manager.watchCommandFile(commandChan)

	var lastCommand string
	for {
		select {
		case command := <-commandChan:
			if command != lastCommand {
				manager.processCommand(command)
				lastCommand = command
			}
		}
	}
}
