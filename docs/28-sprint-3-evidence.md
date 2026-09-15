# Sprint 3 evidence: guarded Inbox, message domain and outbound safety

Owner clarification (2026-09-15): Sprints 0/1/2 COMPLETE; Gates A/B/C APPROVED; Sprint 3 AUTHORIZED; Gate D CLOSED. Company Meta billing currency UNKNOWN. Paid/template authority CLOSED. No currency inference from country, timezone, rate artifacts or locale.

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

Controlled TEST recipient: NOT DESIGNATED for Sprint 3 at this checkpoint. Prior Sprint 2 inbound evidence is not designation. Callback: last verified restored to empty after Sprint 2 acceptance; not rechecked or changed in this pass. No permanent ingress is inferred.

Real Service delivery/pricing horizon: UNVERIFIED. [M73](21-research-register.md#m73-service-delivery-horizon-review-2026-09-15) records first-party Direct Send TTL scope and source hash. That surface does not establish a bound for ordinary Service text. Because the reviewed source announces an October pricing change, a September dispatch with unbounded possible delivery cannot be assumed free. LoadLive provides no delivery-horizon override; preflight returns DELIVERY_PRICING_HORIZON_UNVERIFIED. One-minute bounds exist only in clearly synthetic tests.

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
