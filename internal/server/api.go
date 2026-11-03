package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/alexandrmotologa/apidock/internal/codegen"
	"github.com/alexandrmotologa/apidock/internal/mock"
	"github.com/alexandrmotologa/apidock/internal/proxy"
	"github.com/alexandrmotologa/apidock/internal/spec"
	"github.com/alexandrmotologa/apidock/internal/watcher"
)

// Config sets parameters for the web documentation and API server.
type Config struct {
	Port       int
	MockPort   int
	SpecPath   string
	Spec       *spec.Spec
	SpecBytes  []byte
	MockServer *mock.Server
	Watcher    *watcher.Watcher
}

// APIServer manages HTTP documentation endpoints, CORS proxy, and SSE reload triggers.
type APIServer struct {
	port        int
	mockPort    int
	specPath    string
	spec        *spec.Spec
	specBytes   []byte
	mockServer  *mock.Server
	watcher     *watcher.Watcher
	proxyClient *proxy.Client
	history     *proxy.HistoryRecorder
	validator   *spec.Validator
	parser      *spec.Parser
	mu          sync.RWMutex
	httpServer  *http.Server
}

// NewAPIServer creates a new API and doc server instance.
func NewAPIServer(cfg Config) *APIServer {
	if cfg.Port <= 0 {
		cfg.Port = 8080
	}

	return &APIServer{
		port:        cfg.Port,
		mockPort:    cfg.MockPort,
		specPath:    cfg.SpecPath,
		spec:        cfg.Spec,
		specBytes:   cfg.SpecBytes,
		mockServer:  cfg.MockServer,
		watcher:     cfg.Watcher,
		proxyClient: proxy.NewClient(),
		history:     proxy.NewHistoryRecorder(100),
		validator:   spec.NewValidator(),
		parser:      spec.NewParser(),
	}
}

// Start runs the HTTP listener on the configured port.
func (s *APIServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// API routes
	mux.HandleFunc("/api/spec", s.handleGetSpec)
	mux.HandleFunc("/api/raw", s.handleGetRaw)
	mux.HandleFunc("/api/validate", s.handleValidate)
	mux.HandleFunc("/api/proxy", s.handleProxy)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/codegen", s.handleCodegen)
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/status", s.handleStatus)

	// Embedded static web UI
	mux.HandleFunc("/", s.handleStatic)

	addr := fmt.Sprintf(":%d", s.port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// If watcher is active, listen for disk changes
	if s.watcher != nil {
		reloadCh := s.watcher.Subscribe()
		go func() {
			defer s.watcher.Unsubscribe(reloadCh)
			for {
				select {
				case <-ctx.Done():
					return
				case <-reloadCh:
					s.reloadSpec()
				}
			}
		}()
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind doc server on port %d: %w", s.port, err)
	}

	log.Printf("[APIDock Studio] Web documentation available at http://localhost:%d\n", s.port)
	return s.httpServer.Serve(listener)
}

// Stop gracefully shuts down the API server.
func (s *APIServer) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *APIServer) reloadSpec() {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("[APIServer] Reloading specification: %s", s.specPath)
	newSpec, err := s.parser.Parse(s.specPath)
	if err != nil {
		log.Printf("[APIServer Error] Failed to reload spec: %v", err)
		return
	}

	newBytes, _ := os.ReadFile(s.specPath)

	s.spec = newSpec
	s.specBytes = newBytes

	if s.mockServer != nil {
		s.mockServer.UpdateSpec(newSpec)
	}

	log.Println("[APIServer] Specification reloaded successfully")
}

func (s *APIServer) handleGetSpec(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(s.spec)
}

func (s *APIServer) handleGetRaw(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	contentType := "text/plain; charset=utf-8"
	if strings.HasSuffix(s.specPath, ".json") {
		contentType = "application/json; charset=utf-8"
	} else if strings.HasSuffix(s.specPath, ".yaml") || strings.HasSuffix(s.specPath, ".yml") {
		contentType = "application/yaml; charset=utf-8"
	}

	w.Header().Set("Content-Type", contentType)
	_, _ = w.Write(s.specBytes)
}

func (s *APIServer) handleValidate(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	specCopy := s.spec
	s.mu.RUnlock()

	diagnostics := s.validator.Validate(specCopy)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"valid":       len(diagnostics) == 0,
		"diagnostics": diagnostics,
	})
}

func (s *APIServer) handleProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req proxy.Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON body: %v", err), http.StatusBadRequest)
		return
	}

	resp, err := s.proxyClient.Execute(r.Context(), req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusBadGateway)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":   "Proxy Error",
			"message": err.Error(),
		})
		return
	}

	// Record in history
	s.history.Record(req, *resp)

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *APIServer) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodDelete {
		s.history.Clear()
		w.WriteHeader(http.StatusNoContent)
		return
	}

	entries := s.history.GetAll()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(entries)
}

type CodegenRequest struct {
	Method    string            `json:"method"`
	URL       string            `json:"url"`
	Headers   map[string]string `json:"headers"`
	Body      string            `json:"body"`
}

func (s *APIServer) handleCodegen(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CodegenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	snippets := codegen.GenerateAll(req.Method, req.URL, req.Headers, req.Body)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(snippets)
}

func (s *APIServer) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Send initial ping
	_, _ = fmt.Fprintf(w, "data: {\"type\":\"connected\"}\n\n")
	flusher.Flush()

	if s.watcher == nil {
		<-r.Context().Done()
		return
	}

	reloadCh := s.watcher.Subscribe()
	defer s.watcher.Unsubscribe(reloadCh)

	for {
		select {
		case <-r.Context().Done():
			return
		case <-reloadCh:
			_, _ = fmt.Fprintf(w, "data: {\"type\":\"reload\"}\n\n")
			flusher.Flush()
		}
	}
}

func (s *APIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"serverPort":   s.port,
		"mockPort":     s.mockPort,
		"mockActive":   s.mockServer != nil,
		"specFile":     s.specPath,
		"specTitle":    s.spec.Info.Title,
		"specVersion":  s.spec.Info.Version,
		"endpointCount": len(s.spec.Endpoints),
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Expose-Headers", "*")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
