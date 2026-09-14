# Sprint 2 implementation and evidence

Acceptance review (2026-09-14; live evidence captured 2026-09-12): **Gate B APPROVED; Sprint 2 implementation COMPLETE; Sprint 2 acceptance COMPLETE; Gates C/D CLOSED.** Real Meta GET challenge, authentic signed POST, durable ingestion, processing, authorized Event Center and replay evidence are verified. Sprint 0/1 acceptance remains COMPLETE. No outbound WhatsApp or Sprint 3 work is authorized.

## Implemented scope

Migration 00003_meta_ingestion.sql adds meta_apps, wabas, phone_numbers, business_profiles, asset_sync_runs, webhook_events, webhook_facts, webhook_processing_attempts and webhook_replays. Every table has organization ownership, ENABLE/FORCE RLS and non-owner runtime access. Foreign keys include organization. Published 00001/00002 SQL and historical checksums remain unchanged.

One configured Meta App and target WABA are bound explicitly to the authenticated organization. Configuration is file-only for access token, App Secret and Verify Token; absent configuration disables Meta; partial configuration fails startup. Graph is pinned to reviewed v26.0 with verified TLS, timeout, no redirects, bounded responses, cursor pagination and sanitized failure classes. No Graph mutation method exists.

The API queues coalesced asset-sync jobs. Worker snapshots WABA, all phone pages, selected profile fields and current subscriptions; complete snapshots commit atomically. Absence marks NOT_OBSERVED, preserving records/profile history; failed snapshots do not delete assets. Concurrent workers use row claims, leases and completion fences. Eight attempts bound transient reads; authentication/permission/domain errors are terminal. An already subscribed App is never automatically resubscribed.

GET requires exactly one nonempty value for each mode/token/challenge parameter, with a constant-time token-digest comparison. Additional query metadata is ignored for authorization; malformed encoding and duplicate required keys remain rejected. POST rejects missing, malformed, wrong-algorithm, duplicate or wrong signatures using App Secret HMAC-SHA256 over raw bytes. JSON whitespace mutations invalidate an unchanged signature. After authentication, known asset routing and encrypted raw persistence precede ACK. Unknown/foreign asset envelopes enter restricted quarantine. Ordinary payload inspection never exposes message text, customer identifiers or arbitrary provider fields.

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

Evidence captured 2026-09-12; closure reviewed 2026-09-14. This is bounded verification on the supplied v26.0 App/WABA/phone, separate from official contract evidence quality and production readiness.

