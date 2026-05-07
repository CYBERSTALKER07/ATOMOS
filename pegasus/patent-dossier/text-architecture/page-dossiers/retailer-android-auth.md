# Technical Patent Architecture: retailer-android-auth

Source Document: page-dossiers/retailer-android-auth.md
Generated At: 2026-05-07T14:16:57.466Z
Mode: Text-only architecture extraction (no visual blueprint blocks)

## Technical Abstract
- - apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/auth/AuthScreen.kt
- - AuthViewModel authenticates and onAuthenticated advances into the main shell
- - AuthViewModel sends registration payload with logistics fields

## System Architecture
- Implementation Anchor: apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/auth/AuthScreen.kt
- apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/auth/AuthScreen.kt
- **Shell:** retailer-android-auth
- **Status:** implemented
- **Purpose:** Android retailer authentication and registration screen with expandable onboarding fields, map and GPS location capture, and logistics profile collection.
- **Zoneid:** brand-stack
- **Position:** top centered column

## Feature Set
1. Contents
2. Steps

## Algorithmic and Logical Flow
1. **Flowid:** login-flow
2. login mode
3. registration mode
4. GPS locating state
5. error text state
6. loading CTA state
7. map picker route handoff

## Mathematical Formulations
- No explicit closed-form equations were detected in this source file.

## Interfaces and Data Contracts
- No explicit API contract lines detected.

## Operational Constraints and State Rules
- login mode
- registration mode
- GPS locating state
- error text state
- loading CTA state
- map picker route handoff

## Claims-Oriented Technical Elements
1. Feature family coverage includes Contents; Steps.
2. Algorithmic sequence includes **Flowid:** login-flow | login mode | registration mode.
3. Integrity constraints include login mode; registration mode; GPS locating state; error text state.
