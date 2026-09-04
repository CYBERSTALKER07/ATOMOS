## 2026-08-20T12:21:08Z

Read all `.md` and `.docx` files across the entire repository (/Users/shakhzod/Desktop/V.O.I.D), compare them to the actual codebase (source of truth, particularly pegasusX/ and actual implementation code), and update the documentation in place to align with the current implementation.

Ensure:
1. R1: Comprehensive audit of all `.md` and `.docx` files in the repository.
2. R2: Codebase synchronization — determine actual state of features, schemas, configurations. Identify discrepancies where docs claim "done" or "wired" without implementation.
3. R3: In-place updates — correct obsolete/false claims and document new implementations accurately.
4. Maintain your plan.md, progress.md, and BRIEFING.md in your working directory.
5. Coordinate with specialists and ensure high integrity.
6. When work is complete, send a message back to report completion so independent verification can proceed.

## 2026-08-20T18:49:38Z

You are the Project Orchestrator for the repository documentation audit and synchronization task.
Workspace Root: /Users/shakhzod/Desktop/V.O.I.D
Your Working Directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_orchestrator_1

Authoritative Request: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md

Mission:
Read all markdown and docx documentation files in `/Users/shakhzod/Desktop/V.O.I.D`, compare them against the actual codebase implementation, and update the documentation to accurately reflect the real codebase.

Requirements:
1. R1. Documentation Conversion: Identify all `.docx` files in the repository and convert them to Markdown (`.md`) format. Move original `.docx` files to an archive folder or remove them from active documentation directories.
2. R2. Codebase Alignment: Analyze the current source code implementation (especially in pegasusX/, routes, data models, services, clients). Update all documentation files (both existing `.md` files and converted ones) in place so they accurately reflect current implementation, architecture, and behavior. Remove outdated claims/deprecated features.
3. Quality & Verification: Execute internal multi-agent review rounds.

Existing Context:
Check `.agents/teamwork_preview_orchestrator_1/`, `.agents/teamwork_preview_explorer_survey_1/`, `.agents/teamwork_preview_explorer_survey_2/`, `.agents/teamwork_preview_explorer_survey_3/`, and `.agents/teamwork_preview_worker_m*` to leverage the survey reports already compiled.
Resume and drive all remaining phases to 100% completion.
Maintain your `plan.md`, `progress.md`, and `BRIEFING.md` in your working directory.

When all work is complete and verified, send a message back to the Sentinel reporting victory so the independent Victory Auditor can be triggered.



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
