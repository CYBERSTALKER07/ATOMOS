**Generatedat:** 2026-04-06

**Pageid:** web-supplier-pricing

**Route:** /supplier/pricing

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/pricing/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier pricing-engine page for composing volume-discount rules and auditing the currently active pricing rule ledger.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

- headline: Pricing Engine
- subtitle: B2B Volume Discount Rules — Upsert and Manage

---

**Zoneid:** primary-grid

**Position:** two-column workspace with wider table column

## Contents

- left form panel for new pricing rule composition
- right rules ledger card with table or empty state

---

**Zoneid:** rule-form-panel

**Position:** left column

## Contents

- SKU selector or fallback text input
- Min Pallets numeric input
- Discount percent numeric input with helper copy
- Target Retailer Tier chip row
- Valid Until datetime input
- Tier ID input
- Lock Pricing Rule CTA

---

**Zoneid:** rules-ledger-card

**Position:** right column

## Contents

- rules count header
- loading message or pricing empty state
- rules table with SKU, pallet threshold, discount chip, retailer tier, expiry, status, and actions

---


# Buttonplacements

**Button:** Target Retailer Tier chip

**Zone:** rule-form-panel target tier row

**Style:** chip toggle

---

**Button:** Lock Pricing Rule

**Zone:** rule-form-panel footer

**Style:** full-width primary button

---

**Button:** Deactivate

**Zone:** rules-ledger actions column

**Style:** small danger-tinted button

**Visibilityrule:** rule is active

---


# Iconplacements

**Icon:** pricing

**Zone:** rules empty state

---


# Interactiveflows

**Flowid:** pricing-bootstrap

## Steps

- Page obtains supplier token
- Page fetches /v1/supplier/pricing/rules and /v1/supplier/products in parallel
- SKU selector options and rules ledger render from those responses

---

**Flowid:** rule-composition

## Steps

- Supplier chooses SKU, pallet threshold, discount percent, retailer tier, expiry, and optional tier ID
- Submit generates UUID when tier_id is blank
- Page posts the assembled rule to /v1/supplier/pricing/rules
- Success resets the form and refreshes the rules ledger

---

**Flowid:** rule-deactivation

## Steps

- Supplier presses Deactivate on an active rule row
- Page deletes /v1/supplier/pricing/rules/{tier_id}
- Row status transitions from active to inactive after refresh

---


# Datadependencies

## Readendpoints

- /v1/supplier/pricing/rules
- /v1/supplier/products

## Writeendpoints

- /v1/supplier/pricing/rules
- /v1/supplier/pricing/rules/{tier_id}

**Refreshmodel:** load on mount and reload after successful create or deactivate actions

# Statevariants

- rules loading message state
- empty rules ledger state
- form with product select populated
- form submit in locking state
- rules table with active and inactive badges
- row-level deactivation pending state

# Figureblueprints

- full pricing-engine page with form panel and rules table
- rule composition form close-up showing tier chips and lock CTA
- rules ledger row showing discount chip, expiry, and deactivate action

