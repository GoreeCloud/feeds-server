# GoreeCloud Feeds Server Dependencies

## Current Development dependencies

### PostgreSQL 18

**Role:** Application database.  
**Purpose:** Durable storage target for GoreeCloud Feeds canonical feed/article content, subscription relationships, deduplication identity/source history, and per-user article state.  
**Status:** Development dependency for integration validation; the network-visible server runtime does not yet require a database at startup.  
**Owner:** GoreeCloud Feeds Server owns the Feeds schema and migrations.  
**Version baseline:** PostgreSQL 18; Development CI currently validates against supported minor 18.6.  
**Failure impact:** Database-backed Feeds functions will be unavailable when introduced. Current capability/health-only Development runtime is not yet wired to PostgreSQL.  
**Security:** Runtime database identities must use least privilege. Connection strings/credentials are secrets and must not be committed or logged. Production transport/security policy remains a deployment qualification requirement.  
**Backup/recovery:** Database-aware backup and isolated restore validation are mandatory before durable Feeds data can qualify for release. Everkeep remains the required recovery integration where applicable.  
**Replacement boundary:** Clients never access PostgreSQL directly; storage stays behind Feeds Server APIs and repository boundaries.

### pgx v5.11.0

**Role:** PostgreSQL Go driver and connection-pool library.  
**Purpose:** Provide parameterized PostgreSQL connectivity, pooling, transactions, and low-level migration execution for the Go server.  
**Status:** Development runtime/build dependency beginning with the database-connectivity tranche.  
**License:** MIT.  
**Version:** 5.11.0, pinned in go.mod/go.sum.  
**Failure impact:** Database-backed Feeds functions cannot connect to PostgreSQL.  
**Security:** Connection configuration is supplied at runtime; connection strings are not embedded in source and are never intentionally included in Feeds errors/logs. Pool size and connect timeout are bounded by the Feeds storage package.  
**Replacement boundary:** pgx is an implementation dependency, not part of the GoreeCloud Feeds public protocol. PostgreSQL access remains isolated in internal/storage/postgres.

## Dependency minimization

No cache, queue, external search service, ORM, migration framework, hosted database, or additional database abstraction is introduced by this tranche.
