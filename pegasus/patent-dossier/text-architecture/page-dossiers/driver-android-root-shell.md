# Technical Patent Architecture: driver-android-root-shell

Source Document: page-dossiers/driver-android-root-shell.md
Generated At: 2026-05-07T14:16:57.463Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- - apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/navigation/DriverNavigation.kt
- - apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/navigation/MainTabView.kt
- - Navigation pushes OffloadReviewScreen with orderId and retailerName

## System Architecture
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/navigation/DriverNavigation.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/navigation/MainTabView.kt
- apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/navigation/DriverNavigation.kt
- apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/navigation/MainTabView.kt
- **Shell:** driver-android-main
- **Status:** implemented
- **Purpose:** Authenticated driver execution shell that holds the core four-tab workspace and routes into scanner, offload review, payment waiting, cash collection, and correction flows.
- **Zoneid:** animated-content-region
- **Position:** center full-width

## Feature Set
1. Contents
2. Steps

## Algorithmic and Logical Flow
1. **Flowid:** scanner-to-offload
2. home tab active
3. map tab active
4. rides tab active
5. profile tab active
6. active ride bar present
7. scanner route open
8. offload review route open
9. payment waiting route open
10. cash collection route open
11. correction route open

## Mathematical Formulations
- No explicit closed-form equations were detected in this source file.

## Interfaces and Data Contracts
- No explicit API contract lines detected.

## Operational Constraints and State Rules
- home tab active
- map tab active
- rides tab active
- profile tab active
- active ride bar present
- scanner route open
- offload review route open
- payment waiting route open
- cash collection route open
- correction route open

## Claims-Oriented Technical Elements
1. Feature family coverage includes Contents; Steps.
2. Algorithmic sequence includes **Flowid:** scanner-to-offload | home tab active | map tab active.
3. Integrity constraints include home tab active; map tab active; rides tab active; profile tab active.
