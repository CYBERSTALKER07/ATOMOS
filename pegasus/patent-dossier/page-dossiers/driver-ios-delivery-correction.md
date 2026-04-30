**Generatedat:** 2026-04-06

**Pageid:** ios-driver-delivery-correction

**Viewname:** DeliveryCorrectionView

**Platform:** ios

**Role:** DRIVER

# Sourcefiles

- apps/driverappios/driverappios/Views/DeliveryCorrectionView.swift

**Shell:** driver-ios-main

**Status:** implemented

**Purpose:** Driver amendment screen for toggling manifest items between delivered and rejected states and calculating refund deltas before submission.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

### Left

- Back button
- Delivery Correction title
- order ID

### Right

- StatusPill showing rejected count or all-clear state

---

**Zoneid:** loading-region

**Position:** center body

**Visibilityrule:** visible when vm.isLoading is true

## Contents

- Loading line items progress indicator

---

**Zoneid:** manifest-list

**Position:** scrollable body when loaded

## Contents

- MANIFEST ITEMS section label
- line item cards with sku, quantity x unit price, status pill, line total, bottom status bar

---

**Zoneid:** summary-bar

**Position:** bottom material overlay

## Contents

- original total row
- refund delta row when refundDelta > 0
- divider
- adjusted total row
- Submit Amendment or All Items Delivered CTA

---

**Zoneid:** confirm-alert

**Position:** system alert overlay

**Visibilityrule:** visible when showConfirmAlert is true

## Contents

- Confirm Amendment title
- cancel button
- destructive submit button
- message with rejected count and refund delta

---


# Buttonplacements

**Button:** Back

**Zone:** header-left

**Style:** inline icon-text button

---

**Button:** line-item card tap target

**Zone:** manifest-list

**Style:** whole-card toggle button

---

**Button:** Submit Amendment

**Zone:** summary-bar footer

**Style:** full-width destructive

**Visibilityrule:** one or more items rejected

---

**Button:** All Items Delivered

**Zone:** summary-bar footer

**Style:** disabled muted

**Visibilityrule:** no items rejected

---

**Button:** Cancel

**Zone:** confirm-alert

**Style:** system alert cancel action

---

**Button:** Submit

**Zone:** confirm-alert

**Style:** system alert destructive action

---


# Iconplacements

**Icon:** chevron.left

**Zone:** header back button

---

**Icon:** StatusPill capsule

**Zone:** header-right

---

**Icon:** bottom status bar on each line item card

**Zone:** line item footer

---

**Icon:** ProgressView

**Zone:** loading-region

---


# Interactiveflows

**Flowid:** load-manifest-items

## Steps

- View loads line items on task start
- Loading state is replaced by tappable manifest cards

---

**Flowid:** toggle-item-status

## Steps

- Driver taps a line item card
- Item toggles between delivered and rejected status
- Status pill, strikethrough, and bottom bar update
- Summary bar recalculates refund delta and adjusted total

---

**Flowid:** submit-amendment

## Steps

- Driver rejects one or more items
- Driver taps Submit Amendment
- Confirm Amendment alert appears with refund summary
- Driver submits and page calls submitAmendment(orderId, driverId)
- Successful submission exits through onAmended callback

---


# Statevariants

- loading state
- all-clear manifest state
- mixed delivered and rejected items
- summary bar with refund delta
- confirm amendment alert

# Figureblueprints

- delivery correction full screen
- line item card with rejected state
- summary bar with refund delta
- confirm amendment alert

