# Meta capability matrix

Research attempted **2026-09-08 for every row**; public pricing/policy revisited **2026-09-09**. No live account verification. Links resolve to [source evidence](21-research-register.md), with direct official references there. M06/M07 are official Meta Postman evidence, not third-party BSP material. Old samples prove documented surface only.

Contract evidence and applicability are separate:
- **VERIFIED_CURRENT_CONTRACT**: inspected current official specification for the pinned Graph version, including exact request/response, permissions and constraints. No operation has reached this state here.
- **DOCUMENTED_OFFICIAL_SURFACE**: retrieved official description, historical request or indexed example establishes a surface only; current fields, limits and version compatibility remain unverified.
- **UNVERIFIED**: insufficient inspected evidence. **NOT_PUBLICLY_EXPOSED** requires positive evidence of absence; none is asserted here.
- **INTERNAL_PLATFORM_FEATURE**: planned local responsibility, not completed implementation.
- **OPTIONAL_ACCOUNT_DEPENDENT**: conditional applicability; **DEFERRED**: delivery scope, neither is evidence quality.

**ACCOUNT_VERIFICATION_REQUIRED applies to every external operation in all tables below and is still pending.** Each adapter needs both VERIFIED_CURRENT_CONTRACT and a recorded successful probe for its target app/WABA/phone, permissions, region and supported operation before enablement. Public policy/pricing prose alone cannot satisfy an API contract gate. Table restrictions identify additional checks. A historical request never enables an adapter.

Permission shorthand: WM = whatsapp_business_messaging; WA = whatsapp_business_management; BM = business_management for relevant business-portfolio queries. Possessing a scope is insufficient without app access level and assigned assets. A question mark means the exact permission is unverified. Webhook names marked ? are candidates, not a subscription contract. `none` means no webhook dependency in our plan, not proof Meta emits none.

## Accounts and assets

| Feature / API | Contract evidence / applicability | Permission; webhook | Account/region restrictions | Implementation and limits | Evidence |
| --- | --- | --- | --- | --- | --- |
| Existing Meta app connection / token use | DOCUMENTED_OFFICIAL_SURFACE | WA/WM; none | Company asset assignments; app review/access level must be checked | Write-only credentials, validation and explicit binding. Expiry/revocation observable; no promise of a permanent token | M06 |
| Owned WABA discovery / business owned_whatsapp_business_accounts | DOCUMENTED_OFFICIAL_SURFACE | BM + WA asset access; none | Only accessible portfolios | Paginated sync into staging, administrator binds to org | M06 |
| Shared WABA discovery / client_whatsapp_business_accounts | DOCUMENTED_OFFICIAL_SURFACE | BM + WA asset access; none | Sharing relationship required | Record OWNED/SHARED, do not infer ownership from visibility | M06 |
| Phone list, verified name, quality / WABA phone_numbers | DOCUMENTED_OFFICIAL_SURFACE | WA; phone_number_name_update, phone_number_quality_update | Fields depend on account/version | Preserve reported raw values, nullable unknowns and freshness | M06, M25 |
| Business profile read/update | DOCUMENTED_OFFICIAL_SURFACE | WA; none | Verified editable fields only | Controlled form, async mutation/sync. Display-name approval is separate | M18 |
| Registration / phone register | DOCUMENTED_OFFICIAL_SURFACE | WM; none | Existing registration and verification context matters | DEFERRED: existing Cloud API WABA does not need blanket re-registration | M06 |
| App WABA subscription / subscribed_apps | DOCUMENTED_OFFICIAL_SURFACE | WA; messages and enabled fields | Correct app/WABA mapping required | Inventory existing consumers before mutation; configure app fields as well as WABA subscription | M06, M24 |
| Account/review/name/quality changes | DOCUMENTED_OFFICIAL_SURFACE | WA; account_update, account_review_update, phone_number_name_update, phone_number_quality_update | Selected fields/access level must be verified | Events trigger targeted resync; old tier constants not hardcoded | M25 |
| Credential checks/API version diagnostics | INTERNAL_PLATFORM_FEATURE | Underlying scopes; none | Valid debug context | Redacted checks and version registry; latest Graph version NEEDS VERIFICATION | M06, M27 |
| Display-name mutation, verification appeals, payment methods | UNVERIFIED | UNKNOWN; UNKNOWN | Account/UI workflow dependent | Link to Meta Manager; no internal write contract until exact API verified | M26 |
| Number throughput, portfolio limits, pair throttles | UNVERIFIED | Underlying scopes; quality field where verified | Dynamic account restrictions | Capability observations and conservative configurable limits, never fabricated Meta quotas | M26 |
| QR/deep-link management / message_qrdls | DOCUMENTED_OFFICIAL_SURFACE | WA candidate; none | Current scope/limits unverified | Optional acquisition tool; QR scan is not consent evidence | M20 |
| Block/list/unblock / block_users | DOCUMENTED_OFFICIAL_SURFACE | WM candidate; UNKNOWN | Exact limits and eligibility unverified | Adapter gated; local suppression is independent and cannot be cleared by unblock | M19 |

