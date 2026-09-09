# ADR 001: modular monolith

One Go codebase and PostgreSQL schema with explicit business modules; run API and worker processes independently. Next.js is a separate rendering process but uses the same Go business API.

Alternatives: microservices would make consent, approval and budget checks distributed transactions before team/scale require them. A single unstructured service would obscure ownership and encourage bypasses. A monolith with clear module services permits atomic control checks without a network of internal APIs.

Consequence: enforce module boundaries in code review; only messaging dispatch owns Graph send access. Transactions may coordinate several modules explicitly. Workers can scale separately, but schema releases remain coordinated. No generic repository, global service locator or speculative event-sourced model.

Revisit when a module has independently measured scaling/availability needs and stable ownership that justify the operational cost. Splitting modules is not a prerequisite for multi-user or multi-WABA support.
