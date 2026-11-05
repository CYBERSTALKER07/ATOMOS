# Findings & Decisions: PegasusX End-to-End Implementation

## Requirements
- Complete end-to-end execution of the 9 FMCG roles in correct operational order:
  1. Role 1: Supplier (GS1/GTIN catalog, volume-tiered pricing, trade credit underwriting)
  2. Role 2: Factory (BOM explosion, raw material lot locking, SSCC-18 pallet serialization)
  3. Role 3: Warehouse (Inbound manifest scan, yard bolt seal, FEFO putaway, wave picking)
  4. Role 6: Retailer (Autonomous replenishment, multi-supplier saga, split-tender checkout)
  5. Role 4: Payloader (CVRPTW optimizer solver, LIFO 3D axle loading, RTI crate returns)
  6. Role 5: Driver (8-point Pre-Trip DVIR walkaround, geofenced offline ePOD handshake)
  7. Role 7: Depot Cashier (Dual-Blind UZS counting in Tiyin, RRN reclassification, CIT handover)
  8. Role 8: Admin & FinOps (Soliq OFD PKCS#12 fiscal signing, reverse factoring settlement)
  9. Role 9: Fleet Manager (Red-tag grounding, vehicle telemetry, cold-chain SLA monitoring)
- Anti-Reward-Hacking Laws 0.1–0.5: No fake gates, no mock fallbacks, no bypass flags (`-short` prohibited for full verification).
- Zero Shared UI Packages: Local tokens, native components, independent UI lifecycles for iOS, Android, Desktop, and Web.
- Parity preservation: 100% synchronization between `/Users/shakhzod/Desktop/pegasusx` and `/Users/shakhzod/Desktop/V.O.I.D/pegasusX`.

## Research Findings
- **Backplane & Spanner Emulator**:
  - Live Spanner emulator required on `localhost:9010` for `make schema-drift-gate` and `make money-path-gate`.
  - Greenfield schema DDL bootstrap is defined in `apps/backend-go/migrations/` and orchestrated via `make sandbox-infra-up`.
- **Backend & Service Health**:
  - `apps/backend-go` passes all `go test -race -short ./...` tests cleanly across all domain modules.
  - `apps/ai-worker` passes all race tests cleanly.
  - `services/optimizer-core` passes all 20 Python tests (pytest) and 7 Rust tests (cargo test).
  - All CI hygiene gates (`ci_anti_theatre_gate.sh`, `ci_no_mock_control_tower.sh`, `ci_repo_hygiene_gate.sh`) are passing cleanly.
- **Client App Architectures**:
  - Web Portals: `apps/admin-portal`, `apps/supplier-portal`, `apps/retailer-portal`, `apps/warehouse-portal` use Vite/React with local `components/ui/` and Tailwind CSS.
  - Mobile Apps: `apps/driver-app`, `apps/retailer-app`, `apps/warehouse-app` use native Swift/Kotlin and React Native/Flutter implementations with localized design systems.
  - Desktop: `apps/control-tower-macos` and `apps/desktop-station` use Tauri / SwiftUI for real-time operations.

## Technical Decisions
| Decision | Rationale |
|----------|-----------|
| Dual-Blind UZS Cash Counting | Masks expected minor Tiyin amount from cashier screen to enforce independent physically counted cash verification |
| Cryptographic Offline ePOD Handshake | Uses Ed25519/HMAC asymmetric signature tokens exchanged peer-to-peer between driver and retailer devices |
| 3D Axle Volumetric LIFO Packaging | Prevents vehicle rollover by keeping center-of-gravity low and adheres to reverse-stop unloading order |
| Soliq OFD PKCS#12 Pre-Flight | Validates tax certificates and Uzbekistan MXIK/IKPU product codes before invoice submission to eliminate fiscal rejections |
| Zero Shared UI Component Packages | Enforces extreme isolation across native platforms, preventing coupling cascades across mobile, desktop, and web |
| Deterministic 36-char AR Keys | Spanner primary key column `STRING(36)` cannot be altered; `InvoiceId` uses `"ari_" + 32-hex` and `EntryId` uses `ledgerEntryID()` with SHA-256 formatting |

## Issues Encountered
| Issue | Resolution |
|-------|------------|
| `pnpm test run` failed on portals with "no test files found" | Package script already contains `vitest run`; run `pnpm test` directly without passing `run` argument |
| `make schema-drift-gate` hung waiting for port 9010 | Docker container was offline; need to launch sandbox infrastructure via `make sandbox-infra-up` |
| Working directory mismatch in bash scripts | Ensure all repository commands execute from `/Users/shakhzod/Desktop/pegasusx` |
| `ArInvoices.InvoiceId` exceeded 36 char limit (was 40) | Stripped UUID hyphens in `NewService` so `"ari_" + 32-hex` equals exactly 36 characters |
| `ArLedgerEntries.EntryId` exceeded 36 char limit (was 41) | Added `ledgerEntryID()` helper utilizing SHA-256 hashing to format deterministic 36-character entry keys |
| Spanner DDL update lock conflict during `money_path_gate` | Temporarily stopped `backend-go` and `ai-worker` containers so DDL migrations execute without concurrent read-write transactions |

## Resources
- Project root: `/Users/shakhzod/Desktop/pegasusx`
- Secondary parity root: `/Users/shakhzod/Desktop/V.O.I.D/pegasusX`
- Instructions & Constraints: `agents.md`
- Gate verification makefile targets: `make gap-hunter-gate`, `make parity-contract-full`, `make validate-launch-readiness`, `make money-path-gate`
