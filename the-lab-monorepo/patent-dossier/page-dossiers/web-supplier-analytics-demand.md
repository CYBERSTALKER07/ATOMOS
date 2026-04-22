**Generatedat:** 2026-04-06

**Pageid:** web-supplier-analytics-demand

**Route:** /supplier/analytics/demand

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/analytics/demand/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier advanced demand-forecast page comparing predicted versus actual order volume over time and listing upcoming AI-planned order line items.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

- circular back button linking to analytics hub
- headline: AI Demand Analytics
- subtitle describing predicted versus actual volume over a 30-day window

---

**Zoneid:** kpi-row

**Position:** below header

## Contents

- Prediction Accuracy card
- Upcoming AI Orders card
- Data Points card

---

**Zoneid:** chart-card

**Position:** below KPI row

## Contents

- dual-axis line chart
- legend
- tooltip-driven predicted and actual value inspection
- empty chart message when time series is absent

---

**Zoneid:** upcoming-orders-card

**Position:** bottom full-width

## Contents

- table header
- upcoming AI-planned order rows with date, retailer, SKU, product, predicted quantity
- pagination controls or empty-state message

---


# Buttonplacements

**Button:** Back to analytics

**Zone:** header far-left circular control

**Style:** round surface button

---


# Iconplacements

**Icon:** left-arrow glyph

**Zone:** header back button

---


# Interactiveflows

**Flowid:** demand-history-bootstrap

## Steps

- Page reads supplier token from cookie
- If token is absent, page renders supplier-credentials-required error state
- Otherwise page fetches /v1/supplier/analytics/demand/history and binds time series plus upcoming rows

---

**Flowid:** chart-review

## Steps

- Supplier inspects dual-axis lines for predicted and actual value and quantity
- Tooltip exposes exact UZS and quantity values for a selected date

---

**Flowid:** upcoming-order-review

## Steps

- Supplier reviews paginated upcoming AI-planned order rows
- Shared pagination controls advance through the upcoming dataset

---


# Datadependencies

## Readendpoints

- /v1/supplier/analytics/demand/history

## Writeendpoints


**Refreshmodel:** single fetch on mount with local pagination over the upcoming rows

# Statevariants

- page loading spinner
- unauthorized error card
- history error card
- chart with time-series data
- chart empty state with no time-series data available
- upcoming rows table with pagination
- upcoming rows empty message

# Figureblueprints

- full advanced demand analytics page with header, KPI row, chart, and table
- chart card close-up showing four line series and legend
- upcoming orders table close-up with pagination footer

