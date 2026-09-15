# Security, access and audit

## Permission evaluation

Authorize `(principal, organization, permission, resource, current policy revision)`. Membership must be ACTIVE and global user unsuspended. Effective grants are the union of current member-role and active team-role grants, each constrained by ORG/TEAM/WABA/PHONE/SELF scope and expiry. Explicit suppression and sending policy denials override permission grants. There are no per-user deny/override exceptions initially; custom roles suffice.

TEAM scope follows current conversation team; WABA/PHONE scope constrains sender resources. Contact access is derived from authorized relationships or explicit organization-wide contact permission; it never reveals unauthorized conversation history. API keys intersect explicit scopes and phone allowlist with active sponsor/delegation policy. No wildcard over future phone numbers by default.

Role administration is permission plus delegation ceiling: one cannot grant a permission or resource scope one lacks authority to delegate. Editing a role is evaluated against every affected grant; it cannot escalate users through an indirect role edit. team.manage alone cannot add oneself to a privileged team or alter team roles. The same ceiling applies to invitations and API keys. Preserve at least one active recovery administrator with verified recovery method; role label Owner has no special branch in code.

## Permission catalog and default role presets

| Family | Actions |
| --- | --- |
| Inbox | inbox.view, inbox.reply, inbox.assign, inbox.note, inbox.resolve, inbox.snooze, inbox.handoff |
| Contacts/consent | contacts.view, contacts.edit, contacts.import, contacts.export, consent.manage, consent.suppress, consent.restore |
| Audience/campaign | audience.view, audience.manage; campaign.view, campaign.create, campaign.test, campaign.submit, campaign.approve, campaign.schedule, campaign.send, campaign.pause, campaign.cancel |
| Templates/Flows | template.view, template.create, template.edit, template.delete, template.test; flow.view, flow.manage, flow.responses.view |
| Controls | pricing.confirm, pricing.registry.view, pricing.registry.import, pricing.registry.review, pricing.registry.publish, billing.view, billing.budgets.manage, billing.reconcile, approval.view; action-specific approve permissions |
| Meta | meta.view, meta.assets.manage, meta.phone.manage, meta.phone.block, meta.credentials.manage, meta.webhooks.manage, meta.health.view |
| Organization | members.view, members.invite, members.manage, teams.view, teams.manage, roles.view, roles.manage, roles.assign |
| Integrations | messages.send (API), media.upload, developer.keys.manage, developer.webhooks.manage, developer.webhooks.replay, developer.events.view |
| Automation | automation.view, automation.manage, automation.publish |
| Operations | operations.webhooks.view, operations.payloads.view, operations.webhooks.replay, messaging.reconcile, audit.view, analytics.view |

Origin-specific send permission: agent uses inbox.reply; gateway uses messages.send; campaign uses current run authority originally granted through campaign.send; automation uses published rule authority bounded by its service principal's send scope. All still pass the common controls. Media access derives from underlying resource scope, not a global download permission.

| Preset | Initial grants (not exhaustive hidden wildcard) | Deliberate limits |
| --- | --- | --- |
| Owner | Explicit administrative catalog with org scope and recovery stewardship | No consent/pricing bypass; self-approval policy still applies |
| Admin | Members/teams/scoped roles, Meta assets, operations, configurable budget management | Credential access/replay require MFA/reauth; cannot exceed delegable ceiling |
| Supervisor | Scoped inbox assignment/handoff, campaign review/pause, operational analytics | Approval only within configured monetary/count thresholds |
| Service Agent | Team inbox view/reply/note/resolve/snooze, scoped contact view/edit, consent.suppress | No campaign approval/export/credential access by default |
| Campaign Manager | Campaign draft/test/submit/schedule/send/pause/cancel, audience and template view | No campaign.approve by default; authority/confirmation still required |
| Developer | Scoped integration keys/endpoints/logs and technical diagnostics | No raw payload or financial/credential access by default |
| Viewer | Explicit scoped read grants | No implicit PII export, media download outside accessible records or cost access |

