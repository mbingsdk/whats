# Domain model and state machines

Organization is the operational and data boundary. Users are global identities; memberships and teams are organization-local. Meta apps and WABAs are linked many-to-many within an organization through explicit bindings; a phone belongs to one WABA. Contacts are organization-local identities, not phone strings. A scoped identifier belongs to one contact within its verified identity namespace.

A conversation is the persistent thread `(organization, phone, contact)`. It may contain multiple service episodes. Meta billing conversations and our conversation IDs are unrelated. Customer-service window and free-entry pricing window are separate models. A message records customer-visible content; a send intent records our request and attempts. Internal notes are separate records and never become messages.

A dynamic audience is a versioned query. A campaign audience snapshot freezes membership and personalization inputs. A campaign recipient is an operational row tracking the immutable target and its eligibility/send outcome. Approval signs a specific immutable revision, not a mutable campaign ID.

## State transitions

Only listed transitions are allowed. Resource-changing commands require the permissions in doc 14, matching organization/scope, expected_revision and current policy. Creating a new reply uses its own idempotency key and expected_assignment_revision; it does not require the whole conversation revision, which changes on ordinary messages. Transition, history and outbox commit atomically. Resource statuses are closed internal CHECK sets; unfamiliar Meta statuses are stored as raw text and mapped to UNKNOWN rather than rejected.

| Aggregate | States and valid transitions | Actor and guard |
| --- | --- | --- |
| Conversation | OPEN -> SNOOZED / RESOLVED; SNOOZED -> OPEN / RESOLVED; RESOLVED -> OPEN | Authorized agent; system wakes on due time or qualifying new inbound. Resolve ends an episode; reopening starts a new episode. Assignment does not change lifecycle. |
| Handoff | BOT -> WAITING_AGENT / HUMAN; WAITING_AGENT -> HUMAN; HUMAN -> BOT only explicit release; BOT/HUMAN/WAITING_AGENT -> NONE on resolution; NONE -> WAITING_AGENT on reopen by default | Automation may request agent; authorized human takes over/releases. Each transition increments handoff_epoch. Pending bot sends with prior epoch become ineligible. |
| Campaign | DRAFT -> PENDING_APPROVAL -> APPROVED -> SCHEDULED or QUEUED -> RUNNING -> DRAINING -> COMPLETED; SCHEDULED -> QUEUED; QUEUED/RUNNING/DRAINING -> PAUSED; PAUSED -> QUEUED or DRAINING; nonterminal -> CANCELLED; unrecoverable orchestration -> FAILED | Submitter, policy-approved reviewers, scheduler/dispatcher. Auto-approval is a recorded policy decision. PENDING_APPROVAL -> DRAFT on rejection/withdrawal/expiry; APPROVED/SCHEDULED -> DRAFT only before dispatch and invalidates authority. |
| Campaign recipient | PENDING -> EXCLUDED / READY / CANCELLED; READY -> RESERVED / EXCLUDED / CANCELLED; RESERVED -> DISPATCHING / READY / EXCLUDED / CANCELLED; DISPATCHING -> ACCEPTED / RETRY_WAIT / FAILED / UNCERTAIN; RETRY_WAIT -> READY / EXCLUDED / CANCELLED; UNCERTAIN -> ACCEPTED / FAILED after evidence | Worker owns transitions. Only proven non-acceptance permits retry. ACCEPTED is final for dispatch; delivery status evolves on its message. Expired RESERVED lease can return READY only if no attempt entered DISPATCHING. |
| Approval request | PENDING -> APPROVED / REJECTED / WITHDRAWN / EXPIRED; APPROVED -> INVALIDATED / CONSUMED | Eligible approvers meet quorum; no self-approval by default. Resource/policy mutation invalidates; authorized execution consumes. Per-run child authority is bounded, not unlimited reusable approval. |
| Send intent | PENDING -> BLOCKED / READY / CANCELLED; BLOCKED -> READY after new gate; READY -> RESERVED -> DISPATCHING; RESERVED -> READY / BLOCKED / CANCELLED only when undispatched; DISPATCHING -> ACCEPTED / RETRY_WAIT / FAILED / UNCERTAIN; RETRY_WAIT -> READY; UNCERTAIN -> ACCEPTED / FAILED with evidence | Origin permissions and final guards apply. Changes to immutable payload require a new intent; blocked intent cannot silently substitute a template. |
| Message delivery | PENDING_LOCAL -> ACCEPTED -> SENT -> DELIVERED -> READ, allowing skipped forward states; ACCEPTED/SENT -> FAILED when supported failure evidence; any -> UNKNOWN_CONFLICT when contradictory facts need review | Meta observations only for delivered/read; HTTP success means ACCEPTED, not delivered. Late lower status never regresses. Preserve contradictory history; review cannot fabricate READ. |
| Webhook envelope/job | RECEIVED -> PROCESSING -> PROCESSED / PARTIAL / UNKNOWN / RETRY_WAIT / DEAD_LETTER; RETRY_WAIT -> PROCESSING; DEAD_LETTER/UNKNOWN -> PROCESSING only audited replay | Signature and durable commit precede RECEIVED. Each child event is independently tracked; replay uses existing effect keys. PARTIAL remains until all known children terminal. |
| Automation run | QUEUED -> RUNNING -> WAITING / SUCCEEDED / RETRY_WAIT / FAILED / CANCELLED; WAITING/RETRY_WAIT -> RUNNING | Version immutable, loop/time limits, handoff epoch, per-action gate. Ambiguous HTTP effects suspend to WAITING for review. |
| Template synchronization | IDLE -> SYNCING -> CURRENT / STALE / FAILED; CURRENT/STALE/FAILED -> SYNCING | Synchronizer leases resource; observed Meta lifecycle kept separately. Failed sync never means template deletion. |
| Flow local revision | DRAFT -> VALIDATING -> VALIDATED / INVALID; VALIDATED -> PUBLISH_PENDING -> PUBLISHED / PUBLISH_FAILED | Publisher with flow.manage; actual Meta lifecycle operations require capability evidence. Editing published content creates another local revision/Meta asset as supported. |
| Member invitation | PENDING -> ACCEPTED / REVOKED / EXPIRED | Single-use digest, verified matching email, current inviter delegation policy. Resend creates a new token and invalidates old one. |
| Membership | ACTIVE -> DEACTIVATED; DEACTIVATED -> ACTIVE by authorized administrator | Access revision increments; reactivation does not revive sessions, invitations or API keys. Global user suspension revokes every organization. |
| Consent scope | UNKNOWN -> GRANTED / DENIED; GRANTED -> REVOKED / EXPIRED; DENIED/REVOKED/EXPIRED -> GRANTED only new evidence | Evidence-appending service and authorized user; import never implicitly grants. Suppression is a separate higher-priority deny. |
| Media | PENDING -> FETCHING -> QUARANTINED -> AVAILABLE / REJECTED; FETCHING -> RETRY_WAIT / UNAVAILABLE; AVAILABLE -> EXPIRED / DELETED | Scan before serving; unavailable upstream content cannot be invented/retried indefinitely. |

