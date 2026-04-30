**Generatedat:** 2026-04-06

**Bundleid:** retailer-ios-secondary-surfaces

**Appid:** retailer-app-ios

**Platform:** ios

**Role:** RETAILER

**Status:** implemented

# Surfaces

**Pageid:** ios-retailer-home

**Viewname:** DashboardView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/DashboardView.swift

**Purpose:** Retailer service dashboard with breadbox-style tiles, quick reorder, and AI forecast highlights.

## Layoutzones

- hero service tile grid
- quick reorder section
- AI prediction cards
- refresh scaffold

## Buttonplacements

- service tiles as navigation targets

## Iconplacements

- service icons in tiles

## Interactiveflows

- tap into catalog, orders, procurement, inbox, insights, history, search, profile
- refresh dashboard data

## Datadependencies

### Read

- orders summary
- reorder products
- AI demand forecasts

### Write


## Minifeatures

- service grid
- quick reorder strip
- AI cards
- pull refresh

**Minifeaturecount:** 4

## Statevariants

- normal dashboard
- refreshing

## Figureblueprints

- retailer iOS dashboard tile grid

---

**Pageid:** ios-retailer-category-suppliers

**Viewname:** CategorySuppliersView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/CategorySuppliersView.swift

**Purpose:** Supplier list filtered by category for drill-down browsing.

## Layoutzones

- navigation header
- supplier rows
- empty-state region

## Buttonplacements

- supplier row tap target

## Iconplacements

- supplier initials tiles
- chevron affordance

## Interactiveflows

- select supplier to enter supplier products

## Datadependencies

### Read

- suppliers by category endpoint

### Write


## Minifeatures

- supplier rows
- status badge
- empty state

**Minifeaturecount:** 3

## Statevariants

- rows present
- empty

## Figureblueprints

- category supplier list

---

**Pageid:** ios-retailer-my-suppliers

**Viewname:** MySuppliersView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/MySuppliersView.swift

**Purpose:** Favorite supplier gallery with search, refresh, order counts, and auto-order badges.

## Layoutzones

- search field
- supplier card grid
- empty-state region

## Buttonplacements

- supplier card tap target

## Iconplacements

- avatar initials
- auto-order badge

## Interactiveflows

- search favorite suppliers
- refresh supplier grid
- open supplier products

## Datadependencies

### Read

- favorite suppliers

### Write


## Minifeatures

- search
- grid
- order count badge
- auto-order badge
- refresh

**Minifeaturecount:** 5

## Statevariants

- grid populated
- empty

## Figureblueprints

- favorite supplier grid with badges

---

**Pageid:** ios-retailer-supplier-products

**Viewname:** SupplierProductsView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/SupplierProductsView.swift

**Purpose:** Supplier catalog grouped by category with supplier follow-state and supplier-level auto-order control.

## Layoutzones

- supplier header card
- Add or Remove Supplier control
- supplier auto-order toggle
- category-grouped product sections

## Buttonplacements

- Add or Remove Supplier button
- supplier auto-order toggle
- product row tap target

## Iconplacements

- supplier avatar initials
- OPEN or CLOSED badge

## Interactiveflows

- favorite or unfavorite supplier
- toggle supplier auto-order
- open product detail

## Datadependencies

### Read

- supplier products endpoint

### Write

- favorite supplier endpoint
- supplier auto-order endpoint

## Minifeatures

- supplier header
- favorite button
- auto-order toggle
- status badge
- grouped products

**Minifeaturecount:** 5

## Statevariants

- supplier open
- supplier closed

## Figureblueprints

- supplier products screen with header card

---

**Pageid:** ios-retailer-product-detail

**Viewname:** ProductDetailView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ProductDetailView.swift

**Purpose:** Product detail inspector with imagery, quantity selection, variant logic, and add-to-cart flow.

## Layoutzones

- hero image area
- product info stack
- variant selector
- quantity controls
- bottom add-to-cart action area

## Buttonplacements

- variant chip buttons
- quantity plus and minus
- Add to Cart button

## Iconplacements

- placeholder image glyph
- nutrition or metadata icons

## Interactiveflows

- select variant
- adjust quantity
- add product to cart

## Datadependencies

### Read

- product detail payload

### Write

- cart state mutation

## Minifeatures

- hero image
- variant chips
- quantity stepper
- Add to Cart CTA

**Minifeaturecount:** 4

## Statevariants

- image present
- placeholder image

## Figureblueprints

- product detail screen with bottom CTA

---

**Pageid:** ios-retailer-category-products

**Viewname:** CategoryProductsView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/CategoryProductsView.swift

