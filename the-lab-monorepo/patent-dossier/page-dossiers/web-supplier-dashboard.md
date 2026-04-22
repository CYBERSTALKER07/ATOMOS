**Generatedat:** 2026-04-06

**Pageid:** web-supplier-dashboard

**Route:** /supplier/dashboard

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/dashboard/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier analytics landing page combining forward-demand intelligence, SKU velocity summaries, financial KPIs, and volume-share tables.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

### Left

- headline: Analytics
- subtitle: Financial overview and operational intelligence

### Right

- Dispatch Control Room link button with clipboard-style glyph

---

**Zoneid:** future-demand-card

**Position:** below header when predictions exist

**Visibilityrule:** visible when demand payload exists and prediction_count > 0

## Contents

- lightning icon badge
- AI Future Demand headline and helper copy
- prediction-count status chip
- three-metric strip for retailers, pallets, forecast value
- forecast item pills
- View Advanced Analytics CTA

---

**Zoneid:** kpi-grid

**Position:** below future-demand card

## Contents

- Gross Volume card
- Total Pallets Moved card
- Avg Velocity per SKU card
- Top Performing SKU card

---

**Zoneid:** velocity-chart

**Position:** mid-page primary visualization

## Contents

- VelocityChart component showing SKU volume performance

---

**Zoneid:** sku-breakdown-table

**Position:** bottom card

**Visibilityrule:** visible when velocityData length > 0

## Contents

- SKU ID column
- pallet count column
- gross volume column
- share column with right-aligned progress bar and percentage text

---


# Buttonplacements

**Button:** Dispatch Control Room

**Zone:** header-right

**Style:** filled pill link

**Icon:** manifest clipboard glyph

---

**Button:** View Advanced Analytics

**Zone:** future-demand-card footer

**Style:** filled rounded CTA

**Icon:** right arrow

---


# Iconplacements

**Icon:** manifest clipboard glyph

**Zone:** header-right dispatch link

---

**Icon:** lightning bolt

**Zone:** future-demand-card leading badge

---

**Icon:** right arrow

**Zone:** future-demand-card CTA trailing edge

---

**Icon:** progress fill bar

**Zone:** share column in sku-breakdown-table

---


# Interactiveflows

**Flowid:** analytics-bootstrap

## Steps

- Page reads supplier token from cookie
- Page requests /v1/supplier/analytics/velocity and /v1/supplier/analytics/demand/today in parallel
- KPI cards, chart, demand card, and table compute derived totals from returned payloads
- If demand payload is absent, only analytics baseline regions remain

---

**Flowid:** advanced-demand-drilldown

## Steps

- Supplier reviews AI demand summary card
- Supplier clicks View Advanced Analytics
- Page routes to /supplier/analytics/demand

---

**Flowid:** dispatch-linkout

## Steps

- Supplier uses header CTA
- Page routes to /supplier/manifests legacy dispatch surface

---


# Datadependencies

## Readendpoints

- /v1/supplier/analytics/velocity
- /v1/supplier/analytics/demand/today

## Writeendpoints


**Refreshmodel:** single fetch on mount; data remains static until navigation or reload

# Statevariants

- page-level loading skeleton with placeholder header, KPI cards, and chart block
- unauthorized error card
- analytics-only state with no demand card
- full intelligence state with demand card and forecast pills
- table hidden when velocityData is empty

# Figureblueprints

- full analytics dashboard with future-demand card and KPI grid
- AI Future Demand card close-up with forecast pills
- velocity chart region
- SKU breakdown table with share bar column

