# 6. Per-Role, Per-App, Per-Feature Reality

Method: static code audit of every app under `apps/`, verified against the backend routes it calls. Status vocabulary per §1. API depths are measured (Retrofit endpoint counts for Android, unique `/v1/` path literals for iOS, unique api-client methods for portals).

## 6.1 Retailer — Android (187 kt files) · iOS (144 real Swift files) · Desktop (27 dashboard routes, Tauri)

**Maturity: ~92%. All three variants are WIRED-LIVE; zero mock data found in any variant.**

### What exists and works (WIRED-LIVE unless noted)

| Feature | Evidence |
|---|---|
| Auth (login/register/org memberships/switch) | `retailer-app-android/.../PegasusApi.kt:64-77`; iOS `Services/AuthManager.swift:74,143`; desktop OS-keyring auth |
| Catalog, search, cart, quoted checkout | `PegasusApi.kt:93,133-154`; desktop `components/CheckoutModal.tsx`; idempotency keys per action |
| Orders lifecycle + tracking map (WS live) | `DeliveryTrackingViewModel.kt`, `RetailerWebSocket.kt`; iOS `DeliveryMapView.swift` |
| AI predictions / preorders (confirm/reject with idempotency) | `PegasusApi.kt:174-180`; desktop `lib/api.ts:15-29` |
| Claims (eligibility countdown, file, media upload tickets) | `PegasusApi.kt:112-125`; `FileClaimSheet.kt`; iOS `FileClaimView.swift` |
| Auto-order rules + mode + scopes + shadow inbox | `AutoOrderScreen.kt`; iOS `AutoOrderView.swift`; desktop `auto-order/page.tsx` |
| **POS with offline queue** | `PosScreen.kt` + Room `PendingPosSaleDao.kt`/`PendingPosSaleSync.kt`; iOS `PendingPosStore.swift`; backend `retailer/pos.go`, routes `retailerroutes/routes.go:87-101` (sessions, sales, void, refund, holds) |
| Store stock counts, local SKUs, shifts, sections | `StoreStockScreen.kt`, `LocalSkusScreen.kt`, `ShiftsScreen.kt` |
| Reports/analytics + HQ multi-location (with export) | desktop `hq/page.tsx:72-120` (`/v1/retailer/hq/*`) |
| Payments, saved cards, credit profile | `SavedCardsViewModel.kt`, `CreditProfileViewModel.kt` (`/v1/retailer/credit-profile`) |
| Suppliers discovery/connect, team, locations, setup wizard, capabilities packs | `PegasusApi.kt:151-244` |
| Control tower (hex map, live counts) | `ui/controltower/ControlTowerScreen.kt`; iOS `ControlTowerView.swift:16` |
| Notifications inbox + FCM; auto-updater | `PegasusFirebaseMessagingService.kt`; `service/AutoUpdater.kt:97,147` |
| Offline: Room (catalog/orders/POS/predictions) + workers; iOS file-based replayer; desktop cache + offline tray | `AppDatabase.kt`; `PendingOrderSyncWorker.kt`; iOS `PendingOrderReplayer.swift` |

### Incomplete / decorative / broken

- **Auto-order indicator on My Suppliers is a placeholder** — always shows the icon if the supplier has orders (`MySuppliersScreen.kt:291`). Cosmetic.
- **Retailer-iOS repo hygiene**: ~1,380 vendored SPM build checkouts committed under the app tree, in a directory misspelled `reatilerapp` — build-reproducibility and review-noise hazard.
- **Offline POS contradiction**: client-side offline POS queues are wired (Room/file), yet the project's own status docs list offline POS as product-deferred — the unresolved half is server-side: fiscalization and idempotent acceptance of replayed POS sales. Treat end-to-end offline POS as PARTIAL.
- No E2E-visible refund/return initiation from the retailer side (claims only); HQ export is CSV-only.
- Desktop Control Tower is a simpler surface than mobile parity.

