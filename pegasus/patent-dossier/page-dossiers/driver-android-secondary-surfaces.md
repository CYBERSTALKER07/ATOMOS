**Generatedat:** 2026-04-06

**Bundleid:** driver-android-secondary-surfaces

**Appid:** driver-app-android

**Platform:** android

**Role:** DRIVER

**Status:** implemented

# Surfaces

**Pageid:** android-driver-login

**Navroute:** login

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/auth/LoginScreen.kt

**Purpose:** Android driver sign-in form using phone and PIN with IME management and auth loading feedback.

## Layoutzones

- brand header
- phone and PIN text field column
- PIN visibility icon button
- login CTA and error state

## Buttonplacements

- PIN visibility icon button in PIN field trailing slot
- Login button below fields

## Iconplacements

- brand mark at screen top
- eye visibility icon

## Interactiveflows

- type phone and PIN
- toggle PIN visibility
- submit auth coroutine and persist session
- show loading spinner during auth

## Datadependencies

### Read

- driver login API

### Write

- driver token store

## Minifeatures

- phone prefill
- PIN field
- visibility toggle
- loading spinner
- error message

**Minifeaturecount:** 5

## Statevariants

- idle
- loading
- error

## Figureblueprints

- android login screen with phone and PIN form

---

**Pageid:** android-driver-home

**Navroute:** HOME

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/home/HomeScreen.kt

**Purpose:** Android driver home dashboard with status chips, vehicle card, transit control, and quick actions.

## Layoutzones

- time-based greeting and status chips
- vehicle info card
- transit control card
- today summary band
- quick action row
- recent activity list

## Buttonplacements

- Open Map CTA
- Scan QR CTA

## Iconplacements

- route-state icon
- truck or cargo glyphs in cards

## Interactiveflows

- refresh dashboard state
- jump to map
- jump to scanner

## Datadependencies

### Read

- ManifestViewModel state

### Write


## Minifeatures

- greeting
- status chips
- vehicle card
- transit control
- summary band
- Open Map CTA
- Scan QR CTA
- recent activity list

**Minifeaturecount:** 8

## Statevariants

- idle
- on route
- loading

## Figureblueprints

- android driver home dashboard

---

**Pageid:** android-driver-map

**Navroute:** MAP

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/map/MapScreen.kt

**Purpose:** Stub placeholder reserving the future Google Maps execution surface in the Android driver stack.

## Layoutzones

- centered placeholder icon
- stub title and explanatory subtitle

## Buttonplacements

- none; stub surface

## Iconplacements

- Map icon centered

## Interactiveflows

- static placeholder only

## Datadependencies

### Read


### Write


## Minifeatures

- map pending icon
- placeholder messaging

**Minifeaturecount:** 2

## Statevariants

- single stub state

## Figureblueprints

- stub map placeholder figure

---

**Pageid:** android-driver-rides

**Navroute:** RIDES

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/manifest/ManifestScreen.kt

**Purpose:** Android route manifest ledger with loading-mode reversal for physical truck packing and upcoming stop review.

## Layoutzones

- UPCOMING header with pending count
- Loading Mode switch row
- ride card lazy list
- loading or empty states

## Buttonplacements

- Loading Mode switch in header
- ride card tap target

## Iconplacements

- loading sequence badge
- status pill

## Interactiveflows

- toggle loading mode
- tap ride to focus mission
- refresh manifest

## Datadependencies

### Read

- ManifestViewModel.state

### Write

- selected mission state

## Minifeatures

- pending count badge
- loading mode switch
- ride cards
- sequence badge
- status pill
- empty state

**Minifeaturecount:** 6

## Statevariants

- standard order
- loading order
- empty
- loading

## Figureblueprints

- android rides manifest with loading mode switch

---

**Pageid:** android-driver-profile

**Navroute:** PROFILE

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/profile/ProfileScreen.kt

**Purpose:** Android driver profile screen with truck identity, stats, quick actions, and ride-history review.

## Layoutzones

- profile title header
- identity card
- truck and completion info grid
- quick actions row
- ride history list
- stats section

## Buttonplacements

- Sync quick action
- Logout quick action
- Settings quick action

## Iconplacements

- initials avatar
- quick action icons

## Interactiveflows

- sync state
- logout session
- review history

## Datadependencies

### Read

- ManifestViewModel driver and order stats

### Write

- sync or logout state

## Minifeatures

- identity card
- status pill
- truck grid
- Sync action
- Logout action
- Settings action
- history ledger
- stats band

**Minifeaturecount:** 8

## Statevariants

- active
- idle
- history populated

## Figureblueprints

- android driver profile screen

---

**Pageid:** android-driver-correction

**Navroute:** correction/{orderId}/{retailerName}

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/manifest/DeliveryCorrectionScreen.kt

**Purpose:** Alias dossier for the Android driver delivery-correction workflow already documented as a primary execution surface.

## Layoutzones

- header app bar
- manifest item cards
- sticky summary footer
- correction bottom sheet and confirmation dialog overlays

## Buttonplacements

- Modify item action
- confirm amendment action

## Iconplacements

- item correction glyphs
- dialog warning icon

## Interactiveflows

- open correction editor
- adjust delivered and rejected quantities
- confirm amendment

## Datadependencies

### Read

- delivery correction payload

### Write

- correction submission

## Minifeatures

- item cards
- sticky footer
- bottom sheet editor
- confirmation dialog

**Minifeaturecount:** 4

## Statevariants

- review
- editing
- confirming

## Figureblueprints

- delivery correction alias figure

---


