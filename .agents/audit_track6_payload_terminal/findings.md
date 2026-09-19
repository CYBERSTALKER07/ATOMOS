# Comprehensive Codebase Audit Report: Track 6 — Payload, Terminal, IoT & Hardware Domain

**Audit Date:** 2026-08-30  
**Target Codebase:** `apps/backend-go` (and `pegasusX/apps/backend-go`), `apps/payload-terminal`, `apps/payload-app-android`, `apps/payload-app-ios`, Spanner DDL, and Event Contracts  
**Auditor:** Codebase Explorer (Track 6 Specialist)  
**Status:** COMPLETE — Thorough Line-by-Line Investigation & Contract Verification

---

## Executive Summary

Track 6 encompasses the physical-to-digital bridge of the PegasusX ecosystem: Payload Seal operations, Loading Ledger management, Terminal APIs across Web/Android/iOS, IoT Telemetry ingestion pipelines, Cold-Chain monitoring, Hardware authentication, and Device synchronization.

This audit uncovered **14 critical and high-severity findings**, including:
1. **Catastrophic Cold-Chain Blast Radius Inversion:** An in-transit temperature excursion on a truck mistakenly quarantines the originating warehouse's entire available stock on hand rather than the truck's cargo.
2. **Extreme Database Scalability & Concurrency Bottleneck in `apply.go`:** Every single mutation (e.g. 1 barcode scan or exception report) performs a full-table read and full-table blind overwrite of all manifests, orders, and exceptions for the entire supplier within a single Spanner ReadWriteTransaction.
3. **Dead / Broken Dock Damage Handler with Fatal Schema Incompatibilities:** `HandleDockDamage` targets non-existent tables (`LoadLedger`), invalid primary keys (`Orders.Id`), non-existent columns (`TotalPrice`), and invalid Spanner mutation data types.
4. **Client-Backend Route & Schema Drift:** Multiple API contract mismatches across Expo Terminal (`POST /v1/payload/scan`), Android (`POST /v1/delivery/missing-items`), and iOS (`POST /v1/payload/seal` missing `manifest_id`).
5. **Absence of IoT Hardware Gateways & Cryptographic Device Security:** Zero hardware-level device authentication (mTLS, device certs, TPM attestation), zero hardware tilt/shock/tamper telemetry streams, and zero offline locker/cage cryptographic token mechanisms.

---

## 1. Domain Findings: Line-by-Line Code Analysis

---

### Finding TRK6-001: Catastrophic Warehouse Inventory Quarantining on Truck Temperature Excursions
- **File & Line:** `pegasusX/apps/backend-go/stocklots/coldchain.go:227-287` (specifically lines 254-285)
- **Severity:** CRITICAL (Data Plane & Business Logic Inversion)
- **Observation:**
  ```go
  // stocklots/coldchain.go:254-285
  for pid := range products {
      iter := txn.Query(ctx, spanner.Statement{
          SQL: `SELECT LotId FROM StockLots WHERE WarehouseId = @wid AND ProductId = @pid AND Status = 'AVAILABLE'`,
          Params: map[string]any{"wid": wid, "pid": pid},
      })
      for {
          row, err := iter.Next()
          ...
          if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("StockLots", map[string]any{
              "LotId":     lotID,
              "Status":    "QUARANTINE",
              "UpdatedAt": spanner.CommitTimestamp,
          })}); err != nil {
              return err
          }
      }
      iter.Stop()
      if err := RollupInventoryV2InTxn(ctx, txn, sid, wid, pid); err != nil {
          return err
      }
  }
  ```
- **Flaw Explanation:**
  When an IoT/WMS temperature reading records an excursion on an active manifest (`IngestTemperatureInTxn`), `quarantineManifestLotsInTxn` looks up the manifest's originating warehouse (`wid`) and quarantines all lots in that warehouse where `Status = 'AVAILABLE'`.
  However, the manifest is on a **truck in transit**, and the products experiencing temperature excursion are on the road. The inventory remaining in the warehouse is temperature-controlled and untouched. Furthermore, the products on the truck are already allocated/picked and do NOT have `Status = 'AVAILABLE'`. Consequently:
  1. The spoiled cargo on the truck is NOT quarantined.
  2. The healthy stock remaining in the warehouse IS quarantined and pulled from `SupplierInventoryV2`, stopping all warehouse fulfillment across all other channels and orders.
