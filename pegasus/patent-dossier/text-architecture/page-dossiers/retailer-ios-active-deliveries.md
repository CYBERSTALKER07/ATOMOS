# Technical Patent Architecture: retailer-ios-active-deliveries

Source Document: page-dossiers/retailer-ios-active-deliveries.md
Generated At: 2026-05-07T14:16:57.468Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- - apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ActiveDeliveriesView.swift
- - View loads retailer orders and filters them to active statuses
- - Retailer may alternatively tap Show QR for token-enabled orders

## System Architecture
- Implementation Anchor: apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ActiveDeliveriesView.swift
- apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ActiveDeliveriesView.swift
- **Shell:** retailer-ios-overlay
- **Status:** implemented
- **Purpose:** Retailer active-delivery monitor showing only live orders, detail-sheet drilldown, and QR handoff from a dedicated delivery surface.
- **Zoneid:** delivery-scroll-region
- **Position:** main body

## Feature Set
1. Contents
2. Steps

## Algorithmic and Logical Flow
1. **Flowid:** active-delivery-review
2. loading state
3. empty state
4. active deliveries list
5. detail sheet open
6. QR overlay open

## Mathematical Formulations
- No explicit closed-form equations were detected in this source file.

## Interfaces and Data Contracts
- No explicit API contract lines detected.

## Operational Constraints and State Rules
- loading state
- empty state
- active deliveries list
- detail sheet open
- QR overlay open

## Claims-Oriented Technical Elements
1. Feature family coverage includes Contents; Steps.
2. Algorithmic sequence includes **Flowid:** active-delivery-review | loading state | empty state.
3. Integrity constraints include loading state; empty state; active deliveries list; detail sheet open.