Campaign PAUSED stores previous phase and reason. Resume always reevaluates current authority; DRAINING has no new recipients. A cancelled campaign retains all accepted/uncertain recipient facts. COMPLETED means every recipient has reached a dispatch-terminal state, including UNCERTAIN, and `uncertain_count` remains visible. Delivery analytics can update afterward. FAILED records orchestration failure, not ordinary recipient failures.

## Invariants

1. Every business row, query, job, event, object and cache key has one organization boundary. Composite foreign keys prevent cross-organization references.
2. One active binding owns each external WABA/phone; changing organization ownership is an explicit migration and quarantines ambiguous webhooks.
3. One `(organization, phone, external_message_id)` message exists, including out-of-order status stubs. Child webhook effects are unique independently of HTTP envelope identity.
4. A send intent's recipient, sender, content/version and origin are immutable. One uncertain attempt cannot be automatically replaced by a new attempt.
5. A final dispatch permit cannot be issued while required approval, confirmation, eligibility or budget coverage is absent. These checks serialize with policy changes.
6. Suppression at the final serialized authorization point excludes the recipient. Opt-out cannot recall an already dispatched request; that boundary is audited and shown honestly.
7. Budgets count reserved/uncertain exposure until reliable release/reconciliation. Missing statuses are not grounds to refund a reservation.
8. No member can grant permissions or resource scope beyond current delegation authority. Offboarding leaves historical actor attribution intact.
9. Replay repairs state through existing dedupe/effect keys. New historical effects are disabled unless explicitly reviewed; historical customer events do not silently trigger new sends.
10. Window state derives from eligible inbound event time, not processing time or business replies. Delayed inbound must not extend a window into the future incorrectly.
11. Every bot intent captures handoff_epoch and must still match current epoch/BOT mode at DISPATCHING. Human replies have no per-epoch quota: distinct intents can send consecutively or concurrently if each passes current access/assignment and send guards. Assignment changes increment assignment_revision; message sends do not. Presence/typing supplies visible collision warnings, not exclusion or authorization.
12. A report never labels a computed rate multiplication as a Meta-confirmed invoice amount. Missing read or cost evidence remains UNKNOWN.

## Time, identity and ordering

Use UTC timestamptz for instants; store IANA business/budget timezone and original scheduled wall time separately. Reject ambiguous/nonexistent schedule times unless user selects explicit offset. Windows use trusted provider event timestamps with future-skew quarantine. Replay retains occurred_at and original received_at.

Each thread allocates a monotonically increasing local sequence while holding its row lock. Render chronology by provider time with deterministic ID tie-break and expose late-arrival markers; read watermarks use local sequence so a newly arrived older message remains discoverable. State facts retain observed_at and occurred_at. No global ordering across WABAs or phones is promised.

## Identity mail and rate publication states

Identity mail: QUEUED -> SENDING -> SMTP_ACCEPTED / RETRY_WAIT / FAILED; RETRY_WAIT -> SENDING. QUEUED/RETRY_WAIT -> EXPIRED / CANCELLED when deadline or source generation is invalid. SENDING -> EXPIRED/CANCELLED is allowed before transmission when the final source/deadline check fails; retry exhaustion becomes FAILED. Crash recovery of SENDING enters RETRY_WAIT, EXPIRED or CANCELLED after checking deadline/source; SMTP ambiguity may duplicate a one-use email but never renew its token. Terminal delivery never activates membership. Challenge validity requires current generation, unrevoked/unconsumed digest and expiry; consumption and security effects commit atomically. See docs 05/06/08/14.

Rate import: DRAFT -> VALIDATED -> DIFFED -> IN_REVIEW -> APPROVED -> PUBLISHED; IN_REVIEW -> CHANGES_REQUIRED -> DRAFT. Editing any nonpublished import resets it to DRAFT and invalidates reports/approval; invalid validation does not advance. Worker completion requires the captured import revision. PUBLISHED is terminal and immutable. Current publication pointer and policy_revision change atomically on publish; corrections require a new import/review/publication. See docs 05/06/08/12.
