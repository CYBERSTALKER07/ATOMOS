**Generatedat:** 2026-04-06

**Pageid:** web-supplier-inventory

**Route:** /supplier/inventory

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/inventory/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier stock-control page for quantity adjustments, low-stock visibility, and immutable audit-log inspection.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

- headline: Inventory Management
- subtitle: Stock levels, replenishment controls, and audit trail

---

**Zoneid:** tab-switcher

**Position:** below header

## Contents

- Stock Levels segmented chip
- Audit Log segmented chip

---

**Zoneid:** stock-card

**Position:** primary card when stock tab active

## Contents

- SKU count label
- Refresh button
- inventory table with product, sku, stock, action columns
- inline adjustment editor replacing action cell when row is in adjust mode
- pagination controls

---

**Zoneid:** audit-card

**Position:** primary card when audit tab active

## Contents

- Last 100 adjustments label
- audit table with product, prev, delta, new, reason, date columns
- pagination controls

---


# Buttonplacements

**Button:** Stock Levels

**Zone:** tab-switcher

**Style:** segmented chip

---

**Button:** Audit Log

**Zone:** tab-switcher

**Style:** segmented chip

---

**Button:** Refresh

**Zone:** stock-card header-right

**Style:** outline

---

**Button:** Adjust

**Zone:** stock row action cell

**Style:** outline small

---

**Button:** Apply

**Zone:** inline adjustment editor

**Style:** primary small

---

**Button:** Cancel

**Zone:** inline adjustment editor

**Style:** text small

---


# Iconplacements

**Icon:** inventory

**Zone:** stock empty state

---

**Icon:** ledger

**Zone:** audit empty state

---

**Icon:** reason chip

**Zone:** audit reason column

---


# Interactiveflows

**Flowid:** inventory-bootstrap

## Steps

- Page reads token via useToken
- Page requests /v1/supplier/inventory and /v1/supplier/inventory/audit
- Stock tab renders by default

---

**Flowid:** quantity-adjustment

## Steps

- Supplier clicks Adjust on a row
- Action cell expands into delta input, reason select, Apply, and Cancel controls
- Page patches /v1/supplier/inventory with adjustment payload
- Page refreshes both stock and audit datasets

---

**Flowid:** audit-review

## Steps

- Supplier switches to Audit Log tab
- Page displays signed delta values, reason chip, and timestamped adjustments

---


# Datadependencies

## Readendpoints

- /v1/supplier/inventory
- /v1/supplier/inventory/audit

## Writeendpoints

- /v1/supplier/inventory

**Refreshmodel:** load on mount plus manual refresh button and automatic re-fetch after successful adjustments

# Statevariants

- unauthorized supplier-required card
- stock-tab loading state
- stock empty state
- normal stock table
- row-level inline adjustment mode
- audit empty state
- audit table with positive and negative delta coloring

# Figureblueprints

- inventory page on stock tab
- stock row in inline adjust mode
- inventory page on audit tab
- audit table close-up with reason chip and signed delta

