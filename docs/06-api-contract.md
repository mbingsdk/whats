# REST API contract

Base `/api/v1`. Organization resources use `/organizations/{organization_id}` (abbreviated `O` below). Never trust a tenant ID from a request without principal membership/key validation. All identifiers are opaque strings. This is the target internal API, not Graph API pass-through. Sprint 0 health/readiness and the Sprint 1 identity/access section are implemented. Other product sections remain future contracts. Unverified Meta write operations below are explicitly capability-gated; their external adapter payload is not finalized.

## Common protocol

Cookie sessions require CSRF token and matching Origin for mutations. Machine consumers use bearer API keys with explicit organization/phone scopes. Sensitive routes require recent reauthentication/MFA. `Idempotency-Key` is required for sends, confirmations, approval decisions, campaign commands, imports and external mutations. Store request hash and original resource/result. Same key/different canonical payload returns 409 IDEMPOTENCY_CONFLICT. Response replay rechecks present access before disclosure.

Updates/resource-changing commands require `If-Match: "revision"` for mutable aggregates. Creating independent message intents is exempt from whole-conversation If-Match; it checks expected_assignment_revision for an agent, and captured handoff_epoch for a bot. Missing revision returns 428 PRECONDITION_REQUIRED; stale revision 409 REVISION_CONFLICT. New objects return 201; asynchronous work returns 202 with operation/job ID and status URL; a send 202 is never delivery success. GET uses 200; deletion of an internal revocable credential may use 204.

List response `{items,next_cursor,has_more}`; opaque cursor binds filters, sort and authorization scope, max page size 100. Stable keyset ordering includes ID. Explicit `as_of`/watermark for reports and audience previews. Sprint 1 cursor tampering returns 422 VALIDATION_FAILED. No unbounded exports through list APIs.

Instants are RFC3339 UTC. Money is `{amount:"decimal",currency:"ISO"}` or null with an explicit unknown reason. All enums are internal schema values; unknown external values appear as `{normalized:"UNKNOWN",raw:"..."}`. Read/detail responses include `id,organization_id,revision,created_at,updated_at` where applicable. `server_now` accompanies time-sensitive views.

```json
{
  "error": {
    "code": "PRICING_CONFIRMATION_REQUIRED",
    "message": "Review the estimate before sending.",
    "details": {"estimate_id": "opaque-id", "retryable": false},
    "field_errors": [{"field": "pricing_authorization", "code": "REQUIRED"}],
    "request_id": "opaque-request-id"
  }
}
```

Common errors for every authenticated route: 401 UNAUTHENTICATED, 403 PERMISSION_DENIED or REAUTHENTICATION_REQUIRED, 404 RESOURCE_NOT_FOUND (also hidden/cross-tenant resources), 409 REVISION_CONFLICT, 422 VALIDATION_FAILED, 429 RATE_LIMITED with Retry-After, 503 DEPENDENCY_UNAVAILABLE. Never return raw Meta responses/tokens in the error envelope. Additional domain errors below are stable contracts.

## Identity and access workflows

The executable [OpenAPI contract](../contracts/openapi.yaml) is authoritative for all 41 identity/access operations. Paths below use /api/v1. There is no public signup/organization-create endpoint.

