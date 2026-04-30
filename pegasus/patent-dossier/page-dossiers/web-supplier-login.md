**Generatedat:** 2026-04-06

**Pageid:** web-auth-login

**Route:** /auth/login

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/auth/login/page.tsx

**Status:** implemented

**Purpose:** Supplier portal sign-in page with credential form, stale-cookie clearing, inline error handling, and password visibility toggle.

# Layoutzones

**Zoneid:** mobile-brand-strip

**Position:** top on mobile only

## Contents

- brand icon tile
- Pegasus Hub title
- Supplier Operations Portal subtitle

---

**Zoneid:** login-card

**Position:** central card

## Contents

- Sign in headline
- supporting subtitle
- optional inline error alert
- email field
- password field with show-hide button
- primary Sign In button
- link to create account

---

**Zoneid:** mobile-footer-copy

**Position:** bottom on mobile only

## Contents

- The Lab Industries copyright text

---


# Buttonplacements

**Button:** password visibility toggle

**Zone:** inside password field trailing edge

**Style:** icon button

---

**Button:** Sign In

**Zone:** login-card footer

**Style:** full-width primary CTA

---

**Button:** Create account link

**Zone:** below primary CTA

**Style:** inline text link

---


# Iconplacements

**Icon:** brand warehouse glyph

**Zone:** mobile-brand-strip

---

**Icon:** error alert icon

**Zone:** inline alert row

---

**Icon:** eye or eye-off glyph

**Zone:** password visibility toggle

---

**Icon:** spinner

**Zone:** Sign In button loading state

---


# Interactiveflows

**Flowid:** credential-login

## Steps

- Page clears stale auth cookies on mount
- User enters email and password
- User submits Sign In
- Page posts to /v1/auth/admin/login
- Successful response writes cookies and routes user to /

---

**Flowid:** password-peek

## Steps

- User taps trailing password eye control
- Password field switches between masked and plain text state

---


# Statevariants

- idle form
- inline error state
- submitting state with spinner

# Figureblueprints

- full login page
- login card close-up
- password field with visibility toggle
- error alert state
- submitting CTA state

