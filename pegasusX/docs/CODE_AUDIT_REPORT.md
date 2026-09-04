# PegasusX Deep Code Audit Report

**Generated on:** `2026-09-04 03:13:19`  
**Audit Source:** Memgraph Live CodeGraph (64,432 Nodes / 176,106 Relationships)  

---

## Executive Summary Dashboard

| Audit Suite | Status | Finding Summary | Risk Level |
|---|---|---|---|
| **1. Multi-Tenancy & Tenant Isolation** | ⚠️ Review | `106` isolated / `123` non-isolated | **HIGH** |
| **2. Contract Drift & 404 Hazards** | ❌ Action Required | `103` client methods missing backend route | **CRITICAL** |
| **3. Dead & Unconsumed Routes** | ℹ️ Informational | `421` backend endpoints without client caller | **MEDIUM** |
| **4. Kafka Stream Integrity** | ⚠️ Review | `9` topics with zero consumers | **HIGH** |
| **5. Blast Radius Hotspots** | 🔍 Top 20 Analyzed | Up to `3078` callers per core function | **HIGH** |

---

## 1. Multi-Tenancy & Tenant Isolation Audit

- **Isolated Tables (`SupplierId` Partitioned):** `106`
- **Non-Isolated Tables:** `123`

> [!WARNING]
> Non-isolated tables must be strictly verified to ensure they only store global catalog data or system configurations. Any operational/transactional table in this list represents a potential cross-tenant leakage vulnerability.

### Sample Non-Isolated Tables:
| Table Name | File Definition |
|---|---|
| `BillOfMaterials` | `schema/spanner.ddl` |
| `BillingGlobalMeters` | `schema/spanner.ddl` |
| `ClaimEvidences` | `schema/spanner.ddl` |
| `ClientVersionPolicies` | `schema/spanner.ddl` |
| `ConsumerInbox` | `schema/spanner.ddl` |
| `CreditNoteLines` | `schema/spanner.ddl` |
| `CreditNotes` | `schema/spanner.ddl` |
| `CycleCounts` | `schema/spanner.ddl` |
| `DeliverySessionAdjustments` | `schema/spanner.ddl` |
| `DeviceTokens` | `schema/spanner.ddl` |
| `DriverAvailability` | `schema/spanner.ddl` |
| `DriverScores` | `schema/spanner.ddl` |
| `EventDensitySignals` | `schema/spanner.ddl` |
| `EvidenceDossiers` | `schema/spanner.ddl` |
| `EvidenceItems` | `schema/spanner.ddl` |
| `ExceptionTickets` | `schema/spanner.ddl` |
| `FactoryMachineTelemetry` | `schema/spanner.ddl` |
| `FactoryRawInventory` | `schema/spanner.ddl` |
| `FactorySupplyRequestQC` | `schema/spanner.ddl` |
| `FeatureFlagOverrides` | `schema/spanner.ddl` |
| `FxRates` | `schema/spanner.ddl` |
| `GlobalProducts` | `schema/spanner.ddl` |
| `InventoryAdjustments` | `schema/spanner.ddl` |
| `LotRecallImpactedOrders` | `schema/spanner.ddl` |
| `ManifestLoadLines` | `schema/spanner.ddl` |

---

## 2. Contract Drift & 404 Hazards (Client API Calls without Route)

Found **`103`** client methods in Android Kotlin, iOS Swift, or TypeScript where no matching backend Chi route could be verified.

> [!CAUTION]
> These methods risk triggering runtime `404 Not Found` errors in mobile applications or portals.

