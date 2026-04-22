**Generatedat:** 2026-04-06

**Pageid:** ios-retailer-active-deliveries

**Viewname:** ActiveDeliveriesView

**Platform:** ios

**Role:** RETAILER

# Sourcefiles

- apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ActiveDeliveriesView.swift

**Shell:** retailer-ios-overlay

**Status:** implemented

**Purpose:** Retailer active-delivery monitor showing only live orders, detail-sheet drilldown, and QR handoff from a dedicated delivery surface.

# Layoutzones

**Zoneid:** delivery-scroll-region

**Position:** main body

## Contents

- active delivery card list

---

**Zoneid:** empty-state

**Position:** center body

**Visibilityrule:** visible when orders list is empty

## Contents

- shippingbox icon disk
- No Active Orders headline
- helper copy

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

- QROverlay

---


# Buttonplacements

**Button:** Details

**Zone:** delivery card action row

**Style:** neutral pill button

---

**Button:** Show QR

**Zone:** delivery card action row

**Style:** accent pill button

**Visibilityrule:** order has delivery token

---

**Button:** Awaiting Dispatch

**Zone:** delivery card action row

**Style:** disabled status pill

**Visibilityrule:** order lacks delivery token

---


# Iconplacements

**Icon:** shippingbox or qrcode or clock

**Zone:** delivery cards and empty state

---


# Interactiveflows

**Flowid:** active-delivery-review

## Steps

- View loads retailer orders and filters them to active statuses
- Retailer taps Details for a selected order
- Retailer may alternatively tap Show QR for token-enabled orders

---


# Statevariants

- loading state
- empty state
- active deliveries list
- detail sheet open
- QR overlay open

# Figureblueprints

- retailer iOS active deliveries list
- delivery card close-up
- active deliveries QR overlay