| Method | Path | Operation |
| --- | --- | --- |
| GET | /auth/csrf | CSRF token |
| POST | /auth/login | public.login |
| POST | /auth/logout | logout |
| GET | /auth/session | session |
| POST | /auth/mfa/verify | public.mfa |
| POST | /auth/reauthenticate | reauth |
| POST | /auth/password-reset/request | public.reset.request |
| POST | /auth/password-reset/complete | public.reset.complete |
| POST | /auth/email-verification/request | public.verify.request |
| POST | /auth/email-verification/complete | public.verify.complete |
| POST | /auth/organization-session | organization.select |
| POST | /auth/mfa/enrollment | mfa.enroll |
| POST | /auth/mfa/enrollment/confirm | mfa.confirm |
| POST | /auth/mfa/disable | mfa.disable |
| POST | /auth/mfa/recovery-codes/regenerate | mfa.recovery |
| GET | /security/sessions | sessions |
| POST | /security/sessions/{id}/revoke | session.revoke |
| POST | /security/sessions/revoke-others | sessions.revoke |
| POST | /invitations/accept | public.invitation.accept |
| GET | /organization | org.get |
| GET | /organizations/{org}/members | org.members.list |
| GET | /organizations/{org}/members/{id} | org.member.get |
| POST | /organizations/{org}/members/{id}/deactivate | org.member.deactivate |
| POST | /organizations/{org}/members/{id}/reactivate | org.member.reactivate |
| PUT | /organizations/{org}/members/{id}/role-grants | org.member.roles |
| GET | /organizations/{org}/invitations | org.invitations.list |
| POST | /organizations/{org}/invitations | org.invite |
| POST | /organizations/{org}/invitations/{id}/revoke | org.invite.revoke |
| POST | /organizations/{org}/invitations/{id}/resend | org.invite.resend |
| GET | /organizations/{org}/teams | org.teams.list |
| POST | /organizations/{org}/teams | org.team.create |
| PATCH | /organizations/{org}/teams/{id} | org.team.update |
| POST | /organizations/{org}/teams/{id}/archive | org.team.archive |
| GET | /organizations/{org}/teams/{id}/members | org.team.members.list |
| POST | /organizations/{org}/teams/{id}/members | org.team.member.add |
| DELETE | /organizations/{org}/teams/{id}/members/{member} | org.team.member.remove |
| PUT | /organizations/{org}/teams/{id}/role-grants | org.team.roles |
| GET | /organizations/{org}/roles | org.roles.list |
| POST | /organizations/{org}/roles | org.role.create |
| PATCH | /organizations/{org}/roles/{id} | org.role.update |
| GET | /organizations/{org}/audit | org.audit.list |

New teams/roles return 201. Invite and generic mail requests return 202; other commands return 200. Organization aggregate changes require If-Match; missing is 428 and stale is 409. Invalid/expired action proofs return 400 TOKEN_INVALID_OR_EXPIRED; Owner invariant is 409 LAST_ACTIVE_OWNER; permission/delegation failure is 403 PERMISSION_DENIED. Every protected route rechecks session and live access; an invalidated organization session returns 401.

Session tokens are opaque cookies. Login/MFA returns a CSRF token and authenticated flag or a temporary challenge. GET /auth/session returns active memberships; GET /organization returns metadata, ORG permissions and scoped team permissions. Organization switching preserves absolute expiry and the sensitive reauthentication deadline. Active session management is self-only.

Role grants accept mixed grants[] (role_id, scope ORG/TEAM/SELF, nullable team_id) or role_ids with one common scope. Replacing direct grants does not replace team inheritance. Team lists filter current permission scope and archived teams confer no scoped authority. Owner/Admin/Agent are presets, never authorization branches. Invitation status exposes only sanitized delivery_state; global identity mail is not tenant-visible.

Existing-identity invitation acceptance requires a matching authenticated account and CSRF proof; password replacement by invitation is rejected. No nonsecret draft login-instance protocol, machine keys, delegated cross-user session endpoint or arbitrary email-change endpoint is implemented in Sprint 1. Those future surfaces require their own reviewed implementation.

## Meta, contacts and inbox

