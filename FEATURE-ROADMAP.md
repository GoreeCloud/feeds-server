# GoreeCloud Feeds Server Feature Roadmap

## Status

Lifecycle: Development  
Implementation: Phase 0 toolchain/runtime foundation and bounded ingestion-processing work in progress; product feature phases remain incomplete.

The authoritative product capability scope is maintained in the governed GoreeCloud Feeds roadmap. This file scopes server-owned implementation work.

## Phase 0 — Repository and governance foundation

**Partial / verified:** repository documentation, Platform Contract declaration, licensing identity, Go 1.27.1 server toolchain, minimal Development HTTP runtime, protocol capability endpoint, liveness/readiness endpoints, and CI validation are established. Persistence, authentication, packaging/deployment, and remaining Phase 0 acceptance work are still incomplete.

## Phase 1 — Core ingestion foundation

**Partial / verified:** initial normalized User → Subscription → Feed → Article → ArticleState model is implemented, with feed/article content separated from per-user ArticleState. Dependency-free RSS 2.x and Atom 1.x parsing/normalization is implemented for caller-supplied XML with recoverable warnings and bounded XML parsing.

**Partial / verified:** conservative internal article deduplication now recognizes duplicates only when source-scoped exact evidence agrees: source identifiers, conservatively normalized URLs, or title/publication/content fingerprints. Conflicting evidence fails safe by retaining the article. Fuzzy similarity and cross-source collapsing remain deferred.

**Partial / verified:** a bounded internal remote-feed retrieval transport is implemented with HTTPS-by-default policy, explicit private-network allowlisting, DNS/IP destination checks, proxy isolation, redirect revalidation, HTTPS-downgrade blocking, conditional ETag/Last-Modified requests, body/time limits, and bounded concurrency.

Still incomplete: full per-feed settings/history/folder/tag persistence, retrieval scheduling/queueing, adaptive refresh, priority refresh, retry/backoff, fetch-history and last-success persistence, end-to-end retrieval→parse→deduplicate→persist orchestration, HTML sanitization, authenticated ingestion/API surfaces, and production acceptance.

## Phase 2 — Persistence and processing

**Partial / verified:** PostgreSQL 18 schema/migration source is version-controlled. A pgx v5.11.0 storage boundary provides bounded connectivity, ping verification, transactional migration application, advisory-lock serialization, and a checksum-verified migration ledger. The internal repository layer now durably persists external identity references, feeds, subscriptions, canonical articles, source-scoped deduplication keys, source history, and per-user article state. Article persistence is transactional and rejects conflicting identity/feed/deduplication/source-history bindings rather than reassigning them. Exact-candidate CI validates these operations against PostgreSQL 18.6.

Still incomplete: server-runtime database wiring, article categories/tags/media/images, complete subscription settings/history, retention execution, search indexing, feed-health persistence, media-cache storage, database-aware backup/restore, and production migration/upgrade qualification.

## Phase 3 — Protocol and synchronization

Implement the versioned server API coordinated through GoreeCloud/feeds-protocol, account-scoped synchronization, retry/offline recovery, and conflict/deletion/tombstone behavior.

## Phase 4 — Accounts, rules, Smart Feeds, notifications, and administration

Implement user isolation, rules, Smart Feeds, notification event generation, and authorized administration interfaces/APIs.

## Phase 5 — Platform integration, privacy, security, and recovery

Integrate and validate GoreeCloud Identity, Wardveil Security, Privacy Shield, Everkeep, Manager, Mesh where applicable, Policy, Observability, and supported Glaze UI-facing administration contracts.

## Phase 6 — Release qualification

Complete automated validation and verify migrations, backup/restore, failure recovery, security/privacy behavior, representative deployment, rollback, compatibility, and release provenance.

No phase is complete merely because it is documented here.
