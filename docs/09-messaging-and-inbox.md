# Messaging, inbox and content operations

Owner clarification (2026-09-15): **Sprints 0/1/2 COMPLETE; Gates A/B/C APPROVED; Gate D CLOSED; Sprint 3 AUTHORIZED. COMPANY META BILLING CURRENCY UNKNOWN; PAID-SEND AUTHORITY CLOSED.** Currency must not be inferred from timezone, business/phone/recipient country, available rate cards or locale. Only a currently reviewed, provably zero-cost policy can permit the single operator-triggered controlled TEXT live reply after all recipient/callback/security prerequisites. Paid/template live sends, outbound media, campaigns and Sprint 4 remain unauthorized.

Earlier dated status/review entries below are historical and superseded by this owner clarification where they describe Gate C or Sprint 3 authorization.

## Shared inbox behavior

One persistent thread per organization/phone/contact, with separate episodes for resolution and SLA measurement. Navigation always displays organization and sender number. An agent's list is filtered by team/phone scope before counts, snippets or search results are computed. Filters: team, assignee, state, number, priority, tags, unread, window state, oldest unanswered and search. Default sort prioritizes unresponded inbound, then activity, with stable ID tie-break.

Thread shows typed messages, provider timestamps, late-arrival markers, local pending intents, statuses, error/uncertainty reason and attachment state. Notes are visually and structurally separate. Contact panel shows identifiers, consent/suppression and provenance, tags, fields, campaign history and related threads that the member may access. Do not expose inaccessible team messages through contact timelines.

Reply workflow: persist browser-local draft in sessionStorage (lifecycle below), publish advisory compose presence, request server preflight, resolve required confirmation/approval, submit immutable intent, display Pending, then Accepted/Sent/Delivered/Read as observed. No optimistic delivered checkmark. Preserve a failed draft for editing without automatically resending it. Explicit new send after uncertainty needs reviewed evidence/risk handling.

Assignment checks agent is active, belongs to the target team and can access the number. Assignment-changing commands compare assignment_revision and increment it; unrelated incoming/outgoing messages do not. Every agent reply is an independent immutable intent with a distinct client idempotency key and expected_assignment_revision. The same agent can send text, an image and clarification immediately without reloading conversation state. If two currently authorized agents send concurrently, both intents remain valid unless an explicit assignment policy excludes one; show colleague typing/recent-send warnings, never silently discard a reply. A real assignment change blocks stale queued intent with ASSIGNMENT_CHANGED and preserves content for explicit review. Supervisor takeover/release changes handoff_epoch; queued bot actions with an old epoch fail HANDOFF_CONFLICT. Presence is advisory, with no distributed compose lock. Local submit order can be preserved by the composer, but external delivery order is not guaranteed.

Presence uses expiring Valkey keys, not payroll or customer availability tracking. Typing indicates colleagues composing. Sending/claiming is protected by DB dispatch authorization; a lost browser/Valkey lease does not authorize a second bot send. HUMAN mode rejects bot intents. WAITING_AGENT may emit at most one configured acknowledgement through the same gate. Resolution ends an episode and pauses bot output; reopening defaults to waiting for human unless a reviewed routing rule selects BOT.

Unread is a member's sequence watermark plus optional manual unread override. Sending Meta read receipts is an independent team policy; viewing one member's thread does not automatically mark every employee's inbox read. Mentions notify only current readers. Snooze wakes on inbound/due time. Resolve/reopen, assignment, priority, tags, notes edits/redaction and handoff are audited with resource revisions.

## Customer service window

