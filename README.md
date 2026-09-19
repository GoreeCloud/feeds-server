# GoreeCloud Feeds Server

GoreeCloud Feeds Server is the planned authoritative back end for GoreeCloud Feeds.

## Current state

**Lifecycle:** Planned / repository foundation.

This repository currently establishes project documentation only. It does not yet contain a verified server runtime, API implementation, database schema, feed scheduler, parser, search engine, synchronization engine, packaged artifact, deployment, Release Candidate, production deployment, or Stable release.

## Planned responsibilities

The server is intended to own feed retrieval and scheduling, parsing and normalization, article processing and deduplication, storage, search, rules and Smart Feeds, notifications, media caching, user accounts, synchronization, administration, feed health, backup and restore, and the server API.

The server is intended to remain authoritative for server-side feed ingestion, processed article state, account-scoped server data, and synchronization state exposed to clients through the shared GoreeCloud Feeds protocol.

## Repository relationships

- GoreeCloud/feeds — project-wide coordination and roadmap.
- GoreeCloud/feeds-protocol — planned shared client/server contracts.
- GoreeCloud/feeds-shared — planned genuinely reusable internal code.
- GoreeCloud/feeds-web — planned Glaze UI web client.

## Documentation

See SPECIFICATIONS.md, FEATURES.md, FEATURE-ROADMAP.md, BENEFITS.md, COMPETITIVE-OBJECTIVES.md, BRANDING.md, USER-MANUAL.md, PRIVACY POLICY.md, SECURITY.md, NOTES.md, and CHANGELOG.md.

## Platform integration

The repository declares GoreeCloud Platform Contract schema 0.4 in goreecloud.platform.yaml. All product-specific platform integrations remain blocked/planned until implementation and acceptance evidence exists.

## License

This repository currently uses the GoreeCloud fallback software license: GNU Affero General Public License v3.0 or later (AGPL-3.0-or-later), pending any later controlled project-specific licensing decision. See LICENSE.
