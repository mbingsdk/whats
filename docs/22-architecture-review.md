# Second-pass architecture review

Reviewed **2026-09-09** as a skeptical design review of the written specification. This was a self-review, not an independent security audit, implementation test or live Meta account verification. Findings below were corrected in the referenced documents; remaining evidence gates are explicit.

## Findings corrected in the specification

| Rejection concern | Correction and location | Runtime evidence still required |
| --- | --- | --- |
| Accidental single-user/team/WABA | Global users with org memberships, many team memberships, multiple apps/WABAs/phones; exclusive internal ownership of each external asset, docs 03/05/14 | Multi-org/team negative tests |
| Webhook route trusts payload tenant | App-bound opaque route and known WABA/phone binding; cross-tenant shared external app deferred, docs 05/08 and ADR 005 | Forged/misrouted/multi-entry fixtures |
| Body-hash-only dedupe loses/repeats effects | Separate transport digest and child semantic/effect keys, docs 05/08 | Rebatched and cross-app duplicate tests |
| Status arrives before response or out of order | Orphan status facts/stubs, independent observed timestamps, no regression, docs 04/05/08 | Read-before-sent and timeout reconciliation |
| Worker lease mistaken for exactly-once send | DISPATCHING crash -> UNCERTAIN, no automatic resend, docs 08/10/12 and ADR 007 | Fault injection at every external-call boundary |
| Consent/policy changes between reservation and send | Final DISPATCHING transaction reruns scoped guards; shared organization policy barrier and checked resource revision vector order relevant changes, docs 05/10/12 | Race tests for opt-out, revocation, price/window expiry |
| Human replies incorrectly treated as an epoch winner | Removed one-human-reply quota; handoff_epoch invalidates stale bots, assignment_revision protects routing changes, advisory human collision warnings, docs 04/05/06/09/12/13/17 | C01-C05 sequential/text-media/two-agent/takeover tests |
| Snapshot references require mutation of immutable revision | Snapshot reverse relationship and joint approval digest; removed mutable snapshot pointer, doc 05 | Snapshot sealing/revision invalidation tests |
| Immutable records had mutable publication/attempt fields | Publication state separated; Flow content immutable with mutable lifecycle; attempts mutable until terminal, doc 05 | Schema/transaction tests |
| Budget check-then-send race | All matching periods and authority caps reserved atomically under canonical locks; unknown exposure retained, docs 05/12 | Concurrent last-unit and duplicate settlement tests |
| Pricing based on stale free-service assumptions | All content types use versioned policy, no seeded rate values, future coverage gaps block, docs 02/12/21 | Official current/announced policy verification |
| Webhook pricing flag presented as invoice amount | Source-labeled metadata/estimate/reconciliation with nullable amounts, docs 05/12/15 | Official billing evidence and reconciliation fixtures |
| Approval or API-key bypass | Immutable scope, current delegation ceiling and sponsor policy, separate cost confirmation, docs 06/10/14 | Escalation/self-approval/sponsor-revocation tests |
| CSV import or contact merge clears opt-out | Append-only consent evidence, independent identity suppression and conservative merge, docs 05/11 | Import/reimport/merge/late-event cases |
| Replay sends old automations or resurrects erased data | PROJECT_ONLY default, stable effect keys, tombstones and restored erasure ledger, docs 08/11/16 | Parser replay and restore-purge drills |
| Deleted employee destroys audit or keeps access | Durable member principal, revocable sessions/personal keys, access revisions and current stream checks, docs 05/14 | Offboarding E2E and audit inspection |
| Media source URL expires | Immediate fetch, refresh current URL, private storage, bounded unavailable state and signed access, doc 09 | Expiry/deletion/SSRF/scan fixtures |
| Frontend sees stale or unauthorized state | Server window/pricing authority, resource revisions, org-scoped cache and per-event access, docs 06/07/09 | Org switch/reconnect/access-change E2E |
| Realtime cursor skips late DB commits | Sequence assigned during serialized publication commit, not event UUID ordering, docs 05/07 | Concurrent commit/reconnect test |
| Optional Meta products presented as guaranteed | Capability matrix distinguishes evidence/account gates; advanced modules disabled, docs 02/19/21 | Feature-specific official/account evidence |
| Queue/backup design exceeds ordinary VPS operations | Postgres durable jobs, disposable Valkey, systemd processes, external objects, explicit single-host outage risk, docs 03/16 and ADRs | Load/restore/rollback measurements |
| Security/support policies lack schema records | Added auth challenges, policy/calendar versions, idempotency tombstones and authority reservations, doc 05 | Migrations and policy validation tests |

## Residual risks and unresolved evidence

Current Graph version, signature/payload constraints, template mutation limits, exact pricing/rate cards/announced changes and the company's effective asset permissions are NEEDS VERIFICATION. This is not resolved by the existence of an old official Postman example. Those adapters and paid sends remain gated. The company confirmed existing Cloud API use, so initial migration, coexistence and Embedded Signup are unnecessary.

