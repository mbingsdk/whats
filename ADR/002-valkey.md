# ADR 002: Valkey for disposable state

Choose Valkey for ephemeral presence, short-lived cache, throttling coordination and wake hints. PostgreSQL retains durable jobs, authorizations, frequency exposure and money. [Valkey project FAQ](https://valkey.io/topics/faq/).

Redis is viable if it is the company's supported operational standard. Valkey is preferred for its project/licensing fit and compatibility with the narrow required commands, not an unmeasured speed claim. Confirm maintained Go client compatibility and exact release/license in Sprint 0; do not assume every Redis module is available.

Losing Valkey loses presence, not messages or campaign progress. Durable workers rediscover DB rows. If distributed throttle capacity cannot be enforced during outage, pause affected sends or use an explicitly tested conservative DB fallback. Never reconstruct budget balances from an ephemeral cache.

Revisit if a required supported client/managed platform materially favors Redis. No Valkey cluster initially.
