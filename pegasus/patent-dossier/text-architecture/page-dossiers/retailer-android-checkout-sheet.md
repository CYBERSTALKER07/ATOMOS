# Technical Patent Architecture: retailer-android-checkout-sheet

Source Document: page-dossiers/retailer-android-checkout-sheet.md
Generated At: 2026-05-07T14:16:57.467Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- - apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/CheckoutSheet.kt

## System Architecture
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/CheckoutSheet.kt
- apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/CheckoutSheet.kt
- **Shell:** retailer-android-overlay
- **Status:** implemented
- **Purpose:** Android retailer checkout bottom sheet for reviewing order totals, selecting payment gateway from a split buy control, and showing processing or completion phases.
- **Zoneid:** sheet-header
- **Position:** top of modal bottom sheet

## Feature Set
1. Contents
2. Steps

## Algorithmic and Logical Flow
1. **Flowid:** gateway-selection
2. review phase
3. dropdown open state
4. processing phase
5. complete phase

## Mathematical Formulations
- No explicit closed-form equations were detected in this source file.

## Interfaces and Data Contracts
- No explicit API contract lines detected.

## Operational Constraints and State Rules
- review phase
- dropdown open state
- processing phase
- complete phase

## Claims-Oriented Technical Elements
1. Feature family coverage includes Contents; Steps.
2. Algorithmic sequence includes **Flowid:** gateway-selection | review phase | dropdown open state.
3. Integrity constraints include review phase; dropdown open state; processing phase; complete phase.
