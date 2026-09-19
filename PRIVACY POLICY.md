# GoreeCloud Feeds Server Privacy Policy

## Current Development behavior

The current network-visible Development runtime remains deliberately non-data-bearing.

It:

- does not expose a network ingestion endpoint or retrieve remote feeds;
- does not store user, account, feed, or article content;
- does not authenticate users;
- does not use cookies or sessions;
- does not emit product telemetry;
- does not call third-party analytics services;
- does not make outbound feed requests;
- includes an internal parser that processes only caller-supplied XML in memory and does not persist or transmit parsed content;
- includes internal in-memory deduplication that processes only normalized article metadata/content supplied by the caller and does not persist or transmit it;
- contains a source-controlled PostgreSQL schema describing future durable data ownership, but no database connection, credentials, applied migration, or runtime persistence exists; and
- exposes only static non-sensitive capability metadata plus liveness/readiness responses.

The Development listener binds to loopback by default.

## Planned privacy requirements

Future implementation should keep account and feed data under the administrator's chosen self-hosted storage; avoid advertising, profiling, and mandatory third-party analytics; minimize unnecessary external requests; contact subscribed feed origins only as needed; avoid full-article retrieval unless enabled or requested; isolate users; avoid exposing feed/article content in diagnostics unless necessary and authorized; support deletion/export/backup/restore/migration; and integrate with GoreeCloud Privacy Shield before accepted production qualification where applicable.

Future runtime data flows, retention periods, external endpoints, cookies/tokens, logs, telemetry, and administrator controls must be documented from verified implementation before release.
