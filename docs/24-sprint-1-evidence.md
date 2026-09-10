# Sprint 1 implementation and evidence

Review date: 2026-09-10. Sprint 0 implementation is COMPLETE, Gate A APPROVED and Sprint 1 AUTHORIZED by the owner override. Gates B/C/D remain CLOSED.

**Sprint 1 implementation COMPLETE; Sprint 1 acceptance OPEN** pending controlled SMTP mailbox evidence. Current Sprint 1 hosted verification is PENDING because these workspace changes have not been published; the Sprint 0 baseline has verified attempt-3 hosted success. External SMTP failure does not invalidate the passing local implementation evidence.

## Implemented scope

One-time operator Owner bootstrap; Argon2id email/password login; opaque sessions, CSRF and bounded account/IP throttling; email verification and reset; invitations, activation/reactivation/offboarding; multiple teams and granular direct/inherited ORG/TEAM/SELF permissions; Owner/delegation invariants; TOTP, one-use recovery codes, sensitive reauthentication and active-session management; transactional TLS SMTP with durable outbox; tenant/global audit; frontend identity, members, teams, roles, security and audit pages.

No Meta/WABA adapter, Inbox, Contacts CRM, Templates, WhatsApp messaging, Pricing Guard, Campaign/Broadcast, Automation, Flows or Catalog implementation was added. This does not establish production readiness.

## Migrations and privileges

Added 00002_identity_access.sql; 00001 remains immutable. Schema version is 2. New role bootstrap creates waba_identity with NOINHERIT/NOBYPASSRLS and no object ownership. API and worker do not use migration/authorizer/setup-admin credentials.

| Boundary | New tables |
| --- | --- |
| Organization-owned, FORCE RLS | roles, role_permissions, teams, team_members, member_roles, team_roles, invitations, invitation_roles, audit_log |
| Explicit global identity/catalog/mail exception | identity_bootstrap, password_identities, sessions, auth_challenges, auth_factors, auth_recovery_codes, login_limits, identity_audit, permissions, mail_deliveries, mail_attempts |

users gains verified_at. Tenant relationships use composite organization keys. Global sessions carry authenticated user plus optional organization/access revision. Invitation mail binds organization/invitation; tenant APIs cannot list global verification/reset jobs. Audit is append-only for runtime credentials, with actor snapshot, resource/revision and change digest; completed mail attempts have a protection trigger. Full rationale is in [ADR 011](../ADR/011-identity-runtime.md).

## Authentication, organization and recovery

Passwords use Argon2id (19 MiB, two iterations, one lane, random salt); successful login can rehash. Session tokens and action/recovery proofs are random 256-bit values stored as SHA-256 digests. Cookies are host/path scoped, HttpOnly for session, SameSite=Lax and Secure in production. Mutations require exact Origin, matching CSRF cookie/header and authenticated session digest. No user-ID or organization-header authentication exists.

Sessions expire at 12 hours absolute / 30 minutes idle. Rotation preserves absolute expiry and reauthentication time; changing organization cannot refresh the five-minute sensitive window. Reset and MFA changes invalidate global auth revision and pending login-MFA proofs. Role/team/member access revisions invalidate affected organization sessions; active actors rotate as needed. Offboarded employees can still sign in for global identity security or other valid memberships, but the removed organization's old sessions fail.

Permissions are explicit keys, not role-name conditions. Owner/Admin/Agent are bootstrap presets. Direct grants can combine ORG/TEAM/SELF resources; inherited team grants stay team-scoped. Team lists filter to current permission scope; archived teams confer no team-scoped authority. Grant writers use the exclusive organization barrier, compare revisions and enforce delegation ceilings/at least one active organization-wide owners.manage grant. Invitation acceptance rechecks inviter authority, uses one proof once and cannot replace an existing account password.

TOTP uses the maintained pquerna/otp library, a 30-second step, bounded skew and replay prevention. Per-factor envelope encryption protects seeds. Ten recovery codes are shown once, digested and atomically consumed; regeneration/disable invalidate old proofs and sessions. Sensitive production organization changes require verified email, MFA enrollment and recent reauthentication.

