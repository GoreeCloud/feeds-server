# GoreeCloud Feeds Server Changelog

## Unreleased

### PostgreSQL storage schema Development tranche

- Added additive PostgreSQL 18-compatible migration `0001_initial.sql`.
- Added schema boundaries for feeds, subscriptions, canonical articles, source-scoped deduplication aliases, retrieval/source history, per-user article state, categories/tags, and media/image references.
- Added retention and preservation schema fields without claiming retention execution.
- Kept GoreeCloud Identity authoritative by storing external user references rather than local authentication secrets/accounts.
- Added static migration tests for required tables/boundaries, additive behavior, application-owned textual IDs, and content/user-state separation.
- Added `STORAGE.md` with database Role/Purpose, ownership, security, recovery, and current limitations.
- Kept PostgreSQL driver/connectivity, applied migrations, runtime writes/reads, credentials, backup/restore, deployment, and Stable qualification explicitly incomplete.


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
