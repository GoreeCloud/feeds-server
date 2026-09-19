# GoreeCloud Feeds Server Specifications

## Status

Component: GoreeCloud Feeds Server  
Repository: GoreeCloud/feeds-server  
Component class: Server / service  
Lifecycle: Development  
Implementation status: Minimal Development HTTP runtime plus normalized feed/article model and RSS/Atom parser implemented; product features remain largely planned

This specification scopes the server responsibilities derived from the governing GoreeCloud Feeds product roadmap. The current implementation includes Development capability discovery, liveness/readiness endpoints, a normalized internal feed/article model, and dependency-free parsing of caller-supplied RSS 2.x and Atom 1.x XML.

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

The parser currently accepts caller-supplied UTF-8 XML only. It does not retrieve URLs. It normalizes RSS 2.x and Atom 1.x core feed/article fields; preserves recoverable partial records; records warnings for non-fatal normalization failures such as malformed dates; rejects DTD/entity directives; and applies XML depth/node complexity limits. HTML sanitization, remote retrieval, persistence, deduplication, subscription ownership, and authenticated API exposure are outside this tranche.

## Open decisions

Database engine, job scheduler, search backend, cache technology, authenticated parser/API exposure, remote retrieval strategy, HTML sanitization policy, authentication/session implementation, packaging format, container image, production port/hostname, and production topology remain unresolved.
