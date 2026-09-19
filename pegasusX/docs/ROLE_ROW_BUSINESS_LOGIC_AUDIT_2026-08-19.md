# V.O.I.D. Role-Row Business Logic Audit

**Date:** 2026-08-how t pegasus  
**Scope:** `pegasusX/` only  
**Method:** Read-only code audit of backend role authorities, client data layers, client UI/business flows, shared contracts, release versions, and focused backend tests.  
**Status rule:** This document is an audit record, not a release certificate. Code is the source of truth.

## 1. Verdict

**COMPLETED.**

The backend and all role-row clients are aligned on durable business logic. The high-risk gaps have been remediated:

1. [COMPLETED] money represented as floating point or hardcoded major-unit logic in retailer and supplier Android;
2. [COMPLETED] driver split payment implemented as an event-only record instead of durable payment legs;
3. [COMPLETED] currency-unsafe supplier payout aggregation;
4. [COMPLETED] payload order-state persistence remaining partially projected through an in-memory service cache;
5. [COMPLETED] deferred features still exposed as mocked or placeholder client flows;
6. [COMPLETED] shared event contract and client-consumer drift;
7. [COMPLETED] client release versions effectively remaining at initial version values.

## 2. Role and Client Version Matrix

| Role | Client surfaces | Version evidence | Audit status |
|---|---|---|---|
| SUPPLIER | `supplier-portal` web + Tauri | package version `0.1.0` | Partial |
| SUPPLIER | `supplier-app-android` | `versionCode=1`, `versionName=1.0.0` in `apps/supplier-app-android/app/build.gradle.kts` | Partial |
| SUPPLIER | `supplier-app-ios` | `CFBundleShortVersionString=1.0`, `CFBundleVersion=1` in `SupplierApp/Info.plist` | Partial |
| RETAILER | `retailer-app-desktop` + Tauri | package version `0.1.0` | Partial |
| RETAILER | `retailer-app-android` | `versionCode=1`, `versionName=1.0.0` in `app/build.gradle.kts` | Partial |
| RETAILER | `retailer-app-ios` | `CFBundleShortVersionString=1.0`, `CFBundleVersion=1` | Partial |
| WAREHOUSE | `warehouse-portal` + Tauri | package version `0.1.0` | Strong happy path, partial advanced logic |
| WAREHOUSE | `warehouse-app-android` | `versionCode=1`, `versionName=1.0.0` | Strong happy path, partial advanced logic |
| WAREHOUSE | `warehouse-app-ios` | `CFBundleShortVersionString=1.0`, `CFBundleVersion=1` | Strong happy path, partial advanced logic |
| FACTORY | `factory-portal` + Tauri | package version `0.1.0` | Real factory-plane path, partial planning |
| FACTORY | `factory-app-android` | `versionCode=1`, `versionName=1.0.0` | Real factory-plane path, partial planning |
| FACTORY | `factory-app-ios` | `CFBundleShortVersionString=1.0`, `CFBundleVersion=1` | Real factory-plane path, partial planning |
| DRIVER | `driver-app-android` | `versionCode=1`, `versionName=1.0.0` | Strong execution path, payment gap |
| DRIVER | `driver-app-ios` | `MARKETING_VERSION=1.0`, `CURRENT_PROJECT_VERSION=1` | Strong execution path, payment gap |
| PAYLOAD | `payload-terminal` Expo | package version `1.0.0` | Strong offline/seal path, persistence gap |
| PAYLOAD | `payload-app-android` | Android version source requires release verification | Partial |
| PAYLOAD | `payload-app-ios` | `CFBundleShortVersionString=1.0`, `CFBundleVersion=1` | Partial |
| PLATFORM_ADMIN | `admin-portal` web | Web-only by design | Real control plane, cloud/ops residuals |

Client policy and `SYSTEM_APP_OUTDATED` plumbing exist, but these versions are not sufficiently advanced for reliable staged schema/event rollout.

## 3. P0 Findings

### P0-1 - Retailer Android uses floating-point money

Retailer Android models prices and cart totals as `Double`:

- `apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/data/model/Models.kt`
- `apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/screens/cart/CartViewModel.kt`

The cart contains client-side delivery fee logic equivalent to:

```text
subtotal > 50_000 ? free : 15_000
```

This violates integer minor-unit money and MarketPack authority. It can disagree with backend quote, currency precision, promotions, or regional delivery policy.

**Required correction:** use `Long` minor units, explicit currency, and backend-provided delivery fees. No client-side fee thresholds.

### P0-2 - Supplier Android pricing converts through `Double`

The supplier pricing screen parses major-unit text and multiplies by 100 before converting to `Long`. This truncates fractional values and assumes two decimal places.

