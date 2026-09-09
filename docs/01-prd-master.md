# Master PRD

Design baseline: 2026-09-09. Evidence was inspected on 2026-09-08. Doc 02 governs Meta capability claims; this document specifies internal product behavior.

## Requirements and acceptance

| ID | Requirement and observable acceptance | Detail |
| --- | --- | --- |
| ORG-1 | A member can join multiple teams; organization switching is shown only with more than one active organization membership. Another tenant's resource ID yields 404 without object details. | 05, 14 |
| ORG-2 | An inviter grants only delegable roles. Acceptance checks verified email and current authority. Deactivation revokes organization access, pending personal sends and live subscriptions; audit attribution survives. | 04, 14 |
| META-1 | Connect the existing app with write-only secret input. Discovery shows accessible owned/shared assets; explicit internal binding is required. Failed synchronization retains prior observations with a stale indicator. | 02, 09 |
| META-2 | Health separates reported status from local availability. Missing access gives redacted diagnostics. Registration, migration and payment-method mutation are absent from initial workflows. | 02, 15 |
| IN-1 | Durable incoming events appear once in the correct number/contact thread. Batched, duplicate and reordered events neither duplicate messages nor regress read status. Unknown payloads render safely. | 08, 09 |
| IN-2 | Reply requires inbox.reply and conversation/number access. Backend reevaluates service window, handoff ownership, suppression, pricing and authority at dispatch. UI displays Pending until external acceptance. | 09, 12 |
| IN-3 | Notes never call Meta. Mentions target authorized members only. Assignment uses revision checks. Unread is per member. Snooze ends on due time or new inbound. | 04, 07 |
| IN-4 | An expired or unknown service window prevents free-form dispatch. UI countdown uses server time and state, never authorizes a send. | 09 |
| COST-1 | A paid send requiring confirmation needs an unexpired, single-use authorization bound to actor, recipient, sender, exact payload, policy and cost ceiling. Changed inputs require new preflight. | 12 |
| COST-2 | Concurrent senders cannot over-reserve matching internal budgets. Missing rates/currency/policy validity blocks sends. Estimates, Meta pricing metadata and reconciled money are distinct. | 05, 12 |
| CRM-1 | Contact records include identifiers, source, fields, tags, consent evidence and timeline. Import preview reports ambiguous numbers and duplicates. Import cannot silently grant marketing consent or remove suppression. | 05, 11 |
| CMP-1 | campaign.create permits drafts, not approval. Submission freezes sender, template version, rendered variables, snapshot, schedule bounds and cost ceiling. Approval enforces policy and separation of duties. | 10, 14 |
| CMP-2 | Final eligibility excludes newly suppressed contacts even after approval. Changed template/category/price can invalidate authorization. Scheduling never bypasses preflight. | 10, 12 |
| CMP-3 | Pause prevents new dispatch permits after commit. Already dispatched requests may finish. Cancel cannot recall messages. Unknown outcomes are not retried automatically. | 04, 10 |
| AST-1 | Templates retain observed versions, validation, status history, preview and usage. UI offers only enabled verified operations. Previews are internal approximations. | 02, 09 |
| AST-2 | Flows retain JSON versions, validation and publishing evidence. Completion correlates through an opaque instance token. Dynamic data exchange is separately gated. | 09, 13 |
| AUT-1 | Automation uses immutable published versions, bounded runs, effect deduplication and logs. Human takeover invalidates bot dispatch authority. HTTP destinations are restricted. | 13 |
| DEV-1 | API keys are scoped to organization, permissions and phones. Gateway requests use the same guards. Repeated client idempotency keys cannot create a second intent. | 06, 14 |
| DEV-2 | Outbound event delivery filters event types and resource access. Notes are excluded. Retries preserve event IDs; revoked endpoints stop queued deliveries. | 07, 08 |
| OPS-1 | Health separates Meta, integration and infrastructure. Replay cannot repeat completed effects. Recovery has a tested offsite restore procedure. | 15, 16 |
| AUD-1 | Approval, export, credential, policy, membership, send and replay actions retain actor, resource, change, timestamp and correlation, without secrets or unrestricted content. | 14 |

## Workflows

Support: verify webhook, persist, parse items, update identity/thread/window, notify authorized clients, then evaluate automation. Reply: preflight, confirmation/approval if required, immutable intent, final send gate, dispatch, status reconciliation. Each stage has an explicit failure state.

Campaign: draft, validate, materialize snapshot, estimate, submit, review, schedule/queue, final recipient eligibility, dispatch, drain outstanding work, complete processing. Completion does not mean every recipient received or read a message.

Offboarding: revoke organization membership and organization sessions, increment authorization revision, invalidate personal unexecuted intents, reassign conversations, transfer stewardship of organization-owned campaigns, retain audit principal. A campaign does not depend on the creator's browser session, but losing its required sponsor pauses it.

State transitions and actors are normative in doc 04. No API permits arbitrary status assignment.

## Nonfunctional acceptance

Proposed targets under the documented workload: durable webhook ACK p95 <500 ms/p99 <2 s; inbox visibility p95 <2 s after receipt under normal queue load; list/detail API p95 <300 ms excluding Meta; inbox interactive p75 <2 s on a representative office connection. These are unmeasured targets. Upstream delivery time is a separate metric.

Security acceptance: tenant-enforced queries, scoped permissions, revocable sessions, privileged MFA, encrypted secrets, raw-body signature verification and PII-limited logs. RPO 15 min/RTO 4 h needs a timed restore drill. Accessibility target is WCAG 2.2 AA with keyboard completion of triage, reply and review.

Retention covers database content, media, raw events, search, exports, audit and restored backups (doc 11). No implicit unlimited retention. Local legal obligations and deployment region are UNKNOWN until the company supplies its operating context.

| Failure/edge case | Required behavior |
| --- | --- |
| Missing phone or unfamiliar identity | Preserve scoped opaque identifier; do not invent E.164 or recipient market |
| Status before send response | Store orphan fact and reconcile later through trusted identifiers |
| Two agents or bot/human collision | Dispatch revision/permit arbitrates; presence is advisory |
| Meta outage or credential revocation | Pause affected sending; retain work and recheck guards after recovery |
| Pricing changes mid-campaign | Reestimate unsent work under effective-date policy; renew authority above bounds |
| Send timeout | Quarantine uncertainty and retain budget exposure |
| Opt-out races with dispatch | Serialize suppression with dispatch authorization; disclose already in-flight boundary |
| Replay after erasure | Tombstones prevent recreation and side effects |

Implementation tickets must reference requirement IDs and attach acceptance evidence. A successful demonstration cannot substitute for isolation, concurrency or restore tests.
