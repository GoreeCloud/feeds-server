# GoreeCloud Feeds Server Notes

## Verified repository state

- Repository exists and uses main.
- Governed repository documentation foundation is established.
- Initial server toolchain is Go 1.27.1.
- Minimal Development HTTP runtime is implemented.
- Core normalized User, Subscription, Feed, Article, and ArticleState models are implemented.
- Dependency-free RSS 2.x and Atom 1.x parsing/normalization is implemented for caller-supplied XML with bounded parsing and recoverable warnings.
- Conservative source-scoped article deduplication is implemented in memory using identifiers, normalized URLs, and title/publication/content fingerprints, with ambiguous evidence retained rather than merged.
- ADR-0002 selects PostgreSQL 18 as the Development persistence target. Ordered migrations, pgx v5.11.0 bounded connectivity, transactional migration application, advisory locking, checksum-ledger verification, and durable core repository transactions are implemented and validated against PostgreSQL 18.6 in CI.
- A bounded internal remote-feed retrieval client is implemented with explicit destination policy, DNS/IP validation, redirect controls, conditional retrieval, resource limits, and no third-party runtime dependency.
- Development API contract is `0.1.0-dev` and owned by GoreeCloud/feeds-protocol.
- Implemented routes are `GET /api/v1/capabilities`, `GET /health/live`, and `GET /health/ready`.
- The internal repository layer persists external identity references, feeds, subscriptions, canonical articles, source-scoped deduplication/source-history records, and per-user article state. Categories/tags/media/images remain unsupported and fail closed. The network-visible server is not yet wired to PostgreSQL or the retrieval client. Retrieval scheduling/queueing/retry/history orchestration, authenticated product APIs, synchronization, package/deployment, and release remain incomplete.

## Governing relationships

- Product roadmap: governed GoreeCloud Feeds planning record.
- Project coordination repository: GoreeCloud/feeds.
- Shared protocol owner: GoreeCloud/feeds-protocol.
- Reusable shared-code repository: GoreeCloud/feeds-shared.
- Primary web client: GoreeCloud/feeds-web.

## Pending technical decisions

Schema support for article categories/tags/media/images and complete subscription settings; server-runtime DB configuration; search implementation; retrieval scheduling/queue/adaptive timing/retry/backoff/history orchestration; HTML sanitization policy; subscription ownership/persistence; authentication/session integration; packaging/containerization; deployment topology; backup/restore mechanics; and production network configuration.