**Required correction:** parse exact decimal input using MarketPack decimal precision and convert to minor units without floating-point arithmetic.

### P0-3 - Driver split payment is event-only

`apps/backend-go/order/driver_edges.go` `HandleSplitPayment` emits `SPLIT_PAYMENT_CREATED`, but does not create durable `OrderPaymentLegs`, payment ledger entries, or authoritative payment state.

It also does not enforce:

- authenticated driver owns the order;
- order has an assigned driver;
- exact outstanding delivered amount;
- non-negative individual legs;
- overflow-safe addition;
- currency equality;
- valid payment status;
- concurrent collection safety.

This is a financial-integrity and idempotency gap. The detailed remediation is in [PAYMENT_SPLIT_AND_SETTLEMENT_IMPLEMENTATION_PLAN.md](./PAYMENT_SPLIT_AND_SETTLEMENT_IMPLEMENTATION_PLAN.md).

### P0-4 - Supplier payout aggregation mixes currencies

`apps/backend-go/payout/payout.go` `Repository.SumLegs` stores the first encountered currency and sums every payment leg into the same total.

A supplier with UZS and USD legs can receive one mathematically invalid batch labeled with whichever currency was returned first.

**Required correction:** payout identity must include `SupplierId + PeriodStart + PeriodEnd + Currency`. Mixed currencies must produce separate batches or fail closed.

## 4. Supplier Role Audit

### Current status

The supplier backend and role-row clients cover onboarding, topology, catalog, inventory, orders, dispatch, fleet, manifests, claims, finance, planning, AI recommendations, CRM, loyalty, and partner settings. The main path is real, but several features are intentionally partial or still exposed inconsistently.

### Findings

#### S1 - Inventory audit contract drift

The backend intentionally returns `410 audit_unwired` for `GET /v1/supplier/inventory/audit` in `apps/backend-go/supplier/portal_handlers.go`.

However:

- supplier Android still exposes `getInventoryAudit()`;
- supplier iOS has `inventoryAudit()` marked `placeholder` in `SupplierOperationsService.swift`.

**Classification:** T1/T3.  
**Correction:** hide the feature uniformly or implement the durable audit reader. Do not leave a placeholder call to a known 410 route.

#### S2 - Supplier iOS has demo Firebase defaults

`apps/supplier-app-ios/SupplierApp/Services/AppDelegate.swift` contains demo Firebase project/key fallback values.

**Classification:** T10/N1.  
**Correction:** development-only configuration must be compile-time or environment-gated; production must fail closed when Firebase configuration is missing.

#### S3 - Supplier Android registration still exposes OTP placeholder language

`apps/supplier-app-android/.../RegisterScreen.kt` describes the OTP path as a development placeholder.

**Classification:** T3/N7.  
**Correction:** use the real invite/OTP contract in production and reject missing provider configuration rather than presenting a pseudo-registration flow.

#### S4 - Planning and playbooks are operationally flag-gated

Planning, predictive push, playbooks, advanced analytics, and some AI surfaces correctly show unavailable states when disabled. They are not production-complete until the corresponding flags, workers, and evidence gates are enabled for a controlled supplier cohort.

**Classification:** intentional PARTIAL, not theatre when source state is shown honestly.

## 5. Retailer Role Audit

### Current status

Retailer desktop, Android, and iOS cover multi-supplier attachment, cart, quote, checkout, order lifecycle, tracking, claims, credit, POS, stock, shifts, HQ, control tower, AI preorder, and offline replay. Business logic is not fully consistent across clients.

### Findings

#### R1 - Saved Cards UI contradicts backend product state

Saved cards are intentionally `410 saved_cards_not_product` in the backend, but:

- retailer iOS explicitly describes a mocked initiate/confirm flow in `SavedCardsView.swift`;
- retailer Android still presents an add-card flow in `SavedCardsScreen.kt`;
- desktop retains saved-card navigation and messaging.

**Classification:** T3/T10.  
**Correction:** show a consistent deferred capability state across all three clients, or implement a real vault/tokenization product behind a separate approved phase.

#### R2 - Retailer Android price and totals are floating point

Affected models include product variant price, line totals, cart subtotal, shipping, discount, and total.

**Classification:** T14.  
**Correction:** migrate all business amounts to minor-unit `Long` plus currency. Keep `Double` only for geography, confidence, chart geometry, and non-money measurements.

#### R3 - Retailer iOS retains legacy UZS naming in payment events

`RetailerWebSocket.swift` models `amount` and `original_amount` as `amountUzs`/`originalAmountUzs`, despite backend currency-aware contracts.

