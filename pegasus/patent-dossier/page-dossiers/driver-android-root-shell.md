**Generatedat:** 2026-04-06

**Pageid:** android-driver-main-shell

**Navroute:** main

**Platform:** android

**Role:** DRIVER

# Sourcefiles

- apps/driver-app-android/app/src/main/java/com/thelab/driver/ui/navigation/DriverNavigation.kt
- apps/driver-app-android/app/src/main/java/com/thelab/driver/ui/navigation/MainTabView.kt

**Shell:** driver-android-main

**Status:** implemented

**Purpose:** Authenticated driver execution shell that holds the core four-tab workspace and routes into scanner, offload review, payment waiting, cash collection, and correction flows.

# Layoutzones

**Zoneid:** animated-content-region

**Position:** center full-width

## Contents

- HOME content
- MAP content
- RIDES content
- PROFILE content

---

**Zoneid:** bottom-stack

**Position:** bottom full-width

## Contents

- optional activeRideBar slot
- 80dp NavigationBar

---

**Zoneid:** secondary-routes

**Position:** outside root shell but in same navigation graph

## Contents

- ScannerScreen
- OffloadReviewScreen
- PaymentWaitingScreen
- CashCollectionScreen
- DeliveryCorrectionScreen

---


# Buttonplacements

**Button:** HOME tab

**Zone:** bottom nav

**Style:** NavigationBarItem

---

**Button:** MAP tab

**Zone:** bottom nav

**Style:** NavigationBarItem

---

**Button:** RIDES tab

**Zone:** bottom nav

**Style:** NavigationBarItem

---

**Button:** PROFILE tab

**Zone:** bottom nav

**Style:** NavigationBarItem

---

**Button:** scan entry CTA

**Zone:** home content route handoff

**Style:** screen CTA routed to scanner

---

**Button:** active ride bar tap target

**Zone:** bottom stack above nav

**Style:** floating summary CTA when supplied by host content

---


# Iconplacements

**Icon:** Home filled and outlined

**Zone:** bottom nav

---

**Icon:** Map filled and outlined

**Zone:** bottom nav

---

**Icon:** ListAlt filled and outlined

**Zone:** bottom nav

---

**Icon:** Person filled and outlined

**Zone:** bottom nav

---


# Interactiveflows

**Flowid:** scanner-to-offload

## Steps

- Driver taps scan entry from Home
- Navigation pushes ScannerScreen
- Validated QR result pops scanner
- Navigation pushes OffloadReviewScreen with orderId and retailerName

---

**Flowid:** offload-to-payment-or-cash

## Steps

- Driver confirms offload
- Navigation examines paymentMethod in response
- Cash path routes to CashCollectionScreen
- Card path routes to PaymentWaitingScreen

---

**Flowid:** return-to-main-shell

## Steps

- Completion actions from payment waiting, cash collection, or correction pop back to MAIN without destroying the main workspace

---


# Statevariants

- home tab active
- map tab active
- rides tab active
- profile tab active
- active ride bar present
- scanner route open
- offload review route open
- payment waiting route open
- cash collection route open
- correction route open

# Figureblueprints

- driver main shell with four-tab navigation
- active ride bar plus bottom navigation
- scanner handoff sequence
- offload review to payment branch sequence
- cash collection route state

