# Gate C offline evidence fixtures

Executable specification and sanitized research evidence, not Sprint 3 product code. Gate C CLOSED; Sprint 3 NOT STARTED. [Decision and sources](../../27-gate-c-evidence.md).

| File | Evidence boundary |
| --- | --- |
| source-manifest.json | Official source identities, hashes, times and quality; no signed CDN queries |
| account-evidence.json | Six bounded read-only Graph operations; currency omitted, timezone observed; zero sends/mutations |
| template-inventory.json | Five templates with aliases/hashes; exact IDs/names/content private |
| indonesia-rates.json | 20 actual official numerical/policy rows; USD/IDR alternatives, no company publication |
| indonesia-tiers.json | 36 July tier rows, all intervals/amounts compared with October PDF |
| registry-rehearsal.json | Real-artifact process; DRAFT, preview diff, pending review/publication |
| pricing-cases.json | 39 synthetic decisions A–AL including F1/F2; no actual approvals/recipient/budget |
| check_spec.py | Offline Decimal model; no product imports, database or network |

Run from repository root:

- python docs/fixtures/gate-c/check_spec.py
- python scripts/check.py
- git diff --check

Actual execution on 2026-09-14: 39 scenarios PASS; 20 decimal rate rows and 36 contiguous tier rows PASS. Separate original PDF layout-text comparison: 36 October tier rows match July CSV. No visual PDF verification claimed. Repository source/link/migration check PASS (156 local links, immutable migration checksums, UTF-8/fences and obvious-secret patterns); OpenAPI route coverage also passes that checker. git diff --check PASS. Forty source hashes match retained original bytes; exact loaded Meta credentials are absent from all Gate C artifacts. These are bounded scans, not a universal secret-detection guarantee.

Model assumptions: currency, consent, authority, review freshness, template eligibility, FEP qualification, complete allowance usage, confirmation and budget defaults are synthetic. The model exercises expected decisions; it does not implement the full temporal policy/ledger or prove concurrency. Its fixed October offset represents only the observed 2026 transition; product code must use the IANA WABA timezone. Scenario J is a synthetic increase. Allowance proof stands for complete authoritative usage plus safe reservation, never merely this application's counter.

Rate publication and TEST budget are disabled; review deadline is proposed. PostgreSQL tenant/concurrency/security and real sends remain future Sprint 3 work after authorization. No hosted workflow ran for this local documentation package; prior hosted Sprint 2 success is separate baseline evidence.
