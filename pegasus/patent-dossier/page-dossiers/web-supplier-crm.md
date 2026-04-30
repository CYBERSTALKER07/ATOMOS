**Generatedat:** 2026-04-06

**Pageid:** web-supplier-crm

**Route:** /supplier/crm

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/crm/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier retailer-relationship page for tracking retailer lifetime value, order history, and contact detail in a table-plus-drawer CRM workspace.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

- headline: Retailer CRM
- subtitle describing retailer relationship and lifetime-value tracking

---

**Zoneid:** kpi-grid

**Position:** below header

## Contents

- Total Retailers card
- Active card
- Total Lifetime Value card

---

**Zoneid:** retailer-ledger

**Position:** main card region

## Contents

- loading spinner or CRM empty state
- table of retailers with avatar initials, lifetime value, order count, last order date, and status chip
- pagination controls

---

**Zoneid:** retailer-detail-drawer

**Position:** slide-out side drawer

**Visibilityrule:** visible when slideOpen is true

## Contents

- retailer initials tile
- status chip
- contact links for phone and email
- lifetime value and total orders KPI cards
- order ledger list with state dot, item count, amount, and date

---


# Buttonplacements

**Button:** Retailer row

**Zone:** retailer-ledger body rows

**Style:** full-row tap target opening detail drawer

---


# Iconplacements

**Icon:** crm

**Zone:** CRM empty state

---

**Icon:** phone

**Zone:** detail drawer contact row

---

**Icon:** email

**Zone:** detail drawer contact row

---


# Interactiveflows

**Flowid:** crm-bootstrap

## Steps

- Page reads supplier token and fetches /v1/supplier/crm/retailers
- Returned retailer rows populate KPI cards and paginated ledger

---

**Flowid:** detail-drawer-open

## Steps

- Supplier clicks a retailer row
- Drawer opens immediately in loading state
- Page requests /v1/supplier/crm/retailers/{retailer_id}
- If detail fetch fails, page falls back to a synthesized order-history sample based on the selected base record

---

**Flowid:** contact-escalation

## Steps

- Supplier can use tel: and mailto: links in the drawer contact section to escalate directly to retailer communications

---


# Datadependencies

## Readendpoints

- /v1/supplier/crm/retailers
- /v1/supplier/crm/retailers/{retailer_id}

## Writeendpoints


**Refreshmodel:** single fetch for retailer list plus on-demand detail fetch when a row is opened

# Statevariants

- table loading spinner state
- CRM empty state
- retailer table with pagination
- detail drawer loading spinner
- detail drawer with contact and order ledger
- detail drawer empty order ledger

# Figureblueprints

- full CRM page with KPI grid and retailer ledger
- retailer detail drawer over table background
- drawer order-ledger close-up with state dots and amount column

