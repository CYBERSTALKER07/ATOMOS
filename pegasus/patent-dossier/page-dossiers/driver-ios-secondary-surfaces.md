**Generatedat:** 2026-04-06

**Bundleid:** driver-ios-secondary-surfaces

**Appid:** driver-app-ios

**Platform:** ios

**Role:** DRIVER

**Status:** implemented

# Surfaces

**Pageid:** ios-driver-root-gate

**Viewname:** RootView

**Surfacetype:** root-gate

**Sourcefile:** apps/driverappios/driverappios/PegasusDriverApp.swift

**Purpose:** Session gate that decides between driver login flow and the protected multi-tab shell.

## Layoutzones

- app bootstrap region hosting root state decision
- login presentation branch for unauthenticated driver
- main-tab branch for authenticated driver

## Buttonplacements

- no persistent buttons; state resolution presents login or shell

## Iconplacements

- none at gate level

## Interactiveflows

- read token and driver session state on launch
- show login when token is absent or invalid
- show MainTabView when session is active

## Datadependencies

### Read

- driver token store
- driver session bootstrap

### Write


## Minifeatures

- auth branch resolution
- protected-shell presentation
- guest-shell suppression

**Minifeaturecount:** 3

## Statevariants

- authenticated branch
- unauthenticated branch

## Figureblueprints

- route-flow figure from root gate to login or main shell

---

**Pageid:** ios-driver-login

**Viewname:** LoginView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/LoginView.swift

**Purpose:** Phone and PIN sign-in screen for driver session acquisition.

## Layoutzones

- brand crest and title stack
- phone field and PIN field cluster
- PIN visibility toggle inside secure entry
- login CTA and error message strip

## Buttonplacements

- PIN visibility eye toggle inside PIN field trailing edge
- Login button at form footer

## Iconplacements

- brand disk at page top
- eye or eye-slash in PIN field

## Interactiveflows

- input phone and PIN
- toggle PIN visibility
- submit DriverApi login and persist token
- on success dismiss into protected shell

## Datadependencies

### Read

- DriverApi.login

### Write

- TokenHolder session state

## Minifeatures

- phone prefill
- secure PIN entry
- visibility toggle
- loading disable
- error banner

**Minifeaturecount:** 5

## Statevariants

- idle login form
- auth loading state
- error state

## Figureblueprints

- full login screen with phone and PIN fields
- PIN field close-up with visibility toggle

---

**Pageid:** ios-driver-home

**Viewname:** HomeView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/HomeView.swift

**Purpose:** Driver dashboard summarizing mission status, truck identity, daily metrics, and quick-entry actions into execution surfaces.

## Layoutzones

- dynamic greeting header
- truck and route status chips
- vehicle card and transit control card
- today summary metrics
- Open Map and quick action buttons
- recent activity ledger

## Buttonplacements

- Open Map CTA in transit control zone
- Scan QR quick action button
- View Manifest quick action button

## Iconplacements

- antenna or moon status icon
- truck or route glyphs in status cards

## Interactiveflows

- load missions and driver summary on appear
- pull to refresh home data
- jump to map, scanner, or rides manifest

## Datadependencies

### Read

- FleetViewModel mission summary
- driver metrics
- recent activity feed

### Write


## Minifeatures

- time-of-day greeting
- truck plate chip
- route-state chip
- vehicle card
- daily summary
- Open Map CTA
- Scan QR shortcut
- View Manifest shortcut
- recent activity list

**Minifeaturecount:** 9

## Statevariants

- loading home state
- idle off-route state
- on-route active state

## Figureblueprints

- driver home dashboard with quick action band
- status chip and summary card close-up

---

**Pageid:** ios-driver-map

**Viewname:** FleetMapView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/FleetMapView.swift

**Purpose:** Primary driver execution surface joining map telemetry, mission selection, QR scan initiation, payment branching, and delivery correction entry.

## Layoutzones

- full-screen map region with mission markers
- zoom focus control cycling Me, Target, Both
- selected mission side or bottom detail region
- bottom action strip for Scan QR and Correct Delivery
- navigation bridge into scanner, offload, payment, cash, and correction flows

## Buttonplacements

- zoom focus cycle button over map chrome
- Scan QR primary action in selected mission panel
- Correct Delivery secondary action in selected mission panel

## Iconplacements

- mission markers on map
- location and target glyphs in focus control

## Interactiveflows

- select mission from map marker
- cycle map framing mode
- launch QR scanner for selected mission
- after scan branch into offload review then payment or cash collection
- open correction workflow for selected mission

## Datadependencies

### Read

- TelemetryViewModel live location
- FleetViewModel missions
- geofence and QR validation payloads

### Write

- navigation path for execution subflows

## Minifeatures

- live mission markers
- mission selection
- focus cycle control
- selected mission detail pane
- Scan QR CTA
- Correct Delivery CTA
- payment branch routing
- cash branch routing

**Minifeaturecount:** 8

## Statevariants

- no mission selected
- mission previewing state
- active delivery state

## Figureblueprints

- full operational map with selected mission panel
- map chrome close-up with focus control and CTA band

---

**Pageid:** ios-driver-rides

**Viewname:** RidesListView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/RidesListView.swift

**Purpose:** Route-manifest ledger of upcoming rides with physical loading-sequence toggle.

## Layoutzones

- UPCOMING header with pending count
- Loading Mode toggle row
- mission ride card list
- pull-to-refresh scaffold

## Buttonplacements

