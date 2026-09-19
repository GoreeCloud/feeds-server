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
- does not make outbound feed requests from the network-visible Development server runtime;
- includes an internal parser that processes only caller-supplied XML in memory and does not persist or transmit parsed content;
- includes internal in-memory deduplication that processes only normalized article metadata/content supplied by the caller and does not persist or transmit it;
- includes an internal retrieval client that can contact a configured feed origin only when explicitly invoked; it sends no user credentials, ignores environment proxies, does not follow HTTPS-to-HTTP downgrade redirects, strips ETag/Last-Modified plus ambient credential/referrer headers on cross-origin redirects, and uses bounded transient retries;
- bounds retrieval response size, time, redirect count, and concurrency so remote origins cannot cause unbounded resource consumption through the retrieval primitive;
- contains a PostgreSQL schema, connectivity/migration layer, and internal durable repository methods for the supported core records; the network-visible server runtime still does not invoke those methods;
- core persistence keeps shared feed/article content separate from user-owned subscription/article state;
- unsupported categories/tags/media/images are rejected rather than silently dropped;
- PostgreSQL integration tests use only synthetic CI data and ephemeral CI-only credentials; and
- exposes only static non-sensitive capability metadata plus liveness/readiness responses.

The Development listener binds to loopback by default.

## Planned privacy requirements

Future implementation should keep account and feed data under the administrator's chosen self-hosted storage; avoid advertising, profiling, and mandatory third-party analytics; minimize unnecessary external requests; contact subscribed feed origins only as needed and preserve the existing no-ambient-credential/proxy and cross-host validator-minimization boundaries; avoid full-article retrieval unless enabled or requested; isolate users; avoid exposing feed/article content in diagnostics unless necessary and authorized; support deletion/export/backup/restore/migration; and integrate with GoreeCloud Privacy Shield before accepted production qualification where applicable.

Future runtime data flows, retention periods, external endpoints, cookies/tokens, logs, telemetry, and administrator controls must be documented from verified implementation before release.
