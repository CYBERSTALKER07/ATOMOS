**Generatedat:** 2026-04-06

**Pageid:** android-driver-cash-collection

**Navroute:** cash_collection/{orderId}/{amountUZS}

**Platform:** android

**Role:** DRIVER

# Sourcefiles

- apps/driver-app-android/app/src/main/java/com/thelab/driver/ui/screens/offload/CashCollectionScreen.kt

**Shell:** driver-android-main

**Status:** implemented

**Purpose:** Android driver cash-handling screen requiring explicit confirmation before delivery completion and guarding against accidental back navigation.

# Layoutzones

**Zoneid:** center-stack

**Position:** center vertical stack

## Contents

- Payments icon
- COLLECT CASH heading
- amount text
- instructional helper text

---

**Zoneid:** error-row

**Position:** below helper text when present

**Visibilityrule:** visible when state.error is non-null

## Contents

- centered error text

---

**Zoneid:** completion-cta

**Position:** bottom of center stack

## Contents

- Cash Collected — Complete button

---

**Zoneid:** exit-confirm-dialog

**Position:** modal overlay

**Visibilityrule:** visible when showExitConfirm is true

## Contents

- Leave cash collection title
- warning text
- Stay button
- Leave button

---


# Buttonplacements

**Button:** Cash Collected — Complete

**Zone:** completion-cta

**Style:** full-width primary

---

**Button:** Stay

**Zone:** exit-confirm-dialog confirm button slot

**Style:** text button

---

**Button:** Leave

**Zone:** exit-confirm-dialog dismiss button slot

**Style:** text button

---


# Iconplacements

**Icon:** Payments

**Zone:** center-stack top

---

**Icon:** CircularProgressIndicator

**Zone:** completion CTA while isCompleting

---


# Interactiveflows

**Flowid:** cash-completion

## Steps

- Driver reviews amount to collect
- Driver taps Cash Collected — Complete
- ViewModel calls collectCash
- Route exits through onComplete when state.completed is true

---

**Flowid:** guarded-back-navigation

## Steps

- Driver presses back before completion
- BackHandler opens confirmation dialog
- Driver chooses Stay or Leave
- Submission state suppresses back navigation entirely

---


# Statevariants

- cash collection idle state
- back-navigation confirmation dialog
- completion-in-flight state
- error state

# Figureblueprints

- android cash collection screen
- cash collection exit-confirmation dialog
- cash completion CTA state

