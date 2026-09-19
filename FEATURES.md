# GoreeCloud Feeds Server Features

## Current implemented state

Verified Development capabilities:

- Development HTTP server foundation.
- `GET /api/v1/capabilities` returning product, API version, Development protocol version, lifecycle, and optional capability list.
- `GET /health/live` liveness endpoint.
- `GET /health/ready` readiness endpoint for the current dependency-free Development tranche.
- Loopback-by-default Development listener with explicit alternate listen flag.
- Automated formatting, vet, and race-enabled unit-test workflow.
- Internal normalized models for User, Subscription, Feed, Article, ArticleState, source metadata, media, and images.
- Dependency-free RSS 2.x parsing and normalization for caller-supplied XML.
- Dependency-free Atom 1.x parsing and normalization for caller-supplied XML.
- Recoverable parser warnings for incomplete or malformed metadata where usable feed/article content remains.
- XML DTD/entity rejection plus parser depth/node complexity limits.
- Explicit separation of shared feed/article content from per-user article state.
- Conservative source-scoped article deduplication using exact identifiers, normalized URLs, and title/publication/content fingerprints.
- Conflict-safe deduplication behavior that retains candidates when independent evidence points at different canonical articles.
- Alias propagation so later publisher identifier/URL changes can still resolve to the already selected canonical article.
- PostgreSQL 18 schema/migration foundation with ordered embedded migration discovery and validation.
- Initial relational schema for users, feeds, subscriptions, articles, persistent deduplication identity keys, article source history, and independent per-user article state.
- Schema constraints for referential integrity, read-position range, SHA-256 fingerprint format, preserved user state, and source-scoped deduplication keys.
- Bounded pgx v5.11.0 PostgreSQL connection pool with explicit ping verification and application naming.
- Transactional migration application serialized by advisory transaction lock.
- Checksum/name-verified migration ledger that rejects drift in previously applied migration source.
- PostgreSQL 18.6 CI integration validation, including migration idempotence and schema/ledger readback.
- Durable PostgreSQL repository writes for external identity references, feeds, subscriptions, canonical articles, source-scoped deduplication keys, article source history, and per-user read/saved/favorite/preserved/read-position state.
- Transactional article persistence that rolls back the whole candidate when article identity, feed ownership, deduplication keys, or source-history identifiers conflict.
- Durable article/state readback for the currently supported core schema.
- Fail-closed rejection of categories, tags, media, and images until a later migration adds those storage structures rather than silently discarding them.

The current HTTP runtime does not retrieve feeds, accept feed XML over an API, require PostgreSQL, or expose parser/deduplication/storage operations. The internal storage package can connect, migrate, and durably persist/read the supported core records when explicitly invoked; CI validates these operations against an ephemeral PostgreSQL 18.6 service. Server-runtime database wiring remains incomplete.

## Planned capability groups

- Feed subscription persistence and management — **Partial / core relationship persistence only**
- Scheduled feed retrieval
- Feed parsing and normalization — **Partial / Development**
- Article processing and deduplication — **Partial / Development**
- Article and metadata storage — **Partial / Development durable core repository**
- Search
- Smart Feeds
- Rules engine
- User accounts and accepted GoreeCloud Identity integration
- Synchronization
- Notifications
- Media cache
- Feed health beyond process/readiness state
- Administration API
- Backup and restore
- Expanded versioned server API

Planned capability text must not be interpreted as current functionality.
