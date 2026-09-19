# GoreeCloud Feeds Server Feature Roadmap

## Status

Lifecycle: Development  
Implementation: Phase 0 toolchain/runtime foundation in progress; product feature phases remain incomplete.

The authoritative product capability scope is maintained in the governed GoreeCloud Feeds roadmap. This file scopes server-owned implementation work.

## Phase 0 — Repository and governance foundation

**Partial / verified:** repository documentation, Platform Contract declaration, licensing identity, Go 1.27.1 server toolchain, minimal Development HTTP runtime, protocol capability endpoint, liveness/readiness endpoints, and CI validation are established. Persistence, authentication, packaging/deployment, and remaining Phase 0 acceptance work are still incomplete.

## Phase 1 — Core ingestion foundation

Implement feed subscription models, retrieval scheduling, retry/backoff behavior, parsing/normalization, and initial feed/article models.

## Phase 2 — Persistence and processing

Implement article/feed persistence, deduplication, search indexing, feed health, media-cache boundaries, and data migration/versioning.

## Phase 3 — Protocol and synchronization

Implement the versioned server API coordinated through GoreeCloud/feeds-protocol, account-scoped synchronization, retry/offline recovery, and conflict/deletion/tombstone behavior.

## Phase 4 — Accounts, rules, Smart Feeds, notifications, and administration

Implement user isolation, rules, Smart Feeds, notification event generation, and authorized administration interfaces/APIs.

## Phase 5 — Platform integration, privacy, security, and recovery

Integrate and validate GoreeCloud Identity, Wardveil Security, Privacy Shield, Everkeep, Manager, Mesh where applicable, Policy, Observability, and supported Glaze UI-facing administration contracts.

## Phase 6 — Release qualification

Complete automated validation and verify migrations, backup/restore, failure recovery, security/privacy behavior, representative deployment, rollback, compatibility, and release provenance.

No phase is complete merely because it is documented here.
