**Generatedat:** 2026-04-06

**Pageid:** ios-retailer-root-shell

**Viewname:** ContentView

**Platform:** ios

**Role:** RETAILER

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/ContentView.swift

**Shell:** retailer-ios-root

**Status:** implemented

**Purpose:** Authenticated retailer shell for tab navigation, toolbar-driven controls, floating active-order summary, and modal or sheet-based operational flows.

# Layoutzones

**Zoneid:** tab-layer

**Position:** full-screen base layer

## Contents

- Home tab
- Catalog tab
- Orders tab
- Profile tab
- Suppliers tab

---

**Zoneid:** toolbar

**Position:** top navigation bar within each tab

## Contents

### Left

- circular avatar/menu button

### Center

- leaf icon plus The Lab wordmark

### Right

- cart button with count badge
- notification bell with count badge

---

**Zoneid:** floating-summary

**Position:** bottom above tab bar

## Contents

- FloatingActiveOrdersBar

**Visibilityrule:** visible on home, orders, and suppliers tabs only

---

**Zoneid:** sheet-and-overlay-layer

**Position:** above base layer

## Contents

- SidebarMenu
- BottomSheetOverlay containing ActiveDeliveriesView
- DeliveryPaymentSheetView
- FutureDemandView sheet
- AutoOrderView sheet
- CartView sheet
- InsightsView sheet
- ProfileView sheet

---


# Buttonplacements

**Button:** avatar/menu button

**Zone:** toolbar-left

**Style:** circular gradient button

---

**Button:** cart button

**Zone:** toolbar-right

**Style:** icon button with numeric badge

---

**Button:** notification button

**Zone:** toolbar-right

**Style:** icon button with numeric badge

---

**Button:** floating active orders bar

**Zone:** bottom floating layer

**Style:** pill-like full-width CTA

---

**Button:** sidebar rows

**Zone:** sidebar vertical stack

**Style:** plain button rows with icon tile and chevron

---

**Button:** Done toolbar actions

**Zone:** sheet top-right confirmation slot

**Style:** text confirmation button

---


# Iconplacements

**Icon:** house / square.grid.2x2 / shippingbox / person.circle / building.2

**Zone:** tab bar

---

**Icon:** leaf.fill

**Zone:** toolbar center brand mark

---

**Icon:** cart

**Zone:** toolbar cart action

---

**Icon:** bell

**Zone:** toolbar notification action

---

**Icon:** chevron.up

**Zone:** floating active orders bar expand affordance

---

**Icon:** square.grid.2x2 / chart.bar / chart.line.uptrend.xyaxis / wand.and.stars / sparkles / tray / person / gearshape / rectangle.portrait.and.arrow.right

**Zone:** sidebar rows

---


# Interactiveflows

**Flowid:** active-orders-drilldown

## Steps

- FloatingActiveOrdersBar appears when active orders exist
- Retailer taps floating bar
- BottomSheetOverlay presents ActiveDeliveriesView
- Retailer navigates to delivery detail or payment sheet based on current order state

---

**Flowid:** sidebar-mode-switching

## Steps

- Retailer taps avatar button
- SidebarMenu animates in from left
- Retailer selects dashboard, procurement, insights, auto-order, AI predictions, inbox, profile, or settings
- Shell switches tab or opens target sheet

---

**Flowid:** payment-event-presentation

## Steps

- RetailerWebSocket sets paymentEvent
- DeliveryPaymentSheetView presents as large sheet
- Retailer resolves payment
- Sheet dismiss triggers active-order reload

---


# Statevariants

- base tab shell
- floating active orders visible
- sidebar open with dimmed background
- active deliveries bottom sheet open
- payment sheet open
- future demand sheet open
- cart sheet open
- insights sheet open

# Figureblueprints

- full iOS retailer shell
- toolbar control cluster
- floating active orders bar
- sidebar overlay state
- payment sheet state

