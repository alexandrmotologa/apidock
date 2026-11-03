# APIDock Mock Server Guide

The APIDock mock server provides deterministic, schema-compliant HTTP responses directly from your OpenAPI specifications without writing backend code.

## Starting the Mock Server

The mock server runs automatically when executing `apidock serve`:

```bash
apidock serve ./openapi.yaml --port 8080 --mock-port 4010
```

To run only the mock server without the documentation web UI:

```bash
apidock mock ./openapi.yaml --port 4010
```

## Route Matching

Requests sent to `http://localhost:4010` are matched against the paths declared in the OpenAPI specification:
- HTTP method must match the operation (`GET`, `POST`, `PUT`, `DELETE`, `PATCH`).
- Parameterized paths like `/v1/users/{id}/orders/{orderId}` match concrete requests like `/v1/users/usr_491/orders/ord_990`.
- Trailing slashes are normalized automatically.

## Response Strategy

### 1. Explicit Examples

If an operation defines an example at the response or schema level, the mock server returns that exact structure:

```yaml
responses:
  '200':
    description: Successful response
    content:
      application/json:
        example:
          id: "usr_123"
          name: "Alex Doe"
          role: "admin"
```

### 2. Synthetic Generation

When explicit examples are missing, the generator builds realistic responses from property definitions and formats:

| Schema Format | Generated Sample Value |
| :--- | :--- |
| `uuid` | `3fa85f64-5717-4562-b3fc-2c963f66afa6` |
| `email` | `user@example.com` |
| `uri` / `url` | `https://example.com` |
| `date-time` | `2026-11-03T16:00:00Z` |
| `date` | `2026-11-03` |
| `time` | `16:00:00` |
| `ipv4` | `192.0.2.1` |
| `ipv6` | `2001:db8::1` |
| `hostname` | `api.example.com` |
| `int32`, `int64` | Random integer honoring minimum/maximum bounds |
| `float`, `double` | Floating point number |
| `boolean` | `true` |
| `enum` | First item declared in the enum list |

### 3. Nested Objects and Arrays

- `type: object`: Recursively evaluates `properties` and `required` fields.
- `type: array`: Produces an array with 1 to 3 items conforming to `items` schema.

## Advanced Control Directives

### Forcing Status Codes

You can request a specific response status code (such as error states) using either an HTTP header or query parameter:

```bash
# Using header
curl -H "Prefer: code=404" http://localhost:4010/v1/users/123

# Using query parameter
curl http://localhost:4010/v1/users/123?mock_status=500
```

### Selecting Named Examples

If your OpenAPI specification provides multiple examples in the `examples` map:

```bash
curl -H "Prefer: example=NotFound" http://localhost:4010/v1/users/999
```

### Simulating Network Latency

Test frontend loading skeletons and optimistic UI states by injecting artificial delays:

```bash
# Via header on a per-request basis
curl -H "X-Mock-Delay: 350ms" http://localhost:4010/v1/users

# Globally via CLI flag
apidock mock ./openapi.yaml --port 4010 --latency 250ms
```