**Purpose:** Category-scoped products grouped by supplier with inline per-product auto-order controls.

## Layoutzones

- category header
- collapsible supplier sections
- product rows with quantity and toggle controls

## Buttonplacements

- supplier group expand or collapse
- product auto-order toggle
- product row tap target

## Iconplacements

- section chevrons
- auto-order badge

## Interactiveflows

- expand supplier group
- toggle product auto-order
- open product detail

## Datadependencies

### Read

- products by category

### Write

- product auto-order endpoint

## Minifeatures

- group headers
- expand collapse
- product toggle
- quantity adjuster

**Minifeaturecount:** 4

## Statevariants

- collapsed groups
- expanded groups

## Figureblueprints

- category products grouped by supplier

---

**Pageid:** ios-retailer-active-order

**Viewname:** ActiveOrderView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ActiveOrderView.swift

**Purpose:** Active-order monitor for in-transit orders with live-state emphasis.

## Layoutzones

- active order card list
- status emphasis band
- refresh scaffold

## Buttonplacements

- order card tap target

## Iconplacements

- live indicator dot
- status badge

## Interactiveflows

- refresh active orders
- open selected order detail

## Datadependencies

### Read

- orders filtered by IN_TRANSIT

### Write


## Minifeatures

- live dot
- order cards
- status pill
- refresh

**Minifeaturecount:** 4

## Statevariants

- active orders present
- no active orders

## Figureblueprints

- active order list

---

**Pageid:** ios-retailer-arrival

**Viewname:** ArrivalView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ArrivalView.swift

**Purpose:** Arrival-state order list emphasizing imminent handoff and manual arrival confirmation.

## Layoutzones

- arrival order cards
- ETA countdown label
- acknowledge action row

## Buttonplacements

- manual arrival confirm button
- order card tap target

## Iconplacements

- arrival arrow icon
- status indicator

## Interactiveflows

- acknowledge arrival
- open selected order

## Datadependencies

### Read

- orders filtered by ARRIVED

### Write

- confirm arrival endpoint

## Minifeatures

- ETA countdown
- confirm arrival action
- arrival icon
- order cards

**Minifeaturecount:** 4

## Statevariants

- arrival cards present
- no arrivals

## Figureblueprints

- arrival card list with confirm action

---

**Pageid:** ios-retailer-future-demand

**Viewname:** FutureDemandView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/FutureDemandView.swift

**Purpose:** AI demand forecast gallery with confidence indicators and procurement handoff.

## Layoutzones

- forecast header card
- stats strip
- forecast card list
- close control

## Buttonplacements

- close button
- drill-down to procurement control

## Iconplacements

- sparkles icon
- confidence ring graphics

## Interactiveflows

- review predicted quantities and confidence
- move into procurement workflow

## Datadependencies

### Read

- AI demand forecasts

### Write


## Minifeatures

- sparkles header
- confidence rings
- stats strip
- forecast cards

**Minifeaturecount:** 4

## Statevariants

- forecasts present
- empty forecasts

## Figureblueprints

- future demand modal with confidence rings

---

**Pageid:** ios-retailer-auto-order

**Viewname:** AutoOrderView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/AutoOrderView.swift

**Purpose:** Hierarchical auto-order governance for supplier, category, and product scopes.

## Layoutzones

- header
- suggestion cards
- checkbox or toggle rows
- bulk action bar

## Buttonplacements

- row toggles
- Select All
- Deselect All
- submit action

## Iconplacements

- scope icons and checkmarks

## Interactiveflows

- select targets for auto-order
- choose history or fresh behavior
- submit updated settings

## Datadependencies

### Read

- auto-order settings

### Write

- auto-order action endpoints

## Minifeatures

- global or scoped toggles
- bulk select
- submit
- suggestion cards

**Minifeaturecount:** 4

## Statevariants

- mixed toggles
- all off
- selection pending

## Figureblueprints

- auto-order hierarchy screen

---

**Pageid:** ios-retailer-insights

**Viewname:** InsightsView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/InsightsView.swift

**Purpose:** Expense analytics dashboard with selectable windows, KPI cards, charts, and top supplier or product breakdowns.

## Layoutzones

- date-range filter row
- KPI cards
- chart region
- supplier and product expense tables

## Buttonplacements

- range buttons

## Iconplacements

- chart accents and analytics icons

## Interactiveflows

- change time horizon
- refresh insights dataset

## Datadependencies

### Read

- retailer analytics endpoint

### Write


## Minifeatures

- range filters
- KPI cards
- expense chart
- supplier table
- product table

**Minifeaturecount:** 5

## Statevariants

- analytics loaded
- empty analytics

## Figureblueprints

- insights dashboard with chart

