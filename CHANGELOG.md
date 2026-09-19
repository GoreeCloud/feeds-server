# GoreeCloud Feeds Server Changelog

## Unreleased

### Durable PostgreSQL repository Development tranche

- Added parameterized repository operations for external user identity references, feeds, subscriptions, canonical articles, and per-user article state.
- Added transactional article writes that persist canonical article fields, source-scoped deduplication keys, and source-history records together.
- Added conflict guards that reject user-identity rebinding, article-ID/feed rebinding, deduplication-key reassignment, and source-history-ID reassignment.
- Added core article and article-state readback.
- Added fail-closed rejection of categories, tags, media, and images until a later schema migration can preserve them.
- Added PostgreSQL 18.6 integration tests for durable writes/readback, retrieval history, identity keys, rollback behavior, and independent user article state.
- Kept server-runtime database wiring, full subscription settings, metadata/media schema expansion, retention execution, backup/restore, authenticated APIs, deployment, and Stable qualification incomplete.


### PostgreSQL connectivity and migration-execution Development tranche

- Added pgx v5.11.0 as the pinned PostgreSQL Go dependency.
- Added a bounded connection pool with explicit connectivity verification and GoreeCloud Feeds application naming.
- Added transactional migration execution serialized by a PostgreSQL advisory transaction lock.
- Added a migration ledger that records version, filename, SHA-256 checksum, and application time and rejects applied-source drift.
- Added PostgreSQL 18.6 integration validation for connectivity, migration application, ledger integrity, and idempotence.
- Added module-lock verification to exact-candidate CI.
- Added DEPENDENCIES.md with PostgreSQL/pgx Role, Purpose, security, recovery, failure, ownership, and replacement boundaries.
- Kept server-runtime DB wiring, durable product repositories/data writes, production credentials/TLS policy, backup/restore, deployment, and Stable qualification incomplete.


### PostgreSQL schema/migration Development tranche

- Adopted the ADR-0002 PostgreSQL 18 Development persistence boundary.
- Added an additive version-controlled `0001_initial.sql` migration under the server-owned PostgreSQL storage package.
- Added initial relational tables for users, feeds, subscriptions, articles, persistent deduplication identity keys, article source history, and per-user article state.
- Preserved the shared-content/per-user-state separation and added explicit preserved-state/retention support.
- Added foreign keys, checks, and indexes for referential integrity, deduplication aliases, retrieval/source history, retention eligibility, unread state, and saved/favorite/preserved state.
- Added embedded migration discovery with strict ordered/contiguous filename validation.
- Added tests for migration order, required schema boundaries, user-state separation, and additive/extension-free initial migration behavior.
- Added no PostgreSQL driver, credentials, database connection, runtime migration executor, live persistence, deployment, or Stable claim.

### Article deduplication Development tranche

- Added dependency-free in-memory article deduplication.
- Added source-scoped identifier matching.
- Added conservative URL normalization and source-scoped normalized-URL matching.
- Added source-scoped SHA-256 title/publication/content fingerprints.
- Preserved the first article as canonical and registered duplicate aliases for later publisher identifier/URL changes.
- Added conflict-safe behavior that retains a candidate when independent evidence points to different canonical articles.
- Added tests for source identifiers, source isolation, URL normalization, metadata/content fingerprints, alias propagation, conflicting evidence, insufficient fingerprint evidence, and missing source scope.
- Kept fuzzy similarity, cross-source collapsing, persistence, durable indexes, network retrieval, authenticated API exposure, deployment, and Stable qualification explicitly incomplete.

### Feed model and parser Development tranche

- Added normalized User, Subscription, Feed, Article, ArticleState, media/image, and source models.
- Kept shared feed/article content separate from per-user article state.
- Added dependency-free RSS 2.x and Atom 1.x parsing using the Go standard library.
- Added recoverable warnings for incomplete/malformed metadata where usable content remains.
- Added XML DTD/entity rejection and depth/node complexity bounds.
- Added tests for RSS normalization, Atom normalization, partial records, malformed dates, unsafe directives, complexity limits, and user-state separation.
- Kept network retrieval, persistence, HTML sanitization, authenticated parser/API exposure, deployment, and Stable qualification explicitly incomplete.

### Initial Development runtime

- Selected Go 1.27.1 for the initial server toolchain.
- Added a dependency-free Go HTTP Development runtime.
- Added `GET /api/v1/capabilities` implementing protocol `0.1.0-dev`.
- Added liveness and readiness endpoints.
- Added exact-candidate CI for formatting, `go vet`, and race-enabled unit tests.
- Updated Platform Contract state from Planned to Development while retaining nonconformant release status.
- Kept persistence, ingestion, authentication, synchronization, packaging, deployment, and Stable qualification explicitly incomplete.

### Repository foundation

- Expanded the repository entry point.
- Added planned server specifications and scoped roadmap.
- Added mandatory repository documentation baseline.
- Added Platform Contract schema 0.4 declaration with product-specific integrations blocked pending implementation and acceptance.
- Recorded the current GoreeCloud fallback license as AGPL-3.0-or-later.
- Preserved Planned lifecycle and avoided implementation, release, deployment, production, or Stable claims.
