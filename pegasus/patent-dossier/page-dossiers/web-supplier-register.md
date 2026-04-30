**Generatedat:** 2026-04-06

**Pageid:** web-auth-register

**Route:** /auth/register

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/auth/register/page.tsx

**Status:** implemented

**Purpose:** Four-step supplier registration wizard combining account identity, warehouse location, business and fleet profile, category selection, and payment gateway preference.

# Layoutzones

**Zoneid:** mobile-brand-strip

**Position:** top on mobile only

## Contents

- brand icon tile
- Pegasus Hub title
- Supplier Registration subtitle

---

**Zoneid:** step-indicator

**Position:** above main card

## Contents

- Account step node
- Location step node
- Business step node
- Payments step node

---

**Zoneid:** wizard-card

**Position:** central card

## Contents

- step headline and subtitle
- optional inline error alert
- step-specific form body
- Back and Continue/Create Account buttons
- Sign in link on step 1

---


# Buttonplacements

**Button:** Locate

**Zone:** step 2 location field trailing action

**Style:** secondary inline CTA

---

**Button:** category chips

**Zone:** step 3 category grid

**Style:** multi-select chip

---

**Button:** cold-chain toggle

**Zone:** step 3 fleet profile card

**Style:** switch-like toggle

---

**Button:** payment gateway rows

**Zone:** step 4 payment list

**Style:** full-width selectable card rows

---

**Button:** Back

**Zone:** wizard footer left

**Style:** outline CTA when step > 0

---

**Button:** Continue to next step

**Zone:** wizard footer primary slot

**Style:** full-width primary CTA on non-final steps

---

**Button:** Create Supplier Account

**Zone:** wizard footer primary slot

**Style:** full-width primary CTA on final step

---

**Button:** Sign in link

**Zone:** step 1 footer text

**Style:** inline text link

---


# Iconplacements

**Icon:** brand warehouse glyph

**Zone:** mobile-brand-strip

---

**Icon:** step icons and checkmark states

**Zone:** step indicator nodes

---

**Icon:** Locate spinner or location glyph

**Zone:** step 2 Locate button

---

**Icon:** category checkmark

**Zone:** selected category chips

---

**Icon:** gateway icons

**Zone:** step 4 payment gateway rows

---

**Icon:** spinner

**Zone:** Create Supplier Account submitting state

---


# Interactiveflows

**Flowid:** wizard-progression

## Steps

- User completes account step
- User advances to location step
- User advances to business step with category selection
- User advances to payment step
- User submits registration
- On success cookies are written and user is routed to /supplier/dashboard

---

**Flowid:** geolocation-capture

## Steps

- User presses Locate on step 2
- Browser geolocation retrieves coordinates
- Page reverse-geocodes via Nominatim when possible
- Address and lat/lng fields are populated

---

**Flowid:** category-and-gateway-selection

## Steps

- User filters or browses category chips
- User selects one or more categories
- User selects one payment gateway card row as active

---


# Statevariants

- step 1 account fields
- step 2 location fields
- step 3 business and category selection
- step 4 payment gateway selection
- inline validation error state
- Create Account submitting state

# Figureblueprints

- full registration wizard step 1
- step indicator close-up
- location step with Locate action
- business step category grid
- payment step gateway rows
- final submitting state

