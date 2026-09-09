# Architecture decisions

Status for all records: **accepted architecture baseline for Sprint 0**, product owner Gate A approval, 2026-09-09. This accepts design decisions, not implementation/production evidence. User-confirmed constraint: existing Meta Cloud API WABA. Sprint 0 alone is authorized; record evidence before revising these decisions. Supersede records explicitly rather than erasing rationale.

| Record | Decision |
| --- | --- |
| [001](001-modular-monolith.md) | Modular monolith and separate execution processes |
| [002](002-valkey.md) | Valkey for disposable coordination |
| [003](003-sse.md) | SSE + REST for realtime |
| [004](004-identifiers.md) | UUIDv7 internal IDs, opaque external identities |
| [005](005-tenant-isolation.md) | Shared schema with composite FKs and RLS |
| [006](006-secret-encryption.md) | Envelope encryption and separate root-key recovery |
| [007](007-campaign-jobs.md) | PostgreSQL durable jobs and uncertain-send handling |
| [008](008-webhook-persistence.md) | Durable raw ingress and child-level deduplication |
| [009](009-media-storage.md) | Private external object storage |
| [010](010-vps-processes.md) | systemd-managed VPS processes |

Revisit on evidence: sustained queue contention, unacceptable restore/downtime, required realtime bidirectional signaling, cross-tenant Meta app sharing, regulated key custody, or materially larger media/search workload. None alone justifies automatic adoption of distributed infrastructure.
