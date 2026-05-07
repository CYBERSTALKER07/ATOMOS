# Technical Patent Architecture: driver-ios-root-shell

Source Document: page-dossiers/driver-ios-root-shell.md
Generated At: 2026-05-07T14:16:57.465Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- - apps/driverappios/driverappios/Views/Components/ActiveRideBar.swift
- - RootView switches between LoginView and MainTabView based on TokenStore.isAuthenticated
- - ActiveRideBar when vm.hasActiveRoute and activeMission exist

## System Architecture
- Implementation Anchor: apps/driverappios/driverappios/PegasusDriverApp.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/MainTabView.swift
- Implementation Anchor: apps/driverappios/driverappios/Views/Components/ActiveRideBar.swift
- apps/driverappios/driverappios/PegasusDriverApp.swift
- apps/driverappios/driverappios/Views/MainTabView.swift
- apps/driverappios/driverappios/Views/Components/ActiveRideBar.swift
- **Shell:** driver-ios-main
- **Status:** implemented
- **Purpose:** Authenticated driver shell with tab-based execution workspace, full-screen map mode, and floating active-route summary above the tab bar.
- **Zoneid:** auth-gate
- **Position:** root level above shell

## Feature Set
1. Contents
2. Steps

## Algorithmic and Logical Flow
1. **Flowid:** home-to-map-mode
2. unauthenticated login state
3. home tab active
4. rides tab active
5. profile tab active
6. active route bar visible
7. full-screen map mode

## Mathematical Formulations
- No explicit closed-form equations were detected in this source file.

## Interfaces and Data Contracts
- No explicit API contract lines detected.

## Operational Constraints and State Rules
- unauthenticated login state
- home tab active
- rides tab active
- profile tab active
- active route bar visible
- full-screen map mode

## Claims-Oriented Technical Elements
1. Feature family coverage includes Contents; Steps.
2. Algorithmic sequence includes **Flowid:** home-to-map-mode | unauthenticated login state | home tab active.
3. Integrity constraints include unauthenticated login state; home tab active; rides tab active; profile tab active.
