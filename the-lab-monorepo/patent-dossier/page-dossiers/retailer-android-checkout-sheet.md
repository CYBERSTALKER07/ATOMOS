**Generatedat:** 2026-04-06

**Pageid:** android-retailer-checkout-sheet

**Navroute:** CheckoutSheet

**Platform:** android

**Role:** RETAILER

# Sourcefiles

- apps/retailer-app-android/app/src/main/java/com/thelab/retailer/ui/components/CheckoutSheet.kt

**Shell:** retailer-android-overlay

**Status:** implemented

**Purpose:** Android retailer checkout bottom sheet for reviewing order totals, selecting payment gateway from a split buy control, and showing processing or completion phases.

# Layoutzones

**Zoneid:** sheet-header

**Position:** top of modal bottom sheet

## Contents

- Order details title
- linear progress bar in review phase

---

**Zoneid:** review-phase

**Position:** sheet body when phase is REVIEW

## Contents

- product recap card with placeholder image
- subtotal, shipping, discount, and total rows
- Payment Method label
- split Buy button with payment dropdown segment

---

**Zoneid:** processing-phase

**Position:** sheet body when phase is PROCESSING

## Contents

- CircularProgressIndicator
- Processing payment text

---

**Zoneid:** complete-phase

**Position:** sheet body when phase is COMPLETE

## Contents

- check icon
- Payment complete text

---


# Buttonplacements

**Button:** Buy

**Zone:** review-phase bottom control row

**Style:** left segment of split CTA

---

**Button:** payment dropdown segment

**Zone:** review-phase bottom control row

**Style:** right segment of split CTA

---

**Button:** payment option

**Zone:** dropdown menu

**Style:** menu row

---


# Iconplacements

**Icon:** Eco

**Zone:** review-phase placeholder image

---

**Icon:** Payment

**Zone:** Buy segment

---

**Icon:** KeyboardArrowDown

**Zone:** dropdown segment

---

**Icon:** Check

**Zone:** complete phase and selected dropdown option

---

**Icon:** CircularProgressIndicator

**Zone:** processing phase

---


# Interactiveflows

**Flowid:** gateway-selection

## Steps

- Retailer taps dropdown segment
- DropdownMenu opens with payment options
- Retailer selects gateway and label updates

---

**Flowid:** buy-processing-complete

## Steps

- Retailer taps Buy
- Sheet phase transitions from REVIEW to PROCESSING
- On success the sheet renders COMPLETE state

---


# Statevariants

- review phase
- dropdown open state
- processing phase
- complete phase

# Figureblueprints

- android retailer checkout review sheet
- split buy and payment dropdown control
- processing sheet
- complete sheet

