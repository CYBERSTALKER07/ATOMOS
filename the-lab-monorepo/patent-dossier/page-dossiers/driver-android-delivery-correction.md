**Generatedat:** 2026-04-06

**Pageid:** android-driver-delivery-correction

**Navroute:** correction/{orderId}/{retailerName}

**Platform:** android

**Role:** DRIVER

# Sourcefiles

- apps/driver-app-android/app/src/main/java/com/thelab/driver/ui/screens/manifest/DeliveryCorrectionScreen.kt

**Shell:** driver-android-main

**Status:** implemented

**Purpose:** Android driver reconciliation screen for editing accepted quantities, assigning rejection reasons, previewing refund impact, and submitting amended manifests.

# Layoutzones

**Zoneid:** top-app-bar

**Position:** top scaffold app bar

## Contents

### Left

- back arrow button

### Center

- Verify Cargo title
- optional retailer subtitle

### Right

- modified-count badge when modifications exist

---

**Zoneid:** loading-or-error-state

**Position:** center body when list unavailable

## Contents

- loading state with spinner and Loading manifest text
- error state with Inventory2 icon, failed headline, and message

---

**Zoneid:** manifest-list

**Position:** scrollable body when loaded

## Contents

- section label showing manifest item count
- order ID mono badge
- line item cards with product name, SKU, accepted quantity, total, and modify icon
- reason tag on modified items

---

**Zoneid:** modification-bottom-sheet

**Position:** modal bottom sheet

**Visibilityrule:** visible when editingIndex targets a line item

## Contents

- product header
- accepted quantity stepper
- auto-calculated rejected text
- rejection reason filter chips
- adjusted line total preview
- Apply Modification or No Changes button

---

**Zoneid:** sticky-footer

**Position:** bottom bar

## Contents

- original total row
- animated refund delta row
- adjusted total row
- Submit Amendment or Confirm and Complete Delivery button

---

**Zoneid:** confirm-dialog

**Position:** alert overlay

**Visibilityrule:** visible when showConfirmDialog is true

## Contents

- Warning icon
- Confirm Amendment title
- modification count text
- refund amount panel
- Confirm Amendment button
- Cancel button

---


# Buttonplacements

**Button:** Back

**Zone:** top-app-bar navigation icon

**Style:** icon button

---

**Button:** Modify item

**Zone:** line item card top-right

**Style:** small icon button

---

**Button:** Decrease accepted

**Zone:** modification-bottom-sheet stepper

**Style:** filled icon button

---

**Button:** Increase accepted

**Zone:** modification-bottom-sheet stepper

**Style:** filled icon button

---

**Button:** Rejection reason chip

**Zone:** modification-bottom-sheet

**Style:** filter chip

---

**Button:** Apply Modification

**Zone:** modification-bottom-sheet footer

**Style:** full-width primary

**Visibilityrule:** rejected quantity greater than zero

---

**Button:** No Changes

**Zone:** modification-bottom-sheet footer

**Style:** full-width primary

**Visibilityrule:** rejected quantity equals zero

---

**Button:** Submit Amendment

**Zone:** sticky-footer

**Style:** full-width error-colored button

**Visibilityrule:** state.hasModifications is true

---

**Button:** Confirm & Complete Delivery

**Zone:** sticky-footer

**Style:** full-width primary button

**Visibilityrule:** state.hasModifications is false

---

**Button:** Confirm Amendment

**Zone:** confirm-dialog confirm button

**Style:** error primary

---

**Button:** Cancel

**Zone:** confirm-dialog dismiss button

**Style:** text button

---


# Iconplacements

**Icon:** ArrowBack

**Zone:** top-app-bar navigation icon

---

**Icon:** Edit

**Zone:** line item modify control

---

**Icon:** Warning

**Zone:** modified reason tag, sticky-footer submit CTA, and confirm dialog

---

**Icon:** Remove

**Zone:** bottom-sheet decrement control

---

**Icon:** Add

**Zone:** bottom-sheet increment control

---

**Icon:** CheckCircle

**Zone:** non-modified footer CTA

---

**Icon:** Inventory2

**Zone:** error state

---

**Icon:** CircularProgressIndicator

**Zone:** loading state and submit CTA while submitting

---


# Interactiveflows

**Flowid:** modify-line-item

## Steps

- Driver taps edit icon on a line item card
- Modal bottom sheet opens
- Driver changes accepted quantity with stepper
- Rejected quantity auto-calculates
- Driver optionally selects rejection reason chips
- Driver applies modification and returns to list

---

**Flowid:** footer-summary-updates

## Steps

- Any modification updates modified-count badge
- Refund delta and adjusted total in sticky footer recompute live
- Footer CTA changes from confirm-complete to submit-amendment mode

---

**Flowid:** confirm-amendment

## Steps

- Driver taps Submit Amendment
- Alert dialog summarizes modification count and refund amount
- Driver confirms
- ViewModel submits amendment and route completes on success

---


# Statevariants

- loading manifest state
- error state
- clean manifest state
- modified manifest state with badges and reason tags
- modification bottom sheet open
- confirm amendment dialog
- submitting footer state

# Figureblueprints

- android delivery correction full screen
- modified line item card
- quantity-edit bottom sheet with reason chips
- sticky footer with refund delta
- confirm amendment dialog