## Messaging and customer interaction

All outbound variants below use the common internal send gate. Approved templates are not a consent or budget bypass. `messages` is the webhook subscription field; nested types are not separate fields.

| Feature / API | Contract evidence / applicability | Permission; webhook | Restrictions | Implementation and limits | Evidence |
| --- | --- | --- | --- | --- | --- |
| Text, image, video, audio, document / phone messages | DOCUMENTED_OFFICIAL_SURFACE | WM; messages | Free-form permission governed by service window; MIME/size limits reverify | Typed content + raw pointer; media quarantine and storage | M07, M08 |
| Sticker | DOCUMENTED_OFFICIAL_SURFACE | WM; messages | Static/animated constraints NEEDS VERIFICATION | Typed media; reject unverified send formats | M13 |
| Contacts | DOCUMENTED_OFFICIAL_SURFACE | WM; messages | Exact schema needs test; documentation prose inconsistent | Internal contact-card type separate from CRM contact | M11 |
| Location | DOCUMENTED_OFFICIAL_SURFACE | WM; messages | Live-location support not established | Coordinate bounds, safe map link; no live tracking promise | M12 |
| Reply context and reaction | DOCUMENTED_OFFICIAL_SURFACE | WM; messages | Target/age restrictions NEEDS VERIFICATION | Same-thread context checks, unresolved targets preserved | M07 |
| Interactive list/reply buttons | DOCUMENTED_OFFICIAL_SURFACE | WM; messages | Current component limits reverify | Versioned validators; selected IDs stored distinctly from labels | M14 |
| Approved template send | DOCUMENTED_OFFICIAL_SURFACE | WM; messages | Correct category/language/variables and status | Freeze version; fresh status and pricing checks | M06, M15 |
| Flow interactive/template interaction | DOCUMENTED_OFFICIAL_SURFACE | WM plus WA for management; messages | Client support and template/Flow versions | Capability flag per sender, instance-token correlation | M22, M23 |
| Orders/catalog/product interactions | DOCUMENTED_OFFICIAL_SURFACE; OPTIONAL_ACCOUNT_DEPENDENT | WM, WA; messages | Catalog linkage, commerce policy; detailed scopes UNKNOWN | Preserve inbound order subtype; commerce send adapter gated | M08, M31 |
| Sent/delivered/read/failed observation | DOCUMENTED_OFFICIAL_SURFACE | WM; messages | Read receipts may be absent; exact error mapping reverify | Immutable facts, tolerate reorder; no fabricated read percentage | M09 |
| Mark inbound as read | DOCUMENTED_OFFICIAL_SURFACE | WM; messages for source ID | Separate from internal per-agent unread state | Explicit configurable team policy, idempotent effect | M07 |
| Customer service window | DOCUMENTED_OFFICIAL_SURFACE | WM; messages | Based on last customer message; 24-hour reply rule | Backend window projection; unknown state blocks free-form | M01, M02 |
| Free-entry pricing window | UNVERIFIED; OPTIONAL_ACCOUNT_DEPENDENT | WM; messages/referral details ? | Entry origin, timing and exact eligibility need developer verification | Track separately from service window, never grant free-form authority from pricing alone | M02, M03 |
| Meta-facing typing indicator | UNVERIFIED | WM ?; UNKNOWN | Current API conditions unknown | Disabled until official request verified; internal agent typing is separate | M26 |
| Customer online/typing visibility | UNVERIFIED | UNKNOWN; UNKNOWN | No current evidence inspected | Do not promise customer presence; employee presence is internal | M26 |
| Message edit/recall/delete-for-everyone; historical chat fetch | UNVERIFIED | UNKNOWN; UNKNOWN | Do not infer from old on-premises status fields | No product API; local retention deletion is not WhatsApp recall | M26, M29 |
| Unknown/future payload types | INTERNAL_PLATFORM_FEATURE | Source webhook authorization | May lack recognizable IDs/schema | Store safely, classify unknown, disable automatic external effects | M08 |
| Username/BSUID identity/routing | UNVERIFIED | UNKNOWN; messages shape UNKNOWN | Rollout and identity scope unresolved | Opaque scoped IDs and nullable phone prevent schema lock-in; no guessed send fields | M30 |

## Templates, Flows, analytics and advanced products