### Missing features that matter (Retailer)

1. **Server-verified offline POS acceptance + fiscalization.**
 *Purpose:* make offline POS sales legally and financially real.
 *Why needed:* the client queues sales offline; without server-side replay acceptance tied to fiscal receipts and idempotent dedup, offline sales are either lost, double-counted, or legally non-compliant in an OFD-mandated market.
 *Logic:* each queued sale carries a deterministic idempotency key (SHA-256 of store+session+line content, as `packages/api-client/idempotency.ts` already does for checkout); server accepts into `PosSales` with unique `(StoreId, IdempotencyKey)`; replayed batch opens a fiscalization leg per sale (same `FISCALIZING` machine as orders); conflicts surface in a shift-close report.
 *End-to-end:* cashier sells offline → Room queue → connectivity returns → `PendingPosSaleSync` replays → server dedups/accepts → fiscal receipt issued per sale → shift close reconciles POS cash + card + queue → HQ reports include the sales.
2. **Cross-supplier cart with order splitting.** *Purpose:* one cart, many suppliers. *Why:* the runtime is single-supplier; `Orders` PK embeds `SupplierId`. *Logic:* new `ParentOrders` table; `ParentOrderId` on `Orders`; split engine fans out per-supplier child orders, each with its own credit check, inventory plan, pricing resolution, warehouse assignment; retailer UI rolls status up. *E2E:* cart mixed SKUs → checkout splits → per-supplier legs flow independently → unified tracking view. (Multi-tenancy Phase 2; 1 new table, 2 altered, 30–50 files.)
3. **Shelf-count / sell-through capture UX feeding auto-order.** *Purpose:* make auto-order inputs truthful. *Why:* the inventory-grounded (R,s,S) proposals decay confidence by stock-record age (`UpdatedAt`); without easy shelf counts, shadow acceptance stays low and `place` never justifiably turns on. *Logic:* count sessions write `RetailerStockBalances` (exists); confidence = f(recency, count frequency); proposals only from stock fresher than threshold. *E2E:* weekly guided count (barcode + qty) → balances fresh → auto-order proposals carry real on-hand → acceptance rate measurable in the shadow ledger (`RetailerAutoOrderShadowProposals`).
4. **KYC / business verification on onboarding.** *Purpose:* credit decisions require verified identity. *Why:* credit limits are granted at placement; today onboarding is self-serve with no verification artifact. *Logic:* document capture → review queue (admin console dependency) → status gate on credit eligibility. *E2E:* register → upload → approved → credit programs visible.
5. **Retailer-initiated refund/return request flow.** *Purpose:* close the post-delivery loop without phone calls. *Why:* today only claims exist; refunds are admin/global-pay paths. *Logic:* claim → approved → credit note (`creditnote/`) → refund leg on original payment method or AR credit. *E2E:* claim approved → retailer chooses refund-vs-credit → credit note issued → money leg or AR balance adjustment → fiscal corrective chain (`CreditNotes.OriginalEhfId/CorrectiveEhfId` schema exists, `spanner.ddl:1696-1698`).

## 6.2 Supplier — Portal (882 files, also desktop via Tauri) · Android (143 kt) · iOS (138 Swift)

**Maturity: ~90%. Zero TODO/mock hits in main sources on any variant. Mobile has NO offline mode (no Room/SwiftData) — the worst offline story among role apps.**

### What exists and works

