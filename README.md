# WABA Control

Acceptance review (2026-09-14; live evidence captured 2026-09-12): **Gate B APPROVED; Sprint 2 implementation COMPLETE; Sprint 2 acceptance COMPLETE; Gates C/D CLOSED.** Real Meta GET challenge, authentic signed POST, durable ingestion, processing, authorized Event Center and replay evidence are verified. Sprint 0/1 acceptance remains COMPLETE. No outbound WhatsApp or Sprint 3 work is authorized.

Historical owner override (2026-09-09): **Sprint 0 IMPLEMENTATION COMPLETE; Gate A APPROVED; Sprint 1 AUTHORIZED. Hosted CI verification PENDING - EXTERNAL BLOCKER.** Run 34356265759 was attempted and failed before repository steps because GitHub reported an account billing lock. Accepted local evidence closes implementation, not hosted verification. CI requirements remain intact; rerun hosted CI when available, record the real result and fix any repository failures. Gates B/C/D remain CLOSED.

Historical Sprint 1 verification (2026-09-10): **Sprint 0 IMPLEMENTATION COMPLETE; Gate A APPROVED; Sprint 1 IMPLEMENTATION COMPLETE; HOSTED CI VERIFIED; Sprint 1 acceptance COMPLETE.** Sprint 1 was published at d4fed8ce7b50ed44f0f356e9ebea8f21695fe7bd. Run 34436595874 executed and failed in scripts/check.py because the 00002 manifest hashed CRLF working bytes while Git published LF bytes; this was a repository failure, not billing. Corrective commit 9b2c67187eea6fc2ee6b5a1929f063e76cc6797b passed the full hosted workflow in run 34437626714. Published SQL and historical manifest remain unchanged. Sprint 0 run 34356265759 attempt 1 remains a historical billing failure and attempt 3 passed only the Sprint 0 baseline. Controlled Brevo SMTP accepts all four identity emails; the operator confirms actual receipt of verification, reset, security and invitation emails. Gates B/C/D CLOSED; no Sprint 2 work. Detailed evidence is in docs/24-sprint-1-evidence.md.

Historical Phase A authorization (2026-09-10; superseded by the owner decision above): current Meta contract and protected read-only account verification are authorized. Gate B remains CLOSED for current supported-version/webhook-contract evidence; account verification is limited to the tested GETs; Phase B is conditional and NOT STARTED. See [Gate B evidence](docs/25-gate-b-evidence.md) and [Sprint 2 status](docs/26-sprint-2-evidence.md). No outbound WhatsApp or Sprint 3 work is authorized.

Historical Gate B-only continuation (2026-09-11; superseded by the owner decision above): operator configuration is now PRESENT; v26.0 token/WABA/phone/profile/subscription GETs passed for one supplied WABA and phone. Meta verifies SYSTEM_USER, matching App, WA/WM scopes and no BM; all target GETs pass. Token expires 2026-11-10T12:20:59Z. Verify Token is empty and current authoritative webhook/lifecycle evidence remains unresolved. Gate B CLOSED; stop for separate review before Sprint 2 even if later approved.

Normal deployment serves one PT in one company Organization. The design retains organization isolation, employees in multiple teams, multiple Meta apps, WABAs and phone numbers; an organization switcher appears only for users with multiple active memberships. There is no public organization registration, SaaS subscription billing, tenant marketplace or generic onboarding wizard. Every sending path uses the same authorization, consent, service-window, pricing, approval and budget controls.

## Read this first

1. [Product vision](docs/00-product-vision.md) and [master PRD](docs/01-prd-master.md).
2. [Meta capability matrix](docs/02-meta-capability-matrix.md): documented capabilities, inaccessible references and account verification gates.
3. [Architecture](docs/03-system-architecture.md), [domain states](docs/04-domain-model.md), [database](docs/05-database-design.md), [API](docs/06-api-contract.md).
4. [Decision records](ADR/README.md), [delivery plan](docs/19-sprint-plan.md) and [implementation gates](PLANS.md).

## Specification map