- **Blast Radius:** Total warehouse operational blockage for all affected SKUs across the supplier network, false inventory stockouts, corrupted financial rollups.
- **Recommendation:**
  Change quarantine target to manifest-specific ship units / orders (`ManifestShipUnits` / `Orders` condition report `TEMPERATURE_BREACH`). Never quarantine warehouse-resident `AVAILABLE` stock lots for a transit-level truck excursion unless specifically tracing back a batch/lot contamination.

---

### Finding TRK6-002: Architectural Anti-Pattern: Full-Table Read & Blind Rewrite on Every Single Mutation
- **File & Line:** `pegasusX/apps/backend-go/payload/apply.go:15-81` & `payload/repository_spanner.go:48-76`
- **Severity:** CRITICAL (Scalability, Concurrency & Transaction Integrity)
- **Observation:**
  ```go
  // payload/apply.go:15-79
  return s.repo.RunTx(ctx, func(ctx context.Context, tx PayloadTx) error {
      manifests, err = tx.ListManifests(ctx)
      for _, m := range manifests {
          orders, err := tx.ListManifestOrders(ctx, m.ManifestID)
          if len(orders) > 0 { manifestOrders[m.ManifestID] = orders }
      }
      exceptions, err = tx.ListExceptions(ctx)
      ...
      if err := mutate(tx); err != nil { return err }
      ...
      for _, m := range s.manifests {
          if err := tx.SaveManifest(ctx, m); err != nil { return err }
      }
      for _, orders := range s.manifestOrders {
          for i, o := range orders {
              if err := tx.SaveManifestOrder(ctx, o, int64(i+1)); err != nil { return err }
          }
      }
      for _, e := range s.exceptions {
          if err := tx.SaveException(ctx, e); err != nil { return err }
      }
      return nil
  }, emit)
  ```
- **Flaw Explanation:**
  On **every single payload mutation** (e.g. `start-loading`, `inject-order`, `manifest-exception`, `seal`, `seal-all`, `reassign-order`):
  1. It runs a full table query of all manifests, all manifest orders, and all exceptions for the entire supplier.
  2. It executes an N+1 query loop fetching orders for every manifest.
  3. After executing the single mutation, it loops through the entire in-memory state and performs blind `InsertOrUpdateMap` writes for **EVERY** manifest, **EVERY** order, and **EVERY** exception in Spanner!
  4. In Cloud Spanner, this acquires exclusive lock ranges across the entire table for the supplier. Any concurrent request for ANY manifest of that supplier will conflict and abort with `AbortedDueToContention`. As volume grows, Spanner's 80,000 mutation limit will be breached on single calls.
  5. In-memory race condition: Global service state (`s.manifests`, `s.manifestOrders`, `s.exceptions`) is mutated inside the transaction callback before commit. If Spanner retries the transaction closure, in-memory state is corrupted.
- **Blast Radius:** Complete system lockup under concurrent terminal load, extreme p99 latencies, transaction abort storms, high Spanner CPU utilization.
- **Recommendation:**
  Refactor `Service` and `Repository` to execute fine-grained, entity-scoped Spanner mutations (e.g. `tx.SaveManifest(manifestID)`, `tx.SaveManifestOrder(manifestID, orderID)`). Eliminate full-table re-reads and full-table re-writes. Remove in-memory shared arrays from `Service`.

---

### Finding TRK6-003: Dead / Unrouted `HandleDockDamage` with Fatal Schema Inconsistencies
- **File & Line:** `pegasusX/apps/backend-go/payload/exceptions.go:21-116`
- **Severity:** HIGH (Corrupted Database Logic & Unrouted Endpoint)
- **Observation:**
  ```go
  // payload/exceptions.go:51-86
  stmtLedger := spanner.Statement{
      SQL: `DELETE FROM LoadLedger WHERE ManifestId = @m AND OrderId = @o AND ItemId = @i`,
      Params: map[string]any{"m": manifestID, "o": orderID, "i": req.ItemID},
  }
  ...
  stmtOrder := spanner.Statement{
      SQL: `UPDATE Orders SET TotalPrice = TotalPrice - @deduct WHERE Id = @o`,
      Params: map[string]any{"deduct": req.Price, "o": orderID},
  }
  ...
  m := spanner.InsertMap("OutboxEvents", map[string]any{
      "EventId":       uuid.NewString(),
      "AggregateType": events.AggregateOrder,
      "AggregateId":   orderID,
      "TopicName":     events.TopicMain,
      "Payload":       string(evJSON),
      "CreatedAt":     spanner.CommitTimestamp,
  })
  ```