| Method / route | Permission | Request -> response | Additional failures |
| --- | --- | --- | --- |
| POST O/meta/connections | meta.credentials.manage + reauth | app_id,access_token,app_secret,version -> validation operation (no echo) | META_ACCESS_DENIED, GRAPH_VERSION_UNVERIFIED |
| GET O/meta/connections/{id}/discovery | meta.view | -> accessible WABAs/phones, ownership, observed_at | CONNECTION_INVALID |
| POST O/meta/connections/{id}/bindings | meta.assets.manage | selected existing WABA IDs -> binding operation | ASSET_ALREADY_BOUND, ASSET_ACCESS_UNVERIFIED |
| POST O/meta/connections/{id}/sync | meta.assets.manage | resource_kinds -> sync_run_id | META_RATE_LIMITED |
| POST O/wabas/{id}/subscription-reconcile | meta.webhooks.manage + reauth | proposed_diff,expected_existing_hash -> operation | SUBSCRIPTION_CONFLICT, CAPABILITY_UNVERIFIED |
| GET O/wabas; GET O/phone-numbers | meta.view or authorized inbox sender read | scope filters -> observations, capabilities, stale flags | Common |
| PATCH O/phone-numbers/{id}/business-profile | meta.phone.manage | supported editable fields -> operation | CAPABILITY_UNVERIFIED, META_VALIDATION_FAILED |
| POST O/phone-numbers/{id}/block-actions | meta.phone.block | action BLOCK/UNBLOCK,identifier,reason -> operation | CAPABILITY_UNVERIFIED; does not alter consent |
| GET O/meta/health | meta.health.view | asset filter -> source-labeled checks, observed_at | Common |
| GET O/contacts; GET O/contacts/{id} | contacts.view scoped | query/filters -> contacts/timeline/consent summary | Common |
| POST O/contacts; PATCH O/contacts/{id} | contacts.edit | identifiers,fields,tags -> contact | IDENTITY_CONFLICT, INVALID_IDENTIFIER |
| POST O/contacts/{id}/notes | contacts.edit | text -> note | VALIDATION_FAILED |
| POST O/contacts/{id}/consent-events | consent.manage | category,action,source,evidence,policy_version,occurred_at -> event/state | CONSENT_EVIDENCE_REQUIRED |
| POST O/suppressions | consent.suppress | identifier,scope,reason -> suppression | Common |
| POST O/suppressions/{id}/lift | consent.restore + reauth | new evidence,reason -> decision | CONSENT_EVIDENCE_REQUIRED, APPROVAL_REQUIRED |
| POST O/contact-imports; POST O/contact-imports/{id}/commit | contacts.import | asset,mapping,provenance -> preview job; preview_hash -> import job | IMPORT_PREVIEW_CHANGED, CONSENT_EVIDENCE_REQUIRED |
| POST O/contact-exports | contacts.export | filters,fields,purpose -> scoped expiring export | EXPORT_SCOPE_DENIED |
| GET O/conversations; GET O/conversations/{id} | inbox.view | phone/team/state/cursor -> thread/window/assignment/revision | Common |
| GET O/conversations/{id}/messages | inbox.view | before_sequence,limit -> typed messages/media refs | Common |
| POST O/conversations/{id}/assignment | inbox.assign | team_id,member_id,reason; If-Match is assignment_revision on this route -> assignment revision | MEMBER_NOT_IN_TEAM, ASSIGNMENT_CONFLICT |
| POST O/conversations/{id}/state | inbox.resolve or inbox.snooze | action RESOLVE/REOPEN/SNOOZE,wake_at? -> state | INVALID_STATE_TRANSITION |
| POST O/conversations/{id}/handoff | inbox.handoff | TAKEOVER/RELEASE/REQUEST_AGENT,reason -> handoff_epoch | HANDOFF_CONFLICT |
| POST O/conversations/{id}/notes | inbox.note | text,mentioned_member_ids -> note | MENTION_SCOPE_DENIED |
| PUT O/conversations/{id}/read-marker | inbox.view | seen_through_sequence,mark_unread? -> marker | INVALID_SEQUENCE |
| PUT O/conversations/{id}/typing; PUT O/presence | inbox.reply / active member | typing bool; status -> expiry | RATE_LIMITED; internal employee state only |
| POST O/media/uploads; POST O/media/uploads/{id}/complete | media.upload with workflow scope | filename,size,MIME -> private upload instructions; checksum -> scan job | MEDIA_TOO_LARGE, MIME_NOT_ALLOWED |
| POST O/media/{id}/access | Underlying conversation/contact access | purpose -> expiring scoped URL | MEDIA_QUARANTINED, MEDIA_UNAVAILABLE |

## Send, pricing and campaign workflows

