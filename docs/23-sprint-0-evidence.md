# Sprint 0 engineering evidence

Historical owner override (2026-09-09): **Sprint 0 IMPLEMENTATION COMPLETE; Gate A APPROVED; Sprint 1 AUTHORIZED. Hosted CI verification PENDING - EXTERNAL BLOCKER.** Run 34356265759 was attempted and failed before repository steps because GitHub reported an account billing lock. Accepted local evidence closes implementation, not hosted verification. CI requirements remain intact; rerun hosted CI when available, record the real result and fix any repository failures. Gates B/C/D remain CLOSED.

Current verification (2026-09-10): **Sprint 0 IMPLEMENTATION COMPLETE; Gate A APPROVED; Sprint 1 implementation COMPLETE, acceptance OPEN.** GitHub run 34356265759 attempt 3 actually executed and passed the Sprint 0 baseline at commit 993c67e93fe5945ec820efe67d9d3abf3df8b1b3. Its original attempt 1 failed before steps due to the account billing restriction. Sprint 1 hosted verification is PENDING for these unpublished workspace changes; do not treat the Sprint 0 pass as Sprint 1 evidence. Controlled SMTP was ATTEMPTED and rejected authentication (SMTP_AUTH), so mailbox receipt remains PENDING. Gates B/C/D remain CLOSED.

This is the historical Sprint 0 closure record. Subsequent Sprint 1 code and verification are recorded separately in [Sprint 1 evidence](24-sprint-1-evidence.md); statements below about absent identity features apply to the Sprint 0 baseline.

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
| Backend configuration, HTTP and SMTP unit tests | PASS on Windows; PASS on Linux with race detection; synthetic SMTP tests use TLS/auth/NOOP and never send mail |
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
| Hosted GitHub Actions | Attempt 1: external billing failure before steps. Attempt 3: PASS with actual executed steps at the same Sprint 0 commit; see subsequent verification below |
| Controlled provider SMTP mailbox | NOT RUN: provider/sender credentials and designated mailbox not supplied |
| Product concurrency/load and restore drills | NOT IMPLEMENTED / NOT RUN; future sprint gates |

The Linux verification used golang:1.27.1-bookworm at sha256:648f440f42a0958804efb24df176f806f9d353b41f1c0627f666428e40310f6b, a read-only source mount and isolated test-database configuration. The benchmark observation SQL also executed successfully against an idle database; zero observed rows is not a load or throughput result. A transient Windows npm ci file-lock failure was retried successfully before the final complete check.

The initial integration failure showed that a SELECT-only authorizer RLS policy could not see rows for SHARE locking. The corrected narrow visibility policy permits locking and denies writes with WITH CHECK false; runtime bypass was not introduced. A real ESLint 10/plugin API mismatch was corrected with the official compatibility package; lint rules were not disabled to hide the error.

## Hosted CI inspection, 2026-09-09

