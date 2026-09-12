# Sprint 2 implementation and evidence

Owner decision (2026-09-12, continuing the 2026-09-11 review): **Gate B APPROVED; Sprint 2 AUTHORIZED; implementation COMPLETE; Sprint 2 acceptance OPEN; Gates C/D CLOSED.** Live GET challenge and authentic real Meta POST proof are mandatory Sprint 2 acceptance evidence, not prerequisites for starting the endpoint implementation. Sprint 0/1 acceptance remains COMPLETE. No outbound WhatsApp or Sprint 3 work is authorized.

## Implemented scope

Migration 00003_meta_ingestion.sql adds meta_apps, wabas, phone_numbers, business_profiles, asset_sync_runs, webhook_events, webhook_facts, webhook_processing_attempts and webhook_replays. Every table has organization ownership, ENABLE/FORCE RLS and non-owner runtime access. Foreign keys include organization. Published 00001/00002 SQL and historical checksums remain unchanged.

One configured Meta App and target WABA are bound explicitly to the authenticated organization. Configuration is file-only for access token, App Secret and Verify Token; absent configuration disables Meta; partial configuration fails startup. Graph is pinned to reviewed v26.0 with verified TLS, timeout, no redirects, bounded responses, cursor pagination and sanitized failure classes. No Graph mutation method exists.

The API queues coalesced asset-sync jobs. Worker snapshots WABA, all phone pages, selected profile fields and current subscriptions; complete snapshots commit atomically. Absence marks NOT_OBSERVED, preserving records/profile history; failed snapshots do not delete assets. Concurrent workers use row claims, leases and completion fences. Eight attempts bound transient reads; authentication/permission/domain errors are terminal. An already subscribed App is never automatically resubscribed.

GET verifies exact single mode/token/challenge parameters with a constant-time token-digest comparison. POST rejects missing, malformed, wrong-algorithm, duplicate or wrong signatures using App Secret HMAC-SHA256 over raw bytes. JSON whitespace mutations invalidate an unchanged signature. After authentication, known asset routing and encrypted raw persistence precede ACK. Unknown/foreign asset envelopes enter restricted quarantine. Ordinary payload inspection never exposes message text, customer identifiers or arbitrary provider fields.

Raw transport digest is unique per org/App. Classification produces inbound-message and changed-status facts without full Inbox storage. Stable phone/message and status-fact hashes survive rebatching/replay. Unknown and insufficiently identified events explicitly retain uncertain identity. Replay references original evidence, adds a generation and audit row, preserves ciphertext, and reuses fact keys. Processing uses bounded retries, terminal DEAD_LETTER, permanent INVALID and safe UNKNOWN states. Raw evidence expires after seven days; expired/quarantined evidence cannot be replayed through ordinary tenant APIs.

Frontend routes: /meta-connection, /wabas, /phone-numbers, /business-profiles, /webhook-events and /meta-health. Views use current permissions, existing session/CSRF, scoped lists, actual sync/error/processing state and explicit audited redacted inspection/replay. No outbound or future-product menus.

## Executed local verification

