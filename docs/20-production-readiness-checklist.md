# Production readiness evidence

Owner clarification (2026-09-15): **Sprints 0/1/2 COMPLETE; Gates A/B/C APPROVED; Gate D CLOSED; Sprint 3 AUTHORIZED. COMPANY META BILLING CURRENCY UNKNOWN; PAID-SEND AUTHORITY CLOSED.** Currency must not be inferred from timezone, business/phone/recipient country, available rate cards or locale. Only a currently reviewed, provably zero-cost policy can permit the single operator-triggered controlled TEXT live reply after all recipient/callback/security prerequisites. Paid/template live sends, outbound media, campaigns and Sprint 4 remain unauthorized.

Earlier dated status/review entries below are historical and superseded by this owner clarification where they describe Gate C or Sprint 3 authorization.

Acceptance review (2026-09-14; live evidence captured 2026-09-12): **Gate B APPROVED; Sprint 2 implementation COMPLETE; Sprint 2 acceptance COMPLETE; Gates C/D CLOSED.** Real Meta GET challenge, authentic signed POST, durable ingestion, processing, authorized Event Center and replay evidence are verified. Sprint 0/1 acceptance remains COMPLETE. No outbound WhatsApp or Sprint 3 work is authorized.

Historical owner override (2026-09-09): **Sprint 0 IMPLEMENTATION COMPLETE; Gate A APPROVED; Sprint 1 AUTHORIZED. Hosted CI verification PENDING - EXTERNAL BLOCKER.** Run 34356265759 was attempted and failed before repository steps because GitHub reported an account billing lock. Accepted local evidence closes implementation, not hosted verification. CI requirements remain intact; rerun hosted CI when available, record the real result and fix any repository failures. Gates B/C/D remain CLOSED.

Historical Sprint 1 verification (2026-09-10): **Sprint 0 IMPLEMENTATION COMPLETE; Gate A APPROVED; Sprint 1 IMPLEMENTATION COMPLETE; HOSTED CI VERIFIED; Sprint 1 acceptance COMPLETE.** Sprint 1 was published at d4fed8ce7b50ed44f0f356e9ebea8f21695fe7bd. Run 34436595874 executed and failed in scripts/check.py because the 00002 manifest hashed CRLF working bytes while Git published LF bytes; this was a repository failure, not billing. Corrective commit 9b2c67187eea6fc2ee6b5a1929f063e76cc6797b passed the full hosted workflow in run 34437626714. Published SQL and historical manifest remain unchanged. Sprint 0 run 34356265759 attempt 1 remains a historical billing failure and attempt 3 passed only the Sprint 0 baseline. Controlled Brevo SMTP accepts all four identity emails; the operator confirms actual receipt of verification, reset, security and invitation emails. Gates B/C/D CLOSED; no Sprint 2 work. Detailed evidence is in docs/24-sprint-1-evidence.md.

| Gate | Blocking criterion | Evidence required | Accountable role |
| --- | --- | --- | --- |
| P01 | Organization/team/phone isolation | HTTP, SQL runtime-role/RLS, SSE, search, exports/media negative tests across two tenants | Technical lead |
| P02 | Existing Meta connection safe | Asset/scope/binding inventory, existing subscription diff, test account result | Meta operator |
| P03 | Current API contract verified | Official Graph version/removal date, exact signature and scoped message fixtures, constraints register | Integration lead |
| P04 | Ingress durable/recoverable | ACK-after-commit test, multi-item/dedupe/reorder/unknown fixtures, DB outage recovery | Backend lead |
| P05 | No blind send retries | Crash-at-each-boundary tests, UNCERTAIN queue and reviewed replacement procedure | Backend lead |
| P06 | Window/consent valid | Boundary/skew/replay tests; opt-out/import/merge and final dispatch race | Product/policy owner |
| P07 | Pricing evidence complete | Reviewed Registry import/validation/diff/publication and immutable correction evidence; current official rates/effective dates, currencies/markets, coverage/review deadlines and fixture calculations | Finance + integration |
| P08 | Atomic budgets/confirmations | Concurrent ceiling/token tests; ledger reconciliation; external sender accounting policy | Backend + finance |
| P09 | Approval cannot bypass | Quorum/self-approval/delegation/sponsor/revision invalidation tests | Security lead |
| P10 | Campaign safe to operate | Snapshot consistency, frequency, rate control, pause/cancel/in-flight evidence, operator walkthrough | Outreach owner |
| P11 | Offboarding and MFA | Revocation of sessions/streams/keys/exports/queued personal effects; recovery-admin procedure | Security + administrator |
| P12 | Secret/media protections | Rotation/decrypt-restore drill, key separation, malware/MIME/URL scope/SSRF tests | Security + operations |
| P13 | Retention approved and executable | Company/legal policy, erasure/hold jobs, suppression tombstones, restore-purge test | Policy owner |
| P14 | Developer delivery safe | Signature verification/retry/replay/idempotency and revoked-scope tests | Integration lead |
| P15 | Performance and diagnostics | Mixed agents/webhooks/campaign/opt-out/budget benchmark; per-lock wait/hold and deny latency, doc 05 redesign thresholds, EXPLAIN/lag recovery; alert owner | Technical + operations |
| P16 | Restore objective met | Timed isolated restore, WAL/key/object checks, erasure ledger, uncertainty quarantine within accepted RPO/RTO | Operations |
| P17 | Deployment/rollback proven | Compatible migration rollout/rollback, graceful shutdown, kill/restart tests | Operations |
| P18 | UI usable and accessible | C01-C05 reply/takeover, C09 draft refresh/logout/org-switch tests; agent tasks, keyboard/screen-reader, contrast and mobile error states | Product/design |
| P19 | Monitoring and duty ownership | Alert routing tested, backup/DNS/keys accessible to backup operator, escalation roster | Operations owner |
| P20 | Release scope and residual risks accepted | Named acceptance of single-host downtime, uncertain sends, billing coverage and disabled optional modules | Company owner |
| P21 | Transactional identity mail operational | SMTP TLS/configuration and controlled-mailbox evidence; invite/verify/reset/security notices, one-use/expiry/resend and provider failure tests, redacted logs | Identity + operations |

