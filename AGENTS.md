# Repository working rules

Historical owner override (2026-09-09): **Sprint 0 IMPLEMENTATION COMPLETE; Gate A APPROVED; Sprint 1 AUTHORIZED. Hosted CI verification PENDING - EXTERNAL BLOCKER.** Run 34356265759 was attempted and failed before repository steps because GitHub reported an account billing lock. Accepted local evidence closes implementation, not hosted verification. CI requirements remain intact; rerun hosted CI when available, record the real result and fix any repository failures. Gates B/C/D remain CLOSED.

Current verification (2026-09-10): **Sprint 0 IMPLEMENTATION COMPLETE; Gate A APPROVED; Sprint 1 implementation COMPLETE, acceptance OPEN.** GitHub run 34356265759 attempt 3 actually executed and passed the Sprint 0 baseline at commit 993c67e93fe5945ec820efe67d9d3abf3df8b1b3. Its original attempt 1 failed before steps due to the account billing restriction. Sprint 1 hosted verification is PENDING for these unpublished workspace changes; do not treat the Sprint 0 pass as Sprint 1 evidence. Controlled SMTP was ATTEMPTED and rejected authentication (SMTP_AUTH), so mailbox receipt remains PENDING. Gates B/C/D remain CLOSED.

Read README.md, PLANS.md, the capability matrix and relevant ADRs before changes. Preserve established decisions; revise their rationale and dependent contracts together when changing them.

Meta claims require current official evidence. Record URL, inspection date, evidence quality, Graph/account restrictions and unresolved checks. Official Postman samples can be stale. Never convert an unavailable reference into a verified claim. Classify contract evidence as VERIFIED_CURRENT_CONTRACT, DOCUMENTED_OFFICIAL_SURFACE, UNVERIFIED, INTERNAL_PLATFORM_FEATURE or positively established NOT_PUBLICLY_EXPOSED. Record ACCOUNT_VERIFICATION_REQUIRED separately from evidence and OPTIONAL_ACCOUNT_DEPENDENT separately from DEFERRED delivery scope. Historical/indexed official requests prove surface only; never enable an adapter without a current pinned contract and the relevant account verification.

All business records, jobs, events, keys, search results and storage access require an organization boundary. Use permissions with resource scope, never role-name conditionals. All outbound sends go through the shared send gate. Do not retry ambiguous external sends automatically or claim exactly-once network delivery.

Keep monetary values decimal and currency-qualified. Freeze pricing and approval evidence. Never treat a webhook's category/billable flag as an invoice amount. Consent revocation survives imports and campaign snapshots. Do not log tokens, message bodies or unredacted raw payloads.

Write specific decisions, invariants, tradeoffs and failure behavior. Mark UNKNOWN, NEEDS VERIFICATION and DEFERRED explicitly. Avoid placeholder content, marketing claims and fake performance results. Documentation links must resolve locally. Changes to a state machine require matching database, API, worker and test updates.

Before concluding a change, check cross-document consistency and update the review record. User instructions take precedence over these repository rules.

Sprint 0 checks: python scripts/dev.py up, python scripts/dev.py check; go mod verify and pinned govulncheck plus npm audit for dependency review. Never silently skip explicit integration tests. Keep migration SQL/checksum immutable after publication. Runtime credentials must never use the migration/authorizer/setup-administrator role. Only internal server-authenticated user IDs may enter InOrganization; never add a user-ID or organization-header authorization shortcut.
