# Technical Patent Architecture: Layoutzones

Source Document: page-dossiers/retailer-ios-root-shell.md
Generated At: 2026-05-07T14:16:57.469Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- - Retailer navigates to delivery detail or payment sheet based on current order state
- - Retailer selects dashboard, procurement, insights, auto-order, AI predictions, inbox, profile, or settings

## System Architecture
- Implementation Anchor: apps/retailer-app-ios/reatilerapp/reatilerapp/ContentView.swift
- **Zoneid:** tab-layer
- **Position:** full-screen base layer

## Feature Set
1. Contents
2. Left
3. Center
4. Right
5. Steps

## Algorithmic and Logical Flow
1. **Flowid:** active-orders-drilldown
2. base tab shell
3. floating active orders visible
4. sidebar open with dimmed background
5. active deliveries bottom sheet open
6. payment sheet open
7. future demand sheet open
8. cart sheet open
9. insights sheet open

## Mathematical Formulations
- No explicit closed-form equations were detected in this source file.

## Interfaces and Data Contracts
- No explicit API contract lines detected.

## Operational Constraints and State Rules
- base tab shell
- floating active orders visible
- sidebar open with dimmed background
- active deliveries bottom sheet open
- payment sheet open
- future demand sheet open
- cart sheet open
- insights sheet open

## Claims-Oriented Technical Elements
1. Feature family coverage includes Contents; Left; Center; Right; Steps.
2. Algorithmic sequence includes **Flowid:** active-orders-drilldown | base tab shell | floating active orders visible.
3. Integrity constraints include base tab shell; floating active orders visible; sidebar open with dimmed background; active deliveries bottom sheet open.
