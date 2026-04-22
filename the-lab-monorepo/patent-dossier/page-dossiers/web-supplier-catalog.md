**Generatedat:** 2026-04-06

**Pageid:** web-supplier-catalog

**Route:** /supplier/catalog

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/catalog/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier catalog control page combining product creation, promotion creation, category filtering, product-ledger review, status toggling, and modal-based product editing.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

- headline: Inventory Control
- subtitle: Catalog Injection, Product Ledger and Promotional Routing

---

**Zoneid:** kpi-strip

**Position:** below header

## Contents

- Total SKUs card
- Active card
- Inactive card
- Catalog Value card

---

**Zoneid:** category-filter-row

**Position:** below KPI strip when operating categories exist

## Contents

- All chip with total count
- category chips with per-category counts

---

**Zoneid:** dual-form-region

**Position:** two-column band

## Contents

- SupplierProductForm on left
- SupplierPromotionForm on right

---

**Zoneid:** product-ledger-table

**Position:** bottom full-width ledger card

## Contents

- ledger header with product count chip
- product table with image cell, category chip, price, VU, block, MOQ-step, status, truncated SKU, actions
- empty state when no products match filter

---

**Zoneid:** edit-product-modal

**Position:** centered modal overlay

**Visibilityrule:** visible when editProduct exists

## Contents

- modal header with title and close button
- optional error banner
- name input
- description textarea
- base price input
- MOQ, Step, Units-per-Block triplet
- image preview and file picker
- Cancel and Save Changes footer buttons

---


# Buttonplacements

**Button:** All

**Zone:** category-filter-row

**Style:** chip

---

**Button:** Category chip

**Zone:** category-filter-row

**Style:** chip

---

**Button:** Edit

**Zone:** product-ledger actions column

**Style:** soft accent small

---

**Button:** Deactivate

**Zone:** product-ledger actions column

**Style:** soft danger small

**Visibilityrule:** product is active

---

**Button:** Activate

**Zone:** product-ledger actions column

**Style:** soft success small

**Visibilityrule:** product is inactive

---

**Button:** Close

**Zone:** edit-product-modal header-right

**Style:** small muted

---

**Button:** Cancel

**Zone:** edit-product-modal footer-left

**Style:** pill muted

---

**Button:** Save Changes

**Zone:** edit-product-modal footer-right

**Style:** pill primary

---


# Iconplacements

**Icon:** image placeholder glyph

**Zone:** product table image cell when image_url missing

---

**Icon:** catalog

**Zone:** ledger empty state

---

**Icon:** status badge chip

**Zone:** status column

---


# Interactiveflows

**Flowid:** catalog-bootstrap

## Steps

- Page fetches /v1/supplier/products and /v1/supplier/profile in parallel
- If operating categories are present, page also maps them against /v1/catalog/platform-categories
- KPI strip and filtered ledger compute from catalog payload

---

**Flowid:** category-filtering

## Steps

- Supplier clicks All or a category chip
- Filtered ledger, active count, inactive count, and catalog value recompute client-side

---

**Flowid:** product-editing

## Steps

- Supplier clicks Edit in ledger row
- Edit modal opens prefilled with product data
- Supplier may replace image, update quantities, price, and copy
- Page optionally obtains upload ticket from /v1/supplier/products/upload-ticket and uploads image
- Page puts updated payload to /v1/supplier/products/{sku_id}
- Catalog reloads

---

**Flowid:** status-toggle

## Steps

- Supplier clicks Activate or Deactivate in row action cluster
- Page updates is_active through /v1/supplier/products/{sku_id}
- Ledger refreshes

---


# Datadependencies

## Readendpoints

- /v1/supplier/products
- /v1/supplier/profile
- /v1/catalog/platform-categories

## Writeendpoints

- /v1/supplier/products/{sku_id}
- /v1/supplier/products/upload-ticket

**Refreshmodel:** initial fetch on mount plus targeted reload after edit and status-toggle actions

# Statevariants

- page spinner loading state
- error card state
- ledger empty state with forms still visible
- ledger with image thumbnails and category chips
- row-level toggle pending state
- edit-product modal open
- edit-product modal with upload preview
- edit-product modal saving state

# Figureblueprints

- full catalog page with KPI strip, filters, two-column forms, and ledger
- product-ledger table close-up
- edit-product modal
- ledger row with image placeholder and activation toggle

