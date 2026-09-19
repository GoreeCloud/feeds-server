# GoreeCloud Feeds Server Specifications

## Status

Component: GoreeCloud Feeds Server  
Repository: GoreeCloud/feeds-server  
Component class: Server / service  
Lifecycle: Development  
Implementation status: Minimal Development HTTP runtime, feed/article model, RSS/Atom parser, conservative in-memory deduplication, bounded remote-feed retrieval transport, PostgreSQL schema/migrations, PostgreSQL connectivity/migration application, and durable core repository transactions implemented; network/runtime product wiring remains incomplete

This specification scopes the server responsibilities derived from the governing GoreeCloud Feeds product roadmap. The current implementation includes Development capability discovery, liveness/readiness endpoints, normalized feed/article models, bounded RSS/Atom parsing, conservative in-memory article deduplication, a bounded remote-feed retrieval transport, ordered PostgreSQL 18 migrations, an internal pgx-based database connectivity/migration layer, and durable core PostgreSQL repository transactions.

## Authority model

GoreeCloud Feeds Server is intended to be authoritative for server-side feed ingestion, normalized feed/article processing, account-scoped server data, search indexes, synchronization state, feed health, server configuration, and server API behavior.

Clients must not become independent authorities for server-owned feed state.

## Planned server capabilities

- Feed subscription persistence and server-side management.
- Scheduled feed retrieval with conditional retrieval, retry, and backoff behavior.
- Feed parsing and normalization into stable internal models.
- Article processing and conservative deduplication.
- Feed/article and account-scoped state persistence.
- Search over authorized feed and article data.
- Rules engine and Smart Feeds.
- Notification event generation for configured conditions.
- Media caching according to storage, privacy, retention, and recovery rules.
- Isolated multi-user accounts.
- Versioned synchronization with conflict, deletion/tombstone, retry, and interrupted-sync behavior.
- Authorized administration and feed-health interfaces.
- Backup, restore, export, and migration.
- A versioned server API coordinated through GoreeCloud/feeds-protocol.

## Privacy requirements

The planned product is self-hosted and privacy-focused. Server behavior should minimize external disclosure, avoid advertising/profiling dependencies, and avoid fetching full external article content except where the user has enabled or requested that behavior.

## Security requirements

Authorization must isolate users. Sensitive operations must fail safely. Reusable credentials and protected secrets must remain outside ordinary source control and documentation.

## Current implementation decision

The server runtime uses Go 1.27.1 and the Go standard library. HTTP behavior uses `net/http`; feed parsing uses `encoding/xml` without a new third-party runtime dependency. The shared Development API contract uses versioned HTTP/JSON described by OpenAPI 3.1 in GoreeCloud/feeds-protocol. The current Development listener defaults to loopback `127.0.0.1:8080`.

The current normalized model introduces User, Subscription, Feed, Article, ArticleState, media/image metadata, and source metadata. Article content and per-user ArticleState are separate types so multiple users can consume the same feed/article content without sharing read/saved/favorite/read-position state.

The parser currently accepts caller-supplied UTF-8 XML only. It does not retrieve URLs itself. It normalizes RSS 2.x and Atom 1.x core feed/article fields; preserves recoverable partial records; records warnings for non-fatal normalization failures such as malformed dates; rejects DTD/entity directives; and applies XML depth/node complexity limits. HTML sanitization, persistence, subscription ownership, and authenticated API exposure remain outside the parser boundary.

The retrieval implementation is an internal transport primitive rather than a scheduler or ingestion service. It defaults to HTTPS and rejects embedded credentials. Private, loopback, link-local, carrier-grade NAT, documentation, benchmark, reserved, and other non-public/special-use destinations are denied unless an operator explicitly allowlists a network. DNS resolution is performed before dialing and every returned address must satisfy the destination policy; the dialer then connects to a validated address directly. Environment proxy inheritance is disabled. Redirect targets are revalidated, HTTPS-to-HTTP downgrade redirects are blocked, redirect count is bounded, and ETag/Last-Modified validators are stripped when a redirect changes host. Requests support conditional retrieval and use bounded request/connect/header timeouts, body size, and concurrency. Transient request failures and HTTP 408/425/429/500/502/503/504 responses use a configurable bounded attempt count with exponential backoff; `Retry-After` is honored only within the configured maximum backoff. Permanent policy errors, oversized responses, redirect-policy failures, and permanent HTTP failures are not retried. Scheduler queues, adaptive timing, priority, fetch-history persistence, last-successful retrieval persistence, parser/storage pipeline integration, and runtime/operator configuration are still separate tranches.

The current deduplication implementation is internal and in-memory. It requires a source scope derived from FeedID or Source.URL and uses only exact, explainable signals: source-scoped identifiers, conservatively normalized URLs, and a SHA-256 fingerprint over normalized title, author, publication time, and content/summary. URL normalization lowercases scheme/host, removes fragments and default ports, preserves path/query semantics, and deterministically orders query parameters. Duplicate aliases are registered to the first canonical article so later identifier/URL changes can still resolve. If separate signals point to different canonical articles, no merge occurs. Fuzzy content similarity, cross-source collapsing, and merge/reconciliation of conflicting article bodies remain outside this tranche. The initial PostgreSQL migration defines durable identity-key/source-history structures. The internal storage package uses pgx v5.11.0 to open a bounded pool, verify connectivity, serialize migration application with a transaction-scoped advisory lock, and record migration name/SHA-256 checksums in a schema ledger. The repository layer persists external identity references, feeds, subscriptions, canonical article fields, source-scoped identity keys, source-history records, and per-user article state. Article persistence is one transaction and fails on article-ID/feed rebinding, deduplication-key reassignment, or source-history-ID reassignment. User IDs cannot be silently rebound to another external identity subject. Categories, tags, media, and images are rejected until a later schema migration supports them so data is not silently lost. CI validates these operations against PostgreSQL 18.6. The network-visible server process still does not open PostgreSQL.

## Open decisions

Schema/migration design for article categories/tags/media/images and complete subscription settings, retention execution, server startup/configuration wiring for PostgreSQL, production TLS/credential injection, retrieval scheduler/queue/adaptive refresh/priority/history design, search backend, cache technology, authenticated ingestion/API exposure, HTML sanitization policy, authentication/session implementation, packaging format, container image, production port/hostname, and production topology remain unresolved.