---

**Pageid:** ios-retailer-profile

**Viewname:** ProfileView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ProfileView.swift

**Purpose:** Retailer account hub combining profile identity, settings links, support, and global empathy-engine auto-order toggles.

## Layoutzones

- gradient header card
- stats row
- order history link
- empathy engine toggle card
- company and support sections

## Buttonplacements

- global auto-order toggle
- settings row links
- logout action

## Iconplacements

- avatar initials
- settings row icons

## Interactiveflows

- toggle global auto-order with history or fresh branching
- open settings links
- logout

## Datadependencies

### Read

- retailer profile

### Write

- global auto-order endpoint

## Minifeatures

- gradient profile header
- stats row
- history link
- global auto-order
- settings sections
- logout

**Minifeaturecount:** 6

## Statevariants

- profile loaded
- toggle dialog open

## Figureblueprints

- profile page with gradient header

---

**Pageid:** ios-retailer-procurement

**Viewname:** ProcurementView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ProcurementView.swift

**Purpose:** AI-assisted procurement composer where predicted line items can be accepted, edited, and submitted as an order basket.

## Layoutzones

- header card
- suggested items list
- quantity spinners
- bulk action bar

## Buttonplacements

- item checkbox
- quantity controls
- Select All
- Deselect All
- Submit Order

## Iconplacements

- confidence or suggestion icons

## Interactiveflows

- toggle predicted items
- manually adjust quantities
- submit procurement order

## Datadependencies

### Read

- AI demand forecasts

### Write

- procurement order endpoint

## Minifeatures

- suggestion list
- checkboxes
- quantity spinners
- bulk actions
- Submit Order

**Minifeaturecount:** 5

## Statevariants

- suggestions present
- none selected
- submit pending

## Figureblueprints

- procurement suggestion screen

---

**Pageid:** ios-retailer-inbox

**Viewname:** InboxView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/InboxView.swift

**Purpose:** Operational inbox of arriving, loaded, and in-transit orders for attention routing.

## Layoutzones

- order feed list
- status pills
- ETA countdown or timing labels

## Buttonplacements

- order card tap target

## Iconplacements

- supplier badges
- status pills

## Interactiveflows

- refresh inbox feed
- open order from feed

## Datadependencies

### Read

- orders filtered for transit statuses

### Write


## Minifeatures

- status filter logic
- feed cards
- ETA labels
- supplier badge

**Minifeaturecount:** 4

## Statevariants

- feed populated
- empty feed

## Figureblueprints

- inbox feed with status pills

---

**Pageid:** ios-retailer-history

**Viewname:** HistoryView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/HistoryView.swift

**Purpose:** Historical order browser with status chip filters and drill-down entry.

## Layoutzones

- status chip scroller
- order history card list

## Buttonplacements

- status chips
- order card tap target

## Iconplacements

- status chip accents

## Interactiveflows

- filter by status
- refresh history
- open specific order

## Datadependencies

### Read

- orders by status

### Write


## Minifeatures

- status chips
- history list
- refresh

**Minifeaturecount:** 3

## Statevariants

- all orders
- filtered subset
- empty filter result

## Figureblueprints

- history screen with status chips

---

**Pageid:** ios-retailer-search

**Viewname:** SearchView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/SearchView.swift

**Purpose:** Global product search across suppliers and categories.

## Layoutzones

- search bar
- clear affordance
- empty search prompt
- result card grid

## Buttonplacements

- clear search button
- product card tap target

## Iconplacements

- magnifying glass
- clear xmark

## Interactiveflows

- type search query
- clear query
- open selected product detail

## Datadependencies

### Read

- products search endpoint

### Write


## Minifeatures

- search bar
- clear button
- result grid
- empty prompt

**Minifeaturecount:** 4

## Statevariants

- empty query
- results shown
- no results

## Figureblueprints

- global product search screen

---

**Pageid:** ios-retailer-location-picker

**Viewname:** LocationPickerView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/LocationPickerView.swift

**Purpose:** MapKit location picker for retailer signup and location updates.

## Layoutzones

- map view
- center pin
- address label
- close control
- Confirm Location button

## Buttonplacements

- close xmark button
- Confirm Location button

## Iconplacements

- mappin circle and arrowtriangle center glyph

## Interactiveflows

- move map under fixed pin
- resolve address
- confirm chosen location

## Datadependencies

### Read

- MapKit reverse geocoder state

### Write

- selected location callback

## Minifeatures

- map pin
- address label
- close button
- confirm CTA

**Minifeaturecount:** 4

## Statevariants

- default location
- adjusted location

## Figureblueprints

- iOS location picker with centered pin

---


