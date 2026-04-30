**Generatedat:** 2026-04-06

**Pageid:** web-supplier-products

**Route:** /supplier/products

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/products/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier product-portfolio page for SKU search, category filtering, activation toggling, and entry into per-product detail editing.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

- headline: My Products
- subtitle with registered SKU count
- Add Product CTA linking to supplier catalog

---

**Zoneid:** kpi-strip

**Position:** below header

## Contents

- Total SKUs card
- Active card
- Inactive card
- Catalog Value card

---

**Zoneid:** control-row

**Position:** below KPI strip

## Contents

- left search input with embedded search icon
- right Refresh button

---

**Zoneid:** category-chip-row

**Position:** below control row when categoryOptions exist

## Contents

- All chip with total count
- per-category chips with counts

---

**Zoneid:** product-grid

**Position:** main content grid

## Contents

- responsive product cards with image region, status badge, category pill, title, description, price block, activation icon button, and SKU footer
- empty state when no products match filters

---


# Buttonplacements

**Button:** Add Product

**Zone:** header top-right

**Style:** accent filled link button

---

**Button:** Refresh

**Zone:** control-row right

**Style:** outline button with refresh icon

---

**Button:** All

**Zone:** category-chip-row

**Style:** filter chip

---

**Button:** Category chip

**Zone:** category-chip-row

**Style:** filter chip

---

**Button:** Deactivate

**Zone:** product card bottom-right icon button

**Style:** round danger-tinted icon button

**Visibilityrule:** product is active

---

**Button:** Activate

**Zone:** product card bottom-right icon button

**Style:** round success-tinted icon button

**Visibilityrule:** product is inactive

---


# Iconplacements

**Icon:** add

**Zone:** Add Product CTA leading icon

---

**Icon:** search

**Zone:** search field left inset

---

**Icon:** refresh

**Zone:** Refresh button leading icon

---

**Icon:** image placeholder glyph

**Zone:** product card media region when image_url missing

---

**Icon:** visibility_off

**Zone:** active product toggle button

---

**Icon:** visibility

**Zone:** inactive product toggle button

---

**Icon:** catalog

**Zone:** empty state

---


# Interactiveflows

**Flowid:** products-bootstrap

## Steps

- Page reads supplier token
- Page fetches /v1/supplier/products and /v1/supplier/profile in parallel
- When operating categories exist, page maps them against /v1/catalog/platform-categories
- KPI strip and filtered grid derive from the resulting product dataset

---

**Flowid:** search-and-filter

## Steps

- Supplier types in the search field or taps a category chip
- Grid filters client-side by category, SKU, product name, and description
- KPI totals recompute from the filtered list

---

**Flowid:** product-detail-navigation

## Steps

- Supplier clicks a product card
- Router pushes to /supplier/products/{sku_id}
- Per-product detail workspace opens

---

**Flowid:** activation-toggle

## Steps

- Supplier presses the card-level activation icon button
- Page puts new is_active value to /v1/supplier/products/{sku_id}
- Products dataset reloads and card opacity/status badge update

---


# Datadependencies

## Readendpoints

- /v1/supplier/products
- /v1/supplier/profile
- /v1/catalog/platform-categories

## Writeendpoints

- /v1/supplier/products/{sku_id}

**Refreshmodel:** load on mount, manual refresh button, and automatic reload after activation changes

# Statevariants

- page loading spinner
- error card state
- grid empty state with no products yet
- grid empty state with no search matches
- filtered product grid with mixed active and inactive cards
- row-level activation toggle pending spinner

# Figureblueprints

- full products page with header, KPI strip, filters, and grid
- single product card showing status badge and activation icon button
- empty-state view with search field and refresh button retained

