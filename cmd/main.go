package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"atomicgo.dev/isadmin" 			 // Checks if the program is run with administrative privileges
	"github.com/fatih/color" 		 // Adds color to log messages
	"google.golang.org/grpc"		 // gRPC framework
	"google.golang.org/grpc/codes"   	 // gRPC error codes
	"google.golang.org/grpc/status"  	 // gRPC status handling
	pb "oblivion-helper/gRPC" 		 // Generated protobuf code for the gRPC service
)

// Constants for server setup and configuration
const (
	protocolType       = "tcp"                 // Connection protocol used by the server
	serverAddress      = "127.0.0.1:50051"     // Localhost address for gRPC server
	configFileName     = "config.obv"          // Name of the configuration file
	statusChannelCap   = 100                   // Capacity of the status channel
	gracefulShutdownTimeout = 5 * time.Second  // Timeout for graceful shutdown
)

// Global variable for version
var Version = "dev"

// Logger wraps multiple loggers with different levels (info, warn, error, fatal)
type Logger struct {
	info, warn, error, fatal *log.Logger
}

// NewLogger initializes a Logger instance with colored prefixes
func NewLogger() *Logger {
	return &Logger{
		info:  log.New(os.Stdout, color.GreenString("[INFO] "), log.Ldate|log.Ltime|log.Lmsgprefix),
		warn:  log.New(os.Stdout, color.YellowString("[WARN] "), log.Ldate|log.Ltime|log.Lmsgprefix),
		error: log.New(os.Stderr, color.RedString("[ERROR] "), log.Ldate|log.Ltime|log.Lmsgprefix),
		fatal: log.New(os.Stderr, color.New(color.FgRed, color.Bold).Sprint("[FATAL] "), log.Ldate|log.Ltime|log.Lmsgprefix),
	}
}

// Config represents the configuration for Sing-Box
type Config struct {
	SbConfig string `json:"sbConfig"` // Path to Sing-Box configuration file
	SbBin    string `json:"sbBin"`    // Path to Sing-Box binary
}

// Server is the main gRPC server implementation
type Server struct {
	pb.UnimplementedOblivionServiceServer
	mu           sync.RWMutex   // Synchronizes access to server state
	running      bool           // Indicates if Sing-Box is running
	statusChange chan string    // Channel to broadcast status updates
	dirPath      string         // Directory path of the executable
	config       Config         // Configuration loaded from the file
	sbProcess    *exec.Cmd      // Sing-Box process handler
	logger       *Logger        // Logger for server messages
}

// NewServer creates and initializes a new Server instance
func NewServer(logger *Logger) (*Server, error) {
	execDir, err := getExecutableDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable directory: %w", err)
	}

	return &Server{
		statusChange: make(chan string, statusChannelCap),
		dirPath:      execDir,
		logger:       logger,
	}, nil
}

// getExecutableDir returns the directory of the current executable
func getExecutableDir() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(executable), nil
}

// loadConfig loads the Sing-Box configuration from a file
func (s *Server) loadConfig() error {
	configPath := filepath.Join(s.dirPath, configFileName)

	// Check if the configuration file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return status.Errorf(codes.NotFound, "configuration file not found at %s", configPath)
	}

	// Read and parse the configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to read config file: %v", err)
	}

	if err := json.Unmarshal(data, &s.config); err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to parse config: %v", err)
	}
	return nil
}

// startSingBox starts the Sing-Box process
func (s *Server) startSingBox() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if Sing-Box is already running
	if s.running {
		return status.Errorf(codes.AlreadyExists, "Sing-Box is already running")
	}

	if err := s.loadConfig(); err != nil {
		return err
	}

	// Prepare paths for binary and configuration
	sbBinPath := filepath.Join(s.dirPath, s.config.SbBin)
	sbConfigPath := filepath.Join(s.dirPath, s.config.SbConfig)

	// Verify existence of the binary and configuration
	if _, err := os.Stat(sbBinPath); os.IsNotExist(err) {
		return status.Errorf(codes.NotFound, "Sing-Box binary not found at %s", sbBinPath)
	}
	if _, err := os.Stat(sbConfigPath); os.IsNotExist(err) {
		return status.Errorf(codes.NotFound, "Sing-Box config not found at %s", sbConfigPath)
	}

	// Start the Sing-Box process
	s.sbProcess = exec.Command(sbBinPath, "run", "-c", sbConfigPath)
	if err := s.sbProcess.Start(); err != nil {
		return status.Errorf(codes.Internal, "failed to start Sing-Box: %v", err)
	}

	s.running = true
	s.broadcastStatus("started")
	s.logger.info.Println("Sing-Box started")

	// Monitor the process in a separate goroutine
	go s.monitorProcess()
	return nil
}

// monitorProcess monitors the Sing-Box process for termination
func (s *Server) monitorProcess() {
	s.logger.info.Println("Monitoring Sing-Box process...")
	err := s.sbProcess.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Handle unexpected termination
	if err != nil && (s.running || s.sbProcess != nil) {
		s.logger.error.Printf("Sing-Box exited unexpectedly: %v", err)
		s.running = false
		s.sbProcess = nil
		s.broadcastStatus("terminated")
	}
}