- **Flaw Explanation:**
  1. `HandleDockDamage` is defined in `exceptions.go` but **is never mounted in `payloaderoutes/routes.go`**.
  2. The SQL targets table `LoadLedger` which does not exist in `schema/spanner.ddl` (the real table is `ManifestLoadLines`).
  3. The SQL targets table `Orders` with columns `Id` and `TotalPrice`, but in Spanner DDL the primary key is `OrderId` and the column is `TotalMinor`.
  4. The Outbox mutation sets `"Payload": string(evJSON)`, but `OutboxEvents.Payload` in Spanner DDL is defined as `BYTES(MAX) NOT NULL`. Spanner rejects string assignment with a type mismatch error.
  5. Lines 104-112 broadcast to non-existent websocket channels `"warehouse_ops"`, `"retailer_updates"`, `"fleet_broadcast"`.
- **Blast Radius:** If ever routed or called in production, every invocation crashes with Spanner SQL and type errors.
- **Recommendation:**
  Rewrite `HandleDockDamage` to use `ManifestLoadLines`, `Orders.TotalMinor`, `outbox.EmitJSON`, and canonical WebSocket rooms (`"payload:<supplier_id>"`), or remove this obsolete file if superseded by `ManifestLoadLines` variance flows.

---

### Finding TRK6-004: Fleet Reassign Mutates `Orders` but Desynchronizes `ManifestOrders` and Manifest Volumes
- **File & Line:** `pegasusX/apps/backend-go/payload/fleet_compat.go:47-74`
- **Severity:** HIGH (State Inconsistency & Data Loss)
- **Observation:**
  ```go
  // payload/fleet_compat.go:66-72
  driverID := s.driverIDForRouteLocked(req.NewRouteID)
  if err := tx.UpdateOrderAssignment(ctx, orderID, req.NewRouteID, driverID); err != nil {
      return err
  }
  s.orders[oIdx].RouteID = req.NewRouteID
  ```
- **Flaw Explanation:**
  `HandleFleetReassign` updates the `RouteId` and `DriverId` on the `Orders` table in Spanner via `UpdateOrderAssignment`. However:
  1. It never updates `ManifestOrders` in Spanner or in memory.
  2. It never updates the source `SupplierTruckManifests` (subtracting `TotalVolumeVU` and `StopCount`).
  3. It never updates the destination `SupplierTruckManifests` (adding `TotalVolumeVU` and `StopCount`).
  4. As a result, `Orders` says the order belongs to Route B, but `SupplierTruckManifests` and `ManifestOrders` still show it loaded on Manifest A. When Manifest A is sealed, it seals an order that was reassigned to Route B!
- **Blast Radius:** Double-dispatching of orders, incorrect truck volume capacities, duplicate driver deliveries, and corrupted driver manifest views.
- **Recommendation:**
  `HandleFleetReassign` must execute full manifest rebalancing equivalent to `HandleApplyReassign` (`payload/service.go:1393-1612`), adjusting both source and destination manifest rows and `ManifestOrders` within the same transaction.

---

### Finding TRK6-005: Client-Backend Contract Violation: Expo Terminal Calls Unmounted `POST /v1/payload/scan`
- **File & Line:** `pegasusX/apps/payload-terminal/api.ts:251-261` vs `pegasusX/apps/backend-go/payloaderoutes/routes.go:40`
- **Severity:** HIGH (Frontend-Backend Contract Parity Failure)
- **Observation:**
  ```typescript
  // apps/payload-terminal/api.ts:251-256
  export async function reportScanProgress(manifestId: string, itemId: string, itemVu: number): Promise<{ loaded_vu: number }> {
      const res = await authFetch(`${API_BASE}/v1/payload/scan`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ manifest_id: manifestId, item_id: itemId, item_vu: itemVu }),
      });
  ```
- **Flaw Explanation:**
  The Expo Payload Terminal application exposes `reportScanProgress` calling `POST /v1/payload/scan`.
  However, in `payloaderoutes/routes.go`, the only scan endpoint mounted is:
  `rr.Post("/v1/payloader/manifests/{manifestID}/load-ledger/scan", d.Service.HandleLoadLedgerScan)`
  `POST /v1/payload/scan` is not registered anywhere on the Go backend router. Any call from the terminal results in HTTP 404 Not Found.
