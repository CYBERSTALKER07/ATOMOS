# Dispatch: Backend & Contracts Codebase SoT Inspector

## Identity
- Subagent: teamwork_preview_explorer_survey_2
- Type: teamwork_preview_explorer
- Role: Backend & Data Plane SoT Inspector
- Working Directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_2
- Authoritative Request: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md

## Objective
Investigate the authoritative backend source of truth in `pegasusX/apps/backend-go/`, `pegasusX/schema/spanner.ddl`, `pegasusX/contracts/`, `pegasusX/packages/types/`, `pegasusX/packages/api-client/`, and `cmd/ssmr-smokecheck/`.

Determine what is genuinely implemented:
1. Spanner tables, columns, indexes, outbox patterns.
2. Go packages, routes (`*routes/routes.go`), services, repositories, transactions.
3. Contracts & generated schemas (`contracts/events.schema.json`, `packages/types/`, etc.).
4. Test suites (`*_test.go`), SSMR smoke checks, passing vs failing tests.
5. Identify areas where implementation is missing, partial, or stubbed.

Write your findings with exact `file:line` citations to:
- `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_2/backend_sot_report.md`
- `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_2/handoff.md`
- Update your `progress.md` with liveness.


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
