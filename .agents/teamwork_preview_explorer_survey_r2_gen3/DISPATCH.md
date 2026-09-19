## 2026-08-20T19:32:59Z

You are the Explorer for Requirement 2 (R2: Geography, Maps, and Security).
Your Working Directory is: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_r2_gen3
Repository Root is: /Users/shakhzod/Desktop/V.O.I.D
Authoritative Request: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md

Instructions:
1. Search within the codebase at /Users/shakhzod/Desktop/V.O.I.D/pegasusX for H3 geospatial resolution usages across matching writers, settlement, and perimeter logic. Check where H3 resolution 7 must be enforced in matching writers and where resolution 9 is used in settlement/perimeter logic with distinct named fields (e.g. H3Res9Index).
2. Search for geocode API endpoints and handlers in the backend. Check authentication middleware coverage (RequireRole or RequireAnyAuthenticated) and country-bias handling.
3. Search for factory fleet list endpoint/handler and Spanner schema/models for FactoryTruckManifests. Detail the migration to pull from FactoryTruckManifests.

Deliverables:
- Write /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_r2_gen3/analysis.md
- Write /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_r2_gen3/handoff.md
- Send message back to parent when done.


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