- **Blast Radius:** Loading bay scan progress reporting fails completely on Expo Terminal clients.
- **Recommendation:**
  Update `apps/payload-terminal/api.ts` to call `/v1/payloader/manifests/${manifestId}/load-ledger/scan` matching the backend Go route and payload structure.

---

### Finding TRK6-006: Android App Contract Drift: `missingItems` Calls Unmounted Endpoint
- **File & Line:** `pegasusX/apps/payload-app-android/app/src/main/java/com/pegasus/payload/data/remote/PayloadApi.kt:187-191` vs `payloaderoutes/routes.go:64-66`
- **Severity:** MEDIUM (Native App Contract Drift)
- **Observation:**
  ```kotlin
  // apps/payload-app-android/.../PayloadApi.kt:187-191
  @POST("v1/delivery/missing-items")
  suspend fun missingItems(
      @Body req: MissingItemsRequest,
      @Header("Idempotency-Key") idempotencyKey: String? = null,
  ): StatusResponse
  ```
- **Flaw Explanation:**
  The Android Payload app defines `POST v1/delivery/missing-items`.
  In `payloaderoutes/routes.go:65`, the route registered is `rr.Post("/v1/delivery/exception-report", d.OrderService.HandleExceptionReport)`.
  There is no `/v1/delivery/missing-items` route mounted under `payloaderoutes`. Calls from Android will receive HTTP 404.
- **Blast Radius:** Drivers and payload staff cannot submit missing item reports from Android hardware terminals.
- **Recommendation:**
  Align Retrofit interface on Android to `POST v1/delivery/exception-report` or mount the alias on the Go router.

---

### Finding TRK6-007: iOS App Contract Drift: `sealOrder` Omits Required `manifest_id`
- **File & Line:** `pegasusX/apps/payload-app-ios/payload-app-ios/Services/APIClient.swift:268-276` vs `payload/service.go:1977-1983`
- **Severity:** HIGH (Native App Functional Failure)
- **Observation:**
  ```swift
  // apps/payload-app-ios/.../APIClient.swift:268-276
  func sealOrder(orderId: String, terminalId: String) async throws -> SealOrderResponse {
      try await post(
          "v1/payload/seal",
          body: SealOrderRequest(orderId: orderId, terminalId: terminalId, manifestCleared: true),
          headers: ["Idempotency-Key": PayloadIdempotency.orderSeal(orderId: orderId)]
      )
  }
  ```
- **Flaw Explanation:**
  In `payload/service.go:1977-1983`, the backend explicitly enforces:
  ```go
  if req.ManifestID == "" {
      writeJSON(w, http.StatusBadRequest, map[string]string{
          "error":   "manifest_id_required",
          "message": "POST /v1/payload/seal requires manifest_id; order-only seal is forbidden (use /v1/payloader/manifests/{id}/seal)",
      })
      return
  }
  ```
  The iOS client sends `SealOrderRequest` containing `orderId`, `terminalId`, and `manifestCleared`, but NEVER passes `manifest_id`.
  Every single `sealOrder` call from iOS devices is rejected with HTTP 400 `manifest_id_required`.
- **Blast Radius:** Payload sealing completely broken on iPad / iOS devices when using `sealOrder`.
- **Recommendation:**
  Update iOS `APIClient.sealOrder` and `SealOrderRequest` to accept and serialize `manifestId`.

---

### Finding TRK6-008: Non-Transactional `client.Apply` for GS1 Ship Unit Generation Bypasses Outbox
- **File & Line:** `pegasusX/apps/backend-go/payload/ship_units.go:181-193`
- **Severity:** HIGH (Event Sourcing & Outbox Integrity Violation)
- **Observation:**
  ```go
  // payload/ship_units.go:181-193
  _, err := client.Apply(ctx, []*spanner.Mutation{
      spanner.InsertMap("ManifestShipUnits", map[string]any{
          "ManifestId": u.ManifestID,
          "ShipUnitId": u.ShipUnitID,
          "Sscc":       u.SSCC,
          "OrderId":    u.OrderID,
          "Sequence":   u.Sequence,
          "Gtin":       nullableStr(u.GTIN),
          "CreatedAt":  spanner.CommitTimestamp,
      }),
  })
  return err
  ```
