# Security model

## Trust boundaries

The browser never receives the gateway service key. The server-side UI proxy authenticates the signed-in viewer and adds the service credential. The gateway derives tenant and role only from a hashed API-key lookup; tenant IDs in request bodies are ignored. The intelligence service is private and receives only approved tenant documents selected by the gateway.

Source material is untrusted data. It is normalized as text and never interpreted as instructions for the platform. Generated answers may use only retrieved approved records and must return citations. When evidence is missing, the provider returns an ungrounded response with no citations.

## Controls

- SHA-256 API-key storage, prefixes for identification, expiration, revocation, and last-use tracking
- RBAC roles: viewer, reviewer, admin, owner
- Tenant predicates in every repository query and transaction
- Maximum body sizes, strict JSON decoding, output encoding, CORS allowlist, and security headers
- Per-principal token-bucket rate limiting and upstream timeouts
- Idempotency keys for ingestion writes
- Immutable attribution and before/after state for privileged changes
- Private object storage with server-side encryption
- Operator-controlled connector hostname allowlists and a dedicated connector-secret namespace to prevent SSRF and credential exfiltration
- Non-root, read-only containers with dropped Linux capabilities
- Default-deny Kubernetes network policy and no service-account token mounting
- Dependency updates and CodeQL scanning in GitHub

## Credential rotation

Create a replacement API key, update the consumer, verify last-use moves to the new prefix, then revoke the old row. Rotate database and object credentials through the secret manager and perform a rolling restart. A suspected provider key leak requires immediate revocation at the provider and a review of audit and egress logs.

## Threats requiring platform controls

Deploy behind a managed WAF and DDoS service. Use managed database TLS and encryption. Restrict egress to the selected provider and object store. Configure centralized alerting and tamper-resistant log retention outside the application cluster. Run an external penetration test before handling regulated or highly sensitive material.
