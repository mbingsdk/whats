# ADR 008: persist before ACK

After signature validation, commit bounded encrypted raw bytes and a processing job before returning success. Parse individual domain items asynchronously, deduplicate per semantic fact/effect and retain parser provenance. Unknown payloads remain in restricted quarantine without triggering guessed effects.

ACK-before-persistence risks losing a customer message during a crash. Heavy synchronous processing delays acknowledgement and amplifies retries. Object-store-only raw ingress would create a DB/object atomicity problem on the critical path; PostgreSQL stores the initial bounded envelope. Retention can later archive/purge content explicitly.

Exact body hash handles transport duplicates; child keys handle rebatched/cross-app retries and changed status facts. Replay defaults to projections only and uses original dedupe keys, not parser-version-specific effect keys. Privacy tombstones are checked before reconstruction.

Current Meta signature/envelope limits still need official verification; no permissive parser or signature fallback is authorized. Revisit storage placement only if measured raw volume burdens PostgreSQL.
