**Generatedat:** 2026-04-06

**Pageid:** web-supplier-onboarding

**Route:** /supplier/onboarding

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/onboarding/page.tsx

**Shell:** none

**Status:** implemented-as-redirect

**Purpose:** Deprecated supplier onboarding route retained as a transitional redirect because onboarding is now fully embedded into the registration wizard at /auth/register.

# Layoutzones

**Zoneid:** redirect-indicator

**Position:** centered full-screen state

## Contents

- spinner glyph
- status text: Redirecting to dashboard…

---


# Buttonplacements


# Iconplacements

**Icon:** spinner glyph

**Zone:** centered redirect-indicator

---


# Interactiveflows

**Flowid:** conditional-redirect

## Steps

- Client effect reads supplier token from cookie
- If token is absent, router.replace('/auth/register') executes
- If token is present, router.replace('/supplier/dashboard') executes

---


# Datadependencies

## Readendpoints


## Writeendpoints


**Refreshmodel:** no network fetch; client-side token check determines redirect target

# Statevariants

- transient redirect indicator before navigation resolves
- redirect to registration wizard when unauthenticated
- redirect to supplier dashboard when authenticated

# Figureblueprints

- full-screen redirect indicator with spinner and redirect text
- route-flow figure showing /supplier/onboarding branching to /auth/register or /supplier/dashboard

