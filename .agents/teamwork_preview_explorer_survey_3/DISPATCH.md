# Dispatch: Client Apps & Multi-Role Parity SoT Inspector

## Identity
- Subagent: teamwork_preview_explorer_survey_3
- Type: teamwork_preview_explorer
- Role: Client Apps & Multi-Role Parity Inspector
- Working Directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_3
- Authoritative Request: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md

## Objective
Investigate all client apps across the 6 role rows in `pegasusX/apps/`:
1. Supplier: portal, Android, iOS
2. Retailer: desktop, Android, iOS
3. Driver: Android, iOS
4. Warehouse: portal, Android, iOS
5. Factory: portal, Android, iOS
6. Payload: terminal, Android, iOS

Determine the actual state of UI screens, API client integration, shared packages (`packages/types`, `packages/api-client`), WebSocket handling, state stores. Identify genuine implementations vs facades / stubs / theatre.

Write your findings with exact `file:line` citations to:
- `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_3/clients_parity_report.md`
- `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_3/handoff.md`
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
