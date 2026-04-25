# Maglev Read-Router Rollout Completion

## Scope Guardrails (Phase 0)
- Core router algorithm in `apps/backend-go/bootstrap/spannerrouter/router.go` is unchanged from HEAD baseline.
- No API response shape changes were introduced in rollout files.
- No write-path routing changes were introduced; rollout changes only adjust geo-scoped read-client selection.

## Phase Coverage
- Phase 1: Shared fallback-safe read-client selection helper exists in `apps/backend-go/proximity/read_router.go`.
- Phase 2: Order cold-start and line-item history read paths route via `ReadRouter` with primary fallback.
- Phase 3: Payloader recommend-reassign reads route via retailer-context read client; route deps include `ReadRouter`.
- Phase 4: Supplier pricing scope checks and dispatch/fleet-preview reads route via geo context.
- Phase 5: Warehouse demand forecast and ops orders list/detail route via warehouse-context read client.

## Cross-Surface Verification (Phase 6)
- Backend module build and targeted vet/tests are run from package roots.
- AI-worker runtime code compiles, and line-item history consumer schema matches backend keys:
  - `skuId`, `productId`, `categoryId`, `supplierId`, `warehouseId`
  - `quantity`, `unitPrice`, `orderDate`, `minimumOrderQty`, `stepSize`
- Client call paths confirmed for:
  - Supplier dispatch surfaces (`auto-dispatch`, `dispatch-recommend`, dispatch preview)
  - Payload recommend-reassign on terminal, iOS, and Android
  - Warehouse ops orders and demand forecast views

## Hardening and Rollout Readiness (Phase 7)
- Focused fallback tests added or extended for shared read-router helpers.
- Signature seam test added for payloader route registration with `ReadRouter` in deps.
- Regression checks confirm mutation codepaths remain on primary write client.

## Canary Sequence
1. Deploy backend canary to 5 percent traffic for supplier and warehouse read-heavy endpoints only.
2. Monitor endpoint error rate and latency for:
   - `/v1/orders/line-items/history`
   - `/v1/payloader/recommend-reassign`
   - `/v1/supplier/manifests/{auto-dispatch,dispatch-recommend}`
   - `/v1/supplier/dispatch-preview`
   - `/v1/warehouse/ops/orders`
   - `/v1/warehouse/demand/forecast`
3. Track fallback ratio (`router unavailable or geo unresolved`) versus routed ratio.
4. Expand to 25 percent traffic if p95 latency is improved or neutral and 5xx does not regress.
5. Expand to 50 percent, then 100 percent, with the same gate checks at each step.
6. Roll back immediately if any endpoint shows sustained 5xx increase or schema regressions.
7. After full rollout, retain fallback/routing telemetry for 7 days and close canary.