- python scripts/dev.py up: PASS, real PostgreSQL running.
- Go unit tests including Graph success/error/auth/permission/rate/transport, pagination boundary/loop/redirect/size/version, typed configuration and raw signature/challenge tests: PASS.
- Real PostgreSQL Meta integration suite: PASS for sync concurrency/coalescing/idempotency, WABA/phone/profile/subscription persistence, temporary disappearance, failed sync preservation, GET/POST rejection, raw and semantic duplicates, unknown/quarantine/invalid classification, eight retries then DLQ, replay idempotency, encrypted raw recovery with organization-bound AAD, immutable evidence, CSRF/permission denial and isolation of all nine populated new tables.
- Source/migration/link validation: PASS at this stage.
- python scripts/dev.py check: PASS, including all Go unit/real PostgreSQL integration checks, go vet/gofmt, frontend lint/typecheck, five frontend tests, OpenAPI and production build. Final release check completed with exit 0.
- Identity browser E2E: PASS. Meta browser E2E: PASS, including six operational views, redacted payload, replay/sync, anonymous denial and mobile view. A missing permission-discovery entry found during browser verification was fixed, then the browser check passed.
- python scripts/dev.py migrate: applied 00003; repeated migration applied zero changes.
- go mod verify: PASS. govulncheck@v1.8.0: zero affected code/imported-package vulnerabilities; one required-module advisory outside reachable code remains disclosed. npm audit: zero findings.
- Hosted Sprint 2 CI: PASS for published implementation commit e0a65503e56727041a00899b9663878a86c8eb39 in run [34670353309](https://github.com/mbingsdk/whats/actions/runs/34670353309). The foundation job 103490430582 ran 2026-09-12T03:26:10Z to 03:28:27Z. Every step passed: isolated PostgreSQL, migration and repeat no-op, source/SQL/Go/real-PostgreSQL/frontend/OpenAPI checks, Identity and Meta Chromium E2E, dependency integrity/vulnerability checks and cleanup. Linux WABA_TEST_RACE=1 was enabled. This is Sprint 2 evidence, not reused Sprint 1 evidence.

## Live acceptance evidence

| Condition | Status |
| --- | --- |
| Company account GET inventory | PASS via the implemented client/worker; one WABA/phone/profile, subscription true, observed 2026-09-12 |
| Protected Verify Token | Configured in the existing protected file; runtime configuration PASS |
| Public HTTPS callback | Historical controlled probe PASS at 2026-09-12T03:17:53Z; valid TLS and unauthenticated rejection checks; listener expired and tunnel stopped, no active callback target |
| Real Meta GET challenge | PENDING |
| Real Meta signed POST, durable storage and classification | PENDING |
| Signature rejection tests | PASS locally |
| Hosted Sprint 2 commit | Published e0a65503e56727041a00899b9663878a86c8eb39; own run [34670353309](https://github.com/mbingsdk/whats/actions/runs/34670353309) PASS |
| Sprint 2 implementation | COMPLETE |
| Sprint 2 acceptance | OPEN |
| Gate C / D | CLOSED / CLOSED |

GET App subscriptions returned 200/empty at 2026-09-12T03:18:18Z. The existing WABA App subscription remains true. A concrete temporary callback, messages field and rollback-to-empty plan were presented to the owner; no target-specific approval was received before the listener expired. That proposal is no longer an active target. A new live target must be prepared and explicitly approved before any callback mutation; the earlier tunnel-only permission is not callback-configuration approval.

Callback configuration remains a separate target-specific operator decision. Before mutation record current callback/subscription state, proposed public target, explicit approval, audit and rollback. No callback/subscription mutation has been made by this implementation.

## Source limitations and phone state

Gate B observes SYSTEM_USER with intended App, WA/WM scopes, one WABA and CLOUD_API phone on v26.0. Expiry and the limited scope of those checks remain in [Gate B evidence](25-gate-b-evidence.md). No company-wide discovery claim is made.

The official fbsamples signature example was actually inspected; it demonstrates raw HMAC but conflates token naming and has a broken unconditional rejection branch. It is not copied as a complete contract. A second inspected first-party tech-provider contribution guide explicitly identifies App Secret HMAC and constant-time checking. Primary Graph webhook documentation remains inaccessible. See [research provenance](21-research-register.md).

code_verification_status=NOT_VERIFIED is preserved as observed. Official Postman demonstrates this field beside quality and display fields; the inaccessible primary phone reference prevents establishing exact current registration consequences. No request_code, verify_code, register or deregister is proposed or executed.

## Scope confirmation

No outbound WhatsApp message was sent. No profile/template mutation, phone registration, campaign, automation send or Sprint 3 implementation is included. The accepted Sprint 0/1 evidence and CI requirements remain intact.

The first controlled local listener ended when its database connection became unavailable; it had received zero webhook events and supplied no delivery evidence. The isolated listener was restarted at 2026-09-12T03:16:17Z and the live GET snapshot passed again. No failed local listener run is labeled real Meta acceptance.

The second controlled window ended normally after 90 minutes. Its final report at 2026-09-12T04:46:19Z records runtime_sync=SUCCEEDED, zero durable/processed events, and null last_valid_challenge_at/last_authentic_post_at. The harness PASS records a completed test window only; it is not real Meta acceptance. The remaining tunnel was stopped during the 2026-09-12 closure review. No provider-side rollback was needed because no Meta mutation occurred.
