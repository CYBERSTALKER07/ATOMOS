# PegasusX Current Implementation State

This document describes the active, single-supplier platform known as **PegasusX**, outlining its scope, purpose, and relationship with the reference architecture.

## 1. What is PegasusX?
PegasusX is a **single-supplier execution and planning** logistics ecosystem. It is the active codebase under development in this repository.

### Key Characteristics:
- Operates a single-supplier logistics ecosystem at high reliability for thousands of retailers.
- Spanner is the durable source of truth.
- Kafka handles reliable business-event fanout.
- Redis manages cache invalidation, websocket relay, and operational lookups.
- Designed for scale, aiming to comfortably support 1M+ daily requests.

## 2. Core Components & State
PegasusX spans several domains:
1. **Planning Brain (PX90/PX91):** Handles demand baseline, scenario sandboxes, promotions simulation, confidence scoring, and the Live Enterprise Knowledge Graph (EKG).
2. **Control Tower:** Provides real-time visibility into the logistics network via WebSockets, featuring a Live EKG Network Graph and a Hexagonal Spatial Map (H3).
3. **Execution Portals & Apps:**
   - Supplier (Portal + Tauri desktop, iOS, Android)
   - Retailer (Desktop, iOS, Android)
   - Warehouse (Portal + Tauri desktop, iOS, Android)
   - Factory (Portal + Tauri desktop, iOS, Android)
   - Driver (iOS at `apps/driver-app-ios/driverappios`, Android)
   - Payload (Terminal, iOS, Android)
4. **System services:** `backend-go`, `ai-worker`, optional `handoff-service`, `services/optimizer-core` (VRP sidecar)

## 3. Separation from Pegasus
PegasusX focuses solely on the execution and planning for a single tenant (supplier). Any multi-tenant federation, cross-supplier IBP, and admin-portal tenant routing is deferred to the **Pegasus** reference architecture (see `PEGASUS_REFERENCE.md`). 
However, PegasusX maintains strict architectural compatibility with Pegasus to ensure that schemas, contracts, and events remain migratable. Divergences are documented in `parity-ledger.md`.
