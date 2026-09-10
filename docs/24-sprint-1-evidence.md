# Sprint 1 implementation and evidence

Review date: 2026-09-10. Sprint 0 implementation is COMPLETE, Gate A APPROVED and Sprint 1 AUTHORIZED by the owner override. Gates B/C/D remain CLOSED.

**Sprint 1 IMPLEMENTATION COMPLETE; HOSTED CI VERIFIED; Sprint 1 acceptance COMPLETE.** Sprint 1 is published at d4fed8ce7b50ed44f0f356e9ebea8f21695fe7bd. Hosted run 34436595874 executed and failed a repository migration checksum check. Corrective published commit 9b2c67187eea6fc2ee6b5a1929f063e76cc6797b passed the full workflow in run 34437626714. Brevo now accepts all four controlled identity messages; the operator confirms actual receipt of all four, including the initially delayed invitation. Sprint 2 is not authorized.

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
| Controlled external SMTP mailbox | PASS: verified STARTTLS/authentication/MAIL/RCPT/DATA and relay acceptance for four messages; operator confirmed all four mailbox receipts |
| Hosted GitHub CI | Sprint 1 published at d4fed8c; run 34436595874 FAILED migration checksum validation. Corrective 9b2c671: full run 34437626714 PASS |

The browser harness uses a disposable database and local SMTP mailbox; its test-only mail/TOTP routes are compiled only with integration,e2e build tags. No production test endpoint or real mailbox credential is introduced. Login and management screenshots were inspected locally; screenshots are ignored artifacts, not production user data. These are correctness checks, not load benchmarks or an independent security audit.

## External evidence and gates

