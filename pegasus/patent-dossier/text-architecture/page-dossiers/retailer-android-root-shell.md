# Technical Patent Architecture: Layoutzones

Source Document: page-dossiers/retailer-android-root-shell.md
Generated At: 2026-05-07T14:16:57.467Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- - floating bar only on HOME, ORDERS, and SUPPLIERS tabs when active order count > 0
- - Card path deep-links to external payment app or cash path enters pending state
- - Retailer chooses dashboard, procurement, insights, auto-order, or AI predictions

## System Architecture
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/navigation/RetailerNavigation.kt
- **Zoneid:** top-bar
- **Position:** top full-width

## Feature Set
1. Contents
2. Left
3. Center
4. Right
5. Visibilityrules
6. Steps

## Algorithmic and Logical Flow
1. **Flowid:** global-order-attention
2. base shell with no overlays
3. floating active orders visible
4. sidebar open with scrim
5. active deliveries sheet open
6. order detail sheet open
7. QR overlay open
8. payment sheet choose phase
9. payment sheet processing or failed or success phase

## Mathematical Formulations
- No explicit closed-form equations were detected in this source file.

## Interfaces and Data Contracts
- No explicit API contract lines detected.

## Operational Constraints and State Rules
- floating bar only on HOME, ORDERS, and SUPPLIERS tabs when active order count > 0
- ---
- **Zoneid:** global-overlays
- **Position:** above shell content
- base shell with no overlays
- floating active orders visible
- sidebar open with scrim
- active deliveries sheet open
- order detail sheet open
- QR overlay open
- payment sheet choose phase
- payment sheet processing or failed or success phase

## Claims-Oriented Technical Elements
1. Feature family coverage includes Contents; Left; Center; Right; Visibilityrules; Steps.
2. Algorithmic sequence includes **Flowid:** global-order-attention | base shell with no overlays | floating active orders visible.
3. Integrity constraints include floating bar only on HOME, ORDERS, and SUPPLIERS tabs when active order count > 0; ---; **Zoneid:** global-overlays; **Position:** above shell content.
