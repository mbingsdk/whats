# Testing strategy

Owner clarification (2026-09-15): **Sprints 0/1/2 COMPLETE; Gates A/B/C APPROVED; Gate D CLOSED; Sprint 3 AUTHORIZED. COMPANY META BILLING CURRENCY UNKNOWN; PAID-SEND AUTHORITY CLOSED.** Currency must not be inferred from timezone, business/phone/recipient country, available rate cards or locale. Only a currently reviewed, provably zero-cost policy can permit the single operator-triggered controlled TEXT live reply after all recipient/callback/security prerequisites. Paid/template live sends, outbound media, campaigns and Sprint 4 remain unauthorized.

Earlier dated status/review entries below are historical and superseded by this owner clarification where they describe Gate C or Sprint 3 authorization.

Tests demonstrate specified behavior, especially isolation and external-effect boundaries. Sprint 0 now implements and executes foundation tests, recorded in doc 23. The domain/E2E scenarios below remain future acceptance specifications. Product performance targets are unmeasured until the planned load/restore runs.

## Test layers

| Layer | Meaningful coverage |
| --- | --- |
| Go unit/domain | Window boundaries/skew, policy evaluation, canonical hashes, money rounding, state transitions, permission scope/delegation, error classification, typed payload validation |
| PostgreSQL integration | Real migrations/constraints/RLS using runtime roles, cross-tenant FKs, pooled tenant context reset, concurrent reservations, claims/fences, snapshot consistency, dedupe and orphan status merge |
| Service/API | Cookie/API-key scope, CSRF, revision/idempotency protocol, response redaction, invitation/approval workflows, async status semantics |
| Webhook fixtures | Current official/account-derived sanitized fixtures with Graph/parser provenance; batch/multi-entry/unknown/invalid-signature cases; do not label synthetic fixtures as captured Meta data |
| Worker fault injection | Crash before/after DB commit, before/after HTTP write, after acceptance before result commit, lease expiry and stale worker result |
| Frontend component | Typed message/error/unknown/media states, keyboard/focus, countdown from server clock, accessible data tables, no content leakage in cached org switch |
| E2E | Go+Next+Postgres with controllable fake Meta server and reviewed test-account smoke checks; no real customer campaigns |
| Operations | Backup restore/erasure ledger, rollback across schema versions, secret rotation, dependency outages, disk/queue saturation |

Use fake clock and deterministic policy fixtures for boundaries; no sleeps to test time-dependent transitions. Use actual PostgreSQL locking, not mocked repositories, for race guarantees. Synthetic money scenarios are labeled and never shipped as real rate cards. Gate C also retains separately labeled real official numerical evidence; these draft evidence fixtures are not published runtime configuration. Outbound mock distinguishes pre-connect failure, definite rejection, accepted-then-timeout and reordered callbacks.

## Critical scenarios and expected evidence

| ID | Scenario | Assertions |
| --- | --- | --- |
| E2E-01 | Connect existing Meta account | No registration/migration; correct binding ownership/scope; secrets write-only; conflicting org rejected; stale sync visible |
| E2E-02 | Incoming message | Signature verified, ACK after durable commit, correct scoped thread created once, server window calculated |
| E2E-03 | Agent reply | Authorized sender/handoff, preflight, pending->accepted->observed status; no frontend-only authorization |
| E2E-04 | Expired service window | Free-form denied both API and queued dispatch; approved template suggested without automatic substitution |
| E2E-05 | Paid template confirmation | No dispatch before bound confirmation; altered recipient/content/amount/currency or expired quote rejected; same client key returns same intent |
| E2E-06 | Campaign approval | Creator lacks self-approval; threshold/quorum enforced; modified snapshot invalidates; role change prevents later execution |
| E2E-07 | Campaign send | Snapshot immutable, one intent per recipient; eligible current state checked; all budgets/frequency respected |
| E2E-08 | Campaign pause/resume/cancel | No permits after pause commit; in-flight count honest; resume rechecks rates/consent; cancel never recalls delivered message |
| E2E-09 | Opt-out | Suppressed before final authorization excludes; subsequent CSV/merge/replay cannot restore marketing; in-flight boundary logged |
| E2E-10 | Webhook retry | DB failure returns retryable response; later delivery succeeds once; queue outage cannot lose committed job |
| E2E-11 | Duplicate/reordered webhook | Same message across envelopes/apps yields one row/effect; read-before-sent no regression; status-before-response reconciles |
| E2E-12 | Employee denied/offboarded | Old session, SSE stream, scoped API key, pending export and personal send cannot continue; audit history remains |
| E2E-13 | Human reply and bot takeover concurrency | Multiple authorized human replies remain valid; advisory collision warnings; takeover invalidates queued bot epochs; already-DISPATCHING boundary explicit. Cases C01-C05 below |
| E2E-14 | Unknown payload/media expiry | Unknown message stored/rendered safely; no automatic effects; stale URL refreshed, permanently missing asset visible |
| E2E-15 | Replay and privacy purge | Project-only replay creates no sends/HTTP/notifications; tombstone prevents content resurrection |
| E2E-16 | Outbound delivery replay | Valid signature with fresh timestamp, stable event ID, scope revoked prevents send, downstream dedupe prevents repeated effect |

