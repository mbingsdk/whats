# Meta fixture contract

Fixtures in this directory are never live requests. Each fixture has provenance in manifest.json: SYNTHETIC or CAPTURED_SANITIZED, original official reference where actually inspected, inspection/capture date, Graph version (null if unknown), parser purpose and redaction notes. Captured fixtures additionally require reviewer and source checksum in the restricted evidence register.

The initial multi-entry.json is SYNTHETIC structural data, not an official or account-verified payload. IDs/names/text are fictional; no actual phone, customer content, token, signature or app secret. Preserve envelope/batch nesting and typed values during later sanitization. Unknown extension fields must remain available for compatibility tests. Sign test bytes using a synthetic test secret only after the current signature contract is verified; never commit live signature headers.

A fixture does not enable a parser/adapter or open Gate B. Sprint 2 must acquire current contract/account evidence before relying on exact fields or limits.
