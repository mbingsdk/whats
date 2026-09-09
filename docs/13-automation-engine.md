# Automation and human handoff

Build a bounded WhatsApp rules engine, not a general workflow platform. Versioned trigger/conditions/actions execute through existing domain services. No arbitrary JavaScript, user-defined SQL, unbounded loops or implicit AI decisions.

## Rule contract

| Layer | Supported initial design |
| --- | --- |
| Trigger | message.received, conversation.created/state_changed, flow.completed, contact.tag_changed, explicitly attributed campaign.reply |
| Conditions | Typed contact field/tag, team/phone, service-window state, message type, current consent/suppression, business-hours calendar, campaign attribution |
| Actions | Assign team/agent, add/remove tag, update allowlisted field, internal note, request human, pause bot; guarded text/template/static Flow send |
| Later gated actions | Approved HTTP endpoint and outgoing event; never arbitrary destination credentials or raw Meta forwarding |

Keywords use normalized Unicode and explicit language rules. Regex, if supported, uses a bounded non-backtracking engine and input length limit. Business-hours calendar is organization/timezone-versioned; do not use server locale. Flow completion requires validated correlation. All content-derived conditions treat user payloads as untrusted data.

Rule changes create immutable versions; publish references one validated version. Existing runs retain their version but current consent/access/budget and emergency disable always apply. Disabling a rule cancels unstarted effects; it cannot recall external requests already dispatched. Simulation on retained sanitized events is PROJECT_ONLY with no sends/HTTP/notifications.

## Execution and loop prevention

Run key `(organization, automation_version, source_event_id)`; action key `(run_id, action_index)` with stable effect digest. Persist action intent/result before advancing cursor; transactionally execute local tag/assignment actions with source/root event IDs. An action-produced event carries origin AUTOMATION and root_event_id. Default exclude same-root retriggers of the same rule. Initial caps: depth 5, 20 actions per run, 60-second active execution, 24-hour total wait TTL; these are internal safety limits and configurable downwards.

Idempotency prevents duplicated delivery of one event from repeating effects. Loop limits prevent legitimately new but cyclic events creating infinite work. Both are required. Publish-time validation rejects obvious recursive rules and unknown actions; runtime tracks actual root/depth. One rule failure does not roll back an already accepted external message; result is partial/failed with visible step history.

Transient local failures retry with jitter under job policy. Consent/permission/template/window failures are BLOCKED/terminal with a reason, not repeated attempts hoping policy changes. Outbound message ambiguity follows messaging UNCERTAIN; automation waits for review and cannot automatically issue a replacement. For HTTP actions, only retry ambiguous outcomes if the destination has a verified idempotency contract keyed by effect ID.

## Handoff arbitration

Conversation lifecycle OPEN/SNOOZED/RESOLVED is separate from handoff mode BOT/WAITING_AGENT/HUMAN/NONE. Bot run captures handoff_epoch; dispatcher rereads it with current BOT mode under the conversation lock before external authorization. Human takeover increments that epoch under the conversation lock and a shared organization policy barrier, invalidating pending bot permits. Human messages neither consume nor advance an epoch. If a bot request was already DISPATCHING, UI warns one reply may still arrive; do not claim to revoke network traffic.

WAITING_AGENT exposes queue, requested time and reason. Default action is assignment and internal notification; optional single acknowledgement passes all send guards. HUMAN permits agent replies and rejects bot ones. Release to BOT is explicit with reason and current authorization, not triggered by a missing presence heartbeat. Resolution ends episode and sets NONE. New inbound reopens to WAITING_AGENT unless reviewed routing says otherwise.

Automation cannot grant consent, lift suppression, approve its own campaign, raise budgets, change roles or retrieve secrets. It may capture candidate consent evidence for review through an explicitly approved collection workflow, not treat any Flow answer as permission automatically.

## Operational visibility

Run view displays immutable rule version, trigger, condition outcomes, action status, input reference (redacted), effect IDs, attempts, elapsed time and stop reason. Search by contact/conversation/event within access scope. Alert on run lag, failure rate, loop stops, handoff collisions and uncertain effects. Do not alert every agent about routine nonmatching conditions.

Initial rollout: keyword routing and explicit human escalation without automatic sending; then one guarded acknowledgement/template use case; then static Flow completion action. Dynamic Flow exchange and arbitrary workflow composition remain DEFERRED.
