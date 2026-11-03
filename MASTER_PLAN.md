# APIDock: Master Implementation Plan

> **Single-Binary OpenAPI 3.1 Documentation Engine, Modern Interactive API Playground & Mock Server**
>
> High-performance, offline-first API documentation platform, interactive test client, and schema-driven mock server (Scalar + Bruno alternative). Ingests OpenAPI 3.1 and 3.0 specifications, resolves multi-file `$ref` pointers, simulates realistic mock responses without backend code, and provides a luxury dark-mode developer studio.

---

## 1. Project Overview & Architecture

APIDock delivers a unified, single-binary, offline-first API workstation:
- **Instant Documentation Engine**: Runs via `apidock serve ./openapi.yaml` (or fetches from remote URL), instantly rendering responsive, luxury dark-mode documentation inspired by Stripe and Resend docs.
- **Embedded Interactive API Client**: Send live HTTP requests directly from documentation pages with support for bearer tokens, API keys, OAuth2 flows, query params, and file uploads.
- **Schema-Driven Mock Server (`:4010`)**: Generates deterministic, type-compliant mock JSON responses derived from schema definitions and examples, allowing frontend developers to build before backend implementation is complete.
- **Environment & Secret Manager**: Switch between `Local`, `Staging`, and `Production` with encrypted client-side storage for API keys and environment variables using AES-GCM.
- **Multi-Language Code Generator**: Generates copy-pasteable snippets in cURL, TypeScript (Fetch/Axios), Python (Requests/HTTPX), and Go with full parameter substitution.
- **Zero External Runtime Dependencies**: Standalone Go binary embeds all compiled web assets via `go:embed`.

```
+-----------------------------------------------------------------------------------+
|                                      APIDOCK                                      |
+-----------------------------------------+-----------------------------------------+
                                          |
        +---------------------------------+---------------------------------+
        |                                                                   |
+-------v-------------------------+                 +-----------------------v-------+
|    OpenAPI 3.1 & 3.0 Parser     |                 |   Schema-Driven Mock Server   |
|  * Multi-File $ref Resolution   |                 |  * Realistic Synthetic Mocking|
|  * oneOf, anyOf, Discriminators |                 |  * Port :4010 Local Daemon    |
|  * Strict Validation & Warnings |                 |  * Fulfills Example Payloads  |
+-------+-------------------------+                 +-----------------------+-------+
        |                                                                   |
        +---------------------------------+---------------------------------+
                                          |
                        +-----------------v-----------------+
                        |   Interactive API Client Engine   |
                        |  * Direct & Proxied HTTP Requests |
                        |  * Local Cookie & Auth Manager    |
                        |  * Sub-Millisecond Response Timing|
                        +-----------------+-----------------+
                                          |
                        +-----------------v-----------------+
                        |    Multi-Language Code Generator  |
                        |  * cURL, TypeScript, Python, Go   |
                        |  * Parameterized Request Headers  |
                        |  * 1-Click Clipboard Copy         |
                        +-----------------+-----------------+
                                          |
                        +-----------------v-----------------+
                        |   Interactive Web Studio & CLI    |
                        |  * Luxury Dark-Mode Documentation |
                        |  * Environment Switcher (Loc/Stg) |
                        |  * Single Binary Delivery (:8080) |
                        +-----------------------------------+
```

---

## 2. Directory Structure

