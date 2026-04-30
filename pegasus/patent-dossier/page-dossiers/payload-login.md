**Generatedat:** 2026-04-06

**Pageid:** payload-login

**State:** token == null

**Platform:** react-native-tablet

**Role:** PAYLOAD

# Sourcefiles

- apps/payload-terminal/App.tsx

**Shell:** payload-terminal-state-shell

**Status:** implemented

**Purpose:** Payload worker sign-in state for tablet authentication via phone number and 6-digit PIN.

# Layoutzones

**Zoneid:** brand-header

**Position:** top centered column

## Contents

- Pegasus Payload Terminal label
- Payloader Login headline

---

**Zoneid:** credential-stack

**Position:** centered form column

## Contents

- phone number input
- 6-digit PIN input with centered wide letter spacing
- Sign In button

---


# Buttonplacements

**Button:** Sign In

**Zone:** credential-stack footer

**Style:** full-width filled button

---


# Iconplacements


# Interactiveflows

**Flowid:** payloader-login

## Steps

- Worker enters phone number and PIN
- Worker taps Sign In
- App posts to /v1/auth/payloader/login
- Successful response persists token, name, and supplier ID in SecureStore
- App advances into truck selection state

---


# Statevariants

- idle login state
- authenticating state
- login failure alert

# Figureblueprints

- payload login state
- payload login authenticating CTA state

