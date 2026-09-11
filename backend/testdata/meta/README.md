# Meta fixture contract

Fixtures in this directory are never live requests. Each fixture has provenance in manifest.json: SYNTHETIC or CAPTURED_SANITIZED, original official reference where actually inspected, inspection/capture date, Graph version (null if unknown), parser purpose and redaction notes. Captured fixtures additionally require reviewer and source checksum in the restricted evidence register.

The initial multi-entry.json is SYNTHETIC structural data, not an official or account-verified payload. IDs/names/text are fictional; no actual phone, customer content, token, signature or app secret. Preserve envelope/batch nesting and typed values during later sanitization. Unknown extension fields must remain available for compatibility tests. Sign test bytes using a synthetic test secret only after the current signature contract is verified; never commit live signature headers.

A fixture does not enable a parser/adapter or open Gate B. Sprint 2 must acquire current contract/account evidence before relying on exact fields or limits.


## Gate B account captures, 2026-09-11

gate-b-v26-waba.json, gate-b-v26-phones.json, gate-b-v26-profile.json and gate-b-v26-subscriptions.json are CAPTURED_SANITIZED from successful v26.0 GET responses on the operator's account. Each has a capture metadata wrapper and a response body; tests should select response, not treat the wrapper as Graph JSON. Private IDs use stable aliases; names, numbers, profile values, links and cursors are replaced while types/shape are retained. Selected observed non-PII state values are preserved. These are not original signed webhook bytes.

manifest.json records sanitized checksums and Codex redaction review. Original-response checksums and their capture mapping are in ignored restricted operator evidence. The refreshed 12:21 UTC captures use the verified SYSTEM_USER credential; successful reads do not close Gate B or validate the version lifecycle. No template, subscription, profile, registration or message mutation was performed. See [Gate B evidence](../../../docs/25-gate-b-evidence.md).
