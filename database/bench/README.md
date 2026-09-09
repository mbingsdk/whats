# Future lock measurement harness

No campaign/dispatch benchmark exists in Sprint 0, so no dispatch capacity is reported. [Doc 17](../../docs/17-testing-strategy.md#mixed-dispatch-contention-benchmark) remains the accepted test protocol; [doc 05](../../docs/05-database-design.md#dispatch-locking-scope-absence-checks-and-contention) defines proposed thresholds and redesign triggers.

The observation.sql query is executable against PostgreSQL using an isolated operator/test-administrator connection. It deliberately omits query text/parameters, users and customer identifiers. Runtime roles receive no monitoring privilege. Capture timestamps, wait event/type, blocked PIDs and transaction/query elapsed duration. These durations are not exact lock-wait or lock-hold measurements.

Sprint 3/5 instrumentation must use monotonic times immediately before and after each lock SQL call and before commit/rollback. Record per-tier wait/hold histograms and final-authorization latency using low-cardinality labels; never message/contact IDs in metrics. Separately record offered/admitted/completed work, queued job oldest age, ingress ACK latency and deny command arrival-to-commit. Use test-only trace correlation to prove commit ordering.

Run agents, webhook ingestion, campaign recipients, opt-outs and budget changes concurrently with representative skew and a shared funded org budget. Freeze hardware, DB/tool versions, fixture dimensions, offered rates and accepted recovery thresholds before starting. Keep synthetic/real account evidence distinct. Archive raw histograms and failed assertions, including starvation/timeouts; never replace them with guessed throughput. The SQL sampler is supplementary evidence, not proof the dispatch algorithm is correct.
