# Dispatch: Document & Claims Inventory Explorer

## Identity
- Subagent: teamwork_preview_explorer_survey_1
- Type: teamwork_preview_explorer
- Role: Doc Inventory & Claims Miner
- Working Directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_1
- Authoritative Request: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md

## Objective
Scan and inventory all `.md` and `.docx` files in `/Users/shakhzod/Desktop/V.O.I.D`. Extract the key documentation categories, claimed architecture, parity matrices (especially `ROLE_ROW_PARITY_MATRIX.md`), program guides (`GLOBAL_SCALE_PROGRAM.md`, `GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`), context phase docs (`pegasusX/context/*`), memory docs (`.agents/memory/*`), and root docs.

Identify every specific claim of "Wired", "Done", "Production-Ready", "Cloud-Ready", or specific architectural components.

Write your findings to:
- `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_1/doc_inventory_report.md`
- `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_1/handoff.md`
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