| Feature | Evidence |
|---|---|
| Auth incl. business setup + billing | `packages/api-client/index.ts:434-462` |
| Orders hub/detail; manifests + exceptions; dispatch preview/execute (MapLibre) | `OrdersHubScreen.kt`; `DispatchPreviewScreen.kt`; portal `(portal)/dispatch` |
| Catalog CRUD + barcode + images; inventory CSV import sessions | `CatalogScreen.kt`; `InventoryImportScreen.kt`; `supplier/import_sessions.go` |
| Fleet: live map, org fleet (drivers/vehicles/members), delivery zones, topology, supply lanes | `FleetLiveMapScreen.kt`, `TopologyScreen.kt`; portal `topology`, `delivery-zones` |
| Claims/chargebacks/credit notes; exceptions (negotiations, shop-closed, early-complete) | `ClaimsScreen.kt`, `ChargebacksScreen.kt`; portal `exceptions/*` |
| Finance: payments, ledger, reconciliation, treasury, earnings | `LedgerScreen.kt`, `ReconciliationScreen.kt`; portal `(portal)/reconciliation` |
| Promotions + performance; retailer price overrides | `PromotionsScreen.kt`; `RetailerOverridesScreen.kt` |
| Planning surfaces (S&OP view, policies, seasonal overrides, forecast accuracy) | `PlanningBrainScreen.kt`; portal `settings/planning`; `GET /v1/supplier/analytics/demand/accuracy` |
| Return policy settings; compliance; notification prefs | `ReturnPolicySettingsScreen.kt` |
| Admin-capable ops inside portal (assign driver, status patch, FX rates, partner keys) | api-client `/v1/admin/*`; `apps/admin-portal/README.md` routing table |

### Incomplete / decorative / broken

- **S&OP planning view renders stub math** — backend `GetSAndOP` returns `factories × 700 × 7` and `projectStockouts` emits literal `sku-projection-%d` strings (`planning/service.go:212,252`). The UI is wired; the substance behind it is placeholder.
- **No offline capability on supplier mobile** — field reps on bad networks lose work.
- iOS dual-target file duplication (`CreateDriverSheet.swift` ×2, `CreateVehicleSheet.swift` ×2) — drift risk.
- Payout execution absent (settlement authority endpoint is a reporting view, `GET /v1/payment/settlement/authority`); refund initiation absent (Gateway-side refund exists for Global Pay only).

### Missing features that matter (Supplier)

1. **Payout execution.** *Purpose:* close the money loop for suppliers. *Why:* collections exist; disbursement does not — a marketplace without payouts cannot monetize. *Logic:* settlement authority view already computes `operating_currency_total_minor`; add payout batches = Σ(captured legs) − Σ(refunds) − commission (fee schedule dependency) per supplier per period; execute via PSP payout rail or bank file export; ledger entries per payout with idempotency keys. *E2E:* period close → batch preview → finance approve → rail execution → supplier statement reconciliation.
2. **Refund initiation (full/partial).** *Purpose:* money reversals without engineering tickets. *Why:* the only "Refund" occurrence in non-test Go reads `AmountRefunded` off a Stripe webhook; GP executor has `executeRefund` but no product surface triggers it. *Logic:* refund ≤ captured amount − prior refunds (cap exists at `payment/service.go:713-716`); create reversal ledger legs; fiscal corrective document; AR credit if credit sale. *E2E:* dispute/claim approve → refund wizard → PSP call (must fix §7 P0-1 first) → reversal legs → fiscal correction → retailer notified.
3. **Pricing authority engine.** *Purpose:* governed pricing — who may set/change prices, within what guardrails. *Why:* `pricing/service.go` is a repository delegate (4 files); the design doc is a self-declared stub; promo engine exists but price governance does not. *Logic:* rule table `(role, scope, delta_limit_bps, margin_floor_bps, approval_required)`; proposed change → policy evaluation → auto-apply or approval task → effective-dated `PriceListItem`; audit row per change. *E2E:* rep proposes −12% → floor check → manager approve → new effective window → checkout quotes respect it.
4. **Supplier mobile offline queue.** *Purpose:* field usability. *Why:* only role with zero offline. *Logic:* reuse `packages/mobile-android-kit` offline contract (ACK 409 / retry 5xx / dead 4xx) already proven in driver. *E2E:* queue mutations → flush on reconnect with capture-time coordinates.
5. **Per-supplier delivery perimeter enforcement (E2).** *Purpose:* multi-supplier correctness. *Why:* `retailer/proximity_service.go:24` uses one global key `ssmr:delivery_perimeter` in production reads; the per-supplier helper exists but is design-only. *Logic:* `PerimeterKeyForSupplier(supplierId)` on all reads/writes. *E2E:* supplier A's zone edits never leak into supplier B's eligibility checks.

