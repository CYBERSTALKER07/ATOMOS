**Generatedat:** 2026-04-06

**Pageid:** android-driver-offload-review

**Navroute:** offload_review/{orderId}/{retailerName}

**Platform:** android

**Role:** DRIVER

# Sourcefiles

- apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/offload/OffloadReviewScreen.kt

**Shell:** driver-android-main

**Status:** implemented

**Purpose:** Android driver cargo-review screen for checking accepted totals, excluding damaged units, and confirming offload before payment or cash collection routing.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

### Left

- back icon button
- OFFLOAD REVIEW monospace label
- retailer name

---

**Zoneid:** totals-bar

**Position:** below header

## Contents

- original total cluster
- adjusted total cluster with dynamic color

---

**Zoneid:** line-item-list

**Position:** scrollable body

## Contents

- status icon per line item
- product name
- quantity and unit price line
- accepted total
- rejected quantity stepper

---

**Zoneid:** error-row

**Position:** above footer when present

**Visibilityrule:** visible when state.error is non-null

## Contents

- red error text

---

**Zoneid:** footer-cta

**Position:** bottom full-width container

## Contents

- Confirm Offload or Amend and Confirm Offload button with spinner state

---


# Buttonplacements

**Button:** Back

**Zone:** header-left

**Style:** icon button

---

**Button:** Reduce rejected

**Zone:** line-item stepper

**Style:** icon button

---

**Button:** Increase rejected

**Zone:** line-item stepper

**Style:** icon button

---

**Button:** Confirm Offload

**Zone:** footer-cta

**Style:** full-width primary

---

**Button:** Amend & Confirm Offload

**Zone:** footer-cta

**Style:** full-width primary

**Visibilityrule:** state.hasExclusions is true

---


# Iconplacements

**Icon:** ArrowBack

**Zone:** header-left back control

---

**Icon:** CheckCircle

**Zone:** line-item row when no exclusions

---

**Icon:** RemoveCircleOutline

**Zone:** line-item row when fully rejected

---

**Icon:** RemoveCircle

**Zone:** stepper decrement

---

**Icon:** AddCircle

**Zone:** stepper increment

---

**Icon:** CircularProgressIndicator

**Zone:** footer CTA while submitting

---


# Interactiveflows

**Flowid:** line-item-exclusion-adjustment

## Steps

- Driver uses stepper controls to increase or decrease rejected quantity per line item
- Accepted total and status coloring recompute per row
- Adjusted total in totals bar updates

---

**Flowid:** confirm-offload-clean

## Steps

- Driver leaves all rows fully accepted
- Driver taps Confirm Offload
- OffloadReviewViewModel confirms offload and returns result
- Route branches to payment or cash flow based on response

---

**Flowid:** confirm-offload-amended

## Steps

- Driver excludes one or more units
- Footer label changes to Amend and Confirm Offload
- Submission persists amended quantities before moving to downstream payment handling

---


# Statevariants

- clean offload state
- partially rejected line-item state
- fully rejected line-item state
- submitting state
- error state

# Figureblueprints

- android offload review full screen
- line-item stepper detail
- totals bar before and after exclusions
- submitting offload CTA

