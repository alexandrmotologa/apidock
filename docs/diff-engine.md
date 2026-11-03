# Breaking Changes & Specification Diff Engine

APIDock contains a built-in semantic diff engine that analyzes the structural contracts of two OpenAPI specifications and flags breaking vs backward-compatible changes.

## Classification Rules

Differences are classified into three severity tiers:

### 1. BREAKING
Any modification that could break existing API consumers:
- Removal of an existing path or operation (e.g. `DELETE /pets/{id}`)
- Addition of a new required parameter (in query, path, header, or body)
- An existing optional parameter changing to required
- Removal of a declared HTTP response code (e.g. removing `200` or `404`)
- Removal of a required parameter definition

### 2. NON_BREAKING
Backward-compatible additions that enhance the contract without breaking existing clients:
- Addition of a new endpoint or operation (e.g. `POST /pets`)
- Addition of a new optional parameter
- Addition of a new response status code (e.g. adding `201 Created`)
- Removal of an optional parameter

### 3. INFO
Contextual informational updates:
- Specification version increments (e.g. `1.0.0` to `2.0.0`)
- Title or metadata revisions

## Command Line Usage

### Terminal Human-Readable Report
Compare two local files or remote URLs:

```bash
apidock diff ./v1.yaml ./v2.yaml
```

Example terminal output:
```
APIDock Diff Engine - OpenAPI Breaking Changes Report
Comparing: Petstore API (v1.0.0) -> Petstore API (v2.0.0)

🚨 Breaking Changes (2):
  ❌ [ENDPOINT  ] DELETE /pets/{id}              Endpoint DELETE /pets/{id} was removed
  ❌ [PARAMETER ] GET /pets (query filter)       New required parameter 'filter' added

✨ Non-Breaking Changes (2):
  ✨ [ENDPOINT  ] POST /pets                     Endpoint POST /pets was added
  ✨ [RESPONSE  ] GET /pets [HTTP 201]           Response code 201 was added

Summary: 2 breaking change(s), 2 non-breaking change(s).
```

### CI/CD Automation
To automate breaking change detection inside GitHub Actions or GitLab CI, use `--fail-on-breaking`:

```bash
# Exits with status code 1 if any breaking changes are detected
apidock diff ./main_spec.yaml ./pr_spec.yaml --fail-on-breaking
```

### JSON Output
Output raw JSON for script integration:

```bash
apidock diff ./main_spec.yaml ./pr_spec.yaml --json > diff-report.json
```

## Interactive Web Studio
Open the APIDock Studio (`apidock serve ./openapi.yaml`), click **Diff** in the sidebar navigation or press `Ctrl+D` (or `Cmd+D` on macOS).
You can paste any updated specification YAML/JSON or upload a file to review visual breakage diagnostics directly in your browser.