Optional module release requires its own evidence package; unchecked Calling/Coexistence does not block a core-only release if routes/UI/jobs remain disabled. A missing core pricing/signature/isolation gate cannot be relabeled optional to ship sending.

[Sprint 0 evidence](23-sprint-0-evidence.md) is partial support for P01/P12/P17/P21 only where explicitly tested. This historical foundation evidence does not establish Meta account verification, a restore drill or messaging readiness. Sprint 1 controlled SMTP acceptance is recorded separately below.

## Historical Sprint 1 evidence boundary

[Identity evidence](24-sprint-1-evidence.md) adds local authentication/MFA/offboarding, nine new tenant-table isolation checks, permission/concurrency tests, frontend browser flows and authenticated TLS SMTP delivery tests. P21 controlled-mailbox evidence is satisfied for Sprint 1: all four identity messages passed real verified STARTTLS/authentication/MAIL/RCPT/DATA and operator-confirmed receipt. This does not close the other production readiness requirements. Sprint 1 run 34436595874 executed and failed migration checksum validation; corrective commit 9b2c671 passed full hosted run 34437626714. The Sprint 0 baseline has actual attempt-3 hosted success; only its historical attempt 1 failed before repository steps due to billing. Neither local SMTP acceptance nor passing local tests is a hosted CI pass or production release approval; Gates B/C/D remain CLOSED.

## Historical pre-approval Gate B evidence boundary

P02 and P03 remain OPEN: [Phase A](25-gate-b-evidence.md) has successful v26.0 authentication/inventory/subscription GETs and four sanitized fixtures, now with verified SYSTEM_USER, but without current authoritative webhook/lifecycle contract. P04 ingestion/recovery and the Meta portions of P01/P12/P19 have no Sprint 2 implementation evidence. The conditional [Sprint 2 record](26-sprint-2-evidence.md) does not reuse Sprint 1 hosted success. Gates B/C/D remain CLOSED. No subscription mutation or WhatsApp message occurred.

## Sprint 2 acceptance versus production

Gate B approval supersedes the historical CLOSED statements above. Sprint 2 implementation supplies tested ingestion/recovery, authorization, diagnostics and tenant-isolation evidence; hosted run [34670353309](https://github.com/mbingsdk/whats/actions/runs/34670353309) passed on e0a6550, and the live GET metadata correction passed its own [run 34813315965](https://github.com/mbingsdk/whats/actions/runs/34813315965). [Sprint 2 live evidence](26-sprint-2-evidence.md) records real Meta GET verification, authentic signed POST with durable encrypted persistence before ACK, processing, authorized Event Center and idempotent audited replay on 2026-09-12. Sprint 2 acceptance is COMPLETE.

These results support the tested portions of P01/P02/P03/P04/P12/P19 for the supplied App/WABA and bounded ingestion scope. They do not establish every provider contract, other account capabilities or completion of the production checklist. The public endpoint was a TEMPORARY_ACCEPTANCE_TUNNEL; the approved App callback was deleted back to the observed empty state, the existing WABA subscription retained, and the tunnel/listeners stopped. No permanent production callback is claimed. Gates C/D remain CLOSED; production deployment, restore and operations requirements remain separate. No outbound WhatsApp or Sprint 3 work is authorized.

## Gate C evidence review, 2026-09-14

[Gate C decision](27-gate-c-evidence.md): **CLOSED**, immediate blocker company billing currency UNKNOWN. Sprint 0/1/2 acceptance remains COMPLETE; Gates A/B APPROVED; Gate D CLOSED; Sprint 3 NOT STARTED.

P03 now has bounded current message-type/status contract evidence and six successful read-only Graph GETs, not outbound account acceptance. P05/P06 have explicit no-blind-retry/window decisions and offline boundary fixtures, not implemented send-path tests. P07 has actual official current/future Indonesia USD/IDR artifacts and a DRAFT Registry rehearsal; it does not have an approved company rate publication. P08 has a defined TEST hard-stop/confirmation policy, not executed database budget races or an owner-approved amount. These production criteria are not marked passed from documentation alone.

The October 1 service/utility change and observed America/Los_Angeles WABA timezone are part of rate coverage. Proposed source review deadline: 2026-09-21T00:00:00Z, pending reviewer acceptance. Currency, external usage/payment details, safe recipient/template selection, exact spending authority and approved callback remain explicit before applicable live tests. Sprint 2's callback was restored to empty; a temporary tunnel is not a permanent endpoint. No sends or account mutations occurred in Gate C.