## Race and failure tests that block release

Run two real concurrent DB connections through final budget amount, frequency ceiling, confirmation consume and last campaign slot. Exactly one authorized winner where capacity is one; loser has no attempt/ledger leak. Lock ordering must not deadlock under simultaneous opt-out, policy change and dispatch; bounded retry may resolve serialization conflicts without duplicate effects.

Crash dispatcher after DISPATCHING commit before network, and separately after network acceptance before DB result. Both are conservatively UNCERTAIN without automatic retry; manual resolution must not create acceptance without evidence. Stale worker loses fence but later positive response can reconcile the existing attempt. Budget/frequency exposure persists.

Exercise quote expiry, service-window expiry, policy effective-date gaps, category change, currency mismatch, midnight/month boundary, unknown volume tier/free-entry evidence, external sender spend and duplicate invoice correction. Historical estimates stay unchanged; current guard blocks uncovered pricing.

Isolation tests use at least two organizations with multiple WABAs, overlapping external contact numbers, users in multiple teams and restricted viewers. Test list counts, search snippets, media URLs, contact timeline, notifications, SSE replay, exports, API keys, job payload tampering and raw webhook routing. Execute direct SQL as runtime role to validate RLS and composite FKs, not merely HTTP guards.

## Performance and evidence

Seed representative histories and skewed workloads; evaluate EXPLAIN plans for inbox list, unread, conversation messages, audience rules, due jobs, budget lock contention, status dedupe and retention. Run doc 00 load assumptions with campaigns alongside agents/ingress. Record host specs, versions, dataset, p50/p95/p99, error rate, memory and backlog recovery. Targets in PRD are pass criteria to negotiate if unrealistic, never fabricated results.

Go race detector covers concurrent worker/controller code. Frontend automated accessibility checks supplement keyboard/screen-reader review at desktop/tablet/mobile widths and reduced motion. E2E critical paths run on each release; exact Meta adapter contract fixtures run on Graph upgrades and current test-account verification before rollout. Live sends use explicit allowlisted test recipients, consent and budget.

Each release links requirement IDs, test results, sanitized fixture provenance, restore timings and unresolved defects to the production checklist. No P0 correctness/security defect or unreviewed high-risk exception may be waived by a demo.

## Focused correction acceptance specifications

These are tests to implement, not executed results. All send cases use independently sufficient consent/window/budget/authority fixtures so a separate guard does not disguise collision behavior.

