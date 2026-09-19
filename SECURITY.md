# GoreeCloud Feeds Server Security

## Current status

This repository is in Planned / repository-foundation state. No server runtime is currently accepted as secure, deployable, production-ready, or Stable.

## Reporting security issues

Do not place credentials, private keys, active tokens, restricted exploit details, or private user data in public issues. Use an approved private GitHub security-reporting mechanism or another owner-approved private channel when confidential handling is required.

## Planned security boundaries

Future implementation must isolate users; authorize account-scoped operations; use GoreeCloud Identity for accepted identity/authentication; integrate Wardveil Security before production security acceptance; fail closed on authorization/security uncertainty; protect secrets outside source control; safely process untrusted feed/article content; limit outbound retrieval; protect administrative interfaces; apply network-service rate/resource controls; preserve security-relevant evidence without leaking user content; and validate backup/restore/migration without weakening access control.

A source commit, test pass, or deployment does not by itself establish security acceptance.
