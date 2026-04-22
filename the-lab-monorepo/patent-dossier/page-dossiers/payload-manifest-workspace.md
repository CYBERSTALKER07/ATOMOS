**Generatedat:** 2026-04-06

**Pageid:** payload-manifest-workspace

**State:** token != null && activeTruck != null && allSealed == false

**Platform:** react-native-tablet

**Role:** PAYLOAD

**Sourcefile:** apps/payload-terminal/App.tsx

**Shell:** payload-terminal-state-shell

**Status:** implemented

**Purpose:** Warehouse payloader tablet workspace for selecting orders on a truck, checklist scanning line items, and sealing the load for dispatch.

# Layoutzones

**Zoneid:** left-pane

**Position:** fixed-width left column

**Width:** 288

## Contents

- terminal header with title and active truck
- truck toggle bar
- scrollable order list with active and cleared states

---

**Zoneid:** right-header

**Position:** top of right pane

## Contents

- selected order ID
- retailer ID, payment gateway, amount text line
- truck badge chip

---

**Zoneid:** checklist-region

**Position:** center right pane

## Contents

- scrollable manifest checklist
- tap-to-toggle checkbox control
- brand code line
- item label line

---

**Zoneid:** seal-footer

**Position:** bottom of right pane

## Contents

- Mark as Loaded action button

---


# Buttonplacements

**Button:** truck selector in left-pane toggle bar

**Zone:** left-pane truck toggle row

**Style:** segmented text button

---

**Button:** order selector

**Zone:** left-pane order list row

**Style:** list row button

**Visibilityrule:** disabled for sealed orders

---

**Button:** manifest item checkbox row

**Zone:** checklist-region

**Style:** full-row toggle

---

**Button:** Mark as Loaded

**Zone:** seal-footer

**Style:** primary footer CTA

**Visibilityrule:** enabled only when all selected-order checklist items are checked and not currently sealing

---


# Iconplacements

**Icon:** text-only checkmark glyph

**Zone:** checkbox control when item is scanned

---


# Interactiveflows

**Flowid:** switch-active-truck

## Steps

- Payload operator taps truck label in left-pane toggle row
- handleTruckSelect resets local manifest state
- fetchManifest reloads orders and checklist for that truck

---

**Flowid:** select-order-and-clear-items

## Steps

- Payload operator taps order row in left pane
- Right pane updates selected order header and checklist
- Operator taps each checklist row to toggle scanned state
- Checkbox fills accent color with checkmark when scanned

---

**Flowid:** seal-order

## Steps

- All checklist items for selected order are scanned
- Mark as Loaded becomes enabled
- Operator taps Mark as Loaded
- App posts to /v1/payload/seal
- Order is added to sealedOrderIds and next remaining order is auto-selected or allSealed becomes true

---


# Datadependencies

## Readendpoints

- /v1/payloader/trucks
- /v1/payloader/orders?vehicle_id={truckId}&state=LOADED

## Writeendpoints

- /v1/payload/seal

**Offlinefallback:** manifest fetch attempts SecureStore cache keyed by manifest_{truckId}

# Statevariants

- manifest loading in left pane
- no pending orders in left pane
- active order row styling
- cleared order row styling
- selected order right-pane detail
- no selected order placeholder
- sealing disabled footer
- sealing in-progress footer

# Figureblueprints

- full two-pane manifest workspace
- left-pane truck selector and order list
- right-pane order header
- checklist row with checkbox state
- seal footer CTA

