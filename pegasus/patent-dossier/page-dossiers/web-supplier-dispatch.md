**Generatedat:** 2026-04-06

**Pageid:** web-supplier-dispatch

**Route:** /supplier/dispatch

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/dispatch/page.tsx

**Shell:** admin-shell

**Status:** implemented-as-redirect

**Purpose:** Legacy supplier dispatch route that immediately redirects to the canonical supplier orders surface rather than rendering a dedicated dispatch page.

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

- Supplier navigates to /supplier/dispatch
- Next.js redirect executes immediately
- Browser lands on /supplier/orders where dispatch actions are actually performed

---


# Datadependencies

## Readendpoints


## Writeendpoints


**Refreshmodel:** no local data fetch; route delegates to /supplier/orders

# Statevariants

- immediate server redirect with no rendered intermediate page

# Figureblueprints

- route-flow figure showing /supplier/dispatch aliasing into /supplier/orders

