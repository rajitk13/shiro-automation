package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/rkuthiala/shiro-automation/internal/modules"
)

// ModuleServer implements the HTTP module API
type ModuleServer struct {
	port int
}

// NewModuleServer creates a new module server
func NewModuleServer(port int) *ModuleServer {
	return &ModuleServer{port: port}
}

// handleExecute handles the /execute endpoint
func (s *ModuleServer) handleExecute(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req modules.ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := s.executeModule(req)
	if err != nil {
		resp := modules.ExecuteResponse{
			Success: false,
			Error:   err.Error(),
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	json.NewEncoder(w).Encode(result)
}

// handleMetadata handles the /metadata endpoint
func (s *ModuleServer) handleMetadata(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	metadata := modules.MetadataResponse{
		Name:        "hello-module",
		Description: "A simple hello world module example",
		Version:     "1.0.0",
		InputSchema: map[string]modules.SchemaField{
			"name": {
				Type:        "string",
				Description: "Name to greet",
				Required:    false,
			},
		},
		OutputSchema: map[string]modules.SchemaField{
			"message": {
				Type:        "string",
				Description: "Greeting message",
				Required:    true,
			},
		},
	}
	json.NewEncoder(w).Encode(metadata)
}

// handleHealth handles the /health endpoint
func (s *ModuleServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	resp := modules.HealthResponse{
		Healthy: true,
		Message: "Module is healthy",
	}
	json.NewEncoder(w).Encode(resp)
}

// executeModule implements the actual module logic
func (s *ModuleServer) executeModule(req modules.ExecuteRequest) (modules.ExecuteResponse, error) {
	// Get name from config, default to "World"
	name := "World"
	if n, ok := req.Config["name"].(string); ok {
		name = n
	}

	return modules.ExecuteResponse{
		Success: true,
		Output: map[string]interface{}{
			"message": fmt.Sprintf("Hello, %s!", name),
		},
	}, nil
}

// Start starts the HTTP server
func (s *ModuleServer) Start() error {
	http.HandleFunc("/execute", s.handleExecute)
	http.HandleFunc("/metadata", s.handleMetadata)
	http.HandleFunc("/health", s.handleHealth)

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Starting module server on %s", addr)
	return http.ListenAndServe(addr, nil)
}

func main() {
	port := 8080
	if portStr := os.Getenv("MODULE_PORT"); portStr != "" {
		fmt.Sscanf(portStr, "%d", &port)
	}

	server := NewModuleServer(port)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
