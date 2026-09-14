# Meta capability matrix

Owner clarification (2026-09-15): **Sprints 0/1/2 COMPLETE; Gates A/B/C APPROVED; Gate D CLOSED; Sprint 3 AUTHORIZED. COMPANY META BILLING CURRENCY UNKNOWN; PAID-SEND AUTHORITY CLOSED.** Currency must not be inferred from timezone, business/phone/recipient country, available rate cards or locale. Only a currently reviewed, provably zero-cost policy can permit the single operator-triggered controlled TEXT live reply after all recipient/callback/security prerequisites. Paid/template live sends, outbound media, campaigns and Sprint 4 remain unauthorized.

Earlier dated status/review entries below are historical and superseded by this owner clarification where they describe Gate C or Sprint 3 authorization.

Research attempted **2026-09-08 for every row**; public pricing/policy revisited **2026-09-09**. Initial inspections had no live account verification; the 2026-09-11 read-only result below supersedes that status for tested operations. Links resolve to [source evidence](21-research-register.md), with direct official references there. M06/M07 are official Meta Postman evidence, not third-party BSP material. Old samples prove documented surface only.

Contract evidence and applicability are separate:
- **VERIFIED_CURRENT_CONTRACT**: inspected current official specification for the pinned Graph version, including exact request/response, permissions and constraints. The initial September 8 table had no such operations; the bounded September 14 Gate C rows below now have current first-party contract evidence.
- **DOCUMENTED_OFFICIAL_SURFACE**: retrieved official description, historical request or indexed example establishes a surface only; current fields, limits and version compatibility remain unverified.
- **UNVERIFIED**: insufficient inspected evidence. **NOT_PUBLICLY_EXPOSED** requires positive evidence of absence; none is asserted here.
- **INTERNAL_PLATFORM_FEATURE**: planned local responsibility, not completed implementation.
- **OPTIONAL_ACCOUNT_DEPENDENT**: conditional applicability; **DEFERRED**: delivery scope, neither is evidence quality.

**ACCOUNT_VERIFICATION_REQUIRED applies separately to each external operation. September 11 asset GETs and September 14 template GETs establish bounded account reads; Sprint 2 acceptance establishes genuine inbound webhook delivery. Outbound remains unprobed.** Production enablement requires the bounded current contract and successful applicable account acceptance. Gate C performs no outbound probe; any first controlled send requires separate authorization and all financial/recipient guards. Public policy/pricing prose alone cannot satisfy an API contract gate. Table restrictions identify additional checks. A historical request never enables an adapter.

Permission shorthand: WM = whatsapp_business_messaging; WA = whatsapp_business_management; BM = business_management for relevant business-portfolio queries. Possessing a scope is insufficient without app access level and assigned assets. A question mark means the exact permission is unverified. Webhook names marked ? are candidates, not a subscription contract. `none` means no webhook dependency in our plan, not proof Meta emits none.

## Accounts and assets

