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

The current HTTP runtime does not retrieve feeds, accept feed XML over an API, store user/feed/article data, or expose parser/deduplication output. Parser and deduplication functionality exist only as internal Development libraries in this tranche.

## Planned capability groups

- Feed subscription persistence and management
- Scheduled feed retrieval
- Feed parsing and normalization — **Partial / Development**
- Article processing and deduplication — **Partial / Development**
- Article and metadata storage
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