| Method / route | Permission | Request -> response | Additional failures |
| --- | --- | --- | --- |
| POST O/pricing/preflights | Origin's send permission | phone,contact/identifier,conversation?,content,origin -> eligibility,window,estimate,upper_bound,policy_version,expires_at,required_actions | TEMPLATE_REQUIRED, CONSENT_REQUIRED, PRICING_UNKNOWN |
| POST O/pricing/authorizations | send permission + pricing.confirm | estimate_id,accepted_scope_hash,max_amount,currency -> single-use token,expires_at | ESTIMATE_EXPIRED, COST_CEILING_TOO_LOW |
| POST O/messages | inbox.reply (agent) or messages.send (API); scoped origin authority for worker | immutable send specification,expected_assignment_revision (agent),handoff_epoch (internal bot),estimate_id,authorization?,context_message_id? -> intent_id,status,status_url | SERVICE_WINDOW_EXPIRED, TEMPLATE_REQUIRED, PRICING_CONFIRMATION_REQUIRED, BUDGET_EXCEEDED, CONSENT_REQUIRED, FREQUENCY_EXCEEDED, HANDOFF_CONFLICT, ASSIGNMENT_CHANGED |
| GET O/send-intents/{id} | Origin/resource view | -> status,attempt summary,message_id?,uncertainty/rejection reason | Common |
| POST O/send-intents/{id}/cancel | Origin send authority | reason -> state | ALREADY_DISPATCHED |
| POST O/send-intents/{id}/uncertainty-resolution | messaging.reconcile + reauth | evidence_ref,proven outcome,reason -> resolution | INSUFFICIENT_EVIDENCE; no automatic resend |
| POST O/messages/{id}/reactions | inbox.reply | emoji or remove -> guarded reaction intent | CAPABILITY_UNVERIFIED, CONTEXT_INVALID |
| GET O/budgets; POST O/budgets; PATCH O/budgets/{id} | billing.view / billing.budgets.manage | scope,currency,period,thresholds -> budget revision/exposure | INVALID_CURRENCY, APPROVAL_REQUIRED |
| POST O/billing/reconciliations | billing.reconcile | official evidence asset,source,period -> import review operation | DUPLICATE_SOURCE, TOTAL_MISMATCH |
| GET O/audiences; POST O/audiences; POST O/audiences/{id}/versions | audience.view / audience.manage | typed rule AST -> version/hash | INVALID_RULE |
| POST O/audiences/{id}/preview | audience.view + contacts.view | version_id -> counts,exclusion breakdown,as_of | QUERY_TOO_COMPLEX |
| POST O/campaigns; PATCH O/campaigns/{id}/draft | campaign.create | sender,template,variables,audience,schedule -> draft revision | TEMPLATE_NOT_SENDABLE, INVALID_STATE_TRANSITION |
| POST O/campaigns/{id}/preflight | campaign.submit | revision -> snapshot/estimate job, blockers,excluded counts | PRICING_UNKNOWN, CAPABILITY_UNVERIFIED |
| POST O/campaigns/{id}/test-send | campaign.test | explicit allowlisted recipient,revision -> guarded intent | Same send failures; test sends may cost money |
| POST O/campaigns/{id}/submit | campaign.submit | sealed snapshot,estimate,revision -> approval_request/state | SNAPSHOT_NOT_SEALED, CAMPAIGN_PREFLIGHT_FAILED |
| POST O/approvals/{id}/decisions | Action-specific approve permission + policy eligibility | APPROVE/REJECT,reason,request_hash -> decision/state | SELF_APPROVAL_FORBIDDEN, APPROVAL_EXPIRED, RESOURCE_CHANGED |
| POST O/campaigns/{id}/schedule; /start | campaign.schedule / campaign.send | approved_revision,confirmation_authority?,scheduled_at? -> run/state | CAMPAIGN_APPROVAL_REQUIRED, PRICING_CONFIRMATION_REQUIRED, SCHEDULE_OUTSIDE_AUTHORITY |
| POST O/campaigns/{id}/pause; /resume; /cancel | campaign.pause / campaign.send / campaign.cancel | reason,run_revision -> state,in_flight_count | INVALID_STATE_TRANSITION, APPROVAL_INVALIDATED |
| GET O/campaigns/{id}; GET O/campaigns/{id}/recipients | campaign.view; PII requires contacts.view | -> progress,dispatch/delivery counts; filtered recipient page | Common |

