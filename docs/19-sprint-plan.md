# Delivery plan

Sequence work by correctness dependencies. Sprint duration is a planning placeholder (roughly two weeks for a small dedicated team), not a promised delivery date; staffing is UNKNOWN. Gate definitions are in [PLANS.md](../PLANS.md). Gate A is approved; Sprint 0 foundation work is authorized and in progress. Its evidence/status is in doc 23; Sprint 1 is not authorized.

| Sprint | Deliverables | Exit evidence / dependency |
| --- | --- | --- |
| 0 — foundation validation | Resolve assumptions and ownership; select/test SMTP relay and sender configuration; choose secure stack patches; pin Graph version; verify scope/subscriptions; capture sanitized fixtures; approve schema/API/ADR baseline; threat model; implement only subsequently authorized repository/CI/test foundation | Gate A accepted; no production side effects; reviewed version/fixture register, first tenant-isolation migration/test plan, CI gates, restore experiment plan |
| 1 — identity and tenant boundary | Auth, invitations, verification/reset/security SMTP mail with durable retry and one-use challenges, teams, permission/delegation, sessions/TOTP, audit, scoped DB access | Multi-org/team isolation, offboarding and mail failure/expiry/one-use tests; sensitive grants cannot escalate |
| 2 — ingestion and existing assets | Connection/discovery/sync, raw webhook persistence, child dedupe, durable jobs/outbox, status/media ingestion, basic health | Gates B/V1–V3; duplicate/reordered/unknown fixtures; crash recovery and media expiry tests |
| 3 — guarded inbox | Shared inbox, assignment/notes/unread/handoff, windows, typed messages, reviewed Rate Card Registry, pricing policies/confirmation/reservations, namespaced sessionStorage drafts and safe send | Gate C before any paid test; E2E incoming/reply/expiry/confirmation; budget race and uncertain-send tests |
| 4 — contacts and content | Consent center, import/export, audience versions, template lifecycle/preview, health/history | Revocation survives import/merge/replay; template status/category changes invalidate sends |
| 5 — campaigns | Snapshots, reusable approvals, schedule/recipient engine, frequency, throttle/pause/cancel, analytics | Immutable membership, final eligibility, approval/confirmation separation, campaign crash/pause tests |
| 6 — gateway and bounded automation | Scoped service keys/outgoing events/retries; keyword routing and handoff; static Flow completion after verification | No bypass via API/bot; signed replay tests, SSRF tests, handoff epoch concurrency, verified Flow fixtures |
| 7 — operational release | Load/query optimization, retention/erasure, notifications, restore/rollback/security review, employee usability pilot | All blocking doc 20 gates evidenced; accepted RPO/RTO and residual single-host risk |

## Proposed Sprint 0 backlog

S0-01 product owner: record actual employees/peaks/markets/currency, existing app/WABA topology, other senders and test recipients. S0-02 technical owner: accept/adjust ADRs and state/transaction protocol; turn workflow API into reviewed OpenAPI and migration plan. S0-03 Meta operator: current official Graph version/removal date, signature schema, request/payload limits and asset scopes; sanitize fixtures without committing tokens.

S0-04 finance/policy owner: import reviewed official rate artifacts and effective dates including announced changes; define ceilings, approvers, consent wording and retention. Missing values remain blocked, not filled with sample production settings. S0-05 engineering: after implementation authorization, minimal build/CI and real-Postgres test harness with two tenants; no hundreds of empty module files. S0-06 operations: verify VPS/storage/backup key access and run an isolated restore prototype against synthetic data. S0-07 design: validate inbox/campaign/diagnostics interaction flows with employees before implementing screens. S0-08 identity/operations: select SMTP relay/sender and secure configuration, verify a controlled mailbox, review draft storage controls and assign rate importer/reviewer/publisher permissions. Technical owner accepts or adjusts the proposed mixed-load contention thresholds before benchmarking.

Sprint 0 cannot close with unknown Graph/security contracts disguised as tasks for later implementation. It may finish foundational subwork while a specific adapter gate stays closed. Calling/Coexistence/Payments are not required for this exit.

## Deferred scope and triggers

Calling only after a concrete use case, country/account permission and pricing evidence; coexistence only if Business app use becomes necessary, with echo/history and external spend controls; Embedded Signup only for repeated multi-business onboarding; visual automation/Flow builders only after repeated complexity in real workflows; payments/commerce/groups only after current API/product review. XLSX, SSO/passkeys, advanced search and warehouse analytics wait for demonstrated need.

No deferred feature may quietly expand the approved send/pricing/privacy scope. Add capability evidence, schema/contract changes, threat model and acceptance tests before enabling it.

Sprint 0 delivery tracking: foundational code/schema/CI contracts now exist; local executed evidence is in [doc 23](23-sprint-0-evidence.md). Missing official Graph/signature evidence stays visible and Gate B stays CLOSED. Source files do not close Sprint 0; hosted CI and outstanding acceptance must be resolved first.