Seed permission lists are explicit and reviewable. New permissions are not silently granted to all existing “Admin” roles. Resource access and permission checks occur in backend and realtime/outbound delivery paths; frontend hiding is only usability.

## Authentication and secrets

Passwords: Argon2id with per-password salt and versioned parameters; benchmark on deployment hardware, starting from current OWASP guidance rather than reducing work to improve login benchmarks. Rate-limit login/reset by account and network signals, use generic failure responses, bounded progressive delay and alerting rather than permanent lockout that attackers can exploit. [OWASP password storage](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html).

Sessions: high-entropy opaque tokens stored as digests; Secure, HttpOnly, SameSite=Lax cookies scoped to host/path. Rotate at login/MFA/privilege change. Proposed expiry: 12-hour absolute, 30-minute idle; policy-configurable. Reauthentication window 5 minutes for credentials, role escalation, budget changes, suppression restore, exports at sensitive scale and replay. Revocation checked on every request or transactionally validated access revision; privileged commands bypass stale caches. SameSite is not the sole CSRF defense: use CSRF tokens and Origin checks. No state changes on GET.

TOTP-ready schema is included; require privileged users to enroll before production. Prevent reuse within accepted TOTP timestep, limit clock skew and attempts, encrypt factor secrets, hash recovery codes and record consumption atomically. Enrollment/removal requires recent reauthentication; replacement recovery cannot be an unaudited support override. Passkeys/SSO are DEFERRED; choose a reviewed integration later.

Meta tokens/app secrets, TOTP seeds and outgoing-webhook signing secrets use envelope encryption with AEAD, random nonce, per-secret data key and associated data `(organization,credential ID,purpose,version)`. Wrap data keys under an external root key delivered as a protected systemd credential/file or managed KMS. Root key is not in DB, source, image, logs or DB-only backup. Separate production/staging roots. Initial VPS root-file strategy has host-compromise risk; KMS is preferred when available without unnecessary service complexity. Rotation rewraps keys and validates replacement tokens before cutover; preserve required old keys through backup retention.

API keys: random 256-bit secret, prefix for lookup, keyed hash/digest at rest, constant-time comparison, show once, expiry/rotation/revocation, explicit phones/scopes and optional spend/rate limits. Hashing is appropriate because we verify keys, whereas signing secrets must be recoverable. Personal key revokes with member; service key has active sponsor and organization owner. Logs redact Authorization, access_token query parameters, cookies and credential fingerprints beyond approved prefix.

## Threat controls

| Threat | Concrete controls and residual boundary |
| --- | --- |
| Tenant leakage | Composite FKs, FORCE RLS, tenant-scoped cache/search/object keys, request/job/realtime authorization tests; app DB role cannot bypass |
| Privilege escalation | Delegation ceiling for role edits/team changes/invites/keys; current-policy approval eligibility; last recovery admin guard |
| XSS/content abuse | Escape message/notes; safe link schemes; CSP nonces, no unsafe-inline, no raw customer HTML; attachment isolation and MIME scan |
| CSRF/session theft | Same-origin commands, CSRF token, secure cookies, session rotation/revocation, TLS and reauth |
| SSRF/exfiltration | Outbound URL allowlist, destination IP revalidation, block redirects/private metadata, egress firewall, bounded responses |
| Forged webhook | Verify exact bytes/signature before persistence; app-bound route and WABA/phone checks; current Meta protocol verification gate |
| Duplicate/replayed actions | Unique event/effect/client keys; immutable send scope; confirmation one-use; replay cannot issue new effects |
| Secret leakage | Write-only API, envelope encryption, least-privilege decryption service, redacted errors/audit, no tokens in frontend |
| CSV/media attacks | Formula neutralization, file scan/type limits, no executable inline documents, private short-lived URLs |
| Backup/host failure | Encrypted offsite backups, distinct root-key recovery, restore/erasure runbook and timed drills |

