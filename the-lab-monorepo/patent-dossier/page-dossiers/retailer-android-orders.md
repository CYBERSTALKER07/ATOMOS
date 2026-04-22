**Generatedat:** 2026-04-06

**Pageid:** android-retailer-orders

**Navroute:** ORDERS

**Platform:** android

**Role:** RETAILER

# Sourcefiles

- apps/retailer-app-android/app/src/main/java/com/thelab/retailer/ui/screens/orders/OrdersScreen.kt

**Shell:** retailer-android-root

**Status:** implemented

**Purpose:** Android retailer orders hub with tabbed pager content, pull-to-refresh, detail-sheet drilldown, and QR overlay access for live deliveries.

# Layoutzones

**Zoneid:** tab-row

**Position:** top full-width

## Contents

- Active tab with Inventory2 icon
- Ordered tab with Receipt icon
- AI Planned tab with AutoAwesome icon

---

**Zoneid:** pager-region

**Position:** main body

## Contents

- ActiveOrdersList
- OrderedList
- AiPlannedList

---

**Zoneid:** detail-sheet

**Position:** overlay sheet

**Visibilityrule:** visible when selectedOrder is non-null

## Contents

- OrderDetailSheet

---

**Zoneid:** qr-overlay

**Position:** overlay

**Visibilityrule:** visible when qrOrder is non-null

## Contents

- QROverlay

---


# Buttonplacements

**Button:** tab

**Zone:** tab-row

**Style:** tab selector

---

**Button:** Details

**Zone:** active and ordered card action rows

**Style:** pill button

---

**Button:** Show QR

**Zone:** active card action row

**Style:** primary pill

---

**Button:** Cancel

**Zone:** ordered card action row

**Style:** destructive pill

---


# Iconplacements

**Icon:** Inventory2

**Zone:** Active tab and empty state

---

**Icon:** Receipt

**Zone:** Ordered tab and empty state

---

**Icon:** AutoAwesome

**Zone:** AI Planned tab and empty state

---

**Icon:** QrCode2

**Zone:** Show QR action

---


# Interactiveflows

**Flowid:** tabbed-order-review

## Steps

- Retailer switches between tabs in TabRow
- HorizontalPager swaps the associated list view

---

**Flowid:** order-drilldown-and-qr

## Steps

- Retailer opens OrderDetailSheet from a card
- Retailer may open QROverlay for dispatch-ready orders
- Pending orders can also be cancelled from Ordered cards or sheet actions

---


# Statevariants

- active list
- ordered list
- AI planned list
- pull-to-refresh state
- detail sheet open
- QR overlay visible
- empty lists

# Figureblueprints

- android retailer orders active tab
- android retailer orders ordered tab
- AI planned forecast card
- orders QR overlay

