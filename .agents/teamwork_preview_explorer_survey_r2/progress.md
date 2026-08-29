# Progress Log

- **Last visited**: 2026-08-21T00:30:30Z
- **Status**: Actively investigating H3, Geocode, and Factory Fleet in codebase
- **Tasks**:
  - [ ] 1. Investigate H3 resolution usages (Res 7 matching writers, Res 9 settlement/perimeter distinct field)
  - [ ] 2. Investigate Geocode API endpoints (Auth middleware coverage, country-bias support)
  - [ ] 3. Investigate Factory fleet list & Spanner `FactoryTruckManifests`
  - [ ] 4. Synthesize findings and write `analysis.md`
  - [ ] 5. Write `handoff.md` and notify parent



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
