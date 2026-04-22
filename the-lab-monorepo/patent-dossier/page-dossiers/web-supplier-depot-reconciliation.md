**Generatedat:** 2026-04-06

**Pageid:** web-supplier-depot-reconciliation

**Route:** /supplier/depot-reconciliation

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/depot-reconciliation/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier depot-reconciliation page for processing quarantined returned loads by vehicle, order, and individual line item, with restock and write-off actions.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

- headline: Depot Reconciliation
- subtitle describing returned loads awaiting restock or write-off
- Refresh button

---

**Zoneid:** vehicle-card-stack

**Position:** main vertical stack

## Contents

- one vehicle card per quarantined vehicle with header summary and nested order sections

---

**Zoneid:** vehicle-card-header

**Position:** top of each vehicle card

## Contents

- fleet icon avatar
- vehicle class, driver name, route identifier, and order count
- Restock All button
- Write Off All button

---

**Zoneid:** order-section

**Position:** within each vehicle card

## Contents

- order short ID and retailer name
- quarantine pill
- item table with product, quantity, unit price, and per-item actions

---


# Buttonplacements

**Button:** Retry

**Zone:** error fallback state

**Style:** secondary button

---

**Button:** Refresh

**Zone:** header top-right

**Style:** outline button with leading icon

---

**Button:** Restock All

**Zone:** vehicle-card-header action cluster

**Style:** secondary small button

---

**Button:** Write Off All

**Zone:** vehicle-card-header action cluster

**Style:** outline danger small button

---

**Button:** Restock

**Zone:** item row actions

**Style:** secondary small button

---

**Button:** Write Off

**Zone:** item row actions

**Style:** outline danger small button

---


# Iconplacements

**Icon:** error

**Zone:** error fallback

---

**Icon:** warehouse

**Zone:** empty state

---

**Icon:** refresh

**Zone:** Refresh button

---

**Icon:** fleet

**Zone:** vehicle-card header avatar

---


# Interactiveflows

**Flowid:** quarantine-bootstrap

## Steps

- Page reads supplier token and fetches /v1/supplier/quarantine-stock
- Loading placeholders occupy the card stack until quarantine vehicles resolve

---

**Flowid:** bulk-vehicle-reconciliation

## Steps

- Supplier presses Restock All or Write Off All in a vehicle card header
- All line_item_ids for that vehicle are posted to /v1/inventory/reconcile-returns with the selected action
- Toast confirms success and vehicle card stack reloads

---

**Flowid:** item-level-reconciliation

## Steps

- Supplier presses Restock or Write Off in a line-item row
- Single line_item_id is posted to /v1/inventory/reconcile-returns
- Row and enclosing vehicle dataset refresh after reconciliation

---


# Datadependencies

## Readendpoints

- /v1/supplier/quarantine-stock

## Writeendpoints

- /v1/inventory/reconcile-returns

**Refreshmodel:** load on mount, manual refresh button, and automatic reload after reconciliation actions

# Statevariants

- loading skeleton stack
- error fallback with retry button
- empty state with no quarantine stock
- vehicle stack with nested order sections
- action-in-progress state disabling reconciliation buttons

# Figureblueprints

- full depot reconciliation page with stacked vehicle cards
- single vehicle card header with bulk action buttons
- order section close-up with quarantine pill and per-item restock and write-off controls

