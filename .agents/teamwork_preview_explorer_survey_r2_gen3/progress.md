# Progress — R2 Geography, Maps, and Security Investigation

Last visited: 2026-08-20T19:33:00Z

- [x] Initialized DISPATCH.md and BRIEFING.md
- [ ] Part 1: Investigate H3 resolution usages (Res 7 vs Res 9) in matching writers, settlement, perimeter logic
- [ ] Part 2: Investigate Geocode API endpoints, handlers, auth middleware (RequireRole / RequireAnyAuthenticated), and country-bias handling
- [ ] Part 3: Investigate Factory fleet list endpoint/handler, Spanner schema/models for FactoryTruckManifests, and migration details
- [ ] Write analysis.md
- [ ] Write handoff.md
- [ ] Send completion message to parent


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
