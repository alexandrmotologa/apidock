# APIDock Architecture

This document describes the internal design, component interactions, and execution flow of APIDock.

## Overview

APIDock is structured as a modular Go service with an embedded React 19 frontend application. The binary executes without external runtime dependencies and provides three primary services:
1. OpenAPI specification parser and document server.
2. Cross-origin request proxy.
3. Schema-driven synthetic mock server.

## Component Layout

```
                  +-----------------------------------+
                  |            CLI Engine             |
                  |     (cobra/flags in main.go)      |
                  +-----------------+-----------------+
                                    |
          +-------------------------+-------------------------+
          |                                                   |
+---------v---------+                               +---------v---------+
|   Server Manager  |                               |    Mock Daemon    |
|   (Port :8080)    |                               |    (Port :4010)   |
+---------+---------+                               +---------+---------+
          |                                                   |
    +-----+-----+-------------------+                         |
    |           |                   |                         |
+---v----+ +----v----+     +--------v--------+       +--------v--------+
| Static | | Spec    |     | Proxy Engine    |       | Route Matcher   |
| Assets | | API     |     | (httptrace)     |       | & Data Synth    |
| (embed)| | Endpoint|     |                 |       |                 |
+--------+ +---------+     +-----------------+       +-----------------+
```

## Spec Parsing and Reference Resolution

APIDock utilizes `github.com/pb33f/libopenapi` to read both OpenAPI 3.0 and 3.1 specifications in JSON and YAML formats.
- **Index Building**: When loading a document, the parser indexes all schemas, responses, parameters, and paths.
- **Reference Resolution**: `$ref` pointers targeting local anchors (e.g. `#/components/schemas/Item`) and external file paths are resolved into memory.
- **Circular Reference Safety**: Circular schema references are detected and capped during serialization to avoid infinite loops.
- **Normalized Data Model**: The backend converts OpenAPI definitions into an internal JSON-serializable structure consumed directly by the web UI and mock generator.

## Mock Engine Architecture

The mock server operates as a separate HTTP listener (default port `:4010`):
1. **Routing**: Incoming requests are matched against registered OpenAPI paths using radix-tree / parameterized pattern matching (`/users/{id}`).
2. **Response Selection**:
   - Checks for `Prefer: code=XYZ` or `?mock_status=XYZ`.
   - Defaults to the lowest 2xx response defined in the operation (usually `200` or `201`).
3. **Data Generation**:
   - If explicit examples exist in the spec, the mock engine returns them directly.
   - If no example exists, the synthetic generator constructs a payload recursively based on property types, formats (`uuid`, `email`, `date-time`), and constraints.
4. **Latency Simulation**: If requested via `X-Mock-Delay` or CLI flag, the server injects non-blocking timer delays before flushing the response.

## CORS Proxy and Telemetry

Browsers enforce Same-Origin Policy, preventing frontend JavaScript from executing requests to arbitrary origins unless the target server sets permissive CORS headers.
- When proxy mode is active, the web client sends request payloads to `POST /api/proxy`.
- The Go backend issues the request using `net/http` instrumented with `httptrace.ClientTrace`.
- High-precision timestamps record:
  - DNS resolution duration.
  - TCP handshake duration.
  - TLS handshake duration.
  - Time to First Byte (TTFB).
  - Total transfer time and payload size.
- Headers, cookies, and binary/text payloads are transparently returned to the client alongside telemetry data.

## Security Considerations

- Local Secret Vault: Environment variables and tokens are encrypted on the client side using AES-GCM before storage in browser storage.
- Proxy Target Validation: The proxy rejects requests targeting cloud internal metadata endpoints (e.g. `169.254.169.254`) by default to prevent SSRF vulnerabilities when deployed in network-accessible environments.
