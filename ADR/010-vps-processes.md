# ADR 010: systemd processes on one VPS

Run Go API/workers and Next.js as independent systemd services behind Nginx, with PostgreSQL/Valkey managed by supported Linux packages. CI produces pinned immutable release artifacts; production does not install/build dependencies on deploy.

Containers are a reasonable alternative if the operations team already supports them, but they do not make a single host highly available. systemd gives direct resource limits, logs, restart and recovery with fewer new conventions for this VPS baseline. Avoid both a container supervisor and duplicate host supervision for the same service.

Failure recovery uses offsite base backups/WAL, object manifests and separate key custody. Rollback selects schema-compatible previous artifacts; destructive down migrations are not the default. RPO/RTO are acceptance targets requiring timed restore evidence.

Revisit host separation/managed database when availability or measured load requires it. Kubernetes is not a prerequisite for recoverable deployment.
