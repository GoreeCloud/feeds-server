# GoreeCloud Feeds Server User Manual

## Current availability

There is currently no supported GoreeCloud Feeds Server release to install or operate in production.

A minimal Development executable exists for engineering validation only. It exposes capability discovery and liveness/readiness endpoints. The repository also contains internal RSS/Atom normalization, feed/article models, conservative deduplication, PostgreSQL migration/connectivity code, and durable core repository transactions validated in CI. None of those storage capabilities are wired into the network-visible Development server yet. The server still does not retrieve subscribed feeds, expose durable product storage through an API/runtime workflow, authenticate users, synchronize clients, or provide a usable feed service.

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

No supported container image, production database configuration, production database credentials/TLS policy, full article metadata/media storage, complete subscription settings, authentication setup, network feed-ingestion configuration, backup/restore procedure, upgrade path, public network deployment, or Stable release exists yet.

## Intended audience

This manual currently supports developers validating the first server foundation. It will become the self-hosting administrator manual only after the necessary operational capabilities are implemented and accepted.
