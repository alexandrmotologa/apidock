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

	"github.com/alexandrmotologa/apidock/internal/diff"
	"github.com/alexandrmotologa/apidock/internal/export"
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
	case "diff":
		cmdDiff(args)
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
  serve <spec>          Serve interactive documentation web studio & mock server
  mock <spec>           Start standalone schema-driven mock server
  validate <spec>       Verify OpenAPI specification syntax and reference integrity
  diff <old> <new>      Detect breaking changes between two OpenAPI specifications
  export <spec>         Export static HTML, Postman collection, or Bruno workspace
  version               Display version and build information

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

Flags for 'diff':
  --json              Output raw JSON report instead of formatted terminal summary
  --fail-on-breaking  Return exit code 1 if breaking changes exist (default: true)

Flags for 'export':
  --format string     Export format: static, single-file, postman, bruno (default: static)
  --single-file       Shorthand for single-file zero-dependency HTML bundle
  --out string        Output directory or file path

Examples:
  apidock serve ./openapi.yaml
  apidock mock ./openapi.yaml --port 4010 --latency 150ms
  apidock validate ./openapi.yaml --strict
  apidock diff ./v1.yaml ./v2.yaml
  apidock export ./openapi.yaml --single-file --out ./docs.html
  apidock export ./openapi.yaml --format postman --out ./postman.json
  apidock export ./openapi.yaml --format bruno --out ./bruno`)
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
				if arg == "--no-mock" || arg == "--no-open" || arg == "--strict" || arg == "--single-file" || arg == "--json" || arg == "--fail-on-breaking" {
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

func cmdDiff(args []string) {
	fs := flag.NewFlagSet("diff", flag.ExitOnError)
	outputJSON := fs.Bool("json", false, "Output raw JSON report")
	failOnBreaking := fs.Bool("fail-on-breaking", true, "Exit with error code 1 if breaking changes exist")

	flagArgs, posArgs := separateArgs(args)
	_ = fs.Parse(flagArgs)

	if len(posArgs) < 2 {
		fmt.Fprintln(os.Stderr, "Error: two specifications are required for comparison.\nExample: apidock diff ./old_spec.yaml ./new_spec.yaml")
		os.Exit(1)
	}

	oldTarget := posArgs[0]
	newTarget := posArgs[1]

	parser := spec.NewParser()
	oldSpec, err := parser.Parse(oldTarget)
	if err != nil {
		log.Fatalf("Fatal: failed to parse base specification (%s): %v\n", oldTarget, err)
	}

	newSpec, err := parser.Parse(newTarget)
	if err != nil {
		log.Fatalf("Fatal: failed to parse target specification (%s): %v\n", newTarget, err)
	}

	report := diff.Compare(oldSpec, newSpec)

	if *outputJSON {
		b, _ := json.MarshalIndent(report, "", "  ")
		fmt.Println(string(b))
		if report.BreakingCount > 0 && *failOnBreaking {
			os.Exit(1)
		}
		return
	}

	fmt.Printf("\nAPIDock Diff Engine - OpenAPI Breaking Changes Report\n")
	fmt.Printf("Comparing: %s (v%s) -> %s (v%s)\n\n", report.OldTitle, report.OldVersion, report.NewTitle, report.NewVersion)

	if len(report.Changes) == 0 {
		fmt.Println("✅ No differences found. Both specifications are functionally identical.")
		return
	}

	if report.BreakingCount > 0 {
		fmt.Printf("🚨 Breaking Changes (%d):\n", report.BreakingCount)
		for _, c := range report.Changes {
			if c.Type == diff.ChangeBreaking {
				fmt.Printf("  ❌ [%-10s] %-30s %s\n", c.Category, c.Path, c.Message)
			}
		}
		fmt.Println()
	}

	if report.NonBreakingCount > 0 {
		fmt.Printf("✨ Non-Breaking Changes (%d):\n", report.NonBreakingCount)
		for _, c := range report.Changes {
			if c.Type == diff.ChangeNonBreaking {
				fmt.Printf("  ✨ [%-10s] %-30s %s\n", c.Category, c.Path, c.Message)
			}
		}
		fmt.Println()
	}

	for _, c := range report.Changes {
		if c.Type == diff.ChangeInfo {
			fmt.Printf("ℹ️  [%-10s] %s\n", c.Category, c.Message)
		}
	}

	fmt.Printf("\nSummary: %d breaking change(s), %d non-breaking change(s).\n", report.BreakingCount, report.NonBreakingCount)

	if report.BreakingCount > 0 && *failOnBreaking {
		os.Exit(1)
	}
}

func cmdExport(args []string) {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	outPath := fs.String("out", "", "Output directory or file path")
	format := fs.String("format", "static", "Export format: static, single-file, postman, bruno")
	singleFile := fs.Bool("single-file", false, "Export zero-dependency standalone HTML file")

	flagArgs, posArgs := separateArgs(args)
	_ = fs.Parse(flagArgs)

	if len(posArgs) == 0 {
		fmt.Fprintln(os.Stderr, "Error: spec path or URL is required. Example: apidock export ./openapi.yaml --single-file")
		os.Exit(1)
	}
	specTarget := posArgs[0]

	parser := spec.NewParser()
	parsedSpec, err := parser.Parse(specTarget)
	if err != nil {
		log.Fatalf("Fatal: failed to parse OpenAPI spec: %v\n", err)
	}

	exportFormat := strings.ToLower(*format)
	if *singleFile {
		exportFormat = "single-file"
	}

	switch exportFormat {
	case "single-file", "html":
		distFS, err := server.GetDistFS()
		if err != nil {
			log.Fatalf("Fatal: failed to open embedded dist filesystem: %v\n", err)
		}
		htmlBytes, err := export.BuildSingleFileHTML(parsedSpec, distFS)
		if err != nil {
			log.Fatalf("Fatal: failed to build standalone HTML: %v\n", err)
		}

		targetFile := *outPath
		if targetFile == "" {
			targetFile = "apidock-docs.html"
		} else if strings.HasSuffix(targetFile, string(filepath.Separator)) || strings.HasSuffix(targetFile, "/") {
			targetFile = filepath.Join(targetFile, "apidock-docs.html")
		}

		if err := os.WriteFile(targetFile, htmlBytes, 0644); err != nil {
			log.Fatalf("Fatal: failed to write standalone HTML file %s: %v\n", targetFile, err)
		}
		fmt.Printf("✅ Exported zero-dependency standalone HTML documentation to: %s\n", targetFile)

	case "postman":
		postmanBytes, err := export.ToPostmanJSON(parsedSpec)
		if err != nil {
			log.Fatalf("Fatal: failed to generate Postman collection: %v\n", err)
		}
		targetFile := *outPath
		if targetFile == "" {
			targetFile = "postman_collection.json"
		} else if strings.HasSuffix(targetFile, string(filepath.Separator)) || strings.HasSuffix(targetFile, "/") {
			targetFile = filepath.Join(targetFile, "postman_collection.json")
		}

		if err := os.WriteFile(targetFile, postmanBytes, 0644); err != nil {
			log.Fatalf("Fatal: failed to write Postman file %s: %v\n", targetFile, err)
		}
		fmt.Printf("✅ Exported Postman collection v2.1 to: %s\n", targetFile)

	case "bruno":
		targetDir := *outPath
		if targetDir == "" {
			targetDir = "./bruno"
		}
		if err := export.ExportBrunoToDir(parsedSpec, targetDir); err != nil {
			log.Fatalf("Fatal: failed to export Bruno collection: %v\n", err)
		}
		fmt.Printf("✅ Exported Bruno collection workspace to: %s\n", targetDir)

	default: // static
		targetDir := *outPath
		if targetDir == "" {
			targetDir = "./dist"
		}
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			log.Fatalf("Failed to create export directory: %v\n", err)
		}

		specJSON, err := json.MarshalIndent(parsedSpec, "", "  ")
		if err != nil {
			log.Fatalf("Failed to serialize spec: %v\n", err)
		}

		specOutPath := filepath.Join(targetDir, "spec.json")
		if err := os.WriteFile(specOutPath, specJSON, 0644); err != nil {
			log.Fatalf("Failed to write spec.json: %v\n", err)
		}
		fmt.Printf("✅ Exported static documentation data to: %s\n", targetDir)
	}
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
