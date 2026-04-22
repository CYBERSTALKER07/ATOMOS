**Generatedat:** 2026-04-06

**Pageid:** web-supplier-profile

**Route:** /supplier/profile

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/profile/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier profile and operations-account page for managing identity, warehouse, banking, category, and shift-status data in a sectioned desktop form layout.

# Layoutzones

**Zoneid:** error-banner

**Position:** top of page when error exists

**Visibilityrule:** visible when an error is present

## Contents

- error icon
- error text

---

**Zoneid:** hero-header

**Position:** top full-width card

## Contents

- circular warehouse avatar
- supplier name, configuration status, category-email-phone summary
- edit action cluster

---

**Zoneid:** company-details-section

**Position:** below hero header

## Contents

- section title with accent border
- two-column card containing company and billing fields

---

**Zoneid:** warehouse-section

**Position:** below company details

## Contents

- warehouse address field
- latitude and longitude read-only fields

---

**Zoneid:** banking-section

**Position:** below warehouse section

## Contents

- bank name field
- account number field
- card number field
- payment gateway field

---

**Zoneid:** operating-categories-section

**Position:** below banking when categories exist

**Visibilityrule:** visible when operating_categories is non-empty

## Contents

- section title
- category chips

---

**Zoneid:** shift-status-section

**Position:** bottom card

## Contents

- status dot
- shift-state text
- manual override label when manual_off_shift is true

---


# Buttonplacements

**Button:** Retry

**Zone:** error fallback screen

**Style:** secondary button

**Visibilityrule:** profile failed to load and profile is absent

---

**Button:** Edit Profile

**Zone:** hero-header top-right

**Style:** primary button with leading edit icon

**Visibilityrule:** editing is false

---

**Button:** Cancel

**Zone:** hero-header top-right

**Style:** outline button

**Visibilityrule:** editing is true

---

**Button:** Save

**Zone:** hero-header top-right

**Style:** primary button

**Visibilityrule:** editing is true

---


# Iconplacements

**Icon:** error

**Zone:** error banner and error fallback

---

**Icon:** warehouse

**Zone:** hero avatar

---

**Icon:** verified

**Zone:** configuration status line in hero header

---

**Icon:** edit

**Zone:** Edit Profile button

---


# Interactiveflows

**Flowid:** profile-bootstrap

## Steps

- Page requests /v1/supplier/profile through apiFetch
- Skeleton placeholders render until the profile payload resolves
- Resolved data populates hero, section cards, and status chips

---

**Flowid:** edit-initialization

## Steps

- Supplier presses Edit Profile
- Page copies editable fields into a draft object
- Read-only display cells switch to outline inputs

---

**Flowid:** profile-save

## Steps

- Page diffs the draft against the fetched profile
- Changed fields only are submitted in a PUT request to /v1/supplier/profile
- Profile reloads after save and the page returns to read-only mode

---

**Flowid:** edit-cancel

## Steps

- Supplier presses Cancel during editing
- Draft state clears and read-only cards restore without network writes

---


# Datadependencies

## Readendpoints

- /v1/supplier/profile

## Writeendpoints

- /v1/supplier/profile

**Refreshmodel:** load on mount, retry on failure, and reload after successful profile updates

# Statevariants

- skeleton loading state
- hard error fallback with retry button
- inline error banner above populated profile
- default read-only profile sections
- edit mode with outlined field inputs
- saving state with disabled save button
- operating categories section visible
- manual off-shift marker visible

# Figureblueprints

- full supplier profile page with hero header and stacked sections
- hero header close-up showing avatar, configuration badge, and edit controls
- company-details section in edit mode with input fields
- shift-status card with active or off-shift indicator

