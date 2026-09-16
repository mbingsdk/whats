# Sprint 3 evidence: guarded Inbox, message domain and outbound safety

## Current acceptance decision, 2026-09-16

Sprint 3 implementation remains COMPLETE; acceptance OPEN. Gates A/B/C APPROVED; Gate D CLOSED; Sprint 4 unauthorized. M73 establishes ordinary Service delivery TTL = 30 days. The unresolved dispatch requirement is complete executable pricing coverage across that interval, including October's effective-date transition. The stable denial is PRICING_HORIZON_NOT_FULLY_COVERED.

The operator reports IDR directly from Meta Billing (OPERATOR_REVIEWED_META_BILLING; no independent authenticated Billing inspection). The operator designated one controlled TEST recipient through a protected local file and explicitly confirmed market ID (Indonesia). No full number, billing identifier or credentials are committed. These facts do not write runtime currency state, publish rates or authorize spending.

The owner permits preparation for one bounded-cost TEST Service TEXT acceptance, conditional on complete reviewed Registry coverage and explicit approval of its calculated maximum. No maximum or financial approval exists yet. General paid/template, outbound media, campaigns and automation remain CLOSED. No live provider request or callback mutation has occurred in this review. Historical zero-cost-only and currency-UNKNOWN decisions below are superseded for this acceptance strategy; their evidence remains historical.

Historical owner clarification (2026-09-15): Sprints 0/1/2 COMPLETE; Gates A/B/C APPROVED; Sprint 3 AUTHORIZED; Gate D CLOSED. Company Meta billing currency UNKNOWN. Paid/template authority CLOSED. No currency inference from country, timezone, rate artifacts or locale.

## Status

Implementation: COMPLETE. Local full repository checks and Sprint 3's own hosted CI PASS. Live acceptance: OPEN. No Sprint 3 real outbound Meta request or callback mutation has been performed. No Sprint 4, campaign, automation, CRM import, template mutation/send or outbound media is included.

## Baseline before implementation

