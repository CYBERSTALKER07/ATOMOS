**Generatedat:** 2026-04-06

**Pageid:** ios-driver-main-shell

**Viewname:** MainTabView

**Platform:** ios

**Role:** DRIVER

# Sourcefiles

- apps/driverappios/driverappios/LabDriverApp.swift
- apps/driverappios/driverappios/Views/MainTabView.swift
- apps/driverappios/driverappios/Views/Components/ActiveRideBar.swift

**Shell:** driver-ios-main

**Status:** implemented

**Purpose:** Authenticated driver shell with tab-based execution workspace, full-screen map mode, and floating active-route summary above the tab bar.

# Layoutzones

**Zoneid:** auth-gate

**Position:** root level above shell

## Contents

- RootView switches between LoginView and MainTabView based on TokenStore.isAuthenticated

---

**Zoneid:** tab-layer

**Position:** base layer when not on map

## Contents

- Home tab
- Rides tab
- Profile tab

---

**Zoneid:** map-mode

**Position:** full-screen replacement state

## Contents

- FleetMapView with go-back closure

---

**Zoneid:** bottom-safe-area-inset

**Position:** above tab bar

## Contents

- ActiveRideBar when vm.hasActiveRoute and activeMission exist

---


# Buttonplacements

**Button:** Home tab

**Zone:** tab bar

**Style:** TabView tab item

---

**Button:** Rides tab

**Zone:** tab bar

**Style:** TabView tab item

---

**Button:** Profile tab

**Zone:** tab bar

**Style:** TabView tab item

---

**Button:** Home open-map trigger

**Zone:** home content callback

**Style:** screen CTA transitions to map mode

---

**Button:** ActiveRideBar

**Zone:** bottom safe-area inset

**Style:** floating pill CTA to map mode

---

**Button:** Map goBack

**Zone:** full-screen map mode

**Style:** callback-driven return control

---


# Iconplacements

**Icon:** house.fill

**Zone:** Home tab

---

**Icon:** list.bullet

**Zone:** Rides tab

---

**Icon:** person.fill

**Zone:** Profile tab

---

**Icon:** map.fill

**Zone:** Map mode logical tab target

---

**Icon:** chevron.right

**Zone:** ActiveRideBar trailing affordance

---


# Interactiveflows

**Flowid:** home-to-map-mode

## Steps

- Driver taps map CTA from HomeView
- selectedTab switches to map with snappy animation
- FleetMapView replaces the normal tab shell

---

**Flowid:** active-route-drilldown

## Steps

- Active route exists
- ActiveRideBar appears above tab bar
- Driver taps ActiveRideBar
- Shell transitions into full-screen map mode

---

**Flowid:** authenticated-root-branching

## Steps

- RootView checks TokenStore.isAuthenticated
- Authenticated drivers go directly to MainTabView
- Unauthenticated drivers stay on LoginView

---


# Statevariants

- unauthenticated login state
- home tab active
- rides tab active
- profile tab active
- active route bar visible
- full-screen map mode

# Figureblueprints

- driver iOS shell with tab bar
- active route bar above tab bar
- full-screen map mode
- root auth-gate transition

