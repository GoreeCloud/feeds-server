# GoreeCloud Feeds Server User Manual

## Current availability

There is currently no supported GoreeCloud Feeds Server release to install or operate in production.

A minimal Development executable exists for engineering validation only. It exposes capability discovery and liveness/readiness endpoints. The repository also contains internal RSS/Atom normalization, core feed/article models, conservative article deduplication, and a PostgreSQL 18 schema/migration foundation, but no user-facing or network ingestion interface exposes them. No database is connected or applied by the runtime. The server still does not retrieve subscribed feeds, durably store user data, authenticate users, synchronize clients, or provide a usable feed service.

## Development-only run

With Go 1.27.1:

```sh
go run ./cmd/feeds-server
```

The runtime binds to `127.0.0.1:8080` by default. This loopback Development listener is not a production deployment configuration.

Available Development checks:

```sh
curl http://127.0.0.1:8080/health/live
curl http://127.0.0.1:8080/health/ready
curl http://127.0.0.1:8080/api/v1/capabilities
```

## Unsupported operational areas

No supported container image, connected persistent database, production configuration schema, authentication setup, network feed-ingestion configuration, runtime migration procedure, backup/restore procedure, upgrade path, public network deployment, or Stable release exists yet.

## Intended audience

This manual currently supports developers validating the first server foundation. It will become the self-hosting administrator manual only after the necessary operational capabilities are implemented and accepted.
