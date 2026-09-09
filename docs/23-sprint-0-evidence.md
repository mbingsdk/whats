# Sprint 0 engineering evidence

Gate A approved by product owner on 2026-09-09; D2/D5 accepted as provisional testing/recovery assumptions. Sprint 0 only is authorized. Gates B/C/D remain CLOSED. This record distinguishes implemented foundation from future architecture and pending evidence.

## Toolchain selection

| Component | Pin | Evidence inspected 2026-09-09 |
| --- | --- | --- |
| Go | 1.27.1 | [Official release feed](https://go.dev/dl/?mode=json); downloaded toolchain reports matching version |
| Node / npm | 24.21.0 LTS / 11.19.0 | [Official Node release feed](https://nodejs.org/dist/index.json); runtime installed with fnm |
| Next / React | 16.3.4 / 19.2.8 | Official npm package metadata and [Next release feed](https://nextjs.org/blog) |
| TypeScript | 6.0.3 | Registry version/peer inspection and actual Next compilation; version 7 is not required for this baseline |
| ESLint | 10.10.0 + @eslint/compat 2.1.1 | [ESLint compatibility package](https://github.com/eslint/rewrite/tree/main/packages/compat); Next's React plugin still uses removed context APIs, so compatibility adapter is required |
| PostgreSQL | 18.6 bookworm image, digest in compose.yaml | [PostgreSQL support table](https://www.postgresql.org/support/versioning/); real Docker server used |
| pgx / Goose / UUID | 5.11.0 / 3.28.0 / 1.6.0 | Official Go module metadata; [Goose provider](https://pressly.github.io/goose/documentation/provider/) and [pgx](https://github.com/jackc/pgx) |
| OpenAPI / validator | 3.1.0 / Redocly CLI 2.51.2 | Locked package plus automatic validation |
| Vulnerability checker | govulncheck 1.8.0 | Official Go module metadata; run against resolved Go dependencies |

Valkey, sqlc, Tailwind and TanStack Query are not installed: Sprint 0 has no cache, domain query generation or product design/state requirements. Their approved architectural roles remain available for the sprint that actually uses them. No fake worker, interfaces for hypothetical providers or future route directories exist.

## Verification ledger

Results are recorded after checks execute. Local Windows execution and hosted Linux CI are separate evidence; a workflow file is not a hosted CI pass.

| Check | Status |
| --- | --- |
| Backend configuration, HTTP and SMTP unit tests | PASS on Windows and Linux with race detection; synthetic SMTP tests use TLS/auth/NOOP and never send mail |
| Clean database and repeated migration | PASS on PostgreSQL 18.6; second application is a no-op |
| Two-tenant/RLS/role/pool isolation | PASS on real PostgreSQL through runtime role |
| Go vet/build | PASS on Windows and Linux |
| govulncheck | PASS: no vulnerabilities reported at inspection |
| Frontend typecheck, 2 tests and production build | PASS on Windows |
| Full local check | PASS: python scripts/dev.py check exited 0, including real PostgreSQL integration tests, npm ci, lint, typecheck, frontend tests/build, OpenAPI and source checks |
| Linux race tests | PASS: backend unit and real PostgreSQL integration tests with -race in the pinned Go container |
| Dependency integrity / npm audit | PASS: go mod verify reported all modules verified; npm audit reported 0 vulnerabilities |
| Compiled API smoke | PASS: /healthz and /readyz returned 200 and request ID propagated; API process received runtime credentials only |
| Workflow syntax | PASS: actionlint 1.7.12; this does not establish a hosted workflow result |
| Source and migration verification | PASS: 69 local links, UTF-8/fences, obvious-secret baseline and migration SHA256 manifest |
| Hosted GitHub Actions | NOT RUN: no Git repository/remote or hosted runner connection supplied |
| Controlled provider SMTP mailbox | NOT RUN: provider/sender credentials and designated mailbox not supplied |
| Product concurrency/load and restore drills | NOT IMPLEMENTED / NOT RUN; future sprint gates |

The Linux verification used golang:1.27.1-bookworm at sha256:648f440f42a0958804efb24df176f806f9d353b41f1c0627f666428e40310f6b, a read-only source mount and isolated test-database configuration. The benchmark observation SQL also executed successfully against an idle database; zero observed rows is not a load or throughput result. A transient Windows npm ci file-lock failure was retried successfully before the final complete check.

The initial integration failure showed that a SELECT-only authorizer RLS policy could not see rows for SHARE locking. The corrected narrow visibility policy permits locking and denies writes with WITH CHECK false; runtime bypass was not introduced. A real ESLint 10/plugin API mismatch was corrected with the official compatibility package; lint rules were not disabled to hide the error.

## Scope and outstanding acceptance

Implemented foundation: API bootstrap/config/safe logging/request IDs/body/time limits/graceful shutdown; /healthz and /readyz; schema/role checks; transactional membership scope; migrations and isolation tests; tiny Next shell/API read boundary/error component; validated contract; local runner and CI definition; SMTP probe/config; synthetic fixture conventions.

No authentication/login/invitation feature, tenant HTTP CRUD, inbox, campaigns, contacts, templates, automation, Meta adapter or paid sending is implemented. No real VPS or Meta asset was changed. Local PostgreSQL contains development data only; test databases are disposable.

Sprint 0 remains OPEN until outstanding definition-of-done evidence is resolved, especially a successful hosted CI run and acceptance of unresolved official Meta contract research. SMTP relay/mailbox verification must occur before identity-email rollout. Product owner assumptions are capacity inputs, not benchmark results or Meta limits. No production-readiness checkbox is completed by these foundation tests.

## SMTP configuration and controlled-mailbox procedure

Select an ordinary authenticated SMTP relay; configure SMTP_HOST, SMTP_PORT, SMTP_TLS_MODE (tls or starttls), SMTP_SENDER, SMTP_USERNAME, SMTP_PASSWORD_FILE and SMTP_TIMEOUT using protected configuration. TLS verifies hostname/certificate and requires TLS 1.2 or later; plaintext and opportunistic fallback are rejected. Production requires the complete configuration. Local development may omit SMTP entirely until this check is needed.

Run backend/cmd/smtpcheck with the same validated app configuration and protected secret files. It performs connection, TLS, authentication, NOOP and QUIT, with bounded timeout/cancellation; it issues no MAIL, RCPT or DATA command. Its success establishes relay acceptance only, not inbox delivery. SMTP is not a dependency of /healthz or a restart trigger.

For delivery evidence, the operator must designate an approved sender and controlled recipient, then use the configured relay's standard SMTP/mail client to send one nonsecret test marker. Record UTC time, sanitized delivery/Message-ID and confirmed receipt (and any bounce); never include credentials or customer data. No such email is sent by this agent or automated tests. Full invitation/verification/reset/security notification delivery is Sprint 1, behind this same provider-neutral contract.

## Dependency and source controls

package-lock.json and go.sum freeze dependency integrity. npm install lifecycle scripts are disabled; required platform binaries are supplied as registry packages, verified by actual lint/build. ESLint 10 is the invoked engine; an upstream transitive ESLint 9 deprecation may still be reported by npm. Audit results, rather than that warning alone, determine known vulnerability findings. The official compatibility package keeps the existing Next/React rules active.

The source checker verifies local links, UTF-8/fences and reviewed migration hashes, and rejects high-confidence credential patterns or tracked private local/environment files without printing values. This is an obvious-secret baseline, not a guarantee to recognize arbitrary credentials. GitHub Actions pins action commits with read-only repository permission and does not persist checkout credentials. Runtime, migration and test-administrator environments are separated in the runner; administrative test URLs never enter the API environment.

## Source inventory and document changes

The source inventory contains 87 files, excluding local credentials/build outputs, dependency installations and caches: 50 new foundation files alongside the original 37-document package. There is no Git history in this workspace, so this is a filesystem inventory rather than a Git diff.

| New source area | Files | Concrete purpose |
| --- | ---: | --- |
| Root configuration | 9 | Environment example, ignore/line-ending rules, Node/npm pins and lockfile, Compose and OpenAPI lint configuration |
| backend | 17 | Three commands, configuration, HTTP, database and SMTP boundaries, migrations runner, tests and explicitly synthetic fixtures |
| frontend | 12 | App shell/error boundary, API read client, environment/build/lint/type configuration and tests |
| database | 6 | Role bootstrap, initial SQL/checksum, migration conventions and future contention measurement procedure/query |
| contracts | 1 | Validated OpenAPI foundation |
| scripts | 2 | Local development/check runner and source/migration validator |
| deploy | 1 | VPS process, routing, secrets, migration and recovery foundation |
| .github | 1 | Single-baseline CI workflow |
| docs | 1 | This executed-evidence and acceptance record |

Existing documents maintained in Sprint 0: README.md, AGENTS.md, PLANS.md, ADR/README.md and docs 02, 03, 05, 06, 14, 16, 17, 19, 20, 21 and 22. The ADR index records acceptance; no individual ADR decision was changed during implementation. The earlier documentation correction's ADR 007 change remains part of the approved baseline.
