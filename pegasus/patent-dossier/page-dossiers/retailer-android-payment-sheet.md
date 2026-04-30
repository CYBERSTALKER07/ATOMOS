**Generatedat:** 2026-04-06

**Pageid:** android-retailer-payment-sheet

**Navroute:** DeliveryPaymentSheet

**Platform:** android

**Role:** RETAILER

# Sourcefiles

- apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/DeliveryPaymentSheet.kt

**Shell:** retailer-android-overlay

**Status:** implemented

**Purpose:** Android retailer payment-required bottom sheet for choosing payment path after delivery, waiting for cash confirmation or card settlement, and resolving success or failure.

# Layoutzones

**Zoneid:** choose-phase

**Position:** sheet body when phase is CHOOSE

## Contents

- payments icon disk
- amount due stack with optional struck original amount
- cash option row
- card gateway option rows

---

**Zoneid:** processing-phase

**Position:** sheet body when phase is PROCESSING

## Contents

- progress indicator
- Processing headline
- connection helper text

---

**Zoneid:** cash-pending-phase

**Position:** sheet body when phase is CASH_PENDING

## Contents

- cash icon disk
- Cash Collection Pending headline
- amount text
- waiting chip with progress indicator

---

**Zoneid:** success-phase

**Position:** sheet body when phase is SUCCESS

## Contents

- success icon disk
- Payment Complete headline
- amount text
- Done button

---

**Zoneid:** failed-phase

**Position:** sheet body when phase is FAILED

## Contents

- failure icon disk
- Payment Failed headline
- error message
- Retry button
- Cancel outlined button

---


# Buttonplacements

**Button:** Cash on Delivery

**Zone:** choose-phase

**Style:** option row

---

**Button:** card gateway option

**Zone:** choose-phase

**Style:** option row

---

**Button:** Done

**Zone:** success phase footer

**Style:** full-width primary

---

**Button:** Retry

**Zone:** failed phase footer

**Style:** full-width primary

---

**Button:** Cancel

**Zone:** failed phase footer

**Style:** full-width outlined

---


# Iconplacements

**Icon:** Payments

**Zone:** choose-phase hero

---

**Icon:** LocalAtm

**Zone:** cash option and cash-pending hero

---

**Icon:** CreditCard

**Zone:** card option rows

---

**Icon:** Check

**Zone:** success hero

---

**Icon:** Close

**Zone:** failed hero

---

**Icon:** CircularProgressIndicator

**Zone:** processing and cash-pending states

---


# Interactiveflows

**Flowid:** cash-route

## Steps

- Retailer chooses cash option
- Sheet enters cash-pending state
- Retailer waits for driver-side confirmation

---

**Flowid:** card-route

## Steps

- Retailer chooses card gateway row
- Sheet enters processing state
- External or backend-driven payment settlement updates the phase to success or failed

---

**Flowid:** failure-recovery

## Steps

- Sheet enters FAILED state
- Retailer retries or dismisses

---


# Statevariants

- choose phase
- processing phase
- cash pending phase
- success phase
- failed phase

# Figureblueprints

- android retailer payment choose phase
- android retailer payment cash pending phase
- android retailer payment success phase
- android retailer payment failed phase

