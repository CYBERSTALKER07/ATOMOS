# Technical Patent Architecture: payload-truck-selection

Source Document: page-dossiers/payload-truck-selection.md
Generated At: 2026-05-07T14:16:57.466Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- - vehicle card row with label, license plate, and vehicle class
- - Authenticated worker waits for /v1/payloader/trucks to populate available vehicles
- - handleTruckSelect sets activeTruck and triggers manifest fetch

## System Architecture
- Implementation Anchor: apps/payload-terminal/App.ts
- apps/payload-terminal/App.tsx
- **Shell:** payload-terminal-state-shell
- **Status:** implemented
- **Purpose:** Payload tablet vehicle-selection state for choosing the target truck before loading a manifest.
- **Zoneid:** header-bar
- **Position:** top full-width

## Feature Set
1. Contents
2. Steps

## Algorithmic and Logical Flow
1. **Flowid:** truck-selection
2. vehicle cards available
3. no vehicles available
4. loading vehicles helper text

## Mathematical Formulations
- **State:** token != null && activeTruck == null

## Interfaces and Data Contracts
- Endpoint: /v1/payloader/trucks

## Operational Constraints and State Rules
- vehicle cards available
- no vehicles available
- loading vehicles helper text

## Claims-Oriented Technical Elements
1. Feature family coverage includes Contents; Steps.
2. Algorithmic sequence includes **Flowid:** truck-selection | vehicle cards available | no vehicles available.
3. Contract surface is exposed through /v1/payloader/trucks.
4. Mathematical or scoring expressions are explicitly used for optimization or estimation.
5. Integrity constraints include vehicle cards available; no vehicles available; loading vehicles helper text.
