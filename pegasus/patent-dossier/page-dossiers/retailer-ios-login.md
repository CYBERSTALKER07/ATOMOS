**Generatedat:** 2026-04-06

**Pageid:** ios-retailer-login

**Viewname:** LoginView

**Platform:** ios

**Role:** RETAILER

# Sourcefiles

- apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/LoginView.swift

**Shell:** retailer-ios-auth

**Status:** implemented

**Purpose:** Retailer authentication and registration screen combining login, store onboarding, map-based location capture, and logistics intake fields.

# Layoutzones

**Zoneid:** brand-stack

**Position:** top centered column

## Contents

- gradient storefront logo disk
- Pegasus title
- Retailer Portal subtitle

---

**Zoneid:** credential-core

**Position:** main form column

## Contents

- phone field
- password field

---

**Zoneid:** registration-extension

**Position:** below core credentials when sign-up mode active

**Visibilityrule:** visible when isLoginMode is false

## Contents

- store name field
- owner name field
- store address field
- Open Map button
- Share Location button
- selected location label
- tax ID field
- receiving window open and close fields
- loading access type chip row
- ceiling height field

---

**Zoneid:** error-row

**Position:** below form fields when auth error exists

**Visibilityrule:** visible when auth.errorMessage is non-null

## Contents

- warning icon
- error text

---

**Zoneid:** primary-action-region

**Position:** below form stack

## Contents

- Sign In or Create Account button
- mode toggle link

---


# Buttonplacements

**Button:** Open Map

**Zone:** registration-extension location row

**Style:** outlined pill

---

**Button:** Share Location

**Zone:** registration-extension location row

**Style:** outlined pill

---

**Button:** Street

**Zone:** registration-extension access type row

**Style:** chip toggle

---

**Button:** Alley

**Zone:** registration-extension access type row

**Style:** chip toggle

---

**Button:** Dock

**Zone:** registration-extension access type row

**Style:** chip toggle

---

**Button:** Sign In

**Zone:** primary-action-region

**Style:** full-width gradient CTA

**Visibilityrule:** isLoginMode true

---

**Button:** Create Account

**Zone:** primary-action-region

**Style:** full-width gradient CTA

**Visibilityrule:** isLoginMode false

---

**Button:** mode toggle link

**Zone:** primary-action-region footer

**Style:** text button

---


# Iconplacements

**Icon:** storefront.fill

**Zone:** brand-stack logo disk

---

**Icon:** map

**Zone:** Open Map button

---

**Icon:** location.fill

**Zone:** Share Location button

---

**Icon:** arrow.right

**Zone:** primary CTA trailing edge when not loading

---

**Icon:** ProgressView

**Zone:** primary CTA when auth.isLoading

---

**Icon:** exclamationmark.triangle.fill

**Zone:** error-row

---


# Interactiveflows

**Flowid:** retailer-login

## Steps

- Retailer enters phone and password
- Retailer taps Sign In
- AuthManager login executes and authenticated state transitions into the app shell

---

**Flowid:** retailer-registration

## Steps

- Retailer toggles to sign-up mode
- Additional onboarding fields animate into view
- Retailer may open map picker or share GPS location
- Retailer submits Create Account
- AuthManager register executes with location and logistics metadata

---

**Flowid:** location-capture

## Steps

- Retailer opens map picker or uses current location
- Latitude and longitude populate state
- Selected location label renders below the location row

---


# Statevariants

- login mode
- registration mode
- GPS locating state
- error state
- submitting state
- map picker route handoff

# Figureblueprints

- retailer iOS login mode
- retailer iOS registration mode
- location capture row
- error and submitting CTA state

