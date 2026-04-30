**Generatedat:** 2026-04-06

**Pageid:** ios-retailer-cart

**Viewname:** CartView

**Platform:** ios

**Role:** RETAILER

# Sourcefiles

- apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/CartView.swift

**Shell:** retailer-ios-root

**Status:** implemented

**Purpose:** Retailer basket-management screen with item-level quantity control, destructive removal, summary footer, and full-screen checkout handoff.

# Layoutzones

**Zoneid:** cart-list-region

**Position:** scroll body when cart has items

## Contents

- cart count header
- Clear All button
- cart item cards with image placeholder, product metadata, total price, quantity stepper, and delete affordance

---

**Zoneid:** bottom-bar

**Position:** sticky bottom summary bar

**Visibilityrule:** visible when cart is not empty

## Contents

- subtotal row
- delivery row
- total cluster
- Checkout pill button

---

**Zoneid:** empty-state

**Position:** center body

**Visibilityrule:** visible when cart.isEmpty is true

## Contents

- double-ring cart illustration
- empty headline
- helper copy
- Browse Catalog button

---

**Zoneid:** checkout-cover

**Position:** full-screen cover

**Visibilityrule:** visible when showCheckout is true

## Contents

- CheckoutView full-screen modal

---


# Buttonplacements

**Button:** Clear All

**Zone:** cart-list-region header-right

**Style:** text destructive

---

**Button:** quantity stepper decrement

**Zone:** each cart item card

**Style:** stepper button

---

**Button:** quantity stepper increment

**Zone:** each cart item card

**Style:** stepper button

---

**Button:** delete

**Zone:** cart item trailing overlay

**Style:** small destructive icon button

---

**Button:** Checkout

**Zone:** bottom-bar right

**Style:** accent pill CTA

---

**Button:** Browse Catalog

**Zone:** empty-state

**Style:** accent pill CTA

---


# Iconplacements

**Icon:** leaf.fill

**Zone:** cart item image placeholder

---

**Icon:** trash

**Zone:** cart item delete overlay and swipe action

---

**Icon:** arrow.right

**Zone:** Checkout CTA trailing edge

---

**Icon:** square.grid.2x2

**Zone:** Browse Catalog button

---

**Icon:** cart

**Zone:** empty-state illustration

---


# Interactiveflows

**Flowid:** quantity-adjustment

## Steps

- Retailer uses quantity stepper on a cart item
- CartManager updates item quantity
- line totals and bottom summary recompute

---

**Flowid:** item-removal

## Steps

- Retailer taps trailing delete control or swipe action
- CartManager removes item with animated transition

---

**Flowid:** checkout-handoff

## Steps

- Retailer taps Checkout
- CartView presents CheckoutView as a full-screen cover

---


# Statevariants

- populated cart state
- empty cart state
- quantity update state
- full-screen checkout cover active

# Figureblueprints

- retailer iOS cart populated state
- cart item row with quantity stepper and delete control
- cart bottom summary bar
- empty cart state

