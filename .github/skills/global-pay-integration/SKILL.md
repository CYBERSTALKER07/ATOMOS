---
name: global-pay-integration
description: 'Implement Global Pay integration in this monorepo from documentation-only inputs or live credentials. Use for Global Pay Checkout Service, Cards Service, Payments Service Public, hosted checkout redirects, direct gateway card flows, status polling, reversals, supplier credential setup, GNK item fields, sandbox and production separation, and docs-first rollout when credentials are not available.'
argument-hint: 'Describe the target: checkout flow, direct gateway, cards, refunds, supplier config, docs-only rollout, sandbox, or production cutover'
---

# Global Pay Integration

## What This Skill Does

Use this skill to implement or review Global Pay integration across backend, web, mobile, and supplier configuration surfaces in this workspace.

This skill is built for the current repo reality:

- Global Pay is an additional gateway beside CLICK and PAYME
- supplier gateway credentials live in backend vault-backed configuration
- customer payment execution must stay separate from supplier onboarding
- the team may only have documentation at first, not live credentials or certified access

This skill keeps five concerns separate:

- customer payment execution
- supplier credential configuration
- hosted Checkout Service versus direct Gateway API
- docs-only implementation versus credentialed validation
- sandbox, staging, and production environments

## When to Use

Activate this skill when the request involves any of the following:

- Global Pay Checkout Service flows using user service tokens and redirect URLs
- Global Pay Cards Service flows for card creation and OTP confirmation
- Global Pay Payments Service Public flows for init, perform, status, or revert
- supplier payment configuration for Global Pay credentials in the admin portal
- webhook, callback, status polling, expiry, 3DS, or reconciliation handling for Global Pay
- deciding whether the repo should use hosted checkout or direct gateway card APIs
- implementing Global Pay from documentation before live credentials are available
- reviewing whether Global Pay code incorrectly mixes docs assumptions with production-ready behavior

Do not use this skill for unrelated gateways. Do not treat Checkout Service and Payments Service Public as interchangeable. Do not mark the integration production-ready if only documentation has been reviewed and no credentialed validation has happened.

## Repo-Specific Guardrails

- Customer payment execution belongs in checkout and payment code, not supplier onboarding flows.
- Supplier credential storage stays backend-only in the vault-backed path.
- Global Pay supplier-facing credentials in this repo map to `service_id`, OAuth username, and OAuth password.
- If the shared storage model still uses generic fields like `merchant_id` and `secret_key`, map them truthfully instead of inventing new frontend semantics.
- Never expose raw credentials, PAN, CVV, OTP codes, or auth tokens to the frontend, mobile apps, logs, or analytics.
- Callback, redirect, popup, or app-return states are never authoritative for settlement. Final payment state must be confirmed server-side.
- Amounts must be handled in the smallest currency unit described by the docs.
- Fiscal item data for GNK must be preserved when the payment flow requires itemized fiscalization.

## Decision Flow

### 1. Classify the requested Global Pay surface first

- If the user wants a hosted redirect flow where Global Pay collects card details, choose Checkout Service.
- If the user wants first-party card entry, reusable card tokens, one-click flows, or explicit payment revert control, choose Cards Service plus Payments Service Public.
- If the user wants supplier onboarding or credential UX, treat it as vault-backed config, not customer checkout.

### 2. Decide whether the work is docs-only or credentialed

- `docs-only`: implement types, contracts, routes, config, guards, and state handling from docs, but keep unvalidated assumptions explicit.
- `credentialed`: run sandbox auth, payment, expiry, callback, and status verification before calling the flow complete.

If the team does not yet have live or sandbox credentials, prefer docs-only implementation plus explicit follow-up checkpoints.

### 3. Choose the environment explicitly

- Checkout dev: `https://checkout-api-dev.globalpay.uz/checkout`
- Checkout staging: `https://checkout-api-staging.globalpay.uz/checkout`
- Checkout production: `https://checkout-api.globalpay.uz/checkout`
- Gateway dev: `https://gateway-api-dev.globalpay.uz`
- Gateway staging: `https://gateway-api-staging.globalpay.uz`
- Gateway production: `https://api.globalpay.uz`

Never share one credential set across sandbox, staging, and production.

### 4. Choose the payment execution mode

- Hosted checkout: redirect the user to Global Pay and confirm the result later.
- Direct gateway: initialize and perform payment yourself using card tokens.

Prefer hosted checkout when the fastest safe rollout is required and the product does not need saved-card or custom card-entry behavior.

### 5. Choose the authoritative status verification path

- Checkout Service: verify payment via `GET /v1/payment/{id}` or `GET /v1/payment/servicetoken/{id}`.
- Direct gateway: verify payment via `GET /payments/v1/payment/{id}`.

For Visa and Mastercard, always re-check final status after any 3DS redirect.

## Implementation Procedure

### Step 1. Model Global Pay configuration safely

- Store Global Pay credentials in backend-only storage.
- Keep `service_id`, OAuth username, and OAuth password explicit in capability metadata and validation.
- Separate sandbox, staging, and production credentials.
- Return only safe summary fields to the frontend.
- If the repo uses generic gateway config fields, map them consistently instead of forcing a different storage model mid-rollout.