- **Flaw Explanation:**
  `insertShipUnit` writes `ManifestShipUnits` using standalone `client.Apply` outside of the parent ReadWriteTransaction and without buffering an Outbox event into `OutboxEvents`.
  Downstream warehouse sorting systems, automated RFID/barcode sorters, and EDI/ASN generators never receive an outbox Kafka event (e.g. `SHIP_UNIT_MINTED`) when SSCCs are created.
- **Blast Radius:** Downstream logistics automation and electronic ASN EDI feeds miss ship-unit updates.
- **Recommendation:**
  Move ship-unit generation into the same Spanner `ReadWriteTransaction` as manifest seal, and emit an outbox event (`EventShipUnitsGenerated`).

---

### Finding TRK6-009: Invalid GS1 SSCC Serial Number Calculation Breaches GS1-128 / SSCC-18 Standard
- **File & Line:** `pegasusX/apps/backend-go/payload/ship_units.go:161-168`
- **Severity:** MEDIUM (Hardware / Barcode Scanner Incompatibility)
- **Observation:**
  ```go
  // payload/ship_units.go:161-168
  func ssccSerial(manifestID, orderID string, seq int64) uint64 {
      h := fnv.New64a()
      _, _ = h.Write([]byte(manifestID + "|" + orderID))
      var buf [8]byte
      binary.BigEndian.PutUint64(buf[:], uint64(seq))
      _, _ = h.Write(buf[:])
      return h.Sum64()
  }
  ```
- **Flaw Explanation:**
  GS1 SSCC-18 consists of: 1 Application Identifier (00) + 1 Extension Digit (1 digit) + GS1 Company Prefix (7-10 digits) + Serial Reference (6-9 digits) + 1 Modulo-10 Check Digit (Total 18 digits).
  `ssccSerial` calculates an arbitrary 64-bit uint64 from FNV hash (`h.Sum64()`), which can be up to 20 decimal digits (`18,446,744,073,709,551,615`). When passed to `gs1.GenerateSSCC(prefix, serial)`, formatting as decimal overflows the 18-digit fixed SSCC field length or produces invalid check digits.
- **Blast Radius:** Barcode labels printed at the dock fail verification on industrial handheld 2D/1D scanners and retailer receiving gates.
- **Recommendation:**
  Generate serial references as bounded sequential integers constrained by the prefix length: `serial := (seq % maxSerialForPrefix)`.

---

### Finding TRK6-010: Destination Retailer GLN Hardcoded to Blank in ZPL Label Generator
- **File & Line:** `pegasusX/apps/backend-go/payload/ship_units.go:302-318`
- **Severity:** LOW (Compliance & Label Aesthetics)
- **Observation:**
  ```go
  // payload/ship_units.go:316-317
  _ = manifestID
  return fromGLN, toGLN // toGLN is initialized as "" and never populated
  ```
- **Flaw Explanation:**
  When generating ZPL shipping labels via `POST /v1/payload/manifests/{manifestID}/labels`, `labelGLNs` attempts to read `SupplierProfiles.Gln` for `fromGLN`, but leaves `toGLN` unassigned and returns `""`. Destination GLN on all printed physical crate and pallet labels is blank.
- **Blast Radius:** Incomplete logistics labels failing GS1-128 compliance audits at Tier-1 enterprise retailer distribution centers.
- **Recommendation:**
  Query the destination `Retailers.Gln` or `RetailerLocations.Gln` for the orders attached to the manifest and populate `toGLN`.

---

### Finding TRK6-011: Multi-Pod State Divergence in `stocklots.LoadLine` Memory Fallback
- **File & Line:** `pegasusX/apps/backend-go/stocklots/load_ledger.go:41-45, 127-164`
- **Severity:** HIGH (Distributed State Inconsistency)
- **Observation:**
  ```go
  // stocklots/load_ledger.go:42-45
  var (
      memLoadMu   sync.RWMutex
      memLoadRows = map[string]LoadLine{} // in-process memory map
  )
  ```
- **Flaw Explanation:**
  When Spanner transactions encounter errors or when running fallback paths, `ScanLoadLineInTxn` and `HandleLoadLedgerScan` fall back to mutating the package-global in-process map `memLoadRows`.
  In a multi-pod Kubernetes deployment (standard for backend-go in staging/production), Pod A and Pod B have separate in-memory maps. If Worker 1 scans on Pod A and Worker 2 seals on Pod B, Pod B sees an empty or incomplete load ledger and blocks the seal with `ErrLoadLedgerIncomplete`.