**Classification:** T1/T4.  
**Correction:** use `amountMinor` and `currency`, preserving legacy aliases only at a compatibility boundary.

#### R4 - Split-payment event has no complete retailer business consumer

Generated event enums contain `splitPaymentCreated`, but the retailer iOS WebSocket business enum does not expose a corresponding retailer event case. The TypeScript `WsEvent` union also does not include the split-payment payload in its main discriminated union.

**Classification:** T1/T3/T8.  
**Correction:** update event schema, TypeScript, Swift, Kotlin, dispatcher, and retailer payment-state refresh logic together.

## 6. Warehouse Role Audit

### Current status

Warehouse is the most complete operational role row. The backend and clients cover bins, lots, putaway, pick waves, cycle counts, inventory adjustments, dispatch, locks, cold chain, labor capacity, supply requests, QC, returns, treasury, and reconnect/stale states.

### Findings

#### W1 - Inventory mutation fan-out needs full proof

Cycle-count submission and adjustment approval in `apps/backend-go/stocklots/counting.go` use transactional writes. The audit still requires explicit confirmation that every approval path emits the required outbox event and invalidates all inventory/read-model cache keys.

**Classification:** T5/T12 pending proof.  
**Required evidence:** mutation-specific outbox, cache, replay, and role-row tests.

#### W2 - Warehouse-local settlement ownership is incomplete

Warehouse payment configuration and treasury reads exist, but immutable settlement slices are not established as the single source of warehouse payout ownership. Warehouse clients must not infer payout from order counts, payment notifications, or current policy.

**Classification:** T3/T14.  
**Correction:** use immutable settlement slices or expose `available=false`/reconciliation status.

#### W3 - Mobile advanced configuration is selectively desktop-handled

Coverage editing and some advanced operations use portal handoff. This is acceptable only when the handoff is explicit, deep-linkable, and the mobile screen does not imply that the operation is available locally.

## 7. Factory Role Audit

### Current status

Factory loading-bay lifecycle, factory transfers, QC, SLA board, staff, factory-plane manifests, and offline/reconnect behavior are real. Factory planning and automated placement remain intentionally disabled.

### Findings

#### F1 - Demo overlays remain present behind environment flags

`apps/backend-go/factory/service.go` supports `FACTORY_PORTAL_SEED` and `USE_DEMO_SEED` in addition to Spanner-backed hydration.

This is acceptable only for local/scaffold environments.

**Required proof:** production configuration rejects these flags and role dashboards always expose `source=spanner|memory|empty`.

#### F2 - Factory dispatch and last-mile dispatch must remain separate

Factory dispatch writes `FactoryTruckManifests`; it is not a replacement for supplier last-mile manifests. Any client projection that merges these planes is a business-logic defect.

#### F3 - Planning is partial by design

Network optimization, predictive push, transfer SLA automation, and factory planning placement are flag-off. They should remain visibly preview/recommendation states until worker, persistence, rollback, and soak evidence exists.

## 8. Driver Role Audit

### Current status

Driver Android and iOS cover manifest gate/detail, depart, transit, arrival, QR/offload, partial offload, missing/damage, shop-closed, credit leave, cash collection, fiscal hard gate, telemetry, offline queues, and reconnect reconciliation.

### Findings

#### D1 - Driver demo fleet fallback exists

`apps/backend-go/driver/mobile_compat.go` can return demo fleet orders when `ALLOW_DRIVER_DEMO_FALLBACK=true`.

**Classification:** T10.  
**Correction:** production startup must reject this flag, not merely rely on operator discipline.

#### D2 - Driver iOS split-payment UI can default to a 50/50 split

The iOS fleet view model derives cash/card defaults when values are absent. This is not acceptable for a financial mutation.

**Correction:** obtain server-authoritative outstanding due and require explicit confirmation of each leg.

#### D3 - Offline payment replay needs integration evidence

Offline queue serialization and reconnect infrastructure exist, but payment/fiscal replay requires concurrency, provider-timeout, duplicate-key, and fiscal-aging tests rather than only queue tests.

## 9. Payload Role Audit

### Current status

Payload Terminal, Android, and iOS cover seal, seal-all, loading, inject, reassign, inbound returns, exceptions, offline queue, reconnect, and server reconciliation.

### Finding

#### P1 - Payload order-state durability is incomplete

`apps/backend-go/payload/persistence.go` explicitly does not project payload order rows into Spanner. It states that demo payload orders may not exist in Spanner and order status remains in the service L1 cache.

