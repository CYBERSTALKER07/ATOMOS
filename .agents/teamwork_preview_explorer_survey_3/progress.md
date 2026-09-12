# Progress — Explorer 3 (Client Apps & Multi-Role Parity Inspector)

Last visited: 2026-08-20T17:28:15Z

## Status
- [x] Initialized DISPATCH, BRIEFING, progress
- [x] Surveyed packages/ (types, api-client, ws-refresh-contract, desktop-bridge, desktop-cache, ui-kit, mobile kits, barcode kits)
- [x] Surveyed Supplier clients (portal, Android, iOS)
- [x] Surveyed Retailer clients (desktop, Android, iOS)
- [x] Surveyed Driver clients (Android, iOS)
- [x] Surveyed Warehouse clients (portal, Android, iOS)
- [x] Surveyed Factory clients (portal, Android, iOS)
- [x] Surveyed Payload clients (terminal, Android, iOS)
- [x] Audited WebSocket, State Stores, Mocks/Theatre vs Real API Wiring
- [x] Compared with `ROLE_ROW_PARITY_MATRIX.md` and related docs
- [x] Ran unit tests across client packages and portals (all green)
- [x] Synthesized `clients_parity_report.md`
- [/] Generating 5-component `handoff.md` and dispatching completion message to parent


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