## SMTP and audit

Identity mutations commit their challenge and encrypted outbox payload atomically. The worker claims with SKIP LOCKED, a fenced two-minute lease and at most 90 seconds per SMTP effect. Verified TLS/STARTTLS and authentication are mandatory; plaintext/certificate bypass is rejected. Eight attempts maximum, exponential jitter, 24-hour mail horizon bounded by original challenge expiry. Resend supersedes proofs; source/deadline checks suppress obsolete mail. Terminal states purge payload ciphertext.

Ambiguous SMTP may duplicate the same one-use link and stable Message-ID; it never renews proof expiry. SMTP_ACCEPTED proves relay acceptance only. Permanent rejection fails; auth/TLS errors pause and emit sanitized operational logs. Crashed leases retain UNKNOWN attempt evidence. No SMTP response body, recipient, password, session token, action link, seed or recovery code is logged in audit. Organization audit stores actor/resource/revision/change digest; global security events remain inaccessible to tenant list APIs.

## Frontend and contracts

Ten route surfaces: /login, /forgot-password, /reset-password, /verify-email, /accept-invitation, /security, /members, /teams, /roles and /audit. Root goes to security. Role visibility follows permissions; sensitive actions explain consequences, MFA/recovery secrets remain component memory, action fragments are cleared before completion, and no public company registration exists. Only multiple active memberships expose an organization switcher.

[OpenAPI](../contracts/openapi.yaml) describes 41 identity/access operations plus existing health/readiness. Requests carry current If-Match for organization aggregate changes. Lists return at most 100 rows with signed cursors bound to authenticated user, organization and endpoint/resource; every page reauthorizes. The frontend supports continuation and complete role/member selection lists. Source validation checks implemented route coverage as well as separate OpenAPI syntax validation.

## Executed evidence

| Check | Result |
| --- | --- |
| Real PostgreSQL identity/security suite | PASS locally, including one-winner invitation/reset/verification/recovery consumption, offboarding ordering and last-Owner concurrency |
| Foundation PostgreSQL suite | PASS: nine subtests, runtime privilege/tenant boundary/pool context reuse retained |
| Nine new tenant tables | PASS: FORCE RLS, missing context, bidirectional foreign-tenant reads/writes/moves, composite FK rejection and identity-role limits |
| Regression suite | PASS locally for signed pagination/SELF scope, MFA/reset invalidation, reauthentication expiry, invitation supersession/expiry/existing account protection, SMTP terminal/pause/deadline and account-throttle bypass |
| SMTP transport | PASS: actual local TLS and STARTTLS authentication/MAIL/RCPT/DATA; untrusted certificates rejected |
| Browser E2E | PASS locally through actual Next UI + Go + PostgreSQL + local TLS SMTP: login, invitation receipt/acceptance, hidden/denied permissions, teams, TOTP/recovery login, session revoke, reset invalidation, offboarding, mobile login |
| Frontend lint/typecheck/unit tests | PASS; five unit tests, zero skipped/failed |
| Final complete check/build/contracts | PASS: python scripts/dev.py check exited 0; corrected OpenAPI revalidated with no warnings |
| Linux race detector | PASS: current Go unit + all real PostgreSQL integration packages, -race -tags=integration -count=1 ./..., pinned Linux container |
| Dependency integrity/vulnerability review | go mod verify PASS; npm audit: zero findings; govulncheck 1.8.0: zero affected symbols/imported packages, one unused OpenPGP module advisory (details below) |
| Controlled external SMTP mailbox | ATTEMPTED: SMTP_AUTH before MAIL/RCPT/DATA; receipt PENDING - EXTERNAL CREDENTIAL BLOCKER |
| Hosted GitHub CI | Sprint 0 attempt 1 external failure; attempt 3 actual PASS. Sprint 1 current changes PENDING publication/hosted verification |

