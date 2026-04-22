**Generatedat:** 2026-04-06

**Pageid:** web-supplier-settings

**Route:** /supplier/settings

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/settings/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier operational-settings page for controlling manual off-shift status and day-by-day business-hour windows backed by the shared supplier-shift context.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

- headline: Settings
- subtitle describing business hours and shift availability

---

**Zoneid:** shift-status-card

**Position:** first card below header

## Contents

- section title and explanatory copy
- manual off-shift toggle pill
- effective OPEN or CLOSED status label

---

**Zoneid:** business-hours-card

**Position:** second card below shift-status

## Contents

- section title and scheduling guidance
- seven day rows each with enable checkbox, day label, and open-close time controls or Closed text

---

**Zoneid:** save-row

**Position:** bottom action band

## Contents

- Save Changes button
- saved-success or failed-save status text

---


# Buttonplacements

**Button:** ON SHIFT / OFF SHIFT toggle

**Zone:** shift-status-card

**Style:** pill toggle button

---

**Button:** Day enabled checkbox

**Zone:** business-hours-card day row

**Style:** checkbox control

---

**Button:** Save Changes

**Zone:** save-row left

**Style:** primary button

---


# Iconplacements

**Icon:** spinner ring

**Zone:** loading state

---

**Icon:** status dot

**Zone:** manual off-shift toggle pill

---


# Interactiveflows

**Flowid:** settings-bootstrap

## Steps

- Page reads shared supplier shift context from useSupplierShift
- Context bootstraps from /v1/supplier/profile and exposes manual_off_shift, is_active, and operating_schedule
- When the hook finishes loading, local form state mirrors the shared shift state

---

**Flowid:** schedule-editing

## Steps

- Supplier toggles day checkboxes to enable or disable days
- Enabled days expose open and close time inputs
- Changing a time updates local schedule state only until save

---

**Flowid:** shift-save

## Steps

- Supplier presses Save Changes
- Page assembles final enabled-day schedule and manual off-shift value
- Shared hook patches /v1/supplier/shift with manual_off_shift and operating_schedule
- Save status indicator reports success or failure

---


# Datadependencies

## Readendpoints

- /v1/supplier/profile

## Writeendpoints

- /v1/supplier/shift

**Refreshmodel:** shared-context bootstrap on mount plus explicit save action for schedule changes

# Statevariants

- settings loading spinner state
- default shift and schedule form state
- days enabled with time pickers
- days disabled showing Closed text
- save in progress state
- saved successfully message
- failed save message

# Figureblueprints

- full settings page with shift card, hours card, and save row
- shift-status card close-up with manual off-shift toggle and effective status label
- business-hours rows showing enabled and disabled day variants