Gate C research commit 4bc49539745ead1f3b5e53470e7aa77671ab29cc passed [run 34840457519](https://github.com/mbingsdk/whats/actions/runs/34840457519). Owner-clarification baseline efe31864e5ed0c090d12c292f269eec22941bfb6 passed [run 34879123286](https://github.com/mbingsdk/whats/actions/runs/34879123286), all repository/PostgreSQL/browser/dependency steps (2m39s). This result was recorded before Sprint 3 code changes. Neither run substitutes for Sprint 3's own hosted result.

## Implemented scope

Migration 00004_guarded_inbox.sql adds 21 tenant tables and 12 permissions; readiness version is 4. Prior migrations and historical manifest entries are unchanged. All new tables FORCE RLS and use scoped composite references. The [database record](05-database-design.md#implemented-sprint-3-schema) lists the complete schema.

The message domain classifies TEXT, IMAGE, VIDEO, AUDIO, DOCUMENT, STICKER, CONTACTS, LOCATION, REACTION, INTERACTIVE, TEMPLATE and UNKNOWN independently from raw Graph JSON. Provider IDs/timestamps/context, source linkage, processing/error and delivery state remain separate. WABA/phone/conversation/recipient scope is retained, and outbound preflight additionally freezes a scope snapshot and original window evidence. Media displays metadata only.

Materialization independently consumes already-processed authentic Sprint 2 events and records its own completion. Duplicate source/provider facts cannot duplicate conversation, message or unread increment. Older inbound never moves the eligible basis backwards. Only valid authenticated TEXT opens/extends the conservative 24-hour window. Outbound, notes, statuses and replay processing time do not reset it.

Inbox supports scoped list/search/filtering, message history pages, current sender, priority/status, assignment, resolve/reopen/snooze, member read watermark, internal notes/edit/redaction/mentions, advisory presence and server-time window display. Two authorized humans may submit independent intents within one handoff epoch. The one-provider-attempt TEST slot is a distinct acceptance restriction, not a human reply quota.

SessionStorage drafts are scoped by session/user/org/conversation/mode. They survive refresh and restore only after authorization; logout, 401, organization switch and scope loss purge them. They retain text/idempotency metadata, never tokens or pricing authorization. Ambiguous submit recovers the existing client key and cannot silently create another send.

Preflight binds exact actor/session/organization/phone/WABA/recipient/content/assignment and immutable policy/window evidence. Submit atomically consumes an opaque single-use authorization. The worker rechecks current session/membership/permission, sender/recipient anchors, handoff, assignment, window, policy/review/horizon, payload scope and state. App/WABA/phone must match protected configuration. Fresh challenge and signed eligible inbound must follow the configured acceptance boundary.

The text adapter makes one bounded POST and disallows redirects. A usable returned message ID establishes ACCEPTED only. Timeout, missing ID, malformed response, unclassified error or lost DISPATCHING completion becomes UNCERTAIN. Recovery never blindly resends. Sanitized immutable status facts support orphan, duplicate and out-of-order correlation without inventing money or delivery.

Registry imports the reviewed Gate C bundle: six source identities/hashes, 20 numeric/policy rows and 36 tier rows. It executes DRAFT → VALIDATED → DIFFED → IN_REVIEW → APPROVED → PUBLISHED, with changes-required/revision and new-import correction/supersession. Actual diffs and review fingerprints cover source, artifact, reports, base, revision, rationale and deadline. Importer cannot self-review; publishing rechecks reviewer authority, base CAS and account currency. No company numeric rate head is selected while currency is UNKNOWN.

A standalone owner-reviewed Service-zero policy is recorded separately. Its review deadline is September 21 00:00 UTC; conservative cutoff is October 1 00:00 UTC, earlier than the observed WABA transition. Missing/expired evidence is BLOCKED, not zero. Budget foundations use exact positive decimal reservations, locked period limits and append-only ledger evidence; uncertain exposure remains reserved. Zero-cost paths insert no fake 0 USD/IDR entries. Paid quote/confirmation and live paid settlement remain disabled.

SSE invalidation contains scoped IDs/types only, reauthorizes every four seconds, reconnects within 24 seconds and requires REST refresh. Presence expires in PostgreSQL after 20 seconds and never locks composing. [ADR 012](../ADR/012-sprint-3-inbox-runtime.md) records the explicit minimum-scope refinement of the earlier cursor/Valkey target.

## API and frontend

[OpenAPI](../contracts/openapi.yaml) describes all 21 Inbox method/path pairs: Inbox setup/events/settings; conversation list/thread/state/assignment/read/notes/presence; note edit/redaction; pricing preflight; intent submit/recovery/status; Registry import/lifecycle/read; budget read/create. No generic external mutation proxy exists.

UI routes /inbox and /pricing are implemented. Inbox is the root operational destination. Resource command permissions come from the server, with catalog-driven organization/self/team navigation. Notes and replies are distinct composers. Guard decisions, pending/accepted/delivery/uncertain state and blocked explanations are visible. Pricing administration exposes provenance/reports, reviewer/publication steps and explicit UNKNOWN currency/closed paid authority.

## Tests actually executed

| Check | Result |
| --- | --- |
| python scripts/dev.py up | PASS after restarting Docker Desktop; initial resumed attempt failed because the daemon was stopped |
| python scripts/dev.py check | PASS, exit 0; Go vet/unit/build, real PostgreSQL integration, Python checks, frontend lint/typecheck/tests/build, OpenAPI and migration checks |
| Inbox unit/integration tests | PASS: 22 top-level tests including 17 PostgreSQL scenarios; subcases cover guard boundaries and contention |
| Populated-table RLS | PASS for all 21 tables under waba_runtime and waba_identity; no-context/cross-org SELECT and cross-org INSERT rejected |
| Concurrency | PASS: competing authorization, duplicate key, independent agents/epochs, committed assignment/policy/recipient/expiry denies, final-budget-unit competition |
| Auth and audit | PASS: scope/search/SSE, CSRF/session/header rejection, recent MFA, distinct reviewer, note redaction without body in audit |
| Signed full pipeline | PASS: forged signature denied, valid signature accepted/deduplicated, Sprint 2 processor then Inbox materializer |
| Frontend unit | PASS: eight tests including draft scope, expiry, safe fields and purge |
| Inbox browser E2E | PASS on isolated API/PostgreSQL with fake sender, including lost submit response plus recovery outage: composer remains locked, same intent recovered, one provider attempt |
| Combined identity/Meta/Inbox E2E | PASS: python scripts/dev.py e2e, all three suites, exit 0 |
| Dependency checks | PASS: go mod verify; pinned govulncheck v1.8.0 reports zero reachable vulnerabilities; npm audit reports zero vulnerabilities. One unimported-module advisory is recorded below |
| Sprint 3 hosted CI | PASS: implementation commit a49847151bb6ea3592856d9989201375102eca40, [run 34970609872](https://github.com/mbingsdk/whats/actions/runs/34970609872); all repository, PostgreSQL, browser and dependency steps executed |

The first full check found the foundation test still expected three migrations. That assertion was updated to four; repeated migration remains tested as zero changes. Earlier browser failures exposed missing Inbox permission discovery, inaccessible select labels and stale revision handling after delivery; fixes were retested. An additional direct npm invocation used the terminal's Node 22.13.1 and failed to load TypeScript tests; the supported runner and repeat use pinned Node 24.21.0. No failing repository test was bypassed.

Dependency review: govulncheck reports [GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932) for the unmaintained golang.org/x/crypto/openpgp package in the required x/crypto v0.57.0 module, with no fixed version listed. No imported package or reachable application symbol is affected according to the scanner; the application uses argon2, not openpgp. This is a module-level finding, not a claim that every dependency has no known advisory.

Windows local runs did not use Go race instrumentation because a C compiler was unavailable. Hosted run 34970609872 executed the unchanged WABA_TEST_RACE=1 workflow successfully on Linux. No production performance claim is based on synthetic test timing.

## Published implementation verification

[Commit a498471](https://github.com/mbingsdk/whats/commit/a49847151bb6ea3592856d9989201375102eca40) contains Sprint 3 implementation and initial evidence. Its own [hosted run 34970609872](https://github.com/mbingsdk/whats/actions/runs/34970609872) passed every required step. Baseline runs are not reused as implementation verification. This evidence-only reconciliation is published separately; it does not alter executable code or immutable migration 00004.

## Live acceptance

The September 16 bounded-cost decision above supersedes the following historical zero-cost checkpoint. Current results are recorded in the final section.

Controlled TEST recipient: NOT DESIGNATED for Sprint 3 at this checkpoint. Prior Sprint 2 inbound evidence is not designation. Callback: last verified restored to empty after Sprint 2 acceptance; not rechecked or changed in this pass. No permanent ingress is inferred.

Historical September 15 result, superseded by the focused review below: real Service delivery/pricing horizon UNVERIFIED. [M73](21-research-register.md#m73-service-delivery-horizon-review-2026-09-15) records first-party Direct Send TTL scope and source hash. That surface does not establish a bound for ordinary Service text. Because the reviewed source announces an October pricing change, a September dispatch with unbounded possible delivery cannot be assumed free. LoadLive provides no delivery-horizon override; preflight returns DELIVERY_PRICING_HORIZON_UNVERIFIED. One-minute bounds exist only in clearly synthetic tests.

Before any live acceptance:
1. Resolve the selected Service subtype's delivery/pricing evidence and review any necessary policy update. Do not infer company currency or convert the send into Direct Send/template.
2. Have the operator explicitly designate exactly one controlled TEST recipient in the protected file. Keep campaign exclusion.
3. Prepare the current callback state, proposed reachable HTTPS listener, scope/effect and rollback; obtain target-specific operator approval before mutation. A temporary tunnel remains acceptance infrastructure only.
4. Set the explicit approved acceptance start, obtain a fresh GET challenge and authentic inbound from that TEST recipient, and verify ACTIVE window plus current proof.
5. Only the authenticated authorized operator presses Send for one simple text reply. No agent/tool/automation initiates it.
6. Verify exactly one intent/attempt, actual Meta ID or UNCERTAIN, real status correlation, window non-reset, audit and Event Center evidence, then execute/verify the approved rollback.

No paid/template live test is authorized or required for this free-reply acceptance scope. Billing currency, paid spending ceiling, rate selection and paid consent remain separate future prerequisites. Gate D stays CLOSED.

## Review record

Affected architecture, API, database, UI, security, deployment, observability, testing, sprint/readiness and research documents are reconciled. Existing dated Gate B/C restrictions are historical where superseded by September 15 owner approval. Implementation closure is COMPLETE based on published a498471 and its own successful hosted run 34970609872. All three browser suites, migration repeat/checksums, OpenAPI, lint/typecheck/build, real PostgreSQL and Go race checks passed. Live acceptance stays OPEN until real evidence above exists. Paid authority and Gate D remain CLOSED.

## Historical M73 focused acceptance review, 2026-09-16

Implementation remains COMPLETE. Acceptance remains OPEN. [M73 follow-up](21-research-register.md#m73-focused-follow-up-2026-09-16) records current first-party HTML URLs, UTC retrieval times, original hashes, quality, scope and unresolved checks.

General Cloud API Service TTL is now documented as 30 days with drop-on-expiry behavior. Retention is independently documented and is not the basis for that finding. Current delivered-message pricing plus October's WABA-timezone transition does not establish send/acceptance-time grandfathering. A September 16 acceptance plus even an optimistic 30-day interval reaches October 16. It cannot fit the existing zero policy ending October 1, and no shorter ordinary Service contract was verified.

No executable code, published migration, policy or runtime configuration changes. DELIVERY_PRICING_HORIZON_UNVERIFIED remains active because the complete safe acceptance/pricing proof is not established. The hypothetical tests also show PRICING_DELIVERY_COVERAGE_MISSING if a 30-day latest-delivery calculation is supplied. M73 is partially resolved as contract research, not passed as authority for this live test.

Controlled TEST recipient remains NOT DESIGNATED. Current callback was not probed or changed; last verified state is the empty Sprint 2 rollback. No tunnel/listener, authentic new inbound, live window, live preflight ALLOW, provider attempt, outbound ID/status or rollback result is claimed for this pass. No live dispatch configuration, paid/template request, media send or Sprint 4 work occurred. No operator approval was requested for dependent live operations because the pricing prerequisite does not pass.

Safe alternatives are recorded in M73: precise first-party clock/transition clarification, full-horizon zero-policy evidence, or separately authorized billing/exposure review. No message or support request was sent to Meta or anyone else.

Verification for this evidence commit: two added M73 unit tests (six transition subcases plus runtime fail-closed assertion) PASS. python scripts/dev.py up and the repeated full python scripts/dev.py check PASS (exit 0), including 24 Inbox unit/integration tests, 17 real PostgreSQL scenarios, frontend lint/typecheck/eight tests/build, OpenAPI, migration checksums and 183 local links. The first full attempt passed Go/PostgreSQL but npm ci failed with Windows EPERM because the running local frontend held the SWC binary. Only that frontend was stopped temporarily; the unchanged full check was rerun successfully and the frontend restarted. Hosted CI PASS for [M73 commit 5b3e536](https://github.com/mbingsdk/whats/commit/5b3e5369f3248d8d4bb1b86e26cc59dd4eb4417c), [run 35003395625](https://github.com/mbingsdk/whats/actions/runs/35003395625). All required repository, real PostgreSQL, Linux race, identity/Meta/Inbox browser and dependency checks executed successfully; no live Meta secrets were used. The eight M73 source URLs, UTC retrieval times and SHA-256 entries were checked against retained originals and match. This follow-up only records the result; it does not close live acceptance. Earlier implementation and evidence commits remain tied to passing runs 34970609872 and 34971097556.

Review record: the six requested status/evidence documents and the focused regression tests are the only changes. Company billing currency UNKNOWN; paid/template authority CLOSED; Gate C APPROVED; Gate D CLOSED.


## Current bounded-cost acceptance review, 2026-09-16

This corrects acceptance evidence and the shared pricing predicate; it is not Sprint 4. Baseline implementation remains COMPLETE; acceptance OPEN. No migration or published pricing row changed. No financial authority, rate publication, recipient database fixture or runtime currency write was created.

| Item | Actual result |
| --- | --- |
| M73 / domain denial | Ordinary Service TTL: 30 days. PRICING_HORIZON_NOT_FULLY_COVERED replaces both older horizon/coverage codes; preflight, submit and worker share the gate. |
| Currency | IDR from operator-reviewed Meta Billing answer; not independently inspected; runtime state not modified. |
| TEST recipient / market | Operator-designated protected file, valid format, market ID confirmed; number excluded from source. TEST_ONLY/campaign exclusion awaits acceptance fixture. |
| 30-day coverage | INCOMPLETE. Zero policy ends before the projected horizon; canonical evidence lacks a reviewed complete executable interval mapping. |
| Publications | No complete executable publication chain selected or created. Historical published rows immutable. |
| Maximum / approval | NOT CALCULABLE from available executable evidence; no ceiling, TEST budget or owner financial confirmation exists. |
| Callback / fresh inbound | Current callback not re-read or mutated. Historical empty Sprint 2 rollback is not current proof. No fresh challenge/inbound claimed. |
| Window / preflight | No live ACTIVE window or ALLOW claimed. Synthetic PostgreSQL preflight and queued worker reject incomplete coverage even with verified IDR. |
| Provider / ID / status | Zero live attempts; no new Meta ID, UNCERTAIN outcome or authentic outbound status. |
| Metadata / settlement | No live metadata, financial reservation or invoice settlement. Calculator output is INTERNAL_ESTIMATE only. |
| Idempotency / attempts | Synthetic PostgreSQL recovery retains one reservation/ledger entry including UNCERTAIN; existing send tests prove one intent/provider attempt and no retry. |
| Window non-reset / UI | Existing synthetic integration/browser coverage; new authentic Inbox/Event Center/window evidence pending. |
| Cleanup | No callback/tunnel created in this review; no rollback performed or falsely claimed. |
| Internal work before live acceptance | Reviewed executable Registry mapping plus bounded financial quote/confirmation/dispatch integration and final UI financial confirmation are not enabled. Separate from operator approvals. |
| Gate / authority | Gate D CLOSED; generic paid/template/media/campaign/automation authority CLOSED. One bounded TEST Service TEXT may later be approved for an explicit maximum. |

Focused local verification: 28 Inbox top-level tests (10 unit, 18 real PostgreSQL) PASS, including full-period pricing cases, changed currency evidence, maximum increase, idempotent positive reservations, uncertain retention, tenant isolation, authorization and final-worker blocking. All rate/approval fixtures are synthetic; no test reaches Meta.

Verification of this correction: python scripts/dev.py up, python scripts/dev.py check and python scripts/dev.py e2e all PASS (exit 0). This includes Go vet/unit/build, all actual PostgreSQL suites, 28 Inbox top-level tests, 21-table populated RLS under both runtime roles, concurrency/security cases, frontend lint/typecheck/eight tests/build, OpenAPI, immutable migration checksums and 183 local documentation links. Identity, Meta and Inbox browser suites all passed against isolated databases and fake/local services. No live Meta secrets or provider requests were used.

Dependency review: go mod verify PASS; pinned govulncheck v1.8.0 reports zero reachable vulnerabilities and zero vulnerabilities in imported packages, with one advisory in a required module whose vulnerable package is not used; npm audit reports zero vulnerabilities. Windows local Go tests were not race-instrumented; unchanged hosted CI must run Linux race checks. All eight retained M73 source hashes/timestamps/URLs match. No CI, dependency lock or published SQL/checksum changes.

Hosted CI for this correction: PENDING publication/run. Prior HEAD 7f2d47b passed run 35003849607; it does not verify these new changes. Automatic review rejected a proposed AGENTS.md edit, so that instruction file was left unchanged; current owner decisions are recorded in the authorized project documents instead.
