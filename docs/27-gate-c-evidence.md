# Gate C evidence and decision

Owner clarification (2026-09-15): **Sprints 0/1/2 COMPLETE; Gates A/B/C APPROVED; Gate D CLOSED; Sprint 3 AUTHORIZED. COMPANY META BILLING CURRENCY UNKNOWN; PAID-SEND AUTHORITY CLOSED.** Currency must not be inferred from timezone, business/phone/recipient country, available rate cards or locale. Only a currently reviewed, provably zero-cost policy can permit the single operator-triggered controlled TEXT live reply after all recipient/callback/security prerequisites. Paid/template live sends, outbound media, campaigns and Sprint 4 remain unauthorized.

## Owner decision and hosted baseline, 2026-09-15

The September 14 research is accepted. Unknown billing currency blocks paid authority, not Gate C approval or local guard implementation. Zero cost must come from independent published current policy, never a missing rate/default/unknown currency. October effective-date changes remain binding; no timeless service-free assumption is authorized. All template live sends remain CLOSED under the explicit Sprint 3 scope, including the earlier proposed second test.

Completed research was published at **4bc49539745ead1f3b5e53470e7aa77671ab29cc**. Hosted [run 34840457519](https://github.com/mbingsdk/whats/actions/runs/34840457519) passed all repository, PostgreSQL, browser and dependency steps on September 14, completed 11:55:40Z. It used isolated fixtures, not live Meta. Owner-clarification baseline **efe31864e5ed0c090d12c292f269eec22941bfb6** is published and [run 34879123286](https://github.com/mbingsdk/whats/actions/runs/34879123286) **PASS** on all repository/PostgreSQL/browser/dependency steps. The result was recorded before Sprint 3 implementation began.

## Historical research assessment, 2026-09-14

The following dated research preserves the original assessment. Its Gate C CLOSED/currency-blocker interpretation is superseded above; original source facts, artifacts and actual test results are unchanged.

Inspection/review record: **2026-09-14**. Sprint 0, Sprint 1 and Sprint 2 implementation and acceptance are COMPLETE. Gates A/B are APPROVED. **Gate C CLOSED; Gate D CLOSED; Sprint 3 NOT STARTED.** This package is research and executable specification only. No send adapter, product UI, migration or Sprint 2 webhook change is included.

Current Gate C blocker: the company's billing currency is **UNKNOWN**. An explicit v26.0 WABA currency GET omitted the field; operator confirmation from WhatsApp Manager/Meta Billing is pending. Indonesia is the initial recipient market, not proof of IDR billing. Official USD and IDR numerical evidence has been obtained, but neither is selected as company configuration. If the company uses another currency, obtain its official card and repeat validation. No zero-cost exception is approved in this review.

## Evidence quality and provenance

The [source manifest](fixtures/gate-c/source-manifest.json) records inspection timestamps, original source URLs, SHA-256 hashes and retrieval quality. Original public bytes and protected account evidence remain in ignored local operator storage. Signed CDN query strings, credentials, phone numbers and template content are not published.

Current developer HTML was retrieved and inspected despite failures through another retrieval path. The [developer pricing page](https://developers.facebook.com/documentation/business-messaging/whatsapp/pricing), updated **2026-09-10**, supplies the latest October policy. The [public pricing overview](https://whatsappbusiness.com/products/platform-pricing/) describes current delivered-message pricing; its shorter treatment and the retrieved developer Markdown export omit the latest changes. The Markdown export is explicitly a STALE VARIANT for October policy. The [non-template update](https://developers.facebook.com/documentation/business-messaging/whatsapp/pricing/non-template-messages/), updated August 25, corroborates October charging but predates the September 10 allowance clarification. No stale representation overrides the latest dated substantive HTML.

Specific message guides contain current contracts and v26.0 examples. The generic message/template API reference URLs returned a shell without usable schema; those responses remain UNVERIFIED for undisclosed fields. An official sample using v23 or containing duplicate JSON properties is not copied as the current contract. No historical Postman sample is promoted solely because it is official.

Contract quality is **VERIFIED_CURRENT_CONTRACT for the bounded fields/rules below**, separately from account applicability. System User, App/WABA binding and WA/WM scopes have prior bounded evidence in [Gate B](25-gate-b-evidence.md). This run verifies template GETs only. Outbound account behavior remains ACCOUNT_VERIFICATION_REQUIRED during separately authorized live acceptance; Gate C does not require or authorize a send probe.

## Pricing model and dates

Meta charges per **delivered message**, with category and recipient rate-card market as inputs. Indonesia is the +62 rate-card market; this is not physical geolocation. Categories are MARKETING, UTILITY, AUTHENTICATION and SERVICE. Authentication international is a rate variant. Category, permission, discount eligibility and amount are separate facts.

| Rule | Current period through September 30 in the WABA timezone | Announced from October 1 in the WABA timezone |
| --- | --- | --- |
| Service messages | Free under the current service policy, subject to free-form permission | First 1,000 delivered service messages per business phone number per month free; subsequent service messages use the service list rate |
| Utility templates inside the customer service window | Free under the current utility-reply concession | Chargeable at the applicable utility rate unless another verified concession applies |
| Utility outside that window | Applicable delivered-message rate | Applicable delivered-message rate |
| Marketing / authentication | Applicable rate; active service window alone does not make these free | Applicable rate; same independence from permission |
| Qualified free entry | Eligible messages free for 72 hours | Remains a separate free-entry concession |
| Volume tiers | Utility and authentication, including applicable international variant | These categories; no assumed service or marketing tier discount |

Free entry requires the qualifying Click-to-WhatsApp ad or Facebook Page CTA entry on supported mobile clients and a business response within 24 hours; the 72-hour period starts with that response. A referral field alone does not prove qualification. Outside the customer service window, an approved template remains necessary even during free entry. Desktop/web acquisition is not assumed eligible.

Tier accounting aggregates the business portfolio's WABAs for the relevant market/category/month; charged volume advances tiers. Account counters and external senders are not known here. Do not grant discounts from this application's local count. Use a reviewed list-rate upper bound where sufficient. The October service allowance likewise needs authoritative complete phone usage and safe reservation; absent that proof, evaluate the paid upper bound or block zero-cost-only sending.

The [international authentication policy](https://developers.facebook.com/documentation/business-messaging/whatsapp/pricing/authentication-international-rates/) describes a threshold above 750,000 qualifying messages over a moving 30-day period, a later effective start, enduring eligibility and primary-business-location conditions. Indonesia recipient plus Indonesia primary business location does not itself invoke the international variant. The company's primary business location and eligibility fields were omitted by GET; no company exemption is inferred. Authentication is excluded from the proposed first controlled tests until its variant is established or a reviewed conservative bound covers it.

GET returned timezone_id **1**, mapped by the [official timezone table](https://developers.facebook.com/documentation/business-messaging/whatsapp/timezone-ids/) to **America/Los_Angeles**. October 1 midnight there is **2026-10-01T07:00:00Z**, not WITA midnight. Store IANA timezone and effective local dates, not a permanent fixed offset. The cards are effective July 1 and October 1, respectively. Delivery is the charging point: a September dispatch with possible October delivery cannot rely indefinitely on September's zero-cost rule. Cover the possible delivery horizon with reviewed conservative exposure, or block uncovered boundaries.

Meta documents quarterly rate-change opportunities; reinspection and effective-date publication are required. Proposed evidence review deadline is **2026-09-21T00:00:00Z**, subject to independent reviewer acceptance. This date is not an approval or a guarantee of unchanged rates. Payment method availability, commercial adjustments, taxes, external sender usage and company invoice rounding remain UNKNOWN; there is no claim to bound the entire Meta account invoice from local reservations.

## Customer service window

Current [service-message guidance](https://developers.facebook.com/documentation/business-messaging/whatsapp/messages/send-messages/) says user messages **or user calls** can open/reset the 24-hour window. Thus “only messages qualify at Meta” would be inaccurate. Calling remains DEFERRED. The initial local activation scope is authenticated inbound user text; other qualifying message types require verified typed provenance before activation. A deferred call must not silently enable calling or manufacture a local window.

Business replies, templates, delivery/status callbacks, notes and replay processing time do not reset it. The [text webhook reference](https://developers.facebook.com/documentation/business-messaging/whatsapp/webhooks/reference/messages/text/) describes a Unix event timestamp; it is not a separately queryable authoritative customer-service-window ledger. Preserve signed original event time and first durable ingress time. Proposed conservative basis is the earlier of those times; reject malformed/future/untrusted evidence, retain original ingress through replay, and take the maximum of eligible conservative event times for subsequent user messages. Local conservatism can close earlier than Meta, never deliberately later.

| Projection | Specification |
| --- | --- |
| ACTIVE | Trusted expiry is more than 15 minutes ahead |
| EXPIRING_SOON | Trusted expiry is ahead but within the internal 15-minute warning threshold |
| EXPIRED | Trusted expiry is at or before server/database time |
| UNKNOWN | No usable qualifying evidence, invalid/future timestamp or unavailable governing rule |

Expiry is basis + 24 hours. Final dispatch applies a proposed 5-second safety margin; a preflight can pass and queued dispatch still fail TEMPLATE_REQUIRED. Unknown free-form eligibility fails closed. No automatic conversion into a paid template. Pricing is evaluated independently: an active window does not establish a zero amount after policy changes.

## Outbound v26.0 contract

**Do not execute:** POST /v26.0/{PHONE_NUMBER_ID}/messages. Common body requires messaging_product=whatsapp, to, type and the corresponding type object; recipient_type=individual is explicit in the bounded contract. Use the assigned System User bearer credential with WM and the target phone binding. No client-controlled alternate phone/organization authorization. HTTP success with messages[].id is provider acceptance, not delivery. Account quotas, payment readiness and each selected media/recipient path still need controlled acceptance.

Every non-template row below requires the service window, consent and the shared send gate. Templates require current remote approval/category/language/parameters plus pricing; they can be eligible outside the window. All types remain unimplemented here. Media ID or link is an alternative, not permission to accept arbitrary URLs; future media handling needs tenant scope, MIME/size validation and SSRF controls.

| Type / current official guide | Required fields and bounded limits | Media / template dependency and applicability |
| --- | --- | --- |
| [Text](https://developers.facebook.com/documentation/business-messaging/whatsapp/messages/text-messages/) | text.body, up to 4,096 characters; preview_url optional boolean | No template; no application attachment fetch implied |
| [Image](https://developers.facebook.com/documentation/business-messaging/whatsapp/messages/image-messages/) | image.id or link; optional caption up to 1,024 | JPEG/PNG, 8-bit RGB/RGBA, up to 5 MB |
| [Video](https://developers.facebook.com/documentation/business-messaging/whatsapp/messages/video-messages/) | video.id or link; optional caption up to 1,024 | MP4/3GPP, H.264/AAC, up to 16 MB, single or no audio stream |
| [Audio](https://developers.facebook.com/documentation/business-messaging/whatsapp/messages/audio-messages/) | audio.id or link; optional voice flag | AAC/AMR/MP3/MP4 audio/OGG OPUS up to 16 MB; voice requires OGG OPUS mono |
| [Document](https://developers.facebook.com/documentation/business-messaging/whatsapp/messages/document-messages/) | document.id or link; optional caption up to 1,024 and filename | Supported text/PDF/Word/Excel/PowerPoint formats up to 100 MB |
| [Sticker](https://developers.facebook.com/documentation/business-messaging/whatsapp/messages/sticker-messages/) | sticker.id or link | WebP; static up to 100 KB, animated up to 500 KB; no unverified dimension rule |
| [Contacts](https://developers.facebook.com/documentation/business-messaging/whatsapp/messages/contacts-messages/) | contacts array, name.formatted_name required; optional name/address/phone/email fields | Current guide maximum 257 contacts; initial internal validator may cap one; no CRM implementation |
| [Location](https://developers.facebook.com/documentation/business-messaging/whatsapp/messages/location-messages/) | location.latitude and longitude documented as strings; optional name/address | Validate decimal coordinate ranges; no live tracking |
| [Reaction](https://developers.facebook.com/documentation/business-messaging/whatsapp/messages/reaction-messages/) | reaction.message_id and emoji | Received same-thread target, no older than 30 days, not deleted or a reaction; only sent status; removal syntax unverified |
| [Interactive list](https://developers.facebook.com/documentation/business-messaging/whatsapp/messages/interactive-list-messages/) | type=list, body.text 4,096, action.button 20; 1–10 sections and 1–10 rows total; row id 200/title 24/description 72 | Section title 24; optional text header/footer 60; current body limit differs from older samples |
| [Reply buttons](https://developers.facebook.com/documentation/business-messaging/whatsapp/messages/interactive-reply-buttons-messages/) | type=button, body 1,024; 1–3 unique reply buttons, id 256/title 20 | Optional text/image/video/document header; footer 60 |
| [CTA URL](https://developers.facebook.com/documentation/business-messaging/whatsapp/messages/interactive-cta-url-messages/) | type=cta_url; body 1,024; action.name=cta_url, parameters.display_text 20 and url | Bounded subtype, not needed for first test; no implied Flow/catalog support |
| [Template](https://developers.facebook.com/documentation/business-messaging/whatsapp/templates/overview/) | template.name, language.code; component parameters match selected approved template | Bounded simple utility strategy below; no blanket carousel/authentication/marketing enablement |

These are inspected guide fields, not an exhaustive permissive JSON schema. Unknown optional fields remain rejected/deferred. Reply-context extensions and media-upload/download APIs are not promoted merely by these guides.

## Company template inventory and fresh eligibility

Six read-only Graph requests returned HTTP 200 with response version v26.0: three inventory/currency requests at **06:40:41–43Z** and three timezone/location/international-eligibility requests at **11:25:38–39Z**. Template pagination completed in one page. [Account evidence](fixtures/gate-c/account-evidence.json) and [sanitized inventory](fixtures/gate-c/template-inventory.json) preserve provenance. Exact Meta IDs and names are mapped in protected local evidence; public aliases and hashes avoid publishing business identifiers or content.

| Alias | Meta category | Status / language | Observed components | Controlled-test strategy |
| --- | --- | --- | --- | --- |
| T1 | MARKETING | APPROVED / en_US | HEADER, BODY, FOOTER, BUTTONS; zero text parameters | Not selected |
| T2 | UTILITY | APPROVED / en_US | HEADER, BODY, FOOTER, BUTTONS; three text parameters | Parameterized alternative only after safe content review |
| T3 | MARKETING | APPROVED / en_US | BODY; zero parameters | Not selected |
| T4 | UTILITY | APPROVED / en_US | HEADER, BODY, FOOTER; zero parameters | Candidate simple template, subject to operator content/purpose review and recipient consent |
| T5 | MARKETING | APPROVED / en_US | BODY, BUTTONS, CAROUSEL | Carousel delivery DEFERRED |

All five returned quality score UNKNOWN and rejected_reason NONE. last_updated_time was 2026-09-11T11:38:08+0000; the observed quality date is retained in the fixture. APPROVED does not mean approved by the owner for a safe controlled test. T5's top-level inventory is not recursive carousel validation. Two approved utility candidates exist, but no live-test template has been owner-selected. Do not create/mutate a template to improve this evidence.

[Category rules](https://developers.facebook.com/documentation/business-messaging/whatsapp/templates/template-categorization/) distinguish requested transactional nonpromotional utility, identity verification authentication, and promotional/mixed marketing content. Meta-observed category/status controls eligibility and the pricing input; local labels cannot override it. Actual delivery pricing metadata is retained separately and discrepancies require reconciliation.

[Category-update events](https://developers.facebook.com/documentation/business-messaging/whatsapp/webhooks/reference/template_category_update/) can announce impending or completed changes. Future handling invalidates affected quotes/jobs and refreshes the selected remote template. No subscription changed here. Before final dispatch, perform a fresh selected-template GET, proposed maximum evidence age 30 seconds, and issue a permit no longer than 5 seconds subject to window/price bounds. Bind ID, language, category, status and canonical component hash. Material changes require new authorization; stale/unavailable remote evidence blocks. These are internal conservative limits, not Meta guarantees. Meta can still change after the GET; preserve that race and handle rejection/changed billing without pretending the remote check and send are atomic.

## Indonesia numeric evidence

All amounts below are **per delivered message, list tier**, currency-qualified decimals. These are alternative official cards, not currency conversion and not a company publication.

| Effective local date | Currency | Marketing | Utility | Authentication | Authentication international | Service list rate |
| --- | --- | --- | --- | --- | --- | --- |
| 2026-07-01 | USD | 0.0411 | 0.0250 | 0.0250 | 0.1360 | n/a: current zero-cost policy |
| 2026-07-01 | IDR | 586.33 | 356.65 | 356.65 | 1940.13 | n/a: current zero-cost policy |
| 2026-10-01 | USD | 0.0411 | 0.0250 | 0.0250 | 0.1360 | 0.0250, subject to allowance/concessions |
| 2026-10-01 | IDR | 586.3300 | 356.6500 | 356.6500 | 1940.1300 | 356.6500, subject to allowance/concessions |

Source: downloadable official cards from the [developer pricing page](https://developers.facebook.com/documentation/business-messaging/whatsapp/pricing). July USD/IDR CSV hashes are **9aa0fcb64bd1455aa010b5738a3eca90d290f09ae1a71a153d0d148348408ffa** and **21e429a67aadc420eba36fc73be8bac7b10fc587303e3906bcadb0fd00c90a7c**. October USD/IDR PDF hashes are **10bb591ab1a0197f6f5b7258a861358cbc2e04950c8047982d8e5d0cb6502a97** and **1703a6c639694240c97955afa597226eecb2f5dc1705b8c1dca0495ba9bc4ea0**. The manifest includes original artifact identities and tier CSV hashes.

October downloads advertised as CSV contained XLSX bytes; their mismatch was quarantined. Numerical extraction used official PDF alternatives, not a guessed CSV parser. PDF page 1 supplies rates and page 3 supplies Indonesia tiers; layout-text extraction was used, with no claim of visual PDF verification. All 36 currency/category tier intervals and amounts were compared with July CSV and match. The [20 rate rows](fixtures/gate-c/indonesia-rates.json) and [36 tier rows](fixtures/gate-c/indonesia-tiers.json) are evidence only, not runtime configuration. No Service discount tier is inferred.

## Registry operational rehearsal

The [rehearsal record](fixtures/gate-c/registry-rehearsal.json) exercises the proposed process against these real artifacts. Import capture, decimal validation, uniqueness, contiguous tier ranges and July/October comparison passed. Company currency/operational coverage did not validate. Actual candidate state remains **DRAFT**. A diff preview finds new October service pricing and changed service/utility concessions, with unchanged Indonesia template list amounts. It is not a formal DIFFED result against an approved company publication.

| State | Exact required evidence |
| --- | --- |
| DRAFT | Organization-scoped source capture, artifact hash, parser/version, candidate revision and proposed base; validation failures remain here |
| VALIDATED | Revision-bound successful provenance, currency, required market/category/date/tier/variant coverage, decimal/interval/rounding and policy-predicate validation |
| DIFFED | Validated revision compared to current publication base; immutable rate/rule/coverage diff and affected schedules |
| IN_REVIEW | Frozen source/import/validation/diff hashes submitted to a distinct currently authorized reviewer |
| APPROVED | Reviewer explicitly approves exact hashes, coverage and next_review_at; editing invalidates approval |
| PUBLISHED | Authorized publisher with recent MFA verifies approval/permissions and compare-and-set base, then atomically selects an immutable publication under the organization policy barrier |
| SUPERSEDED | Derived publication relationship after a new approved publication references the old one; old import remains terminal PUBLISHED with immutable bytes/history |

CHANGES_REQUIRED returns through DRAFT. SUPERSEDED is a publication view, not a new mutable import state; this preserves the existing [domain](04-domain-model.md), [database](05-database-design.md), [API](06-api-contract.md) and [pricing protocol](12-pricing-and-budget-guard.md). Corrections always create a new import/review/publication. No approval, reviewer identity, production publication or selected company currency was fabricated. Independent review/publication is required before potentially paid live testing; Gate C requires the workflow to be defined, not a pretend runtime Registry.

Missing required coverage or overdue review blocks potentially paid dispatch. Published rows never edit historical estimates/reservations. Reviewed upper bounds may ignore uncertain discounts, but must cover the actual possible variant/delivery interval. Pending policy inputs include company invoice rounding and treatment of non-Meta costs; conservative decimal exposure is not a reconciled invoice.

## Executable specification and future tests

Run: python docs/fixtures/gate-c/check_spec.py. The [fixture matrix](fixtures/gate-c/pricing-cases.json) contains **39 offline scenarios**. Synthetic authorization, consent, usage and budget inputs are test assumptions, not real authority. Scenario J's increased amount is synthetic, not an announced Meta increase.

| Required case | Expected decision |
| --- | --- |
| A: inbound 5 minutes ago, text | Permission allowed; September service zero-cost policy evaluated independently |
| B: preflight at 23h59m, dispatch after expiry | Final free-form dispatch blocked |
| C: last inbound 25 hours ago | TEMPLATE_REQUIRED |
| D: expired window, approved Utility | Template eligibility plus current paid evidence/confirmation/budget required |
| E: active window, Marketing | Active window does not imply free pricing |
| F: qualified free entry | Template may be zero-cost; expired-window free-form still blocked |
| G: missing potentially paid coverage | PRICING_COVERAGE_MISSING |
| H: overdue rate review | RATE_REVIEW_OVERDUE |
| I: template category changes | TEMPLATE_CHANGED; authorization invalid |
| J: price exceeds confirmed maximum | NEW_CONFIRMATION_REQUIRED |
| K: repeated same organization/key/payload | Same modeled intent/reservation; changed payload conflicts |
| L: possible write followed by timeout | UNCERTAIN, exposure retained, no automatic retry |

Additional cases cover October allowance exhaustion/unknown usage, utility charging, delivery crossing the October boundary, exact timezone transition, unknown/future/replayed timestamps, status non-reset, safety margin, paused/stale templates, currency unknown, cross-organization keys, 5xx/missing provider ID, definite rejection, permission/consent revocation, budget exhaustion, insufficient referral evidence, unknown authentication variant and future-rate coverage.

Executed result: **39 scenarios PASS; 20 rate rows and 36 contiguous tier rows PASS**. This is a decision specification using Decimal and a fake time basis, without network, database or product imports. In-memory idempotency does not prove database concurrency. Sprint 3 must add real PostgreSQL last-budget-unit races, single-use authorization/idempotency races, tenant FK/RLS/runtime/pool tests, permission revocation versus permit commit, template/policy publication races, status reorder/orphans, and crash injection around HTTP writes/result persistence. No Sprint 3 implementation correctness is claimed.

## Status, money and uncertain sends

The [status webhook contract](https://developers.facebook.com/documentation/business-messaging/whatsapp/webhooks/reference/messages/status/) provides statuses[].id/status/timestamp/recipient_id in the phone/WABA envelope. Preserve sent, delivered, read and failed as immutable facts. Read may arrive without a delivered callback; projections may advance without inventing a received callback. Duplicates/reorder never regress; contradictory evidence produces UNKNOWN_CONFLICT. Reactions document only sent; voice played is optional.

Pricing may include pricing_model, type, category and currently billable. billable is documented for future deprecation, so tolerate absence. v24+ normally omits conversation except free-entry cases; conversation expiry is not the service window. Prose and examples differ in which status carries pricing: accept validated metadata wherever provided, never require it on every event or infer free from absence.

| Fact | Meaning |
| --- | --- |
| META STATUS FACT | Provider acceptance/status evidence; sent is not delivered |
| META PRICING METADATA | Category/model/type/billable observations, without invoice amount |
| WABA CONTROL ESTIMATE | Decimal estimate from frozen reviewed rates/policy, explicitly internal |
| RESERVED EXPOSURE | Held financial authority while outcome/cost is unresolved |
| ACTUAL INVOICE / RECONCILED COST | Amount supported by an official financial source, separately attributed |

Existing states remain: intent PENDING, READY after authorization, RESERVED, DISPATCHING, then ACCEPTED/FAILED/RETRY_WAIT/UNCERTAIN. Attempt ACCEPTED requires a usable provider ID; acceptance with missing ID or lost persistence leaves correlation/outcome uncertain. Message delivery progresses separately from ACCEPTED through SENT/DELIVERED/READ, or supported FAILED/UNKNOWN_CONFLICT. HTTP success never fabricates delivery or settlement.

Possible request bytes written, timeout, unclassified server failure, worker crash in DISPATCHING, missing ID or lost response means **UNCERTAIN**; retain exposure and do not resend from lease expiry. Proven zero-byte failure or narrowly proven definite non-acceptance can become retry-eligible only after fresh guards. Generic error prose saying “retry” is insufficient for an ambiguous POST. No blind automatic retry allowlist is granted. A reviewed replacement is a new linked intent with explicit duplicate-risk disclosure and authority; missing status is not proof of non-delivery. Callback opaque data is correlation, not a provider idempotency guarantee.

## Controlled live acceptance policy

No recipient was designated or added during Gate C. A previous Sprint 2 inbound test does not authorize outbound use of that number. Before live testing, the operator must designate a controlled personal WhatsApp number, record explicit owner/recipient consent and known market, mark TEST, exclude it from future campaigns by default, and approve harmless content without customer/employee PII.

Define a separate TEST budget with hard stop, one designated recipient, one paid-message authorization at a time, no campaign and no batch. Currency and monetary ceiling remain unset/disabled until the owner approves the exact maximum after rates are verified. Reserved plus posted/adjusted exposure counts; uncertain sends retain authority. No invented “tiny” monetary value is configured.

After separate Sprint 3 implementation authorization and passing local tests:

1. Arrange an explicitly approved stable or controlled HTTPS callback. Confirm signature verification, correct target and ingress/status durability.
2. Reinspect effective pricing, company currency, usage/payment readiness and reviewed rate publication. Refresh template/category/status; designate the TEST recipient and obtain exact budget authority.
3. Operator sends a fresh inbound message; observe trusted ACTIVE window. One authorized agent submits one simple reply through the shared Pricing Guard. Require proven zero-cost coverage for its delivery horizon. After October, unknown global allowance usage cannot establish the free test. Observe one Meta ID and real lifecycle; accepted/sent is not delivered.
4. After all guards pass, select the safely reviewed simple utility candidate (or stop if unsuitable), show maximum/currency and obtain confirmation. If the estimate exceeds zero, require **explicit owner approval immediately before the test**, then authorize exactly one template. This run does not supply that approval.
5. Record status/pricing metadata and estimated/reserved exposure separately; reconcile only against financial evidence. Ambiguity stops further sends pending review.

Sprint 2's temporary tunnel callback was rolled back to the prior **empty App callback** state; App–WABA subscription was retained and temporary listeners/tunnel stopped. No permanent callback exists by inference. Local Sprint 3 implementation can later proceed after authorization; live Inbox acceptance requires reachable inbound/status delivery. Permanent deployment is a separate operational decision.

## Review outcome and remaining conditions

| Criterion | This review |
| --- | --- |
| Outbound/window/template requirement and status/retry specification | Bounded first-party contracts established; no send probe |
| Template inventory / safe strategy | Five approved observed, two utility candidates; bounded T4 strategy subject to actual content/consent review |
| Official pricing and Indonesia numeric evidence | Available for current and October periods; original artifacts hashed |
| Correct company billing currency | **BLOCKER: omitted by GET, operator confirmation pending** |
| Registry process and executable fixtures | Defined against real artifacts; candidate DRAFT; 39 offline scenarios pass |
| Controlled-recipient and test-budget policy | Defined; no recipient or monetary authority invented |
| Gate decision | **CLOSED**, no zero-cost exception; Sprint 3 NOT STARTED |

Currency is the immediate Gate C evidence blocker. Authentication variant can remain excluded from initial text/simple-utility tests; unknown discounts require conservative verified coverage. Independent publication, selected safe template/recipient, exact TEST cap and immediate paid-test approval, payment readiness and an approved reachable callback are subsequent live-acceptance conditions. They do not authorize sends in this run or justify fabricated production operations.

Review changes are confined to the requested seven documents, this evidence document and offline fixtures. Existing unrelated bootstrap/environment edits are preserved. Six authorized Graph GETs and **zero Graph mutations/outbound WhatsApp sends** were performed. Sprint 2 webhook code and migrations are untouched. Source/documentation/migration-integrity checks are recorded with their actual results in the [fixture README](fixtures/gate-c/README.md). Prior hosted success is baseline evidence only, not CI verification of this unpublished Gate C package.

## Sprint 3 implementation handoff, 2026-09-15

The owner-approved documentation baseline and hosted PASS above precede product changes. Sprint 3 uses reviewed artifacts as controlled Registry inputs and separate zero policy evidence. It does not auto-publish numeric rates or infer currency. [Sprint 3 evidence](28-sprint-3-evidence.md) owns subsequent verification.

[M73](21-research-register.md#m73-service-delivery-horizon-review-2026-09-15) records why the inspected Direct Send TTL contract cannot bound ordinary Service text delivery. Real Service pricing coverage remains UNVERIFIED/fail-closed. Synthetic bounded-delivery tests are not account or live evidence. Gate C APPROVED; paid/template authority and Gate D CLOSED.
