# Work plan and phase gates

## Completed in this phase

Empty workspace inspected; product requirements analyzed; official evidence researched; architecture, schema, contracts, UI behavior, operations, tests and ADRs documented; second-pass review incorporated. Gate A was APPROVED by the product owner on 2026-09-09; Sprint 0 implementation is authorized and OPEN. Sprint 1 and product features remain unauthorized.

## Decision register

These are provisional internal design assumptions, not facts about the company.

| ID | Decision needed | Proposed baseline | Consequence if unanswered |
| --- | --- | --- | --- |
| D1 | Existing Meta Cloud API WABA confirmed by user; exact app/ownership topology still needed | Connect/synchronize existing assets; no initial migration or re-registration | Verify asset assignments and existing consumers before subscription changes |
| D2 | Agents, volume, peaks, countries | 50 members, 25 concurrent agents, 5 WABAs, 20 numbers; 300k messages/month; 100k campaign snapshot | ACCEPTED by product owner as provisional testing/capacity assumptions; not product limits or Meta throughput promises |
| D3 | Billing currency, market mix, budget limits and approvers | One organization accounting currency; all commercial values unset | All potentially paid sends blocked until configured |
| D4 | Deployment region, privacy obligations and retention | Single Linux VPS; encrypted offsite backups; proposed retention in doc 11 | Production PII onboarding blocked until owner accepts policy |
| D5 | Availability and recovery objectives | 99.5% monthly internal availability target; RPO 15 min, RTO 4 h | ACCEPTED provisionally by product owner; single VPS is not HA. Targets may change after measurements; restore proof still required |
| D6 | Authentication and identity source | Local password + TOTP; privileged users must enroll MFA | SMTP transactional mail configuration/sender selected in Sprint 0 and verified in Sprint 1; SSO/passkeys deferred |
| D7 | Languages, timezone, business hours | English UI initially; IANA timezone explicitly set per organization | Asia/Makassar is a proposal, not inferred billing jurisdiction |
| D8 | First automation/Flow use case | Keyword routing and human handoff; one static Flow after messaging | Dynamic data exchange and arbitrary workflow builder deferred |

## Safe to begin implementation

Gate A: **APPROVED**, product owner, 2026-09-09. The current architecture package, D2 capacity baseline and D5 availability/recovery baseline are accepted for Sprint 0. **SPRINT 0 ONLY is authorized**. This does not authorize Sprint 1, Meta adapters, production services, production asset mutations or real messaging. Gates B/C/D remain CLOSED.

Gate B: **CLOSED**, implement Meta adapters against test assets: select and record a supported Graph version and removal date from official documentation; confirm messaging and management permissions/asset assignment; read current signature/verification and payload specifications; obtain sanitized fixtures with provenance; resolve the exact message subtypes in scope; verify the connection topology under D1. Keep optional adapters disabled. Changes to schema/contracts from verification are reviewed before coding those adapters.

Gate C: **CLOSED**, enable paid test sends: current official rate policy and rate-card evidence (including known effective-date changes) are approved; currency, markets, taxes/provider charges and reconciliation limitations documented; budget and approval policies set; consent evidence available; pricing and concurrency tests pass. Test sends are real sends and use these same gates.

Gate D: **CLOSED**, production: all blocking rows in doc 20 have evidence, named owners and dates; D1–D7 resolved where relevant; restore drill meets accepted recovery targets; offboarding, replay, suppression and uncertain-send drills pass; production account capability probes pass; operations owner accepts residual risks. A checked box without evidence is not completion.

## Next proposed work

Sprint 0 is specified in doc 19. It is now authorized and OPEN; completion requires executed evidence, not source-file existence. Production account mutation, campaigns and advanced Meta capabilities are later gates; this plan is not permission to execute them now.

## Sprint 0 implementation record

Gate A approval accepts D2 (50 members, 25 concurrent agents, 5 WABAs/20 numbers, 300k monthly messages and 100k snapshot recipients) solely as provisional test/capacity assumptions. D5 is the provisional 99.5% monthly internal target, RPO 15 minutes and RTO 4 hours; single VPS is not highly available. These are not measured performance or contractual/Meta limits.

Implemented work and executed results are maintained in [Sprint 0 evidence](docs/23-sprint-0-evidence.md). Sprint 0 remains **OPEN**; a hosted CI result and unresolved official Meta version/webhook contract evidence remain outstanding. SMTP controlled-mailbox validation awaits an approved relay/sender and designated recipient. No Sprint 1 work is authorized by this record. Gates B/C/D remain CLOSED.
