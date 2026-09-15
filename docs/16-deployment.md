# Initial VPS deployment and recovery

Proposed topology: one Linux VPS with Nginx, Go API, separate Go worker pools, Next.js Node process, PostgreSQL and Valkey; external private S3-compatible object storage and encrypted offsite backups. Start load testing on a provisional 8-vCPU/16-GiB host with SSD storage sized from measured message/index growth. This is a test baseline, not a purchased host or benchmark guarantee. No Kubernetes, Kafka or microservice orchestration.

## Process strategy

Use systemd for API, worker pools and Next.js; distribution-supported services for Nginx/PostgreSQL/Valkey. Build artifacts in CI, never on the production VPS. Immutable versioned release directories and explicit environment/secret files; symlink switch plus service restart deploys one release. Advantages: familiar Linux recovery, bounded CPU/memory and few moving parts. Containers remain viable if the company already operates them; adding both supervision models would increase toil without current benefit. [Deployment ADRs](../ADR/README.md).

Run separate Unix accounts, private loopback sockets/ports, filesystem write restrictions and service restart limits. Public ports only TLS/HTTP redirect and tightly restricted administration. PostgreSQL and Valkey never bind publicly. Nginx sends `/api`/`webhooks`/`events` to Go and other paths to Next.js; SSE disables buffering and has heartbeat-compatible timeout. Upload/body limits are route-specific. TLS certificates renew automatically with expiry alert; use explicit canonical host and trusted proxy settings.

Pin Go/Next/Postgres/Valkey supported versions and dependency locks after release/security verification. Record build ID, schema compatibility range, configured Graph version and pricing policy coverage in release manifest. Separate staging from production credentials, DB and object prefixes. Do not load production raw payloads into development without approved redaction.

## Configuration and secrets

Nonsecret config: required SMTP host/port, TLS mode with certificate verification, sender address, timeouts and identity template versions; public origin, DB pool sizing, queue concurrency, object bucket/region, Graph version, request limits, allowed egress, timeouts, policy identifiers. Validate required config at startup and reject unknown/unsupported API version. Secrets: SMTP authentication credential, database role password, wrapping root key reference, session/API HMAC keys, object credentials and monitoring tokens via protected service credentials; Meta secrets themselves encrypted in DB. Never embed secrets in frontend environment or build outputs.

Distinct DB roles for migrations, runtime API, workers, backup and restricted routing. Runtime roles have no table ownership/BYPASSRLS. Set tenant context with SET LOCAL within transactions, never session-sticky settings in pooled connections. Size pools below DB connection cap; reserve connections for ingestion and operational recovery. Resource limits prioritize DB/ingress over bulk campaigns and media scanning.

## Deploy and rollback

1. CI compiles/tests/scans pinned artifacts and publishes checksums. Confirm backups and pricing/Graph coverage.
2. Apply forward-compatible expand migration with explicit statement/lock timeout. Add nullable fields or new tables first; use concurrent index creation where appropriate outside transaction. Backfills run resumably with load controls.
3. Deploy API compatible with old/new schema, then workers/Next.js; smoke-test auth/health and controlled test assets. No automatic customer-message smoke test.
4. Shift traffic/restart with graceful drain. Observe error/lag/guard metrics. Keep prior artifact available.
5. Contract/drop migration only in a later release after old binaries/jobs no longer rely on fields and backfill validated.

Rollback selects previous compatible artifact. Do not blindly run destructive down migrations or restore an old DB over new customer messages. If data corruption requires PITR, follow isolated restore and reconciliation; sending stays disabled. An irreversible schema change requires a separate migration plan/review before deployment.

Graceful shutdown: mark not ready, stop job claims and SSE connections with reconnect hint, drain bounded requests, finish local transactions. Dispatch attempts started but not conclusively recorded become UNCERTAIN on recovery. SIGKILL or lease expiry never permits automatic resend. On startup, perform schema compatibility check and recover durable jobs with these rules.

## Health and availability

Liveness checks process responsiveness without depending on Meta. Readiness requires compatible schema, DB access and necessary local dependencies; Meta outage should degrade sender capabilities rather than restart-loop the API. Worker heartbeat includes build and queue; job oldest-age detects stuck-but-alive workers. Object/Valkey/Meta/SMTP probes are separate health entries. Missing required mail configuration prevents identity readiness; transient relay failure degrades mail health and queues delivery without restarting API processes or blocking existing-user login. Configure and test sender/relay access in Sprint 0/1; alert on authentication/TLS failures and oldest identity-mail age.

Proposed internal availability target 99.5% monthly. Single-host failure causes downtime; this design has no HA claim. UPS/provider durability is not a substitute for offsite recovery. Move DB/replication to separate hosts when accepted downtime or capacity no longer fits; keep guard transactions on the authoritative writer.

## Backup and restore runbook

