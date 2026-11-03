# APIDock Exporters

APIDock supports exporting OpenAPI specifications into three portable formats without requiring external tooling.

## 1. Single-File Standalone HTML (`--single-file`)

Builds an offline-first, zero-dependency HTML document embedding the complete React studio, Monaco Editor viewer, Tailwind CSS stylesheet, and the OpenAPI specification payload:

```bash
apidock export ./openapi.yaml --single-file --out ./apidock-docs.html
```

### Characteristics
- Zero external network requests required at runtime.
- Works by double-clicking the file in any browser or attaching it to an email or internal wiki.
- CSS and JavaScript bundles are inlined directly into `<style>` and `<script type="module">` tags.
- Spec payload is stored in `<script id="apidock-spec" type="application/json">` and hydrated on startup.

## 2. Postman Collection v2.1 (`--format postman`)

Converts the OpenAPI specification into a Postman Collection v2.1.0 format JSON file:

```bash
apidock export ./openapi.yaml --format postman --out ./postman_collection.json
```

### Structure
- Requests are grouped into folders matching OpenAPI tags.
- Standard headers and query parameters are mapped.
- JSON request bodies and examples are populated.
- `{{baseUrl}}` collection variable is configured from OpenAPI `servers`.

## 3. Bruno Workspace (`--format bruno`)

Converts the OpenAPI specification into a Bruno collection workspace directory:

```bash
apidock export ./openapi.yaml --format bruno --out ./bruno-collection
```

### Generated Files
- `bruno.json`: Collection root configuration.
- `environments/Local.bru`: Environment variable definitions for `baseUrl`.
- Tag folders containing discrete `.bru` files for each endpoint with meta tags, method declarations, headers, query parameters, and example JSON payloads.

## Web Studio Export
Inside the APIDock Web Studio sidebar, download buttons are available at the bottom for quick one-click export of both the Single-File HTML and the Postman Collection.
