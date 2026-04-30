**Generatedat:** 2026-04-06

**Pageid:** ios-driver-cash-collection

**Viewname:** CashCollectionView

**Platform:** ios

**Role:** DRIVER

# Sourcefiles

- apps/driverappios/driverappios/Views/CashCollectionView.swift

**Shell:** driver-ios-main

**Status:** implemented

**Purpose:** Driver cash-confirmation screen used when retailer payment is collected physically before delivery completion.

# Layoutzones

**Zoneid:** top-close-row

**Position:** top safe-area inset

## Contents

- circular close button aligned right

---

**Zoneid:** center-cash-stack

**Position:** center vertical stack

## Contents

- banknote icon
- Collect Cash title
- order ID
- amount
- helper copy

---

**Zoneid:** error-row

**Position:** above footer CTA when present

**Visibilityrule:** visible when errorMessage exists

## Contents

- destructive error text

---

**Zoneid:** footer-cta

**Position:** bottom full-width

## Contents

- Cash Collected — Complete button with optional spinner

---


# Buttonplacements

**Button:** Close

**Zone:** top-close-row right

**Style:** icon button

---

**Button:** Cash Collected — Complete

**Zone:** footer-cta

**Style:** full-width primary

---


# Iconplacements

**Icon:** xmark

**Zone:** top-close-row

---

**Icon:** banknote.fill

**Zone:** center-cash-stack top

---

**Icon:** ProgressView

**Zone:** footer CTA when completing

---


# Interactiveflows

**Flowid:** cancel-cash-collection

## Steps

- Driver taps close button
- View exits through onCancel callback

---

**Flowid:** collect-cash-and-complete

## Steps

- Driver confirms physical collection
- Driver taps Cash Collected — Complete
- View calls collectCash(orderId)
- Successful response exits through onCompleted callback

---


# Statevariants

- cash collection idle state
- completion-in-flight state
- inline error state

# Figureblueprints

- cash collection screen
- cash collection CTA state
- cash collection error state

