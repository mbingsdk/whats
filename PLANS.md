# Work plan and phase gates

## Current implementation and verification

Historical owner override (2026-09-09): **Sprint 0 IMPLEMENTATION COMPLETE; Gate A APPROVED; Sprint 1 AUTHORIZED. Hosted CI verification PENDING - EXTERNAL BLOCKER.** Run 34356265759 was attempted and failed before repository steps because GitHub reported an account billing lock. Accepted local evidence closes implementation, not hosted verification. CI requirements remain intact; rerun hosted CI when available, record the real result and fix any repository failures. Gates B/C/D remain CLOSED.

Current verification (2026-09-10): **Sprint 0 IMPLEMENTATION COMPLETE; Gate A APPROVED; Sprint 1 closure verification OPEN.** Sprint 1 was published at d4fed8ce7b50ed44f0f356e9ebea8f21695fe7bd. Run 34436595874 executed and failed in scripts/check.py: the 00002 manifest hashed Windows CRLF bytes, while Git published LF bytes. This is a repository failure, not billing. An additive checksum correction preserves the published SQL and historical manifest; corrective hosted verification is PENDING. Sprint 0 run 34356265759 attempt 1 remains a historical billing failure and attempt 3 passed only the Sprint 0 baseline. Controlled Brevo SMTP now accepts all four identity emails; mailbox receipt is confirmed for verification/reset/security; invitation receipt remains PENDING. Sprint 1 acceptance OPEN; Gates B/C/D CLOSED. See docs/24-sprint-1-evidence.md for exact evidence.

## Decision register

These are provisional internal design assumptions, not facts about the company.

| ID | Decision needed | Proposed baseline | Consequence if unanswered |
| --- | --- | --- | --- |
| D1 | Existing Meta Cloud API WABA confirmed by user; exact app/ownership topology still needed | Connect/synchronize existing assets; no initial migration or re-registration | Verify asset assignments and existing consumers before subscription changes |
| D2 | Agents, volume, peaks, countries | 50 members, 25 concurrent agents, 5 WABAs, 20 numbers; 300k messages/month; 100k campaign snapshot | ACCEPTED by product owner as provisional testing/capacity assumptions; not product limits or Meta throughput promises |
| D3 | Billing currency, market mix, budget limits and approvers | One organization accounting currency; all commercial values unset | All potentially paid sends blocked until configured |
| D4 | Deployment region, privacy obligations and retention | Single Linux VPS; encrypted offsite backups; proposed retention in doc 11 | Production PII onboarding blocked until owner accepts policy |
| D5 | Availability and recovery objectives | 99.5% monthly internal availability target; RPO 15 min, RTO 4 h | ACCEPTED provisionally by product owner; single VPS is not HA. Targets may change after measurements; restore proof still required |
| D6 | Authentication and identity source | Local password + TOTP; privileged users must enroll MFA | Provider-neutral SMTP boundary accepted for Sprint 0; controlled real-mailbox delivery required before Sprint 1 identity email is complete; SSO/passkeys deferred |
| D7 | Languages, timezone, business hours | English UI initially; IANA timezone explicitly set per organization | Asia/Makassar is a proposal, not inferred billing jurisdiction |
| D8 | First automation/Flow use case | Keyword routing and human handoff; one static Flow after messaging | Dynamic data exchange and arbitrary workflow builder deferred |

## Gate decisions

Gate A is APPROVED; Sprint 1 is AUTHORIZED. Hosted account restrictions do not block product implementation.

Gate B: **CLOSED**, implement Meta adapters against test assets: select and record a supported Graph version and removal date from official documentation; confirm messaging and management permissions/asset assignment; read current signature/verification and payload specifications; obtain sanitized fixtures with provenance; resolve the exact message subtypes in scope; verify the connection topology under D1. Keep optional adapters disabled. Changes to schema/contracts from verification are reviewed before coding those adapters.

Gate C: **CLOSED**, enable paid test sends: current official rate policy and rate-card evidence (including known effective-date changes) are approved; currency, markets, taxes/provider charges and reconciliation limitations documented; budget and approval policies set; consent evidence available; pricing and concurrency tests pass. Test sends are real sends and use these same gates.

Gate D: **CLOSED**, production: all blocking rows in doc 20 have evidence, named owners and dates; D1–D7 resolved where relevant; restore drill meets accepted recovery targets; offboarding, replay, suppression and uncertain-send drills pass; production account capability probes pass; operations owner accepts residual risks. A checked box without evidence is not completion.

## Current authorized work

Finish the focused Sprint 1 closure pass: publish the evidenced checksum correction, obtain a full green hosted workflow, and confirm controlled SMTP receipt. Do not start Sprint 2. The [Sprint 1 evidence record](docs/24-sprint-1-evidence.md) separates implementation checks, controlled SMTP acceptance and hosted verification. Keep Sprint 1 acceptance OPEN for missing controlled mailbox evidence; do not start Meta or any sending feature.

## Sprint 0 implementation record

Gate A approval accepts D2 (50 members, 25 concurrent agents, 5 WABAs/20 numbers, 300k monthly messages and 100k snapshot recipients) solely as provisional test/capacity assumptions. D5 is the provisional 99.5% monthly internal target, RPO 15 minutes and RTO 4 hours; single VPS is not highly available. These are not measured performance or contractual/Meta limits.



Sprint 0 accepted local evidence and the attempted hosted run are preserved in [the Sprint 0 record](docs/23-sprint-0-evidence.md). The external billing restriction is not a repository test failure. Rerun hosted CI when account access is restored and repair any real test failures before claiming hosted verification.
