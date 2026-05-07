# Technical Patent Architecture: Surfaces

Source Document: page-dossiers/driver-android-secondary-surfaces.md
Generated At: 2026-05-07T14:16:57.464Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- No explicit abstract paragraph detected; refer to system and algorithm sections below.

## System Architecture
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/auth/LoginScreen.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/home/HomeScreen.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/map/MapScreen.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/manifest/ManifestScreen.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/profile/ProfileScreen.kt
- Implementation Anchor: apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/manifest/DeliveryCorrectionScreen.kt
- brand header
- phone and PIN text field column
- PIN visibility icon button
- login CTA and error state
- time-based greeting and status chips
- vehicle info card
- transit control card
- today summary band
- quick action row
- recent activity list
- centered placeholder icon
- stub title and explanatory subtitle
- UPCOMING header with pending count
- Loading Mode switch row
- ride card lazy list
- loading or empty states
- profile title header
- identity card
- truck and completion info grid
- quick actions row
- ride history list
- stats section
- header app bar
- manifest item cards

## Feature Set
1. Interactiveflows
2. Datadependencies
3. Read
4. Write
5. Minifeatures
6. Statevariants

## Algorithmic and Logical Flow
1. type phone and PIN
2. toggle PIN visibility
3. submit auth coroutine and persist session
4. show loading spinner during auth
5. idle
6. loading
7. error
8. refresh dashboard state
9. jump to map
10. jump to scanner
11. on route
12. static placeholder only
13. single stub state
14. toggle loading mode
15. tap ride to focus mission
16. refresh manifest
17. standard order
18. loading order
19. empty
20. sync state
21. logout session

## Mathematical Formulations
- No explicit closed-form equations were detected in this source file.

## Interfaces and Data Contracts
- No explicit API contract lines detected.

## Operational Constraints and State Rules
- idle
- loading
- error
- on route
- single stub state
- standard order
- loading order
- empty
- active
- history populated
- review
- editing
- confirming

## Claims-Oriented Technical Elements
1. Feature family coverage includes Interactiveflows; Datadependencies; Read; Write; Minifeatures; Statevariants.
2. Algorithmic sequence includes type phone and PIN | toggle PIN visibility | submit auth coroutine and persist session.
3. Integrity constraints include idle; loading; error; on route.
