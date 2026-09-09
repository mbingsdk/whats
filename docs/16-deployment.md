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
