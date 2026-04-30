**Generatedat:** 2026-04-06

**Pageid:** android-retailer-auth

**Navroute:** AuthScreen

**Platform:** android

**Role:** RETAILER

# Sourcefiles

- apps/retailer-app-android/app/src/main/java/com/thelab/retailer/ui/screens/auth/AuthScreen.kt

**Shell:** retailer-android-auth

**Status:** implemented

**Purpose:** Android retailer authentication and registration screen with expandable onboarding fields, map and GPS location capture, and logistics profile collection.

# Layoutzones

**Zoneid:** brand-stack

**Position:** top centered column

## Contents

- black storefront icon disk
- The Lab title
- Retailer Portal subtitle

---

**Zoneid:** credential-core

**Position:** main form column

## Contents

- phone field
- password field

---

**Zoneid:** registration-extension

**Position:** below core credentials when login mode is off

**Visibilityrule:** visible when isLoginMode is false

## Contents

- store name field
- owner name field
- address field
- Open Map button
- Use GPS button
- selected location label
- tax ID field
- receiving window fields
- access type chip buttons
- ceiling height field

---

**Zoneid:** primary-action-region

**Position:** below form fields

## Contents

- Sign In or Create Account button
- error text when present
- mode-toggle text button

---


# Buttonplacements

**Button:** Open Map

**Zone:** registration-extension location row

**Style:** outlined button

---

**Button:** Use GPS

**Zone:** registration-extension location row

**Style:** outlined button

---

**Button:** Street

**Zone:** registration-extension access row

**Style:** outlined chip toggle

---

**Button:** Alley

**Zone:** registration-extension access row

**Style:** outlined chip toggle

---

**Button:** Dock

**Zone:** registration-extension access row

**Style:** outlined chip toggle

---

**Button:** Sign In

**Zone:** primary-action-region

**Style:** full-width filled pill

**Visibilityrule:** isLoginMode true

---

**Button:** Create Account

**Zone:** primary-action-region

**Style:** full-width filled pill

**Visibilityrule:** isLoginMode false

---

**Button:** mode toggle

**Zone:** primary-action-region footer

**Style:** text button

---


# Iconplacements

**Icon:** Storefront

**Zone:** brand-stack

---

**Icon:** Map

**Zone:** Open Map button

---

**Icon:** MyLocation

**Zone:** Use GPS button

---

**Icon:** CircularProgressIndicator

**Zone:** Use GPS button when locating and primary CTA when state.isLoading

---


# Interactiveflows

**Flowid:** login-flow

## Steps

- Retailer enters phone and password
- Retailer taps Sign In
- AuthViewModel authenticates and onAuthenticated advances into the main shell

---

**Flowid:** registration-flow

## Steps

- Retailer switches out of login mode
- AnimatedVisibility expands onboarding fields
- Retailer captures location by map picker or GPS
- Retailer submits Create Account
- AuthViewModel sends registration payload with logistics fields

---


# Statevariants

- login mode
- registration mode
- GPS locating state
- error text state
- loading CTA state
- map picker route handoff

# Figureblueprints

- android retailer login mode
- android retailer registration mode
- location capture row
- loading and error state

