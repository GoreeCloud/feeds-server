# GoreeCloud Feeds Server Notes

## Verified repository state

- Repository exists and uses main.
- Governed repository documentation foundation is established.
- Initial server toolchain is Go 1.27.1.
- Minimal Development HTTP runtime is implemented.
- Core normalized User, Subscription, Feed, Article, and ArticleState models are implemented.
- Dependency-free RSS 2.x and Atom 1.x parsing/normalization is implemented for caller-supplied XML with bounded parsing and recoverable warnings.
- Conservative source-scoped article deduplication is implemented in memory using identifiers, normalized URLs, and title/publication/content fingerprints, with ambiguous evidence retained rather than merged.
- Development API contract is `0.1.0-dev` and owned by GoreeCloud/feeds-protocol.
- Implemented routes are `GET /api/v1/capabilities`, `GET /health/live`, and `GET /health/ready`.
- No persistent datastore, network feed retrieval/ingestion service, authenticated parser/API exposure, synchronization, package, deployment, or release exists.

## Governing relationships

- Product roadmap: governed GoreeCloud Feeds planning record.
- Project coordination repository: GoreeCloud/feeds.
- Shared protocol owner: GoreeCloud/feeds-protocol.
- Reusable shared-code repository: GoreeCloud/feeds-shared.
- Primary web client: GoreeCloud/feeds-web.

## Pending technical decisions

Persistent database/storage and durable deduplication model; search implementation; scheduling/background jobs; remote retrieval strategy; HTML sanitization policy; subscription ownership/persistence; authentication/session integration; packaging/containerization; deployment topology; backup/restore mechanics; and production network configuration.
