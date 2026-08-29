# Worker M2 Working Directory
Assigned: Milestone 2 (Geography, Maps, and Security)
- Enforce H3 resolution 7 in matching writers (`MatchingResolution = 7`, `MatchingH3Cell`), use distinct named field/helper for resolution 9 in settlement/perimeter (`SettlementH3Cell`, `H3CellRes9`).
- Add authentication middleware (`RequireRole` or `RequireAnyAuthenticated`) to geocode routes in `platformroutes/routes.go` and add country-bias support (`components=country:uz` / `countrycodes=uz` and namespaced cache keys) in `geolocation/`.
- Switch factory fleet list (`GET /v1/factory/fleet` and `HandleFleetVehicles`) in `factory/ios_compat.go` / `factory/service.go` from in-memory mock data to Spanner `Vehicles` joined with `FactoryTruckManifests` and `Drivers`.
- Run tests in `./proximity/...`, `./geolocation/...`, `./order/...`, and `./factory/...` to verify.


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
