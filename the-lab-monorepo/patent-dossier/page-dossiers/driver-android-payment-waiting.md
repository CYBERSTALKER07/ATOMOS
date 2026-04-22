**Generatedat:** 2026-04-06

**Pageid:** android-driver-payment-waiting

**Navroute:** payment_waiting/{orderId}/{amountUZS}

**Platform:** android

**Role:** DRIVER

# Sourcefiles

- apps/driver-app-android/app/src/main/java/com/thelab/driver/ui/screens/offload/PaymentWaitingScreen.kt

**Shell:** driver-android-main

**Status:** implemented

**Purpose:** Android driver settlement screen that waits for electronic payment completion before enabling delivery finalization.

# Layoutzones

**Zoneid:** status-stack

**Position:** center vertical stack

## Contents

- hourglass or check-circle icon
- AWAITING PAYMENT or PAYMENT RECEIVED heading
- amount text
- credit-card icon
- Payme label

---

**Zoneid:** waiting-copy

**Position:** below payment method label

**Visibilityrule:** visible when state.paymentSettled is false

## Contents

- Waiting for retailer to complete payment text

---

**Zoneid:** error-row

**Position:** above completion CTA

**Visibilityrule:** visible when state.error is non-null

## Contents

- centered error text

---

**Zoneid:** completion-cta

**Position:** bottom of central stack

## Contents

- Complete Delivery button with disabled state until settlement

---


# Buttonplacements

**Button:** Complete Delivery

**Zone:** completion-cta

**Style:** full-width primary, disabled until paymentSettled

---


# Iconplacements

**Icon:** HourglassTop

**Zone:** status-stack when awaiting

---

**Icon:** CheckCircle

**Zone:** status-stack when settled

---

**Icon:** CreditCard

**Zone:** payment method indicator

---

**Icon:** CircularProgressIndicator

**Zone:** completion CTA while isCompleting

---


# Interactiveflows

**Flowid:** waiting-to-settled

## Steps

- Driver remains on waiting screen after offload confirmation
- ViewModel observes payment settlement state
- Heading, icon, and CTA state update once paymentSettled becomes true

---

**Flowid:** complete-after-settlement

## Steps

- Driver taps Complete Delivery once enabled
- ViewModel completes order
- Route exits through onComplete when state.completed becomes true

---


# Statevariants

- awaiting-payment state
- payment-received state
- CTA completing state
- error state

# Figureblueprints

- android awaiting payment screen
- android payment received screen
- enabled and disabled completion CTA comparison

