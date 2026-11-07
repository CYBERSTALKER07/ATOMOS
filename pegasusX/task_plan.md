# Task Plan: End-to-End PegasusX 9-Role FMCG Implementation & Verification

## Goal
Implement, wire, and execute the complete, unbroken end-to-end multi-role transaction lifecycle across PegasusX in correct operational order (from Supplier catalog to Retailer checkout, Factory BOM, Warehouse DC ops, Payloader 3D loading, Driver CVRPTW delivery, Depot Cashier dual-blind vaulting, and Soliq fiscal settlement) verified against live Spanner, Kafka, and native client surfaces.

## Current Phase
Phase 9: Full Multi-Platform Verification, Control Tower Sign-Off & Dual-Root Parity

## Phases

### Phase 1: Environment Baseline & Sandboxed Backplane Ignition
- [x] Verify local Docker / compose infra availability
- [x] Boot Cloud Spanner emulator, Redis HA backplane, and Kafka broker (`make sandbox-infra-up`)
- [x] Apply Spanner greenfield DDL and execute live schema-drift check (`make schema-drift-gate`)
- [x] Confirm baseline service readiness (`backend-go`, `ai-worker`, `optimizer-core`)
- **Status:** complete

### Phase 2: Upstream Commercial & Production S&OP (Roles 1 & 2: Supplier & Factory)
- [x] Verify GlobalProducts GS1/GTIN catalog publishing and multi-tier volume matrices
- [x] Validate bilateral trade credit underwriting & dunning ladder configuration
- [x] Execute Factory BOM explosion, raw material lot locking, and in-line QC inspection
- [x] Generate SSCC-18 serialized pallet transfer manifests to DC
- **Status:** complete

### Phase 3: DC Inbound Receiving, FEFO Lot Allocation & Inventory Gating (Role 3: Warehouse)
- [x] Process inbound transfer manifest and yard gate bolt seal verification
- [x] Verify ABC velocity putaway slotting and FEFO stock lot batch allocation
- [x] Validate cold-chain thermal tolerance logging and CRDT offline inventory ledger
- [x] Test wave picking reservation and abandoned picker tote heartbeat takeover
- **Status:** complete

### Phase 4: Retailer Ordering, S&OP Replenishment & Split-Tender Checkout (Role 6: Retailer)
- [x] Validate shelf gap detection and autonomous S&OP replenishment cart generation
- [x] Execute multi-supplier checkout saga creating `ParentOrders` and child orders
- [x] Enforce split-tender authorization waterfall (trade credit balance check before cash/card)
- [x] Verify order state machine transition `DRAFT` → `PLACED` → `CONFIRMED`
- **Status:** complete

### Phase 5: 3D Axle Loading & Dispatch Optimization (Role 4: Payloader & Optimizer Core)
- [x] Run CVRPTW optimizer solver via `services/optimizer-core` (Python OR-Tools & Rust)
- [x] Enforce reverse-stop LIFO load-out sequencing and 3D volumetric/axle weight scaling
- [x] Account for returnable transport items (RTI crates/pallets) in reverse space
- [x] Bind cryptographic tamper-evident bolt seal to route manifest
- **Status:** complete

### Phase 6: Fleet Inspection, Route Dispatch & Doorstep Handshake (Roles 5 & 9: Driver & Fleet)
- [x] Enforce mandatory 8-point Pre-Trip DVIR walkaround clearance (Red-Tag Grounding gate)
- [x] Dispatch route and verify 150m/50m doorstep geofence progression
- [x] Execute offline cryptographic JWT proof-of-delivery (ePOD) handshake & QR verification
- [x] Verify doorstep partial rejection (volume-break tier preservation) and mid-route breakdown rescue custody split
- **Status:** complete

### Phase 7: Depot Cashier Dual-Blind Recount & Vault Settlement (Role 7: Cashier & Vault Ops)
- [x] Ingest tamper-evident cash bag at depot vault
- [x] Enforce Dual-Blind UZS Denomination Counting in integer Tiyin (strictly masking `ExpectedCashMinor`)
- [x] Trigger recount ratchet alert (`BLIND_RECOUNT_REQUIRED`) on variance before supervisor reveal
- [x] Execute RRN-verified tender reclassification (Cash to Card) within ±15m window
- [x] Generate CIT armored courier bank handover manifest
- **Status:** complete

### Phase 8: Fiscal Clearance, Tax Invoicing & Payout Settlement (Role 8: Admin & FinOps)
- [x] Run Soliq OFD PKCS#12 two-phase fiscal pre-flight (IKPU/MXIK code validation & .p12 certificate health)
- [x] Register fiscal tax invoice (`REGISTERED_SOLIQ`) and generate OFD QR code
- [x] Generate immutable `InvoiceSettlementSlices` and execute reverse factoring payout
- [x] Verify post-settlement concealed damage claims tribunal (48h rolling reserve holdbacks)
- **Status:** complete