We cannot promise zero external duplicates after an accepted request with lost response, exact account-wide spending when other senders exist, guaranteed read receipts, or recovery of unavailable Meta media. The design exposes uncertainty, retains budget exposure and requires reviewed recovery instead of hiding these limits.

Single-VPS availability, restore objectives, estimated capacity, UI usability/accessibility and query performance are unmeasured. Proposed limits/retention/approvers/currency/markets must be accepted or replaced by company owners. At the documentation-review stage, no application had been deployed and no dependencies had been installed.

## Document verification

The focused correction reran coverage, file/fragment link, UTF-8, fence and obsolete-semantics checks on 2026-09-09: 44 local Markdown links passed across 37 Markdown files (23 product/technical documents, ten ADRs, ADR index and three root documents), zero broken local links, zero unbalanced fenced blocks, no invalid UTF-8 replacement characters and zero implementation/dependency files. The package includes product/PRD, capability evidence, schema, workflows, state machines, API/events, jobs, cost/consent/security, operations, UX, test/release plans and ten decision records. External links are research citations, some deliberately marked BLOCKED; they are not all claimed reachable or account-tested.

Documentation-phase readiness judgement: **documentation baseline complete; implementation and production readiness remain gated by PLANS.md and doc 20**. Foundational implementation can begin after Gate A is accepted and subsequently authorized. Unverified Meta adapters require Gate B; paid sends require Gate C; production requires Gate D. This distinction avoids both pretending verification is complete and blocking internal foundation work on optional Calling/Coexistence.

## Focused correction pass, 2026-09-09

| Finding | Resolved decision / dependent contracts | Remaining validation |
| --- | --- | --- |
| Human second messages rejected | Distinct replies have distinct intents/keys, no reply quota; bot handoff and actual assignment checks remain; docs 04/05/06/09/12/13/17 | C01-C05, no tests executed |
| Organization row serializes all dispatch | Ordinary FOR SHARE barrier; narrower recipient/contact/conversation/run/budget/authority locks, phantom-policy protection, explicit hot-budget risk; docs 05/10/12 and ADR 007 | Mixed agents/ingress/campaign/opt-out/budget benchmark and documented redesign thresholds |
| Identity depends on future mail provider | Required provider-neutral SMTP boundary, durable one-use/expiry/retry/failure behavior; docs 03/04/05/06/08/14/15/16/19 and PLANS | Configure relay/sender Sprint 0/1; C08 |
| Historical API example looks production-verified | Contract evidence separated from applicability, account gates and deferred scope; no current-contract promotion; AGENTS and docs 02/21 | Pinned official contracts and target-account probes before adapter enablement |
| Rates lack operational review pipeline | Explicit import/validate/diff/review/publish with hashes, distinct reviewer, immutable corrections, scoped current head; docs 04/05/06/08/12/14 | C10 and real official rate evidence before paid sending |
| Refresh contradicts memory-only drafts | Per-tab sessionStorage keyed by login/user/org/thread/mode; refresh restore after access check, org-switch retention, logout purge, attachments revalidated; docs 03/07/09/18 | C09 including suspended tabs and storage failure |
| Isolation mistaken for SaaS product scope | Normal one PT/one org, conditional switcher, no public signup/subscriptions/marketplace/generic wizard; README and docs 00/01/18 | C11; existing multi-org isolation tests retained |
| Legacy policy wording determines billing | Pricing and messaging policy reread; discrepancy recorded with source dates and unresolved exact rates/account predicates; docs 02/12/21 | Gate C current effective-dated numerical evidence; no invented rates |

Only ADR 007 changes a recorded architecture decision (dispatch coordination); modular monolith, PostgreSQL jobs, Valkey, SSE/REST, private storage, tenant boundaries and VPS topology remain the baseline. That correction pass added specifications only: no code, dependencies, application scaffolding, live account mutation, mail or WhatsApp send.

Subsequent owner decision: Gate A **APPROVED** on 2026-09-09, with D2/D5 accepted provisionally and Sprint 0 ONLY authorized. This supersedes the correction-phase readiness recommendation; see PLANS and doc 23. Gate B official/account contracts, Gate C pricing and send evidence, and Gate D runtime/operational acceptance remain open; optional integrations do not block foundation approval.

## Sprint 0 implementation review

The earlier 37-file/no-code verification above describes the completed documentation phase, not current repository state. Gate A approval subsequently authorized the foundation recorded in [doc 23](23-sprint-0-evidence.md). Review focused on actual RLS roles and pool reuse, same-origin API access, safe dependency errors, no secret output, SMTP TLS/auth without email transmission, migration-role separation and real integration execution. No empty domain packages, generic repositories, future feature routes or Meta send behavior were added.

The final local check passed, including frontend compilation and real PostgreSQL integration tests; Linux backend race tests, the compiled API health/readiness smoke test and actionlint also passed. Dependency integrity and vulnerability checks passed. See doc 23 for the separate evidence entries.

Remaining acceptance is explicit: hosted CI is not yet observed; official Graph/version/webhook evidence remains blocked; SMTP mailbox/production recovery and product-level tests have not run. No other gate is opened by local foundation test results.
