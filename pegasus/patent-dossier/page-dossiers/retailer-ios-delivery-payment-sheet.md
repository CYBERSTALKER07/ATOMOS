**Generatedat:** 2026-04-06

**Pageid:** ios-retailer-payment-sheet

**Viewname:** DeliveryPaymentSheetView

**Platform:** ios

**Role:** RETAILER

# Sourcefiles

- apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/DeliveryPaymentSheetView.swift

**Shell:** retailer-ios-overlay

**Status:** implemented

**Purpose:** Retailer payment-required overlay for choosing cash or card gateways after offload, waiting for settlement, and confirming successful completion.

# Layoutzones

**Zoneid:** phase-container

**Position:** sheet body

## Contents

- choose content
- processing content
- cash pending content
- success content
- failed content

---

**Zoneid:** choose-phase

**Position:** sheet body when phase is choose

**Visibilityrule:** visible when phase equals choose

## Contents

- warning icon disk
- amount due stack with optional struck original amount
- payment method choice list
- cash option button
- one or more card gateway option buttons

---

**Zoneid:** processing-phase

**Position:** sheet body when phase is processing

## Contents

- ProgressView
- Processing headline
- connecting helper text

---

**Zoneid:** cash-pending-phase

**Position:** sheet body when phase is cashPending

## Contents

- banknote icon disk
- Cash Collection Pending headline
- amount text
- waiting pill with progress indicator

---

**Zoneid:** success-or-failure-phase

**Position:** sheet body when phase is success or failed

## Contents

- success checkmark or failure xmark disk
- result headline
- amount or error message
- Done or Retry and Cancel actions

---


# Buttonplacements

**Button:** Close

**Zone:** navigation bar cancellation action

**Style:** text button

**Visibilityrule:** phase is choose or failed

---

**Button:** Cash on Delivery option

**Zone:** choose-phase

**Style:** full-width option row

---

**Button:** card gateway option

**Zone:** choose-phase

**Style:** full-width option row

---

**Button:** Done

**Zone:** success phase footer

**Style:** full-width primary

---

**Button:** Try Again

**Zone:** failed phase footer

**Style:** full-width primary

---

**Button:** Cancel

**Zone:** failed phase footer

**Style:** text button

---


# Iconplacements

**Icon:** banknote.fill

**Zone:** choose and cash-pending hero disks

---

**Icon:** creditcard.fill

**Zone:** card gateway option rows

---

**Icon:** checkmark.circle.fill

**Zone:** success phase hero

---

**Icon:** xmark.circle.fill

**Zone:** failed phase hero

---

**Icon:** ProgressView

**Zone:** processing and cash-pending phases

---

**Icon:** chevron.right

**Zone:** payment option rows

---


# Interactiveflows

**Flowid:** cash-selection

## Steps

- Retailer chooses Cash on Delivery
- Sheet enters processing then cashPending phase
- Sheet listens for driver confirmation or websocket completion

---

**Flowid:** card-selection

## Steps

- Retailer chooses Click, Payme, or Global Pay
- Sheet posts checkout request
- External payment URL opens when available
- Sheet remains in processing until paymentSettled or orderCompleted websocket event arrives

---

**Flowid:** failure-retry

## Steps

- Payment attempt fails
- Sheet renders failed phase
- Retailer retries or cancels

---


# Statevariants

- choose phase
- processing phase
- cash pending phase
- success phase
- failed phase

# Figureblueprints

- retailer iOS payment choose phase
- cash pending phase
- processing phase
- success and failure states

