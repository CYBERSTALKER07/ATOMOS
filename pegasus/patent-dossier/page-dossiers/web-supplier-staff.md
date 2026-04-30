**Generatedat:** 2026-04-06

**Pageid:** web-supplier-staff

**Route:** /supplier/staff

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/staff/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier warehouse-staff management page for provisioning payloader accounts, listing worker credentials, and revealing one-time login PINs.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

- back link to supplier dashboard
- headline: Warehouse Staff
- subtitle describing payloader provisioning
- Provision Worker CTA

---

**Zoneid:** kpi-row

**Position:** below header

## Contents

- Total Workers card
- Active card
- Inactive card

---

**Zoneid:** worker-ledger

**Position:** main card region

## Contents

- loading spinner or empty message
- worker table with name, phone, worker ID, provision date, and status chip
- pagination controls

---

**Zoneid:** provision-drawer

**Position:** slide-out drawer

**Visibilityrule:** visible when showAdd is true

## Contents

- name input
- phone input
- error text when form invalid or request fails
- Provision Worker and Generate PIN CTA

---

**Zoneid:** pin-reveal-modal

**Position:** center overlay modal

**Visibilityrule:** visible when createdPin exists

## Contents

- success glyph
- worker name and phone text
- dashed PIN reveal panel
- warning helper banner
- Done button

---


# Buttonplacements

**Button:** Supplier Dashboard

**Zone:** header top-left

**Style:** text link

---

**Button:** + Provision Worker

**Zone:** header top-right

**Style:** primary button

---

**Button:** Provision Worker and Generate PIN

**Zone:** provision-drawer footer

**Style:** full-width primary button

---

**Button:** Done

**Zone:** pin-reveal-modal footer

**Style:** full-width primary button

---


# Iconplacements

**Icon:** warning

**Zone:** PIN reveal warning banner

---

**Icon:** success checkmark glyph

**Zone:** PIN reveal modal hero circle

---


# Interactiveflows

**Flowid:** staff-bootstrap

## Steps

- Page fetches /v1/supplier/staff/payloader on mount
- Worker rows populate KPI counts and the paginated table

---

**Flowid:** worker-provisioning

## Steps

- Supplier opens the provision drawer
- Supplier enters worker name and phone
- Page posts to /v1/supplier/staff/payloader
- On success, drawer closes, worker table refreshes, and the one-time PIN reveal modal opens

---

**Flowid:** pin-disclosure

## Steps

- Modal displays generated login PIN exactly once
- Supplier acknowledges via Done and the PIN overlay is dismissed

---


# Datadependencies

## Readendpoints

- /v1/supplier/staff/payloader

## Writeendpoints

- /v1/supplier/staff/payloader

**Refreshmodel:** load on mount and reload after successful worker provisioning

# Statevariants

- table loading spinner state
- empty worker roster state
- worker table with pagination
- provision drawer open
- provision drawer validation error
- PIN reveal modal visible

# Figureblueprints

- full staff page with KPI row and worker table
- provision drawer with name and phone fields
- PIN reveal modal with dashed PIN panel and warning banner

