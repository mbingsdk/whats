# Observability, analytics and notifications

Operational views answer what is blocked, why, since when, whose scope is affected and what action is safe. Every Meta observation carries source and observed_at; every internal metric carries definition, interval and watermark. A stale health reading is not healthy.

## Health center

| Layer | Signals | Operator action |
| --- | --- | --- |
| Meta-reported | WABA/phone/template status, quality, name/review observations, supported capability fields | View source/time, affected senders and official management task |
| Integration | Token checks/expiry, app/WABA subscriptions, version coverage, API error classes, sync freshness | Validate scope/binding, rotate credential, resync, pause affected routes |
| Infrastructure | ACK latency, queue oldest age, worker heartbeat/leases, DLQ, DB/storage capacity, SSE disconnects | Diagnose local service and follow runbook |
| Sending control | Unknown pricing, coverage expiry, reservations, uncertain attempts, approval wait, suppression latency | Block/pause/review without modifying Meta-reported quality |

Last webhook received is an activity signal, not proof of webhook failure; a quiet number can legitimately have no events. Use verified subscription probes and explicit test traffic only with authorization, never automatic customer messages. No fake “green” WABA score from local queue health.

## Telemetry

JSON logs: severity,time,service,build,environment,org_id where known,request_id,trace_id,job_id,attempt,event_id,resource type/opaque ID,error_code,duration. Omit bodies, token headers, phone/email/names and raw URLs with secrets. Redacted raw-payload access lives in restricted UI and is audited. Rotate and bound logs; failure storms cannot fill the database disk.

Metrics: request duration/error rate, durable ACK p95/p99, parse/project lag, oldest runnable age by queue, retries/DLQ, uncertain send count/age, accepted/delivered/read/failed counters, budget reserved/posting mismatches, rate-limit responses, media fetch/scan duration/failure, SSE resets, auth denials and backup age. Avoid contact/message IDs and arbitrary error text as metric labels. Org/phone labels only where bounded and operationally justified.

OpenTelemetry spans join ingress, domain transaction, outbox, worker and outbound adapter. Use span links for async processing rather than pretending one HTTP request lasted a day. Sample normal traffic, retain errors selectively with redaction. Sentry is optional for frontend/backend exceptions after PII scrubbing; no raw payload or session replay capture by default.

## Alerts and runbook entry points

Internal starting thresholds to validate in pilot: ingestion failure sustained 2 min; oldest interactive job >30 s for 5 min; campaign queue age beyond scheduled deadline; any new uncertain send; DLQ above baseline for 5 min; DB disk >80%; backup age >24 h or WAL archive gap beyond RPO; pricing coverage <7 days; privileged auth anomaly; token invalid. Separate urgent correctness/availability incidents from warning digests. Group by asset/error class and suppress duplicate alerts until state changes.

ACK failure -> DB capacity/connectivity and ingress log, return 5xx until durable. Queue lag -> stop campaign pool first, inspect locks and worker saturation. Credential denial -> pause sender, verify assignment/rotation. Uncertainty -> freeze automatic retries, inspect attempt/provider evidence. Billing variance -> retain exposure, reconcile source, do not force balance to zero. These procedures are expanded in doc 16.

## Metric definitions

| Metric | Definition / denominator / limitation |
| --- | --- |
| Accepted | Unique intents with positive API acceptance; separate from Meta sent status |
| Delivered/read | Unique messages with observed corresponding fact; coverage shown; absence of read is UNKNOWN |
| Failure rate | Known failed dispatch/delivery divided by attempted recipients in interval, with uncertain separately reported |
| First response time | First human customer-visible reply acceptance minus episode's first unanswered inbound; bots and notes excluded; delivery-based alternative separately labeled |
| Resolution time | Episode close minus open; reopening starts next episode, no overwritten duration |
| SLA | Versioned business-hours policy elapsed time and due timestamps; paused periods explicit; service window is not SLA |
| Workload | Open assigned episodes per agent/team at watermark; presence is not productivity |
| Campaign delivery/read | Unique observed messages divided by accepted messages, with cohort/time/coverage explicit |
| Campaign reply | Context-based or declared heuristic attribution, stored method/version; avoid double attribution |
| Template performance | Internal message outcomes grouped by frozen template version/category; quality from Meta only if observed |
| Cost | Frozen estimates, reserved exposure, Meta metadata and reconciled money as separate series |
| System latency | Local ingress-to-projection and queue lag, excluding unsupported claims about Meta delivery time |

Use PostgreSQL aggregations/materialized bucket tables initially. Unique event keys prevent retry inflation. Recompute recent buckets for late events; report as_of/watermark and changed data, never silently rewrite a frozen financial report. Imported Meta analytics are labeled separately from our counters and may differ due to timezone, scope, lag and external senders.

## Notification center

Target by resource access and responsibility: assignment -> assignee; mention -> mentioned authorized member; approval request -> currently eligible approvers; budget warning -> billing stewards; template/quality issues -> responsible content/Meta operators; credential/worker/DLQ -> on-call operators. Never broadcast every event to every employee.

Materialize unique `(member,event,type)` records after current permission check. Preferences control noncritical categories, digest interval and channel. Group repeated errors by incident; notification resolves/deep-links to current state. No secret or customer content in push/email previews. In-app operational notifications first; invitation, verification, reset and credential/security mail are mandatory through the identity mail boundary (doc 14), initially SMTP configured in Sprint 0/1. Monitor queue age, retry/terminal failure and SMTP acceptance separately from inbox delivery, without recipient addresses or token-bearing URLs in logs. Outgoing communications to staff are not sent during this documentation phase.
