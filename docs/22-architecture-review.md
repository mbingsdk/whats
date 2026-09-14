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

## Historical Sprint 0 implementation review

The earlier 37-file/no-code verification above describes the completed documentation phase, not current repository state. Gate A approval subsequently authorized the foundation recorded in [doc 23](23-sprint-0-evidence.md). Review focused on actual RLS roles and pool reuse, same-origin API access, safe dependency errors, no secret output, SMTP TLS/auth without email transmission, migration-role separation and real integration execution. No empty domain packages, generic repositories, future feature routes or Meta send behavior were added.

The final local check passed, including frontend compilation and real PostgreSQL integration tests; Linux backend race tests, the compiled API health/readiness smoke test and actionlint also passed. Dependency integrity and vulnerability checks passed. See doc 23 for the separate evidence entries.

## Owner acceptance and hosted CI reconciliation, 2026-09-09

The product owner accepts the Sprint 0 architecture/implementation baseline subject to successful hosted CI. Review of the committed workflow at 993c67e93fe5945ec820efe67d9d3abf3df8b1b3 and GitHub run 34356265759 found a completed failure, with the foundation job blocked before any steps by an account billing lock. GitHub's exact annotation and timestamps are preserved in doc 23. This run provides no hosted test result; no repository-caused test failure was reported. The workflow's checks and migration SQL were not weakened or changed.

Historical interpretation, superseded by the owner override below: Sprint 0 was considered OPEN pending a successful required hosted workflow. Unresolved current Meta contracts remain Gate B blockers only; the provider-neutral SMTP boundary satisfies Sprint 0, while controlled mailbox delivery is required for Sprint 1 identity-email completion. That review treated Sprint 1 authorization as conditional; the owner subsequently explicitly authorized it. Gates B/C/D remain CLOSED; no Sprint 1 or Meta functionality was implemented in this reconciliation.

## Subsequent owner override, 2026-09-09

Owner override (2026-09-09): **Sprint 0 IMPLEMENTATION COMPLETE; Gate A APPROVED; Sprint 1 AUTHORIZED. Hosted CI verification PENDING - EXTERNAL BLOCKER.** Run 34356265759 was attempted and failed before repository steps because GitHub reported an account billing lock. Accepted local evidence closes implementation, not hosted verification. CI requirements remain intact; rerun hosted CI when available, record the real result and fix any repository failures. Gates B/C/D remain CLOSED.

This supersedes the prior closure interpretation above. Sprint 1 implementation proceeds without waiting for external CI or SMTP evidence. Controlled mailbox evidence remains required before Sprint 1 email acceptance; automated tests use local SMTP only.

## Sprint 1 implementation review, 2026-09-10

Implemented the approved identity/access/security scope with dedicated limited identity credentials, nine FORCE-RLS tenant tables, explicit scoped permissions, revision/Owner invariants, TOTP/recovery, session invalidation and durable authenticated SMTP. Review findings corrected: UUID JSON encoding, sensitive-window renewal on organization switching, pending login-MFA proofs surviving password/security changes, account-rate-limit evasion via unrelated fields, stale organization sessions after role/offboarding changes, archived-team direct grants and incomplete list pagination. Matching regression tests exercise these behaviors.

Frontend routes and OpenAPI now match implementation; source validation checks route coverage. CI retains its existing checks and adds pinned Chromium/local SMTP E2E. [Sprint 1 evidence](24-sprint-1-evidence.md) records executed checks and external blockers separately. At that review, controlled mailbox receipt remained pending and hosted run 34356265759 attempt 1 was an external billing/account failure; attempt 3 later passed the Sprint 0 baseline. Gates B/C/D remain CLOSED.

Historical pre-publication inspection distinguishes historical billing failure (run 34356265759 attempt 1) from actual Sprint 0 hosted success (attempt 3, job 102491330281). Sprint 1 code is locally complete with full check, browser and Linux race evidence, but is not yet published/hosted-verified. Controlled SMTP was authorized and attempted; authentication failed before transmission, leaving the sole Sprint 1 acceptance blocker as external SMTP credentials/receipt evidence. The unused OpenPGP module advisory is disclosed in doc 24; imported packages and symbols have no govulncheck finding.

## Focused Sprint 1 closure review, 2026-09-10

Published d4fed8c and authenticated run 34436595874 supersede the unpublished status above. The hosted checksum failure is explained by the CRLF manifest versus LF Git artifact, with exact hashes in doc 24. The additive correction preserves published SQL/manifest history and strict integrity checks; seven regression tests reject tampering and alternate line endings. The full local check, browser E2E and dependency checks passed again after the correction; corrective commit 9b2c67187eea6fc2ee6b5a1929f063e76cc6797b passed full hosted run 34437626714 (job 102745898048). Brevo accepted all four controlled identity emails after operator credentials were corrected; the operator initially confirmed 3/4 receipts, then confirmed the delayed invitation; all four receipts are now established. No accepted identity architecture or SMTP implementation was redesigned; no Sprint 2 work was added. Gates B/C/D remain CLOSED.

