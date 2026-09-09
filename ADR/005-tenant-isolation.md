# ADR 005: shared schema with enforced tenant boundaries

Use organization_id on all business tables, composite foreign keys, explicit application scope checks and FORCE RLS under non-owner runtime DB roles. Global users/auth and permission catalog are narrowly documented exceptions. [PostgreSQL row security](https://www.postgresql.org/docs/current/ddl-rowsecurity.html).

Database-per-organization provides a stronger deployment boundary but complicates backups/migrations and cross-organization user identity for the initial single-company platform. Schema-per-tenant has similar operational overhead and does not eliminate privilege risk. A shared schema is acceptable only with direct-SQL isolation tests and properly constrained roles.

Use SET LOCAL inside each transaction to avoid pooled session tenant leakage. Workers enter explicit tenant context; narrowly privileged routing resolves trusted app bindings. Cache keys, object URLs, exports, notifications, SSE and search enforce the same boundary.

Initial external Meta app/WABA ownership is exclusive to one internal organization; app-to-WABA relations are many-to-many inside it. Shared external apps across internal tenants require a future routing/envelope-isolation ADR. Revisit physical separation for legal or customer requirements, not merely because a second team is added.