| Area | Document |
| --- | --- |
| Live application | [Realtime](docs/07-realtime-events.md), [inbox and assets](docs/09-messaging-and-inbox.md) |
| Reliable processing | [Webhooks and background jobs](docs/08-webhook-architecture.md), [campaigns](docs/10-campaign-engine.md), [automation](docs/13-automation-engine.md) |
| Sending controls | [Consent](docs/11-consent-and-compliance.md), [pricing and budgets](docs/12-pricing-and-budget-guard.md), [security and RBAC](docs/14-security-and-rbac.md) |
| Operations | [Observability and analytics](docs/15-observability.md), [deployment and recovery](docs/16-deployment.md), [tests](docs/17-testing-strategy.md) |
| Product execution | [UI specification](docs/18-ui-ux-product-spec.md), [sprints](docs/19-sprint-plan.md), [release checklist](docs/20-production-readiness-checklist.md) |
| Evidence | [Research register](docs/21-research-register.md), [second-pass review](docs/22-architecture-review.md) |

## Status and limits

**Design baseline, with explicit implementation gates.** Official policy, public pricing overview and Meta's official Postman examples were inspected. Many developer pages returned HTTP 429, login-only content or fetch failures. The Postman examples include old Graph versions and contradictory prose; they prove API surface, not compatibility with an untested 2026 account.

Read-only Meta account checks now pass for the supplied WABA/phone on v26.0; see the Gate B evidence record. Current rate cards, upcoming pricing changes, Graph version, permissions on the company's assets and advanced feature eligibility remain NEEDS VERIFICATION. The identity application works locally; the full WhatsApp product is not implemented or production-ready. No real prices, credentials, customers or fabricated analytics are included.

## Implemented identity application

Sprint 1 provides invitation-only company access, authentication, teams, scoped permissions, MFA, account security, audit and durable SMTP delivery. Implementation verification is recorded in [Sprint 1 evidence](docs/24-sprint-1-evidence.md). Sprint 1 acceptance is COMPLETE with all four controlled SMTP messages confirmed received; hosted CI verification passed for corrective commit 9b2c671 in run 34437626714. Gate B is now APPROVED for Sprint 2.

## Local workflow

Install Go 1.27.1, Node 24.21.0 (npm 11.19.0), Python 3 and Docker Compose. With fnm installed, use fnm install 24.21.0; the local runner selects it. From the repository root:

```text
python scripts/dev.py up
python scripts/dev.py migrate
python scripts/dev.py backend
```

In another terminal run npm ci with the pinned Node version, then python scripts/dev.py frontend. Open http://localhost:3000; /healthz and /readyz go to the Go backend through the development proxy. Run python scripts/dev.py test for Go/unit/real-PostgreSQL tests, or python scripts/dev.py check for the complete local validation sequence including frontend production build and OpenAPI. Stop PostgreSQL with python scripts/dev.py down; its volume remains.

The runner generates ignored local credentials. Never copy them to production. The API receives the limited runtime URL plus the dedicated identity-service URL and root-key file; it receives no migration/setup-administrator credentials; test setup uses separate disposable-database administrative access. SMTP configuration names are in [.env.example](.env.example). No automatic .env loading is used by Go; use the runner or explicit protected environment/files.

Use python scripts/dev.py bootstrap with protected operator inputs to initialize the first Owner, and python scripts/dev.py mailworker to process identity mail. See the exact [bootstrap and mail procedure](docs/16-deployment.md#sprint-1-identity-operations). Browser verification uses npx playwright install chromium followed by python scripts/dev.py e2e against a disposable database and local TLS SMTP server.

[Database workflow](database/README.md), [deployment foundation](deploy/README.md), [OpenAPI](contracts/openapi.yaml) and [Sprint 0 evidence/status](docs/23-sprint-0-evidence.md) describe actual scope and limitations.

## Sprint 2 Meta operations

Read-only asset sync and signed webhook ingestion pass local implementation verification. Hosted Sprint 2 CI passed for implementation commit e0a6550 in run [34670353309](https://github.com/mbingsdk/whats/actions/runs/34670353309). Real Meta GET/POST acceptance is VERIFIED; the required GET metadata correction passed its own hosted [run 34813315965](https://github.com/mbingsdk/whats/actions/runs/34813315965). The temporary acceptance callback was rolled back and the tunnel stopped. See [Sprint 2 evidence](docs/26-sprint-2-evidence.md). Set the protected Meta variables in [.env.example](.env.example), bind the configured App using Meta Connection, and run python scripts/dev.py metaworker alongside the API. Startup never changes a Meta subscription or callback. The worker performs GET asset reads and local event processing only.
