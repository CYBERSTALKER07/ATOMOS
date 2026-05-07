# Technical Patent Architecture: driver-ios-payment-waiting

Source Document: page-dossiers/driver-ios-payment-waiting.md
Generated At: 2026-05-07T14:16:57.465Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- - apps/driverappios/driverappios/Views/PaymentWaitingView.swift
- - View opens websocket to /v1/ws/driver with driver_id and bearer token
- - Page listens for PAYMENT_SETTLED matching the current orderId

## System Architecture
- Implementation Anchor: apps/driverappios/driverappios/Views/PaymentWaitingView.swift
- apps/driverappios/driverappios/Views/PaymentWaitingView.swift
- **Shell:** driver-ios-main
- **Status:** implemented
- **Purpose:** Driver payment-settlement holding screen that waits for a websocket settlement event before enabling delivery completion.
- **Zoneid:** status-stack
- **Position:** center vertical stack

## Feature Set
1. Contents
2. Steps

## Algorithmic and Logical Flow
1. **Flowid:** settlement-wait-loop
2. awaiting payment state
3. settled state
4. completion-in-flight state
5. error state
6. websocket reconnect behavior after failure

## Mathematical Formulations
- No explicit closed-form equations were detected in this source file.

## Interfaces and Data Contracts
- Endpoint: /v1/ws/driver

## Operational Constraints and State Rules
- awaiting payment state
- settled state
- completion-in-flight state
- error state
- websocket reconnect behavior after failure
- **Flowid:** settlement-wait-loop

## Claims-Oriented Technical Elements
1. Feature family coverage includes Contents; Steps.
2. Algorithmic sequence includes **Flowid:** settlement-wait-loop | awaiting payment state | settled state.
3. Contract surface is exposed through /v1/ws/driver.
4. Integrity constraints include awaiting payment state; settled state; completion-in-flight state; error state.