- Loading Mode switch in header row
- ride card tap target to select or focus mission

## Iconplacements

- sequence badge when loading mode is enabled
- status badge on ride cards

## Interactiveflows

- toggle loading mode to reverse sequence for warehouse loading
- tap ride card to select mission and synchronize with map
- refresh pending missions

## Datadependencies

### Read

- FleetViewModel.pendingMissions

### Write

- FleetViewModel.selectMission

## Minifeatures

- pending count badge
- loading mode toggle
- sequence badge
- ride amount summary
- item count summary
- status pill

**Minifeaturecount:** 6

## Statevariants

- standard route order
- loading-sequence order
- empty rides list

## Figureblueprints

- rides manifest screen with loading mode toggle
- single ride card with sequence badge

---

**Pageid:** ios-driver-profile

**Viewname:** ProfileView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/ProfileView.swift

**Purpose:** Driver identity, truck metadata, quick operations, and ride-history review surface.

## Layoutzones

- driver title header
- driver identity card with status pill
- truck and metrics info grid
- quick actions row
- ride history ledger
- stats section

## Buttonplacements

- Sync quick action button
- Logout quick action button
- Offline Verifier quick action button or sheet trigger

## Iconplacements

- driver initials avatar
- status pill
- quick action glyphs

## Interactiveflows

- sync local and live route state
- open offline verifier
- logout driver session

## Datadependencies

### Read

- FleetViewModel driver profile and mission history
- TelemetryService state

### Write

- logout session
- offline verifier sheet state

## Minifeatures

- identity card
- on-duty status pill
- truck info grid
- Sync action
- Logout action
- Offline Verifier access
- ride history list
- revenue stat
- completed-orders stat

**Minifeaturecount:** 9

## Statevariants

- on duty
- idle
- history populated
- history sparse

## Figureblueprints

- driver profile with quick actions and stats
- identity card close-up

---

**Pageid:** ios-driver-offline-verifier

**Viewname:** OfflineVerifierView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/OfflineVerifierView.swift

**Purpose:** Cryptographic offline verification terminal for zero-connectivity proof of delivery and fraud detection.

## Layoutzones

- terminal header with protocol name
- protocol status band
- state-driven body switching among idle, syncing, ready, scanning, verified, fraud, and error cards

## Buttonplacements

- Sync Route Manifest button in idle state
- Start Scan button in ready state

## Iconplacements

- scanner overlay in scanning state
- success or fraud glyphs in verification result cards

## Interactiveflows

- sync manifest hash locally
- start offline scan
- verify QR against manifest hash
- show verified order result or fraud reason

## Datadependencies

### Read

- OfflineDeliveryStore
- AVFoundation camera feed
- SHA256Helper manifest validation

### Write

- offline verification state machine

## Minifeatures

- protocol status pill
- sync progress state
- manifest hash display
- Start Scan CTA
- scanner overlay
- verified result card
- fraud result card
- error result card

**Minifeaturecount:** 8

## Statevariants

- idle
- syncing
- ready
- scanning
- verified
- fraud
- error

## Figureblueprints

- offline verification terminal in ready state
- two-panel verified versus fraud outcome figure

---

**Pageid:** ios-driver-mission-detail-sheet

**Viewname:** MissionDetailSheet

**Surfacetype:** overlay

**Sourcefile:** apps/driverappios/driverappios/Views/MissionDetailSheet.swift

**Purpose:** Bottom-sheet mission inspector exposing geofence clearance, endpoint distance, payment badge, and scan or correction actions.

## Layoutzones

- order header with monospaced order ID and gateway badge
- delivery endpoint card with coordinates and geofence state
- distance and proximity indicators
- footer action cluster

## Buttonplacements

- Delivery Correction text button
- Scan QR primary button

## Iconplacements

- geofence status dot
- gateway badge
- location glyph in endpoint card

## Interactiveflows

- present mission detail over map
- launch correction from sheet
- launch scanner from sheet

## Datadependencies

### Read

- Mission object
- distance calculation
- geofence validation state

### Write

- scan callback
- correction callback

## Minifeatures

- monospaced order ID
- gateway badge
- amount display
- geofence dot
- distance meter
- Delivery Correction CTA
- Scan QR CTA

**Minifeaturecount:** 7

## Statevariants

- cleared geofence
- fault geofence

## Figureblueprints

- mission detail sheet over map backdrop
- endpoint card close-up with geofence state

---

**Pageid:** ios-driver-map-marker-detail-sheet

**Viewname:** MapMarkerDetailSheet

**Surfacetype:** overlay

**Sourcefile:** apps/driverappios/driverappios/Views/Components/MapMarkerDetailSheet.swift

**Purpose:** Compact marker drill-down for map stops, emphasizing stop identity and route semantics without entering the full mission sheet.

## Layoutzones

- marker header with stop label
- order or stop metadata stack
- micro action row

## Buttonplacements

- compact dismiss or expand controls inside overlay

## Iconplacements

- marker glyph
- status dot or stop-type icon

## Interactiveflows

- tap map marker to open compact stop detail
- dismiss or escalate into fuller mission inspection

## Datadependencies

### Read

- selected map marker payload

### Write

- marker-detail dismissal or expansion state

## Minifeatures

- marker label
- stop metadata lines
- compact overlay
- expand affordance

**Minifeaturecount:** 4

## Statevariants

- compact marker summary
- expanded marker context

## Figureblueprints

- map marker detail overlay figure

---


