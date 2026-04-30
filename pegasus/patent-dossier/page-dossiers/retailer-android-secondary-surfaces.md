**Generatedat:** 2026-04-06

**Bundleid:** retailer-android-secondary-surfaces

**Appid:** retailer-app-android

**Platform:** android

**Role:** RETAILER

**Status:** implemented

# Surfaces

**Pageid:** android-retailer-location-picker

**Navroute:** LocationPickerScreen

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/auth/LocationPickerScreen.kt

**Purpose:** Signup or profile location selector using a map-centered pin and confirm affordance.

## Layoutzones

- top app bar
- map canvas
- center pin indicator
- address label
- Confirm Location footer

## Buttonplacements

- close or back control
- Confirm Location button

## Iconplacements

- mappin or location indicator at center

## Interactiveflows

- pan map under fixed pin
- reverse geocode displayed address
- confirm selected coordinates

## Datadependencies

### Read

- map and geocoder state

### Write

- selected location result

## Minifeatures

- map pin
- address label
- confirm CTA

**Minifeaturecount:** 3

## Statevariants

- default map state
- confirm-ready state

## Figureblueprints

- android location picker with centered pin

---

**Pageid:** android-retailer-home

**Navroute:** HOME

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/dashboard/DashboardScreen.kt

**Purpose:** Retailer dashboard of service-entry tiles, reorder intelligence, and date-range driven spend snapshots.

## Layoutzones

- service-tile grid
- pull-to-refresh scaffold
- reorder strip
- date-range buttons
- summary cards

## Buttonplacements

- service tiles as tap targets
- date-range segmented buttons

## Iconplacements

- service icons inside M3 cards

## Interactiveflows

- tap into catalog, orders, procurement, inbox, insights
- refresh dashboard
- change analytics date range

## Datadependencies

### Read

- orders count
- reorder products
- AI demand forecasts

### Write


## Minifeatures

- service tiles
- reorder strip
- date-range filter
- pull refresh
- summary cards

**Minifeaturecount:** 5

## Statevariants

- normal dashboard
- refreshing
- sparse data

## Figureblueprints

- android retailer dashboard tile grid

---

**Pageid:** android-retailer-category-suppliers

**Navroute:** CATEGORY_SUPPLIERS/{categoryId}/{categoryName}

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/catalog/CategorySuppliersScreen.kt

**Purpose:** Category-scoped supplier browser for narrowing supplier selection before catalog exploration.

## Layoutzones

- top app bar with category name
- supplier row list
- empty state

## Buttonplacements

- back button
- supplier row tap target

## Iconplacements

- supplier avatar initials
- chevron row affordance

## Interactiveflows

- return to catalog
- open selected supplier catalog

## Datadependencies

### Read

- suppliers by category

### Write


## Minifeatures

- back nav
- supplier rows
- status badge
- empty state

**Minifeaturecount:** 4

## Statevariants

- list populated
- empty

## Figureblueprints

- category suppliers list

---

**Pageid:** android-retailer-supplier-catalog

**Navroute:** SUPPLIER_CATEGORY_CATALOG/{supplierId}/{supplierName}/{supplierCategory}/{supplierIsActive}

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/suppliers/SupplierCatalogScreen.kt

**Purpose:** Supplier-specific catalog grouped by product category with top-bar supplier identity and availability badge.

## Layoutzones

- top app bar with supplier name and category
- OPEN or CLOSED badge
- grouped product list
- category headers

## Buttonplacements

- back button
- product row tap target

## Iconplacements

- availability status dot
- back icon

## Interactiveflows

- return to supplier list
- open product detail from grouped list

## Datadependencies

### Read

- supplier products grouped by category

### Write


## Minifeatures

- supplier title
- category subtitle
- availability badge
- grouped list

**Minifeaturecount:** 4

## Statevariants

- supplier open
- supplier closed
- empty catalog

## Figureblueprints

- supplier catalog grouped by category

---

**Pageid:** android-retailer-product-detail

**Navroute:** PRODUCT_DETAIL/{productId}

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/product/ProductDetailScreen.kt

**Purpose:** Retailer product inspector with variant choice, quantity control, per-variant auto-order toggle, and fixed add-to-cart bar.

## Layoutzones

- hero image region
- product info section
- variant selector
- quantity stepper
- nutrition or metadata section
- fixed bottom Add to Cart bar

## Buttonplacements

- variant chips
- quantity plus and minus controls
- auto-order toggle
- Add to Cart button

## Iconplacements

- placeholder leaf icon
- toggle or snackbar icons

## Interactiveflows

- switch variants
- adjust quantity
- toggle auto-order with history or fresh dialog
- add selected configuration to cart

## Datadependencies

### Read

- product detail payload
- variant auto-order settings

### Write

- cart mutation
- auto-order settings update

## Minifeatures

- hero image
- variant chips
- quantity stepper
- auto-order toggle
- Add to Cart bar
- history or fresh dialog

**Minifeaturecount:** 6

## Statevariants

- product loaded
- placeholder image
- auto-order dialog open

## Figureblueprints

- product detail screen with bottom add-to-cart bar

---

**Pageid:** android-retailer-analytics

**Navroute:** ANALYTICS

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/analytics/AnalyticsScreen.kt

**Purpose:** Retailer expense and supplier-spend analytics dashboard with date-range filters and charts.

## Layoutzones

- date-range chip row
- KPI cards
- expense line chart
- top suppliers chart
- top products table

## Buttonplacements

- date-range filter chips

## Iconplacements

- chart glyphs inside KPI cards

