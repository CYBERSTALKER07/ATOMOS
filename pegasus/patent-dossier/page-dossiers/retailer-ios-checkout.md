**Generatedat:** 2026-04-06

**Pageid:** ios-retailer-checkout

**Viewname:** CheckoutView

**Platform:** ios

**Role:** RETAILER

# Sourcefiles

- apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/CheckoutView.swift

**Shell:** retailer-ios-modal

**Status:** implemented

**Purpose:** Retailer order-finalization screen with cart recap, payment-method selection, supplier-closed confirmation, offline retry fallback, and success state.

# Layoutzones

**Zoneid:** toolbar

**Position:** top navigation bar

## Contents

- Checkout title
- circular xmark dismiss button

---

**Zoneid:** checkout-scroll-stack

**Position:** main scroll body while showSuccess is false

## Contents

- Cart card with line items and quantity steppers
- Payment card with change button
- Summary card with subtotal, delivery, and total

---

**Zoneid:** submit-bar

**Position:** bottom sticky region

**Visibilityrule:** visible when showSuccess is false

## Contents

- Place Order button

---

**Zoneid:** payment-picker-sheet

**Position:** modal sheet

**Visibilityrule:** visible when showPaymentPicker is true

## Contents

- payment method list rows
- selected checkmark

---

**Zoneid:** success-state

**Position:** full-screen success replacement

**Visibilityrule:** visible when showSuccess is true

## Contents

- success icon cluster
- Order Placed headline
- supporting copy
- Done button

---


# Buttonplacements

**Button:** dismiss

**Zone:** toolbar trailing edge

**Style:** circular icon button

---

**Button:** Change

**Zone:** payment card trailing edge

**Style:** text button

---

**Button:** Place Order

**Zone:** submit-bar

**Style:** full-width primary

---

**Button:** payment method row

**Zone:** payment-picker-sheet

**Style:** list row

---

**Button:** I Understand, Place Order

**Zone:** supplier-closed confirmation dialog

**Style:** confirm action

---

**Button:** Done

**Zone:** success-state footer

**Style:** full-width primary

---


# Iconplacements

**Icon:** xmark

**Zone:** toolbar dismiss control

---

**Icon:** cart.fill

**Zone:** cart section header

---

**Icon:** creditcard.fill

**Zone:** payment section header

---

**Icon:** creditcard or wallet.pass or banknote

**Zone:** payment picker rows

---

**Icon:** checkmark.circle.fill

**Zone:** success-state hero

---


# Interactiveflows

**Flowid:** payment-method-selection

## Steps

- Retailer taps Change in payment card
- Payment picker sheet opens
- Retailer selects Click, Payme, Global Pay, or Cash on Delivery
- selectedPayment updates and sheet dismisses

---

**Flowid:** order-submission

## Steps

- Retailer taps Place Order
- If supplier is inactive, confirmation dialog interposes
- Checkout posts to /v1/checkout/unified with gateway-mapped code
- Success clears cart and shows success state
- Failure stores PendingOrder for retry and shows alert

---


# Statevariants

- review state
- payment picker sheet
- supplier closed confirmation dialog
- error alert
- submitting state
- success replacement state

# Figureblueprints

- retailer iOS checkout review state
- payment picker sheet
- supplier closed confirmation dialog
- checkout success state

