**Generatedat:** 2026-04-06

**Pageid:** ios-retailer-catalog

**Viewname:** CatalogView

**Platform:** ios

**Role:** RETAILER

# Sourcefiles

- apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/CatalogView.swift

**Shell:** retailer-ios-root

**Status:** implemented

**Purpose:** Retailer category-browse and product-search screen using a bento-grid catalog overview and product-grid search results.

# Layoutzones

**Zoneid:** search-bar

**Position:** top full-width

## Contents

- magnifying glass icon
- search text field
- clear-search button when text is non-empty

---

**Zoneid:** category-browse-region

**Position:** scroll body when search is empty

**Visibilityrule:** visible when searchText is empty

## Contents

- Categories header row with count
- mixed-size bento category cards
- remaining categories two-column grid

---

**Zoneid:** search-results-grid

**Position:** scroll body when search has results

**Visibilityrule:** visible when searchText is non-empty and filteredProducts is non-empty

## Contents

- two-column ProductCardView grid

---

**Zoneid:** no-results-state

**Position:** center body

**Visibilityrule:** visible when searchText is non-empty and filteredProducts is empty

## Contents

- search icon disk
- No Results headline
- query-specific helper text

---


# Buttonplacements

**Button:** clear search

**Zone:** search-bar trailing edge

**Style:** icon button

---

**Button:** bento category card

**Zone:** category-browse-region

**Style:** card tap target

---

**Button:** product card

**Zone:** search-results-grid

**Style:** card tap target

---


# Iconplacements

**Icon:** magnifyingglass

**Zone:** search-bar leading edge

---

**Icon:** xmark.circle.fill

**Zone:** search-bar trailing clear button

---

**Icon:** category icon glyph

**Zone:** bento cards

---

**Icon:** magnifyingglass

**Zone:** no-results-state hero icon

---


# Interactiveflows

**Flowid:** category-browse

## Steps

- Retailer lands on category overview
- Retailer taps a bento category card
- Navigation pushes CategorySuppliersView

---

**Flowid:** product-search

## Steps

- Retailer types into search bar
- Catalog switches from category bento layout to filtered product grid
- Retailer taps a product card
- Navigation pushes ProductDetailView

---


# Statevariants

- loading skeleton grid
- category bento state
- search results state
- no-results state
- failed-load alert

# Figureblueprints

- retailer iOS catalog bento grid
- search bar close-up
- product search results grid
- no-results state