- **Blast Radius:** Intermittent, non-reproducible loading gate blockages in clustered environments.
- **Recommendation:**
  Disallow silent in-memory fallbacks in production (`auth.IsProduction()` / `auth.IsSandbox()`). All load line scans must persist durably to Cloud Spanner `ManifestLoadLines`.

---

### Finding TRK6-012: In-Transit IoT Telemetry Pipelines Lack Sensor Modalities (Humidity, Tilt, Shock, Tamper)
- **File & Line:** `pegasusX/apps/backend-go/telemetryroutes/routes.go:70-124` & `telemetry/location_store.go:38-50`
- **Severity:** HIGH (Architecture & Feature Completeness Gap)
- **Observation:**
  The `telemetryroutes` package only mounts `POST /v1/telemetry/location`.
  The telemetry payload struct only models: `lat`, `lng`, `latitude`, `longitude`, `velocity`, `heading`, `timestamp`.
  There is zero pipeline support for:
  - Humidity sensors (% RH)
  - 3-axis accelerometer shock/impact detection (G-force threshold breaches)
  - Angular tilt sensors (for hazardous/liquid/fragile cargo upright orientation)
  - Electronic hardware tamper loops / magnetic door contact switches
  - BLE beacon / RFID smart container proximity pings
- **Blast Radius:** Inability to monitor high-value or fragile shipments, inability to detect cargo tilt/drop in real-time, lack of automated tamper alarms.
- **Recommendation:**
  Extend `telemetryroutes` and `events/events.go` to support `POST /v1/telemetry/sensors` accepting multimodal telemetry frames and emitting `LOGISTICS_TELEMETRY` outbox events with automated threshold alerts.

---

### Finding TRK6-013: Absence of Hardware-Level Authentication (mTLS, HMAC, Device Certificates)
- **File & Line:** `pegasusX/apps/backend-go/payload/auth_login.go:38-117`
- **Severity:** HIGH (Hardware Security & Zero-Trust Gap)
- **Observation:**
  Terminal authentication is strictly password/PIN or Firebase Phone OTP.
  There is no support for:
  - Mutual TLS (mTLS) client certificate verification for dedicated fixed dock terminals or IoT hardware gateways.
  - HMAC token signing using hardware secure element (SE) / TPM keys.
  - Device serial registration / hardware fingerprint binding.
- **Blast Radius:** Any actor who obtains a phone number and PIN can authenticate as a terminal from any untrusted arbitrary IP, mobile device, or script.
- **Recommendation:**
  Implement hardware device certificate registration and mTLS validation middleware for payload terminals and IoT gateways (`auth.RequireDeviceCert` or HMAC signature header `X-Device-Signature`).

---

### Finding TRK6-014: Hardcoded Regional Code (`UZ-TAS`) and Static Reassignment Distances
- **File & Line:** `pegasusX/apps/backend-go/payload/manifest_list.go:58, 103` & `payload/tablet_wire.go:42`
- **Severity:** MEDIUM (Localization & Multi-Market Parity)
- **Observation:**
  ```go
  // payload/manifest_list.go:58
  RegionCode: "UZ-TAS",
  // payload/tablet_wire.go:42
  DistanceKm: 1.2,
  ```
- **Flaw Explanation:**
  1. `listManifestWiresLocked` and `manifestDetailWireLocked` hardcode `RegionCode: "UZ-TAS"`. In global multi-country deployments (e.g. US, DE, KZ cells), all payload manifests report as Tashkent, Uzbekistan.
  2. `buildTruckRecommendationsLocked` hardcodes `DistanceKm: 1.2` for all candidate reassignments rather than computing real road/haversine distances from driver telemetry.
- **Blast Radius:** Breaks multi-region cell isolation, tax jurisdiction routing, and provides misleading proximity metrics to loading bay operators.
- **Recommendation:**
  Derive `RegionCode` dynamically from the supplier/warehouse country pack (`proximity.ResolveNodeCountry`). Compute `DistanceKm` using `proximity.HaversineDistance` between warehouse and truck coordinates.

---

## 2. Deep Architectural & Edge-Case Open Questions