| Feature / API | Contract evidence / applicability | Permission; webhook | Account/region restrictions | Implementation and limits | Evidence |
| --- | --- | --- | --- | --- | --- |
| Existing Meta app connection / token use | DOCUMENTED_OFFICIAL_SURFACE | WA/WM; none | Company asset assignments; app review/access level must be checked | Write-only credentials, validation and explicit binding. Expiry/revocation observable; no promise of a permanent token | M06 |
| Owned WABA discovery / business owned_whatsapp_business_accounts | DOCUMENTED_OFFICIAL_SURFACE | BM for portfolio query; exact WA requirement NEEDS VERIFICATION; none | Only accessible portfolios | Paginated sync into staging, administrator binds to org | M06 |
| Shared WABA discovery / client_whatsapp_business_accounts | DOCUMENTED_OFFICIAL_SURFACE | BM for portfolio query; exact WA requirement NEEDS VERIFICATION; none | Sharing relationship required | Record OWNED/SHARED, do not infer ownership from visibility | M06 |
| Phone list, verified name, quality / WABA phone_numbers | DOCUMENTED_OFFICIAL_SURFACE | WA; phone_number_name_update, phone_number_quality_update | Fields depend on account/version | Preserve reported raw values, nullable unknowns and freshness | M06, M25 |
| Business profile read/update | DOCUMENTED_OFFICIAL_SURFACE | Exact read/update scope NEEDS VERIFICATION; none | Verified editable fields only | Controlled form, async mutation/sync. Display-name approval is separate | M18 |
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
| Text, image, video, audio, document / phone messages | VERIFIED_CURRENT_CONTRACT, bounded guides | WM; messages | Service-window permission; current fields/MIME/limits in doc 27; outbound ACCOUNT_VERIFICATION_REQUIRED | Typed payloads; media IDs/links require tenant and SSRF controls; NOT IMPLEMENTED | M66 |
| Sticker | VERIFIED_CURRENT_CONTRACT, bounded guide | WM; messages | WebP static 100 KB / animated 500 KB; account path untested | No unverified optional format; NOT IMPLEMENTED | M66 |
| Contacts | VERIFIED_CURRENT_CONTRACT, bounded guide | WM; messages | formatted_name required; current guide maximum 257; outbound account untested | Contact-card payload separate from CRM | M66 |
| Location | VERIFIED_CURRENT_CONTRACT, bounded guide | WM; messages | Required latitude/longitude; optional name/address; service window | No live-location claim | M66 |
| Reply context and reaction | Reaction VERIFIED_CURRENT_CONTRACT; reply-context extensions DOCUMENTED_OFFICIAL_SURFACE | WM; messages | Reaction same-thread received target, at most 30 days; only sent status; account untested | Unverified extensions/removal remain disabled | M66 |
| Interactive list/reply buttons | VERIFIED_CURRENT_CONTRACT, bounded guides | WM; messages | Exact component/body limits in doc 27; account untested | List/reply/CTA evidence does not enable Flows/catalog | M66 |
| Approved template send | VERIFIED_CURRENT_CONTRACT for bounded simple template fields | WM; messages | Current Meta category/status/language/parameters and price; account send untested | Safe utility candidate strategy only; no mutation/send in Gate C | M65, M66, M68 |
| Flow interactive/template interaction | DOCUMENTED_OFFICIAL_SURFACE | WM plus WA for management; messages | Client support and template/Flow versions | Capability flag per sender, instance-token correlation | M22, M23 |
| Orders/catalog/product interactions | DOCUMENTED_OFFICIAL_SURFACE; OPTIONAL_ACCOUNT_DEPENDENT | WM, WA; messages | Catalog linkage, commerce policy; detailed scopes UNKNOWN | Preserve inbound order subtype; commerce send adapter gated | M08, M31 |
| Sent/delivered/read/failed observation | VERIFIED_CURRENT_CONTRACT, bounded status fields | WM; messages | May skip callbacks/reorder; real outbound lifecycle untested | Facts separate from estimates; no invented amount/read | M67 |
| Mark inbound as read | DOCUMENTED_OFFICIAL_SURFACE | WM; messages for source ID | Separate from internal per-agent unread state | Explicit configurable team policy, idempotent effect | M07 |
| Customer service window | VERIFIED_CURRENT_CONTRACT, bounded current rule | WM; messages | User messages or calls reset 24 hours; calling DEFERRED | Initial authenticated text basis; UNKNOWN blocks free-form; price independent | M62, M66, M67 |
| Free-entry pricing window | VERIFIED_CURRENT_CONTRACT; OPTIONAL_ACCOUNT_DEPENDENT | WM; qualifying entry evidence | Qualifying mobile ad/CTA and response within 24h; 72h from response; account qualification unverified | Does not extend free-form permission | M62 |
| Meta-facing typing indicator | UNVERIFIED | WM ?; UNKNOWN | Current API conditions unknown | Disabled until official request verified; internal agent typing is separate | M26 |
| Customer online/typing visibility | UNVERIFIED | UNKNOWN; UNKNOWN | No current evidence inspected | Do not promise customer presence; employee presence is internal | M26 |
| Message edit/recall/delete-for-everyone; historical chat fetch | UNVERIFIED | UNKNOWN; UNKNOWN | Do not infer from old on-premises status fields | No product API; local retention deletion is not WhatsApp recall | M26, M29 |
| Unknown/future payload types | INTERNAL_PLATFORM_FEATURE | Source webhook authorization | May lack recognizable IDs/schema | Store safely, classify unknown, disable automatic external effects | M08 |
| Username/BSUID identity/routing | UNVERIFIED | UNKNOWN; messages shape UNKNOWN | Rollout and identity scope unresolved | Opaque scoped IDs and nullable phone prevent schema lock-in; no guessed send fields | M30 |

