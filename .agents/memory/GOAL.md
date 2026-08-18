# FINAL GOAL (load on every new session)

**Set:** 2026-08-16  
**Product tree:** `pegasusX/` (`pegasus/` = legacy port source)

## Canonical program — these files ARE the destination

| File | What it is |
|------|------------|
| [`pegasusX/docs/GLOBAL_SCALE_PROGRAM.md`](../../pegasusX/docs/GLOBAL_SCALE_PROGRAM.md) | Global multi-supplier program: register + home cell + MarketPack + Class A. Phases **GS-A → T → M → C**, then I / R / P. |
| [`pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md`](../../pegasusX/docs/GLOBAL_SCALE_LOCAL_ECOSYSTEM.md) | Local-first same-market topology + pack-owned PSP. Phases **GS-L0–L4** + **GS-K1–K3** (W1–W26). Extends A/T/M/C; does not replace them. |
| [`pegasusX/docs/GLOBAL_SCALE_CLIENT_UI.md`](../../pegasusX/docs/GLOBAL_SCALE_CLIENT_UI.md) | Client visualization: command dashboards + Plan & Brain tabs (GS-U0–U9) on web, desktop, iOS (phone+iPad), Android (phone+tablet). Extends GS-R. Not status. |

`PROD_ECOSYSTEM_GOAL.md` is the **Class A coverage rule** (outbox + consumer + role-row client). It is not a competing destination.

This file is the **goal**, not status. Status needs `file:line` this session. Code wins.

## North star

A **global, local-first, multi-supplier** logistics ecosystem.

Many companies, in many markets, **register** as isolated suppliers (`SupplierId`), land in a **home cell**, receive a **market pack that checkout / fiscal / proximity / PSP catalog actually use**, invite their roles, and run Class A:

`order → stock → truck → cash/credit → fiscal → payout`

Retailers attach **more than one** supplier; mixed carts split into per-supplier child orders (`ParentOrders`) **only inside the same pack country**. Same code, **cloned cells**. Not a UZ-only fork. Not a second tenant key.

Local law: factory → warehouse → retailer store is **same-country**, **closest covering node**, with supplier pins as override. Empty geography is incomplete, not worldwide.

## Product laws (do not violate)

From `GLOBAL_SCALE_LOCAL_ECOSYSTEM.md` §1 — product law, not UI preference:

1. **One tenant key** — `SupplierId`. Pack, cell, country, city, region are attributes.
2. **Market owns money** — currency, decimals, PSP list, fiscal adapter, payout rail come from shipped `MarketPack`.
3. **Same-market orders only** — retailer / warehouse / factory / pack country must match. Else `422 cross_market_deferred`.
4. **Local-first default** — closest covering warehouse; closest factory on a `SupplyLane` (or same-country factories).
5. **Supplier override wins** — city / region / store pins beat closest.
6. **Empty geography fail-closed** — missing `CountryCode` → `422 geography_incomplete`.
7. **Pack filters PSP UI** — GET/POST gateways = pack ∩ registered executors.
8. **Unkeyed ≠ success** — missing PSP/fiscal/SMS keys → honest 501 / `no_live_keys`. Never a fake 200 redirect.
9. **One country = one pack + 1–3 adapters** — not a fork.
10. **Class A stays** — integer minor money, fiscal hard-gate, pay-at-delivery, dual manifests, H3 res 7, outbox-in-txn, factory planning / auto-order **place** flag-off.

## How we get there

Implement **phases**, not 250 `BF-*` inventory rows in one PR.

```
GS-A  Auth + session market pack     (A0–A2 claimed in program)
GS-T  Self-serve tenant register     (T1–T5 claimed in program)
GS-M  Checkout/fiscal/maps READ pack (M1–M7 claimed; checkout_reads_this still false)
GS-C  Regional cell scaffold         (C1–C5 plan/files; no apply)
GS-I / R / P                         (bind claimed; leftovers continuous)

GS-L  Local matching                 L0 → L1 → L2 → L3 → L4
GS-K  Pack PSP catalog               K1 → K2 → K3
GS-U  Client visualization           U0 → U9 (dashboards + Plan & Brain)
```

**Next claimed slice:** leftovers (GS-M flag, cells apply, live PSP) — Layer B, do not execute. Named + continuous empty-currency invent train closed. Not Layer B. GS-U0–U9 shipped 2026-08-16. `checkout_reads_this` still false. Not terraform apply. Not Stripe keys. Not flipping the flag. Not swapping MapLibre for Google Maps. Factory planning / auto-order **place** stay off.

A slice is not done until every W-item it owns has a live-path test.

## Not the claim

- Open-world public discovery / ads marketplace
- Cross-country orders, payments, payouts, fiscal, credit, FX checkout
- One global Spanner / second tenant key / merged factory+supplier manifests
- `terraform apply` of `cells/eu` or a second live cell
- Stripe/Adyen/Payme live charges, PEPPOL execute, SAML/SCIM
- Factory planning or auto-order **place** on by default
- Linguistic-complete i18n
- “We listed 50 countries therefore we operate in 50 countries”

## Do not break

- SoT tree is `pegasusX/`
- Dual manifest planes (factory vs supplier trucks)
- Integer money, fiscal hard-gate, pay-at-delivery
- Seed fail-closed in ssmr/prod
- No side vector DB for agents — memory is this folder + Grok `[memory]` + graph walker + live code

## Honesty

Re-verify every status claim in code. Docs and this file are the destination, not a wiring certificate.

Working memory: `.agents/memory/WORKSPACE.md`  
Feature inventory: `pegasusX/docs/GLOBAL_SCALE_BACKEND_FEATURES.md`  
Class A coverage: `pegasusX/docs/PROD_ECOSYSTEM_GOAL.md`
