**Generatedat:** 2026-04-06

**Pageid:** ios-retailer-orders

**Viewname:** OrdersView

**Platform:** ios

**Role:** RETAILER

# Sourcefiles

- apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/OrdersView.swift

**Shell:** retailer-ios-root

**Status:** implemented

**Purpose:** Retailer order-tracking hub with active, pending, and AI-planned tabs, detail-sheet drilldown, and QR overlay access for dispatched orders.

# Layoutzones

**Zoneid:** top-tabs

**Position:** top full-width

## Contents

- Active tab with count badge
- Pending tab with count badge
- AI Planned tab with count badge

---

**Zoneid:** tab-content-pager

**Position:** main body

## Contents

- active order card list
- pending order card list
- AI planned forecast list

---

**Zoneid:** detail-sheet

**Position:** bottom sheet

**Visibilityrule:** visible when selectedOrder is non-null

## Contents

- OrderDetailSheet at 75 percent height

---

**Zoneid:** qr-overlay

**Position:** full-screen overlay

**Visibilityrule:** visible when qrOverlayOrder is non-null and status has delivery token

## Contents

- QROverlay over current tab content

---


# Buttonplacements

**Button:** tab selector

**Zone:** top-tabs

**Style:** tab button

---

**Button:** Details

**Zone:** active and pending card action row

**Style:** pill button

---

**Button:** Show QR

**Zone:** active order card action row

**Style:** accent pill button

**Visibilityrule:** order has delivery token

---

**Button:** Pre-Order

**Zone:** AI planned card trailing action

**Style:** accent pill button

---

**Button:** View

**Zone:** pending order card trailing action

**Style:** neutral pill button

---


# Iconplacements

**Icon:** bolt.fill

**Zone:** Active tab

---

**Icon:** clock.fill

**Zone:** Pending tab

---

**Icon:** sparkles

**Zone:** AI Planned tab

---

**Icon:** shippingbox.fill or clock.fill or qrcode

**Zone:** order cards and action pills

---


# Interactiveflows

**Flowid:** tabbed-order-navigation

## Steps

- Retailer switches between Active, Pending, and AI Planned tabs
- TabView page content swaps without index dots

---

**Flowid:** order-drilldown

## Steps

- Retailer taps Details or View on an order card
- OrderDetailSheet opens with logistics, line items, totals, and QR content when available

---

**Flowid:** qr-surface

## Steps

- Retailer taps Show QR on an active order
- QROverlay appears over the orders interface

---


# Statevariants

- active tab populated
- pending tab populated
- AI planned tab populated
- empty tab states
- loading state
- detail sheet open
- QR overlay visible

# Figureblueprints

- retailer iOS orders active tab
- retailer iOS orders pending tab
- AI planned forecast card
- orders QR overlay