Hosted [run 34356265759](https://github.com/mbingsdk/whats/actions/runs/34356265759), commit 993c67e93fe5945ec820efe67d9d3abf3df8b1b3: attempt 1 failed before any repository steps because of the external account billing lock. Subsequent attempt 3 executed the full Sprint 0 workflow successfully; authenticated API inspection verified every job step and the [historical/latest record](23-sprint-0-evidence.md) preserves both outcomes. This is historical Sprint 0 evidence only. Sprint 1 publication and the actual repository failure are recorded below; Sprint 0 success is not Sprint 1 verification.

The operator authorized controlled identity mail and updated the protected Brevo SMTP credentials on 2026-09-10. The earlier attempt failed SMTP_AUTH before MAIL/RCPT/DATA. The repeated opt-in TestControlledSMTPMailbox passed using isolated local databases and the real outbox/SMTP path: verified STARTTLS, authentication, MAIL FROM, RCPT TO and DATA completed successfully for verification, reset, security notification and invitation. All one-use proofs were consumed successfully. No SMTP implementation change was made.

| Purpose | Relay acceptance UTC | Delivery evidence ID |
| --- | --- | --- |
| Email verification | 2026-09-10T04:28:59Z | 01a08993-7702-7d12-ae2c-159941bb05b3 |
| Password reset | 2026-09-10T04:29:01Z | 01a08993-7dda-769b-85be-bba0d611fc1c |
| Account security notification | 2026-09-10T04:29:02Z | 01a08993-8423-727d-857d-31fff65cbe7b |
| Company invitation | 2026-09-10T04:29:04Z | 01a08993-8ba2-7508-93f6-9c9b896cfc56 |

Relay acceptance is not mailbox receipt. The operator confirmed receipt of verification, reset and security messages, and explicitly reported that Company invitation had not arrived. The operator subsequently confirmed receipt of Company invitation during this closure pass on 2026-09-10. Controlled mailbox evidence is now 4/4; acceptance is based on actual receipt as well as relay acceptance. Protected configuration and raw test artifacts remain ignored; recipient address, SMTP key and action proofs are not recorded here. The [controlled procedure](16-deployment.md#sprint-1-identity-operations) remains unchanged.

Sprint 1 implementation is COMPLETE and hosted verification PASSED. Sprint 1 acceptance is COMPLETE: verification, reset, security and invitation messages are all confirmed received. No external SMTP or hosted CI evidence blocker remains.

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

## Published Sprint 1 CI correction, 2026-09-10

Authenticated gh logs for [run 34436595874](https://github.com/mbingsdk/whats/actions/runs/34436595874), commit d4fed8ce7b50ed44f0f356e9ebea8f21695fe7bd, job 102743170944, show successful runner setup, PostgreSQL startup and both migration commands. At 04:19:44Z, python3 scripts/dev.py check failed in its first child, python3 scripts/check.py: migration checksum mismatch. The subsequent Go/frontend/OpenAPI checks did not execute; browser and dependency steps were skipped because of that failure. This was not the historical account restriction.

The original manifest records 00002_identity_access.sql as 8f3ec7ab37753b3b2a6e233e0a8cb53c271d7ac88dd9a8795ecb839c454b9660, the digest of its Windows working file with 229 CRLF endings. The published Git blob has 229 LF endings and digest d3ac6980f25afc379e9964dc2d799997d902856e601d9c86a758042f132513d2. Expanding that published LF blob to CRLF reproduces the original manifest digest exactly. The existing *.sql text eol=lf rule explains the difference. Previous local checks, including the Linux container reading the Windows working tree, did not exercise Git checkout normalization.

Published SQL blobs and manifest.sha256 remain immutable. The additive checksum-corrections.json records the original digest, actual published artifact digest, source commit and rationale. Validation applies a correction only to its matching historical manifest entry and requires the exact published bytes; it does not normalize bytes, accept both line endings, regenerate hashes or skip integrity checks. Seven regression tests cover exact LF acceptance, CRLF rejection, SQL tampering, missing/extra migrations, correction mismatch/duplicates/malformed metadata and uncorrected migration integrity. scripts/dev.py check runs these tests in addition to all existing checks. No schema migration or application architecture changed.

Corrective commit: 9b2c67187eea6fc2ee6b5a1929f063e76cc6797b. Full hosted outcome: [run 34437626714](https://github.com/mbingsdk/whats/actions/runs/34437626714) SUCCESS, foundation job 102745898048, 2026-09-10T04:33:18Z to 04:36:28Z. Authenticated run metadata and completed logs were inspected. Every mandatory step passed; this is Sprint 1 evidence from a corrective descendant of d4fed8c, not reused Sprint 0 evidence.

Post-correction local execution: python scripts/dev.py up and the full check both exited 0, including seven new checksum regressions, all existing explicit real-PostgreSQL identity/foundation/security tests, Go format/vet/build, five frontend tests, lint/typecheck, OpenAPI validation and Next production build. Browser E2E passed again (14.079 seconds). go mod verify passed; pinned govulncheck 1.8.0 again reported zero affected symbols/imported packages and one unused module advisory; npm audit reported zero findings. Source validation now resolves 92 local links. A clean git archive of d4fed8c reproduced the original checksum failure; applying only the additive correction/checker made that same LF SQL and historical manifest pass. Neither migration SQL nor the original manifest differs from its published Git blob.

## Corrective hosted execution and closure review

| Hosted stage | Actual result |
| --- | --- |
| PostgreSQL and migration/repeated migration | PASS |
| Source, links, strict SQL integrity and seven Python integrity regressions | PASS |
| Go formatting, vet, unit tests and real PostgreSQL integration tests with race detector; Go build | PASS |
| npm ci, lint, typecheck, five frontend tests, OpenAPI validation and Next production build | PASS |
| Pinned Chromium browser identity flows with PostgreSQL and local TLS SMTP, race detector enabled | PASS |
| go mod verify, govulncheck 1.8.0, npm audit and unchanged dependency manifests | PASS; zero affected symbols/imported packages, one unused OpenPGP module advisory disclosed above; npm zero findings |
| Cleanup and complete workflow | PASS |

Security regressions executed again locally and on the corrective hosted commit: cross-organization access and every new tenant table's FORCE RLS/missing-context rejection; runtime privileges and pooled-context reuse; session/reauthentication/MFA-reset invalidation and CSRF; invitation/reset/verification concurrent one-use consumption; recovery-code one-use and TOTP replay; offboarding ordering, scope/delegation escalation rejection and concurrent last-Owner protection; mail claim/terminal replay/lease fencing and expiry; audit redaction and append-only protection. All passed, with no reported Go data race. No new application security defect was found in this focused pass; the discovered defect was publication/checksum provenance. These tests are not an independent security audit.

All five requested status documents are reconciled, with the directly dependent database checksum explanation and review/agent status updated. CI configuration, published migration SQL and the historical manifest were preserved. No new migration, endpoint, SMTP architecture change or Sprint 2 functionality was introduced. Implementation COMPLETE and hosted CI VERIFIED are accompanied by acceptance COMPLETE after all four controlled mailbox receipts were confirmed. Gates B/C/D remain CLOSED.