Set HSTS after HTTPS deployment validation, CSP, frame-ancestors, nosniff and restrictive referrer policy. Trust forwarded IP headers only from known Nginx. Keep DB/Valkey private. Application workers use least-privilege OS/DB accounts; credentials are unavailable to frontend build processes. Dependency scanning/patch cadence is explicit in deployment process.

## Audit and offboarding

Critical mutation transaction writes actor kind, stable user/member ID, historical actor label, org/team, action, resource/revision, redacted before/after or change digest, UTC timestamp, IP/UA if available, request/correlation and policy/approval references. No token, raw message body, consent document or secret ciphertext in audit. Raw diagnostic access and exports themselves are audited. Retain actor rows after deactivation; legal erasure may pseudonymize labels while preserving attributable authorized records under company policy.

Offboarding transaction sets inactive/access revision, revokes org sessions and personal keys, invalidates relevant personal approvals/queued intents, removes active team assignments and starts reassignment. Current mandatory approver authority is reevaluated before pending execution. Organization-owned campaigns with another valid sponsor may continue under policy; otherwise pause. Close live streams and cancel queued exports. Signed media links have a short documented residual lifetime; revoke/rotate object keys for urgent incidents when necessary.

Audit append-only DB permissions deter application modification; periodic hash-chained/exported manifests to separate immutable storage can add tamper evidence. Do not claim a local append-only table survives a malicious database administrator. Security incidents require credential rotation, session/key revocation, send pause, evidence preservation and restoration testing.

## Transactional identity mail (required)

Identity owns a small provider-neutral MailSender boundary: versioned template, purpose, recipient, delivery ID and sanitized SMTP_ACCEPTED/REJECTED/UNKNOWN result. Initial transport is authenticated SMTP over verified TLS, configured in Sprint 0 and exercised in Sprint 1. Invitation, email verification, password reset and notices of password/email/MFA or other credential changes are required; there is no marketing-email subsystem.

Create the challenge and durable mail delivery in one DB transaction. Send outside that transaction. SMTP acceptance is not proof of inbox delivery and never activates an invitation or marks an address verified. Proposed deadlines: invitation 72 hours (administrator may shorten), verification 24 hours, reset 30 minutes. Store challenge digests, bind purpose/user or invitation/email/generation, and consume exactly once atomically on explicit completion POST; GET/link scanners cannot consume. Retry reuses the same token and deadline. Explicit resend creates a new generation, revokes the old challenge and cancels its unsent deliveries. An already transmitted older email may arrive; its link fails safely. Reset revokes sessions and emits a security notice; email change notifies the old verified address and verifies the challenge-bound new address before replacement. Invitation acceptance can establish a new verified identity and password only for its bound address; an existing identity requires authentication and cannot have its password replaced by invitation proof. This is invitation-only setup, not public signup.

Only the required short-lived, single-purpose action link may carry a credential. Never email passwords, Meta/API/session tokens, MFA seeds/recovery codes or customer content. Put the action token in a URL fragment, clear it before subsequent navigation and submit by HTTPS POST; no-referrer and no third-party scripts on completion screens. Do not log email bodies, recipient addresses, fragments, request tokens or provider payloads. Encrypt the minimum recipient/template payload needed by the worker and purge it after terminal delivery or expiry, retaining only sanitized delivery evidence.

