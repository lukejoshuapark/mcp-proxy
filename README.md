![mcp-proxy](./icon.png)

# mcp-proxy

An OAuth 2.0 proxy for [Model Context Protocol](https://modelcontextprotocol.io) (MCP) servers. It sits between MCP clients and an upstream MCP server, presenting itself as an OAuth authorization server to clients while delegating actual authentication to an upstream OAuth provider.

## How It Works

1. The MCP client discovers OAuth metadata at `/.well-known/oauth-authorization-server`.
2. The client starts an authorization code flow (with PKCE) against `/authorize`.
3. mcp-proxy validates the client's metadata document, creates a session, and redirects the user's browser to the upstream OAuth provider.
4. After the user authenticates, the upstream provider redirects back to `/callback`.
5. mcp-proxy exchanges the upstream authorization code for tokens, stores the token response, and redirects the client to its `redirect_uri` with a proxy-issued code.
6. The client exchanges the proxy code at `/token` (PKCE-verified) and receives the upstream token response directly.
7. Subsequent requests to `/` are proxied through to the upstream MCP server with the client's token intact.

Refresh token requests are forwarded to the upstream token endpoint transparently.

## Endpoints

| Path | Description |
|------|-------------|
| `/.well-known/oauth-authorization-server` | OAuth 2.0 authorization server metadata (RFC 8414) |
| `/.well-known/oauth-protected-resource` | OAuth protected resource metadata (RFC 9728) |
| `/authorize` | Authorization endpoint — begins the OAuth flow |
| `/callback` | Receives the redirect from the upstream provider |
| `/token` | Token endpoint — `authorization_code` and `refresh_token` grants |
| `/` | Reverse proxy to the upstream MCP server |
| `/healthz` | Health check — returns `200 OK` with body `ok` |

## Configuration

All configuration is via environment variables.

### Required

| Variable | Description |
|----------|-------------|
| `MCP_PROXY_PUBLIC_URL` | The public base URL of this proxy (must use `https`). Used to build OAuth metadata, redirect URIs, and the callback URL. |
| `MCP_PROXY_REMOTE_AUTH_URL` | Authorization endpoint of the upstream OAuth provider. |
| `MCP_PROXY_REMOTE_TOKEN_URL` | Token endpoint of the upstream OAuth provider. |
| `MCP_PROXY_REMOTE_CLIENT_ID` | Client ID registered with the upstream OAuth provider. |
| `MCP_PROXY_REMOTE_CLIENT_SECRET` | Client secret for the upstream OAuth provider. |
| `MCP_PROXY_UPSTREAM_MCP_URL` | URL of the upstream MCP server to proxy requests to. |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `MCP_PROXY_LISTEN_ADDR` | `:8080` | Address and port the HTTP server listens on. |
| `MCP_PROXY_AZURE_STORAGE_ACCOUNT` | *(unset)* | Azure Storage account name. When set (along with `MCP_PROXY_AZURE_STORAGE_KEY`), the proxy uses Azure Table Storage for sessions and authorization codes. When omitted, an in-memory store is used — suitable for single-instance deployments and development. |
| `MCP_PROXY_AZURE_STORAGE_KEY` | *(unset)* | Azure Storage account key. Required when `MCP_PROXY_AZURE_STORAGE_ACCOUNT` is set. |
| `MCP_PROXY_ENCRYPTION_KEY` | *(unset)* | A 32-byte key encoded as raw-URL-base64 (no padding). When set, authorization codes stored in Azure Table Storage are encrypted at rest with AES-256-GCM. Has no effect when using the in-memory store. |
| `MCP_PROXY_SCOPES` | *(unset)* | Space-separated OAuth scopes to request from the upstream provider (e.g. `openid profile email`). When set, this value is used regardless of what the MCP client requests. When omitted, the client-supplied scope is forwarded as-is. |
| `MCP_PROXY_REMOTE_AUDIENCE` | *(unset)* | When set, adds an `audience` query parameter to the upstream authorization URL. Required by some providers (e.g. Auth0) to identify the target API. |
| `MCP_PROXY_PRETTY` | *(unset)* | When set to any non-empty value, logs are emitted in human-readable text format instead of JSON. |

## Security Features

- **PKCE (S256)** — All authorization code exchanges require Proof Key for Code Exchange.
- **SSRF protection** — Client metadata fetches block connections to private, loopback, link-local, and IPv6 unique-local addresses.
- **Rate limiting** — Per-IP token-bucket rate limiter (100 requests/minute) on all endpoints.
- **Encryption at rest** — Optional AES-256-GCM encryption for stored authorization codes in Azure Table Storage.
- **Atomic operations** — Azure Table Storage uses ETag-based conditional deletes so authorization codes can only be redeemed once.
- **Security headers** — All responses include `X-Content-Type-Options: nosniff` and `X-Frame-Options: DENY`.
- **Client metadata validation** — Client `client_id` must be an HTTPS URL with a path. The metadata document at that URL must match the `client_id` and declare at least one `redirect_uri`.
- **Structured logging** — All requests are logged with method, path, status, duration, and client IP via `log/slog`.
