# Work plan and phase gates

## M73 acceptance evidence review, 2026-09-16

M73 follow-up (2026-09-16): general Cloud API Service delivery TTL of 30 days is now documented, independently of retention. A current send's possible delivery crosses October pricing; exact acceptance-clock/rate-lock proof remains incomplete. Safe zero-cost acceptance remains OPEN and the runtime guard is unchanged. No live recipient/callback/send setup was performed.

## Sprint 3 implementation review, 2026-09-15

Gates A/B/C APPROVED; Sprints 0/1/2 COMPLETE. Sprint 3 implementation COMPLETE: local checks and its own hosted CI passed for commit a498471 (run 34970609872). Live acceptance is OPEN. Company billing currency UNKNOWN; paid/template authority and Gate D CLOSED. Sprint 4 unauthorized. See [Sprint 3 evidence](docs/28-sprint-3-evidence.md).

Owner clarification (2026-09-15): **Sprints 0/1/2 COMPLETE; Gates A/B/C APPROVED; Gate D CLOSED; Sprint 3 AUTHORIZED. COMPANY META BILLING CURRENCY UNKNOWN; PAID-SEND AUTHORITY CLOSED.** Currency must not be inferred from timezone, business/phone/recipient country, available rate cards or locale. Only a currently reviewed, provably zero-cost policy can permit the single operator-triggered controlled TEXT live reply after all recipient/callback/security prerequisites. Paid/template live sends, outbound media, campaigns and Sprint 4 remain unauthorized.

Earlier dated status/review entries below are historical and superseded by this owner clarification where they describe Gate C or Sprint 3 authorization.

Acceptance review (2026-09-14; live evidence captured 2026-09-12): **Gate B APPROVED; Sprint 2 implementation COMPLETE; Sprint 2 acceptance COMPLETE; Gates C/D CLOSED.** Real Meta GET challenge, authentic signed POST, durable ingestion, processing, authorized Event Center and replay evidence are verified. Sprint 0/1 acceptance remains COMPLETE. No outbound WhatsApp or Sprint 3 work is authorized.

## Current implementation and verification

Historical owner override (2026-09-09): **Sprint 0 IMPLEMENTATION COMPLETE; Gate A APPROVED; Sprint 1 AUTHORIZED. Hosted CI verification PENDING - EXTERNAL BLOCKER.** Run 34356265759 was attempted and failed before repository steps because GitHub reported an account billing lock. Accepted local evidence closes implementation, not hosted verification. CI requirements remain intact; rerun hosted CI when available, record the real result and fix any repository failures. Gates B/C/D remain CLOSED.

Historical Sprint 1 verification (2026-09-10): **Sprint 0 IMPLEMENTATION COMPLETE; Gate A APPROVED; Sprint 1 IMPLEMENTATION COMPLETE; HOSTED CI VERIFIED; Sprint 1 acceptance COMPLETE.** Sprint 1 was published at d4fed8ce7b50ed44f0f356e9ebea8f21695fe7bd. Run 34436595874 executed and failed in scripts/check.py because the 00002 manifest hashed CRLF working bytes while Git published LF bytes; this was a repository failure, not billing. Corrective commit 9b2c67187eea6fc2ee6b5a1929f063e76cc6797b passed the full hosted workflow in run 34437626714. Published SQL and historical manifest remain unchanged. Sprint 0 run 34356265759 attempt 1 remains a historical billing failure and attempt 3 passed only the Sprint 0 baseline. Controlled Brevo SMTP accepts all four identity emails; the operator confirms actual receipt of verification, reset, security and invitation emails. Gates B/C/D CLOSED; no Sprint 2 work. Detailed evidence is in docs/24-sprint-1-evidence.md.

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

Gate B: **APPROVED** by the product owner for the existing v26.0 account topology and bounded ingestion scope. Official Postman, inspected first-party signature examples and successful SYSTEM_USER account reads are accepted with explicit source limitations. Protected Verify Token, public HTTPS, real GET challenge and exact-byte signed POST proof are Sprint 2 acceptance conditions. No permissive authenticity fallback is allowed.

Historical Gate C interpretation (superseded 2026-09-15): **CLOSED**, enable paid test sends: current official rate policy and rate-card evidence (including known effective-date changes) are approved; currency, markets, taxes/provider charges and reconciliation limitations documented; budget and approval policies set; consent evidence available; pricing and concurrency tests pass. Test sends are real sends and use these same gates.

Gate D: **CLOSED**, production: all blocking rows in doc 20 have evidence, named owners and dates; D1–D7 resolved where relevant; restore drill meets accepted recovery targets; offboarding, replay, suppression and uncertain-send drills pass; production account capability probes pass; operations owner accepts residual risks. A checked box without evidence is not completion.

## Historical Sprint 2 authorization and acceptance

Acceptance review (2026-09-14; live evidence captured 2026-09-12): **Gate B APPROVED; Sprint 2 implementation COMPLETE; Sprint 2 acceptance COMPLETE; Gates C/D CLOSED.** Real Meta GET challenge, authentic signed POST, durable ingestion, processing, authorized Event Center and replay evidence are verified. Sprint 0/1 acceptance remains COMPLETE. No outbound WhatsApp or Sprint 3 work is authorized.

Existing-asset read synchronization, durable signed webhook ingestion and operational identity-authorized surfaces are complete. Callback/subscription mutation still requires a concrete current-versus-proposed target review, explicit operator approval and rollback information. [Sprint 2 evidence](docs/26-sprint-2-evidence.md) records passing local implementation checks and hosted run [34670353309](https://github.com/mbingsdk/whats/actions/runs/34670353309) for e0a6550. Live Meta GET/POST, authorized Event Center and captured-event replay are VERIFIED; the approved callback rollback and tunnel shutdown are complete. Sprint 2 acceptance is COMPLETE; Gates C/D remain CLOSED and Sprint 3 is not authorized.

## Sprint 0 implementation record

Gate A approval accepts D2 (50 members, 25 concurrent agents, 5 WABAs/20 numbers, 300k monthly messages and 100k snapshot recipients) solely as provisional test/capacity assumptions. D5 is the provisional 99.5% monthly internal target, RPO 15 minutes and RTO 4 hours; single VPS is not highly available. These are not measured performance or contractual/Meta limits.



Sprint 0 accepted local evidence and the attempted hosted run are preserved in [the Sprint 0 record](docs/23-sprint-0-evidence.md). The external billing restriction is not a repository test failure. Rerun hosted CI when account access is restored and repair any real test failures before claiming hosted verification.

## Sprint 3 live acceptance remaining

Implementation is published at a49847151bb6ea3592856d9989201375102eca40 with its own passing hosted run 34970609872. Gate C baseline efe31864e5ed0c090d12c292f269eec22941bfb6 passed run 34879123286; this is not Sprint 3 verification. Keep live acceptance OPEN until the operator designates one TEST recipient, approves current callback/rollback, supplies fresh signed inbound and personally triggers one provably free reply. A general 30-day Service delivery TTL is now verified, but usable acceptance/pricing coverage remains missing; synthetic bounds or Direct Send categories cannot replace it.

Currency/paid-template authority are separate future owner decisions, not prerequisites for local guard implementation. No paid/template test is performed or required for the accepted free-reply scope. Gate D and Sprint 4 remain closed.