`content` is a discriminated union: text, image/video/audio/document/sticker asset, contacts cards, location, interactive subtype, template version plus variables, or Flow instance. Only validated enabled outbound variants are accepted; UNKNOWN is receive-only. Replies use internal target IDs resolved to same-thread Meta IDs. API clients cannot submit a raw Graph object or choose a fake principal origin.

Preflight response includes `{send_allowed,free_form_allowed,template_required,window:{state,expires_at,server_now},pricing:{state,category,market,policy_version,estimate,upper_bound,expires_at},budget:{available,reserved,policy_revision},required_actions,blockers,scope_hash}`. `send_allowed` means the current dry run passes; it does not reserve funds or promise a later dispatch.

## Content, automation, developer and operational routes

| Method / route | Permission | Request -> response | Additional failures |
| --- | --- | --- | --- |
| GET O/templates; GET O/templates/{id} | template.view | -> versions,status,history,usage,observed_at | Common |
| POST O/templates; POST O/templates/{id}/revisions; POST O/templates/{id}/deletion-requests | template.create / template.edit / template.delete | supported definition or reason -> validation/Meta mutation operation | CAPABILITY_UNVERIFIED, META_TEMPLATE_REJECTED, TEMPLATE_MUTATION_RESTRICTED |
| POST O/templates/{id}/test-send | template.test + origin send permission | recipient,variables -> guarded intent | Same send failures |
| GET O/flows; POST O/flows/{id}/versions | flow.view / flow.manage | definition/schema -> validation result | FLOW_SCHEMA_UNSUPPORTED |
| POST O/flows; POST O/flows/{id}/sync; /publish; /preview | flow.manage (preview: flow.view) | name/categories; version/revision -> operation or expiring preview | CAPABILITY_UNVERIFIED, FLOW_INVALID, FLOW_IMMUTABLE |
| GET O/flows/{id}/responses | flow.responses.view + contact scope | -> filtered responses | Common |
| POST O/automations; POST O/automations/{id}/versions | automation.manage | allowlisted trigger/conditions/actions -> draft version | LOOP_RISK, ACTION_NOT_ALLOWED |
| POST O/automations/{id}/publish; /disable | automation.publish | version/hash -> active state | APPROVAL_REQUIRED |
| GET O/automation-runs; GET O/automation-runs/{id} | automation.view | -> steps/effects/errors | Common |
| POST O/api-keys; POST O/api-keys/{id}/rotate; DELETE O/api-keys/{id} | developer.keys.manage + reauth | scopes,phones,expiry,sponsor -> secret once or revocation | GRANT_EXCEEDS_AUTHORITY |
| POST O/webhook-endpoints; PATCH O/webhook-endpoints/{id} | developer.webhooks.manage | HTTPS URL,types,scope -> endpoint/config revision | DESTINATION_NOT_ALLOWED |
| GET O/webhook-deliveries; POST O/webhook-deliveries/{id}/replay | developer.events.view / developer.webhooks.replay | delivery filter; reason -> retry using same event ID | ENDPOINT_REVOKED, PAYLOAD_EXPIRED |
| GET O/webhook-events; GET O/webhook-events/{id} | operations.webhooks.view (raw needs operations.payloads.view) | -> redacted state/items/attempts | PAYLOAD_EXPIRED |
| POST O/webhook-events/{id}/replay | operations.webhooks.replay + reauth | parser_version,mode PROJECT_ONLY,reason -> replay job | REPLAY_NOT_ALLOWED, PAYLOAD_EXPIRED |
| GET O/notifications; PATCH O/notifications/{id} | Self | -> scoped notifications; read/dismiss -> state | Common |
| GET O/approvals | approval.view within eligible scope | -> pending/history | Common |
| GET O/audit | audit.view | actor/resource/time/cursor -> redacted records | Common |
| GET O/analytics/{messages,inbox,campaigns,templates,system,costs} | analytics.view + resource/PII/cost permissions | interval,timezone,dimensions -> source-labeled buckets,watermark,coverage | UNSUPPORTED_DIMENSION |
| GET O/operations/{id} | Original workflow permission | -> async status,result refs,error | Common |

