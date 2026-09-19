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

Still incomplete: subscription persistence/ownership enforcement, network feed retrieval, scheduling, retry/backoff, conditional requests, fetch history, runtime/durable deduplication indexes, database connectivity, HTML sanitization, authenticated ingestion/API surfaces, and production acceptance.

## Phase 2 — Persistence and processing

**Partial / verified schema foundation:** ADR-0002 selects PostgreSQL 18; `migrations/0001_initial.sql` defines the first additive relational schema for feeds, subscriptions, canonical articles, source-scoped deduplication aliases, retrieval/source history, user article state, categories/tags, and media/image references. Static tests verify required boundaries and reject destructive/row-order identity patterns in the initial migration.

Still incomplete: PostgreSQL driver/connectivity, runtime migration execution, transactions, durable read/write repositories, live schema-version tracking, retention execution, durable deduplication behavior, search indexing, feed health persistence, media-cache storage, backup/restore, and data migration qualification.

## Phase 3 — Protocol and synchronization

Implement the versioned server API coordinated through GoreeCloud/feeds-protocol, account-scoped synchronization, retry/offline recovery, and conflict/deletion/tombstone behavior.

## Phase 4 — Accounts, rules, Smart Feeds, notifications, and administration

Implement user isolation, rules, Smart Feeds, notification event generation, and authorized administration interfaces/APIs.

## Phase 5 — Platform integration, privacy, security, and recovery

Integrate and validate GoreeCloud Identity, Wardveil Security, Privacy Shield, Everkeep, Manager, Mesh where applicable, Policy, Observability, and supported Glaze UI-facing administration contracts.

## Phase 6 — Release qualification

Complete automated validation and verify migrations, backup/restore, failure recovery, security/privacy behavior, representative deployment, rollback, compatibility, and release provenance.

No phase is complete merely because it is documented here.
