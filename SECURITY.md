# GoreeCloud Feeds Server Security

## Current status

The repository is in Development. The current runtime is intentionally minimal and is **not** accepted as production-ready or Stable.

Current security-relevant properties verified by source/tests:

- the Development listener binds to loopback by default;
- the implemented endpoints expose only static non-sensitive capability metadata and health state;
- no user/feed/article data is stored by the runtime;
- parser input is caller-supplied in-memory XML only and is not exposed through the current HTTP surface;
- XML DTD/entity directives are rejected and XML depth/node complexity is bounded;
- deduplication is internal/in-memory and requires source-scoped exact evidence;
- ambiguous deduplication evidence fails safe by retaining the candidate rather than merging it;
- deduplication URL handling rejects credential-bearing URLs and does not make network requests;
- no authentication credentials, sessions, tokens, or secrets are implemented;
- no outbound feed retrieval occurs;
- JSON responses use `no-store` and `X-Content-Type-Options: nosniff`; and
- unsupported HTTP methods on the capability path are rejected by the method-specific router.

## Reporting security issues

Do not place credentials, private keys, active tokens, restricted exploit details, or private user data in public issues. Use an approved private GitHub security-reporting mechanism or another owner-approved private channel when confidential handling is required.

## Outstanding security boundaries

Before product functionality or production exposure, implementation must isolate users; authorize account-scoped operations; use accepted GoreeCloud Identity integration; integrate Wardveil Security; fail closed on authorization/security uncertainty; protect secrets outside source control; sanitize/render untrusted feed/article content safely; constrain outbound retrieval against SSRF and unsafe redirect/destination behavior; protect administrative interfaces; apply rate/resource controls; preserve privacy-safe security evidence; and validate backup/restore/migration without weakening access control.

A source commit, test pass, or deployment does not by itself establish production security acceptance.
