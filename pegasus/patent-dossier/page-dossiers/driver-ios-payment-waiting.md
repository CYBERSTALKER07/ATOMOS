**Generatedat:** 2026-04-06

**Pageid:** ios-driver-payment-waiting

**Viewname:** PaymentWaitingView

**Platform:** ios

**Role:** DRIVER

# Sourcefiles

- apps/driverappios/driverappios/Views/PaymentWaitingView.swift

**Shell:** driver-ios-main

**Status:** implemented

**Purpose:** Driver payment-settlement holding screen that waits for a websocket settlement event before enabling delivery completion.

# Layoutzones

**Zoneid:** status-stack

**Position:** center vertical stack

## Contents

- status icon
- title
- order ID
- amount

---

**Zoneid:** waiting-copy

**Position:** below amount when payment is unsettled

**Visibilityrule:** visible when isSettled is false

## Contents

- ProgressView spinner
- Retailer is completing payment helper text

---

**Zoneid:** error-row

**Position:** above completion CTA when present

**Visibilityrule:** visible when errorMessage exists

## Contents

- destructive error text

---

**Zoneid:** completion-cta

**Position:** bottom full-width

## Contents

- Complete Delivery button disabled until settlement

---


# Buttonplacements

**Button:** Complete Delivery

**Zone:** completion-cta

**Style:** full-width primary when settled and muted disabled button when unsettled

---


# Iconplacements

**Icon:** clock.fill

**Zone:** status-stack icon when unsettled

---

**Icon:** checkmark.seal.fill

**Zone:** status-stack icon when settled

---

**Icon:** ProgressView

**Zone:** waiting-copy

---


# Interactiveflows

**Flowid:** settlement-wait-loop

## Steps

- View opens websocket to /v1/ws/driver with driver_id and bearer token
- Driver watches awaiting-payment state
- Page listens for PAYMENT_SETTLED matching the current orderId

---

**Flowid:** settlement-received

## Steps

- PAYMENT_SETTLED websocket message arrives
- isSettled flips to true
- status icon changes from clock to seal
- Complete Delivery button becomes enabled

---

**Flowid:** complete-delivery

## Steps

- Driver taps Complete Delivery after settlement
- Page calls completeOrder
- Successful completion exits through onCompleted callback

---


# Statevariants

- awaiting payment state
- settled state
- completion-in-flight state
- error state
- websocket reconnect behavior after failure

# Figureblueprints

- awaiting payment screen
- payment received screen
- disabled versus enabled completion CTA comparison