| ID | Scenario | Required assertions |
| --- | --- | --- |
| C01 | Same agent sends two sequential messages | Distinct idempotency keys yield two intents/dispatches without changing assignment_revision or requiring a new handoff epoch; retry of either key produces no additional effect. |
| C02 | Same agent quickly sends text then media | Both are admitted, attachment access/scan/sendability checked separately; returning the text result cannot erase the newer draft; expose local intent order without promising network delivery order. |
| C03 | Two authorized agents send concurrently | Both payloads preserved and independently gated; presence/typing/recent-send warning is visible but no compose lock or same-epoch rejection. A real concurrent assignment change returns ASSIGNMENT_CHANGED/BLOCKED for the stale action; never silently drops it. |
| C04 | Bot queued before takeover | Human takeover commits new handoff_epoch before final authorization: old bot intent blocks, no external request, safe undispatched reservations released; release to BOT does not revive the old epoch. |
| C05 | Bot already DISPATCHING at takeover | Takeover does not recall or resend the request; outcome can be accepted or uncertain, visible in UI/audit. New bot intents cannot authorize under the prior epoch. |
| C06 | Shared organization barrier and narrower deny writers | Different contacts/uncapped fixtures authorize concurrently under SHARE. Committed disable/policy change precedes any later authorization using new policy; unknown-contact opt-out creation and concurrent alias/merge cannot slip through an absent suppression check. |
| C07 | Budget topology, price head, pause, frequency and authority races | New/re-scoped budget cannot escape matching set; numeric limit decrease conflicts with reservations; pause conflicts with final run read; contact frequency and all financial caps conserve exposure; stale publication never passes using an old permit. |
| C08 | Required identity mail | Invitation/verification/reset each expire and consume once, including simultaneous completion; resend invalidates old generation even if old mail arrives later. New-user invite setup works without public signup and cannot replace an existing user's credentials; verification binds the intended address. SMTP 4xx/5xx/TLS failure, lost response and lease crash obey bounded retry/deadline; notices contain no reusable credentials; provider failure does not activate an account or reveal existence. Assert secret-free logs and scoped invitation status. |
| C09 | Draft lifecycle | Refresh restores only same user/org/thread/mode after access check; switch preserves the old namespace but clears rendering; logout/account switch/expiry purge across active and suspended tabs before restore. Quota/storage failure warns; stale attachment requires reattachment; ambiguous submit recovers existing intent/key, never resends automatically. |
| C10 | Rate Card Registry | Import/manual-entry validation rejects missing evidence/dimensions, overlaps and currency errors. Changes invalidate review; importer cannot self-review; stale worker/base cannot publish; immutable publication/correction retains original estimate and invoice evidence. Coverage gaps, overdue review and effective-date boundaries block affected dispatch. |
| C11 | Capability and initial product scope | Historical/index-only evidence never enables an adapter; exact current contract plus target-account probe required. Single PT has no org switch/signup/subscriptions/marketplace/onboarding; a genuine second membership exposes switching with all existing two-org isolation tests retained. |

## Mixed dispatch contention benchmark

Use real PostgreSQL, a controllable fake Meta endpoint and synthetic reviewed price fixtures; do not generate real customer sends. On the proposed doc 16 host, record hardware/storage, DB settings, dataset size/skew and transaction implementation. After warmup, run each offered campaign stage at 10, 25, 50 and 100 authorizations/second for 30 minutes, with 25 active agents producing an aggregate five reply intents/second, 50 webhook requests/second with short 100/second bursts, one opt-out/second and a numeric budget change every ten seconds. These are proposed stress inputs, not provider throughput promises; owner-approved actual peaks replace them before acceptance.

Run both disjoint-contact traffic and a skewed hot-contact workload; include a common funded org budget and campaign authority to expose their serialization. Ensure fixture capacities do not reject most sends before locks are exercised. Add budget creation/re-scoping, periodic campaign pause/resume, publication and emergency disable/re-enable phases. Keep ingress ACK independent of final dispatch; project inbound windows under normal contact/conversation locks.

Capture offered/admitted/completed throughput, rejected reasons, final-authorization and ingress ACK p50/p95/p99, each lock tier's wait/hold, deny command arrival-to-commit latency, deadlocks/timeouts, pool use, queue oldest age and recovery. Verify commit-order invariants in the trace; separate true policy-write intervals from steady state, including writer starvation under continuous readers. Apply doc 05's proposed latency and sustained-five-minute redesign thresholds and the existing PRD ingress target. After removing load, backlog must drain within an owner-approved recovery bound recorded before the run. No silent threshold adjustment after failure. If the common budget/authority row fails the accepted workload, redesign allocation or capacity with explicit conservation proofs before release; do not remove the hard cap or hide failures behind aggregate throughput.

Foundation tests are in backend/internal/config, httpapi, smtpprobe and the integration-tagged database package. python scripts/dev.py test runs the real database suite and fails if its required isolated PostgreSQL configuration is absent. CI sets WABA_TEST_RACE=1 on Linux. The complete local sequence is python scripts/dev.py check; see [executed evidence](23-sprint-0-evidence.md) and [future lock measurement procedure](../database/bench/README.md).

