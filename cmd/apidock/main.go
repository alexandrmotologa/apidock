package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/alexandrmotologa/apidock/internal/mock"
	"github.com/alexandrmotologa/apidock/internal/server"
	"github.com/alexandrmotologa/apidock/internal/spec"
	"github.com/alexandrmotologa/apidock/internal/watcher"
)

var (
	version = "1.0.0"
	commit  = "main"
	date    = "2026-11-03"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "serve":
		cmdServe(args)
	case "mock":
		cmdMock(args)
	case "validate":
		cmdValidate(args)
	case "export":
		cmdExport(args)
	case "version", "-v", "--version":
		fmt.Printf("APIDock v%s (%s, built %s) %s/%s\n", version, commit, date, runtime.GOOS, runtime.GOARCH)
	case "help", "-h", "--help":
		printUsage()
	default:
		// If first argument looks like a spec file or URL, treat as `serve <arg>`
		if strings.HasSuffix(command, ".yaml") || strings.HasSuffix(command, ".yml") || strings.HasSuffix(command, ".json") || strings.HasPrefix(command, "http://") || strings.HasPrefix(command, "https://") {
			cmdServe(os.Args[1:])
			return
		}
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`APIDock - Single-Binary OpenAPI 3.1 Documentation Engine & Mock Server

Usage:
  apidock <command> [arguments]

Commands:
  serve <spec>     Serve interactive documentation web studio & mock server
  mock <spec>      Start standalone schema-driven mock server
  validate <spec>  Verify OpenAPI specification syntax and reference integrity
  export <spec>    Export standalone static documentation files for web hosting
  version          Display version and build information

Flags for 'serve':
  --port int          Port for web studio documentation (default: 8080)
  --mock-port int     Port for schema-driven mock daemon (default: 4010)
  --no-mock           Disable embedded mock server
  --no-open           Do not automatically open default browser
  --watch             Auto-reload web studio when local spec changes (default: true)
  --latency duration  Simulated default latency for mock responses (e.g. 150ms)

Flags for 'mock':
  --port int          Port for mock server (default: 4010)
  --latency duration  Simulated default latency for mock responses (default: 0ms)
  --watch             Auto-reload mock server when local spec changes (default: true)

Flags for 'validate':
  --strict            Treat warnings as errors

Flags for 'export':
  --out string        Output directory for exported assets (default: ./dist)

Examples:
  apidock serve ./openapi.yaml
  apidock serve https://api.example.com/openapi.json --port 3000 --mock-port 4010
  apidock mock ./openapi.yaml --port 4010 --latency 150ms
  apidock validate ./openapi.yaml --strict
  apidock export ./openapi.yaml --out ./public-docs`)
}

func separateArgs(args []string) ([]string, []string) {
	var flagArgs []string
	var posArgs []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			// Check if flag takes a separate value
			if !strings.Contains(arg, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				// Peek if it's a known boolean flag
				if arg == "--no-mock" || arg == "--no-open" || arg == "--strict" {
					continue
				}
				flagArgs = append(flagArgs, args[i+1])
				i++
			}
		} else {
			posArgs = append(posArgs, arg)
		}
	}
	return flagArgs, posArgs
}

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	port := fs.Int("port", 8080, "Web studio port")
	mockPort := fs.Int("mock-port", 4010, "Mock server port")
	noMock := fs.Bool("no-mock", false, "Disable mock server")
	noOpen := fs.Bool("no-open", false, "Do not open browser")
	watch := fs.Bool("watch", true, "Auto-reload on file changes")
	latency := fs.Duration("latency", 0, "Default mock latency")

	flagArgs, posArgs := separateArgs(args)
	_ = fs.Parse(flagArgs)

	if len(posArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: spec path or URL is required. Example: apidock serve ./openapi.yaml")
		os.Exit(1)
	}
	specTarget := posArgs[0]

	parser := spec.NewParser()
	log.Printf("[APIDock] Parsing specification: %s\n", specTarget)

	parsedSpec, err := parser.Parse(specTarget)
	if err != nil {
		log.Fatalf("Fatal: failed to parse OpenAPI spec: %v\n", err)
	}

	rawBytes, _ := os.ReadFile(specTarget)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("\n[APIDock] Shutting down services...")
		cancel()
	}()

	var mockDaemon *mock.Server
	if !*noMock {
		mockDaemon = mock.NewServer(mock.Config{
			Port:           *mockPort,
			DefaultLatency: *latency,
			Spec:           parsedSpec,
		})
		go func() {
			if err := mockDaemon.Start(); err != nil && err != http.ErrServerClosed {
				log.Printf("[Mock Server Error] %v\n", err)
			}
		}()
	}

	var fileWatcher *watcher.Watcher
	isLocalFile := !strings.HasPrefix(specTarget, "http://") && !strings.HasPrefix(specTarget, "https://")
	if *watch && isLocalFile {
		fileWatcher = watcher.NewWatcher(specTarget)
		if err := fileWatcher.Start(ctx); err != nil {
			log.Printf("[Watcher Warning] Failed to start file watcher: %v\n", err)
		}
	}

	apiServer := server.NewAPIServer(server.Config{
		Port:       *port,
		MockPort:   *mockPort,
		SpecPath:   specTarget,
		Spec:       parsedSpec,
		SpecBytes:  rawBytes,
		MockServer: mockDaemon,
		Watcher:    fileWatcher,
	})

	studioURL := fmt.Sprintf("http://localhost:%d", *port)
	if !*noOpen {
		go func() {
			time.Sleep(300 * time.Millisecond)
			openBrowser(studioURL)
		}()
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, sCancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer sCancel()
		_ = apiServer.Stop(shutdownCtx)
		if mockDaemon != nil {
			_ = mockDaemon.Stop(shutdownCtx)
		}
	}()

	fmt.Printf("\n✨ APIDock Studio: %s\n", studioURL)
	if !*noMock {
		fmt.Printf("⚡ Mock Server:     http://localhost:%d\n\n", *mockPort)
	}

	if err := apiServer.Start(ctx); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server stopped with error: %v\n", err)
	}
}

