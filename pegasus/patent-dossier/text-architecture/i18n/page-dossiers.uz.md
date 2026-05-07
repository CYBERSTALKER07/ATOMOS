# Technical Patent Architecture: Notes

Source Document: i18n/page-dossiers.uz.md
Generated At: 2026-05-07T14:16:57.460Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- - This file is a localized overlay for detailed page dossiers.
- - Asosiy inglizcha JSON fayllar kanonik dalil manbai bo'lib qoladi; lokalizatsiya qilingan gaplar aniq UI label va texnik identifikatorlarni source anchor sifatida saqlaydi.
- - Route, endpoint, file path, page ID va icon nomlari texnik anchor sifatida o'zgartirilmaydi.

## System Architecture
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/offload/CashCollectionScreen.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/manifest/DeliveryCorrectionScreen.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/offload/OffloadReviewScreen.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/offload/PaymentWaitingScreen.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/navigation/DriverNavigation.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/navigation/MainTabView.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/scanner/ScannerScreen.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/auth/LoginScreen.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/home/HomeScreen.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/map/MapScreen.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/manifest/ManifestScreen.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/profile/ProfileScreen.kt
- Implementation Anchor: apps/driverappios/driverappios/Views/CashCollectionView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/DeliveryCorrectionView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/OffloadReviewView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/PaymentWaitingView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/QRScannerView.swift
- Implementation Anchor: apps/driverappios/driverappios/PegasusDriverApp.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/MainTabView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/Components/ActiveRideBar.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/LoginView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/HomeView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/FleetMapView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/RidesListView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/ProfileView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/OfflineVerifierView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/MissionDetailSheet.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/Components/MapMarkerDetailSheet.swift
- Implementation Anchor: apps/payload-terminal/App.ts
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/ActiveDeliveriesSheet.kt

## Feature Set
1. Sourcefiles
2. Localized
3. Layoutoverview
4. Controloverview
5. Iconoverview
6. Flowoverview
7. Steps
8. Stateoverview
9. Figureoverview
10. Surfaces
11. Dependencyoverview
12. Reads
13. Writes
14. Localizednotes
15. Minifeatureoverview

## Algorithmic and Logical Flow
1. Holat: "cash collection idle state".
2. Holat: "back-navigation confirmation dialog".
3. Holat: "completion-in-flight state".
4. Holat: "error state".
5. Holat: "loading manifest state".
6. Holat: "clean manifest state".
7. Holat: "modified manifest state with badges and reason tags".
8. Holat: "modification bottom sheet open".
9. Holat: "confirm amendment dialog".
10. Holat: "submitting footer state".
11. Holat: "clean offload state".
12. Holat: "partially rejected line-item state".
13. Holat: "fully rejected line-item state".
14. Holat: "submitting state".
15. Holat: "awaiting-payment state".
16. Holat: "payment-received state".
17. Holat: "CTA completing state".
18. Holat: "home tab active".
19. Holat: "map tab active".
20. Holat: "rides tab active".
21. Holat: "profile tab active".

## Mathematical Formulations
- **State:** allSealed == true
- **State:** token == null
- **State:** token != null && activeTruck != null && allSealed == false
- **State:** token != null && activeTruck == null

## Interfaces and Data Contracts
- Endpoint: /v1/auth/admin/login
- Endpoint: /v1/auth/payloader/login
- Endpoint: /v1/catalog/platform-categories
- Endpoint: /v1/checkout/unified
- Endpoint: /v1/fleet/active
- Endpoint: /v1/fleet/capacity
- Endpoint: /v1/fleet/reassign
- Endpoint: /v1/inventory/reconcile-returns
- Endpoint: /v1/payload/seal
- Endpoint: /v1/payloader/orders
- Endpoint: /v1/payloader/trucks
- Endpoint: /v1/supplier/analytics/demand/history
- Endpoint: /v1/supplier/analytics/demand/today
- Endpoint: /v1/supplier/analytics/velocity
- Endpoint: /v1/supplier/crm/retailers
- Endpoint: /v1/supplier/crm/retailers/{retailer_id}
- Endpoint: /v1/supplier/fleet/capacity
- Endpoint: /v1/supplier/fleet/drivers
- Endpoint: /v1/supplier/fleet/drivers/{driverId}/assign-vehicle
- Endpoint: /v1/supplier/fleet/drivers/{id}
- Endpoint: /v1/supplier/fleet/vehicles
- Endpoint: /v1/supplier/fleet/vehicles/{vehicleId}
- Endpoint: /v1/supplier/inventory
- Endpoint: /v1/supplier/inventory/audit
- Endpoint: /v1/supplier/orders
- Endpoint: /v1/supplier/orders/vet
- Endpoint: /v1/supplier/payment-config
- Endpoint: /v1/supplier/pricing/rules
- Endpoint: /v1/supplier/pricing/rules/{tier_id}
- Endpoint: /v1/supplier/products
- Endpoint: /v1/supplier/products/upload-ticket
- Endpoint: /v1/supplier/products/{sku_id}
- Endpoint: /v1/supplier/profile
- Endpoint: /v1/supplier/quarantine-stock
- Endpoint: /v1/supplier/returns
- Endpoint: /v1/supplier/returns/resolve

## Operational Constraints and State Rules
- Holat: "partially rejected line-item state".
- Holat: "fully rejected line-item state".

## Claims-Oriented Technical Elements
1. Feature family coverage includes Sourcefiles; Localized; Layoutoverview; Controloverview; Iconoverview; Flowoverview.
2. Algorithmic sequence includes Holat: "cash collection idle state". | Holat: "back-navigation confirmation dialog". | Holat: "completion-in-flight state"..
3. Contract surface is exposed through /v1/auth/admin/login, /v1/auth/payloader/login, /v1/catalog/platform-categories, /v1/checkout/unified, /v1/fleet/active, /v1/fleet/capacity.
4. Mathematical or scoring expressions are explicitly used for optimization or estimation.
5. Integrity constraints include Holat: "partially rejected line-item state".; Holat: "fully rejected line-item state"..