| Condition | Status |
| --- | --- |
| Company account GET inventory | PASS via implemented client/worker; supplied WABA/phone/profile and existing App subscription |
| Protected Verify Token | Configured; matched during real Meta verification; value not logged |
| Public HTTPS callback | TEMPORARY_ACCEPTANCE_TUNNEL; only the required webhook route exposed; approved callback rolled back and tunnel stopped |
| Real Meta GET challenge | VERIFIED, HTTP 200 and correct challenge, 2026-09-12T14:03:24.6444842Z |
| Real Meta signed POST | VERIFIED, HTTP 200, 2026-09-12T14:04:02.9038397Z |
| Durable ingestion | VERIFIED, exact raw body encrypted and committed before ACK |
| Processing | VERIFIED, one event and one INBOUND_MESSAGE fact, PROCESSED |
| Authorized Event Center | VERIFIED through existing UI/session/permissions; timestamp, context, attempts and redacted inspection |
| Duplicate and replay | VERIFIED, one event/fact retained; two authorized replays and two replay audit rows; original raw unchanged |
| Signature rejection tests | PASS locally and in hosted CI |
| Hosted correction commit | cd4d9d9be0678e91a945ea0b7fc915aa8444c661; own [run 34813315965](https://github.com/mbingsdk/whats/actions/runs/34813315965) PASS |
| No outbound WhatsApp | VERIFIED; only operator-originated inbound traffic and local captured-event replay |
| Sprint 2 implementation / acceptance | COMPLETE / COMPLETE |
| Gate C / D | CLOSED / CLOSED |

### Callback inventory, approval and rollback

Before mutation, GET App subscriptions returned HTTP 200 with an empty list at 2026-09-12T13:50:05Z. GET WABA subscribed_apps returned HTTP 200 at 13:50:06Z with one matching App subscription and no next page. The WABA-specific override field was NOT_RETURNED_BY_GET; absence of an override was not established. No existing App callback URL was observable.

The operator explicitly approved the exact temporary HTTPS target and CURRENT / PROPOSED / EFFECT / ROLLBACK plan: configure only App object whatsapp_business_account, field messages, then delete that newly created object callback to restore the observed empty App state. The disclosed effect included other subscribed WABAs without overrides; unknown assets remained quarantined. The exact temporary URL, approval and before/after operation audit are retained privately and are not permanent deployment configuration. The application runtime still has no Graph mutation adapter.

After evidence capture, the approved App-only DELETE succeeded at 2026-09-12T14:06:41.423281Z. Readback at 14:06:42.047692Z confirmed App subscriptions empty. WABA readback at 14:06:42.823388Z confirmed the target App still subscribed, with the override field still not returned. No WABA unsubscribe/resubscribe, phone registration, profile or template mutation occurred. Cleanup recorded at 14:10:35.752457Z confirms the tunnel and acceptance-only frontend stopped and ports 8092/8093 closed. The temporary callback is not left active.

### Real GET and narrowly required correction

Two approved configuration attempts at 13:56:48Z and 14:01:11Z failed with Graph HTTP 400/code 2200 after callback HTTP 403. The listener observed six query keys, including exactly one correct required mode/token/challenge value. The implementation incorrectly required exactly three total keys. A local diagnostic GET at 13:59:26Z passed but is explicitly LOCAL_DIAGNOSTIC_NOT_META_ACCEPTANCE.

The correction removes only the total-key-count restriction. Required keys remain single and nonempty, mode must be subscribe, token comparison remains constant-time, and challenge bounds/control-character and malformed-query rejection remain intact. Regression tests cover extra metadata plus duplicate required fields, wrong mode/token, missing challenge and malformed encoding. POST signature behavior is unchanged. No new product feature or migration was added.

The subsequent approved Graph configuration flow succeeded with HTTP 200 and matching readback. Its observed ingress GET at 14:03:24.6444842Z received subscribe, matched the protected token and returned the exact challenge with HTTP 200; the response write succeeded. Request ID: 01a095ee-1a4e-71f7-9019-770449582918. Correlation with the successful Meta configuration flow establishes provider provenance; the earlier local diagnostic is not substituted for it.

### Authentic POST, durable processing and Event Center

The operator confirmed one inbound message from their controlled WhatsApp account. At 14:04:02.9038397Z the public listener received X-Hub-Signature-256 and verified HMAC-SHA256 over the exact HTTP body bytes with the configured App Secret using hmac.Equal. The signature matched before acceptance. Encrypted raw evidence committed before HTTP 200 ACK; response writing succeeded. No decode/reencode was used for authenticity.

Event ID: 01a095ee-afc8-7e03-8963-a8f4771cef49. Request ID: 01a095ee-afc8-7617-900a-cdeb0c807b56. The worker produced one INBOUND_MESSAGE fact with WABA/phone context and final state PROCESSED. Decrypting the stored ciphertext with the actual protected root and organization/event AAD reproduced the captured raw bytes.

At 14:06:37.294Z, a browser check of the existing Event Center passed using an authenticated, authorized synthetic Owner in the isolated acceptance database. List/detail/payload API calls returned 200. The real event's timestamp, class, processing state, request identifier and processing generations 0/1/2 were visible; WABA/phone context was verified and masked in the screenshot. Ordinary payload inspection showed redacted values without the message body. This exercised the actual frontend and existing session/permission checks; no production user identity or authorization shortcut was added.

### Captured-event duplicate, replay and audit

Re-submitting the captured raw body and original signature locally returned HTTP 200 while retaining one durable event and one fact. Two authorized replay API calls returned HTTP 200, produced two append-only replay records and two replay audit rows, and completed processing generations 1 and 2. Fact count remained one; the original ciphertext and raw digest were unchanged. This is idempotent local processing, not a claim of exactly-once network delivery. No additional WhatsApp message was needed.

The controlled wrapper disabled raw request/query logging. A scan for all three configured Meta secret values in acceptance logs passed. The sanitized evidence bundle, masked Event Center screenshot, operation audit and encrypted original archive are retained under ignored local storage; neither raw plaintext nor credentials are published.

### Closure verification and review

On 2026-09-14, python scripts/dev.py up and python scripts/dev.py check completed successfully with exit 0 after starting Docker. The check executed source/link/migration validation, 12 Python tests, Go formatting/vet/unit/build and explicit real PostgreSQL integration tests, frontend lint/typecheck/five tests, OpenAPI validation and production build. The interrupted prior session's missing process exit was not treated as new completion evidence. Targeted challenge/signature and real PostgreSQL Meta regressions had also passed on 2026-09-12.

Corrective commit cd4d9d9be0678e91a945ea0b7fc915aa8444c661 passed the complete unchanged hosted workflow in [run 34813315965](https://github.com/mbingsdk/whats/actions/runs/34813315965), including Linux race-enabled integration tests, Identity/Meta browser E2E and dependency review. This run verifies the correction independently of the earlier e0a6550/1b15244 runs. Existing unrelated local bootstrap/operator-file changes are excluded from this correction and acceptance publication. Published migrations/checksums remain unchanged.

Review outcome: all requested live acceptance conditions are VERIFIED and Sprint 2 acceptance is COMPLETE under the owner's conditional closure instruction. Gate B remains APPROVED; Gates C/D remain CLOSED. No production deployment or Sprint 3 authorization follows.

### Historical earlier acceptance window

GET App subscriptions returned 200/empty at 2026-09-12T03:18:18Z. The existing WABA App subscription remains true. A concrete temporary callback, messages field and rollback-to-empty plan were presented to the owner; no target-specific approval was received before the listener expired. That proposal is no longer an active target. A new live target must be prepared and explicitly approved before any callback mutation; the earlier tunnel-only permission is not callback-configuration approval.

That earlier unapproved target/window is historical and is superseded by the separately approved successful flow above. Future callback mutations still require target-specific operator review and rollback.

## Source limitations and phone state

Gate B observes SYSTEM_USER with intended App, WA/WM scopes, one WABA and CLOUD_API phone on v26.0. Expiry and the limited scope of those checks remain in [Gate B evidence](25-gate-b-evidence.md). No company-wide discovery claim is made.

The official fbsamples signature example was actually inspected; it demonstrates raw HMAC but conflates token naming and has a broken unconditional rejection branch. It is not copied as a complete contract. A second inspected first-party tech-provider contribution guide explicitly identifies App Secret HMAC and constant-time checking. Primary Graph webhook documentation remains inaccessible. See [research provenance](21-research-register.md).

code_verification_status=NOT_VERIFIED is preserved as observed. Official Postman demonstrates this field beside quality and display fields; the inaccessible primary phone reference prevents establishing exact current registration consequences. No request_code, verify_code, register or deregister is proposed or executed.

## Scope confirmation

No outbound WhatsApp message was sent. No profile/template mutation, phone registration, campaign, automation send or Sprint 3 implementation is included. The accepted Sprint 0/1 evidence and CI requirements remain intact.

Historical unsuccessful windows: the first controlled local listener ended when its database connection became unavailable; it had received zero webhook events and supplied no delivery evidence. The isolated listener was restarted at 2026-09-12T03:16:17Z and the live GET snapshot passed again. No failed local listener run is labeled real Meta acceptance.

The second controlled window ended normally after 90 minutes. Its final report at 2026-09-12T04:46:19Z records runtime_sync=SUCCEEDED, zero durable/processed events, and null last_valid_challenge_at/last_authentic_post_at. The harness PASS records a completed test window only; it is not real Meta acceptance. The remaining tunnel was stopped during the 2026-09-12 closure review. No provider-side rollback was needed because no Meta mutation occurred.
