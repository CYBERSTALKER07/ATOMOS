# Technical Patent Architecture: driver-android-offload-review

Source Document: page-dossiers/driver-android-offload-review.md
Generated At: 2026-05-07T14:16:57.463Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- - apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/offload/OffloadReviewScreen.kt
- - Confirm Offload or Amend and Confirm Offload button with spinner state
- - Driver uses stepper controls to increase or decrease rejected quantity per line item

## System Architecture
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/offload/OffloadReviewScreen.kt
- apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/offload/OffloadReviewScreen.kt
- **Shell:** driver-android-main
- **Status:** implemented
- **Purpose:** Android driver cargo-review screen for checking accepted totals, excluding damaged units, and confirming offload before payment or cash collection routing.
- **Zoneid:** header
- **Position:** top full-width

## Feature Set
1. Contents
2. Left
3. Steps

## Algorithmic and Logical Flow
1. **Flowid:** line-item-exclusion-adjustment
2. clean offload state
3. partially rejected line-item state
4. fully rejected line-item state
5. submitting state
6. error state

## Mathematical Formulations
- No explicit closed-form equations were detected in this source file.

## Interfaces and Data Contracts
- No explicit API contract lines detected.

## Operational Constraints and State Rules
- clean offload state
- partially rejected line-item state
- fully rejected line-item state
- submitting state
- error state

## Claims-Oriented Technical Elements
1. Feature family coverage includes Contents; Left; Steps.
2. Algorithmic sequence includes **Flowid:** line-item-exclusion-adjustment | clean offload state | partially rejected line-item state.
3. Integrity constraints include clean offload state; partially rejected line-item state; fully rejected line-item state; submitting state.
