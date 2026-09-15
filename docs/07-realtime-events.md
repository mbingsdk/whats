# Realtime and application event contract

Use one organization-scoped SSE stream per browser session, multiplexed across tabs where practical. REST handles durable commands and presence updates. SSE events are hints/projections of committed state; REST is the recovery source. No direct Meta payloads reach the frontend.

```json
{
  "event_id": "opaque-uuid",
  "organization_id": "opaque-org",
  "type": "message.created",
  "version": 1,
  "timestamp": "2026-09-08T12:00:00Z",
  "resource": {"type": "message", "id": "opaque-id", "revision": 1},
  "payload": {"conversation_id": "opaque-thread", "thread_sequence": 42},
  "correlation_id": "opaque-correlation"
}
```

Event IDs are stable across internal delivery retries and outbound webhook replay. Timestamp is occurrence time; publication sequence is separate. Resource revision supports rejection of stale updates. The envelope version is independent of Graph version and resource schema. Additive fields are allowed within v1; breaking changes require new schema version and subscriber migration.

| Event | Payload minimum | Audience and delivery |
| --- | --- | --- |
| conversation.created / conversation.updated | conversation_id,revision,changed_fields | Authorized inbox readers; summary only |
| conversation.assigned | conversation_id,team_id?,member_id?,revision | Authorized readers and assignee notification |
| conversation.resolved / conversation.reopened | conversation_id,episode_id,occurred_at | Authorized readers; external consumers if explicitly subscribed |
| conversation.handoff_changed | conversation_id,mode,handoff_epoch | Authorized readers; invalidates composer authority |
| conversation.window_changed | conversation_id,state,expires_at?,server_now | Authorized readers; clients refresh preflight before send |
| message.created | message_id,conversation_id,thread_sequence,direction | Scoped readers; fetch content through REST |
| message.status_changed | message_id,state,observed_at,revision | Scoped readers; includes uncertainty separately from delivery |
| message.received / message.accepted / message.sent / message.delivered / message.read / message.failed | message_id,conversation_id,occurred_at,source | Outbound developer events and automation. accepted = HTTP acceptance; sent requires Meta status evidence |
| typing.updated / presence.changed | member_id,conversation_id?,state,expires_at | Internal authorized colleagues only, ephemeral; never customer presence |
| note.created / note.mentioned | note_id,conversation_id | Internal readers with note permission only; not exported as message events |
| campaign.state_changed / campaign.progress | campaign_id,run_id,revision,counts,as_of | campaign.view scoped; rate-coalesce progress to ~1/s |
| campaign.started / campaign.completed | campaign_id,run_id,uncertain_count | Authorized subscribers; completion semantics from doc 04 |
| flow.completed | response_id,conversation_id,flow_version_id,correlation_state | Authorized Flow consumers; response fields require explicit scope |
| template.state_changed / meta.health_changed | resource_id,normalized/raw status,observed_at,source | Template/health readers |
| approval.requested / approval.decided | request_id,resource,decision? | Requester and current eligible approvers only |
| notification.created | notification_id,type,resource | Target member only |
| access.changed / stream.reset_required | access_revision / recovery_cursor | Session control; purge stale cache, reopen/refetch |

No arbitrary event names from Meta become public event types. Unknown provider events remain in operations diagnostics until normalized intentionally.

## Authorization and reconnection

GET `/events/organizations/{id}` uses the org session cookie and verifies active membership. Validate Origin and do not allow wildcard CORS. Resolve current team/phone/resource access per event. Subscription filters only narrow access. Membership/scope changes increment access revision and close affected streams; each event also checks current authority so missed invalidation cannot leak future content. A bounded TTL auth cache must fail closed if revision cannot be validated. No connection authenticated once forever.

Durable publication assigns per-organization `stream_sequence` under a short row lock at commit. `Last-Event-ID` is an opaque signed cursor containing org/sequence/schema, not the event UUID. Clients never choose another organization's stream or reuse a cursor across orgs. Hidden events advance the cursor through a content-free checkpoint, not a payload revealing hidden resources. Publication contains no client-specific permissions; filtering happens at delivery.

Proposed replay retention is 24 hours, with max 5,000 events per reconnect; larger gaps emit reset_required and fetch authoritative lists with a new cursor. Authorization is rechecked for historical events. Reconnect captures a watermark, fetches REST state at/after it, then applies later events using revisions. Rate publication emits a scoped pricing.publication.published event carrying publication ID/policy revision, coverage summary and next_review_at to pricing.registry.view holders; no rates leak to other users. It invalidates displayed estimates but never grants dispatch authority. TanStack Query keys include organization/access revision; switching org saves the current namespaced sessionStorage draft, clears its in-memory rendering/query cache and closes the prior stream. Restore that draft only after returning to the same account/org/conversation and reauthorizing access (doc 09); logout purges it.

Ephemeral presence heartbeat every 15 seconds, expiry 45 seconds; typing heartbeat at most every 3 seconds, expiry 8 seconds. These are internal tunable defaults. Coalesce heartbeats and discard during backlog. SSE heartbeat every 20 seconds; Nginx buffering disabled for this route. Bound connection count and per-client queue; slow clients disconnect and resync rather than consuming unlimited memory.

Outbound webhooks use the same versioned domain vocabulary but independently filtered, immutable payloads. They do not subscribe to the browser SSE stream. See doc 08 for signing, retries and scope checks.

## Internal-only normalized trigger events

contact.updated and contact.tag_changed carry contact ID, changed field names/tag ID and revision; consent.changed carries contact ID, category and state without evidence documents. campaign.reply_attributed carries campaign/recipient/message IDs plus attribution method/version. These events feed authorized automation and projections; they are not automatically exposed to external subscriptions. Agent-only notes and policy/role changes remain restricted. Every derived event retains origin/root ID for loop protection.

SSE hidden-event checkpoints contain only the signed cursor. They must not reveal skipped event count, resource IDs or event types. Permission revision changes require cache invalidation even if no resource event is delivered.

## Sprint 3 implemented minimum

Earlier envelope/cursor sections describe the broader target. GET /api/v1/inbox/events emits inbox.invalidate with scoped IDs, safe event types and refresh_required=true. It reauthorizes every four seconds and reconnects after 24 seconds. Clients refresh authoritative REST on connection/event and use a 15-second fallback. No ordered replay or lossless browser delivery is claimed. Scope loss purges rendering/drafts. [ADR 012](../ADR/012-sprint-3-inbox-runtime.md) records rationale and limits.
