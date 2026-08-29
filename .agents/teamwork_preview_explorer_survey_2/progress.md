# Progress — Explorer 2 (Backend & Contracts SoT Inspector)

Last visited: 2026-08-20T17:28:15+05:00

## Current Status
- Completed comprehensive investigation of Spanner DDL (3,648 lines, 220+ tables).
- Audited all 29 route packages and domain controllers in `pegasusX/apps/backend-go/`.
- Inspected contracts (`events.schema.json`, OpenAPI specs, marker registries), `packages/types/` (6,682 lines), `packages/api-client/` (3,669 lines), and Quicktype stubs across Android and iOS.
- Ran full backend test suite (`go test ./...`), cataloged passing vs failing suites, and identified root cause citations.
- Pinpointed exact unwired endpoints, feature flags, and gated flows with `file:line` citations.
- Compiling full findings report to `backend_sot_report.md` and writing `handoff.md`.


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
