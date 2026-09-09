# WABA Control

Architecture and product specification for the company's WhatsApp operations platform. Meta research dates: **2026-09-08 to 2026-09-09**; design/review completed **2026-09-09**. Gate A is approved and Sprint 0 foundation is implemented locally; Sprint 0 remains OPEN pending the evidence listed in docs/23. Gates B/C/D remain CLOSED. No product workflows or Meta adapters are enabled.

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

No live Meta account was queried. Current rate cards, upcoming pricing changes, Graph version, permissions on the company's assets and advanced feature eligibility remain NEEDS VERIFICATION. Do not describe this repository as a working or production-ready system. No real prices, credentials, customers or fabricated analytics are included.

Sprint 0 alone is authorized under [PLANS.md](PLANS.md). Unverified optional modules stay disabled; foundational work need not wait for Calling or Coexistence.

## Local foundation workflow

Install Go 1.27.1, Node 24.21.0 (npm 11.19.0), Python 3 and Docker Compose. With fnm installed, use fnm install 24.21.0; the local runner selects it. From the repository root:

```text
python scripts/dev.py up
python scripts/dev.py migrate
python scripts/dev.py backend
```

In another terminal run npm ci with the pinned Node version, then python scripts/dev.py frontend. Open http://localhost:3000; /healthz and /readyz go to the Go backend through the development proxy. Run python scripts/dev.py test for Go/unit/real-PostgreSQL tests, or python scripts/dev.py check for the complete local validation sequence including frontend production build and OpenAPI. Stop PostgreSQL with python scripts/dev.py down; its volume remains.

The runner generates ignored local credentials. Never copy them to production. The API receives only the runtime URL; test setup uses separate disposable-database administrative access. SMTP configuration names are in [.env.example](.env.example). No automatic .env loading is used by Go; use the runner or explicit protected environment/files.

[Database workflow](database/README.md), [deployment foundation](deploy/README.md), [OpenAPI](contracts/openapi.yaml) and [Sprint 0 evidence/status](docs/23-sprint-0-evidence.md) describe actual scope and limitations.