```
apidock/
├── cmd/
│   └── apidock/
│       └── main.go               # CLI entrypoint with subcommands: serve, mock, validate, export
├── internal/
│   ├── spec/
│   │   ├── parser.go             # Ingests JSON/YAML OpenAPI 3.0 & 3.1 specs with libopenapi
│   │   ├── resolver.go           # Recursive $ref pointer resolution across local/remote files
│   │   ├── model.go              # Normalized spec representation for UI & mock engine
│   │   └── validator.go          # Schema conformity and syntax checks
│   ├── mock/
│   │   ├── server.go             # Mock HTTP server listening on :4010
│   │   ├── generator.go          # Generates synthetic data matching schema types & constraints
│   │   └── matcher.go            # Route matching by method, path, and path parameters
│   ├── proxy/
│   │   ├── client.go             # Hardened HTTP proxy to bypass browser CORS with httptrace
│   │   └── history.go            # In-memory execution history recorder
│   ├── codegen/
│   │   ├── curl.go               # cURL command generator
│   │   ├── typescript.go         # TypeScript (Fetch/Axios) generator
│   │   ├── python.go             # Python (Requests/HTTPX) generator
│   │   └── golang.go             # Go standard library generator
│   ├── watcher/
│   │   └── watcher.go            # File watcher for hot-reloading spec via SSE
│   └── server/
│       ├── api.go                # REST API for spec serving, proxy, and SSE reload
│       └── static.go             # Embedded Vite frontend assets via go:embed
├── web/
│   ├── src/
│   │   ├── components/
│   │   │   ├── ApiDocViewer.tsx      # Sidebar navigation & endpoint documentation
│   │   │   ├── InteractiveClient.tsx # Param inputs, auth headers, Monaco editor & "Send"
│   │   │   ├── ResponseViewer.tsx    # Monaco response viewer, headers, status & latency
│   │   │   ├── CodeSnippetDrawer.tsx # Multi-language code snippets
│   │   │   ├── EnvSwitcherModal.tsx  # Environment variable and secret manager with AES-GCM
│   │   │   ├── SearchModal.tsx       # Quick search modal (Cmd+K)
│   │   │   └── SchemaViewer.tsx      # Tree view for JSON Schemas with types and constraints
│   │   ├── hooks/
│   │   │   ├── useOpenApiSpec.ts
│   │   │   └── useEnvironment.ts
│   │   ├── types/
│   │   │   └── spec.ts
│   │   ├── App.tsx
│   │   └── main.tsx
│   ├── package.json
│   ├── tailwind.config.js
│   └── vite.config.ts
├── testdata/
│   └── petstore.yaml             # Full OpenAPI 3.1 test specification
├── docs/
│   ├── architecture.md
│   ├── mock-server.md
│   ├── cors-proxy.md
│   └── client-vault.md
├── go.mod
├── go.sum
└── README.md
```

---

## 3. Detailed Implementation Roadmap & Tasks

### Task 0: Foundation, GitHub Repository & Documentation [CURRENT]
- [x] Initialize Git repository with proper user identity.
- [x] Create `.gitignore` and `LICENSE` (MIT).
- [x] Create comprehensive `README.md` following Humanizer guidelines (no em/en dashes, no hype words, clean tables).
- [x] Create `docs/architecture.md`, `docs/mock-server.md`, `docs/cors-proxy.md`, `docs/client-vault.md`.
- [ ] Create public GitHub repository `alexandrmotologa/apidock`.
- [ ] Set GitHub About description and 14 curated topics across 4 dimensions.
- [ ] Perform Phase 1 Portfolio Registration in `mtlg-site/projects.json` and `mtlglabs-space/projects.json`.
- [x] Refine `MASTER_PLAN.md` with enhanced features and execution breakdown.

### Task 1: OpenAPI 3.0/3.1 Parser & Normalized Model (`internal/spec`)
- [ ] Initialize Go module `github.com/alexandrmotologa/apidock` (Go 1.23+).
- [ ] Install `github.com/pb33f/libopenapi`.
- [ ] Implement `internal/spec/model.go` defining clean JSON-serializable structures for endpoints, operations, schemas, parameters, security schemes, and server definitions.
- [ ] Implement `internal/spec/parser.go` loading specs from file path or remote HTTP/HTTPS URL.
- [ ] Implement `internal/spec/resolver.go` resolving internal and external `$ref` pointers.
- [ ] Implement `internal/spec/validator.go` validating structural integrity and reporting human-readable diagnostics.
- [ ] Add unit tests in `internal/spec/parser_test.go` with complex OpenAPI 3.0 and 3.1 fixtures.

### Task 2: Schema-Driven Mock Server (`internal/mock`)
- [ ] Implement `internal/mock/matcher.go` for parameterized route matching (`/v1/users/{id}` vs `/v1/users/123`).
- [ ] Implement `internal/mock/generator.go` producing synthetic schema data:
  - Format heuristics (`uuid`, `email`, `uri`, `date-time`, `ipv4`, etc.).
  - Respects schema constraints (`minimum`, `maximum`, `minLength`, `maxLength`, `enum`, `pattern`).
  - Fallback to explicit `example` and `examples` defined in responses or schemas.
- [ ] Support control directives:
  - `Prefer: code=XYZ` and `?mock_status=XYZ` for status code override.
  - `Prefer: example=NAME` for specific named example selection.
  - `X-Mock-Delay: DURATION` for simulated network latency.
