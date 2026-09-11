# Sprint 2 implementation and evidence

Initial review: 2026-09-10; authorization update: 2026-09-11. Sprint title: **META INGESTION + EXISTING ASSETS**.

**Sprint 2 NOT STARTED; completion OPEN. Gate B CLOSED.** The latest owner instruction authorizes Gate B only and requires stopping for separate evidence review even if Gate B becomes APPROVED. Sprint 2 is not authorized in this run. This is a status record, not an implementation claim. Sprint 0/1 remain COMPLETE; Sprint 1 acceptance, hosted CI and controlled SMTP remain VERIFIED.

## Evidence boundary

[Gate B evidence](25-gate-b-evidence.md) records the actual official-source access failures, documented surfaces and remaining authoritative webhook/lifecycle evidence. v26.0 GET probes now verify one supplied WABA, phone, profile and App subscription, with four sanitized fixtures. The replacement credential is verified SYSTEM_USER; current authoritative webhook/lifecycle evidence remains unresolved and Verify Token is empty. No Phase B code was written to bypass those dependencies.

| Deliverable | Actual status |
| --- | --- |
| Migration / tenant tables | None added; published identity migrations unchanged |
| Meta credentials / Graph client | Not implemented; protected System User strategy documented in Gate B record |
| WABA / phone / profile sync | Not implemented; Gate B inventory verified for one supplied WABA/phone; no implemented synchronization |
| GET verification / POST authenticity | Not implemented; current authoritative contract is a mandatory blocker |
| Durable ingress / dedupe / attempts / DLQ / replay | Not implemented; accepted ADR 008 remains the future design |
| WABA subscriptions | Gate B GET confirms configured App already subscribed; no subscribe/unsubscribe/override executed |
| Meta Health / assets / Webhook Event Center | Not implemented; no empty navigation added |
| OpenAPI / frontend | Sprint 1 executable surfaces unchanged |
| Sprint 2 tests / live webhook / hosted CI | No Sprint 2 tests/live webhook/hosted run; Gate B read-only probes passed separately |
| Outbound WhatsApp | Zero messages sent; no sending capability enabled |

## Conditional implementation boundary

After Gate B approval, separate owner review and implementation authorization, implement a narrow backend-only Graph boundary and existing-asset synchronization, then signed durable ingestion and asynchronous classification of messages, statuses, relevant asset/template events and unknown events. Keep full Inbox/message-domain persistence and media downloads outside this sprint unless a later explicit scope decision authorizes them. Preserve unknown payloads securely, deterministic semantic dedupe, bounded retries, fenced claims, audited permission-controlled replay and runtime RLS for every new organization-owned table.

Read subscription state first. A later actual subscription action requires explicit operator approval of the target and account effect; synchronization never auto-subscribes or unsubscribes. No send endpoint, Inbox reply, campaign, pricing guard, automation, template mutation or Sprint 3 work is authorized.

Completion requires Gate B approval, implemented/tested scope, sanitized fixtures, controlled live webhook evidence and green hosted CI on the Sprint 2 published commit. Sprint 1 run 34460443468 cannot satisfy that requirement.

Phase A documentation validation: python scripts/check.py PASS (117 local links, strict migration integrity, executable route coverage and source checks); git diff --check PASS. These checks do not constitute Sprint 2 integration, live Meta or hosted verification.
