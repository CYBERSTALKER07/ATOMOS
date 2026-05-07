# Technical Patent Architecture: Layoutzones

Source Document: page-dossiers/web-supplier-fleet.md
Generated At: 2026-05-07T14:16:57.471Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- - subtitle: Provision drivers, register vehicles, manage fleet capacity
- - vehicle table with class, label, plate, capacity, assigned driver, status, actions when vehicles tab active
- - driver table with clickable rows, phone, type badge, assignment select, status when drivers tab active

## System Architecture
- Implementation Anchor: apps/admin-portal/app/supplier/fleet/page.ts
- **Zoneid:** header
- **Position:** top full-width

## Feature Set
1. Contents
2. Left
3. Right
4. Steps
5. Readendpoints
6. Writeendpoints

## Algorithmic and Logical Flow
1. **Flowid:** driver-provisioning
2. drivers-tab loading spinner
3. drivers empty state
4. vehicles empty state
5. drivers table with assignment select
6. vehicles table with action chips
7. add-driver drawer open
8. add-vehicle drawer open in class mode
9. add-vehicle drawer open in dimension mode with computed VU
10. PIN reveal modal
11. driver detail drawer open

## Mathematical Formulations
- No explicit closed-form equations were detected in this source file.

## Interfaces and Data Contracts
- Endpoint: /v1/supplier/fleet/capacity
- Endpoint: /v1/supplier/fleet/drivers
- Endpoint: /v1/supplier/fleet/drivers/{driverId}/assign-vehicle
- Endpoint: /v1/supplier/fleet/drivers/{id}
- Endpoint: /v1/supplier/fleet/vehicles
- Endpoint: /v1/supplier/fleet/vehicles/{vehicleId}
- Endpoint: /v1/vehicle/{vehicleId}/clear-returns
- /v1/supplier/fleet/drivers
- /v1/supplier/fleet/vehicles
- /v1/supplier/fleet/capacity
- /v1/supplier/fleet/drivers/{id}
- /v1/supplier/fleet/drivers/{driverId}/assign-vehicle
- /v1/supplier/fleet/vehicles/{vehicleId}
- /v1/vehicle/{vehicleId}/clear-returns
- **Refreshmodel:** initial fetch on mount followed by targeted reloads after create, assign, deactivate, and clear-return actions

## Operational Constraints and State Rules
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

## Claims-Oriented Technical Elements
1. Feature family coverage includes Contents; Left; Right; Steps; Readendpoints; Writeendpoints.
2. Algorithmic sequence includes **Flowid:** driver-provisioning | drivers-tab loading spinner | drivers empty state.
3. Contract surface is exposed through /v1/supplier/fleet/capacity, /v1/supplier/fleet/drivers, /v1/supplier/fleet/drivers/{driverId}/assign-vehicle, /v1/supplier/fleet/drivers/{id}, /v1/supplier/fleet/vehicles, /v1/supplier/fleet/vehicles/{vehicleId}.
4. Integrity constraints include drivers-tab loading spinner; drivers empty state; vehicles empty state; drivers table with assignment select.