// stopSingBox stops the Sing-Box process
func (s *Server) stopSingBox() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if Sing-Box is running
	if !s.running {
		return status.Errorf(codes.FailedPrecondition, "Sing-Box is not running")
	}

	// Terminate the process
	if err := s.sbProcess.Process.Kill(); err != nil {
		return status.Errorf(codes.Internal, "failed to stop Sing-Box: %v", err)
	}

	s.sbProcess = nil
	s.running = false
	s.broadcastStatus("stopped")
	s.logger.info.Println("Sing-Box stopped")
	return nil
}

// Start handles the gRPC Start request to initiate Sing-Box
func (s *Server) Start(ctx context.Context, req *pb.StartRequest) (*pb.StartResponse, error) {
	if err := s.startSingBox(); err != nil {
		s.logger.error.Printf("Start error: %v", err)
		return nil, err
	}
	return &pb.StartResponse{Message: "Sing-Box started successfully."}, nil
}

// Stop handles the gRPC Stop request to terminate Sing-Box
func (s *Server) Stop(ctx context.Context, req *pb.StopRequest) (*pb.StopResponse, error) {
	if err := s.stopSingBox(); err != nil {
		s.logger.error.Printf("Stop error: %v", err)
		return nil, err
	}
	return &pb.StopResponse{Message: "Sing-Box stopped successfully."}, nil
}

// Exit handles the gRPC Exit request to shut down the service gracefully
func (s *Server) Exit(ctx context.Context, req *pb.ExitRequest) (*pb.ExitResponse, error) {
	s.logger.info.Println("Exiting Oblivion-Helper...")

	// Stop Sing-Box if it is running
	if s.running {
		if err := s.stopSingBox(); err != nil {
			s.logger.error.Printf("Exit stop error: %v", err)
		}
	}

	// Schedule a delayed exit to allow response to be sent
	go func() {
		time.Sleep(gracefulShutdownTimeout)
		os.Exit(0)
	}()

	return &pb.ExitResponse{}, nil
}

// StreamStatus streams the current status of Sing-Box to the client
func (s *Server) StreamStatus(req *pb.StatusRequest, stream pb.OblivionService_StreamStatusServer) error {
	var lastStatus string
	for {
		select {
		case <-stream.Context().Done(): // Handle client disconnection
			s.logger.warn.Println("Stream closed by client")
			if s.running {
				// Stop Sing-Box if it is still running
				if err := s.stopSingBox(); err != nil {
					s.logger.error.Printf("Stream stop error: %v", err)
					return status.Errorf(codes.Aborted, "failed to stop service during stream closure: %v", err)
				}
			}
			return stream.Context().Err()

		case status, ok := <-s.statusChange: // Receive status updates
			if !ok {
				s.logger.warn.Println("Status channel closed")
				return nil // The status channel was closed
			}

			if status == lastStatus { // Avoid sending duplicate status updates
				continue
			}
			lastStatus = status

			if err := stream.Send(&pb.StatusResponse{Status: status}); err != nil {
				s.logger.error.Printf("Status stream error: %v", err)
				return err // Failed to send status update
			}
		}
	}
}

// broadcastStatus sends a status update to the status channel
func (s *Server) broadcastStatus(status string) {
	select {
	case s.statusChange <- status: // Send status if the channel is not full
	default: // Log a warning if the channel is full
		s.logger.warn.Println("Status channel full, dropping update")
	}
}

// main initializes the logger, checks admin privileges, creates the server, and starts the gRPC server
func main() {
	logger := NewLogger()

	// Handle any command-line arguments
	handleCommandLineArgs(logger)

	// Ensure the program is running as an administrator/root
	if !isadmin.Check() {
		logger.fatal.Fatal("Oblivion-Helper must be run as an administrator/root.")
	}

	// Create a new Server instance
	server, err := NewServer(logger)
	if err != nil {
		logger.fatal.Fatalf("Failed to create server: %v", err)
	}

	// Start the gRPC server
	startGRPCServer(server, logger)
}

// handleCommandLineArgs processes command-line arguments like "version"
func handleCommandLineArgs(logger *Logger) {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			logger.info.Printf("Oblivion-Helper Version: %s\n", Version)
			logger.info.Printf("Environment: %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		default:
			logger.warn.Printf("Unknown command '%s'.\nUse 'version' to display version information.\n", os.Args[1])
		}
		os.Exit(0)
	}
}

// startGRPCServer starts the gRPC server and handles termination signals
func startGRPCServer(server *Server, logger *Logger) {
	lis, err := net.Listen(protocolType, serverAddress) // Listen on the specified address and port
	if err != nil {
		logger.fatal.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer() // Create a new gRPC server
	pb.RegisterOblivionServiceServer(grpcServer, server) // Register the server implementation

	// Handle OS signals for graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Start serving gRPC requests in a separate goroutine
	go func() {
		logger.info.Printf("Server started on: %s", serverAddress)
		if err := grpcServer.Serve(lis); err != nil {
			logger.fatal.Fatalf("Failed to serve: %v", err)
		}
	}()

	// Wait for a termination signal
	<-shutdown
	logger.warn.Println("Received termination signal, shutting down...")

	// Perform cleanup
	if server.running {
		if err := server.stopSingBox(); err != nil {
			logger.error.Printf("Shutdown stop error: %v", err)
		}
	}

	// Close the status channel and gracefully stop the gRPC server
	close(server.statusChange)
	grpcServer.GracefulStop()
	logger.info.Println("Server terminated gracefully")
}