Public Meta callback: GET `/webhooks/meta/{opaque_app_route}` handles challenge verification; POST handles bounded raw bytes/signature then durable persistence. It never accepts organization ID as authority. Public tracked redirects are optional opaque tokens; destinations are fixed during campaign review. SSE contract is in doc 07. Live Flow data-exchange routes are DEFERRED until separately verified; they must not reuse the asynchronous webhook ACK contract.

## Domain error mapping

422 SERVICE_WINDOW_EXPIRED/TEMPLATE_REQUIRED/CONSENT_REQUIRED/TEMPLATE_NOT_SENDABLE; 409 PRICING_CONFIRMATION_REQUIRED/APPROVAL_REQUIRED/CAMPAIGN_APPROVAL_REQUIRED/BUDGET_EXCEEDED/FREQUENCY_EXCEEDED; 409 ESTIMATE_EXPIRED/APPROVAL_INVALIDATED/HANDOFF_CONFLICT/ASSIGNMENT_CHANGED; 503 PRICING_UNKNOWN/PRICING_REVIEW_OVERDUE/CAPABILITY_UNVERIFIED. Upstream rate limits become META_RATE_LIMITED with verified retryability and retry_at; an accepted async operation instead records that status in its intent. META_TEMPLATE_REJECTED carries a redacted reason if provided. No HTTP response should suggest a retry is safe when acceptance is uncertain.

OpenAPI includes implemented health/readiness and Sprint 1 identity/access schemas. Source checks compare its method/path set to the runtime route registry. Later product schemas/handlers require their actual sprint authorization.

For reply concurrency, HANDOFF_CONFLICT means a bot's captured epoch/mode became invalid (or an explicitly requested human handoff failed), never merely that another human replied. ASSIGNMENT_CHANGED means the relevant assignment/access-routing revision changed; an already-created queued intent becomes BLOCKED with that reason and retains content. Colleague composing/recent-send information is advisory metadata, not a 409 or a prerequisite to send. Two legitimate payloads use two idempotency keys; one retried payload reuses its original key. An ASSIGNMENT_CHANGED human intent remains blocked until explicitly cancelled/replaced after review using the current assignment revision and fresh authorization; a replacement has a new key and audit link. Never refresh a queued bot's captured handoff_epoch. No draft, warning acknowledgement or assignment revision itself authorizes spending.

## Identity mail completion and retry contract

POST /auth/email-verification/request returns generic 202; POST /auth/email-verification/complete takes the one-use token in the body and returns verified status or TOKEN_INVALID_OR_EXPIRED (400). Request and password-reset routes are rate-limited and do not reveal whether an account exists or SMTP failed. Completion never occurs on GET.

POST O/invitations/{id}/resend requires members.invite, current delegation, If-Match and Idempotency-Key; returns 202 with invitation/delivery IDs, state and expiry, never a raw token. It rotates the challenge generation and invalidates old pending deliveries; INVALID_STATE_TRANSITION (409) applies after acceptance/revocation. GET O/invitations/{id} exposes sanitized mail status/error only within scope. Challenge deadlines and retry semantics are in doc 14; arbitrary requested invitation expiry beyond 72 hours is rejected (422 INVALID_EXPIRY). Global reset/verification mail has no tenant-admin listing endpoint.

## Rate Card Registry REST contract

All routes are organization-scoped; mutation commands use Idempotency-Key and existing-import commands also require If-Match. No endpoint updates published rows. Amounts are decimal strings with currency; evidence downloads require pricing.registry.view and short-lived private access.

