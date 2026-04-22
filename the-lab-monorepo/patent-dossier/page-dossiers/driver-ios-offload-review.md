**Generatedat:** 2026-04-06

**Pageid:** ios-driver-offload-review

**Viewname:** OffloadReviewView

**Platform:** ios

**Role:** DRIVER

# Sourcefiles

- apps/driverappios/driverappios/Views/OffloadReviewView.swift

**Shell:** driver-ios-main

**Status:** implemented

**Purpose:** Driver delivery-review screen for confirming offload, partially rejecting damaged units, and branching into payment collection flows.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

### Left

- OFFLOAD REVIEW label
- order ID in monospace

### Right

- circular close button

---

**Zoneid:** retailer-total-row

**Position:** below header

## Contents

- retailer name
- total amount

---

**Zoneid:** line-item-list

**Position:** scrollable middle region

## Contents

- product name
- quantity x unit price line
- minus button
- rejected quantity counter
- plus button

---

**Zoneid:** error-row

**Position:** above footer CTA when present

**Visibilityrule:** visible when errorMessage is non-null

## Contents

- inline destructive error text

---

**Zoneid:** footer-cta

**Position:** bottom full-width

## Contents

- Confirm Offload button with optional spinner

---


# Buttonplacements

**Button:** Close

**Zone:** header-right circular button

**Style:** icon button

---

**Button:** minus reject quantity

**Zone:** each line-item stepper

**Style:** icon stepper control

---

**Button:** plus reject quantity

**Zone:** each line-item stepper

**Style:** icon stepper control

---

**Button:** Confirm Offload

**Zone:** footer-cta

**Style:** full-width primary

---


# Iconplacements

**Icon:** xmark

**Zone:** header-right close control

---

**Icon:** minus.circle.fill

**Zone:** line-item stepper decrement

---

**Icon:** plus.circle.fill

**Zone:** line-item stepper increment

---

**Icon:** ProgressView

**Zone:** Confirm Offload button when submitting

---


# Interactiveflows

**Flowid:** quantity-rejection-adjustment

## Steps

- Driver reviews line items
- Driver uses plus or minus buttons to set rejected quantity per item
- Item styling changes to delivered, partial, or fully rejected visual state

---

**Flowid:** confirm-offload-no-rejections

## Steps

- Driver leaves all rejected quantities at zero
- Driver taps Confirm Offload
- Page calls confirmOffload on fleet service
- Successful response exits through onConfirm callback

---

**Flowid:** confirm-offload-with-amendment

## Steps

- Driver marks one or more rejected quantities
- Page first calls amendOrder with derived status per line item
- Page then calls confirmOffload
- Workflow branches downstream based on returned payment mode

---


# Statevariants

- all items delivered state
- partial rejection state
- full rejection for a line-item
- submitting CTA state
- inline error state

# Figureblueprints

- offload review full screen
- line-item row with stepper controls
- mixed accepted and rejected quantities
- confirm-offload submitting state

