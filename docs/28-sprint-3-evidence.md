# Sprint 3 evidence: guarded Inbox, message domain and outbound safety

Owner clarification (2026-09-15): **Sprints 0/1/2 COMPLETE; Gates A/B/C APPROVED; Gate D CLOSED; Sprint 3 AUTHORIZED. COMPANY META BILLING CURRENCY UNKNOWN; PAID-SEND AUTHORITY CLOSED.** Currency must not be inferred from timezone, business/phone/recipient country, available rate cards or locale. Only a currently reviewed, provably zero-cost policy can permit the single operator-triggered controlled TEXT live reply after all recipient/callback/security prerequisites. Paid/template live sends, outbound media, campaigns and Sprint 4 remain unauthorized.

## Baseline before implementation

Gate C research commit: 4bc49539745ead1f3b5e53470e7aa77671ab29cc. Hosted [run 34840457519](https://github.com/mbingsdk/whats/actions/runs/34840457519) succeeded, with all repository/PostgreSQL/browser/dependency steps executed. Owner-clarification baseline publication and its hosted verification are pending; no Sprint 3 product changes have started.

Implementation: OPEN. Acceptance: OPEN. Paid/template authority: CLOSED. Gate D: CLOSED.

## Authorized implementation sequence

1. Publish Gate C clarification/evidence and record hosted result before divergence.
2. Add coherent tenant-scoped domain/guard/Registry/budget migration and permissions; retain immutable historical SQL.
3. Add idempotent inbound/status materialization, scoped conversations/assignment/notes/read state, service-window engine and realtime.
4. Add text-only immutable intents, independently reviewed zero-cost policy, final authorization and uncertain-send worker. Potentially paid sends fail closed while currency is unverified; all template live sends stay closed.
5. Add operational Inbox, sessionStorage drafts, presence warnings and pricing administration.
6. Execute local integration/security/concurrency/browser checks; publish implementation with its own hosted CI.
7. Obtain explicit controlled TEST recipient and current approved callback before a single human-triggered free TEXT reply. Never send from automation or the agent itself.

## Live acceptance inputs

Controlled recipient: NOT DESIGNATED for Sprint 3; prior inbound evidence does not designate it.
Callback: prior temporary Sprint 2 callback restored to empty; new current-versus-proposed review required.
Billing currency: UNKNOWN. TEST paid maximum: UNSET. Paid/template sends: NOT AUTHORIZED.
No Sprint 3 live outbound request has been made.
