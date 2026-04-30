**Generatedat:** 2026-04-06

**Pageid:** web-supplier-payment-config

**Route:** /supplier/payment-config

**Platform:** web

**Role:** SUPPLIER

**Sourcefile:** apps/admin-portal/app/supplier/payment-config/page.tsx

**Shell:** admin-shell

**Status:** implemented

**Purpose:** Supplier gateway-credential administration page for Click, Payme, and Global Pay, supporting manual onboarding, activation-state review, update, and deactivation.

# Layoutzones

**Zoneid:** header

**Position:** top constrained column

## Contents

- headline: Payment Gateways
- subtitle explaining Click, Payme, and Global Pay configuration

---

**Zoneid:** toast-region

**Position:** below header when toast exists

**Visibilityrule:** visible when toast state is non-null

## Contents

- success or error colored banner
- leading status icon
- toast message

---

**Zoneid:** provider-stack

**Position:** vertical list of provider cards

## Contents

- one card per gateway capability
- icon tile
- display name
- active or not-configured chip
- merchant or service preview text when configured
- connect placeholder badge or button
- manual-setup or update button
- optional deactivate button

---

**Zoneid:** expanded-manual-form

**Position:** inside provider card below divider

**Visibilityrule:** visible when expandedGateway matches card gateway

## Contents

- shield-led security hint
- merchant-id input
- optional service-id input
- secret-key password input
- per-field helper copy
- Cancel and Save or Update Configuration buttons

---

**Zoneid:** empty-state

**Position:** center card

**Visibilityrule:** visible when no capabilities and no configs are returned

## Contents

- payment icon
- No payment gateways available headline
- administrator-support body copy

---


# Buttonplacements

**Button:** Connect

**Zone:** provider-card action cluster

**Style:** primary small

**Visibilityrule:** provider supports redirect onboarding

---

**Button:** Connect coming soon badge

**Zone:** provider-card action cluster

**Style:** disabled status badge

**Visibilityrule:** manual-only provider

---

**Button:** Manual setup

**Zone:** provider-card action cluster

**Style:** outline or primary small

**Visibilityrule:** provider not configured

---

**Button:** Update

**Zone:** provider-card action cluster

**Style:** outline or primary small

**Visibilityrule:** provider configured

---

**Button:** Deactivate

**Zone:** provider-card action cluster

**Style:** danger-soft small

**Visibilityrule:** config active

---

**Button:** Cancel

**Zone:** expanded-manual-form footer-left

**Style:** outline

---

**Button:** Save Configuration

**Zone:** expanded-manual-form footer-right

**Style:** primary

**Visibilityrule:** new config

---

**Button:** Update Configuration

**Zone:** expanded-manual-form footer-right

**Style:** primary

**Visibilityrule:** editing existing config

---


# Iconplacements

**Icon:** gateway svg badge

**Zone:** provider-card leading tile

---

**Icon:** check-circle

**Zone:** active status chip and success toast

---

**Icon:** x-circle

**Zone:** error toast

---

**Icon:** clock

**Zone:** not-configured chip

---

**Icon:** shield

**Zone:** manual-form helper lines

---

**Icon:** key-round

**Zone:** manual-setup toggle button

---

**Icon:** chevron-down or chevron-up

**Zone:** manual-setup toggle button trailing edge

---

**Icon:** link2

**Zone:** connect action

---


# Interactiveflows

**Flowid:** gateway-bootstrap

## Steps

- Page requests /v1/supplier/payment-config
- Configured gateways and provider capabilities render as stacked cards
- Merchant and service previews appear without secret prefill

---

**Flowid:** manual-credential-save

## Steps

- Supplier expands Manual setup or Update on a gateway card
- Page seeds merchant and service values from existing config when present
- Supplier enters required fields
- Page posts to /v1/supplier/payment-config
- Success toast appears and cards reload

---

**Flowid:** gateway-deactivation

## Steps

- Supplier clicks Deactivate on an active gateway card
- Page deletes through /v1/supplier/payment-config with config_id payload
- Success toast appears and list refreshes

---


# Datadependencies

## Readendpoints

- /v1/supplier/payment-config

## Writeendpoints

- /v1/supplier/payment-config

**Refreshmodel:** initial fetch on mount plus reload after save and deactivate operations

# Statevariants

- loading spinner row
- provider stack with all cards collapsed
- expanded manual form for Click
- expanded manual form for Payme
- expanded manual form for Global Pay with service-id helper text
- success toast state
- error toast state
- no-capabilities empty state

# Figureblueprints

- full payment gateway stack
- configured gateway card close-up
- expanded Global Pay manual form
- success toast over gateway stack

