# GoreeCloud Feeds Server

GoreeCloud Feeds Server is the authoritative back-end project for GoreeCloud Feeds.

## Current state

**Lifecycle:** Development.

The repository contains a minimal verified Development runtime written in Go plus internal normalized feed/article models, RSS/Atom parsing, conservative article deduplication, and a version-controlled PostgreSQL schema/migration foundation. The network-visible runtime surface remains intentionally limited to:

- `GET /api/v1/capabilities` — non-sensitive Development protocol/capability metadata;
- `GET /health/live` — process liveness;
- `GET /health/ready` — readiness for the current dependency-free Development tranche.

Feed parsing and normalization are partially implemented as an internal, dependency-free library for caller-supplied XML. The parser supports RSS 2.x and Atom 1.x normalization, recoverable warnings for incomplete metadata, XML complexity limits, and explicit separation between shared article content and per-user article state.

Article deduplication is also partially implemented as an internal dependency-free library. The current tranche preserves the first article as canonical and recognizes duplicates only when exact source-scoped evidence agrees: normalized source identifiers, conservatively normalized URLs, or a title/publication/content fingerprint. If independent evidence points to different canonical articles, the candidate is retained rather than merged. Fuzzy similarity and cross-source collapsing are intentionally deferred.

ADR-0002 selects PostgreSQL 18 for durable persistence. The repository contains ordered embedded SQL migrations and now also includes a bounded pgx v5.11.0 connection-pool/migration layer. Migrations are serialized with a PostgreSQL advisory transaction lock and recorded in a checksum-verified schema ledger. This storage package is integration-tested against PostgreSQL 18.6, but the network-visible Feeds Server runtime is not yet wired to require or use a database.

Durable feed/article repository queries and writes, feed subscription runtime persistence, network retrieval, search, synchronization, accounts/authentication, administration, notifications, backup/restore, packaging, deployment, Release Candidate, production acceptance, and Stable release remain **not** implemented.

## Development toolchain

- Go 1.27.1
- Go standard-library `net/http`
- Go standard-library `encoding/xml` for the current RSS/Atom parser tranche
- Go standard-library cryptographic hashing and URL handling for conservative deduplication
- Go standard-library testing
- PostgreSQL 18 Development schema target with ordered SQL migrations
- pgx v5.11.0 for PostgreSQL connectivity and bounded pooling
- GitHub Actions validation on exact pull-request candidates
- Development API contract version `0.1.0-dev` owned by GoreeCloud/feeds-protocol

pgx v5.11.0 is the only new Go runtime dependency in the connectivity tranche. PostgreSQL 18.6 is exercised in CI, while the current network-visible Development server still runs without a database because durable repository wiring is a later tranche.

## Development run

From the repository root:

```sh
go run ./cmd/feeds-server
```

The Development runtime binds to `127.0.0.1:8080` by default. An alternate development address may be supplied with `-listen`.

Example non-sensitive checks:

```sh
curl http://127.0.0.1:8080/health/live
curl http://127.0.0.1:8080/health/ready
curl http://127.0.0.1:8080/api/v1/capabilities
```

These commands are Development entry points, not production deployment instructions.

## Planned responsibilities

The server is intended to own feed retrieval and scheduling, parsing and normalization, article processing and deduplication, storage, search, rules and Smart Feeds, notifications, media caching, user accounts, synchronization, administration, feed health, backup and restore, and the server API.

The server remains intended to be authoritative for server-side feed ingestion, processed article state, account-scoped server data, and synchronization state exposed to clients through the shared GoreeCloud Feeds protocol.

## Repository relationships

- GoreeCloud/feeds — project-wide coordination and roadmap.
- GoreeCloud/feeds-protocol — shared client/server contract authority.
- GoreeCloud/feeds-shared — reusable implementation only when genuine reuse exists.
- GoreeCloud/feeds-web — Glaze UI web client.

## Documentation

See SPECIFICATIONS.md, FEATURES.md, FEATURE-ROADMAP.md, DEPENDENCIES.md, BENEFITS.md, COMPETITIVE-OBJECTIVES.md, BRANDING.md, USER-MANUAL.md, PRIVACY POLICY.md, SECURITY.md, NOTES.md, and CHANGELOG.md.

## Platform integration

The repository declares GoreeCloud Platform Contract schema 0.4 in goreecloud.platform.yaml. Product-specific platform integrations remain blocked until implementation and acceptance evidence exists.

## License

This repository currently uses the GoreeCloud fallback software license: GNU Affero General Public License v3.0 or later (AGPL-3.0-or-later), pending any later controlled project-specific licensing decision. See LICENSE.