## 6.3 Driver — Android (178 kt) · iOS (129 Swift)

**Maturity: ~95% — the most production-hardened role. 56 endpoints per platform; zero mock/TODO hits.**

### What exists and works

| Feature | Evidence |
|---|---|
| Manifest load, arrive/deliver/complete lifecycle (all through the FSM-validated funnel) | `DriverApi.kt:80,109,141,170`; backend `order/service.go:2153` |
| POD: QR validate/scan, signature pad, photo proof (credit leave requires photo, fail-closed) | `DriverApi.kt:123,134`; `SignaturePad.kt`; iOS `SignaturePadView.swift` |
| Cash collection with server-computed expectation | `DriverApi.kt:148`; `CashCollectionViewModel.kt`; `cashrecon/service.go:39-57` |
| Delivery correction/amend/offload review | `CorrectionViewModel.kt`, `OffloadReviewViewModel.kt` |
| Fiscal retry UI (FISCALIZING/FISCAL_FAILED) | `FiscalizingView.kt`, `FiscalFailedView.kt` |
| Telemetry: adaptive filter (15s/20m/15°), WS + Room-buffered sync, boot resume | `TelemetryService.kt`, `TelemetrySyncWorker.kt`, `BootReceiver` |
| Geofencing, route deviation, navigation cue banners | `DriverGeofence.kt`, `RouteDeviation.kt`, `NavigationCueAnnouncer.kt` |
| Offline: Room action queue + verifier + sync-queue UI; iOS SwiftData offline delivery store | `DriverOfflineQueue.kt`; `OfflineSyncWorker.kt` (409 ACK / 5xx retry / 4xx dead); iOS `OfflineDeliveryStore.swift` |
| Earnings/history, availability, rescue/reassign handshake, supply transfers, handoff inbox, scanner (real), notifications | `DriverApi.kt:84,177,191-209`; `RequestRescueSheet.kt` |

### Incomplete / decorative / broken

- **Card collection at the door is hostage to backend P0-1** — the driver can complete a card flow whose capture will silently fail server-side (§7).
- **No turn-by-turn navigation engine** — cue banners over backend geometry only.
- **iOS has no durable offline telemetry buffer** (Android buffers telemetry in Room before WS send); no server-side telemetry ACK frame (OkHttp enqueue ≠ delivery proof).
- iOS target-file duplication (`AutoUpdater.swift` ×2).

### Missing features that matter (Driver)

1. **Server-acknowledged telemetry.** *Purpose:* provable location trail (disputes, insurance, SLA). *Why:* fire-and-forget WS send loses points silently. *Logic:* server ACK frame per batch id; client deletes buffer only on ACK; gap detection metric. *E2E:* telemetry batch → server persist → ACK → client purge.
2. **Turn-by-turn navigation.** *Purpose:* stop-level ETAs and driver efficiency. *Why:* ETAs are Haversine heuristics (`eta/calculator.go:21`); real navigation needs OSRM/Valhalla guidance or SDK integration. *Logic:* route geometry exists (`/v1/driver` route geometry endpoint); add maneuver extraction + voice cues; recompute on deviation (deviation detection already exists). *E2E:* manifest → navigate → auto-advance stop → ETA recompute on each location update (already wired at `eta/service.go:249`).
3. **iOS telemetry durability parity.** *Purpose:* platform parity. *Logic/E2E:* mirror Android's insert-before-send with SwiftData buffer + flush worker.
4. **Offline maps tiles.** *Purpose:* dead-zone operation. *Logic:* prefetch corridor tiles on manifest load; bounded cache.
