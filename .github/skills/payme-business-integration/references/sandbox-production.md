# Sandbox and Production Guidance

## Environment Split

Keep sandbox and production fully separate.

Separate at least these values:

- Merchant API login and password
- Subscribe API cash register id and password key
- checkout hosts
- callback URLs
- fiscalization settings
- monitoring labels and alert routes

Do not switch environments using only a frontend flag.

## Sandbox

Use only sandbox hosts:

- `https://test.paycom.uz`
- `https://checkout.test.paycom.uz/api`

Expected testing inputs:

- test cards from Payme docs
- fixed SMS code `666666`

Recommended sandbox validation:

1. create and cancel an unconfirmed transaction
2. create, confirm, and cancel a confirmed transaction
3. verify retry safety for `CreateTransaction`, `PerformTransaction`, and `CancelTransaction`
4. verify `GetStatement` output against stored local records
5. verify token and receipt state handling if Subscribe API is used

## Production

Use only production hosts:

- `https://paycom.uz`
- `https://checkout.paycom.uz/api`

Production readiness checks:

- credentials are production-specific
- statement reconciliation is working
- merchant and subscribe auth are both configured correctly for the modes you use
- rollback path is prepared

## Repo-Specific Modeling Note

This repo currently has a shared supplier payment config model for CLICK and PAYME.

Official Payme Business integration may require extending storage to preserve distinctions like:

- Merchant API versus Subscribe API mode
- sandbox versus production environment
- cash register id versus generic merchant id
- login/password versus generic secret key

Do not hide those distinctions behind overloaded generic fields if they affect runtime correctness.


# Universal Agent & Engineering Guidelines
When developing, designing, or planning, always ensure to account for:
- Gaps, edge cases, and comprehensive feature validation.
- Best practices and optimized integration for Kafka, Redis, Backend, Optimizers, AI, and UI.
- Real-time concepts including WebSockets, webhooks, and their native app equivalents.
- Thorough business logic for features, understanding how the role, app, and ecosystem work together, and engagements with other roles and features.
- Best practices for backend, frontend, and infrastructure libraries/packages. Always prefer existing, high-quality open-source libraries and packages that best suit our features before creating our own.
- Optimal UI infrastructure and UX patterns (e.g., optimal screen positioning for drivers during an active route), applying the same high standards to backend and cloud architecture.
- ALWAYS search the web to find open-source code, libraries, packages, math, algorithms, approaches, and best practices for anything we are doing. If none exist, then create our own.
- Always search the web to get the correct logic, and incorporate edge cases, business logic for features, operations (ops), workflow, data consistency, finance, and AI into everything we do.