### Question 1: How should Physical vs. Logical Seal Desynchronization be Handled During Network Partitions?
- **Scenario:** A dock worker presses "Seal Manifest" on a physical tablet while the warehouse is experiencing an internet outage. The local tablet marks the manifest as sealed, and the driver drives away. Meanwhile, the backend still considers the manifest in `LOADING` state, and an automated solver tries to inject late-arriving urgent orders into the departing truck.
- **Inquiry:** How should offline seal leases and physical e-seal cryptographic tokens be validated when connectivity resumes to prevent invalid post-seal order injections?

### Question 2: What is the Failure Recovery Protocol for False-Positive Temperature / Shock Breaches in Transit?
- **Scenario:** An IoT sensor experiences a transient hardware glitch or battery dip, reporting a false -50°C reading or 10G shock.
- **Inquiry:** Given that `IngestTemperatureInTxn` automatically triggers condition reports, order holds, and inventory quarantines, what is the administrative override and sensor calibration audit workflow to release falsely held stock without violating cold-chain compliance guarantees?

### Question 3: How Should Multi-Compartment Multi-Temperature Vehicles (Reefer + Ambient) Be Modeled?
- **Scenario:** A delivery vehicle has dual zones (Zone A: -18°C Frozen, Zone B: +4°C Chilled, Zone C: Ambient). Currently, `SupplierTruckManifests` only has a single `TotalVolumeVU` and `MaxVolumeVU`, and `TemperatureReadings` attaches to the `ManifestId` as a whole.
- **Inquiry:** How should zone-level volume packing, independent compartment sensor telemetry, and compartment-specific lock/unlock mechanisms be structured in Spanner schema and solver constraints?

### Question 4: How Should Offline Locker / Cage Handoffs Function Without Internet Access for Driver or Customer?
- **Scenario:** A smart locker bank or secure cage is located in a basement depot with zero cellular or Wi-Fi connectivity.
- **Inquiry:** What cryptographic offline challenge-response protocol (e.g. BLE Time-based One-Time Password / TOTP or asymmetric public-key signed leases) should be implemented so that a driver can deposit goods and a recipient can retrieve goods without real-time backend round-trips?

### Question 5: How Should Partial Order Splitting and Reassignment Synchronize with Digital Twin State?
- **Scenario:** When an order is partially reassigned (`is_partial = true`) during loading due to truck capacity exhaustion, a new `SplitGroupID` is minted.
- **Inquiry:** How is the digital twin state (`twin/` service), retailer notification pipeline, and payment capture coordinated so the retailer only pays for the delivered portion upon arrival of the first truck while tracking the second truck?

---

## 3. Summary Scorecard & Priority Action Items

| Priority | Finding ID | Area | Summary |
|---|---|---|---|
| **P0** | TRK6-001 | Cold Chain / WMS | Fix inverse inventory quarantine bug in `coldchain.go` |
| **P0** | TRK6-002 | Spanner / Apply | Eliminate full-table read/rewrite in `payload/apply.go` |
| **P0** | TRK6-003 | Dock Exceptions | Fix/rewrite broken SQL and mutations in `payload/exceptions.go` |
| **P0** | TRK6-004 | Fleet Reassign | Synchronize `ManifestOrders` & manifest volumes on fleet reassign |
| **P1** | TRK6-005 | Contract Parity | Fix Expo Terminal scan API route (`/v1/payload/scan`) |
| **P1** | TRK6-006 | Contract Parity | Fix Android missing-items route mapping |
| **P1** | TRK6-007 | Contract Parity | Fix iOS `sealOrder` missing `manifest_id` |
| **P1** | TRK6-008 | Event Outbox | Enforce Spanner Tx + Outbox emit for GS1 Ship Units |
| **P1** | TRK6-011 | Concurrency | Disallow silent in-memory load ledger fallbacks in production |
| **P2** | TRK6-009 | GS1 Standard | Format GS1 SSCC serials to strictly comply with 18-digit spec |
| **P2** | TRK6-010 | GS1 Standard | Populate destination Retailer GLN on ZPL shipping labels |
| **P2** | TRK6-012 | IoT Telemetry | Build multi-sensor telemetry pipeline (tilt, shock, tamper, humidity) |
| **P2** | TRK6-013 | Security | Introduce mTLS & device certificate hardware authentication |
| **P3** | TRK6-014 | Multi-Region | Dynamic `RegionCode` and calculated distance for recommendations |