The browser harness uses a disposable database and local SMTP mailbox; its test-only mail/TOTP routes are compiled only with integration,e2e build tags. No production test endpoint or real mailbox credential is introduced. Login and management screenshots were inspected locally; screenshots are ignored artifacts, not production user data. These are correctness checks, not load benchmarks or an independent security audit.

## External evidence and gates

Hosted [run 34356265759](https://github.com/mbingsdk/whats/actions/runs/34356265759), commit 993c67e93fe5945ec820efe67d9d3abf3df8b1b3: attempt 1 failed before any repository steps because of the external account billing lock. Subsequent attempt 3 executed the full Sprint 0 workflow successfully; authenticated API inspection verified every job step and the [historical/latest record](23-sprint-0-evidence.md) preserves both outcomes. Current Sprint 1 workspace changes are not published and have no hosted result. The unchanged mandatory CI checks plus added Chromium E2E must run on the published Sprint 1 commit; any actual repository failure must be fixed.

The user supplied protected SMTP configuration, confirmed authorization and designated a controlled mailbox on 2026-09-10. Configuration was retained outside source under .local/operator-settings-20260910.env; uppercase STARTTLS was normalized only in the private test runner. Database URL values incorrectly entered in *_FILE fields were not used; the controlled test used isolated local database credentials. A manually opted-in integration,controlledsmtp test exercised the real outbox/SMTP path, but the relay returned SMTP_AUTH before message transmission. No controlled email was accepted or receipt confirmed. Fix the SMTP username/app-password file and rerun the [controlled procedure](16-deployment.md#sprint-1-identity-operations). Do not commit mailbox addresses, passwords or action proofs.

Sprint 1 acceptance remains OPEN for this explicit external SMTP evidence blocker. The full SMTP/mail-outbox implementation has passing local TLS/STARTTLS and failure/concurrency tests. Hosted verification of the new commit is reported separately, not as the old unresolved billing restriction.

Gate B remains CLOSED throughout Sprint 1. Gate C and Gate D remain CLOSED. No paid send, Meta account mutation, deployment or production readiness claim is authorized by this implementation record.

## Implemented endpoint inventory

All paths below use /api/v1; /healthz and /readyz remain foundation endpoints.

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

## Final local verification details

python scripts/dev.py up passed. Repeated migrate calls each applied zero migrations to the already version-2 development database; its current session access_revision, audit change_digest, mail template_version and authorizer result schema were inspected. Clean disposable databases apply both migrations successfully in integration tests. Compiled API smoke returned 200 health/readiness with request-ID propagation and only limited runtime/identity credentials.

python scripts/dev.py check passed source/format/vet, unit and explicit PostgreSQL tests, Go build, npm ci, lint/typecheck, five frontend tests, OpenAPI and production Next build. A mixed-grant schema warning was corrected, then npm run contracts passed without warnings. Final python scripts/dev.py e2e passed after the scope/session changes. Actionlint 1.7.12 and git diff --check passed; source verification counted 91 resolving local links and unchanged 00001 SQL/checksum.

Current Linux race execution used golang:1.27.1-bookworm@sha256:648f440f42a0958804efb24df176f806f9d353b41f1c0627f666428e40310f6b with read-only source and isolated protected test configuration. Every unit/integration package passed; no race was reported. This is current Sprint 1 evidence, not reused Sprint 0 output.

[GO-2026-5932](https://pkg.go.dev/vuln/GO-2026-5932), inspected 2026-09-10, applies to the unmaintained OpenPGP packages within golang.org/x/crypto and has no fixed version. This application imports Argon2id, not OpenPGP; go list -deps confirmed no OpenPGP package in the compiled dependency graph. govulncheck reports zero affected symbols/imported packages, with this one module-level advisory. No vulnerability check or advisory was suppressed.

Final password-policy review counts Unicode characters (12-256), with a bounded UTF-8 byte length for verification; regression tests reject four multibyte characters and accept a valid long Unicode password. Generated Python bytecode was removed from source and ignored.
