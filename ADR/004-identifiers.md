# ADR 004: internal and external identifiers

Generate [UUIDv7](https://www.rfc-editor.org/rfc/rfc9562.html) internally, with unique constraints and a vetted implementation selected at Sprint 0. UUIDs avoid coordination between API/workers; temporal locality is useful for insertion. UUIDv4 is acceptable if the selected dependency lacks a trustworthy v7 implementation; no custom UUID algorithm. Ordered IDs are not authorization tokens or event commit cursors.

Alternatives: global bigserial is compact but couples creation to the DB and exposes sequence; UUIDv4 has random index locality; ULID adds a different encoding/ecosystem without present need. All would work at initial scale; correctness comes from constraints, not identifier fashion.

Store Meta app/WABA/phone/message IDs as opaque text. Contact identifiers include kind and namespace; verified normalized E.164 is nullable. Do not force future business-scoped identifiers into phone strings or assume a hidden phone implies known pricing market.

SSE ordering uses a separately committed stream sequence. Thread unread uses local thread sequence. Neither relies on UUID timestamp order across transactions.