Use daily encrypted PostgreSQL base backup plus continuous WAL archiving to offsite storage, with archive freshness alert targeting RPO 15 min. Tool choice (e.g. pgBackRest) is confirmed during Sprint 0; no backup configuration was installed here. Include schema, role definitions, object manifest, release manifest, policy/rate evidence and independent erasure ledger. Object storage versioning/replication policies must cover deletion/corruption recovery. Root encryption keys are backed up through a separate restricted recovery channel and tested; database-only backup cannot decrypt secrets.

Proposed 35-day backup retention; audit/finance retention is separate. Weekly automated verification checks readable manifests and WAL continuity. Monthly timed restore into isolated network is required to establish RTO 4 h; a successful backup command does not prove restore.

Restore steps: provision clean host/network; retrieve approved artifact/config and separate keys; restore base backup/WAL to selected point; verify schema/checksums/counts and object inventory; replay independently retained erasure/suppression ledger; reconcile durable jobs/outbox; quarantine all attempts that may have dispatched after restore point; rotate suspect secrets and revoke restored sessions/keys as appropriate; verify bindings/rates and incoming callbacks; allow read-only operations first; review send uncertainty with operator; enable sending only after reconciliation evidence. Reuse external message/effect tombstones so replay cannot double-send.

Loss after last recoverable WAL may include outbound effects absent from restored DB. No architecture can reconstruct them by assuming Meta has a message-history API. Keep offsite dispatch/audit manifests where practical; affected time range remains quarantined, with conservative budget exposure and manual reconciliation. This residual risk is a reason to improve RPO before high-volume campaigns, not promise exactly-once recovery.

## Incident procedures

| Incident | Immediate action | Recovery evidence |
| --- | --- | --- |
| Meta/token outage | Pause affected dispatch, retain intents, keep ingress running | Fresh credential/access/capability validation; guards rerun |
| DB full/down | Stop campaign/import load, fail webhook persist with 5xx, protect WAL/log capacity | DB healthy, retries deduped, lag drained |
| Valkey loss | Drop ephemeral presence; pause throttled sends unless safe DB fallback active | Rebuild hints, durable jobs unchanged |
| Object store outage | Queue media, show unavailable, block new unscanned uploads | Fetch/scan backlog recovered; no expired URL loop |
| Poison webhook/parser | Isolate child DLQ, preserve raw/evidence, disable affected automation | Fixture fix + project-only replay diff approved |
| Budget/pricing mismatch | Stop affected paid sends, keep reservations | Reviewed policy/reconciliation and renewed authority |
| Compromised key/account | Revoke key/session, pause sponsor scope, preserve redacted audit | Rotation/access review, scoped reenable |

Rotate journal/Nginx logs with size caps; monitor disk and backup separation. Operations owner needs documented access to domain/DNS, VPS, backups, object storage, recovery keys and Meta administration. Runbooks must work when the sole application maintainer is unavailable.

## Sprint 0 local operations

[Compose](../compose.yaml) pins PostgreSQL 18.6 by digest and binds only 127.0.0.1:55432; it provisions disposable development roles, not production infrastructure. Go/Next run on the host. [Local database/migration instructions](../database/README.md) and [deployment foundation](../deploy/README.md) describe credentials, readiness and later process layout. No real systemd/Nginx service or offsite backup was provisioned; RPO/RTO remain unmeasured.

## Sprint 1 identity operations

Development prerequisites and checks are in [README](../README.md). Run python scripts/dev.py up, then python scripts/dev.py migrate. Migration 00002 creates identity/access tables and requires the dedicated waba_identity role provisioned by up. The current Sprint 2 runtime requires schema version 3, including migration 00003. Use a reviewed forward migration for repairs; do not edit published SQL or regenerate a missing root key.

For initial Owner bootstrap, place a unique 12-256 character password in an operator-only file outside source control. Set BOOTSTRAP_PASSWORD_FILE to its absolute path, BOOTSTRAP_EMAIL to the intended Owner address, BOOTSTRAP_NAME to the initial display/company name, BOOTSTRAP_ORGANIZATION_SLUG to the company slug, BOOTSTRAP_TIMEZONE to the chosen IANA zone, and PUBLIC_ORIGIN to the real application origin. Optionally set BOOTSTRAP_ORGANIZATION_ID only when an existing organization has been explicitly selected by the operator. No real company values are assumed.

For local bootstrap, explicitly load the protected file:

```powershell
python scripts/dev.py bootstrap --env-file .local/operator.env
```

Replace the example path with your actual operator file. Merely editing an .env file does not load it into the terminal environment. The same option works for backend, mailworker and metaworker; it is rejected for frontend and test/build commands. It accepts only the setting names in .env.example, literal KEY=VALUE lines, full-line comments and optional surrounding quotes. UTF-8 BOM and Windows paths are supported; shell expressions/interpolation are never evaluated. Repeat --env-file to combine files; later files override earlier files and the explicit values override inherited environment. The runner still pins development mode, local database credentials and the existing local root key, and strips bootstrap inputs from other server commands. Password and secret files must use absolute paths. A password validation error means the file content must be valid UTF-8 and 12-256 characters; trailing CR/LF is ignored. Configuration values are never printed.

