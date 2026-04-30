**Generatedat:** 2026-04-06

**Pageid:** android-retailer-cart

**Navroute:** CART

**Platform:** android

**Role:** RETAILER

# Sourcefiles

- apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/cart/CartScreen.kt

**Shell:** retailer-android-root

**Status:** implemented

**Purpose:** Android retailer cart screen with list-based basket control, checkout sheet launch, supplier-closed guard dialog, and empty-cart branch.

# Layoutzones

**Zoneid:** cart-list-region

**Position:** main list when items exist

## Contents

- item count header
- Clear All text button
- cart item cards with placeholder image, size and pack pills, price, and quantity stepper

---

**Zoneid:** bottom-bar

**Position:** sticky bottom bar

**Visibilityrule:** visible when cart is not empty

## Contents

- subtotal row
- delivery row
- total cluster
- Checkout surface button

---

**Zoneid:** checkout-sheet

**Position:** modal bottom sheet

**Visibilityrule:** visible when uiState.showCheckout is true

## Contents

- CheckoutSheet overlay

---

**Zoneid:** supplier-closed-dialog

**Position:** alert dialog

**Visibilityrule:** visible when showSupplierClosedDialog is true

## Contents

- warning title
- supplier closed message
- I Understand, Place Order button
- Cancel button

---

**Zoneid:** empty-state

**Position:** center body

**Visibilityrule:** visible when uiState.isEmpty is true

## Contents

- double-ring shopping cart illustration
- empty headline
- helper copy
- Browse Catalog button

---


# Buttonplacements

**Button:** Clear All

**Zone:** cart-list-region header-right

**Style:** text destructive

---

**Button:** quantity decrement

**Zone:** each item stepper

**Style:** icon button

---

**Button:** quantity increment

**Zone:** each item stepper

**Style:** icon button

---

**Button:** Checkout

**Zone:** bottom-bar right

**Style:** filled pill surface

---

**Button:** I Understand, Place Order

**Zone:** supplier-closed-dialog confirm action

**Style:** filled button

---

**Button:** Browse Catalog

**Zone:** empty-state

**Style:** filled pill surface

---


# Iconplacements

**Icon:** Eco

**Zone:** cart item placeholder

---

**Icon:** Delete or Remove

**Zone:** quantity decrement control

---

**Icon:** Add

**Zone:** quantity increment control

---

**Icon:** ArrowForward

**Zone:** Checkout CTA trailing icon

---

**Icon:** ShoppingCart

**Zone:** empty-state hero

---

**Icon:** GridView

**Zone:** Browse Catalog CTA

---


# Interactiveflows

**Flowid:** basket-editing

## Steps

- Retailer increments or decrements quantities from each item row
- quantity, item total, and summary totals update
- decrement icon changes to delete when quantity reaches one

---

**Flowid:** checkout-gating

## Steps

- Retailer taps Checkout
- if supplier is closed, alert dialog interposes
- otherwise CheckoutSheet opens immediately

---


# Statevariants

- populated cart state
- checkout sheet active
- supplier closed dialog
- empty cart state
- snackbar feedback state

# Figureblueprints

- android retailer cart populated state
- cart bottom bar
- supplier closed dialog
- empty cart state

