# Gate B: Meta contract and account verification

Latest review: 2026-09-11. **Gate B CLOSED. Read-only account probes PASS for the tested endpoints; System User verified; current authoritative webhook/lifecycle evidence remains unresolved. Sprint 2 NOT STARTED.** The owner requires a separate Gate B review even if the gate becomes approved. No WhatsApp send or Meta mutation is authorized in this run.

## Configuration and token

The operator configuration is now PRESENT at .local/meta-gate-b.env. META_GRAPH_VERSION and all supplied numeric identifiers passed format validation. Access-token and App Secret files are PRESENT and non-empty. The referenced webhook Verify Token file is PRESENT but empty (MISSING value); this did not prevent independent GET account probes. *_FILE values were treated as paths, including absolute Windows paths, and secrets were read only at runtime.

Latest Meta debug_token check at 2026-09-11T12:21:29Z returned HTTP 200, is_valid=true, matching configured App ID and type=SYSTEM_USER. Token expiry is 2026-11-10T12:20:59Z (20:20:59 WITA); data_access_expires_at=0 was observed and is not interpreted as a guarantee against revocation. App Secret was used only in the protected debug authentication context, never as Verify Token.

Historical attempts returned USER: initial expiry 13:00 UTC, then 14:00 UTC after the first operator-reported replacement. Those observations are retained as history and are superseded by the successful SYSTEM_USER result. No secret value or prefix was emitted.

| Permission | Captured token scope | Bounded conclusion |
| --- | --- | --- |
| whatsapp_business_management | PRESENT | Available on this credential during the successful reads |
| whatsapp_business_messaging | PRESENT | Available on this credential; no send authorization or send test |
| business_management | MISSING | The tested direct-WABA workflow succeeded without this scope. Portfolio enumeration was not called; no claim about its requirements was inferred from this run |

Production strategy remains a dedicated company-owned System User, assigned only the intended App/WABAs with operation-specific scopes. Actual SYSTEM_USER type, intended App match, WA/WM scopes and target read access are now verified. Rotation/revocation controls remain required; no silent credential/version fallback is permitted. Complete portfolio ownership remains outside this direct-WABA probe.

## Live request evidence

Four explicit GET-only passes were executed: initial account checks at 12:07:44-12:07:48 UTC, then fixture/provenance capture at 12:09:30-12:09:33 UTC on 2026-09-11. The second pass added original-response checksums and documented profile field selection because the default profile read returned only messaging_product. A third five-GET pass after operator-reported token replacement at 12:14:43 UTC also passed, while token type remained USER. A fourth five-GET pass at 12:21:29-12:21:32 UTC verified the new SYSTEM_USER and all target reads. All twenty requests returned HTTP 200; no Meta error code/subcode was returned.

Every response reported facebook-api-version=v26.0. Each private request record includes configured version, UTC, endpoint class, HTTP result, pagination presence, observed diagnostic header names, request/trace IDs and a sanitized response. No Authorization header, token-bearing URL or raw body was logged.

| Endpoint class | Request surface | Actual result |
| --- | --- | --- |
| Token debug | GET /v26.0/debug_token; query token held only in runtime request | Valid SYSTEM_USER token, App match, WA/WM scopes; no BM |
| Direct WABA | GET /v26.0/{WABA-ID}?fields=id,name,currency,timezone_id | One supplied WABA readable; returned id, name, timezone_id. Requested currency was not returned |
| Phone inventory | GET /v26.0/{WABA-ID}/phone_numbers | One phone returned, matching configured phone ID. Paging cursors present, no next page |
| Default profile | GET /v26.0/{Phone-Number-ID}/whatsapp_business_profile | HTTP 200; messaging_product only |
| Selected profile fields | Same GET with about,address,description,email,profile_picture_url,websites,vertical | HTTP 200; returned vertical and messaging_product only. Other requested fields were not returned |
| Subscription list | GET /v26.0/{WABA-ID}/subscribed_apps | One subscribed app; ID matches configured App. No subscription mutation |

The captured phone has platform_type=CLOUD_API, quality_rating=GREEN and code_verification_status=NOT_VERIFIED. Also returned: verified_name, display_phone_number, id and throughput.level. Private names/numbers/IDs and throughput value are redacted in repository fixtures. No separate display-name status or registration/connectivity field was returned. Do not translate NOT_VERIFIED into a registration/connection diagnosis or trigger registration; no such action is authorized.

Business Portfolio discovery was not needed for the explicitly supplied WABA and was not attempted. The verified relationships are configured App -> subscribed WABA -> configured phone. Portfolio ownership, complete company-wide WABA inventory and any other assets remain UNVERIFIED. WABA name/timezone and phone values were observed at runtime; public artifacts preserve structure using aliases. No timezone_id-to-IANA mapping was invented.

## Version and evidence classification

v26.0 is now VERIFIED_ON_COMPANY_ACCOUNT for the exact successful read operations above, using the verified SYSTEM_USER credential. It is not a claim that all WhatsApp capabilities or future writes are compatible. No version downgrade occurred.