| Feature / API | Contract evidence / applicability | Permission; webhook | Restrictions | Implementation and limits | Evidence |
| --- | --- | --- | --- | --- | --- |
| Template sync/create/edit/delete | DOCUMENTED_OFFICIAL_SURFACE | WA; message_template_status_update | Exact mutation quotas, name reuse and editable fields unresolved | Sync/read first; mutations capability-gated; never promise arbitrary edits | M15, M16, M17 |
| Template status/rejection | DOCUMENTED_OFFICIAL_SURFACE | WA; message_template_status_update | Rejection reason may be absent | Immutable observed history; refresh on events | M17 |
| Template quality/performance/category changes | UNVERIFIED | WA; message_template_quality_update ?, template_category_update ? | Current fields/analytics eligibility unknown | Store observations only; internal usage metrics separately labeled | M15, M21, M26 |
| Flow list/create/JSON validation/publish/preview | DOCUMENTED_OFFICIAL_SURFACE | WA for management, WM for send; messages | Exact lifecycle edits and schema versions reverify | Internal versioning and validation UI; publish only with current evidence | M22, M23 |
| Flow deprecate/delete | DOCUMENTED_OFFICIAL_SURFACE | WA; none | Allowed source states NEEDS VERIFICATION | Disabled until verified; official historical sample has command mismatch | M23 |
| Flow data exchange endpoint / encryption | UNVERIFIED; OPTIONAL_ACCOUNT_DEPENDENT | WA; synchronous endpoint distinct from webhook | Encryption, signatures, deadlines and keys NEEDS VERIFICATION | DEFERRED; separate threat model and endpoint specification required | M23, M26 |
| WABA analytics | DOCUMENTED_OFFICIAL_SURFACE | WA; none/polling | Supported dimensions/date ranges must be tested | Source-labeled imported aggregates, no invoice claims | M21 |
| Pricing metadata from statuses; exact cost/invoice API | UNVERIFIED | WM/WA as applicable; messages | Current per-message billing fields/amounts unverified | Preserve metadata; amount nullable; manual official statement reconciliation supported | M03, M09 |
| Calling | UNVERIFIED; OPTIONAL_ACCOUNT_DEPENDENT | UNKNOWN; UNKNOWN | Country/account, user call permission, pricing unknown | DEFERRED and disabled; no call UI or recording promise | M28 |
| Embedded Signup | DOCUMENTED_OFFICIAL_SURFACE; OPTIONAL_ACCOUNT_DEPENDENT | BM/WA and app review per selected flow; account fields | Provider/app access requirements | DEFERRED; existing company assets can use controlled connection | M24 |
| Business app coexistence | UNVERIFIED; OPTIONAL_ACCOUNT_DEPENDENT | UNKNOWN; history/echo fields UNKNOWN | Existing company uses Cloud API; no demonstrated need | DEFERRED; require echo/history ingestion and external-spend accounting before enablement | M29 |
| Payments, marketing-specific APIs, groups | UNVERIFIED | UNKNOWN; UNKNOWN | Region/account and product-specific terms | DEFERRED; do not treat collection headings as complete support evidence | M26 |

## Features built by WABA Control

RBAC, team assignment, presence, collision control, notes, mentions, unread state, SLA clocks, audiences, campaign lifecycle, approvals, budgets, estimates, internal frequency limits, consent evidence, imports, notification targeting, automation, API keys, outgoing webhooks and audit are all **INTERNAL_PLATFORM_FEATURE**. Their limits and scores are our policies. None is represented as a Meta-provided guarantee.

Link-click tracking is an optional internal redirect with privacy controls. Meta-reported analytics, if exposed and verified, has separate provenance. Neither link requests nor read receipts prove a person read or clicked.

## Verification work packages

V1 existing app/WABA access and subscription inventory; V2 pinned Graph version/security/payload fixtures; V3 exact message/media constraints and eligible markets; V4 official effective-dated pricing and billing evidence; V5 template mutations/status/category/quality; V6 Flow lifecycle/static completion; V7 optional products. Each package needs source evidence, a named reviewer, account result and sanitized fixture. V1–V4 block the relevant integration/send gates. V7 does not block core implementation.

## Sprint 0 implementation status

Only engineering health/config/database/frontend foundations are implemented. No Meta adapter exists or is enabled. [Sprint 0 research](21-research-register.md#sprint-0-contract-verification-attempt-2026-09-09) re-inspected official references; developer pages remained blocked. Graph version/removal timeline remain UNKNOWN; official Postman and historical SDK evidence stays DOCUMENTED_OFFICIAL_SURFACE, with account verification pending. No capability classification was promoted solely to make Sprint 0 appear complete.
