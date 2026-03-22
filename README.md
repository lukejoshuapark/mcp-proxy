Make TableStorage Pop atomic. Use an ETag-based conditional delete (Azure Table Storage supports If-Match with ETags) so that only the first Pop caller succeeds and the second gets a 412 Precondition Failed.

Add a health check endpoint (/healthz or similar). Essential for container orchestration (Kubernetes liveness/readiness probes, Azure Container Apps health checks, etc.).

Graceful shutdown. Use signal.NotifyContext + httpServer.Shutdown(ctx) so in-flight requests complete before the process exits.

Cache client metadata documents. Currently fetched on every /authorize request. A short-lived cache (e.g. 5 minutes) would reduce latency and avoid the amplification concern.

Structured access logging. Successful operations are unlogged. A middleware logging method, path, status, and duration would be valuable for operational visibility.

Critical
The proxy endpoint performs no authentication. proxy.go:5-7 unconditionally forwards every request to the upstream MCP server. There is no Bearer token validation, no session check — nothing. The entire OAuth flow is cosmetic; an attacker can bypass it entirely by sending requests directly to the proxy's catch-all / route. This is the single most important finding.

High
Authorization code replay in distributed deployments. TableStorageStore.Pop is Get + Delete as two separate, non-atomic operations. With multiple proxy instances, two requests bearing the same authorization code can race: both Get succeeds before either Delete executes. This allows code replay, which PKCE alone doesn't prevent since the verifier is known to the attacker who initiated the flow.

Token response stored in plaintext at rest. The full upstream token response (access token, refresh token, etc.) is stored as a plain JSON string in Azure Table Storage inside StoredCode. While the window is short (~10 mins), this is sensitive credential material sitting unencrypted in a shared data store.

Medium
No rate limiting on any endpoint. /authorize creates store entries unconditionally. An attacker can flood the store with millions of sessions — a classic resource exhaustion DoS. No throttling exists on /token either, enabling brute-force attempts against authorization codes (though 32 bytes of randomness makes guessing impractical).

InMemory store never evicts expired entries. Sessions and codes that are never redeemed stay in memory forever. Only Pop (successful exchange) removes them. Over time this is an unbounded memory leak and a DoS vector.

IPv6 unique-local addresses not blocked in SSRF check. httputil.go:102-118 covers IPv4 RFC 1918, loopback, link-local, and IPv6 loopback/link-local — but misses the fc00::/7 unique-local address range (the IPv6 equivalent of RFC 1918). A client metadata URL resolving to an fd00:: address would bypass the SSRF filter.
