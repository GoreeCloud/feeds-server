# GoreeCloud Feeds Server Security

## Current status

The repository is in Development. The current runtime is intentionally minimal and is **not** accepted as production-ready or Stable.

Current security-relevant properties verified by source/tests:

- the Development listener binds to loopback by default;
- the implemented endpoints expose only static non-sensitive capability metadata and health state;
- no user/feed/article data is stored by the runtime;
- parser input is caller-supplied in-memory XML only and is not exposed through the current HTTP surface;
- XML DTD/entity directives are rejected and XML depth/node complexity is bounded;
- deduplication is internal/in-memory and requires source-scoped exact evidence;
- ambiguous deduplication evidence fails safe by retaining the candidate rather than merging it;
- deduplication URL handling rejects credential-bearing URLs and does not make network requests;
- the PostgreSQL migration foundation contains schema only: no database credentials, active connection strings, database roles, or deployment secrets;
- the initial migration is additive and extension-free, and automated tests reject destructive table/schema/drop/truncate/delete operations in that migration;
- pgx v5.11.0 is pinned with module checksums and CI verifies the committed module lock;
- database connection strings are runtime inputs and are not intentionally included in Feeds errors or logs;
- the connection pool is bounded and uses a finite connect timeout;
- migration application uses a transaction-scoped advisory lock plus name/SHA-256 ledger verification to reject migration-source drift;
- PostgreSQL integration testing uses an ephemeral CI-only database and credentials, not production secrets;
- repository writes use parameterized queries and transaction boundaries rather than string-built SQL;
- user IDs cannot be silently rebound to a different external identity subject;
- article IDs cannot be silently rebound to another feed;
- source-scoped deduplication keys and source-history IDs cannot be silently reassigned to another article; conflicting article transactions roll back;
- unsupported article metadata collections are rejected rather than silently discarded;
- no authentication credentials, sessions, tokens, or secrets are implemented;
- the network-visible server does not invoke outbound feed retrieval;
- the internal retrieval client defaults to HTTPS and rejects URL-embedded credentials;
- private, loopback, link-local, carrier-grade NAT, documentation, benchmark, reserved, and other special-use destinations are denied unless an operator explicitly allowlists the network;
- every DNS answer for a retrieval hostname must satisfy destination policy before any connection is attempted, reducing DNS-rebinding and mixed-answer SSRF exposure;
- environment proxy inheritance is disabled for retrieval transport so local proxy configuration cannot silently redirect feed traffic or supply ambient credentials;
- redirect targets are revalidated on every hop, redirect count is bounded, HTTPS-to-HTTP downgrade redirects are blocked, and cross-host redirects drop ETag/Last-Modified validators;
- retrieval uses TLS 1.2 or later for HTTPS, bounded connect/request/header timeouts, bounded response-body size, and bounded concurrency;
- transient retrieval failures use a bounded configurable attempt count and exponential backoff; `Retry-After` is clamped to the configured maximum rather than trusted as an unbounded delay;
- permanent policy errors, oversized responses, blocked/downgrade redirects, and non-retryable HTTP failures fail without retry amplification;
- cross-origin redirects remove conditional validators, Authorization, Proxy-Authorization, Cookie, and Referer headers before continuation;
- JSON responses use `no-store` and `X-Content-Type-Options: nosniff`; and
- unsupported HTTP methods on the capability path are rejected by the method-specific router.

## Reporting security issues

Do not place credentials, private keys, active tokens, restricted exploit details, or private user data in public issues. Use an approved private GitHub security-reporting mechanism or another owner-approved private channel when confidential handling is required.

## Outstanding security boundaries

Before product functionality or production exposure, implementation must isolate users; authorize account-scoped operations; use accepted GoreeCloud Identity integration; integrate Wardveil Security; fail closed on authorization/security uncertainty; protect secrets outside source control; sanitize/render untrusted feed/article content safely; preserve the retrieval SSRF/redirect/destination controls through scheduler/runtime wiring and operator-configurable allowlists; preserve bounded retry/backoff behavior through scheduler/runtime wiring and define queue/adaptive-refresh/priority behavior; protect administrative interfaces; apply rate/resource controls; preserve privacy-safe security evidence; and validate backup/restore/migration without weakening access control.

A source commit, test pass, or deployment does not by itself establish production security acceptance.
