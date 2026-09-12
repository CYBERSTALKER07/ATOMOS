## 2026-08-21T00:26:28Z

Explorer for Requirement 2 (R2: Geography, Maps, and Security).
Repository Root: /Users/shakhzod/Desktop/V.O.I.D
Authoritative Request: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md

Tasks:
1. Locate all H3 geospatial resolution usages across matching writers, settlement, and perimeter logic.
   - Check where H3 resolution 7 must be enforced in matching writers.
   - Check where resolution 9 is used in settlement/perimeter logic and identify if it needs a distinct named field (e.g., `H3Res9Index` or similar distinct field name) rather than ambiguous/shared fields.
2. Locate all geocode API endpoints and handlers in the backend.
   - Check authentication middleware coverage. Verify where `RequireRole` or `RequireAnyAuthenticated` is missing and must be added.
   - Check how country-bias is passed, parsed, or missing in geocode requests/handlers, and identify how it should be implemented.
3. Locate the factory fleet list endpoint/handler and its current data source.
   - Check the Spanner schema and models for `FactoryTruckManifests`.
   - Identify the exact changes needed to switch the factory fleet list to pull from Spanner `FactoryTruckManifests`.


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