Current September 14 first-party evidence permits non-template replies within the 24-hour customer service window and requires approved templates outside it. The current service-message guide also includes user calls as qualifying events; calling remains DEFERRED and is not an initial local activation source. [Gate C window and timestamp contract](27-gate-c-evidence.md#customer-service-window) records the bounded authenticated-text scope, original event/ingress basis and source limitations. Business replies and status events do not reset the window.

Internal model stores last eligible inbound ID/time, initial opened_at, expires_at, evidence source/confidence and separate free-entry observation. For each eligible inbound, `last_inbound_at = max(previous,event_time)` and expiry follows the currently verified window policy. Business replies, template sends, status receipts, notes and replay processing time do not reopen the service window. Unknown/future inbound subtypes that are not verified qualifying events do not activate free-form eligibility automatically.

ACTIVE: trusted expiry is more than 15 minutes ahead; EXPIRING_SOON: future expiry within the internal 15-minute warning threshold; EXPIRED: trusted expiry <= server now; UNKNOWN: insufficient valid evidence or rule unavailable. UI receives server_now/expires_at and displays remaining time with clock-drift correction. At dispatch, backend compares DB time with expiry using a small configurable safety margin (proposed 5 s); a valid preflight can expire while queued. Reject with TEMPLATE_REQUIRED; never auto-convert text into a template or silently charge.

Free-entry pricing is a separate policy decision with origin and qualification evidence. A possible 72-hour pricing concession does not extend the free-form reply window. If its exact conditions are unverified, treat cost as unknown or use a verified conservative paid upper bound with approval; never assume free from a referral object alone.

## Message model

| Internal type | Contents and handling |
| --- | --- |
| TEXT | Unicode text and allowed formatting; render escaped with safe links |
| IMAGE / VIDEO / AUDIO / DOCUMENT / STICKER | Asset reference, optional caption/filename, media state; audio can include voice-note metadata if verified |
| CONTACTS | Contact cards; explicitly separate from CRM contact creation/consent |
| LOCATION | Coordinates, optional label/address; no inferred continuous location |
| REACTION | Target ID, reactor, emoji/removal; retain unresolved context |
| INTERACTIVE | Verified subtype and schema-versioned payload: list/button/Flow/product variants |
| TEMPLATE | Local immutable template version, name/language/category snapshot, rendered parameters |
| ORDER / FLOW_RESPONSE | Structured inbound payload and correlation status; not proof of paid purchase or completed business action |
| UNKNOWN | Provider type, safe summary, protected raw-item reference; no automatic unsafe render/effect |

Provider IDs, context and status facts are separate fields, not hidden inside text. Unknown properties remain in protected raw storage. Sendable subtypes/MIME/size/component limits come from the capability registry, not guessed constants. M07–M14 and M22–M23 in the [matrix](02-meta-capability-matrix.md) distinguish official surfaces from exact constraints still awaiting verification.

## Media pipeline

Incoming media: store metadata/reference, enqueue high-priority fetch, retrieve current authenticated Meta URL, download with constrained network client, verify claimed hash/size where available, inspect MIME, scan malware, then persist to private S3-compatible storage and mark AVAILABLE. [Meta's media URL example](https://www.postman.com/meta/whatsapp-business-platform/request/fpj02x0/retrieve-media-url) describes a short expiry; retrieve a fresh URL on expiry, not repeated retries of a stale link. Upstream retention and re-fetch horizon remain NEEDS VERIFICATION.

Object key is tenant prefix plus opaque asset ID; no phone number/filename in keys. Stream downloads with verified maximum size, bounded decompression and timeout. User-supplied arbitrary fetch URLs are disabled initially. Meta URLs are accepted only from verified API responses, validated against allowed hosts and redirects, and never logged. Do not forward Graph credentials to an untrusted redirect.

Upload path: authorize intended workflow, bounded private upload grant, complete/checksum, scan, verified format validation, Meta upload when needed, then sendable asset. Resumable profile/template-header handles are different from message media IDs; adapters must not interchange them. No server-side media conversion in the request path. Unsupported formats have explicit rejection.

Deduplicate only within organization after scan, with reference counts and retention determined by all live uses. Private download URLs expire after a proposed 60 s and are minted after current resource authorization. Highly sensitive downloads may use authenticated streaming to permit immediate revocation; a previously issued signed URL has bounded residual access until expiry. Content-Disposition attachment for untrusted documents; no inline active SVG/HTML. Quarantine previews cannot execute content.

Outage behavior: message remains visible with attachment pending/unavailable. Expired upstream media can become permanently unavailable; record reason and last attempt instead of promising recovery. Erasure removes object, thumbnails, derived previews and cached URLs; retain a non-content tombstone.

## Existing Meta asset connection

Read-only validation first: inspect app identity/token scope, discover owned/shared WABAs, list existing numbers, review existing webhook subscriptions, bind explicit assets to organization. Do not register existing numbers again. Record verification status, timestamps and integration health per app/WABA/phone, including conflicting bindings. Credential rotation validates the replacement before controlled cutover; rollback references old encrypted credential only while still valid.

Profile updates show diff and only send verified editable fields. Display name, registration, billing, appeals or migrations requiring Meta UI open a clearly labeled external task. No fake success toast for unimplemented API actions. Sync runs are asynchronous, paginated, resumable and source-aware; incomplete pagination never archives missing assets.

## Template workspace

Read/sync first. View category, language, components, parameters, buttons, quality if observed, status/rejection reason, freshness, local version and associated campaigns. Compose validated local revisions with variable samples and deterministic previews. Preview and fallback interpolation are our UI features, not Meta APIs. A test send is a real guarded send to an explicit test recipient, never an exemption from consent or cost.

Create/edit/delete operations require current per-operation constraints: permitted source states, editable components, quotas, name/language immutability, deletion/name-reuse rules and approval behavior. **NEEDS VERIFICATION** until M15/M16 are confirmed against selected Graph version. Retain immutable observed versions; do not overwrite campaign-bound content. Meta can change status/category after approval; resync invalidates dependent authority where material. Our version number cannot force Meta to send old content: final dispatch requires current remote hash/category to match approved version. A mismatch pauses sending.

## Flow workspace

Sync Flow identity, status, JSON/data API version, validation errors and observed health where verified. Start with JSON editor plus validation and expiring official preview link; no visual-builder API assumption. Internal revision remains immutable after publication; create a new revision/asset as required by verified lifecycle. Publishing/deprecating is an audited external mutation. Sample inconsistencies are called out in M23; do not implement from copied commands alone.

Create a random opaque Flow instance token bound to org/contact/conversation/version/expiry. Completion response is untrusted input: validate schema, map token digest, enforce tenant and expected Flow, deduplicate source message, store response, emit flow.completed. Missing/invalid correlation is quarantined for review. Never accept recipient IDs or privileged business changes from user response fields. A Flow completion can start a reviewed automation, not implicitly approve spending.

Static Flows can ship after lifecycle and response fixtures pass. Dynamic data exchange, encrypted request handling, endpoint health protocol and synchronous deadlines are DEFERRED and require a separate verified contract. Do not process these through the asynchronous messaging webhook pipeline.

## MVP draft persistence

Choose sessionStorage for tab-local drafts, not IndexedDB or shared server drafts. Key by schema version + nonsecret login-instance ID + user ID + organization ID + conversation ID + Reply/Note mode. Persist bounded text/template inputs and updated_at; restore after refresh only once current account/membership/conversation access is verified. Use a proposed 24-hour TTL. Another tab/device has its own draft; no collaborative drafts or guaranteed crash/device recovery. [Web storage model](https://html.spec.whatwg.org/multipage/webstorage.html).

On organization switch, flush and retain the old namespaced draft, remove it from active memory, and load only the new organization's authorized namespace. On return, restore with fresh assignment/window/price checks. Account switch/logout/expired session purges application draft namespaces. Broadcast logout to active tabs and keep a non-content logout-generation marker; a suspended/restored tab checks generation and session before displaying any draft, then purges obsolete data. Lost access to a conversation also purges its draft.

Do not persist bearer/session/Meta tokens, pricing authorizations, signed media URLs, file bytes or browser File/object-URL handles. Retain only an optional attachment label and authorized internal asset reference, treated as untrusted; revalidate availability/access after refresh, otherwise ask to reattach. Remove only the submitted draft version when a durable intent is returned, leaving any newer text untouched. An ambiguous submit keeps its idempotency key/intent reference for status recovery, never creates a fresh send automatically. Draft contents never authorize sending.

sessionStorage is accessible to same-origin JavaScript; it is not encrypted secret storage. CSP/XSS controls, no third-party scripts on authenticated screens, bounded TTL and security review are required. If storage fails, clearly mark the draft unsaved and warn on navigation instead of falsely promising persistence.

## Gate C outbound specification, 2026-09-14

**Gate C CLOSED; Sprint 3 NOT STARTED.** The [bounded message-type contracts](27-gate-c-evidence.md#outbound-v260-contract) record required fields, media limits, window/template dependency and still-pending outbound account acceptance. No Sprint 2 webhook change is needed or authorized in this evidence run.

Store original authenticated provider timestamp and first durable ingress; the proposed conservative window basis is their minimum, preserved through replay. Do not accept future/malformed timestamps or use delivery receipts as user activity. The 15-minute warning and 5-second final safety margin are internal policy proposals, not Meta guarantees.

Message permission and price are independent. Current service/utility-reply concessions do not survive the October 1 policy change automatically: the new service allowance requires verified complete phone usage, and utility in an active window may be paid. Qualified 72-hour free entry still does not extend free-form permission. Use WABA effective dates and a covered delivery horizon.

For templates, Meta-observed category/status and exact components govern. Proposed final selected-template GET age is at most 30 seconds, with a permit at most 5 seconds and shorter window/price limits taking precedence. Bind ID/language/category/status/component hash; material drift invalidates authorization, unavailable/stale evidence blocks. A remote change after GET remains possible and must be visible in rejection/reconciliation. T4 is only a simple utility test candidate, pending safe content review and consent; no template is owner-selected for sending.

[Status and uncertain-send semantics](27-gate-c-evidence.md#status-money-and-uncertain-sends) preserve the existing domain states: HTTP acceptance is not delivery, metadata is not an invoice, and an ambiguous POST retains exposure without automatic retry. Future controlled tests need an explicitly approved reachable callback because Sprint 2 restored the App callback to empty.
