# GoreeCloud Feeds Storage

## Status

**Lifecycle:** Development schema foundation.

This record documents the first PostgreSQL persistence schema for GoreeCloud Feeds Server. It does not establish an operational database, database credentials, runtime connectivity, applied production migration, backup, restore, deployment, or Stable storage acceptance.

## Component

**Component:** PostgreSQL 18 relational datastore  
**Role:** Required application database once durable persistence is enabled  
**Purpose:** Store GoreeCloud Feeds canonical feed/article content, subscription relationships, user-specific article state, deduplication aliases, retrieval/source history, and article media/metadata.

PostgreSQL remains a Development architecture target until database connectivity is implemented and accepted.

## Ownership

GoreeCloud Feeds Server owns the database schema.

Clients and other GoreeCloud applications must not connect directly to the database. Supported cross-component access must occur through GoreeCloud Feeds APIs/protocols.

User identifiers are external identity references. The initial schema does not create a local authentication user table and does not store authentication secrets.

## Migration 0001

`migrations/0001_initial.sql` is an additive PostgreSQL 18-compatible schema foundation.

It creates:

- feeds;
- subscriptions;
- canonical articles;
- source-scoped article aliases used by conservative deduplication;
- article source/retrieval history;
- per-user article state;
- categories;
- tags;
- media references; and
- image references.

## Data separation

Canonical feed/article content is separated from user-owned subscription/article state.

This supports multiple users consuming the same content without sharing read, saved, favorite, preservation, or read-position state.

## Retention and preservation boundary

The initial schema provides:

- optional per-subscription `retention_days`; and
- `article_states.preserved_at` for a future explicit preservation workflow.

No deletion/retention worker exists yet. These fields establish schema support only; they do not constitute implemented retention behavior.

## Source history and deduplication

`article_aliases` stores source-scoped identifier, normalized-URL, and content-fingerprint aliases separately from canonical article rows.

`article_source_history` stores retrieval history, original source content, source identifiers/URLs, source format, and content fingerprint information without rewriting canonical article identity.

Fuzzy similarity and cross-source duplicate collapsing are still deferred.

## Security and privacy

The schema:

- does not contain passwords, authentication tokens, or reusable secrets;
- uses external textual user identity references;
- uses foreign keys and checks for referential/data integrity;
- is additive in the first migration; and
- does not define network exposure, hostnames, credentials, or privileged database roles.

Runtime database identities must use least privilege when connectivity is introduced.

## Backup and recovery

Once this schema is actually used to store durable Feeds data, database-aware backup and isolated restore validation become release blockers.

Everkeep recovery integration remains required before production acceptance where applicable.

## Current limitations

Not yet implemented:

- PostgreSQL Go driver;
- database connection pool;
- runtime migration executor;
- schema-version tracking in a live database;
- durable repository/query layer;
- actual writes/reads;
- retention cleanup;
- preserved-article workflow;
- backup/restore;
- database health/readiness integration;
- production credentials/network policy; and
- deployment.
