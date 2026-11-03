# CORS Proxy and Network Telemetry

The APIDock interactive client executes HTTP requests to any destination URL without requiring backend CORS configurations on the target server.

## Problem Statement

When testing APIs from a browser-based single-page application, requests to third-party endpoints or staging servers are subject to Cross-Origin Resource Sharing (CORS) enforcement. If the target server does not return `Access-Control-Allow-Origin`, the browser blocks the response from reaching the client script.

## The APIDock Proxy Solution

APIDock includes a lightweight, secure forward proxy endpoint at `POST /api/proxy`:

```
[ Browser UI ]
      |
      |  POST /api/proxy (JSON envelope with target URL, method, headers, body)
      v
[ Go APIDock Backend ]
      |
      |  Direct HTTP request with httptrace telemetry
      v
[ Target API Server ]
      |
      |  Raw HTTP Response
      v
[ Go APIDock Backend ]
      |
      |  Returns status, headers, body, and timing telemetry
      v
[ Browser UI ]
```

## Request Envelope Structure

The client dispatches the following JSON payload to `/api/proxy`:

```json
{
  "url": "https://api.example.com/v1/orders",
  "method": "POST",
  "headers": {
    "Authorization": "Bearer tok_live_xyz",
    "Content-Type": "application/json"
  },
  "body": "{\"quantity\": 2}"
}
```

## Telemetry Response Envelope

The proxy returns the response alongside connection performance metrics:

```json
{
  "status": 200,
  "statusText": "OK",
  "headers": {
    "content-type": "application/json",
    "x-request-id": "req_88f91a"
  },
  "body": "{\"id\":\"ord_123\",\"status\":\"confirmed\"}",
  "timing": {
    "dnsLookupMs": 12.4,
    "tcpConnectionMs": 18.2,
    "tlsHandshakeMs": 35.1,
    "ttfbMs": 78.5,
    "totalMs": 112.3,
    "sizeBytes": 1420
  }
}
```

## Direct vs Proxy Toggle

Users can toggle execution modes directly in the UI:
- **Direct Mode**: The browser issues `fetch()` directly to the target URL. Useful for local APIs that already allow localhost origins.
- **Proxy Mode**: Routes through APIDock Go backend to bypass CORS constraints and measure detailed network timing breakdown.
