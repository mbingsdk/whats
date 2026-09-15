# ADR 012: Sprint 3 Inbox runtime

Status: implemented refinement under the 2026-09-15 owner-authorized minimum scope. Gates A/B/C APPROVED; Gate D CLOSED.

A separate Inbox worker materializes already-processed signed Sprint 2 evidence and independently records completion. Immutable text intents commit DISPATCHING before one network attempt. Expired DISPATCHING becomes UNCERTAIN, never automatic retry. Ordinary authorization uses shared policy/session barriers and narrow sender/recipient/conversation locks.

SSE sends scoped invalidation every four seconds with a 24-second connection lifetime and current resource authorization on each fetch. REST refresh on connect/reconnect/event is authoritative. UUID event IDs are not represented as a total commit-order cursor. Gaps and reordered commits cause the same scoped page refresh. This narrows ADR 003's planned replay-cursor delivery; ordered external events and cross-tab multiplexing are not implemented. Write deadlines disconnect slow clients.

The owner permits advisory presence without requiring Valkey. Sprint 3 stores 20-second-expiry PostgreSQL rows with ten-second composing heartbeats and worker cleanup. Presence never authorizes sends and can be lost harmlessly. This refines ADR 002's initial delivery scope; Valkey remains the future choice if measured load warrants it. No unmeasured performance claim justifies infrastructure.

Registry consumes only reviewed Gate C artifacts with original hashes. UNKNOWN billing currency blocks active account rate publication and paid sends. The owner-reviewed zero policy is independent, with hard review/cutoff dates. The inspected Direct Send TTL surface does not establish ordinary Service delivery horizon. Real text eligibility remains closed until that evidence resolves; tests inject an explicitly synthetic bound. Operator settings cannot override missing pricing proof.

References: [message behavior](../docs/09-messaging-and-inbox.md), [pricing](../docs/12-pricing-and-budget-guard.md), [evidence](../docs/28-sprint-3-evidence.md).