## Templates, Flows, analytics and advanced products

| Feature / API | Contract evidence / applicability | Permission; webhook | Restrictions | Implementation and limits | Evidence |
| --- | --- | --- | --- | --- | --- |
| Template sync/create/edit/delete | Read DOCUMENTED_OFFICIAL_SURFACE plus VERIFIED account GET; mutations DOCUMENTED_OFFICIAL_SURFACE only | WA; message_template_status_update | Five observed approved templates; mutation constraints unresolved | Read field inventory in doc 27; no mutation authorized | M15, M16, M17, M65 |
| Template status/rejection | Observed GET fields VERIFIED account evidence; status webhook contract remains separately gated | WA; message_template_status_update | APPROVED/rejected_reason/updated time captured; metadata may be absent | Fresh selected-template GET before dispatch | M65, M68 |
| Template quality/performance/category changes | Category-update VERIFIED_CURRENT_CONTRACT; quality GET observed; performance UNVERIFIED | WA; template_category_update; quality subscription unverified | Quality UNKNOWN on all five; no analytics promotion | Invalidate authorization on changed category/status/components | M65, M68 |
| Flow list/create/JSON validation/publish/preview | DOCUMENTED_OFFICIAL_SURFACE | WA for management, WM for send; messages | Exact lifecycle edits and schema versions reverify | Internal versioning and validation UI; publish only with current evidence | M22, M23 |
| Flow deprecate/delete | DOCUMENTED_OFFICIAL_SURFACE | WA; none | Allowed source states NEEDS VERIFICATION | Disabled until verified; official historical sample has command mismatch | M23 |
| Flow data exchange endpoint / encryption | UNVERIFIED; OPTIONAL_ACCOUNT_DEPENDENT | WA; synchronous endpoint distinct from webhook | Encryption, signatures, deadlines and keys NEEDS VERIFICATION | DEFERRED; separate threat model and endpoint specification required | M23, M26 |
| WABA analytics | DOCUMENTED_OFFICIAL_SURFACE | WA; none/polling | Supported dimensions/date ranges must be tested | Source-labeled imported aggregates, no invoice claims | M21 |
| Pricing metadata from statuses; exact cost/invoice API | Metadata VERIFIED_CURRENT_CONTRACT; exact invoice API UNVERIFIED | WM/WA as applicable; messages | Optional model/type/category/billable, no monetary amount established | Keep estimate/reservation/reconciled invoice separate | M62, M67 |
| Calling | UNVERIFIED; OPTIONAL_ACCOUNT_DEPENDENT | UNKNOWN; UNKNOWN | Country/account, user call permission, pricing unknown | DEFERRED and disabled; no call UI or recording promise | M28 |
| Embedded Signup | DOCUMENTED_OFFICIAL_SURFACE; OPTIONAL_ACCOUNT_DEPENDENT | BM/WA and app review per selected flow; account fields | Provider/app access requirements | DEFERRED; existing company assets can use controlled connection | M24 |
| Business app coexistence | UNVERIFIED; OPTIONAL_ACCOUNT_DEPENDENT | UNKNOWN; history/echo fields UNKNOWN | Existing company uses Cloud API; no demonstrated need | DEFERRED; require echo/history ingestion and external-spend accounting before enablement | M29 |
| Payments, marketing-specific APIs, groups | UNVERIFIED | UNKNOWN; UNKNOWN | Region/account and product-specific terms | DEFERRED; do not treat collection headings as complete support evidence | M26 |

## Features built by WABA Control

