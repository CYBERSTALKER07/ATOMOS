# 2. Human / Field-Agent Displacement

## 2.1 The 22-step reality check

The field agent's job, decomposed from need-detection to cash in the bank. "Automatable" means a flag, policy, sweeper, or worker exists in code today; each row cites its evidence.

| # | Step | Human needed? | Status today | Evidence |
|---|---|---|---|---|
| 1 | Detect the reorder need | No | **Automated (FLAG-GATED quality)** — demand sensing worker + forecast baseline + predictive push; forecast algo flag-gated off by default | `apps/backend-go/demand/worker_sensing.go`; `apps/backend-go/planning/forecast/`; `apps/ai-worker/predictivepush/analyzer.go:36-66` |
| 2 | Retailer confirms the suggestion | Default yes | **Automatable (FLAG-GATED)** — per-scope auto-order toggles (global/category/supplier/product/variant) + `off\|shadow\|draft\|place` execution mode; default `off` | `apps/backend-go/retailer/auto_order_worker.go`; `retailer/auto_order_policy.go` |
| 3 | Supplier accepts the order | Default yes | **Automatable** — midnight-guard sweeper promotes `SCHEDULED → AUTO_ACCEPTED` | `apps/backend-go/order/preorder_sweeper.go:168` |
| 4 | Credit decision | No | **Automated at placement — limit + status only** (scoring deliberately removed; `RiskTier` blanked) | `apps/backend-go/credit/service.go:49-94` |
| 5 | Price determination | No | **Automated** — override → promotion (basis-points) → price list | `apps/backend-go/promotion/evaluator.go`; `pricing/service.go` |
| 6 | Stock allocation / backorder | No | **PARTIAL** — FEFO/FIFO lot reservation + constrained warehouse selection; **no partial allocation or backorder queue: insufficient stock is a hard error (lost sale)** | `apps/backend-go/stocklots/fefo.go`; `allocation/constrained.go`; `order/inventory_reservation.go:65-92` |
| 7 | Delivery date agreement | Sometimes | **PARTIAL** — auto by default; `NegotiationProposals` two-sided negotiation exists but is flag-gated/product-deferred | `apps/backend-go/order/negotiation.go` |
| 8 | Transfer approval | Sometimes | **Automatable (FLAG-GATED)** — touchless with daily unit budget; CRITICAL urgency escalates by design | `apps/backend-go/replenishment/touchless.go`; `replenishment/policies.go` |
| 9 | Dispatch planning & driver assignment | Optional | **Automatable** — 60s auto-dispatch worker, real closed loop; solver is heuristic in prod (OR-Tools sidecar at 0 replicas) | `apps/backend-go/warehouse/auto_dispatch.go:28,120-131`; `infra/k8s/overlays/prod/kustomization.yaml:44-50` |
| 10 | Physical picking | **YES** | **Software-assisted, portal-only** — pick waves + FEFO + seal gate exist in backend and warehouse-portal; **warehouse mobile apps cannot execute picks**; the Android scanner is a dead stub | `apps/backend-go/stocklots/picking.go`; `warehouse-portal/app/pick-waves/page.tsx`; `warehouse-app-android/.../ScannerViewModel.kt:22,47` |
| 11 | Truck loading | **YES** | Human-driven by design — a 38-endpoint payloader API + loading-bay terminal exist so a human can drive it | `apps/backend-go/payload*/`; `apps/payload-app-android/.../PayloadApi.kt` |
| 12 | Manifest sealing | Yes | PARTIAL — `seal-all` batches the clicks; SSCC minted per ship unit | `packages/api-client` `/v1/payloader/manifests/seal-all`; `gs1/checkdigit.go:142-171` |
| 13 | Driving | **YES** | — | — |
| 14 | QR handshake at the store | **YES — by design** | Geofence auto-detects arrival; the handshake is an intentional two-party control, not a gap | `apps/backend-go/geolocation/`; driver apps |
| 15 | Offload & condition verification | **YES** | Human-reported at dock; post-delivery damage/shortage/concealed file as claims → QUARANTINE + reverse logistics | `apps/backend-go/claims/`; `stocklots/coldchain.go` |
| 16 | Cash collection | **YES — structural** | The most developed manual path in the platform: server-computed expected cash vs declared, accept/write-off, shift-close gate, nightly escalation | `apps/backend-go/cashrecon/service.go:39-161`; `cashrecon/escalation_worker.go` |
| 17 | Card capture | No | **BROKEN TODAY (P0 bug)** — capture routing key mismatch; leg pre-recorded CAPTURED; stub-success when creds empty | `payment/service.go:653` vs `payment/execution.go:140`; `order/service.go:1899-1929`; `payment/global_pay_executor.go:251-258` |
| 18 | Fiscal receipt | No | **Automated shape; NOT legal** — PEGASUS provider issues commercial receipts (`"tax_ofd": false`); legal Soliq OFD adapter exists but its signer is never injected (100% failure if enabled) | `order/fiscal_provider_pegasus.go:78-79`; `order/fiscal_provider.go:129,232-234` |
| 19 | Reconciliation | On exception | PARTIAL — detection automated (cashrecon, settlement exceptions), resolution manual | `apps/backend-go/cashrecon/`; `order/settlement_hardening.go` |
| 20 | Returns disposition | Yes | **Strong** — FileClaim + approve/reject + stock hold + reverse open; human confirms disposition | `apps/backend-go/claims/service.go`; `returns/` |
| 21 | Exceptions (shop closed, delay, overflow, rescue) | **YES** | Every path ends in a human decision; shop-closed timeout worker exists but can mis-record credit debt (P0 risk #4) | `order/worker_shop_closed.go:91-165` |
| 22 | Dunning / collections | Partial | **In-app wired (FLAG-GATED)** — step machine DUE_SOON→…→COLLECTIONS, auto CREDIT_HOLD, FCM+inbox; **off-app SMS/email/WhatsApp absent** | `apps/backend-go/ar/dunning.go:41-74`; `ar/dunning_worker.go` |

## 2.2 What is automatable today — the exact zero-touch path

The complete, code-verified chain for a store replenishment order with no human touch:

1. `demand.RunDemandSensingWorker` (scheduled) computes velocities from order history with day-of-week/payday/signal factors — including a real Open-Meteo weather ingest (`apps/backend-go/demand/worker_weather.go:97-138`) — and upserts `DemandAdjustments`.
2. The after-sensing hook (`apps/backend-go/bootstrap/bootstrap.go:1535`) runs `replenishment.RunBatch` (`replenishment/reorder_suggestion_batch.go`), computing suggested quantities from adjustments + safety stock (service-level formula when `SAFETY_STOCK_V2_ENABLED`).
3. `retailer.RunAutoOrderWorker` (`retailer/auto_order_worker.go`) loads inventory-grounded (R,s,S) proposals from `RetailerStockBalances`, applies the scope policy filter, and — **only when `execution_mode=place`** — writes a real order via `order.Service.Create`.
4. Credit gate runs inline at creation (`credit/service.go:49-94`); inventory is reserved in the same transaction (`order/inventory_reservation.go:65-92`).
5. Supplier-side, `replenishment/touchless.go` auto-approves within policy caps; the 60s auto-dispatch worker commits manifests (`warehouse/auto_dispatch.go:85-90`).

**Where the chain breaks by default:** `execution_mode` defaults to `off`; `draft` requires human confirmation; supplier touchless requires `AutoApproveEnabled` + confidence floor; the forecast algorithm and safety-stock v2 flags are off; and after dispatch, the physical world takes over (steps 10–16 above).

## 2.3 What remains a hard human requirement

- **Physical execution**: picking, loading, driving, offload verification. The platform's correct design choice is to *instrument* these humans (dedicated payload terminal, driver apps, geofenced handshake), not pretend them away.
- **Cash collection**: in a COD-dominant market the driver *is* the collections function. The platform converts that human from sales to logistics; it does not remove him.
- **Relationship commerce**: negotiating shelf space, promo calendars, dispute diplomacy, and reading a store the way a rep does. The negotiation primitive exists for delivery dates only; price/assortment negotiation is absent.
- **Off-app collections**: retailers without the app are unreachable by the dunning engine (no SMS/email/WhatsApp transports).

## 2.4 Realistic trajectory

**Near-term (6–12 months, assuming P0/P1 gaps in §8 close):** the honest product is **"replace the order pad, instrument the rest."** For app-adopting urban retailers, routine replenishment visits largely disappear; the field force re-composes toward exception handling, onboarding, collections, and relationship management. Expect headcount *shift* (sales → logistics/collections), not headcount deletion.

**3–5 year view (if all gaps close):** for the routine-replenishment slice — which is the majority of agent visit volume in FMCG — touchless operation is credible: forecast + inventory-grounded auto-order with proven shadow acceptance, touchless supplier approval, optimized dispatch, fiscalized completion, AR dunning with off-app reach. Even then: cash collection remains structural where COD persists; new-store acquisition, key-account negotiation, and exception recovery remain human. **Realistic outcome is a 50–70% reduction of routine field visits for covered retailers — a hybrid reduction, not a wipe-out.** The market evidence agrees: the best-funded B2B commerce platforms (MaxAB-Wasoko, Udaan, Jumbotail) all converged on monetizing **credit and data on top of instrumented transactions**, not on eliminating the field layer entirely.
