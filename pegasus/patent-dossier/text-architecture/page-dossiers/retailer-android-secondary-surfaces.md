# Technical Patent Architecture: Surfaces

Source Document: page-dossiers/retailer-android-secondary-surfaces.md
Generated At: 2026-05-07T14:16:57.468Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- - show QR token for driver scan or scan driver token depending on state

## System Architecture
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/auth/LocationPickerScreen.kt
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/dashboard/DashboardScreen.kt
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/catalog/CategorySuppliersScreen.kt
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/suppliers/SupplierCatalogScreen.kt
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/product/ProductDetailScreen.kt
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/analytics/AnalyticsScreen.kt
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/autoorder/AutoOrderScreen.kt
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/profile/ProfileScreen.kt
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/suppliers/MySuppliersScreen.kt
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/OrderDetailSheet.kt
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/QROverlay.kt
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/SidebarMenu.kt
- top app bar
- map canvas
- center pin indicator
- address label
- Confirm Location footer
- service-tile grid
- pull-to-refresh scaffold
- reorder strip
- date-range buttons
- summary cards
- top app bar with category name
- supplier row list
- empty state
- top app bar with supplier name and category
- OPEN or CLOSED badge
- grouped product list
- category headers
- hero image region

## Feature Set
1. Interactiveflows
2. Datadependencies
3. Read
4. Write
5. Minifeatures
6. Statevariants

## Algorithmic and Logical Flow
1. pan map under fixed pin
2. reverse geocode displayed address
3. confirm selected coordinates
4. default map state
5. confirm-ready state
6. tap into catalog, orders, procurement, inbox, insights
7. refresh dashboard
8. change analytics date range
9. normal dashboard
10. refreshing
11. sparse data
12. return to catalog
13. open selected supplier catalog
14. list populated
15. empty
16. return to supplier list
17. open product detail from grouped list
18. supplier open
19. supplier closed
20. empty catalog
21. switch variants
22. adjust quantity
23. toggle auto-order with history or fresh dialog
24. add selected configuration to cart

## Mathematical Formulations
- No explicit closed-form equations were detected in this source file.

## Interfaces and Data Contracts
- No explicit API contract lines detected.

## Operational Constraints and State Rules
- default map state
- confirm-ready state
- normal dashboard
- refreshing
- sparse data
- list populated
- empty
- supplier open
- supplier closed
- empty catalog
- product loaded
- placeholder image
- auto-order dialog open
- chart populated
- empty analytics
- all disabled
- mixed enabled
- enable dialog open
- normal profile
- dialog open
- grid populated
- error

## Claims-Oriented Technical Elements
1. Feature family coverage includes Interactiveflows; Datadependencies; Read; Write; Minifeatures; Statevariants.
2. Algorithmic sequence includes pan map under fixed pin | reverse geocode displayed address | confirm selected coordinates.
3. Integrity constraints include default map state; confirm-ready state; normal dashboard; refreshing.