RBAC, team assignment, presence, collision control, notes, mentions, unread state, SLA clocks, audiences, campaign lifecycle, approvals, budgets, estimates, internal frequency limits, consent evidence, imports, notification targeting, automation, API keys, outgoing webhooks and audit are all **INTERNAL_PLATFORM_FEATURE**. Their limits and scores are our policies. None is represented as a Meta-provided guarantee.

Link-click tracking is an optional internal redirect with privacy controls. Meta-reported analytics, if exposed and verified, has separate provenance. Neither link requests nor read receipts prove a person read or clicked.

## Verification work packages

V1 existing app/WABA access and subscription inventory; V2 pinned Graph version/security/payload fixtures; V3 exact message/media constraints and eligible markets; V4 official effective-dated pricing and billing evidence; V5 template mutations/status/category/quality; V6 Flow lifecycle/static completion; V7 optional products. Each package needs source evidence, a named reviewer, account result and sanitized fixture. V1–V4 block the relevant integration/send gates. V7 does not block core implementation.

## Historical Sprint 0 implementation status

Only engineering health/config/database/frontend foundations are implemented. No Meta adapter exists or is enabled. [Sprint 0 research](21-research-register.md#sprint-0-contract-verification-attempt-2026-09-09) re-inspected official references; developer pages remained blocked. Graph version/removal timeline remain UNKNOWN; official Postman and historical SDK evidence stays DOCUMENTED_OFFICIAL_SURFACE, with account verification pending. No capability classification was promoted solely to make Sprint 0 appear complete.

## Gate B and Sprint 2 dependency matrix, 2026-09-10

This table refines only the ingestion/existing-assets scope. [Gate B evidence](25-gate-b-evidence.md) is the decision record; M32-M48 are in the [research register](21-research-register.md#gate-b-contract-inspection-2026-09-10). This 2026-09-10 snapshot preceded account access; the latest table below supersedes its account column for successfully tested GETs.

Evidence axes remain separate: VERIFIED_CURRENT_CONTRACT is a selected-version contract result; VERIFIED_ON_COMPANY_ACCOUNT is a successful scoped account observation. ACCOUNT_DEPENDENT describes applicability (OPTIONAL_ACCOUNT_DEPENDENT for optional features), NEEDS_VERIFICATION describes missing checks, and DEFERRED describes delivery scope. None substitutes for another.

| Sprint 2 dependency | Contract evidence | Account / applicability | Delivery decision and unresolved check |
| --- | --- | --- | --- |
| Explicit Graph version/lifecycle | UNVERIFIED (M33/M49); operator candidate v26.0, untested | ACCOUNT_VERIFICATION_REQUIRED | Blocked; no default/fallback version |
| Company System User token | DOCUMENTED_OFFICIAL_SURFACE (M32/M34) | Token type/scopes/expiry/assignments unknown | Strategy chosen; no credential provisioned or production lifetime assumed |
| WA/WM and operation permissions | DOCUMENTED_OFFICIAL_SURFACE (M32/M34/M41) | Grants/access levels untested | Exact selected-version operation mapping remains NEEDS_VERIFICATION |
| BM portfolio queries | DOCUMENTED_OFFICIAL_SURFACE (M32/M37) | ACCOUNT_DEPENDENT on discovery workflow | No broad runtime BM default; confirm whether complete operator inventory avoids portfolio queries |
| WABA object / inventory | DOCUMENTED_OFFICIAL_SURFACE (M36/M37) | Company ownership/inventory unknown | Direct reads and optional portfolio discovery gated |
| Phone inventory / quality / name | DOCUMENTED_OFFICIAL_SURFACE (M38) | Company phone inventory unknown | Registration/name/connectivity fields and enums unverified |
| Business profile read | DOCUMENTED_OFFICIAL_SURFACE (M39) | Available fields/access unknown | Read-only sync gated; all profile writes excluded |
| WABA subscription list | DOCUMENTED_OFFICIAL_SURFACE (M40) | Existing consumers unknown | Read-only inventory mandatory |
| WABA subscribe/unsubscribe | DOCUMENTED_OFFICIAL_SURFACE (M41) | App assignment unknown | One WABA-level subscription surface; later subscribe needs explicit operator approval; no automatic subscribe/unsubscribe |
| Callback override | DOCUMENTED_OFFICIAL_SURFACE (M42) | OPTIONAL_ACCOUNT_DEPENDENT | DEFERRED unless topology demonstrates a need; no configuration change |
| GET webhook challenge | UNVERIFIED (M35) | Callback verification untested | Mandatory blocker; verify-token candidate is distinct from App Secret |
| POST webhook authenticity | UNVERIFIED (M35) | Signing-secret/live delivery untested | Mandatory blocker; no algorithm/header implementation from historical evidence |
| Envelope/incoming messages | DOCUMENTED_OFFICIAL_SURFACE historical envelope; incoming shape UNVERIFIED (M43/M45) | No captured fixture | Parser gated; no full Inbox or media download scope |
| Message statuses | DOCUMENTED_OFFICIAL_SURFACE (M44) | No captured fixture | Classification only in conditional Sprint 2; no invoice/price interpretation |
| Name/quality/account/review/template events | DOCUMENTED_OFFICIAL_SURFACE (M45/M46) | Field availability untested | Verify Cloud subscription field set and exact schemas; old tier constants excluded |
| Pagination | DOCUMENTED_OFFICIAL_SURFACE (M32/M47) | No multi-page probe | Cursor/limits/termination unverified |
| Error envelope / request IDs / usage headers | UNVERIFIED (M48) | No response evidence | Diagnostic names/semantics unknown; no fabricated quotas |
| Durable ingress/dedupe/attempts/DLQ/replay | INTERNAL_PLATFORM_FEATURE | Depends on verified routing/authenticity | Conditional scope; no implementation while Gate B CLOSED |
| Asset sync / Meta Health / event UI | INTERNAL_PLATFORM_FEATURE | Depends on account inventory | Conditional scope; Meta-reported and infrastructure states stay separate |

Sprint 0/1 are complete. No Meta adapter is implemented or enabled. At the initial inspection no real fixture existed; four sanitized account fixtures were added in the 2026-09-11 follow-up below. Gate B CLOSED; Phase B NOT STARTED; all outbound WhatsApp remains prohibited.

## Gate B continuation, 2026-09-11

At the first continuation the supplied .local/meta-gate-b.env was MISSING; the later successful validation and live results below supersede that observation. No account row was promoted and no live fixture captured. v26.0 is an explicitly authorized candidate for read-only testing, not a selected verified integration contract; no downgrade or live request occurred. Current Graph webhook documentation remains inaccessible (M50); the retrieved official Horizon overview (M51) is insufficient for WhatsApp authenticity. Gate B CLOSED. The owner requires a separate Gate B review before any Sprint 2 implementation, even if all checks later pass.

## Latest account verification, 2026-09-11

Evidence M52/M53 and [Gate B record](25-gate-b-evidence.md); exact account/version/time scope only. No VERIFIED_CURRENT_CONTRACT promotion: current authoritative lifecycle/webhook evidence remains unresolved.

| Capability | Contract evidence | Account evidence and restriction |
| --- | --- | --- |
| v26.0 tested GETs | DOCUMENTED_OFFICIAL_SURFACE | VERIFIED_ON_COMPANY_ACCOUNT; all responses reported v26.0, no downgrade |
| Token/App/scopes | DOCUMENTED_OFFICIAL_SURFACE | VERIFIED_ON_COMPANY_ACCOUNT: valid SYSTEM_USER and App match, WA/WM present, BM absent; expiry 2026-11-10T12:20:59Z |
| Direct WABA read | DOCUMENTED_OFFICIAL_SURFACE | VERIFIED_ON_COMPANY_ACCOUNT for supplied WABA; currency omitted, ownership/full inventory unverified |
| Phone inventory | DOCUMENTED_OFFICIAL_SURFACE | VERIFIED_ON_COMPANY_ACCOUNT: one matching phone, CLOUD_API, GREEN, code_verification_status NOT_VERIFIED; cursors but no next page |
| Business profile selected read | DOCUMENTED_OFFICIAL_SURFACE | VERIFIED_ON_COMPANY_ACCOUNT: returned vertical/messaging_product only; no profile mutation |
| WABA subscription list | DOCUMENTED_OFFICIAL_SURFACE | VERIFIED_ON_COMPANY_ACCOUNT: one subscription matching configured App |
| Portfolio discovery / BM | DOCUMENTED_OFFICIAL_SURFACE | Not attempted; direct workflow passed with BM absent. No universal requirement claim |
| Request/version/usage header presence | UNVERIFIED semantics | Observed safe header names and correlation IDs; quota/reset interpretation remains NEEDS_VERIFICATION |
| GET challenge / POST authenticity | UNVERIFIED | No handler or live verification; Verify Token empty, App Secret separately present |
| Subscribe/unsubscribe/override and every send | DOCUMENTED_OFFICIAL_SURFACE or UNVERIFIED as above | Not tested or changed; no authorization to mutate/send |

Four real sanitized GET fixtures now coexist with the unchanged SYNTHETIC multi-entry fixture. Gate B CLOSED pending current authoritative contract evidence; Sprint 2 NOT STARTED. No unrelated capability is promoted.

Historical credential recheck M54: operator-reported replacement still returns USER with expiry 2026-09-11T14:00:00Z; no production System User promotion. Exact v26.0 GETs remain account-verified. Verify Token is still empty; current authoritative webhook contract remains unresolved.

M55 latest credential recheck, 2026-09-11 12:21:29 UTC: Meta now reports SYSTEM_USER, valid, matching App and expected WA/WM scopes. All v26.0 target GETs pass without BM. Four fixtures refreshed from this actual capture. Token-type blocker resolved; current authoritative webhook/lifecycle contract and empty Verify Token remain open. No unrelated capability or Gate B approval follows from token replacement.

## Sprint 2 gate clarification

Owner decision (2026-09-12, continuing the 2026-09-11 review): **Gate B APPROVED; Sprint 2 AUTHORIZED; implementation COMPLETE; Sprint 2 acceptance OPEN; Gates C/D CLOSED.** Live GET challenge and authentic real Meta POST proof are mandatory Sprint 2 acceptance evidence, not prerequisites for starting the endpoint implementation. Sprint 0/1 acceptance remains COMPLETE. No outbound WhatsApp or Sprint 3 work is authorized.

The approved boundary implements the supplied existing-App/WABA GET topology and strict webhook ingestion. Approval is a delivery decision, not automatic promotion of every contract claim to VERIFIED_CURRENT_CONTRACT. Official Postman/first-party examples are DOCUMENTED_OFFICIAL_SURFACE, tested account GETs are separately VERIFIED_ON_COMPANY_ACCOUNT, and real GET/POST delivery proof is still required for Sprint 2 acceptance. Optional adapters and all outbound capabilities stay disabled.

## Gate C dependency review, 2026-09-14

[Gate C evidence](27-gate-c-evidence.md) supersedes initial evidence only for the outbound/window/template-read/category/status/pricing dependencies identified above. **Gate C CLOSED; Sprint 3 NOT STARTED.** No campaign, automation, Flow, catalog, calling or registration capability is promoted.

| Additional dependency | Evidence / account boundary | Operational result |
| --- | --- | --- |
| Delivered-message pricing / October policy | VERIFIED_CURRENT_CONTRACT from latest dated developer HTML; stale Markdown export excluded | Permission and price separate; October service allowance and utility charging require effective dates |
| Indonesia rate evidence | Official July CSV and October PDF retrieved/hashed; USD and IDR alternatives | Numeric evidence available, company billing currency UNKNOWN; draft only |
| Registry / confirmation / budgets | INTERNAL_PLATFORM_FEATURE; 39 offline specification cases | No runtime implementation, independent approval or publication |
| Delivery semantics | HTTP acceptance distinct from status facts and invoice amounts | UNCERTAIN retains exposure; no blind POST retry |

The target WABA timezone GET maps to America/Los_Angeles, not the operator's WITA timezone. Account discounts, external usage, payment readiness and authentication variant remain unverified; no discounted/free scope is inferred from missing fields. Six Graph GETs, zero mutations/sends. Detailed fields, limits, source hashes and deferred live account checks are in doc 27.
