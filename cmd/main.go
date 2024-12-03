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
	"sync"
	"syscall"
	"time"
	"runtime"

	pb "oblivion-helper/gRPC"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"atomicgo.dev/isadmin"
)

var Version = "dev"

const (
	defaultPort    = ":50051"
	configFileName = "config.obv"
)

type Config struct {
	SbConfig string `json:"sbConfig"`
	SbBin    string `json:"sbBin"`
}

type Server struct {
	pb.UnimplementedOblivionServiceServer
	mu           sync.Mutex
	running      bool
	statusChange chan string
	dirPath      string
	config       Config
	sbProcess    *exec.Cmd
}

func NewServer() (*Server, error) {
	execDir, err := getExecutableDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable directory: %w", err)
	}

	return &Server{
		statusChange: make(chan string, 10),
		dirPath:     execDir,
	}, nil
}

func getExecutableDir() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("error getting executable path: %w", err)
	}
	return filepath.Dir(executable), nil
}

func (s *Server) loadConfig() error {
	configPath := filepath.Join(s.dirPath, configFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}
	return json.Unmarshal(data, &s.config)
}

func (s *Server) Start(ctx context.Context, req *pb.StartRequest) (*pb.StartResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return nil, status.Error(codes.AlreadyExists, "Sing-Box is already running")
	}

	if err := s.loadConfig(); err != nil {
		return nil, status.Errorf(codes.FailedPrecondition, "Failed to load config: %v", err)
	}

	if err := s.startSingBox(); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to start Sing-Box: %v", err)
	}

	return &pb.StartResponse{}, nil
}

func (s *Server) startSingBox() error {
	sbBinPath := filepath.Join(s.dirPath, s.config.SbBin)
	sbConfigPath := filepath.Join(s.dirPath, s.config.SbConfig)

	s.sbProcess = exec.Command(sbBinPath, "run", "-c", sbConfigPath)
	if err := s.sbProcess.Start(); err != nil {
		return err
	}

	s.running = true
	s.broadcastStatus("started")
	log.Println("Sing-Box started")

	go s.monitorProcess()
	return nil
}

func (s *Server) monitorProcess() {
	log.Println("Monitoring Sing-Box process...")
	err := s.sbProcess.Wait()

	s.mu.Lock()
	defer s.mu.Unlock()

	if err != nil && (s.running || s.sbProcess != nil) {
		log.Printf("Sing-Box exited unexpectedly: %v", err)
		s.running = false
		s.sbProcess = nil
		s.broadcastStatus("stopped")
	}
}

func (s *Server) Stop(ctx context.Context, req *pb.StopRequest) (*pb.StopResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil, status.Error(codes.FailedPrecondition, "Sing-Box is not running")
	}

	if err := s.stopSingBox(); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to stop Sing-Box: %v", err)
	}

	return &pb.StopResponse{}, nil
}

func (s *Server) stopSingBox() error {
	if err := s.sbProcess.Process.Kill(); err != nil {
		return err
	}

	s.sbProcess = nil
	s.running = false
	s.broadcastStatus("stopped")
	log.Println("Sing-Box stopped")
	return nil
}

func (s *Server) Exit(ctx context.Context, req *pb.ExitRequest) (*pb.ExitResponse, error) {
	log.Println("Received exit command")
	s.broadcastStatus("exited")
	
	if s.running {
		if err := s.stopSingBox(); err != nil {
			log.Printf("Error stopping Sing-Box: %v", err)
		}
	}
	
	go func() {
		time.Sleep(1000 * time.Millisecond)
		os.Exit(0)
	}()
	
	return &pb.ExitResponse{}, nil
}

func (s *Server) Status(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := "stopped"
	if s.running {
		status = "running"
	}
	return &pb.StatusResponse{Status: status}, nil
}

func (s *Server) StreamStatus(req *pb.StatusRequest, stream pb.OblivionService_StreamStatusServer) error {
	var lastStatus string
	for status := range s.statusChange {
		if status == lastStatus {
			continue
		}
		lastStatus = status
		
		if err := stream.Send(&pb.StatusResponse{Status: status}); err != nil {
			log.Printf("Error streaming status: %v", err)
			return err
		}
	}
	return nil
}

func (s *Server) broadcastStatus(status string) {
	select {
	case s.statusChange <- status:
	default:
		log.Println("broadcastStatus: Status channel full, dropping update")
	}
}

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			fmt.Printf("Oblivion-Helper Version: %s\n", Version)
			fmt.Printf("Environment: %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		default:
			fmt.Printf("Unknown command '%s'. Use 'version' to display version information.\n", os.Args[1])
		}
		os.Exit(0)
	}
	
	if !isadmin.Check() {
		fmt.Println("Oblivion-Helper must be run as an administrator/root.")
		time.Sleep(3 * time.Second)
		os.Exit(1)
	}
	
	server, err := NewServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	lis, err := net.Listen("tcp", "127.0.0.1"+defaultPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterOblivionServiceServer(grpcServer, server)

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	go func() {
		log.Printf("Server started on port%s", defaultPort)
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	<-shutdown
	log.Println("Received termination signal, shutting down...")
	
	if server.running {
		if err := server.stopSingBox(); err != nil {
			log.Printf("Error stopping Sing-Box: %v", err)
		}
	}

	close(server.statusChange)
	grpcServer.GracefulStop()
	log.Println("Server terminated gracefully")
}
