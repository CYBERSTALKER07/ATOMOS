**Generatedat:** 2026-04-06

**Pageid:** payload-dispatch-success

**State:** allSealed == true

**Platform:** react-native-tablet

**Role:** PAYLOAD

# Sourcefiles

- apps/payload-terminal/App.tsx

**Shell:** payload-terminal-state-shell

**Status:** implemented

**Purpose:** Payload terminal dispatch-complete success state confirming manifest sealing and exposing dispatch codes before starting a new manifest.

# Layoutzones

**Zoneid:** success-center

**Position:** centered body

## Contents

- active truck mono label
- Manifest Secured headline
- Fleet Dispatched headline

---

**Zoneid:** dispatch-code-panel

**Position:** center body below headlines

**Visibilityrule:** visible when dispatchCodes has entries

## Contents

- Dispatch Codes heading
- rows of order ID to code pairs

---

**Zoneid:** new-manifest-action

**Position:** below code panel

## Contents

- New Manifest outlined button

---


# Buttonplacements

**Button:** New Manifest

**Zone:** new-manifest-action

**Style:** outlined button

---


# Iconplacements


# Interactiveflows

**Flowid:** dispatch-complete-reset

## Steps

- All loaded orders on the truck are sealed
- App enters success state
- Worker reviews dispatch codes if present
- Worker taps New Manifest
- App clears activeTruck, allSealed, and dispatchCodes, then returns to truck selection

---


# Statevariants

- success state without dispatch codes
- success state with dispatch code panel

# Figureblueprints

- payload dispatch success state
- dispatch code panel close-up

