# Technical Patent Architecture: Surfaces

Source Document: page-dossiers/driver-ios-secondary-surfaces.md
Generated At: 2026-05-07T14:16:57.465Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- - no persistent buttons; state resolution presents login or shell
- - navigation bridge into scanner, offload, payment, cash, and correction flows
- - Correct Delivery secondary action in selected mission panel

## System Architecture
- Implementation Anchor: apps/driverappios/driverappios/PegasusDriverApp.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/LoginView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/HomeView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/FleetMapView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/RidesListView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/ProfileView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/OfflineVerifierView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/MissionDetailSheet.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/Components/MapMarkerDetailSheet.swift
- app bootstrap region hosting root state decision
- login presentation branch for unauthenticated driver
- main-tab branch for authenticated driver
- brand crest and title stack
- phone field and PIN field cluster
- PIN visibility toggle inside secure entry
- login CTA and error message strip
- dynamic greeting header
- truck and route status chips
- vehicle card and transit control card
- today summary metrics
- Open Map and quick action buttons
- recent activity ledger
- full-screen map region with mission markers
- zoom focus control cycling Me, Target, Both
- selected mission side or bottom detail region
- bottom action strip for Scan QR and Correct Delivery
- navigation bridge into scanner, offload, payment, cash, and correction flows
- UPCOMING header with pending count
- Loading Mode toggle row
- mission ride card list

## Feature Set
1. Interactiveflows
2. Datadependencies
3. Read
4. Write
5. Minifeatures
6. Statevariants

## Algorithmic and Logical Flow
1. read token and driver session state on launch
2. show login when token is absent or invalid
3. show MainTabView when session is active
4. authenticated branch
5. unauthenticated branch
6. input phone and PIN
7. toggle PIN visibility
8. submit DriverApi login and persist token
9. on success dismiss into protected shell
10. idle login form
11. auth loading state
12. error state
13. load missions and driver summary on appear
14. pull to refresh home data
15. jump to map, scanner, or rides manifest
16. loading home state
17. idle off-route state
18. on-route active state
19. select mission from map marker
20. cycle map framing mode
21. launch QR scanner for selected mission
22. after scan branch into offload review then payment or cash collection
23. open correction workflow for selected mission
24. no mission selected

## Mathematical Formulations
- No explicit closed-form equations were detected in this source file.

## Interfaces and Data Contracts
- No explicit API contract lines detected.

## Operational Constraints and State Rules
- authenticated branch
- unauthenticated branch
- idle login form
- auth loading state
- error state
- loading home state
- idle off-route state
- on-route active state
- no mission selected
- mission previewing state
- active delivery state
- standard route order
- loading-sequence order
- empty rides list
- on duty
- idle
- history populated
- history sparse
- syncing
- ready
- scanning
- verified
- fraud

## Claims-Oriented Technical Elements
1. Feature family coverage includes Interactiveflows; Datadependencies; Read; Write; Minifeatures; Statevariants.
2. Algorithmic sequence includes read token and driver session state on launch | show login when token is absent or invalid | show MainTabView when session is active.
3. Integrity constraints include authenticated branch; unauthenticated branch; idle login form; auth loading state.
