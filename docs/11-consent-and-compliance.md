# Consent, focused CRM and data lifecycle

WhatsApp's policy requires permission for subsequent business communication, respect for opt-out requests and a clear escalation path when using automation. It also places responsibility for notices and applicable law on the business. Those requirements are distinct from our stricter operational controls below. [Official policy](https://business.whatsapp.com/policy).

The company legal entity, industry, markets, hosting region and applicable retention obligations remain UNKNOWN. This design does not invent Indonesian or other jurisdictional legal requirements from the workspace timezone or “PT” description. Before production, the company's designated policy owner must define approved consent wording, categories, evidence, retention and handling of access/deletion requests.

## Consent state and suppression

Append evidence events; materialize a current state per contact/category. Categories initially MARKETING, UTILITY and AUTHENTICATION plus scoped SERVICE_REQUEST context. These are internal consent scopes; Meta template category is not proof of recipient permission. Each grant records source, event time, collection actor/system, evidence reference, consent text/policy version and optional expiry. Policy defines acceptable proof for each purpose; uploading a CSV column called opt_in is insufficient on its own.

Eligible campaign recipient = valid current target + applicable consent policy + no overriding suppression + frequency eligibility + sender/content capability + other send guards. Every campaign, including utility campaigns, needs documented purpose/evidence under configured policy. A customer initiating a support request permits an appropriate reply context; it does not automatically grant marketing consent. Free service-window availability is not consent.

Suppression is an independent deny layer keyed by normalized/scoped identity digest and optional contact. Scope ALL, MARKETING or a category. Any matched active deny wins over grant, import or audience snapshot. External/off-channel opt-outs enter the same ledger. A trusted STOP-style intent detector can immediately suppress marketing using configured language/keyword/button rules; ambiguous requests create urgent human review and conservatively stop promotional automation. No universal keyword list is asserted as Meta's rule.

Grant after revocation requires new evidence of renewed permission and separate consent.restore authority; lifting ALL suppression requires reason and policy approval. Import never clears suppression because a row says subscribed or is absent from a file. Merge contacts by verified evidence, combine suppressions conservatively and preserve both ledgers. Old events processed late cannot override newer revocation; equal timestamps resolve to deny. If evidence ordering is ambiguous, state is UNKNOWN/denied pending review.

An ALL opt-out stops automated and campaign outreach, removes the person from active outreach lists and retains a minimal keyed digest to prevent reimport. Handling a later user-initiated support request requires an explicit service-request policy; it does not silently restore marketing or clear ALL suppression. Do not send an automatic opt-out acknowledgement if the current rules disallow it. Record opt-out detection latency and any already-dispatched race boundary.

## Focused contact operations

Contact timeline combines permitted conversations, consent events, import provenance, campaign outcomes, notes and Flow responses. Scope-filter each component independently. Keep external IDs and fields useful for WhatsApp work; no sales pipeline, opportunity forecasting or arbitrary CRM object builder.

CSV import first; XLSX DEFERRED until needed because of parser/zip complexity. Background import workflow: private upload and scan, select country assumption explicitly for local numbers, map fields, normalize/validate, preview duplicate/invalid/ambiguous/consent conflicts, approve preview hash, import in chunks with per-row idempotency. Never guess local-number country from employee locale. Return downloadable error report with row number, field and reason, protected like an export. Existing contact updates are limited to selected fields; blank values do not erase consent/evidence. Evidence bulk import is a separate reviewed operation.

Exports require contacts.export plus current record/field scope and purpose. Recheck at generation and download. Quote/escape spreadsheet-formula-leading cells (`=`, `+`, `-`, `@`, control characters) so customer content cannot execute when opened in spreadsheet tools. Phone values remain text. Limit export lifetime and audit requester, filters, row count and download; never log entire exported contents.

## Proposed retention defaults

These are internal starting proposals, subject to company/legal approval and documented holds. Retention applies to replicas, object variants and search projections, not only main tables.

| Data | Proposed baseline | Deletion behavior |
| --- | --- | --- |
| Message/notes/Flow content | 180 days after last relevant interaction | Erase content and search tokens, retain necessary non-content outcome/tombstone |
| Raw webhook bytes | 30 days | Purge encrypted body; retain event/effect digests and processing metadata |
| Media | 30 days after last permitted use, bounded by linked content retention | Delete object, previews, references/URLs; display expired marker |
| Consent evidence | While relied upon plus 24 months after last use/revocation | Policy owner reviews; minimal suppression digests may outlive evidence |
| Suppression/erasure tombstones | While necessary to prevent prohibited reimport/replay; annual review | Keyed HMAC with version, minimum scope/time/reason; explicit policy controls erasure |
| Send idempotency/semantic effect keys | 24 months minimum, and never shorter than enabled replay/import recovery horizon | Compact tombstone; do not retain content solely for dedupe |
| Audit | 24 months | Redacted immutable history, documented holds; member attribution retained |
| Billing evidence | UNKNOWN: company accounting retention decision required | Production financial import blocked until policy assigned |
| API/infra logs | 30 days searchable | Redacted and rotated; aggregated metrics 13 months |
| Import row files/errors | 7 days after completion | Purge private objects and per-row PII |
| Exports | 24 hours | Expiring link plus object cleanup; revoke early on offboarding |
| Tracked link raw events | 30 days | Aggregate without recipient association thereafter if permitted |
| Backups | 35-day rolling recovery window | Encrypted offsite expiry; erasure ledger reapplied on any restore |

Restored databases remain quarantined until tombstones and deletions after the restore point are reapplied from an independently protected erasure ledger. Document backup expiry limitations to the policy owner. Legal holds are explicit resource-scoped records with authorized release; not a blanket excuse to retain every chat forever.

## Optional link tracking

Default off. A campaign can use a first-party redirect with random token, approved immutable destination, optional UTM and optional recipient association. Link token must not expose phone/name/campaign internals. Serve only fixed reviewed HTTP(S) destinations, no arbitrary redirect parameter, and provide expiry/disable handling. Classify requests as bot/unknown/human-possible; link scanners and previews can trigger clicks. Do not report every request as a human or claim precise attribution. Privacy notice/consent policy and retention must cover enabled tracking. Avoid storing full IP/UA where coarse security metadata suffices.

## Acceptance invariants

Opt-out followed by CSV import remains suppressed. Contact merge cannot lose either identity's deny. Audience snapshot cannot freeze away revocation. API/bot routes cannot bypass consent. A retained raw event cannot recreate erased content through replay. Exports and contact timelines cannot leak another team's conversation. Policy changes invalidate queued sends where eligibility changes.
