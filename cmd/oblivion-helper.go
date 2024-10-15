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
	
	"github.com/shirou/gopsutil/process"
)

const (
	InfoLevel    = "INFO"
	WarningLevel = "WARNING"
	ErrorLevel   = "ERROR"
)

type Config struct {
	SbConfig  string `json:"sbConfig"`
	SbBin     string `json:"sbBin"`
	WpBin     string `json:"wpBin"`
	ObBin     string `json:"obBin"`
	MonitorWp bool   `json:"monitorWp"`
	MonitorOb bool   `json:"monitorOb"`
}

type SingBoxManager struct {
	config         Config
	configFile     string
	commandFile    string
	singBoxProcess *exec.Cmd
	mu             sync.Mutex
	dirPath        string
}

func logMessage(level, function, message string) {
	log.Printf("[%s] %s: %s\n", level, function, message)
}

func init() {
	logFile, err := os.OpenFile("oblivion-helper.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		fmt.Printf("Error initializing log file: %v\n", err)
		os.Exit(1)
	}
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime)
	logMessage(InfoLevel, "init", "-----------< Logging started >-----------")
}

func getExecutableDir() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("error getting executable path: %w", err)
	}
	return filepath.Dir(executable), nil
}

func (m *SingBoxManager) loadConfig() error {
	byteValue, err := os.ReadFile(m.configFile)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	return json.Unmarshal(byteValue, &m.config)
}

func (m *SingBoxManager) startSingBox() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.singBoxProcess != nil {
		logMessage(WarningLevel, "startSingBox", "Sing-Box is already running")
		return
	}

	if err := m.loadConfig(); err != nil {
		logMessage(ErrorLevel, "startSingBox", fmt.Sprintf("Failed to load config: %v", err))
		return
	}

	sbBinFullPath := filepath.Join(m.dirPath, m.config.SbBin)
	sbConfigFullPath := filepath.Join(m.dirPath, m.config.SbConfig)

	m.singBoxProcess = exec.Command(sbBinFullPath, "run", "-c", sbConfigFullPath)
	if err := m.singBoxProcess.Start(); err != nil {
		logMessage(ErrorLevel, "startSingBox", fmt.Sprintf("Failed to start Sing-Box: %v", err))
		return
	}
	logMessage(InfoLevel, "startSingBox", "Sing-Box started")

	go m.monitorProcess(m.config.SbBin, m.monitorSingBox)
	if m.config.MonitorWp {
		go m.monitorProcess(m.config.WpBin, m.monitorWarpPlus)
	}
	if m.config.MonitorOb {
		go m.monitorProcess(m.config.ObBin, m.monitorOblivionHelper)
	}
}

func (m *SingBoxManager) stopSingBox() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.singBoxProcess == nil {
		logMessage(WarningLevel, "stopSingBox", "No Sing-Box process to stop")
		return
	}

	err := m.singBoxProcess.Process.Kill()
	if err != nil {
		logMessage(ErrorLevel, "stopSingBox", fmt.Sprintf("Failed to stop Sing-Box: %v", err))
		return
	}

	logMessage(InfoLevel, "stopSingBox", "Sing-Box stopped")
	m.singBoxProcess = nil
}

func (m *SingBoxManager) isProcessRunning(processName string) bool {
	switch runtime.GOOS {
	case "windows":
		processes, err := process.Processes()
		if err != nil {
			logMessage(ErrorLevel, "isProcessRunning", fmt.Sprintf("Failed to get processes: %v", err))
			return false
		}
	
		for _, process := range processes {
			name, err := process.Name()
			if err != nil {
				continue
			}
			if name == processName {
				return true
			}
		}
		return false
	case "darwin", "linux":
		var cmd *exec.Cmd
		cmd = exec.Command("pgrep", "-f", processName)
		output, err := cmd.Output()
		return err == nil && len(output) > 0
	default:
		logMessage(WarningLevel, "isProcessRunning", fmt.Sprintf("Unsupported operating system: %s", runtime.GOOS))
		return false
	}
}

func (m *SingBoxManager) monitorProcess(processName string, callback func()) {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		if !m.isProcessRunning(processName) {
			callback()
			return
		}
	}
}

