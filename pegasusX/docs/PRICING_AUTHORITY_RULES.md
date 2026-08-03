# Pricing Authority Rules

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



## Purpose

Define who owns list prices, promotions, and checkout totals so support and clients never invent money.

## Current runtime authority (pegasusX)

1. **Catalog list price** lives in Spanner `Products.PriceMinor` (supplier catalog).
2. **Per-retailer overrides** live in supplier pricing/override tables and are applied server-side.
3. **Promotions** (`promotion` package) discount **server** list/override prices only.
4. **Order create** (`POST /v1/order/create` / retailer create paths) runs `normalizeAndQuoteLineItems` — **client `unit_price_minor` is ignored** when Spanner is available; missing SKUs fail closed.
5. **Unified checkout / preview** (`authoritativeCheckoutLines`) uses the same catalog authority when Spanner is available, then optional promo quote. Scaffold/unit tests without Spanner may still accept client unit prices.
6. **Supplier APIs:** `GET|PATCH /v1/supplier/pricing/rules`, retailer-overrides, promotions CRUD.

## Support-facing policy

1. Treat supplier catalog + overrides as canonical.
2. Retailer price disputes: compare catalog/override/promo at order `created_at`, not cart UI alone.
3. Do not tell clients to hardcode prices as a workaround.
4. Escalate with request/response + order id + SKU when totals disagree with policy.

## Operational handling

| Category | Definition | Owner |
|----------|------------|--------|
| Policy clarification | Expected tier/promo not understood | Supplier ops |
| Suspected backend defect | Reproducible mismatch vs catalog | Backend platform |
| Client stale cache | UI showed old price; server charged catalog | Retailer support (refresh) |
| Scope mismatch | Wrong supplier/retailer context | Product + backend |

## Evidence for defect escalation

1. Retailer id + supplier id  
2. SKU list + quantities  
3. Request payload and response totals  
4. UTC timestamp  
5. Expected catalog `PriceMinor` / override / promo at that time  

## Non-goals (v1)

- Client-trusted unit prices in production (Spanner present).  
- Quantity negotiation (410 `feature_disabled`).  