- [ ] Implement `internal/mock/server.go` HTTP server on port 4010 with CORS headers.
- [ ] Add comprehensive unit and integration tests in `internal/mock/mock_test.go`.

### Task 3: CORS Proxy & Request Engine with Telemetry (`internal/proxy`)
- [ ] Implement `internal/proxy/client.go`:
  - `POST /api/proxy` endpoint accepting target URL, method, headers, and body.
  - Integration with `net/http/httptrace` measuring DNS lookup, TCP connect, TLS handshake, TTFB, and total duration.
  - Transparent header forwarding and response streaming.
  - Protection against SSRF (blocking cloud internal metadata IP 169.254.169.254).
- [ ] Implement `internal/proxy/history.go` for recording recent request history.
- [ ] Unit tests for proxy logic and telemetry metrics.

### Task 4: Multi-Language Code Snippet Generator (`internal/codegen`)
- [ ] Implement `internal/codegen/curl.go` generating formatted cURL commands.
- [ ] Implement `internal/codegen/typescript.go` generating native `fetch` and `axios` snippets.
- [ ] Implement `internal/codegen/python.go` generating `requests` and `httpx` snippets.
- [ ] Implement `internal/codegen/golang.go` generating standard `net/http` snippets.
- [ ] Unit tests for code snippet accuracy with parameters, headers, and request bodies.

### Task 5: Web UI Studio in React 19 (`web/`)
- [ ] Scaffold Vite + React 19 + TypeScript + Tailwind CSS project in `web/`.
- [ ] Install `@monaco-editor/react`, `lucide-react`, and utility libraries.
- [ ] Configure luxury dark theme (inspired by Stripe and Resend: rich zinc/slate background, subtle borders, luminous method badges).
- [ ] Build UI Components:
  - `Sidebar.tsx`: Search bar (`Cmd+K`), API title/version/badges, tag navigation with operation counters.
  - `ApiDocViewer.tsx`: Center column showing endpoint title, method badge, markdown description, server selector, security badges, query/path parameters table, request body schema explorer, response tabs with status codes.
  - `InteractiveClient.tsx`: Right column test runner with environment variables substitution, parameter inputs, auth configuration, Monaco editor for JSON body, "Send" button, direct vs proxy toggle.
  - `ResponseViewer.tsx`: Status code pill, timing breakdown chips (DNS, TTFB, Total), response size, header inspection table, Monaco editor for JSON response with copy and folding.
  - `CodeSnippetDrawer.tsx`: Language tabs (cURL, TypeScript Fetch/Axios, Python Requests/HTTPX, Go), code block with syntax styling and 1-click copy.
  - `EnvSwitcherModal.tsx`: Environment manager with Local, Staging, Production tabs, variable key-value editor, and master passphrase for client-side AES-GCM encryption.
  - `SearchModal.tsx`: Instant fuzzy search over all paths, methods, and tags with keyboard navigation.
- [ ] Test frontend build (`npm run build`) ensuring zero bundle errors.

### Task 6: Hot Reload Watcher & CLI Server Integration (`internal/watcher` & `cmd/apidock`)
- [ ] Implement SSE endpoint `/api/events` in `internal/server/` for hot-reloading UI when local spec file changes.
- [ ] Implement `internal/watcher/watcher.go` watching input spec file on disk.
- [ ] Implement `internal/server/static.go` with Go `//go:embed` embedding `web/dist`.
- [ ] Implement CLI commands in `cmd/apidock/main.go`:
  - `apidock serve <spec>`: Serve docs + mock server + proxy + hot reload.
  - `apidock mock <spec>`: Standalone mock server.
  - `apidock validate <spec>`: Structural validation with CLI report.
  - `apidock export <spec>`: Export standalone static HTML/JS bundle.
  - `apidock version`: Version, git commit, build time.

### Task 7: End-to-End Verification & Benchmarks
- [ ] Test with official OpenAPI specifications (Petstore 3.0 & 3.1, Stripe sample, GitHub API subset).
- [ ] Run benchmark testing parsing speed and memory consumption.
- [ ] Verify mock server responses with real cURL and browser calls.
- [ ] Verify CORS proxy with live endpoints.
- [ ] Verify environment secret encryption and variable substitution.
- [ ] Build final standalone executable for Windows/Linux/macOS.
