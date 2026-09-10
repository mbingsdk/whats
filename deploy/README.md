# Deployment foundation, not a deployed service

Sprint 0 runs PostgreSQL in a local-only Compose container and the Go/Next processes on the host. No real VPS, hostname, certificate or production credential is configured.

Later deployment follows [doc 16](../docs/16-deployment.md): immutable release under /opt/waba/releases/<build>; API executable and reviewed migration files; Next standalone output with public and .next/static assets copied into its output layout. The runtime secret directory /etc/waba must be owned by operations and unavailable to frontend builds. Separate API and migration credentials; load secret paths through systemd credentials or protected files, never command-line passwords.

Nginx routes /api/, /healthz and /readyz to Go on loopback :8080, the frontend to loopback :3000. /events/ and /webhooks/ are future routes, disabled until implemented. Only the reverse proxy terminates public TLS. Restrict readiness diagnostics to monitoring where appropriate. Future SSE disables buffering; Sprint 1 adds the compiled mailworker process for identity mail only.

Define systemd units at actual host provisioning: non-root service users, working directory at the selected release, Restart=on-failure, bounded restart rate, NoNewPrivileges, filesystem protections and termination grace longer than SHUTDOWN_TIMEOUT. Do not put an external Meta/SMTP probe in process liveness. Database/schema incompatibility makes readiness return 503; it does not restart-loop a process.

Deployment order is backup evidence -> migration with waba_migrator -> API/frontend -> liveness/readiness -> observed rollout. No auto-migration on API startup. Forward-compatible schema versions allow binary rollback; no destructive down migration. Later worker shutdown must preserve send uncertainty.

Backup/restore remains [doc 16's runbook](../docs/16-deployment.md#backup-and-restore-runbook). Local Docker volume persistence is not a backup. An offsite WAL/base-backup and recovery-key drill is still required for RPO/RTO evidence; no deployment or restore result is claimed here.

Sprint 1 deploys the identity pool/root key alongside the limited runtime pool. Keep frontend/build processes free of both database URLs and root keys; keep migration/setup credentials out of API and worker environments. Readiness covers runtime schema and the required identity connection. The [identity runbook](../docs/16-deployment.md#sprint-1-identity-operations) defines bootstrap, mail worker, key recovery and controlled mailbox evidence.
