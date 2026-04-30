**Generatedat:** 2026-04-06

**Pageid:** web-supplier-fleet

**Route:** /supplier/fleet

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/fleet/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier fleet command center for driver provisioning, vehicle registration, assignment control, capacity visibility, and credential reveal for newly provisioned operators.

# Layoutzones

**Zoneid:** header

**Position:** top full-width

## Contents

### Left

- back link to supplier dashboard
- headline: Fleet Management
- subtitle: Provision drivers, register vehicles, manage fleet capacity

### Right

- conditional + Add Driver CTA when drivers tab active
- conditional + Add Vehicle CTA when vehicles tab active

---

**Zoneid:** tab-selector

**Position:** below header

## Contents

- Drivers tab pill with count
- Vehicles tab pill with count

---

**Zoneid:** kpi-row

**Position:** below tab-selector

## Contents

- driver metrics row when drivers tab active
- vehicle metrics row when vehicles tab active

---

**Zoneid:** primary-table-region

**Position:** main content card

## Contents

- vehicle table with class, label, plate, capacity, assigned driver, status, actions when vehicles tab active
- driver table with clickable rows, phone, type badge, assignment select, status when drivers tab active

---

**Zoneid:** driver-add-drawer

**Position:** right slide-out overlay

**Visibilityrule:** open when showAdd is true

## Contents

- name input
- phone input
- driver-type chip toggle
- assign-vehicle select
- license-plate input
- inline error copy
- Provision Driver button

---

**Zoneid:** pin-reveal-modal

**Position:** centered modal overlay

**Visibilityrule:** visible when createdPin exists

## Contents

- success icon disk
- driver identity text
- dashed login-pin panel
- warning banner instructing copy-once behavior
- Done button

---

**Zoneid:** driver-detail-drawer

**Position:** right slide-out overlay

**Visibilityrule:** open when selectedDriver exists

## Contents

- initial avatar circle
- driver name and phone
- type badge
- detail cell grid
- optional current-location row

---

**Zoneid:** vehicle-add-drawer

**Position:** right slide-out overlay

**Visibilityrule:** open when showAddVehicle is true

## Contents

- class-versus-dimensions mode toggle
- vehicle-class select
- computed capacity readout
- dimension inputs when LxWxH mode selected
- label input
- license plate input
- Register Vehicle button

---


# Buttonplacements

**Button:** + Add Driver

**Zone:** header-right

**Style:** primary

**Visibilityrule:** drivers tab active

---

**Button:** + Add Vehicle

**Zone:** header-right

**Style:** primary

**Visibilityrule:** vehicles tab active

---

**Button:** Drivers tab

**Zone:** tab-selector

**Style:** segmented tab

---

**Button:** Vehicles tab

**Zone:** tab-selector

**Style:** segmented tab

---

**Button:** Deactivate

**Zone:** vehicle row action cluster

**Style:** ghost-danger

**Visibilityrule:** vehicle is active

---

**Button:** Clear Returns

**Zone:** vehicle row action cluster

**Style:** outline-warning

**Visibilityrule:** vehicle active, assigned, and pending returns exist

---

**Button:** Provision Driver

**Zone:** driver-add-drawer footer

**Style:** full-width primary

---

**Button:** Done

**Zone:** pin-reveal-modal footer

**Style:** full-width primary

---

**Button:** Class

**Zone:** vehicle-add-drawer top-right mode toggle

**Style:** segmented button

---

**Button:** LxWxH

**Zone:** vehicle-add-drawer top-right mode toggle

**Style:** segmented button

---

**Button:** Register Vehicle

**Zone:** vehicle-add-drawer footer

**Style:** full-width primary

---


# Iconplacements

**Icon:** warning

**Zone:** pin-reveal-modal warning banner

---

**Icon:** success checkmark disk

**Zone:** pin-reveal-modal top

---

**Icon:** driver initial avatar

**Zone:** driver-detail-drawer header

---

**Icon:** driver-type badge

**Zone:** driver rows and driver-detail-drawer header

---


# Interactiveflows

**Flowid:** driver-provisioning

## Steps

- Supplier stays on Drivers tab
- Clicks + Add Driver
- Completes drawer form and optionally preassigns vehicle
- Page posts to /v1/supplier/fleet/drivers
- Drawer closes and one-time PIN modal appears

---

**Flowid:** vehicle-registration

## Steps

- Supplier switches to Vehicles tab
- Clicks + Add Vehicle
- Supplier either keeps class mode or enters dimensions for computed VU
- Page posts to /v1/supplier/fleet/vehicles
- Vehicle list reloads

---

**Flowid:** assignment-control

## Steps

- Supplier changes assignment select inside a driver row
- Page patches /v1/supplier/fleet/drivers/{driverId}/assign-vehicle
- Drivers and vehicles refresh to reflect occupancy state

---

**Flowid:** driver-inspection

## Steps

- Supplier clicks a driver row
- Page requests /v1/supplier/fleet/drivers/{id}
- Right-side drawer opens with identity, stats, and current location when present

---


# Datadependencies

## Readendpoints

- /v1/supplier/fleet/drivers
- /v1/supplier/fleet/vehicles
- /v1/supplier/fleet/capacity
- /v1/supplier/fleet/drivers/{id}

## Writeendpoints

- /v1/supplier/fleet/drivers
- /v1/supplier/fleet/vehicles
- /v1/supplier/fleet/drivers/{driverId}/assign-vehicle
- /v1/supplier/fleet/vehicles/{vehicleId}
- /v1/vehicle/{vehicleId}/clear-returns

**Refreshmodel:** initial fetch on mount followed by targeted reloads after create, assign, deactivate, and clear-return actions

# Statevariants

- drivers-tab loading spinner
- drivers empty state
- vehicles empty state
- drivers table with assignment select
- vehicles table with action chips
- add-driver drawer open
- add-vehicle drawer open in class mode
- add-vehicle drawer open in dimension mode with computed VU
- PIN reveal modal
- driver detail drawer open

# Figureblueprints

- fleet page on drivers tab with KPI row and assignment table
- fleet page on vehicles tab with capacity metrics and vehicle table
- add-driver drawer
- PIN reveal modal
- driver detail drawer
- add-vehicle drawer in dimensions mode

