**Generatedat:** 2026-04-06

**Pageid:** web-supplier-orders

**Route:** /supplier/orders

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/orders/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier order-lifecycle command page for approval, reassignment, search, filtering, history viewing, and order inspection.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

### Left

- headline: Orders
- subtitle: Manage the full order lifecycle — approval, dispatch, tracking, and history

### Right

- Refresh button

---

**Zoneid:** tab-row

**Position:** below divider

## Contents

- Active tab button with orders icon and count badge
- Scheduled tab button with schedule icon and count badge when active

---

**Zoneid:** filter-row

**Position:** below tabs

## Contents

- search input for order ID or retailer
- state filter select
- History toggle chip with ledger icon

---

**Zoneid:** bulk-action-bar

**Position:** conditional row above table

**Visibilityrule:** visible when one or more rows are selected

## Contents

### Left

- selected count

### Right

- Reassign Truck button when selection is eligible
- Clear button

---

**Zoneid:** table-region

**Position:** primary content card

## Contents

- select-all checkbox header
- order ID column
- retailer column
- state badge column
- truck or route column
- delivery date column
- items column
- amount column
- payment column
- created timestamp column
- actions column

---


# Buttonplacements

**Button:** Refresh

**Zone:** header-right

**Style:** secondary

**Icon:** returns

---

**Button:** Active tab

**Zone:** tab-row-left

**Style:** tab

**Icon:** orders

---

**Button:** Scheduled tab

**Zone:** tab-row-left

**Style:** tab

**Icon:** schedule

---

**Button:** History

**Zone:** filter-row-right

**Style:** chip-toggle

**Icon:** ledger

---

**Button:** Approve

**Zone:** table-row-actions

**Style:** primary

**Visibilityrule:** active tab and order state is PENDING

---

**Button:** Reject

**Zone:** table-row-actions

**Style:** outline-danger

**Visibilityrule:** active tab and order state is PENDING

---

**Button:** Reassign

**Zone:** table-row-actions

**Style:** secondary

**Visibilityrule:** active tab and order state is PENDING or LOADED and route_id exists

---

**Button:** Reassign Truck

**Zone:** bulk-action-bar-right

**Style:** primary

**Visibilityrule:** selected rows all eligible for reassignment

---

**Button:** Clear

**Zone:** bulk-action-bar-right

**Style:** ghost

---

**Button:** Cancel

**Zone:** reassign-dialog-footer-left

**Style:** ghost

---

**Button:** Reassign N Order(s)

**Zone:** reassign-dialog-footer-right

**Style:** primary

---

**Button:** Reassign to Different Truck

**Zone:** detail-drawer-footer

**Style:** secondary

**Visibilityrule:** drawer open and order has route_id and state is PENDING or LOADED

---


# Iconplacements

**Icon:** returns

**Zone:** header-right refresh button

---

**Icon:** orders

**Zone:** Active tab button

---

**Icon:** schedule

**Zone:** Scheduled tab button

---

**Icon:** ledger

**Zone:** History filter chip

---

**Icon:** StatusBadge

**Zone:** state column and detail drawer

---


# Interactiveflows

**Flowid:** approve-pending-order

## Steps

- Supplier stays on Active tab
- Clicks Approve in row actions
- Page posts to /v1/supplier/orders/vet
- Toast shows result
- Orders reload

---

**Flowid:** reject-pending-order

## Steps

- Supplier clicks Reject in row actions
- Inline reason input appears in actions column
- Supplier types reason and confirms Reject
- Page posts to /v1/supplier/orders/vet with decision REJECTED
- Toast shows result and rows reload

---

**Flowid:** single-or-bulk-reassign

## Steps

- Supplier clicks Reassign in a row or selects multiple eligible rows
- Dialog opens
- Supplier selects target truck
- Capacity metrics and capacity bar render
- Supplier confirms reassignment
- Page posts to /v1/fleet/reassign
- Toast summarizes reassign or conflict results

---

**Flowid:** row-inspection

## Steps

- Supplier clicks any table row
- Order detail drawer opens from right
- Drawer shows ID, status, retailer, payment, assignment, timestamps, and optional reassign CTA

---


# Datadependencies

## Readendpoints

- /v1/supplier/orders
- /v1/fleet/active
- /v1/fleet/capacity

## Writeendpoints

- /v1/supplier/orders/vet
- /v1/fleet/reassign

**Refreshmodel:** manual refresh plus 30-second polling when not in history mode

# Statevariants

- loading skeleton rows
- empty state by active or scheduled or history context
- normal data table
- selected rows accent-soft highlighting
- reject inline-input mode
- detail drawer open
- reassignment dialog open

# Figureblueprints

- full-page command view with tab row and filter row
- table row with approve and reject controls
- bulk-action bar with selected rows and reassign CTA
- reassignment dialog with capacity bar
- order detail drawer