| Platform | Method | Target Endpoint Template |
|---|---|---|
| `android` | `acceptSupplyRequest` | `None` |
| `android` | `addSupplier` | `None` |
| `android` | `adjustInventory` | `None` |
| `android` | `applyImportSession` | `None` |
| `android` | `approveClaim` | `None` |
| `android` | `approveImportSession` | `None` |
| `android` | `assignDriverVehicle` | `None` |
| `android` | `completeManifest` | `None` |
| `android` | `confirmPickTask` | `None` |
| `android` | `createImportSession` | `None` |
| `android` | `deactivateOrgMember` | `None` |
| `android` | `deactivatePromotion` | `None` |
| `android` | `deleteRetailerPriceOverride` | `None` |
| `android` | `dispatchManifest` | `None` |
| `android` | `factoryManifestDetail` | `None` |
| `android` | `factorySealManifest` | `None` |
| `android` | `factoryStartLoading` | `None` |
| `android` | `fileOrderClaim` | `None` |
| `android` | `getCatalogProduct` | `None` |
| `android` | `getCategorySuppliers` | `None` |
| `android` | `getClaimEligibility` | `None` |
| `android` | `getEarlyCompleteRequest` | `None` |
| `android` | `getImportMapping` | `None` |
| `android` | `getImportSession` | `None` |
| `android` | `getInventory` | `None` |
| `android` | `getLaborDriverScore` | `None` |
| `android` | `getManifestDetail` | `None` |
| `android` | `getManifestDetail` | `None` |
| `android` | `getNotificationPreferences` | `None` |
| `android` | `getNotifications` | `None` |
| `android` | `getNotifications` | `None` |
| `android` | `getNotifications` | `None` |
| `android` | `getNotifications` | `None` |
| `android` | `getNotifications` | `None` |
| `android` | `getOrder` | `None` |
| `android` | `getOrder` | `None` |
| `android` | `getOrderReceipt` | `None` |
| `android` | `getOrderReceipt` | `None` |
| `android` | `getOrderTimeline` | `None` |
| `android` | `getOrders` | `None` |

---

## 3. Dead / Unconsumed Backend Routes

Found **`421`** backend endpoints mounted in Chi router packages that have **zero verified client callers**.

> [!NOTE]
> Many of these are internal ops diagnostics (e.g. `/ops/runtime`, `/ops/outbox/*`). Operational routes should be segregated or documented.

| HTTP Method | Path | Source File |
|---|---|---|
| `GET` | `/` | `featureflags/handlers.go` |
| `POST` | `/` | `storageroutes/routes.go` |
| `PATCH` | `/` | `demandroutes/routes.go` |
| `POST` | `/adapters/onec/import` | `partner/routes.go` |
| `GET` | `/adjustments` | `demandroutes/routes.go` |
| `POST` | `/as2` | `partner/routes.go` |
| `GET` | `/as2/config` | `partner/routes.go` |
| `PUT` | `/as2/config` | `partner/routes.go` |
| `GET` | `/audit` | `platformadmin/handlers.go` |
| `GET` | `/catalog` | `partner/routes.go` |
| `PUT` | `/catalog/prices` | `partner/routes.go` |
| `PUT` | `/catalog/products` | `partner/routes.go` |
| `PUT` | `/coa` | `partner/routes.go` |
| `GET` | `/coa` | `partner/routes.go` |
| `POST` | `/confirm` | `mfa/handlers.go` |
| `POST` | `/deactivate` | `demandroutes/routes.go` |
| `GET` | `/debug/infra/redis` | `infraroutes/routes.go` |
| `POST` | `/demand/pos-feed` | `partner/routes.go` |
| `POST` | `/driver-availability` | `laborcapacityroutes/routes.go` |
| `GET` | `/driver-score/{driverId}` | `laborcapacityroutes/routes.go` |
| `GET` | `/edi/documents` | `partner/routes.go` |
| `GET` | `/edi/documents/{documentID}` | `partner/routes.go` |
| `POST` | `/edi/documents/{documentID}/replay` | `partner/routes.go` |
| `GET` | `/edi/profile` | `partner/routes.go` |
| `PUT` | `/edi/profile` | `partner/routes.go` |
| `POST` | `/enroll` | `mfa/handlers.go` |
| `GET` | `/exports` | `partner/routes.go` |
| `POST` | `/exports` | `partner/routes.go` |
| `GET` | `/exports/{jobID}` | `partner/routes.go` |
| `GET` | `/fee-schedules` | `internal/services/billing/handlers.go` |

---

## 4. Kafka Event Stream Integrity

Out of **`13`** defined Kafka topics, **`9`** have zero registered consumer workers.

| Topic Name | Outbox Emitters | Registered Consumers | Status |
|---|---|---|---|
| `demand.adjustment.updated` | `1` | `0` | ⚠️ Black Hole (0 Consumers) |
| `route.eta.updated` | `1` | `0` | ⚠️ Black Hole (0 Consumers) |
| `pegasusx-freezelocks` | `1` | `0` | ⚠️ Black Hole (0 Consumers) |
| `driver.score.updated` | `1` | `0` | ⚠️ Black Hole (0 Consumers) |
| `pegasusx-inventoryimportevents` | `1` | `0` | ⚠️ Black Hole (0 Consumers) |
| `pegasusx-demand` | `1` | `0` | ⚠️ Black Hole (0 Consumers) |
| `capacity.zone.updated` | `1` | `0` | ⚠️ Black Hole (0 Consumers) |
| `pegasusx-realtime` | `0` | `0` | ⚠️ Black Hole (0 Consumers) |
| `pegasusx-telemetrylogistics` | `0` | `0` | ⚠️ Black Hole (0 Consumers) |
| `pegasusx-dispatch` | `0` | `2` | ✅ Active Pipeline |
| `pegasusx-orders` | `0` | `3` | ✅ Active Pipeline |
| `logistics.exceptions.v1` | `4` | `4` | ✅ Active Pipeline |
| `pegasusx-main` | `27` | `27` | ✅ Active Pipeline |

