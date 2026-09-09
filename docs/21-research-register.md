# Research register

Initial inspection date: **2026-09-08**; pricing/policy and focused engineering references re-inspected **2026-09-09**. No authenticated Meta account/API test was performed. Research used official developer references first, then WhatsApp's public site, then Meta's official Postman workspace. No BSP documentation is used as normative technical evidence.

## Evidence rules

READ means substantive page content was retrieved. INDEX means an official page's indexed excerpt was inspected; re-open before implementing. BLOCKED means login, HTTP 429 or fetch failure; its URL is a verification target, not evidence of behavior. Historical samples and folder listings establish a documented surface, not current limits or exact request validity. A recently crawled old example is still old.

READ/INDEX are retrieval quality, not VERIFIED_CURRENT_CONTRACT. No adapter is enabled by DOCUMENTED_OFFICIAL_SURFACE; ACCOUNT_VERIFICATION_REQUIRED remains pending for all external operations in doc 02. Capability enablement requires an evidence record: source URL/title, retrieved_at, relevant facts and effective dates, Graph version, permitted fields/scopes, account/region results, sanitized fixture hashes, reviewer and next_review_at. Store facts/small permitted excerpts and hashes of approved rate-card artifacts, not copied documentation dumps.

## Meta and WhatsApp sources

