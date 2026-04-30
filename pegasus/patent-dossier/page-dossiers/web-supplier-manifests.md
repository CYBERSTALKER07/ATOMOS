**Generatedat:** 2026-04-06

**Pageid:** web-supplier-manifests

**Route:** /supplier/manifests

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/manifests/page.tsx

**Shell:** admin-shell

**Status:** implemented-as-redirect

**Purpose:** Legacy supplier manifests route that immediately redirects to the canonical supplier orders surface instead of rendering standalone UI.

# Layoutzones

**Zoneid:** redirect-guard

**Position:** server-side route handler

## Contents

- no persisted UI; redirect('/supplier/orders') executes during route resolution

---


# Buttonplacements


# Iconplacements


# Interactiveflows

**Flowid:** route-alias-redirect

## Steps

- Supplier navigates to /supplier/manifests
- Next.js redirect executes immediately
- Browser lands on /supplier/orders where the actual manifest and order workflow resides

---


# Datadependencies

## Readendpoints


## Writeendpoints


**Refreshmodel:** no local data fetch; route delegates to /supplier/orders

# Statevariants

- immediate server redirect with no rendered intermediate page

# Figureblueprints

- route-flow figure showing /supplier/manifests aliasing into /supplier/orders

