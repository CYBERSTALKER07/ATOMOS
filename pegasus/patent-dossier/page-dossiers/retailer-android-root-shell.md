**Generatedat:** 2026-04-06

**Pageid:** android-retailer-root-shell

**Navroute:** RetailerNavigation

**Platform:** android

**Role:** RETAILER

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/thelab/retailer/ui/navigation/RetailerNavigation.kt

**Shell:** retailer-android-root

**Status:** implemented

**Purpose:** Authenticated retailer shell that anchors primary navigation, active-order visibility, and global payment, QR, detail, and sidebar overlays.

# Layoutzones

**Zoneid:** top-bar

**Position:** top full-width

## Contents

### Left

- avatar circle button

### Center

- The Lab title

### Right

- cart icon button with badge
- notification icon button with badge

---

**Zoneid:** content-navhost

**Position:** center full-width

## Contents

- HOME dashboard
- CATALOG
- ORDERS
- PROFILE
- SUPPLIERS
- CART
- ANALYTICS
- AUTO_ORDER
- PRODUCT_DETAIL
- CATEGORY_SUPPLIERS
- SUPPLIER_CATEGORY_CATALOG

---

**Zoneid:** bottom-stack

**Position:** bottom full-width

## Contents

- FloatingActiveOrdersBar
- LabBottomBar

## Visibilityrules

- floating bar only on HOME, ORDERS, and SUPPLIERS tabs when active order count > 0

---

**Zoneid:** global-overlays

**Position:** above shell content

## Contents

- ActiveDeliveriesSheet
- OrderDetailSheet
- QROverlay
- SidebarMenu
- DeliveryPaymentSheet

---


# Buttonplacements

**Button:** avatar button

**Zone:** top-bar-left

**Style:** circular filled button

---

**Button:** cart button

**Zone:** top-bar-right

**Style:** icon button with badge

---

**Button:** notification button

**Zone:** top-bar-right

**Style:** icon button with badge

---

**Button:** bottom nav tabs

**Zone:** bottom-stack lower row

**Style:** NavigationBarItem set of five

---

**Button:** floating active orders bar

**Zone:** bottom-stack upper row

**Style:** full-width floating card CTA

---

**Button:** sidebar menu rows

**Zone:** sidebar overlay vertical list

**Style:** full-width menu row buttons

---


# Iconplacements

**Icon:** Outlined.ShoppingCart

**Zone:** top-bar cart action

---

**Icon:** Outlined.Notifications

**Zone:** top-bar notification action

---

**Icon:** Home/Store/Inventory2/AccountCircle/Person

**Zone:** bottom nav

---

**Icon:** LocalShipping

**Zone:** floating active orders progress ring center

---

**Icon:** KeyboardArrowUp

**Zone:** floating active orders expand affordance

---

**Icon:** GridView/BarChart/Insights/AutoAwesome/Inbox/Person/Settings/ExitToApp

**Zone:** sidebar rows

---


# Interactiveflows

**Flowid:** global-order-attention

## Steps

- Retailer sees active-order summary in floating bar
- Retailer taps floating bar
- ActiveDeliveriesSheet opens
- Retailer can drill into detail or QR overlay

---

**Flowid:** websocket-payment-resolution

## Steps

- NavigationViewModel receives PAYMENT_REQUIRED event
- DeliveryPaymentSheet renders
- Retailer chooses cash or card gateway
- Card path deep-links to external payment app or cash path enters pending state
- ORDER_COMPLETED or failure events drive final phase

---

**Flowid:** sidebar-navigation

## Steps

- Retailer taps avatar
- SidebarMenu slides in from left
- Retailer chooses dashboard, procurement, insights, auto-order, or AI predictions
- Shell navigates or dismisses accordingly

---


# Statevariants

- base shell with no overlays
- floating active orders visible
- sidebar open with scrim
- active deliveries sheet open
- order detail sheet open
- QR overlay open
- payment sheet choose phase
- payment sheet processing or failed or success phase

# Figureblueprints

- full retailer shell
- top-bar control cluster
- bottom-stack with floating active orders bar
- sidebar overlay state
- payment sheet state