## Sprint 1 execution

python scripts/dev.py check runs source links/checksums/route coverage, formatting/vet, all Go unit and explicit real-PostgreSQL integration packages, build, npm ci, lint/typecheck/frontend tests, OpenAPI and production Next build. python scripts/dev.py e2e runs actual Chromium/Next/Go/PostgreSQL with a local authenticated TLS SMTP mailbox. Install the pinned browser with npx playwright install chromium (CI also installs Linux system dependencies). WABA_TEST_RACE=1 enables Go race detection in test/check/e2e on Linux. No explicit integration suite silently skips.

[Current Sprint 1 evidence](24-sprint-1-evidence.md) distinguishes actual test results from controlled mailbox and hosted-account evidence. C08 is locally exercised; all four controlled SMTP receipts are confirmed in doc 24. C01-C07/C09-C11 remain relevant later acceptance tests where they require unimplemented messaging/product modules. The browser harness exposes test-only mail/TOTP helpers exclusively in the integration,e2e test binary.

## Gate B and conditional Sprint 2 boundary

[Gate B](25-gate-b-evidence.md) remains CLOSED, so no Sprint 2 integration test has executed. Gate B GETs now pass separately; four CAPTURED_SANITIZED fixtures accompany the unchanged SYNTHETIC fixture. No webhook signature or live notification test has run. After approval, required tests include Graph errors/pagination/auth/permissions, sync idempotency/disappearance/concurrency, challenge/signature rejection, duplicate/unknown events, retries/DLQ/replay, credential/payload redaction, unauthorized subscription and runtime RLS for every new tenant table. Ordinary CI must use sanitized fixtures and real PostgreSQL, never production Meta credentials. Controlled read-only/account-webhook tests stay explicit; all WhatsApp sends remain prohibited.

## Sprint 2 executed test boundary

The Meta package uses httptest Graph/signature fixtures and a disposable real PostgreSQL database. CI requires no live Meta credentials. Cases include auth/permission/rate/error redaction, pagination loop/foreign-host/size/version failures; coalesced concurrent snapshot persistence and non-destructive absence; exact raw signature/duplicate GET rejection; durable duplicate ACK, semantic rebatching, encrypted evidence binding, unknown/quarantine/permanent invalid states, injected SQL failure through eight retries/DLQ, replay generation/idempotency and populated-table isolation/privilege denial.

Live account synchronization, real public challenge and authentic Meta POST are separate operator-controlled evidence. Locally generated HMAC requests demonstrate implementation correctness only; they cannot be relabeled as real Meta delivery. Hosted success must refer to the actual Sprint 2 commit. See [Sprint 2 evidence](26-sprint-2-evidence.md).

## Gate C executable specification, 2026-09-14

**Gate C CLOSED; Sprint 3 NOT STARTED.** Run python docs/fixtures/gate-c/check_spec.py. The [39-case fixture matrix](fixtures/gate-c/pricing-cases.json) includes A–L from the owner's request plus October allowance/charging, WABA timezone boundaries, delivery crossing policy dates, replay/future timestamps, stale/paused templates, permissions/consent, currency/coverage, budget, organization-scoped idempotency and uncertain outcomes. [Detailed evidence and expected decisions](27-gate-c-evidence.md#executable-specification-and-future-tests).

Executed: 39 offline decisions PASS, 20 decimal rate rows PASS and 36 contiguous tier rows PASS. Separate original-artifact comparison confirms all 36 October PDF tier intervals/amounts match July CSV. The model makes zero network calls and imports no product code. Synthetic assumptions do not create real recipient consent, rate publication, budget or owner approval. In-memory duplicate handling is not a PostgreSQL concurrency test.

Required later Sprint 3 acceptance: two-tenant runtime RLS/FK/pool tests; concurrent last-unit budget reservation; single-use confirmation/idempotency conflict; denial committed before permit authorization; template/rate publication versus queued dispatch; expiry at dispatch; post-write timeout/crash and lost provider ID; duplicate/reordered/orphan statuses and unknown pricing metadata. Keep these as unexecuted product requirements until code exists. Live tests follow the controlled recipient/budget/callback and immediate paid-test approval policy in doc 27.
