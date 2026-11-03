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
	"github.com/alexandrmotologa/apidock/internal/diff"
	"github.com/alexandrmotologa/apidock/internal/export"
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
	mux.HandleFunc("/api/diff", s.handleDiff)
	mux.HandleFunc("/api/export", s.handleExport)
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

	// Support live streaming if requested by client
	if req.Stream || r.URL.Query().Get("stream") == "true" {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported by HTTP transport", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		flusher.Flush()

		err := s.proxyClient.ExecuteStream(r.Context(), req, proxy.StreamCallback{
			OnStart: func(statusCode int, statusText string, headers map[string]string) error {
				meta := map[string]any{
					"status":     statusCode,
					"statusText": statusText,
					"headers":    headers,
				}
				metaBytes, _ := json.Marshal(meta)
				_, _ = fmt.Fprintf(w, "event: meta\ndata: %s\n\n", string(metaBytes))
				flusher.Flush()
				return nil
			},
			OnChunk: func(chunk []byte) error {
				chunkMeta := map[string]any{
					"chunk": string(chunk),
				}
				chunkBytes, _ := json.Marshal(chunkMeta)
				_, _ = fmt.Fprintf(w, "event: chunk\ndata: %s\n\n", string(chunkBytes))
				flusher.Flush()
				return nil
			},
		})

		if err != nil {
			errMeta := map[string]string{"error": err.Error()}
			b, _ := json.Marshal(errMeta)
			_, _ = fmt.Fprintf(w, "event: error\ndata: %s\n\n", string(b))
			flusher.Flush()
		} else {
			_, _ = fmt.Fprintf(w, "event: done\ndata: {}\n\n")
			flusher.Flush()
		}
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

type DiffRequest struct {
	OldSpec string `json:"oldSpec,omitempty"`
	NewSpec string `json:"newSpec"`
}

func (s *APIServer) handleDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DiffRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON request: %v", err), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.NewSpec) == "" {
		http.Error(w, "newSpec is required", http.StatusBadRequest)
		return
	}

	var oldSpec *spec.Spec
	if strings.TrimSpace(req.OldSpec) != "" {
		parsedOld, err := s.parseSpecInput(req.OldSpec)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to parse oldSpec: %v", err), http.StatusBadRequest)
			return
		}
		oldSpec = parsedOld
	} else {
		s.mu.RLock()
		oldSpec = s.spec
		s.mu.RUnlock()
	}

	newSpec, err := s.parseSpecInput(req.NewSpec)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse newSpec: %v", err), http.StatusBadRequest)
		return
	}

	report := diff.Compare(oldSpec, newSpec)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(report)
}

func (s *APIServer) parseSpecInput(input string) (*spec.Spec, error) {
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") || strings.HasSuffix(trimmed, ".yaml") || strings.HasSuffix(trimmed, ".yml") || strings.HasSuffix(trimmed, ".json") {
		if _, err := os.Stat(trimmed); err == nil || strings.HasPrefix(trimmed, "http") {
			return s.parser.Parse(trimmed)
		}
	}
	return s.parser.ParseBytes([]byte(input))
}

func (s *APIServer) handleExport(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	s.mu.RLock()
	currentSpec := s.spec
	s.mu.RUnlock()

	switch strings.ToLower(format) {
	case "postman":
		data, err := export.ToPostmanJSON(currentSpec)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to export Postman collection: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=\"postman_collection.json\"")
		_, _ = w.Write(data)

	case "single-file", "html":
		distFS, err := GetDistFS()
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to access static assets: %v", err), http.StatusInternalServerError)
			return
		}
		htmlData, err := export.BuildSingleFileHTML(currentSpec, distFS)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to build standalone HTML: %v", err), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=\"apidock-docs.html\"")
		_, _ = w.Write(htmlData)

	default:
		http.Error(w, "Invalid format. Supported: postman, single-file", http.StatusBadRequest)
	}
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