Open the browser at exactly PUBLIC_ORIGIN, including scheme, host and port. With PUBLIC_ORIGIN=http://localhost:3000, use http://localhost:3000/login; http://127.0.0.1:3000 is a different origin and login POSTs are rejected by CSRF protection. If changing the canonical origin intentionally, update the protected setting and restart the backend with that file. Do not disable Origin/CSRF checks to address a local URL mismatch.

Run python scripts/dev.py bootstrap locally when the inputs are already exported. In production run the compiled bootstrap command with IDENTITY_DATABASE_URL_FILE and IDENTITY_ROOT_KEY_FILE plus the protected inputs above. The identity role, not migrator/admin, executes bootstrap. A singleton and transaction advisory lock prevent a second initialization; reruns do not create or replace an Owner. Remove the temporary password file using the operator's approved secret handling procedure after successful initialization. Sign in, complete email verification and enroll TOTP before privileged production changes. There is no public signup or bootstrap HTTP endpoint.

The local runner creates ignored random database credentials and a 32-byte base64 root-key file. Production must supply independent values via protected files/systemd credentials and verified PostgreSQL TLS. Never use local keys in production. Back up the identity root separately from database backups; test decryption during the later restore drill. Missing keys stop startup. Neither credentials nor action tokens belong in shell history, screenshots, source, URLs outside action fragments, or logs.

Configure SMTP_HOST, SMTP_PORT, SMTP_TLS_MODE (tls/starttls), SMTP_SENDER, SMTP_USERNAME, SMTP_PASSWORD_FILE and SMTP_TIMEOUT. The existing smtpcheck command checks TLS/auth/NOOP only. Run python scripts/dev.py mailworker locally, or deploy the compiled mailworker under a separate non-root process user in production. The worker shares the configured identity database/root-key boundary, never migration/setup credentials. It sends identity mail only, honors shutdown/cancellation and persists retry state. SMTP errors pause the worker for one minute and emit a sanitized operational log; route that log to the operations alert destination when production is provisioned. Keep API liveness independent of SMTP.

Controlled Sprint 1 mailbox evidence is COMPLETE: after the historical SMTP_AUTH failure, the operator corrected the relay credentials and confirmed receipt of all four identity emails; see [Sprint 1 evidence](24-sprint-1-evidence.md). Future deployment checks must use the implemented application/outbox to deliver an Owner verification, employee invitation, password reset and security-change notice to controlled addresses. Confirm actual receipt and successful explicit completion; verify old/superseded links fail and no email exposes passwords, session tokens, TOTP seeds or recovery codes. Record UTC time, application delivery ID, sanitized Message-ID, purpose, SMTP result, receipt confirmation and any bounce in the Sprint 1 evidence record. Store no secret proof or recipient address in the repository. This evidence does not require or authorize Meta messaging.

Account recovery uses saved one-use recovery codes or an already authenticated recent session; there is no unaudited support reset endpoint. If all MFA/recovery paths are lost, a separately reviewed, attributable operations recovery procedure is required before database repair. Production offboarding/restore/recovery drills remain Gate D work.

Hosted GitHub CI is a separate verification track. The historical billing restriction was resolved and Sprint 1/Sprint 2 hosted results are recorded in their evidence files. For later changes, run the workflow for the reviewed commit, inspect actual steps/results and fix repository failures. Neither local success nor an earlier hosted run proves a later commit passed.

The manually opted-in Go test uses -tags=integration,controlledsmtp -run TestControlledSMTPMailbox with the protected SMTP environment and CONTROLLED_SMTP_RECIPIENT. It never runs in ordinary CI. It sends four real identity messages only to the approved address, consumes proofs in disposable accounts and records sanitized relay evidence. Operator confirmation of actual mailbox receipt is still required. Local private *_FILE values must be file paths, not embedded URLs; keep real provider configuration separate from generated development database credentials.

## Sprint 3 Inbox operations

Apply migration 00004 before schema-version-4 binaries. Start separate backend, metaworker and inboxworker commands with the same protected --env-file where applicable. Mailworker stays separate. UI routes are /inbox and permission-protected /pricing. Continue using the exact configured public Origin.

INBOX_LIVE_ACCEPTANCE_ENABLED defaults false. Keep it false until [live prerequisites](28-sprint-3-evidence.md#live-acceptance) are resolved. INBOX_TEST_RECIPIENT_FILE holds one designated number; INBOX_LIVE_ACCEPTANCE_NOT_BEFORE is the approved setup start in RFC3339. Restart API/Inbox worker together after changes. Fresh challenge and authentic inbound must follow that boundary. These variables do not install a callback or establish delivery pricing coverage.

Disable proxy buffering for /api/v1/inbox/events and permit its bounded 24-second stream. REST refresh is required on reconnect. Lost presence cannot lose messages. DISPATCHING recovery never retries a possible provider send. No permanent callback or production deployment is claimed by local testing.
