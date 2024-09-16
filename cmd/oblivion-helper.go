package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var singBoxProcess *exec.Cmd

const commandFile = "cmd.obv"

func startSingBox() {
	if singBoxProcess == nil {
		var err error
		singBoxProcess = exec.Command("./sing-box", "run", "-c", "sbConfig.json")
		err = singBoxProcess.Start()
		if err != nil {
			fmt.Printf("Failed to start Sing-Box: %v\n", err)
			return
		}
		fmt.Println("Sing-Box started.")
	} else {
		fmt.Println("Sing-Box is already running.")
	}
}

func stopSingBox() {
	if singBoxProcess != nil {
		var err error
		if runtime.GOOS == "windows" {
			err = singBoxProcess.Process.Kill()
			if err != nil {
				fmt.Printf("Failed to kill process on Windows: %v\n", err)
			} else {
				fmt.Println("Sing-Box was stopped on Windows.")
			}
		} else {
			err = singBoxProcess.Process.Signal(syscall.SIGTERM)
			if err != nil {
				fmt.Printf("Failed to send termination signal: %v\n", err)
				if killErr := singBoxProcess.Process.Kill(); killErr != nil {
					fmt.Printf("Failed to kill process: %v\n", killErr)
				} else {
					fmt.Println("Sing-Box was forcefully stopped.")
				}
			} else {
				done := make(chan error, 1)
				go func() {
					done <- singBoxProcess.Wait()
				}()

				select {
				case <-time.After(5 * time.Second):
					if killErr := singBoxProcess.Process.Kill(); killErr != nil {
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

		singBoxProcess = nil
	} else {
		fmt.Println("No Sing-Box process to stop.")
	}
}

func handleExit() {
	stopSingBox()
	fmt.Println("Exiting helper.")
	os.Exit(0)
}

func readCommandFromFile() (string, error) {
	file, err := os.Open(commandFile)
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

func main() {
	fmt.Println("Oblivion helper started. Waiting for commands...")

	var lastCommand string

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nExiting by user request.")
		handleExit()
	}()

	for {
		command, err := readCommandFromFile()
		if err != nil {
			fmt.Printf("Error reading command: %v\n", err)
			time.Sleep(time.Second)
			continue
		}

		if command != "" && command != lastCommand {
			switch command {
			case "start":
				startSingBox()
			case "stop":
				stopSingBox()
			case "exit":
				handleExit()
			default:
				fmt.Printf("Unknown command: %s\n", command)
			}
			lastCommand = command
		}

		time.Sleep(time.Second)
	}
}