Closure review outcome: Sprint 1 IMPLEMENTATION COMPLETE / HOSTED CI VERIFIED for that corrective commit. All requested isolation, concurrency, session/CSRF, MFA/recovery, authorization/Owner, outbox and audit regressions passed locally and with the hosted race detector. No new application security finding; the known unused OpenPGP module advisory remains disclosed. Acceptance is COMPLETE after the operator confirmed all four controlled receipts, including the delayed invitation. No Sprint 1 evidence blocker remains. Gates B/C/D remain CLOSED; no Sprint 2 work.

## Gate B Phase A review, 2026-09-10

The owner accepts Sprint 1 and authorizes current contract/account research, with Sprint 2 conditional on complete Gate B evidence. Reviewed README/PLANS, capability/research records, ADRs 005/006/008, fixture provenance and current identity hosted outcome. Official Postman has useful unversioned/historical surfaces, including WABA-level subscriptions; developer version/security/error/permission references remain unavailable. No version or signature contract was inferred from old examples.

[Gate B evidence](25-gate-b-evidence.md) separates contract quality, account verification and scope, chooses protected company System User custody and records absent located Meta configuration. No actual token/account fixture or live probe exists. Source status was reconciled across data/API/webhook/observability/test plans without changing Sprint 0/1 architecture, executable contracts, migrations or CI. [Sprint 2](26-sprint-2-evidence.md) remains NOT STARTED; Gate B CLOSED. No Meta mutation, WhatsApp send or Sprint 3 work occurred. No new runtime security defect was established by this documentation-only pass.

## Gate B resumption review, 2026-09-11

Validated the exact operator-supplied path: .local/meta-gate-b.env MISSING. A private, ignored runtime validator reports only presence/format; no secret was available/read and no account request executed. Nearby filename inspection found no matching config, and the operator was asked to correct the path. The candidate v26.0 probe is authorized without silently downgrading or claiming final contract approval first.

Current official release, webhook and debug-token references remained unavailable. Official-domain indexed searches supplied no usable current contract; an official Horizon overview is adjacent-product evidence only, and third-party mirrors were excluded. Matrix/account classifications remain unpromoted. No application, migration, endpoint, signature handling or fixture implementation was added. The new owner instruction requires stopping after Gate B for separate review; Sprint 2 remains NOT STARTED. No WhatsApp send or Meta mutation occurred.

Operator configuration follow-up, 2026-09-11: added all eight agreed Gate B variables to [.env.example](../.env.example), with blank values, explicit candidate-version guidance, optional inventory identifiers and separate secret-file paths. The example directs operators to .local/meta-gate-b.env; it introduces no runtime Meta adapter or implicit environment loading. Source/link/checksum validation passed after the edit.


## Successful Gate B read-only follow-up, 2026-09-11

Operator config and token/App Secret paths are now valid; Verify Token file remains empty. Missing verification secret did not block independent GETs. Ten v26.0 GETs passed across initial checks and explicit provenance/profile capture; response version matched, no fallback occurred. One supplied WABA/phone and matching App subscription were observed. Token debug reports USER with WA/WM but no BM; expiry 2026-09-11T13:00:00Z. Production System User remains unverified and current webhook/lifecycle sources unavailable.

Reviewed four sanitized real fixtures, preserving source hashes privately and sanitized checksums publicly. Runtime exclusion checks found none of the actual secrets or asset IDs in the artifacts. Phone code_verification_status NOT_VERIFIED is recorded without inferring registration/connectivity or mutating the number. Profile field selection was based on an official Postman example; only actually returned fields were recorded. No signature handler, Meta application integration, migration or Sprint 2 UI/code was added. Gate B remains CLOSED, pending separate review and missing credential/contract evidence.

Operator-reported token replacement was rechecked at 12:14:43 UTC: five further GETs passed, but Meta still reports USER (expiry 14:00 UTC). No credential type was inferred from operator intent. Prior capture provenance remains intact; no repeated mutation, subscription change or send occurred.

Latest System User verification, 2026-09-11 12:21 UTC: actual Meta response now confirms SYSTEM_USER, valid and matching App, with WA/WM and no BM. All five read-only checks pass; expiry 2026-11-10T12:20:59Z. Fixture metadata/hashes refreshed from the new responses, with original hashes kept privately and redaction exclusion rechecked. Earlier USER observations are historical. Token blocker is resolved; Gate B remains CLOSED for current authoritative webhook/lifecycle evidence, with Verify Token still empty. No Sprint 2 implementation or Meta mutation was performed.