### Step 2. Implement the chosen payment surface

#### Hosted Checkout Service path

- Authenticate the merchant with `POST {checkout_base}/v1/merchant/auth`.
- Refresh tokens with `POST {checkout_base}/v1/merchant/auth/refresh` when needed.
- Create the payment token with `POST {checkout_base}/v1/user-service-tokens`.
- Redirect the customer to `userRedirectUrl` from the token response.
- Accept callback payloads only as hints.
- Confirm final status with `GET {checkout_base}/v1/payment/servicetoken/{id}` or `GET {checkout_base}/v1/payment/{id}` before marking the local order as paid.
- Handle `PAYMENT_EXPIRED` and token lifespan bounds explicitly.
- Deactivate stale or abandoned service tokens if the business flow needs active cleanup.

#### Direct gateway path with Cards Service and Payments Service Public

- Authenticate the merchant with `POST {gateway_base}/payments/v1/merchant/auth`.
- Refresh tokens with `POST {gateway_base}/payments/v1/merchant/auth/refresh`.
- If custom card capture is required, create a card with `POST {gateway_base}/cards/v1/card`.
- Confirm OTP when required with `POST {gateway_base}/cards/v1/card/confirm/{cardToken}`.
- Use `POST {gateway_base}/payments/v2/payment/init` by default.
- Use v1 init only when the supplier service requires additional `paymentFields` beyond the standard v2 structure.
- Perform the payment with `POST {gateway_base}/payments/v2/payment/perform`.
- If the perform response includes `securityCheckUrl`, redirect the customer for 3DS and re-verify final payment status afterward.
- Use `POST {gateway_base}/payments/v2/payment/revert` for full or partial revert flows when refunds are in scope.
- Confirm final state with `GET {gateway_base}/payments/v1/payment/{id}` before settling local business state.

### Step 3. Handle network-specific card behavior

- Uzcard and HUMO may require OTP during checkout or card confirmation.
- Visa and Mastercard may require cardholder name, CVV or card security code, client IP, and asynchronous 3DS handling.
- A successful redirect or 3DS screen completion does not guarantee final approval.

### Step 4. Preserve fiscalization and merchant traceability

- Populate payment items accurately when the supplier flow requires GNK or fiscal data.
- Preserve item fields like title, price, count, code, units, VAT percent, and package code when the docs require them.
- Keep Global Pay `externalId` traceable to the repo's internal order, attempt, or payment session identifier.
- Persist provider payment IDs, service token IDs, and settlement timestamps needed for reconciliation.
- Keep returned GNK details, receipt IDs, fiscal signs, QR code links, and failure reasons available for support and ledger work.

### Step 5. Build docs-only safely when credentials are missing

- Treat documentation as the current source of truth, but label unvalidated assumptions in code comments, task notes, or follow-up items.
- Implement feature flags, disabled production toggles, or environment guards so docs-only code cannot accidentally act like a live rollout.
- Do not fabricate undocumented callback signatures, retry guarantees, or hidden auth rules.
- Add explicit follow-up checkpoints for sandbox auth, callback registration, production allowlists, and refund certification.

### Step 6. Validate with real credentials later

- Confirm auth and token refresh in sandbox.
- Run the full create-payment and completion path in the chosen integration mode.
- Validate expiry handling for abandoned or expired tokens.
- Validate callback and polling consistency.
- Validate Visa and Mastercard completion after 3DS.
- Validate GNK data presence for successful transactions where applicable.
- Validate revert flows if refunds are part of scope.

## Completion Checks

- Hosted checkout and direct gateway flows are not accidentally mixed.
- Secrets remain backend-only and environment-separated.
- Supplier onboarding logic is not conflated with customer payment execution.
- Amounts use the documented smallest currency unit.
- Redirect, callback, popup, or app-return signals do not settle payment without server verification.
- Expired, failed, pending, and approved states are all modeled explicitly.
- Visa and Mastercard asynchronous completion is rechecked through status endpoints.
- Provider IDs and timestamps needed for reconciliation are persisted.
- Docs-only limitations are documented if no live credential testing has occurred.

## Known Unknowns To Confirm During Credentialed Testing

- whether the merchant tenant requires any auth headers or registration details beyond what is shown in the docs
- whether Cards Service auth behavior differs in the actual merchant environment
- which callback URLs, IP allowlists, or merchant-side registration steps are required in production
- whether any supplier verticals require mandatory additional payment or fiscal fields beyond the base examples
- whether partial revert and receipt data must be mirrored into treasury or ledger flows differently in this repo

## References Derived From The Provided Docs

- Checkout Service v2.0.0: merchant auth, refresh, user service token create and deactivate, payment status by service token or payment ID
- Cards Service v2.0.0: create card and confirm card
- Payments Service Public v2.3.0: merchant auth, refresh, payment init, perform, get, and revert
- WooCommerce and OpenCart guides: confirm credential naming, sandbox and production separation, and GNK item requirements for catalog-backed payments