## Interactiveflows

- change date range
- refresh analytics dataset

## Datadependencies

### Read

- retailer analytics endpoint

### Write


## Minifeatures

- range chips
- KPI cards
- line chart
- supplier chart
- products table

**Minifeaturecount:** 5

## Statevariants

- chart populated
- empty analytics
- refreshing

## Figureblueprints

- analytics screen with range chips and charts

---

**Pageid:** android-retailer-auto-order

**Navroute:** AUTO_ORDER

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/autoorder/AutoOrderScreen.kt

**Purpose:** Hierarchy-based auto-order settings page covering global, supplier, category, and product enablement.

## Layoutzones

- sparkles header card
- settings list
- toggle rows
- confirmation dialog

## Buttonplacements

- toggle switches for each scope
- Use History action
- Start Fresh action
- Cancel action

## Iconplacements

- sparkles icon
- scope icons in rows

## Interactiveflows

- toggle auto-order at any scope
- open dialog when enabling
- persist selection

## Datadependencies

### Read

- auto-order settings

### Write

- auto-order enable or disable endpoints

## Minifeatures

- global toggle
- supplier toggle
- category toggle
- product toggle
- enable dialog

**Minifeaturecount:** 5

## Statevariants

- all disabled
- mixed enabled
- enable dialog open

## Figureblueprints

- auto-order settings hierarchy screen

---

**Pageid:** android-retailer-profile

**Navroute:** PROFILE

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/profile/ProfileScreen.kt

**Purpose:** Retailer profile, support, company settings, and global auto-order governance surface.

## Layoutzones

- profile header card
- stats row
- auto-order card
- settings sections
- support rows

## Buttonplacements

- global auto-order switch
- settings row tap targets
- logout action

## Iconplacements

- avatar initials
- settings row icons

## Interactiveflows

- toggle global auto-order
- open history or fresh dialog
- navigate into settings items
- logout

## Datadependencies

### Read

- retailer profile endpoint

### Write

- global auto-order endpoint

## Minifeatures

- profile card
- status pill
- stats row
- global auto-order toggle
- settings rows
- logout

**Minifeaturecount:** 6

## Statevariants

- normal profile
- dialog open

## Figureblueprints

- profile screen with auto-order card

---

**Pageid:** android-retailer-suppliers

**Navroute:** SUPPLIERS

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/suppliers/MySuppliersScreen.kt

**Purpose:** Favorite suppliers grid with pull-to-refresh and retry-capable empty or error fallback.

## Layoutzones

- supplier card grid
- pull refresh scaffold
- empty or retry state

## Buttonplacements

- supplier card tap target
- Retry button in fallback state

## Iconplacements

- supplier initials tile
- empty-state icon

## Interactiveflows

- refresh supplier list
- open supplier catalog from card
- retry after failure

## Datadependencies

### Read

- favorite suppliers endpoint

### Write


## Minifeatures

- grid view
- pull refresh
- retry action
- order count badge
- auto-order badge

**Minifeaturecount:** 5

## Statevariants

- grid populated
- empty
- error

## Figureblueprints

- favorite suppliers grid

---

**Pageid:** android-retailer-order-detail-sheet

**Navroute:** OrderDetailSheet

**Surfacetype:** overlay

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/OrderDetailSheet.kt

**Purpose:** Bottom-sheet order drill-down showing status, items, amounts, and terminal actions tied to an active order.

## Layoutzones

- sheet header
- order metadata section
- line-item list
- footer action row

## Buttonplacements

- sheet close affordance
- status-specific footer actions

## Iconplacements

- status icon or ring

## Interactiveflows

- open from order card
- inspect line items
- launch payment or QR step when relevant

## Datadependencies

### Read

- selected order payload

### Write

- order action callbacks

## Minifeatures

- sheet header
- item list
- footer actions
- status indicator

**Minifeaturecount:** 4

## Statevariants

- standard order review
- actionable order state

## Figureblueprints

- order detail bottom sheet

---

**Pageid:** android-retailer-qr-overlay

**Navroute:** QROverlay

**Surfacetype:** overlay

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/QROverlay.kt

**Purpose:** Retailer QR verification overlay for delivery acceptance and handoff confirmation.

## Layoutzones

- camera or QR display frame
- instruction label
- dismiss region

## Buttonplacements

- dismiss control
- confirm or continue action if present

## Iconplacements

- QR framing corners

## Interactiveflows

- show QR token for driver scan or scan driver token depending on state
- dismiss overlay

## Datadependencies

### Read

- QR payload state

### Write

- verification callback

## Minifeatures

- QR frame
- instruction text
- dismiss control

**Minifeaturecount:** 3

## Statevariants

- display mode
- scan mode

## Figureblueprints

- QR overlay figure

---

**Pageid:** android-retailer-sidebar-menu

**Navroute:** SidebarMenu

**Surfacetype:** overlay

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/SidebarMenu.kt

**Purpose:** Drawer-style navigation overlay exposing secondary retailer destinations and profile context.

## Layoutzones

- avatar header
- menu rows
- footer utility area

## Buttonplacements

- menu row tap targets
- dismiss touch scrim

## Iconplacements

- row icons
- avatar initials

## Interactiveflows

- open from shell menu trigger
- navigate to selected destination
- dismiss drawer

## Datadependencies

### Read

- retailer identity summary

### Write

- navigation state

## Minifeatures

- avatar header
- menu rows
- icon stack
- dismiss scrim

**Minifeaturecount:** 4

## Statevariants

- open drawer

## Figureblueprints

- sidebar drawer overlay

---