The original attempt of the hosted [Sprint 0 foundation run](https://github.com/mbingsdk/whats/actions/runs/34356265759) was inspected through authenticated GitHub CLI/API, including run metadata, jobs, commit check runs and check annotations. The hosted result is an attempted failure caused by an external restriction.

| Evidence | Observed value |
| --- | --- |
| Repository / branch | mbingsdk/whats / main |
| Commit | 993c67e93fe5945ec820efe67d9d3abf3df8b1b3 (matches local HEAD) |
| Workflow / trigger | .github/workflows/ci.yml / push |
| Run ID / status / conclusion | 34356265759 / completed / failure |
| Created / updated UTC | 2026-09-09T13:18:40Z / 2026-09-09T13:18:44Z |
| Job/check ID | 102481617622, foundation |
| Job timestamps UTC | 2026-09-09T13:18:40Z to 2026-09-09T13:18:43Z |
| Job steps | Empty list; no build, dependency installation or test step ran |
| Failure classification | EXTERNAL_ACCOUNT_BILLING_RESTRICTION; not an observed repository test/code failure |

GitHub's [check annotation](https://api.github.com/repos/mbingsdk/whats/check-runs/102481617622/annotations) states:

> The job was not started because your account is locked due to a billing issue.



## Historical reconciliation validation, 2026-09-09

After inspecting the hosted failure, this review changed only README.md, AGENTS.md, PLANS.md and docs 19, 20, 22 and 23. The existing filename remains docs/23-sprint-0-evidence.md; no duplicate evidence document was created for the alternate spelling in the request.

Executed again on the local Windows environment: python scripts/dev.py up (healthy PostgreSQL); python scripts/dev.py check (exit 0: source/format/vet, Go unit results PASS with cached packages, all nine PostgreSQL integration subtests PASS with -count=1, Go build, npm ci, lint, typecheck, two frontend tests PASS with zero skips/failures, OpenAPI validation and production Next build); go mod verify (all modules verified); pinned govulncheck 1.8.0 (no vulnerabilities); npm audit --audit-level=high (zero vulnerabilities). This rerun did not use the Linux race detector; the separate Linux race evidence above is from the earlier implementation verification.

The final source validator passed 69 local links, UTF-8/fences, the obvious-secret scan and unchanged migration checksums. git diff --check passed. A path-scoped diff confirmed no changes to workflow, migration SQL, backend, frontend, scripts or dependency locks. The latest hosted run was checked again and remained completed/failure with the same commit. Local checks do not satisfy the hosted CI requirement.

## Scope and outstanding acceptance

Implemented foundation: API bootstrap/config/safe logging/request IDs/body/time limits/graceful shutdown; /healthz and /readyz; schema/role checks; transactional membership scope; migrations and isolation tests; tiny Next shell/API read boundary/error component; validated contract; local runner and CI definition; SMTP probe/config; synthetic fixture conventions.

At Sprint 0 closure, no authentication/login/invitation feature, tenant HTTP CRUD, inbox, campaigns, contacts, templates, automation, Meta adapter or paid sending was implemented. No real VPS or Meta asset was changed. Local PostgreSQL contains development data only; test databases are disposable.

## SMTP configuration and controlled-mailbox procedure

Select an ordinary authenticated SMTP relay; configure SMTP_HOST, SMTP_PORT, SMTP_TLS_MODE (tls or starttls), SMTP_SENDER, SMTP_USERNAME, SMTP_PASSWORD_FILE and SMTP_TIMEOUT using protected configuration. TLS verifies hostname/certificate and requires TLS 1.2 or later; plaintext and opportunistic fallback are rejected. Production requires the complete configuration. Local development may omit SMTP entirely until this check is needed.

Run backend/cmd/smtpcheck with the same validated app configuration and protected secret files. It performs connection, TLS, authentication, NOOP and QUIT, with bounded timeout/cancellation; it issues no MAIL, RCPT or DATA command. Its success establishes relay acceptance only, not inbox delivery. SMTP is not a dependency of /healthz or a restart trigger.

For delivery evidence, the operator must designate an approved sender and controlled recipient, then use the configured relay's standard SMTP/mail client to send one nonsecret test marker. Record UTC time, sanitized delivery/Message-ID and confirmed receipt (and any bounce); never include credentials or customer data. No controlled provider email was sent for Sprint 0. Sprint 1 now exercises actual identity mail through a local test relay; the [current controlled-mailbox procedure](16-deployment.md#sprint-1-identity-operations) requires the implemented outbox and actual identity flows.

## Dependency and source controls

package-lock.json and go.sum freeze dependency integrity. npm install lifecycle scripts are disabled; required platform binaries are supplied as registry packages, verified by actual lint/build. ESLint 10 is the invoked engine; an upstream transitive ESLint 9 deprecation may still be reported by npm. Audit results, rather than that warning alone, determine known vulnerability findings. The official compatibility package keeps the existing Next/React rules active.

The source checker verifies local links, UTF-8/fences and reviewed migration hashes, and rejects high-confidence credential patterns or tracked private local/environment files without printing values. This is an obvious-secret baseline, not a guarantee to recognize arbitrary credentials. GitHub Actions pins action commits with read-only repository permission and does not persist checkout credentials. Runtime, migration and test-administrator environments are separated in the runner; administrative test URLs never enter the API environment.

## Source inventory and document changes

The original pre-publication Sprint 0 source inventory contained 87 files, excluding local credentials/build outputs, dependency installations and caches: 50 new foundation files alongside the original 37-document package. That inventory was recorded before Git publication. The current workspace is on main with origin https://github.com/mbingsdk/whats.git and baseline commit 993c67e93fe5945ec820efe67d9d3abf3df8b1b3; the closure review began with a clean working tree.

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

## Subsequent hosted verification inspected 2026-09-10

Authenticated GitHub API inspection found that run 34356265759 was rerun. Attempt 3 is completed/success for the same baseline commit 993c67e93fe5945ec820efe67d9d3abf3df8b1b3. Job 102491330281 ran 2026-09-09T13:45:36Z to 2026-09-09T13:47:22Z; the run updated at 13:47:23Z. Setup, PostgreSQL startup, clean/repeated migration, source/Go/PostgreSQL/frontend/OpenAPI checks, dependency checks and cleanup all have success conclusions. [Attempt 3](https://github.com/mbingsdk/whats/actions/runs/34356265759/attempts/3) is actual hosted Sprint 0 evidence.

The attempt-1 jobs endpoint was also rechecked: job 102481617622 remains failure with an empty steps array. Its external account/billing failure is preserved above. This new evidence resolves the historical Sprint 0 hosted-verification blocker; it does not rewrite the failed attempt as a pass and does not cover unpublished Sprint 1 changes. No CI check was weakened.