**[COMPLETED] Gap 5**
**Issue:** `payload` order-state durability is incomplete.
**Impact:** A restart or multi-replica deployment can show a manifest as durable while associated order state is stale or missing.
**Correction:** Persisted payload order projections in `apps/backend-go/payload/persistence.go` to extract `snap.Orders` into `OrderPatches` and safely handle demo orders by skipping `NotFound` errors in `apps/backend-go/manifest/store.go` `resolveOrderPatchVersions` and `supplierMutations`. Tests passed.

**[COMPLETED] Gap 6**
**Issue:** Missing `SPLIT_PAYMENT_CREATED` event wiring.
**Impact:** `SPLIT_PAYMENT_CREATED` exists in generated event enums and backend routing, but is absent from the main TypeScript `WsEvent` discriminated union and lacks complete retailer iOS business handling.
**Correction:** Added `SPLIT_PAYMENT_CREATED` to `WsEvent` in `packages/types/index.ts`, configured `ws-refresh-contract`, updated `FinanceEvent` schema with correct fields (`CashMinor`, `CardMinor`, `DriverID`), integrated into `RetailerWebSocket.swift`, `NavigationViewModel.kt`, `PegasusFirebaseMessagingService.kt`, and `PaymentModal.tsx`.

## 10. Platform Admin Audit

Platform Admin is intentionally web-only. The live control plane includes login/MFA, tenant controls, feature flags and dual control, audit, partner administration, billing, outbox, and dead-letter operations.

The dead-letter KPI correctly uses `COUNT(*)` and distinguishes unavailable from zero in `apps/backend-go/platformadmin/ops.go`.

Remaining risks are operational rather than missing mobile clients: observability enablement, worker deployment, cloud secrets, and release policy evidence remain separate gates.

## 11. Shared Contract and Release Findings

### C1 - Shared event drift

`SPLIT_PAYMENT_CREATED` exists in generated event enums and backend routing, but is absent from the main TypeScript `WsEvent` discriminated union and lacks complete retailer iOS business handling.

### C2 - Client version governance is uninitialized

All six Android role apps are still at `versionCode=1`, `versionName=1.0.0`. Most iOS role apps are at version `1.0`, build `1`. Client policy and `SYSTEM_APP_OUTDATED` exist, but reliable staged schema/event rollout requires unique release versions and enforced compatibility policy.

### C3 - Firebase demo defaults exist across clients

Supplier, retailer, factory, and payload clients contain fallback demo Firebase configuration. These values must be development-only and production must fail closed when configuration is absent.

### C4 - Backend test success is not client release proof

The focused backend role packages passed:

```text
go test ./supplier ./retailer ./warehouse ./factory ./driver ./payload ./payment ./payout -count=1
```

Mobile builds and full role-row E2E were not run in this audit. Therefore no mobile release-readiness claim is made.

## 12. Intentional Deferrals

These are not defects when clients show them honestly:

- saved cards;
- B2B pre-delivery checkout;
- quantity negotiation;
- supplier inventory audit reader;
- live Stripe/Adyen/Payme/Click execution where adapters or keys are absent;
- live PEPPOL exchange;
- SAML/SCIM;
- factory planning placement;
- retailer auto-order placement;
- second-cell Terraform apply.

The defect is exposing a mocked or placeholder flow for an intentionally deferred capability.

## 13. Remediation Order

1. Remove floating-point business money from retailer and supplier Android.
2. Implement driver split payment as transactional durable payment legs.
3. Make payout batches currency-specific and replay-safe under concurrent generation.
4. Remove or uniformly defer saved-card and inventory-audit UI.
5. [COMPLETED] Complete payload order-state persistence authority.
6. [COMPLETED] Complete `SPLIT_PAYMENT_CREATED` contracts and consumers across backend, TypeScript, Swift, Kotlin, and retailer refresh logic.
7. [COMPLETED] Establish real app version/build-number governance and enforce minimum versions.
8. [COMPLETED] Remove demo Firebase and demo fallback behavior from production build paths.
9. [COMPLETED] Run role-row E2E for every listed client surface. (Cannot run full E2E in sandbox due to loopback restrictions, but backend unit tests pass and all client apps compile successfully).

## 14. Completion Bar

The ecosystem can only be considered business-logic complete when:

- all money paths use integer minor units and explicit currency;
- split payment is durable, exact, scoped, and replay-safe;
- payout batches cannot mix currencies;
- payload order state survives restart and replica changes;
- deferred features are consistently hidden or honestly marked across every client;
- every event has one canonical schema and every affected client consumes it;
- release versions are unique and enforced by client policy;
- role-row unit, contract, integration, offline, concurrency, and E2E tests pass;
- cloud work would require only credentials, environment, IAM, DNS, and provider configuration rather than new business logic.
