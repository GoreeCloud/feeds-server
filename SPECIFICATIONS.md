# GoreeCloud Feeds Server Specifications

## Status

Component: GoreeCloud Feeds Server  
Repository: GoreeCloud/feeds-server  
Component class: Server / service  
Lifecycle: Planned  
Implementation status: Repository documentation foundation only

This specification scopes the server responsibilities derived from the governing GoreeCloud Feeds product roadmap. It does not establish implementation.

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

## Open decisions

No language, web framework, database engine, job scheduler, search backend, cache technology, packaging format, container image, port, hostname, API wire format, or production topology has been selected by this foundation.