## Sprint 2 implementation review, 2026-09-12

Gate B owner approval supersedes the earlier stop instruction. The review retains source-access failures and separates live-provider acceptance from local correctness. Added tenant migration and permissions are additive; published identity SQL/checksum history is unchanged. Raw evidence is encrypted and immutable, original routing is trusted-config-bound, callback validation fails closed, and no outbound Graph method or subscription mutation is introduced.

Initial local Meta unit/PostgreSQL tests PASS, including injected storage failure through DLQ, replay dedupe, concurrency, signature byte mutation, forbidden raw modification and all nine populated-table RLS reads. Full-suite, browser, hosted commit and real callback evidence are still pending at this review stage. Evidence status is maintained in [Sprint 2 record](26-sprint-2-evidence.md).

Release review update: implementation COMPLETE with full local check exit 0, identity/Meta browser tests, all-table RLS WITH CHECK and cross-organization HTTP access/replay denial. Browser verification found and fixed missing Meta permission discovery. Live runtime sync passed on the company assets; callback-only HTTPS boundary checks passed. At that pre-publication review, real Meta challenge/POST and own hosted CI remained acceptance evidence to record; the hosted result below supersedes only that CI status. New schema hashes are LF-based; old published migrations/manifests remain unchanged.

## Sprint 2 hosted verification and acceptance review, 2026-09-12

Published implementation commit e0a65503e56727041a00899b9663878a86c8eb39 passed its own hosted run [34670353309](https://github.com/mbingsdk/whats/actions/runs/34670353309) (foundation job 103490430582, 03:26:10Z-03:28:27Z). All repository checks executed successfully, including Linux race tests, real PostgreSQL integration, both browser suites, OpenAPI/build and dependency review. CI and published migrations remain unchanged during this evidence reconciliation. No historical billing failure or Sprint 1 run is reused as Sprint 2 verification.

The controlled listener expired normally with zero events and no valid Meta challenge or authentic POST. Its harness PASS is not provider-delivery evidence. The remaining temporary tunnel was stopped; the earlier proposed callback is inactive and never configured in Meta. Callback mutation requires a newly prepared reachable target, explicit target-specific approval, audit and rollback. README, plan, sprint/checklist and evidence records now distinguish implementation COMPLETE / hosted CI VERIFIED from acceptance OPEN. Gate B APPROVED; Gates C/D CLOSED; no outbound WhatsApp or Sprint 3 work.

## Local bootstrap configuration correction, 2026-09-12

Operator troubleshooting established that dev.py bootstrap inherited only process environment despite the populated private operator file. Add explicit --env-file loading for bootstrap/backend/mailworker/metaworker, with literal parsing, known configuration names, redacted errors and rejection for frontend/CI commands. Local identity/runtime credentials and root-key custody remain pinned; bootstrap variables do not enter other server commands. A second observed rejection was a password shorter than the existing 12-character minimum. The CLI now identifies that constraint without printing the password; no password rule or Owner invariant is weakened. No Owner was created by the failed attempts. After the operator corrected the private password file, the exact --env-file bootstrap command completed with exit 0 and initialized the Owner. Email verification and MFA enrollment remain operator follow-up; no mail worker was started by this correction. Gate B remains APPROVED, Sprint 2 acceptance OPEN, Gates C/D CLOSED; no Meta mutation or sending is part of this correction.

Verification: all 12 Python regression tests passed (including five operator-config tests); the full python scripts/dev.py check completed with exit 0, covering Go unit/real PostgreSQL integration, vet/gofmt, frontend lint/typecheck/tests/build and OpenAPI. Final source validation checked 126 local links and immutable migration checksums. Actual bootstrap succeeded after the operator's password correction; a repeat returned Already initialized without creating or replacing an Owner. Hosted CI has not been rerun for this local correction; earlier Sprint 2 hosted evidence remains tied to its recorded commits.

## Local Owner login origin diagnosis, 2026-09-12

The operator reported PERMISSION_DENIED immediately on login and confirmed using http://127.0.0.1:3000/login. The configured public origin is http://localhost:3000. Bounded probes confirmed that a valid CSRF cookie/header on the 127.0.0.1 origin is rejected with 403 before authentication; the localhost origin reaches authentication. The Owner is active with the expected ORG-scoped owners.manage grant. Actual protected-credential login on localhost returned 200, followed by successful session lookup, organization selection and permission discovery; only the diagnostic session was logged out. No role grant, password, verification flag or Origin/CSRF security rule was changed. The deployment runbook now explains the exact-origin requirement. This operational check is not a new hosted CI result.