### Phase 9: Full Multi-Platform Verification, Control Tower Sign-Off & Dual-Root Parity
- [ ] Run `make money-path-gate` and assert clean exit code 0 across the entire transaction cascade
- [ ] Execute `make ci-enterprise-gates` and `make validate-launch-readiness`
- [ ] Verify macOS Control Tower (`apps/control-tower-macos`) and Web Admin (`apps/admin-portal`) live telemetry displays
- [ ] Synchronize all validated changes to `/Users/shakhzod/Desktop/V.O.I.D/pegasusX` with 0 diff
- **Status:** in_progress

### Phase 4: Retailer Ordering, S&OP Replenishment & Split-Tender Checkout (Role 6: Retailer)
- [ ] Validate shelf gap detection and autonomous S&OP replenishment cart generation
- [ ] Execute multi-supplier checkout saga creating `ParentOrders` and child orders
- [ ] Enforce split-tender authorization waterfall (trade credit balance check before cash/card)
- [ ] Verify order state machine transition `DRAFT` → `PLACED` → `CONFIRMED`
- **Status:** pending

### Phase 5: 3D Axle Loading & Dispatch Optimization (Role 4: Payloader & Optimizer Core)
- [ ] Run CVRPTW optimizer solver via `services/optimizer-core` (Python OR-Tools & Rust)
- [ ] Enforce reverse-stop LIFO load-out sequencing and 3D volumetric/axle weight scaling
- [ ] Account for returnable transport items (RTI crates/pallets) in reverse space
- [ ] Bind cryptographic tamper-evident bolt seal to route manifest
- **Status:** pending

### Phase 6: Fleet Inspection, Route Dispatch & Doorstep Handshake (Roles 5 & 9: Driver & Fleet)
- [ ] Enforce mandatory 8-point Pre-Trip DVIR walkaround clearance (Red-Tag Grounding gate)
- [ ] Dispatch route and verify 150m/50m doorstep geofence progression
- [ ] Execute offline cryptographic JWT proof-of-delivery (ePOD) handshake & QR verification
- [ ] Verify doorstep partial rejection (volume-break tier preservation) and mid-route breakdown rescue custody split
- **Status:** pending

### Phase 7: Depot Cashier Dual-Blind Recount & Vault Settlement (Role 7: Cashier & Vault Ops)
- [ ] Ingest tamper-evident cash bag at depot vault
- [ ] Enforce Dual-Blind UZS Denomination Counting in integer Tiyin (strictly masking `ExpectedCashMinor`)
- [ ] Trigger recount ratchet alert (`BLIND_RECOUNT_REQUIRED`) on variance before supervisor reveal
- [ ] Execute RRN-verified tender reclassification (Cash to Card) within ±15m window
- [ ] Generate CIT armored courier bank handover manifest
- **Status:** pending

### Phase 8: Fiscal Clearance, Tax Invoicing & Payout Settlement (Role 8: Admin & FinOps)
- [ ] Run Soliq OFD PKCS#12 two-phase fiscal pre-flight (IKPU/MXIK code validation & .p12 certificate health)
- [ ] Register fiscal tax invoice (`REGISTERED_SOLIQ`) and generate OFD QR code
- [ ] Generate immutable `InvoiceSettlementSlices` and execute reverse factoring payout
- [ ] Verify post-settlement concealed damage claims tribunal (48h rolling reserve holdbacks)
- **Status:** pending

### Phase 9: Full Multi-Platform Verification, Control Tower Sign-Off & Dual-Root Parity
- [ ] Run `make money-path-gate` and assert clean exit code 0 across the entire transaction cascade
- [ ] Execute `make ci-enterprise-gates` and `make validate-launch-readiness`
- [ ] Verify macOS Control Tower (`apps/control-tower-macos`) and Web Admin (`apps/admin-portal`) live telemetry displays
- [ ] Synchronize all validated changes to `/Users/shakhzod/Desktop/V.O.I.D/pegasusX` with 0 diff
- **Status:** pending

## Key Questions
1. Is local Docker daemon running to host the Spanner emulator (`:9010`), Redis, and Kafka?
2. Are all Spanner migrations already applied, or do we need `cmd/setup` to bootstrap tables?
3. Does `make money-path-gate` exercise all 18 statuses and 50 FSM transition edges?

## Decisions Made
| Decision | Rationale |
|----------|-----------|
| Use persistent planning files (`task_plan.md`, `findings.md`, `progress.md`) in repo root | Complies with `/planning-with-files` doctrine: context window = RAM, filesystem = persistent disk |
| Strictly preserve 9-role operational sequence (Supplier → Factory → Warehouse → Retailer → Payloader → Driver → Cashier → Admin) | Matches real physical FMCG supply chain physics and Palantir Foundry semantic-kinetic-dynamic architecture |
| Execute against live Spanner emulator rather than `-short` | Conforms to Anti-Reward-Hacking Law 0.2 (never mask failures with bypass flags) |

## Errors Encountered
| Error | Attempt | Resolution |
|-------|---------|------------|
|       | 1       |            |

## Notes
- Update phase status as you progress: pending → in_progress → complete
- Re-read this plan before major decisions (attention manipulation)
- Log ALL errors - they help avoid repetition
- Never repeat a failed action - mutate your approach instead