Mail states: QUEUED -> SENDING -> SMTP_ACCEPTED / RETRY_WAIT / FAILED; RETRY_WAIT -> SENDING; undispatched QUEUED/RETRY_WAIT -> EXPIRED / CANCELLED. SMTP_ACCEPTED, FAILED, EXPIRED and CANCELLED are terminal. Temporary transport/4xx failures or ambiguous SMTP acceptance allow bounded exponential backoff with jitter, at most eight attempts within 24 hours and never beyond the challenge deadline. Duplicate email with the same one-use link is acceptable; this exception never permits ambiguous WhatsApp send retries. Stable Message-ID aids diagnosis but does not guarantee deduplication. Permanent recipient rejection fails delivery; authentication/TLS errors pause the mail pool and alert operations. SENDING may also become EXPIRED/CANCELLED when a final source/deadline check proves transmission has not started. Retry exhaustion becomes FAILED; expiry never extends a challenge. A crashed SENDING lease is reconciled to RETRY_WAIT, or EXPIRED/CANCELLED when no longer eligible; no token renewal.

Request endpoints give generic accepted responses regardless of account existence/provider failure. An authorized inviter can see sanitized delivery health and resend; tenant administrators cannot inspect global reset/verification jobs. Security notices have a 24-hour delivery deadline and no action token. Suppress obsolete challenges immediately before SMTP; races after transmission cannot recall email. Token expiry and one-use enforcement remain authoritative.

Sprint 0 implemented SMTP configuration and a TLS/NOOP probe. Sprint 1 implements authenticated TLS/STARTTLS delivery, the durable outbox, invitation/verification/reset/security messages and local SMTP/browser tests. The controlled-mailbox procedure and remaining provider information are recorded in [Sprint 0 evidence](23-sprint-0-evidence.md).

## Sprint 1 concrete security policy

[ADR 011](../ADR/011-identity-runtime.md) and [OpenAPI](../contracts/openapi.yaml) define the implemented identity boundary. Argon2id uses 19 MiB, two iterations and one lane with random 16-byte salt and a 32-byte result, bounded verification parameters and rehash on successful login. The current policy is fixed in code, not a runtime administrator setting. Cookies have 12-hour absolute and 30-minute idle expiry; rotation preserves the original absolute deadline. Only actual reauthentication refreshes the five-minute sensitive window; changing organization cannot refresh it. Password reset/MFA changes invalidate pending login-MFA proofs as well as sessions.

Organization membership/role/team changes invalidate affected organization sessions using live access revisions. Those sessions must sign in again; an authorized acting session rotates if its own access changes and remains active. Offboarding retains identity/audit history, removes active team assignments and immediately rejects old organization sessions. A user may still sign in to manage their own identity or use another active organization. Future keys/streams/exports have no implemented surfaces to revoke in Sprint 1.

Every production organization mutation requires verified email, enrolled MFA and recent authentication, with permission/resource scope and delegation checked inside the organization barrier. Privileged users who disable MFA lose privileged mutation capability until re-enrollment. SMTP sends are bounded to 90 seconds within a two-minute fenced lease. Retry is at most eight attempts within 24 hours and never beyond source expiry. Mail-pool errors emit a sanitized operational log and pause the worker for one minute. Production alert routing and controlled external receipt remain operational acceptance evidence.

## Sprint 3 permissions and boundaries

New permissions: inbox.view, messages.send, conversations.manage, conversations.assign, notes.write, notes.redact, pricing.view, pricing.registry.import, pricing.registry.review, pricing.registry.publish, budgets.manage and sending.manage. Migration backfills through owners.manage authority; request authorization never checks role names. Organization discovery reads the permission catalog including self/team grants; each thread returns current command permissions.

The cookie-derived domain session rechecks user status/auth revision, active membership/access revision, session expiry/idle/revocation, verified-email writes and CSRF transactionally. Sensitive operations require recent MFA. SSE repeats authorization every four seconds. Search, history, notes, presence and events apply organization/team scope. Both runtime roles pass populated-table RLS tests; setup/migration credentials never enter API/worker execution.

TEST designation is protected-file configuration, not inferred from a stored recipient, and remains campaign-excluded. Final guard binds configured App/WABA/phone and requires challenge plus authentic eligible inbound after INBOX_LIVE_ACCEPTANCE_NOT_BEFORE. This setting records an approved acceptance start; it neither changes callback nor overrides pricing.