---

## 5. Critical Blast Radius Hotspots (AST In-Degree)

Functions with highest inbound call volume across the repository. Any change to these functions carries high regression potential:

| Function / Method | Inbound Callers | Fully Qualified AST Symbol |
|---|---|---|
| `t` | **`3,078`** | `pegasusX__b7089529.packages.ui-maps.src.GenericFleetLiveMap.GenericFleetLiveMap.t` |
| `string` | **`1,502`** | `pegasusX__b7089529.apps.factory-app-ios.FactoryApp.Views.Insights.InsightsView_swift.string` |
| `supplierID` | **`1,055`** | `pegasusX__b7089529.apps.backend-go.stocklots.handlers.supplierID` |
| `[]` | **`808`** | `pegasusX__b7089529.apps.warehouse-app-ios..gem.gems.xcodeproj-1.28.1.lib.xcodeproj.project_rb.[]` |
| `int64` | **`755`** | `pegasusX__b7089529.apps.warehouse-app-ios.WarehouseApp.Views.Components.ForecastConfidenceView_swift.int64` |
| `writeJSON` | **`728`** | `pegasusX__b7089529.apps.backend-go.retailer.service.writeJSON` |
| `body` | **`650`** | `pegasusX__b7089529.apps.factory-app-ios.FactoryApp.Theme.PegasusTheme_swift.body` |
| `message` | **`566`** | `pegasusX__b7089529.apps.driver-app-ios.driverappios.driverappios.Views.DriverNotificationInboxView_swift.message` |
| `warehouseID` | **`506`** | `pegasusX__b7089529.apps.backend-go.stocklots.handlers.warehouseID` |
| `writeJSON` | **`506`** | `pegasusX__b7089529.apps.backend-go.order.service.writeJSON` |
| `value` | **`488`** | `pegasusX__b7089529.apps.warehouse-portal.vitest.setup.value` |
| `writeJSON` | **`476`** | `pegasusX__b7089529.apps.backend-go.supplier.service.writeJSON` |
| `IsNil` | **`441`** | `pegasusX__b7089529.sdk.partner.go.utils.IsNil` |
| `writeJSON` | **`389`** | `pegasusX__b7089529.apps.backend-go.warehouse.service.writeJSON` |
| `load` | **`362`** | `pegasusX__b7089529.apps.payload-app-ios.payload-app-ios.Views.InboundReturnsView_swift.load` |
| `usePortalT` | **`351`** | `pegasusX__b7089529.apps.factory-portal.lib.i18n.usePortalT` |
| `WithClaims` | **`348`** | `pegasusX__b7089529.apps.backend-go.auth.claims.WithClaims` |
| `New` | **`347`** | `pegasusX__b7089529.apps.backend-go.cache.cache.New` |
| `open` | **`342`** | `pegasusX__b7089529.apps.warehouse-app-ios..gem.gems.xcodeproj-1.28.1.lib.xcodeproj.project_rb.open` |
| `FromContext` | **`325`** | `pegasusX__b7089529.apps.backend-go.auth.claims.FromContext` |
| `EmitJSON` | **`295`** | `pegasusX__b7089529.apps.backend-go.outbox.outbox.EmitJSON` |
| `apiFetch` | **`285`** | `pegasusX__b7089529.apps.factory-portal.lib.auth.apiFetch` |
| `items` | **`272`** | `pegasusX__b7089529.apps.factory-app-ios.FactoryApp.Layout.FactoryAdaptiveShell_swift.items` |
| `name` | **`247`** | `pegasusX__b7089529.apps.warehouse-app-ios..gem.gems.xcodeproj-1.28.1.lib.xcodeproj.project.object.root_object_rb.name` |
| `writeJSON` | **`223`** | `pegasusX__b7089529.apps.backend-go.factory.service.writeJSON` |

---
*Report generated automatically by PegasusX CodeGraph Studio static audit engine.*
