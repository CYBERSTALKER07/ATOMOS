# Technology Inventory

Canonical inventory of tools, technologies, and external services used across Pegasus.

This file is the human-readable companion to `pegasus/context/technology-inventory.json` and must stay synchronized with it.

## Inventory Sources

- `pegasus/**/package.json`
- `pegasus/**/go.mod`
- `pegasus/**/build.gradle.kts`
- `pegasus/**/Cargo.toml`
- `pegasus/docker-compose.yml`
- `pegasus/infra/terraform/*.tf`
- keyword sweep across `pegasus/**/*.{go,ts,tsx,js,json,kts,tf,yml,yaml,md}`

## Languages And Runtimes

- Go
- TypeScript and JavaScript
- Kotlin
- Swift and SwiftUI
- Rust
- Python
- Shell

## Web And Desktop Stack

- Next.js + React (admin, factory, warehouse, retailer desktop)
- Tailwind CSS + HeroUI + Motion + Recharts + Mapbox/MapLibre
- Tauri 2 desktop shells with Rust backends
- Expo/React Native payload terminal

## Backend Core Stack

- Go services with `chi` HTTP routing and WebSockets
- Spanner data access
- Kafka eventing
- Redis cache and Pub/Sub invalidation
- Firebase integration
- OpenTelemetry and Prometheus metrics

## Android Stack

- Jetpack Compose + Material 3
- Hilt DI + Retrofit/OkHttp networking
- Room + DataStore local state
- Firebase Auth/Messaging + Google Maps Compose
- CameraX + ML Kit barcode scanning

## iOS Stack

- SwiftUI native apps
- APNs push channel and Apple-native design patterns

## Data, Messaging, And Local Emulators

From `pegasus/docker-compose.yml`:

- Kafka (KRaft) + Kafka UI + topic bootstrap job
- Redis
- Spanner emulator
- Firebase Auth emulator
- WireMock Global Pay mock

## Cloud And Infrastructure Services

From Terraform under `pegasus/infra/terraform`:

- Cloud Run
- GKE + Workload Identity + KEDA via Helm
- Cloud Spanner + Memorystore Redis
- Cloud Armor + Cloud CDN + Cloud DNS + Cloud NAT + Private Service Connect
- Artifact Registry
- Google Cloud Monitoring alert policies and uptime checks
- Multi-region Spanner/GKE topology options

## Security And Reliability Patterns

- Transactional outbox for durable state-change events
- Redis Pub/Sub invalidation after commit
- Maglev-style consistent hash affinity
- Cloud Armor WAF + OWASP rules + per-IP throttling
- Circuit-breaker and priority-guard readiness patterns

## Engineering Guard Tooling

- Contract Guard MCP: `pegasus/scripts/contract_guard_mcp.py`
	- Enforces codebase-first MCP context weighting on contract-triggered diffs (runtime code surfaces must dominate context-doc touches).
- Architecture Guard MCP: `pegasus/scripts/architecture_guard_mcp.py`
	- Enforces codebase-first MCP context weighting on architecture-triggered diffs (runtime code surfaces must dominate context-doc touches).
- Design System Guard MCP: `pegasus/scripts/design_system_guard_mcp.py`
	- Enforces codebase-first MCP context weighting on design-triggered diffs (runtime code surfaces must dominate context-doc touches).
- Production Safety Guard: `pegasus/scripts/production_safety_guard.py`
- Visual + Test Intelligence Guard: `pegasus/scripts/visual_test_intelligence_guard.py`
- Security Guard: `pegasus/scripts/security_guard.py`
- Aggregated PR workflow: `.github/workflows/one-eye-guards.yml`

## External Integrations And Providers

- Payme
- Click
- Global Pay
- Stripe
- Telegram
- Firebase
- Google Maps ecosystem

## Sync Contract

If any feature, dependency, service, or runtime changes, update all of:

1. `pegasus/context/technology-inventory.md`
2. `pegasus/context/technology-inventory.json`
3. `.github/ACT.md`
4. `.github/copilot-instructions.md`
5. `.github/gemini-instructions.md`
6. `pegasus/context/architecture.md`
7. `pegasus/context/architecture-graph.json`
