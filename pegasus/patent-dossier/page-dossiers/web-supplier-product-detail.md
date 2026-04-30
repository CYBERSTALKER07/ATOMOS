**Generatedat:** 2026-04-06

**Pageid:** web-supplier-product-detail

**Route:** /supplier/products/[sku_id]

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/products/[sku_id]/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier per-SKU detail workspace for inspecting metadata, editing commercial fields, and adjusting logistics constraints and activation status.

# Layoutzones

**Zoneid:** back-nav

**Position:** top-left above header

## Contents

- Back to Products text button with arrow icon

---

**Zoneid:** header-row

**Position:** top full-width below back-nav

## Contents

- thumbnail block
- title, status pill, category label, and SKU metadata
- action cluster with activate or deactivate button and edit-mode controls

---

**Zoneid:** save-message

**Position:** below header when saveMsg exists

**Visibilityrule:** visible after save or no-op response

## Contents

- success or error message banner

---

**Zoneid:** detail-grid

**Position:** two-column main region

## Contents

- Product Details card with name, description, image URL, and base price
- Logistics and Ordering card with MOQ, step size, block settings, volumetric unit, dimensions, and created date

---


# Buttonplacements

**Button:** Back to Products

**Zone:** back-nav

**Style:** text button with leading icon

---

**Button:** Deactivate

**Zone:** header-row action cluster

**Style:** outline button

**Visibilityrule:** product is active

---

**Button:** Activate

**Zone:** header-row action cluster

**Style:** outline button

**Visibilityrule:** product is inactive

---

**Button:** Edit Product

**Zone:** header-row action cluster

**Style:** primary button

**Visibilityrule:** editing is false

---

**Button:** Cancel

**Zone:** header-row action cluster

**Style:** outline button

**Visibilityrule:** editing is true

---

**Button:** Save Changes

**Zone:** header-row action cluster

**Style:** primary button

**Visibilityrule:** editing is true

---


# Iconplacements

**Icon:** arrow_back

**Zone:** Back to Products control

---

**Icon:** image

**Zone:** thumbnail placeholder when product image is absent

---


# Interactiveflows

**Flowid:** detail-bootstrap

## Steps

- Page reads sku_id from route params and supplier token from auth context
- Page requests /v1/supplier/products/{sku_id}
- Fetched data populates the read-only detail view and edit draft state

---

**Flowid:** edit-session

## Steps

- Supplier presses Edit Product
- All editable fields switch to inputs or textarea controls
- Supplier modifies commercial or logistics values
- Page computes a diff against original product state
- PUT request to /v1/supplier/products/{sku_id} persists changed fields only

---

**Flowid:** activation-toggle

## Steps

- Supplier presses Activate or Deactivate in the header action cluster
- Page submits an is_active update to /v1/supplier/products/{sku_id}
- Detail view reloads and the status pill flips

---

**Flowid:** save-feedback

## Steps

- After save, page emits a success or error banner beneath the header
- No-change saves collapse edit mode with informational confirmation

---


# Datadependencies

## Readendpoints

- /v1/supplier/products/{sku_id}

## Writeendpoints

- /v1/supplier/products/{sku_id}

**Refreshmodel:** load on mount and reload after successful save or activation changes

# Statevariants

- page loading spinner
- error or not-found card with back button
- default read-only detail mode
- edit mode with form controls in both cards
- saving state with disabled buttons
- success message banner
- error message banner

# Figureblueprints

- full product-detail page in read-only mode
- product-detail page in edit mode with both cards active
- header close-up showing thumbnail, status pill, SKU, and action cluster
- logistics card close-up showing MOQ, step size, units-per-block, and dimensions

