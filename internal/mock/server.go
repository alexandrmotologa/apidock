package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/alexandrmotologa/apidock/internal/spec"
)

var preferCodeRegex = regexp.MustCompile(`code=([0-9]{3})`)
var preferExampleRegex = regexp.MustCompile(`example=([a-zA-Z0-9_-]+)`)

// Server runs an OpenAPI schema-driven mock HTTP server.
type Server struct {
	port           int
	spec           *spec.Spec
	matcher        *Matcher
	generator      *Generator
	defaultLatency time.Duration
	httpServer     *http.Server
}

// Config defines the configuration for creating a mock server.
type Config struct {
	Port           int
	DefaultLatency time.Duration
	Spec           *spec.Spec
}

// NewServer initializes a new mock HTTP server.
func NewServer(cfg Config) *Server {
	if cfg.Port <= 0 {
		cfg.Port = 4010
	}

	return &Server{
		port:           cfg.Port,
		spec:           cfg.Spec,
		matcher:        NewMatcher(cfg.Spec.Endpoints),
		generator:      NewGenerator(cfg.Spec),
		defaultLatency: cfg.DefaultLatency,
	}
}

// UpdateSpec updates the active spec and re-initializes route matching.
func (s *Server) UpdateSpec(newSpec *spec.Spec) {
	s.spec = newSpec
	s.matcher = NewMatcher(newSpec.Endpoints)
	s.generator = NewGenerator(newSpec)
}

// Start runs the mock HTTP server on its configured port.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRequest)

	addr := fmt.Sprintf(":%d", s.port)
	s.httpServer = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to bind mock server to port %d: %w", s.port, err)
	}

	log.Printf("[Mock Server] Listening on http://localhost:%d\n", s.port)
	return s.httpServer.Serve(listener)
}

// Stop gracefully shuts down the mock server.
func (s *Server) Stop(ctx context.Context) error {
	if s.httpServer != nil {
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for all mock requests
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS, HEAD")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Expose-Headers", "*")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 1. Check artificial delay
	delay := s.defaultLatency
	if reqDelay := r.Header.Get("X-Mock-Delay"); reqDelay != "" {
		if d, err := time.ParseDuration(reqDelay); err == nil {
			delay = d
		}
	}
	if delay > 0 {
		time.Sleep(delay)
	}

	// 2. Extract requested status code & example directives
	preferHeader := r.Header.Get("Prefer")
	preferredStatus := r.Header.Get("X-Mock-Status")
	if preferredStatus == "" {
		preferredStatus = r.URL.Query().Get("mock_status")
	}
	if preferredStatus == "" && preferHeader != "" {
		if m := preferCodeRegex.FindStringSubmatch(preferHeader); len(m) > 1 {
			preferredStatus = m[1]
		}
	}

	preferredExample := r.URL.Query().Get("mock_example")
	if preferredExample == "" && preferHeader != "" {
		if m := preferExampleRegex.FindStringSubmatch(preferHeader); len(m) > 1 {
			preferredExample = m[1]
		}
	}

	// 3. Match Route
	match := s.matcher.Match(r.Method, r.URL.Path)
	if match == nil {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":   "Not Found",
			"message": fmt.Sprintf("No matching OpenAPI operation found for %s %s", r.Method, r.URL.Path),
			"hint":    "Verify the path parameters and HTTP method match your OpenAPI document.",
		})
		return
	}

	// 4. Generate mock response
	statusCode, headers, payload := s.generator.GenerateResponse(match.Endpoint, preferredStatus, preferredExample)

	for k, v := range headers {
		w.Header().Set(k, v)
	}

	// Include path params in debug header
	if len(match.PathParams) > 0 {
		var paramPairs []string
		for pk, pv := range match.PathParams {
			paramPairs = append(paramPairs, pk+"="+pv)
		}
		w.Header().Set("X-Mock-Path-Params", strings.Join(paramPairs, "; "))
	}

	w.WriteHeader(statusCode)

	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}
