# Client-Side Secret Vault

APIDock allows developers to manage sensitive API keys, tokens, and environment parameters directly in the browser with encryption at rest.

## Security Architecture

1. **Zero Server Storage**: Secrets and environment configurations are stored solely in the client browser (IndexedDB or LocalStorage). The APIDock server never stores your production tokens.
2. **AES-GCM Encryption**: When a user sets a master password for their environment vault, sensitive values are encrypted using AES-GCM 256-bit encryption via the standard Web Crypto API (`window.crypto.subtle`).
3. **Key Derivation**: Passwords derive encryption keys using PBKDF2 with SHA-256 and 100,000 iterations.
4. **Environment Isolation**: Supports `Local`, `Staging`, `Production`, and custom environments with scoped variable overrides.

## Variable Substitution Syntax

APIDock supports variable interpolation across:
- Base URLs (e.g. `{{baseUrl}}/v1/users`)
- Path parameters (e.g. `/orders/{{orderId}}`)
- Headers (e.g. `Authorization: Bearer {{token}}`)
- Query parameters (e.g. `?apiKey={{apiKey}}`)
- Request body JSON

```
Environment: Staging
Variables:
  baseUrl: https://staging.example.com
  token: sec_stg_99214

Rendered Request:
  POST https://staging.example.com/v1/orders
  Authorization: Bearer sec_stg_99214
```
