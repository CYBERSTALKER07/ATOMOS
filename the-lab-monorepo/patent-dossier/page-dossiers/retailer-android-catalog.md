**Generatedat:** 2026-04-06

**Pageid:** android-retailer-catalog

**Navroute:** CATALOG

**Platform:** android

**Role:** RETAILER

# Sourcefiles

- apps/retailer-app-android/app/src/main/java/com/thelab/retailer/ui/screens/catalog/CatalogScreen.kt

**Shell:** retailer-android-root

**Status:** implemented

**Purpose:** Android retailer catalog surface combining search-driven product discovery with a mixed-scale bento category browser.

# Layoutzones

**Zoneid:** search-field

**Position:** top full-width

## Contents

- outlined pill search field
- Search icon

---

**Zoneid:** search-results-grid

**Position:** main body when query length is at least two and results exist

**Visibilityrule:** visible when searchQuery length >= 2 and filteredProducts not empty

## Contents

- two-column ProductCard grid

---

**Zoneid:** category-bento-list

**Position:** main body when search branch inactive

## Contents

- Categories header with count
- rows of large, wide, compact, and remainder category cards

---


# Buttonplacements

**Button:** category card

**Zone:** category-bento-list

**Style:** surface tap target

---

**Button:** product card

**Zone:** search-results-grid

**Style:** card tap target

---


# Iconplacements

**Icon:** Search

**Zone:** search-field leading edge

---

**Icon:** Inventory2 or category glyph

**Zone:** category cards

---


# Interactiveflows

**Flowid:** category-navigation

## Steps

- Retailer taps a category bento card
- Catalog routes to category-specific supplier or product inventory

---

**Flowid:** search-navigation

## Steps

- Retailer types at least two characters
- Catalog switches to filtered product grid
- Retailer taps a product card
- Catalog routes to product detail

---


# Statevariants

- category bento state
- search results state
- empty search branch fallback to categories

# Figureblueprints

- android retailer category bento layout
- android retailer search results grid

