# Production readiness evidence

Historical owner override (2026-09-09): **Sprint 0 IMPLEMENTATION COMPLETE; Gate A APPROVED; Sprint 1 AUTHORIZED. Hosted CI verification PENDING - EXTERNAL BLOCKER.** Run 34356265759 was attempted and failed before repository steps because GitHub reported an account billing lock. Accepted local evidence closes implementation, not hosted verification. CI requirements remain intact; rerun hosted CI when available, record the real result and fix any repository failures. Gates B/C/D remain CLOSED.

Current verification (2026-09-10): **Sprint 0 IMPLEMENTATION COMPLETE; Gate A APPROVED; Sprint 1 implementation COMPLETE, acceptance OPEN.** GitHub run 34356265759 attempt 3 actually executed and passed the Sprint 0 baseline at commit 993c67e93fe5945ec820efe67d9d3abf3df8b1b3. Its original attempt 1 failed before steps due to the account billing restriction. Sprint 1 hosted verification is PENDING for these unpublished workspace changes; do not treat the Sprint 0 pass as Sprint 1 evidence. Controlled SMTP was ATTEMPTED and rejected authentication (SMTP_AUTH), so mailbox receipt remains PENDING. Gates B/C/D remain CLOSED.

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

[Sprint 0 evidence](23-sprint-0-evidence.md) is partial support for P01/P12/P17/P21 only where explicitly tested. No product release criterion above is marked passed; no account verification, production mail delivery, restore drill or messaging test has occurred.

## Sprint 1 evidence boundary

[Identity evidence](24-sprint-1-evidence.md) adds local authentication/MFA/offboarding, nine new tenant-table isolation checks, permission/concurrency tests, frontend browser flows and authenticated TLS SMTP delivery tests. P21 remains OPEN for controlled external mailbox receipt. Sprint 1 hosted verification remains pending independently of implementation correctness; the Sprint 0 baseline now has actual hosted success. The attempted run 34356265759 failed before any repository steps due to the external account billing restriction. Neither local SMTP acceptance nor passing local tests is a hosted CI pass or production release approval; Gates B/C/D remain CLOSED.
