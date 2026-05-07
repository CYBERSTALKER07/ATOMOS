# Technical Patent Architecture: retailer-android-active-deliveries-sheet

Source Document: page-dossiers/retailer-android-active-deliveries-sheet.md
Generated At: 2026-05-07T14:16:57.466Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- - apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/ActiveDeliveriesSheet.kt
- - active delivery cards with progress ring, order metadata, countdown row, and action buttons

## System Architecture
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/ActiveDeliveriesSheet.kt
- apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/ActiveDeliveriesSheet.kt
- **Shell:** retailer-android-overlay
- **Status:** implemented
- **Purpose:** Android retailer active-deliveries bottom sheet listing in-progress orders with detail and QR actions.
- **Zoneid:** sheet-header
- **Position:** top of modal bottom sheet

## Feature Set
1. Contents
2. Steps

## Algorithmic and Logical Flow
1. **Flowid:** delivery-sheet-review
2. active deliveries sheet open
3. delivery card with QR enabled
4. delivery card awaiting dispatch

## Mathematical Formulations
- No explicit closed-form equations were detected in this source file.

## Interfaces and Data Contracts
- No explicit API contract lines detected.

## Operational Constraints and State Rules
- active deliveries sheet open
- delivery card with QR enabled
- delivery card awaiting dispatch

## Claims-Oriented Technical Elements
1. Feature family coverage includes Contents; Steps.
2. Algorithmic sequence includes **Flowid:** delivery-sheet-review | active deliveries sheet open | delivery card with QR enabled.
3. Integrity constraints include active deliveries sheet open; delivery card with QR enabled; delivery card awaiting dispatch.
