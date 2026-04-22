---
name: click-payment-integration
description: 'Implement CLICK payment integration in this monorepo. Use for CLICK-only web redirect checkout, checkout.js overlay flows, Merchant API server calls, Android/iOS deeplinks, CLICK test emulator setup, test-first rollout, later production cutover, and keeping testing and production credentials separate. Also use when reviewing code so customer payment flows are not confused with supplier gateway onboarding.'
argument-hint: 'Describe the target: web checkout, merchant API, mobile deeplink, supplier config, testing, or production cutover'
---

# CLICK Payment Integration

## What This Skill Does

Use this skill to implement or review CLICK-only integration across web, backend, and mobile surfaces in this workspace.

This skill is specifically designed to keep three concerns separate:

- Customer payment execution
- Supplier credential configuration
- Testing versus production environments

This skill also enforces the rollout order:

1. Build and validate the testing path first
2. Pass emulator scenarios and reporting
3. Connect real production APIs only after testing is complete

## When to Use

Activate this skill when the request involves any of the following:

- CLICK redirect checkout using `https://my.click.uz/services/pay`
- CLICK in-page checkout via `https://my.click.uz/pay/checkout.js`
- CLICK Merchant API calls to `https://api.click.uz/v2/merchant/`
- CLICK mobile deeplink handling for Android or iOS
- CLICK invoice creation, payment status checks, reversals, or card token flows
- CLICK emulator testing, Prepare/Complete endpoint validation, or production registration
- Separating CLICK testing and production credentials, callbacks, and rollout logic
- Reviewing whether code incorrectly mixes supplier onboarding with customer payment flows

Do not use this skill for unrelated payment gateways or for generic OAuth onboarding. CLICK merchant payment docs are not the same thing as supplier account-connect docs.

## Repo-Specific Guardrails

- Customer payment redirects belong to checkout execution, not supplier merchant onboarding.
- Supplier payment credential storage stays in the vault flow and secrets must remain backend-only.
- Manual supplier credential setup remains the source of truth unless CLICK publishes official supplier connect or merchant auth redirect docs.
- Never expose `secret_key` to the frontend, mobile clients, logs, or analytics payloads.
- Never reuse one credential set for both testing and production.

## Decision Flow

### 1. Classify the integration surface first

- If the task is customer checkout on web, use redirect checkout or `checkout.js`.
- If the task is server-to-server invoice or token operations, use Merchant API.
- If the task is Android or iOS payment launch, use deeplinks and app-link handling.
- If the task is supplier configuration inside the admin portal, treat it as credential management, not customer payment execution.

### 2. Decide the environment before writing code

- `testing`: emulator-first, test credentials, test callbacks, test report required
- `production`: production credentials, public URLs, monitoring, rollback plan

If the request says to support both, implement explicit environment separation instead of adding conditionals around a single shared config row.

Default policy for this workspace:

- Start with `testing`
- Keep production calls disabled or unconfigured until the testing workflow is green
- Treat production enablement as a separate cutover step, not part of the first implementation pass

### 3. Choose the payment mode

- Redirect checkout: simplest hosted payment page
- `checkout.js`: embedded overlay flow for web
- Merchant API invoice flow: push request to a phone number
- Card token flow: request, verify, charge, and optionally delete token
- Mobile deeplink: open CLICK app or browser fallback

## Implementation Procedure

### Step 1. Model configuration safely

- Store `merchant_id`, `service_id`, `merchant_user_id`, and `secret_key` separately from business data.
- Encrypt `secret_key` at rest.
- Keep testing and production credentials as separate records or separately scoped secrets.
- Make the environment explicit in types, storage, and API responses.

For this repo, prefer extending backend vault models rather than hardcoding CLICK values into frontend code.

### Step 2. Implement authentication correctly for Merchant API

- Use endpoint base `https://api.click.uz/v2/merchant/`.
- Build the `Auth` header as `merchant_user_id:digest:timestamp`.
- Compute `digest` as `sha1(timestamp + secret_key)`.
- Send `Accept`, `Content-Type`, and `Auth` headers on authenticated Merchant API requests.

See [CLICK API reference](./references/click-api.md).

### Step 3. Implement the requested surface

#### Web redirect checkout

- Build `https://my.click.uz/services/pay` with `service_id`, `merchant_id`, `amount`, and `transaction_param`.
- Include `return_url` only if the return surface is implemented and validated.
- Optionally include `merchant_user_id` and `card_type` when required.

#### Web in-page checkout

- Load `https://my.click.uz/pay/checkout.js`.
- Pass the documented parameters to `createPaymentRequest`.
- Treat callback `status` as a UI signal only and always re-check the authoritative backend payment state.

#### Merchant API

- Implement invoice creation first when you need server-controlled payment initiation.
- Add status lookup by `payment_id` and `merchant_trans_id`.
- Implement reversal only for eligible successful payments.
- Implement card token flows only if one-click payment is actually required.

#### Mobile

- Launch CLICK using the standard `https://my.click.uz/services/pay` URL.
- If `return_url` is used, bind a verified app link or deep link and refresh backend state on resume.
- Treat app resume as a signal to fetch payment status by your own transaction ID.

#### Supplier admin portal

- Keep supplier-facing config as manual credential setup unless official supplier-connect docs exist.
- Expose truthful capability metadata so the UI can distinguish `manual only` from `connect supported`.
- Do not pretend redirect payment checkout is supplier account onboarding.

### Step 4. Separate testing and production explicitly

- Use separate credentials and service registrations.
- Use separate callback URLs and test reports.
- Ensure test code cannot accidentally charge production accounts.
- Keep environment selection visible in backend config and operational tooling.

See [environment separation guidance](./references/environment-separation.md).

### Step 5. Validate end-to-end

- Run CLICK emulator scenarios for Prepare, Complete, cancellation, and timeout paths.
- Confirm all mandatory emulator scenarios pass.
- Generate the test report.
- Verify registration state at `merchant.click.uz`.
- Confirm production rollout uses production credentials only after test completion.

### Step 6. Cut over to production later

- Add production credentials only after the test workflow is stable.
- Switch to production callback URLs and monitoring only after test verification is complete.
- Keep rollback ready so production can be disabled without touching the tested code path.

## Completion Checks

- Secrets remain encrypted and backend-only.
- Testing and production credentials are fully separated.
- Testing is implemented and validated before any production API cutover.
- `transaction_param` is unique and traceable to your internal order or session.
- Redirect and return handling cannot create false-success UI states.
- Backend status checks are authoritative over popup or deeplink callbacks.
- Reversal and token flows are gated by explicit business need.
- Supplier onboarding logic is not conflated with CLICK customer payment execution.

## References

- [CLICK API reference](./references/click-api.md)
- [Environment separation guidance](./references/environment-separation.md)