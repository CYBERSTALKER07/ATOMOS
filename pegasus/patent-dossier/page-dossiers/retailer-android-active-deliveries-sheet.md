**Generatedat:** 2026-04-06

**Pageid:** android-retailer-active-deliveries

**Navroute:** ActiveDeliveriesSheet

**Platform:** android

**Role:** RETAILER

# Sourcefiles

- apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/ActiveDeliveriesSheet.kt

**Shell:** retailer-android-overlay

**Status:** implemented

**Purpose:** Android retailer active-deliveries bottom sheet listing in-progress orders with detail and QR actions.

# Layoutzones

**Zoneid:** sheet-header

**Position:** top of modal bottom sheet

## Contents

- Active Deliveries title
- order count subtitle
- Done action

---

**Zoneid:** delivery-card-list

**Position:** sheet body

## Contents

- active delivery cards with progress ring, order metadata, countdown row, and action buttons

---


# Buttonplacements

**Button:** Done

**Zone:** sheet-header trailing action

**Style:** text button

---

**Button:** Details

**Zone:** delivery card action row

**Style:** pill button

---

**Button:** Show QR

**Zone:** delivery card action row

**Style:** primary pill

**Visibilityrule:** order has delivery token

---


# Iconplacements

**Icon:** progress ring

**Zone:** delivery card leading visual

---

**Icon:** QrCode2

**Zone:** Show QR action or awaiting-dispatch status pill

---

**Icon:** CountdownTimer

**Zone:** countdown row

---


# Interactiveflows

**Flowid:** delivery-sheet-review

## Steps

- Retailer opens active deliveries sheet
- Retailer reviews each active-delivery card
- Retailer opens details or QR flow from a selected card
- Retailer dismisses sheet with Done

---


# Statevariants

- active deliveries sheet open
- delivery card with QR enabled
- delivery card awaiting dispatch

# Figureblueprints

- android retailer active deliveries sheet
- delivery card with countdown and QR action