| Ref | Official source | Evidence and limits |
| --- | --- | --- |
| M01 | [WhatsApp Business Messaging Policy](https://business.whatsapp.com/policy) | READ 2026-09-08 and 2026-09-09; redirects to whatsappbusiness.com. Opt-in, opt-out, template/window, escalation and data-use requirements. Legacy conversation-pricing wording persists; see discrepancy below. Local legal requirements not determined. |
| M02 | [Platform pricing](https://business.whatsapp.com/products/platform-pricing) | READ 2026-09-08 and 2026-09-09; delivered-message pricing, market/category, service/applicable utility reply exemptions, utility/authentication tiers and eligible 72-hour free-entry overview. Interactive rate values were not retrieved. Do not treat general prose as an effective-dated rate policy. |
| M03 | [Developer pricing](https://developers.facebook.com/documentation/business-messaging/whatsapp/pricing) | BLOCKED: 429 on initial inspection; fetch failed again 2026-09-09. Current rates, international authentication rules, volume tiers and scheduled changes need review. |
| M04 | [Non-template pricing](https://developers.facebook.com/documentation/business-messaging/whatsapp/pricing/non-template-messages) | BLOCKED. Reported future changes are unconfirmed here. All message types must be capable of becoming billable under future policies. |
| M05 | [Official Postman workspace](https://www.postman.com/meta/whatsapp-business-platform/overview) | READ; identifies official collections and notes legacy management/Flows collections moved into Cloud API. |
| M06 | [Cloud API collection](https://www.postman.com/meta/whatsapp-business-platform/documentation/wlk6lh4/whatsapp-cloud-api) | READ; app/WABA subscriptions, owned/shared discovery, phone registration and token scope examples. Examples include v16/v18; not a version recommendation. |
| M07 | [Messages](https://www.postman.com/meta/whatsapp-business-platform/folder/o48mro7/messages) | READ; sends, context/reaction and media examples, read marking and messaging permission. |
| M08 | [Incoming message object](https://www.postman.com/meta/whatsapp-business-platform/folder/1dtuocp/messages-object) | INDEX; incoming types and metadata. Direct page later returned empty. Exact current shape needs fixtures. |
| M09 | [Status update notifications](https://www.postman.com/meta/whatsapp-business-platform/request/rgtfq23/message-status-update-notifications) | INDEX; sent/delivered/read notifications may arrive out of order; messages field envelope. |
| M10 | [Retrieve media URL](https://www.postman.com/meta/whatsapp-business-platform/request/fpj02x0/retrieve-media-url) | INDEX; authenticated retrieval and short-lived URL (sample says five minutes). No assumption of indefinite ID retention. |
| M11 | [Contact message](https://www.postman.com/meta/whatsapp-business-platform/request/e9dulgq/send-contact-message) | INDEX; contacts JSON example. Prose says contact while payload says contacts: validate current schema, do not copy blindly. |
| M12 | [Location message](https://www.postman.com/meta/whatsapp-business-platform/request/3pwqfn8/send-location-message) | INDEX; location send surface. |
| M13 | [Sticker by ID](https://www.postman.com/meta/whatsapp-business-platform/request/90ihbw3/send-sticker-message-by-id?tab=body) | INDEX; sticker send surface; current MIME/size constraints unresolved. |
| M14 | [List messages](https://www.postman.com/meta/whatsapp-business-platform/request/r3ufenw/send-list-message) and [reply buttons](https://www.postman.com/meta/whatsapp-business-platform/request/x0kd1at/send-reply-button) | INDEX; interactive variants, not unlimited button support. |
| M15 | [Templates collection](https://www.postman.com/meta/whatsapp-business-platform/folder/lczy75a/templates) | INDEX; sync, create, edit, delete variants. |
| M16 | [Edit template](https://www.postman.com/meta/whatsapp-business-platform/request/bpcsm6i/edit-template) | INDEX; historical management example. Edit quota, immutable fields and status restrictions NEEDS VERIFICATION. |
| M17 | [Template approved webhook](https://www.postman.com/meta/whatsapp-business-platform/request/enu4z5g/message-template-approved) | INDEX; message_template_status_update field and identifiers. |
| M18 | [Business profile](https://www.postman.com/meta/whatsapp-business-platform/request/i21qxwn/get-business-profile) and [profile permissions](https://www.postman.com/meta/whatsapp-business-platform/folder/13382743-2e0565c0-6b03-4ced-ae8e-bf6534ef1125) | INDEX; profile surface and management permission. Profile image uses a separate upload handle process. |
| M19 | [Block](https://www.postman.com/meta/whatsapp-business-platform/request/ywjuxcf/block-user-s), [unblock](https://www.postman.com/meta/whatsapp-business-platform/request/uv3p1z9/unblock-user-s), [list](https://www.postman.com/meta/whatsapp-business-platform/folder/lmh7s5u/block-users) | INDEX; block_users API exists. Exact scope/limits require current reference/account test. |
| M20 | [QR code examples](https://www.postman.com/meta/whatsapp-business-platform/documentation/3kru5r6/moved-whatsapp-business-management-api?entity=request-13382743-6a3f1859-533a-43d5-865b-5217e8970892) | INDEX; message_qrdls management; moved collection, reverify active API. |
| M21 | [Analytics examples](https://www.postman.com/meta/whatsapp-business-platform/folder/r6b0bp9/analytics) | INDEX; historical analytics/conversation analytics. Not proof of a 2026 invoice API. |
| M22 | [List Flows](https://www.postman.com/meta/whatsapp-business-platform/request/a1g0zfr/list-flows), [create](https://www.postman.com/meta/whatsapp-business-platform/request/30jvxag/create-flow), [get](https://www.postman.com/meta/whatsapp-business-platform/request/9cwjfve/get-flow) | INDEX; lifecycle/validation/preview fields; old versions in examples. |
| M23 | [Publish Flow](https://www.postman.com/meta/whatsapp-business-platform/request/wcidrlg/publish-flow) and [Flow operation examples](https://www.postman.com/meta/whatsapp-business-platform/documentation/wlk6lh4/whatsapp-cloud-api?entity=request-13382743-071cfa60-0704-41d2-bca2-36ba6bd33dfe) | INDEX; JSON assets, publish and deprecate surface. Deprecate sample command incorrectly repeats publish: implementation must use verified reference. |
| M24 | [Embedded Signup](https://www.postman.com/meta/whatsapp-business-platform/documentation/du6gzjv/embedded-signup?entity=request-13382743-3252619f-be88-4fe4-8da3-851038d1e2de) | INDEX; onboarding/review requirements; unnecessary for initial existing-asset connection. |
| M25 | [Account webhook components](https://www.postman.com/meta/whatsapp-business-platform/request/j09tht8/components) | INDEX; phone_number_name_update, phone_number_quality_update, account_update, account_review_update. Old example tiers are not adopted. |
| M26 | [Cloud API overview](https://developers.facebook.com/docs/whatsapp/cloud-api/overview), [messages](https://developers.facebook.com/docs/whatsapp/cloud-api/reference/messages), [webhooks](https://developers.facebook.com/docs/whatsapp/cloud-api/webhooks) | BLOCKED. Current throughput, pair limits, retry horizon, signatures and message constraints require direct verification. |
| M27 | [Graph changelog](https://developers.facebook.com/docs/graph-api/changelog) | BLOCKED. Latest supported Graph version UNKNOWN. Never use example v18 as default. |
| M28 | [Calling](https://developers.facebook.com/documentation/business-messaging/whatsapp/calling) | BLOCKED. Account/country eligibility, call permissions, webhook fields and pricing UNKNOWN. |
| M29 | [Coexistence onboarding](https://developers.facebook.com/docs/whatsapp/embedded-signup/custom-flows/onboarding-business-app-users) | BLOCKED. Eligibility, history/echo events and operational restrictions unresolved. |
| M30 | [Business-scoped user IDs](https://developers.facebook.com/documentation/business-messaging/whatsapp/business-scoped-user-ids) | BLOCKED. Current rollout and routing fields unresolved; internal identity design remains opaque and scope-aware. |
| M31 | [Catalog template example](https://www.postman.com/meta/whatsapp-business-platform/request/ark9lha/create-catalog-template) | INDEX; commerce template surface, not proof of payment availability. |

## Conflicts that affect implementation

On **2026-09-09**, [current public pricing](https://business.whatsapp.com/products/platform-pricing) describes charging for delivery by market/category, free service and applicable utility replies, utility/authentication volume tiers, and 72-hour free entry following qualifying ads/Page CTA contact. In contrast, the [messaging policy](https://business.whatsapp.com/policy) still says "Standard pricing applies for all conversation categories" and describes opening a conversation with a delivered business reply. This is an explicit source discrepancy, not evidence to implement conversation billing.

Decision: use inspected current pricing/rate evidence and its effective dates for executable charges; use messaging policy for messaging permission and consent. The public overview is not a complete machine-executable contract: exact eligibility, account/market/currency rules, tier accounting, numerical rates and effective-date changes still require verified official evidence and account checks. Neither historical status examples nor legacy wording overrides a reviewed effective pricing publication.

Third-party claims of changes from 2026-10-01 remain unconfirmed; M03/M04 do not provide retrieved supporting content here. **Future and exact current executable pricing policy: NEEDS VERIFICATION.** Gate C requires reviewed official artifacts in the Rate Card Registry. No numeric rates were reliably retrieved or invented; uncovered pricing stays blocked.

Old status examples refer to conversation pricing and include fields/statuses shared with deprecated on-premises APIs. Retain them only as historical parser fixtures. Do not infer Cloud API delete/recall support or invoice amounts from them.

## Engineering and product-pattern sources

| Ref | Source inspected | Use |
| --- | --- | --- |
| E01 | [Next.js release blog](https://nextjs.org/blog) | Next.js 16.3 release line observed; exact secure patch must be rechecked at implementation. |
| E02 | [Go releases](https://go.dev/doc/devel/release) | Go 1.27.1 observed; pin supported patch at Sprint 0. |
| E03 | [Valkey FAQ](https://valkey.io/topics/faq/) | Valkey operational/licensing evaluation. |
| E04 | [PostgreSQL row security](https://www.postgresql.org/docs/current/ddl-rowsecurity.html) | RLS limitations including privileged/table-owner bypass; motivates separate DB roles. |
| E05 | [OWASP password storage](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html) | Argon2id parameters must be benchmarked and versioned. |
| U01 | [Intercom Inbox explained](https://www.intercom.com/help/en/articles/6258745-the-inbox-explained) | INDEX; list/thread and table views, keyboard discoverability. |
| U02 | [Zendesk context panel](https://support.zendesk.com/hc/en-us/articles/4408828503450-Configuring-the-context-panel-in-the-Zendesk-Agent-Workspace) | INDEX; adjacent customer context rather than forced navigation. |
| U03 | [Stripe webhook workbench documentation](https://docs.stripe.com/webhooks) | READ; event/delivery inspection pattern, not Meta behavior. |
| U04 | [Mailchimp campaign checklist](https://mailchimp.com/help/create-and-send-regular-email/?v=114) | INDEX; explicit pre-send completeness, adapted to WhatsApp consent/cost. |

The UI study is documentation-based. No authenticated competitor UI was tested. The visual specification is an original design recommendation; comparative usability remains untested.

## Supplemental engineering verification, 2026-09-09

[PostgreSQL SELECT/locking documentation](https://www.postgresql.org/docs/current/sql-select.html) was read to support bounded SKIP LOCKED queue claims; it is not a consistency mechanism for budget checks. [RFC 9562](https://www.rfc-editor.org/rfc/rfc9562.html) was read for the UUID strategy. [WCAG 2.2](https://www.w3.org/TR/WCAG22/) was read as the accessibility target; no accessibility conformance test has been performed because no UI exists.

Focused correction sources, inspected 2026-09-09: [PostgreSQL row locks](https://www.postgresql.org/docs/current/explicit-locking.html) READ, supporting compatible FOR SHARE readers and conflicting policy writers; [WHATWG Web Storage](https://html.spec.whatwg.org/multipage/webstorage.html) READ, supporting the tab-local sessionStorage choice. Neither establishes measured performance or completed browser security testing. Meta/account restrictions are not applicable to these engineering references; runtime tests remain pending.

## Sprint 0 contract verification attempt, 2026-09-09

Gate B remains CLOSED. No authenticated API request, production token, subscription mutation or messaging occurred.

| Required contract | Official source inspected | Evidence quality and result | Remaining restriction/check |
| --- | --- | --- | --- |
| Selected Graph version and removal date | [Graph changelog](https://developers.facebook.com/docs/graph-api/changelog/), [version list](https://developers.facebook.com/docs/graph-api/changelog/versions), [v26 reference target](https://developers.facebook.com/docs/graph-api/changelog/version26.0) | BLOCKED: 429/fetch failure. Selected version = UNKNOWN, removal date = UNKNOWN; no default is configured | Read current official version/lifetime table. Search mirrors are not accepted evidence |
| GET webhook verification | [Meta webhook setup](https://developers.facebook.com/docs/graph-api/webhooks/getting-started), [historical official WhatsApp SDK](https://whatsapp.github.io/WhatsApp-Nodejs-SDK/api-reference/webhooks/start/) | Setup BLOCKED 429; SDK READ, historical DOCUMENTED_OFFICIAL_SURFACE for verify-token/challenge exchange | Exact current parameters, response/status/error rules and app/version context still UNVERIFIED |
| Authenticity/signature | Same webhook setup and historical SDK | Setup BLOCKED; historical SDK describes x-hub-signature-256 validation, not a newly verified current contract | Confirm current raw-byte algorithm, app-secret binding, missing/malformed signatures and current header handling before implementation |
| App/WABA permissions | [Official Cloud API collection](https://www.postman.com/meta/whatsapp-business-platform/documentation/wlk6lh4/whatsapp-cloud-api) | READ again: documented WM/WA permissions and BM for portfolio queries; DOCUMENTED_OFFICIAL_SURFACE only | Current scopes, asset assignments, app access level and account probes pending |
| Required subscriptions | Same official collection; [Cloud webhook overview](https://developers.facebook.com/documentation/business-messaging/whatsapp/webhooks/overview) | Postman READ documents WABA subscribed_apps; current developer overview BLOCKED 429 | Verify app object/fields and existing callbacks separately from WABA subscription; no changes made |
| Envelope and constraints | [Cloud API get started](https://developers.facebook.com/docs/whatsapp/cloud-api/get-started), webhook overview and M08/M09 | Current pages BLOCKED 429; historical shapes remain documented surface | Batch cardinality, payload/media limits, acknowledgement/retry horizon, event names and exact versioned fixtures pending |

No row has been promoted to VERIFIED_CURRENT_CONTRACT. The one local fixture is explicitly SYNTHETIC with null Graph version; it does not fill the missing account evidence. The backend's generic 1 MiB body limit is a local foundation setting, not a claimed Meta webhook maximum.

Current pricing evidence remains M01/M02 inspected 2026-09-09, including the recorded policy/pricing discrepancy. Effective numerical rate artifacts and account eligibility remain unavailable/unverified; no rates are seeded and Gate C remains CLOSED. Before Gate B, an integration reviewer must inspect the blocked official references, bind exact operations to a supported version, record sanitized account-derived results and approve the contract package.
