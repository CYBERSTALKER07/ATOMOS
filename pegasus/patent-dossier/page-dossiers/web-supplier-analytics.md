**Generatedat:** 2026-04-06

**Pageid:** web-supplier-analytics

**Route:** /supplier/analytics

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/analytics/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier analytics hub presenting financial velocity, AI demand highlights, and deep links into advanced forecast review and dispatch operations.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

- headline: Analytics
- subtitle describing financial overview and operational intelligence
- Demand Forecast CTA
- Dispatch Room CTA

---

**Zoneid:** ai-demand-card

**Position:** below header when demand predictions exist

**Visibilityrule:** visible when demand prediction_count is greater than zero

## Contents

- analytics avatar circle
- prediction count chip
- three-column forecast metrics
- forecasted SKU pills
- View Advanced Analytics CTA

---

**Zoneid:** kpi-grid

**Position:** below demand card or directly below header

## Contents

- Gross Volume card
- Total Pallets card
- Avg Velocity per SKU card
- Top SKU card

---

**Zoneid:** velocity-chart-region

**Position:** below KPI grid

## Contents

- VelocityChart component spanning page width

---

**Zoneid:** sku-breakdown-table

**Position:** bottom full-width card when velocity data exists

## Contents

- table with SKU ID, pallets, volume, and share bar

---


# Buttonplacements

**Button:** Demand Forecast

**Zone:** header top-right

**Style:** soft accent link button

---

**Button:** Dispatch Room

**Zone:** header top-right

**Style:** filled accent link button

---

**Button:** View Advanced Analytics

**Zone:** ai-demand-card footer

**Style:** pill button

**Visibilityrule:** visible when demand card is rendered

---


# Iconplacements

**Icon:** error

**Zone:** error card

---

**Icon:** analytics

**Zone:** Demand Forecast header CTA and AI demand card avatar

---

**Icon:** orders

**Zone:** Dispatch Room header CTA

---

**Icon:** arrow_forward

**Zone:** View Advanced Analytics CTA trailing icon

---


# Interactiveflows

**Flowid:** analytics-bootstrap

## Steps

- Page fetches /v1/supplier/analytics/velocity and /v1/supplier/analytics/demand/today in parallel using apiFetch
- Loading skeletons occupy the KPI and chart regions until both requests settle
- Velocity metrics derive from the returned data array and top SKU is computed client-side

---

**Flowid:** forecast-escalation

## Steps

- Supplier selects Demand Forecast in the header or View Advanced Analytics in the AI demand card
- Navigation moves to /supplier/analytics/demand for deeper analysis

---

**Flowid:** dispatch-escalation

## Steps

- Supplier selects Dispatch Room
- Navigation leaves analytics and returns to the operational command surface linked by the app root

---


# Datadependencies

## Readendpoints

- /v1/supplier/analytics/velocity
- /v1/supplier/analytics/demand/today

## Writeendpoints


**Refreshmodel:** single load on mount with computed metrics derived in memory

# Statevariants

- skeleton loading state
- error card state
- analytics hub with AI demand card visible
- analytics hub without AI demand card
- velocity chart with SKU breakdown table
- analytics hub with no velocity rows and no bottom table

# Figureblueprints

- full analytics hub with AI demand card, KPI grid, chart, and breakdown table
- AI demand card close-up with metric triplet and forecast pills
- SKU breakdown row with share bar