Official Postman documents the read surfaces and profile field-selection example (M32-M48/M53). The selected account responses prove behavior for this scope, but current authoritative release/lifecycle and complete permission/webhook specifications remain unavailable. Contract classification remains DOCUMENTED_OFFICIAL_SURFACE plus bounded VERIFIED_ON_COMPANY_ACCOUNT; no operation is automatically promoted to VERIFIED_CURRENT_CONTRACT solely from a successful request. The operator-provided July 29, 2026 release date remains unverified in this environment.

Observed header names include x-fb-request-id, x-fb-trace-id, facebook-api-version, x-app-usage on debug, and x-business-use-case-usage on asset reads. Presence is VERIFIED_ON_COMPANY_ACCOUNT; quota meanings, resets and global availability remain NEEDS VERIFICATION. Only safe correlation IDs and header names were retained, not request headers or usage values.

## Webhook contracts remain independent

GET verification uses the proposed hub.mode / hub.verify_token / hub.challenge mechanism. POST authenticity uses the proposed X-Hub-Signature-256 / HMAC-SHA256 over exact raw bytes using the App Secret. They are different mechanisms and credentials. No signature handler, callback endpoint, challenge test or live notification ingestion was implemented or exercised.

On 2026-09-11, current Graph Webhooks setup, WhatsApp overview and debug-token reference still returned HTTP 429. Official-domain indexed searches provided no usable current representation. The official Horizon page is adjacent-product evidence only; third-party mirrors were excluded as primary evidence. See M49-M51 and the follow-up M52 in the [research register](21-research-register.md#gate-b-live-read-verification-2026-09-11). Missing Verify Token is a configuration observation, not the reason that authoritative signature evidence is missing.

## Sanitized fixtures and provenance

Four real CAPTURED_SANITIZED fixtures were reviewed and added to [the fixture directory](../backend/testdata/meta/README.md): gate-b-v26-waba.json, gate-b-v26-phones.json, gate-b-v26-profile.json and gate-b-v26-subscriptions.json. The original multi-entry.json remains SYNTHETIC.

Each capture records actual Graph version/UTC/GET surface/status, requested fields, paging presence, safe correlation IDs, sanitization and reviewer. Asset IDs use stable aliases: asset_1 is the configured App, asset_2 the WABA, asset_3 the phone. All private names, numbers, links, profile values and cursor strings were redacted; shapes/types and selected non-PII provider states were preserved. Their sanitized bytes cannot serve as an original signed webhook fixture.

The manifest records sanitized SHA-256 values. Original raw-response SHA-256 values and the mapping to sanitized artifacts remain in ignored operator evidence; raw response bodies were not retained. A runtime exclusion check verified that none of the actual three secrets or configured private asset identifiers appears in the artifacts. No token-debug body was added as a public fixture. This is structural/redaction review by Codex, not a claim of independent human acceptance.

## Gate decision and continuation

| Requirement | Current status |
| --- | --- |
| Configuration and read authentication | PASS for access token/App Secret; Verify Token empty |
| WABA / phone / profile / subscription reads | PASS for supplied WABA and its one phone; direct workflow requires no observed BM scope |
| Complete portfolio ownership/inventory | UNVERIFIED; no portfolio query needed for supplied-WABA workflow |
| Production token strategy | PASS for actual SYSTEM_USER type, App/scopes and target reads; expiry and rotation recorded |
| v26.0 account compatibility | PASS for tested GETs only; authoritative lifecycle/current-contract review OPEN |
| GET challenge / POST authenticity evidence | OPEN; current primary contract unavailable |
| Sanitized account fixtures | PASS, four captures with reviewed provenance |
| No secrets / no mutation / no sending | Maintained; twenty GETs only |
| Gate B | CLOSED |
| Sprint 2 | NOT STARTED; separate review required even after Gate B approval |

Historical attempts on 2026-09-10 and earlier on 2026-09-11 had missing configuration and no live requests. Those missing-file observations are superseded by this successful account probe; source access limitations remain. Accepted Sprint 1 hosted CI and SMTP evidence are unchanged and are not reused as Meta integration tests.

Continue Gate B with adequate current version/webhook evidence; the intended System User credential is now verified. Keep credentials in protected files; do not print secret prefixes, put secrets in argv, broaden scopes without operation-specific need, or use a temporary USER token as the production strategy. Rotation must validate replacement scope/assets before switching protected sources and revoking the old credential; audit only actor, generation, timestamps and sanitized outcome. Any future recoverable database secret follows [ADR 006](../ADR/006-secret-encryption.md); no raw Meta token is stored in PostgreSQL.

No WhatsApp message was sent. No WABA subscription, callback override, phone registration, business profile or template was changed. Gates C/D remain CLOSED.

Latest replacement check: Meta reports SYSTEM_USER. The four sanitized fixtures and manifest were refreshed from the 12:21 UTC responses with matching timestamps, type and checksums. Earlier USER-token request/provenance evidence remains in ignored operator history; no old capture is relabeled as a System User response.