func (m *SingBoxManager) monitorSingBox() {
	if m.isProcessRunning(m.config.WpBin) {
		logMessage(WarningLevel, "monitorSingBox", "Sing-Box process not found. Stopping Warp-Plus...")
		m.killWarpPlus()
	}
	m.singBoxProcess = nil
}

func (m *SingBoxManager) monitorWarpPlus() {
	if m.isProcessRunning(m.config.SbBin) {
		logMessage(WarningLevel, "monitorWarpPlus", "Warp-Plus process not found. Stopping Sing-Box...")
		m.stopSingBox()
	}
}

func (m *SingBoxManager) monitorOblivionHelper() {
	logMessage(WarningLevel, "monitorOblivionHelper", "Oblivion-Desktop process not found. Stopping Oblivion-Helper...")
	m.handleExit()
}

func (m *SingBoxManager) processCommand(command string) {
	logMessage(InfoLevel, "processCommand", fmt.Sprintf("Processing command: %s", command))
	switch command {
	case "start":
		m.startSingBox()
	case "stop":
		m.stopSingBox()
	case "exit":
		m.handleExit()
	default:
		logMessage(WarningLevel, "processCommand", fmt.Sprintf("Unknown command: %s", command))
	}
}

func (m *SingBoxManager) watchCommandFile(commandChan chan<- string) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var lastModTime time.Time
	for range ticker.C {
		fileInfo, err := os.Stat(m.commandFile)
		if err != nil {
			if !os.IsNotExist(err) {
				logMessage(ErrorLevel, "watchCommandFile", fmt.Sprintf("Error getting file info: %v", err))
			}
			continue
		}
		if fileInfo.ModTime().After(lastModTime) {
			if command, err := m.readCommandFromFile(); err == nil && command != "" {
				commandChan <- command
				lastModTime = fileInfo.ModTime()
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

func (m *SingBoxManager) shouldKillWarpPlus() bool {
	if !m.isProcessRunning(m.config.WpBin) {
		return false
	}

	settingsFile := filepath.Join(m.dirPath, "settings.json")
	content, err := os.ReadFile(settingsFile)
	if err != nil {
		logMessage(ErrorLevel, "shouldKillWarpPlus", fmt.Sprintf("Error reading settings.json: %v", err))
		return false
	}

	return strings.Contains(string(content), `"proxyMode":"tun"`)
}

func (m *SingBoxManager) killWarpPlus() {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("taskkill", "/F", "/IM", fmt.Sprintf("%s.exe", m.config.WpBin))
	case "darwin", "linux":
		cmd = exec.Command("pkill", m.config.WpBin)
	default:
		logMessage(ErrorLevel, "killWarpPlus", fmt.Sprintf("Unsupported OS: %s", runtime.GOOS))
		return
	}

	if err := cmd.Run(); err != nil {
		logMessage(ErrorLevel, "killWarpPlus", fmt.Sprintf("Failed to kill Warp-Plus process: %v", err))
		return
	}

	logMessage(InfoLevel, "killWarpPlus", "Warp-Plus process killed successfully")
}

func (m *SingBoxManager) handleExit() {
	if m.isProcessRunning(m.config.SbBin) {
        m.stopSingBox()
    }
	if m.shouldKillWarpPlus() {
		m.killWarpPlus()
	}
	logMessage(InfoLevel, "handleExit", "Exiting helper")
	os.Exit(0)
}

func main() {
	execDir, err := getExecutableDir()
	if err != nil {
		logMessage(ErrorLevel, "main", fmt.Sprintf("Failed to get executable directory: %v", err))
		os.Exit(0)
	}

	manager := &SingBoxManager{
		commandFile: filepath.Join(execDir, "cmd.obv"),
		configFile:  filepath.Join(execDir, "config.obv"),
		dirPath:     execDir,
	}

	fmt.Println("Oblivion helper started. Waiting for commands...")
	logMessage(InfoLevel, "main", "Oblivion helper started. Waiting for commands...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		logMessage(WarningLevel, "main", "Exiting by an interrupt")
		manager.handleExit()
	}()

	commandChan := make(chan string)
	go manager.watchCommandFile(commandChan)

	for command := range commandChan {
		manager.processCommand(command)
	}
}
