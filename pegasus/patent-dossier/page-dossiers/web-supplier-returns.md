**Generatedat:** 2026-04-06

**Pageid:** web-supplier-returns

**Route:** /supplier/returns

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/returns/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier dispute-resolution page for reviewing rejected or damaged line items and resolving them as write-offs or returns to stock.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

- headline: Dispute and Returns
- subtitle describing write-off versus return-to-stock resolution intent

---

**Zoneid:** summary-strip

**Position:** below header

## Contents

- Open Returns metric card
- Total Damage Value metric card

---

**Zoneid:** returns-card-header

**Position:** top of main card

## Contents

- Damaged and Rejected Items label
- Refresh button

---

**Zoneid:** returns-ledger

**Position:** main card body

## Contents

- loading message or empty state
- paginated return-item rows with retailer, quantity, value, order reference, and action cluster

---

**Zoneid:** inline-resolution-cluster

**Position:** row action area when resolvingId matches row

**Visibilityrule:** visible for the selected line item

## Contents

- resolution select
- notes input
- Resolve button
- dismiss x control

---


# Buttonplacements

**Button:** Refresh

**Zone:** returns-card-header right

**Style:** outline button

---

**Button:** Resolve

**Zone:** row action column

**Style:** outline small button

---

**Button:** Resolve

**Zone:** inline-resolution-cluster

**Style:** small primary button

**Visibilityrule:** resolution editor is open

---

**Button:** x

**Zone:** inline-resolution-cluster trailing edge

**Style:** small text dismiss control

**Visibilityrule:** resolution editor is open

---


# Iconplacements

**Icon:** returns

**Zone:** empty state

---


# Interactiveflows

**Flowid:** returns-bootstrap

## Steps

- Page reads supplier token
- Page requests /v1/supplier/returns
- Summary cards derive from returned line items

---

**Flowid:** resolution-open

## Steps

- Supplier presses Resolve on a row
- Action cell expands into resolution select, notes field, confirm control, and dismiss control

---

**Flowid:** resolution-submit

## Steps

- Supplier selects WRITE_OFF or RETURN_TO_STOCK and optionally enters notes
- Page posts to /v1/supplier/returns/resolve with line_item_id, resolution, and notes
- Successful resolution clears the inline editor and reloads the ledger

---

**Flowid:** pagination-review

## Steps

- Supplier moves through paginated rows using shared pagination controls
- Visible rows update while metric cards remain global

---


# Datadependencies

## Readendpoints

- /v1/supplier/returns

## Writeendpoints

- /v1/supplier/returns/resolve

**Refreshmodel:** load on mount, manual refresh button, and automatic reload after resolution

# Statevariants

- unauthorized supplier-required card
- returns loading state
- returns empty state
- returns ledger list state
- inline resolution editor open
- row-level resolution submit pending state

# Figureblueprints

- full returns page with summary cards and paginated ledger
- return row in inline resolution mode with dropdown and notes field
- empty-state view with refresh control

