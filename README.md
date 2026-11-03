# APIDock

Single-binary OpenAPI 3.0 and 3.1 documentation server, interactive HTTP client, and schema-driven mock server.

APIDock reads OpenAPI specifications (YAML or JSON) and serves an interactive web studio. It bundles an offline-first documentation viewer, an in-browser request runner with CORS proxy bypass, an automated mock engine on port 4010, and an encrypted client-side environment manager into one standalone binary.

## Features

- **OpenAPI 3.0 and 3.1 Parser**: Full support for YAML and JSON specifications with recursive `$ref` pointer resolution across internal schemas and external files.
- **Embedded Web Studio**: Three-column documentation interface with dark-mode styling, tag-based navigation, search filtering, schema hierarchy explorer, and Monaco Editor.
- **Integrated HTTP Client**: Send live API requests directly from the browser with authentication headers, path parameters, query parameters, and raw JSON request bodies.
- **CORS Bypass Proxy & Streaming**: Built-in backend proxy endpoint (`POST /api/proxy`) to test third-party endpoints without browser cross-origin restrictions, with full streaming support for Server-Sent Events (SSE) and NDJSON.
- **Deterministic Mock Server**: High-speed mock server on port 4010 that parses response schemas and generates realistic synthetic responses matching data formats, constraints, and explicit examples.
- **Breaking Changes Diff Engine**: Compare two specification revisions in the terminal or web studio to detect removed endpoints, newly required parameters, altered schemas, or removed status codes before deployment.
- **Multi-Format Exporters**: Export self-contained standalone HTML documents (`--single-file`), Postman Collections v2.1 (`--format postman`), or Bruno collection workspaces (`--format bruno`).
- **cURL Command Import**: Parse raw cURL commands and populate the interactive playground parameters, headers, and request body automatically.
- **Response Contract Validator**: Evaluate live API responses against declared OpenAPI schemas with percentage conformity scores and path diagnostics.
- **Environment and Secret Vault**: Manage Local, Staging, and Production environments with client-side encryption using AES-GCM.
- **Code Snippet Generator**: Automatic generation of copy-pasteable request snippets in cURL, TypeScript (Fetch and Axios), Python (Requests and HTTPX), and Go.
- **Hot-Reload Watcher**: Automatically reloads the web UI when local OpenAPI specification files change on disk.
- **Single Static Binary**: Go binary embeds all compiled web assets via `go:embed`, requiring zero runtime dependencies.

## Architecture

APIDock operates as a unified Go process containing three primary internal engines:

```
+-------------------------------------------------------------------+
|                             APIDock                               |
+---------------------------------+---------------------------------+
                                  |
        +-------------------------+-------------------------+
        |                                                   |
+-------v-------------------+               +---------------v-------+
|  OpenAPI 3.1 / 3.0 Parser |               |   Mock Server (:4010) |
|  * Recursive $ref engine  |               |  * Schema synthesis   |
|  * Type & format mapping  |               |  * Route matcher      |
|  * Multi-file resolver    |               |  * Example fallback   |
+-------+-------------------+               +---------------+-------+
        |                                                   |
        +-------------------------+-------------------------+
                                  |
                +-----------------v-----------------+
                |   CORS Proxy & Client Gateway     |
                |  * DNS and TLS timing metrics     |
                |  * Header and cookie forwarding   |
                |  * Sub-millisecond latency report |
                +-----------------+-----------------+
                                  |
                +-----------------v-----------------+
                |   Embedded Web Server (:8080)     |
                |  * Single-binary React 19 UI      |
                |  * Monaco Editor integration      |
                |  * Multi-language code snippets   |
                +-----------------------------------+
```

## Quick Start

### Installation

Download pre-built binaries from GitHub Releases or compile from source using Go:

```bash
# Install with Go
go install github.com/alexandrmotologa/apidock/cmd/apidock@latest

# Verify installation
apidock version
```

### Basic Usage

Serve documentation and mock server from a local OpenAPI file:

```bash
apidock serve ./openapi.yaml
```

Serve from a remote URL:

```bash
apidock serve https://api.example.com/openapi.json --port 8080 --mock-port 4010
```

Start only the mock server:

```bash
apidock mock ./openapi.yaml --port 4010
```

Validate an OpenAPI specification file:

```bash
apidock validate ./openapi.yaml --strict
```

Detect breaking changes between two specification versions:

```bash
apidock diff ./v1.yaml ./v2.yaml
```

Export standalone zero-dependency HTML documentation:

```bash
apidock export ./openapi.yaml --single-file --out ./docs.html
```

Export Postman collection or Bruno workspace:

```bash
apidock export ./openapi.yaml --format postman --out ./postman.json
apidock export ./openapi.yaml --format bruno --out ./bruno
```

## CLI Reference

| Command | Flags | Description |
| :--- | :--- | :--- |
| `serve <spec>` | `--port 8080`, `--mock-port 4010`, `--no-mock`, `--no-open`, `--watch` | Starts web documentation studio and mock server. |
| `mock <spec>` | `--port 4010`, `--latency 0ms`, `--cors` | Runs standalone schema-driven mock server. |
| `validate <spec>` | `--strict` | Verifies syntax, schema references, and structure. |
| `diff <old> <new>` | `--json`, `--fail-on-breaking` | Compares two specifications and reports breaking changes. |
| `export <spec>` | `--out <path>`, `--single-file`, `--format postman\|bruno\|static` | Exports static HTML, single-file bundle, Postman JSON, or Bruno workspace. |
| `version` | none | Prints version, commit hash, and build date. |

## Mock Server Capabilities

The mock server matches incoming HTTP requests against defined paths and operations:

- **Example Fallback**: If an operation defines explicit `example` or `examples` objects in its response schema, the mock server returns that data directly.
- **Synthetic Generation**: In the absence of explicit examples, values are generated using OpenAPI format rules:
  - `format: uuid`: `123e4567-e89b-12d3-a456-426614174000`
  - `format: email`: `user@example.com`
  - `format: date-time`: RFC 3339 timestamp
  - `format: uri`: `https://example.com/resource`
  - `format: ipv4` / `ipv6`: standard IP addresses
  - `type: integer`, `type: number`: values respecting `minimum` and `maximum`
  - `type: array`: arrays populated with items matching item schema constraints
- **Status Override**: Pass `Prefer: code=404` header or `?mock_status=404` query parameter to simulate error responses.
- **Latency Simulation**: Pass `X-Mock-Delay: 200ms` or configure `--latency 200ms` to simulate realistic network delay.

## Tech Stack

- **Backend**: Go 1.23+
- **OpenAPI Engine**: `github.com/pb33f/libopenapi`
- **Frontend**: React 19, TypeScript, Vite, Tailwind CSS
- **Code Editor**: Monaco Editor (`@monaco-editor/react`)
- **Icons**: Lucide React

## Development Setup

```bash
# Clone the repository
git clone https://github.com/alexandrmotologa/apidock.git
cd apidock

# Install web dependencies
cd web
npm install

# Run frontend in development mode
npm run dev

# In another terminal, run Go backend
cd ..
go run ./cmd/apidock serve ./testdata/petstore.yaml --no-open
```

## License

MIT License. See [LICENSE](LICENSE) for details.