func cmdMock(args []string) {
	fs := flag.NewFlagSet("mock", flag.ExitOnError)
	port := fs.Int("port", 4010, "Mock server port")
	latency := fs.Duration("latency", 0, "Default mock latency")

	flagArgs, posArgs := separateArgs(args)
	_ = fs.Parse(flagArgs)

	if len(posArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: spec path or URL is required. Example: apidock mock ./openapi.yaml")
		os.Exit(1)
	}
	specTarget := posArgs[0]

	parser := spec.NewParser()
	parsedSpec, err := parser.Parse(specTarget)
	if err != nil {
		log.Fatalf("Fatal: failed to parse OpenAPI spec: %v\n", err)
	}

	mockDaemon := mock.NewServer(mock.Config{
		Port:           *port,
		DefaultLatency: *latency,
		Spec:           parsedSpec,
	})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("\n[Mock Server] Stopping...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = mockDaemon.Stop(shutdownCtx)
		os.Exit(0)
	}()

	fmt.Printf("⚡ APIDock Mock Server running at http://localhost:%d\n", *port)
	if err := mockDaemon.Start(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Mock server stopped with error: %v\n", err)
	}
}

func cmdValidate(args []string) {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	strict := fs.Bool("strict", false, "Treat warnings as errors")

	flagArgs, posArgs := separateArgs(args)
	_ = fs.Parse(flagArgs)

	if len(posArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: spec path or URL is required. Example: apidock validate ./openapi.yaml")
		os.Exit(1)
	}
	specTarget := posArgs[0]

	parser := spec.NewParser()
	parsedSpec, err := parser.Parse(specTarget)
	if err != nil {
		fmt.Printf("❌ Specification parsing failed: %v\n", err)
		os.Exit(1)
	}

	validator := spec.NewValidator()
	diagnostics := validator.Validate(parsedSpec)

	hasErrors := false
	hasWarnings := false

	fmt.Printf("Auditing specification: %s (OpenAPI %s)\n", parsedSpec.Info.Title, parsedSpec.OpenAPI)
	fmt.Printf("Endpoints checked: %d\n\n", len(parsedSpec.Endpoints))

	for _, d := range diagnostics {
		if d.Severity == "ERROR" {
			hasErrors = true
			fmt.Printf("  ❌ [ERROR] %-30s %s\n", d.Path, d.Message)
		} else {
			hasWarnings = true
			fmt.Printf("  ⚠️  [WARN]  %-30s %s\n", d.Path, d.Message)
		}
	}

	if len(diagnostics) == 0 {
		fmt.Println("✅ Specification is valid! Zero errors or structural warnings found.")
		return
	}

	if hasErrors || (*strict && hasWarnings) {
		fmt.Printf("\nValidation failed with %d issue(s).\n", len(diagnostics))
		os.Exit(1)
	}

	fmt.Printf("\nSpecification passed with %d warning(s).\n", len(diagnostics))
}

func cmdExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	outDir := fs.String("out", "./dist", "Output directory")

	flagArgs, posArgs := separateArgs(args)
	_ = fs.Parse(flagArgs)

	if len(posArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: spec path or URL is required. Example: apidock export ./openapi.yaml --out ./dist")
		os.Exit(1)
	}
	specTarget := posArgs[0]

	parser := spec.NewParser()
	parsedSpec, err := parser.Parse(specTarget)
	if err != nil {
		log.Fatalf("Fatal: failed to parse OpenAPI spec: %v\n", err)
	}

	if err := os.MkdirAll(*outDir, 0755); err != nil {
		log.Fatalf("Failed to create export directory: %v\n", err)
	}

	specJSON, err := json.MarshalIndent(parsedSpec, "", "  ")
	if err != nil {
		log.Fatalf("Failed to serialize spec: %v\n", err)
	}

	specOutPath := filepath.Join(*outDir, "spec.json")
	if err := os.WriteFile(specOutPath, specJSON, 0644); err != nil {
		log.Fatalf("Failed to write spec.json: %v\n", err)
	}

	fmt.Printf("✅ Exported static documentation data to: %s\n", *outDir)
}

func openBrowser(url string) {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	case "darwin":
		cmd = "open"
		args = []string{url}
	default:
		cmd = "xdg-open"
		args = []string{url}
	}

	_ = exec.Command(cmd, args...).Start()
}
