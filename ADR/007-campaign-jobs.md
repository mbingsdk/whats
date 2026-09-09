# ADR 007: durable PostgreSQL jobs

Store jobs, recipient state, outbox and leases in PostgreSQL. Claim bounded batches with [SKIP LOCKED](https://www.postgresql.org/docs/current/sql-select.html) and fencing tokens. Valkey can wake workers but cannot be the only durable queue. This fits initial scale and keeps intent/eligibility/reservation/job creation atomic.

Redis/Valkey Streams could provide queue features but introduces a DB-to-queue recovery boundary; a dedicated broker adds deployment work without measured need. Kafka is not justified. PostgreSQL polling requires careful indexes, bounded claims, vacuuming and queue separation to avoid starving inbox queries.

Worker fencing prevents stale local commits, not remote double-send. DISPATCHING attempts whose outcome is lost become UNCERTAIN and are never automatically retried solely from lease expiry. Every safe retry rechecks consent, price, authority and budgets.

Revisit a broker when measured DB queue contention/throughput warrants it. Keep intent/outbox/idempotency in PostgreSQL even then; a new broker does not solve external exactly-once semantics.

Correction 2026-09-09: ordinary dispatch now takes a shared organization policy barrier and scoped conflicting locks; it no longer serializes all sends with an exclusive organization row. Global policy/topology writers take the exclusive barrier; contact, run and budget changes use their own anchors. Recipient progress is kept out of the shared run row. The exact ordering/absence guarantees and unmeasured contention thresholds are in doc 05; the mixed-workload benchmark in doc 17 gates acceptance. Queue technology and external uncertainty handling are unchanged.