| Route | Permission | Result and guarded behavior |
| --- | --- | --- |
| GET O/pricing/imports; GET O/pricing/imports/{id}; GET O/pricing/publications; GET O/pricing/publications/{id} | pricing.registry.view | Paginated state, source/review metadata, reports and immutable rows within org; no foreign artifacts. |
| POST O/pricing/imports; PATCH O/pricing/imports/{id} | pricing.registry.import | Create/edit mode, source artifact/reference, parser/import version, dimensions/effective dates/rows, next_review_at; returns DRAFT revision. Edits reset prior reports/approval. |
| POST O/pricing/imports/{id}/validate; POST O/pricing/imports/{id}/diff | pricing.registry.import | 202 operation ID, revision-bound durable job. Validate DRAFT -> VALIDATED only on success; diff VALIDATED -> DIFFED against current base; failures attach sanitized reports without advancing. |
| POST O/pricing/imports/{id}/submit | pricing.registry.import | DIFFED -> IN_REVIEW, freezes review hashes. |
| POST O/pricing/imports/{id}/review | pricing.registry.review | APPROVE -> APPROVED or REQUEST_CHANGES -> CHANGES_REQUIRED; explicit rationale, distinct reviewer, exact hashes. |
| POST O/pricing/imports/{id}/publish | pricing.registry.publish + recent MFA | APPROVED -> PUBLISHED, immutable publication ID and new head; transaction rejects stale base/evidence/review. |

Errors: 409 STALE_RATE_BASE, PUBLICATION_IMMUTABLE, INVALID_STATE_TRANSITION or REVISION_CONFLICT; 422 RATE_COVERAGE_GAP, RATE_DIMENSION_CONFLICT, RATE_SOURCE_UNVERIFIED; 403 SELF_REVIEW_FORBIDDEN or PERMISSION_DENIED. Async validation errors live in the operation report with the same domain codes. A stale base requires reset to DRAFT, validation/diff against the new head and fresh review. No operation result grants permission to send; dispatch also rejects 503 PRICING_REVIEW_OVERDUE or 503 PRICING_UNKNOWN for absent/unverified coverage when applicable.

The implemented OpenAPI foundation is [contracts/openapi.yaml](../contracts/openapi.yaml), checked by Redocly in CI. Health endpoints are outside /api/v1 deliberately; future business endpoints retain that namespace. Liveness ignores external services; readiness checks PostgreSQL/schema/runtime role. Supported read methods are GET/HEAD; unknown paths and method/body failures use the safe error envelope. The declared auth schemes are future contracts, not a working login bypass.

## Gate B and conditional Sprint 2 boundary

Phase A added no executable endpoint or OpenAPI operation. Meta reads, synchronization, webhook GET/POST and subscription actions above remain target contracts until [Gate B](25-gate-b-evidence.md) passes. POST authenticity and GET challenge must be verified independently. No outbound send route is authorized for Sprint 2; an eventual subscription action requires explicit operator approval.

## Sprint 2 runtime endpoints

The executable [OpenAPI contract](../contracts/openapi.yaml) now includes:

- GET/POST /api/v1/meta/connection; POST binds configured assets internally, never mutates Meta.
- POST /api/v1/meta/sync; GET /api/v1/meta/sync-runs.
- GET /api/v1/meta/wabas, /phone-numbers, /business-profiles, /health.
- GET /api/v1/meta/webhook-events and /{id}; GET /{id}/payload exposes audited redacted structure only.
- POST /api/v1/meta/webhook-events/{id}/replay accepts no arbitrary payload.
- GET/POST /api/v1/meta/webhooks/{callback}; public authentication is Verify Token for GET and App Secret HMAC for POST.

Asset access uses meta.view/manage; event access uses webhooks.view, webhooks.payload.view and webhooks.replay. Resource scope is ORG; TEAM/SELF grants cannot authorize an organization-wide asset. Organization derives from the authenticated session, never a header or arbitrary supplied ID. Existing membership locks, current grants, verified email, reauthentication and production MFA protect mutations; Origin/double-cookie/session-bound CSRF remain mandatory. List responses use accessible UUID cursors, 100 rows and has_more/next_cursor. Queued commands return 200 with queued=true, not an external delivery claim.
