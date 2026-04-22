**Generatedat:** 2026-04-06

**Pageid:** payload-truck-selection

**State:** token != null && activeTruck == null

**Platform:** react-native-tablet

**Role:** PAYLOAD

# Sourcefiles

- apps/payload-terminal/App.tsx

**Shell:** payload-terminal-state-shell

**Status:** implemented

**Purpose:** Payload tablet vehicle-selection state for choosing the target truck before loading a manifest.

# Layoutzones

**Zoneid:** header-bar

**Position:** top full-width

## Contents

- terminal title
- worker name
- Sign Out action

---

**Zoneid:** selection-center

**Position:** centered body

## Contents

- Select Target Vehicle label
- vehicle card row with label, license plate, and vehicle class
- loading-or-empty helper text

---


# Buttonplacements

**Button:** Sign Out

**Zone:** header-bar right

**Style:** text action

---

**Button:** truck card

**Zone:** selection-center

**Style:** card button

---


# Iconplacements


# Interactiveflows

**Flowid:** truck-selection

## Steps

- Authenticated worker waits for /v1/payloader/trucks to populate available vehicles
- Worker taps a truck card
- handleTruckSelect sets activeTruck and triggers manifest fetch
- App transitions into manifest workspace

---

**Flowid:** logout-from-selector

## Steps

- Worker taps Sign Out
- SecureStore credentials are cleared
- App returns to payload login state

---


# Statevariants

- vehicle cards available
- no vehicles available
- loading vehicles helper text

# Figureblueprints

- payload truck selection state
- payload truck card close-up

