package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Config struct {
	SbConfig   string `json:"sbConfig"`
	SbBin      string `json:"sbBin"`
	WpBin      string `json:"wpBin"`
	ObBin      string `json:"obBin"`
	MonitorWp  bool   `json:"monitorWp"`
	MonitorOb  bool   `json:"monitorOb"`
}

type SingBoxManager struct {
	config         Config
	configFile     string
	commandFile    string
	singBoxProcess *exec.Cmd
	mu             sync.Mutex
}

func init() {
	logFile, err := os.OpenFile("oblivion-helper.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		fmt.Printf("Error initializing log file: %v\n", err)
		os.Exit(1)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime)
	log.Println("Logging initialized.")
}

func getExecutableDir() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("error getting executable path: %w", err)
	}
	return filepath.Dir(executable), nil
}

func (m *SingBoxManager) loadConfig() error {
	execDir, err := getExecutableDir()
	if err != nil {
		return fmt.Errorf("failed to get executable directory: %w", err)
	}

	byteValue, err := os.ReadFile(m.configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(byteValue, &m.config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if !filepath.IsAbs(m.config.SbBin) {
		m.config.SbBin = filepath.Join(execDir, m.config.SbBin)
	}
	if !filepath.IsAbs(m.config.SbConfig) {
		m.config.SbConfig = filepath.Join(execDir, m.config.SbConfig)
	}

	return nil
}

func (m *SingBoxManager) startSingBox() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.singBoxProcess != nil && m.singBoxProcess.ProcessState == nil {
		log.Println("Sing-Box is already running.")
		return nil
	}

	if err := m.loadConfig(); err != nil {
		log.Printf("failed to load config: %v\n", err)
		return nil
	}

	m.singBoxProcess = exec.Command(m.config.SbBin, "run", "-c", m.config.SbConfig)
	if err := m.singBoxProcess.Start(); err != nil {
		return fmt.Errorf("failed to start Sing-Box: %w", err)
	}
	log.Println("Sing-Box started.")

	if m.config.MonitorWp {
		go m.monitorProcess(m.config.WpBin, func() {
			log.Println("Warp-Plus process not found. Stopping Sing-Box...")
			if err := m.stopSingBox(); err != nil {
				log.Printf("Error stopping Sing-Box: %v\n", err)
			}
		})
	}

	if m.config.MonitorOb {
		go m.monitorProcess(m.config.ObBin, func() {
			log.Println("Oblivion-Desktop process not found. Stopping Oblivion-Helper...")
			m.handleExit()
		})
	}

	return nil
}

func (m *SingBoxManager) stopSingBox() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.singBoxProcess == nil {
		log.Println("No Sing-Box process to stop.")
		return nil
	}

	var err error
	if runtime.GOOS == "windows" {
		err = m.singBoxProcess.Process.Kill()
	} else {
		err = m.singBoxProcess.Process.Signal(syscall.SIGTERM)
		if err == nil {
			done := make(chan error, 1)
			go func() {
				done <- m.singBoxProcess.Wait()
			}()

			select {
			case <-time.After(5 * time.Second):
				err = m.singBoxProcess.Process.Kill()
			case err = <-done:
			}
		}
	}

	if err != nil {
		return fmt.Errorf("failed to stop Sing-Box: %w", err)
	}

	log.Println("Sing-Box stopped.")
	m.singBoxProcess = nil
	return nil
}

func (m *SingBoxManager) isProcessRunning(processName string) bool {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("powershell", "-Command", fmt.Sprintf("Get-Process -Name %s -ErrorAction SilentlyContinue", processName))
	case "darwin", "linux":
		cmd = exec.Command("pgrep", processName)
	default:
		log.Printf("Unsupported operating system: %s\n", runtime.GOOS)
		return false
	}

	output, err := cmd.Output()
	return err == nil && len(output) > 0
}

func (m *SingBoxManager) monitorProcess(processName string, stopCallback func()) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if !m.isProcessRunning(processName) {
				stopCallback()
				return
			}
		}
	}
}

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

func (m *SingBoxManager) processCommand(command string) {
	switch command {
	case "start":
		if err := m.startSingBox(); err != nil {
			log.Printf("Error starting Sing-Box: %v\n", err)
		}
	case "stop":
		if err := m.stopSingBox(); err != nil {
			log.Printf("Error stopping Sing-Box: %v\n", err)
		}
	case "exit":
		m.handleExit()
	default:
		log.Printf("Unknown command: %s\n", command)
	}
}

func (m *SingBoxManager) watchCommandFile(commandChan chan<- string) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastModTime time.Time
	for {
		select {
		case <-ticker.C:
			fileInfo, err := os.Stat(m.commandFile)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				log.Printf("Error getting file info: %v\n", err)
				continue
			}
			if fileInfo.ModTime().After(lastModTime) {
				command, err := m.readCommandFromFile()
				if err != nil {
					log.Printf("Error reading command: %v\n", err)
					continue
				}
				if command != "" {
					commandChan <- command
					lastModTime = fileInfo.ModTime()
				}
			}
		}
	}
}

func (m *SingBoxManager) shouldKillWarpPlus() bool {
	if !m.isProcessRunning("warp-plus") {
		return false
	}

	execDir, err := getExecutableDir()
	if err != nil {
		return false
	}

	settingsFile:= filepath.Join(execDir, "settings.json")

	content, err := os.ReadFile(settingsFile)
	if err != nil {
		return false
	}

	if strings.Contains(string(content), `"proxyMode":"tun"`) {
		log.Println("Proxy mode is set to TUN and warp-plus is running. Proceeding to kill warp-plus.")
		return true
	}

	return false
}

func (m *SingBoxManager) killWarpPlus() error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("taskkill", "/F", "/IM", fmt.Sprintf("%s.exe", m.config.WpBin))
	case "darwin", "linux":
		cmd = exec.Command("pkill", m.config.WpBin)
	default:
		log.Printf("Unsupported operating system: %s\n", runtime.GOOS)
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	if err := cmd.Run(); err != nil {
		log.Printf("Failed to kill warp-plus process: %v\n", err)
		return err
	}

	log.Println("Warp-Plus process killed successfully.")
	return nil
}


func (m *SingBoxManager) handleExit() {
	if err := m.stopSingBox(); err != nil {
		log.Printf("Error stopping Sing-Box during exit: %v\n", err)
	}
	log.Println("Exiting helper.")
	os.Exit(0)
}

func main() {
	execDir, err := getExecutableDir()
	if err != nil {
		log.Println(err)
		return
	}

	manager := &SingBoxManager{
		commandFile: filepath.Join(execDir, "cmd.obv"),
		configFile:  filepath.Join(execDir, "config.obv"),
	}

	log.Println("Oblivion helper started. Waiting for commands...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Exiting by user request.")
		if manager.shouldKillWarpPlus() {
			if err := manager.killWarpPlus(); err != nil {
				log.Printf("Error killing warp-plus during exit: %v\n", err)
			}
		}
		manager.handleExit()
	}()

	commandChan := make(chan string)
	go manager.watchCommandFile(commandChan)

	for command := range commandChan {
		manager.processCommand(command)
	}
}
