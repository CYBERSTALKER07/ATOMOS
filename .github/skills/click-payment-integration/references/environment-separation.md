# Environment Separation Guidance

## Recommendation

Yes, CLICK integration is implementable with clean testing and production separation.

The correct split is not two different implementations. The correct split is one implementation with two isolated environment configurations and an explicit environment dimension in backend state.

For this workspace, the expected rollout order is:

1. testing first
2. emulator validation and report generation
3. production connection later

## Required Separation

Keep these values separate for `testing` and `production`:

- `merchant_id`
- `service_id`
- `merchant_user_id`
- `secret_key`
- callback or return URLs
- monitoring and alerting labels
- emulator readiness state and report status

## Backend Modeling Guidance

Preferred approach:

- add an explicit environment field like `TEST` or `PRODUCTION`
- scope vault records by supplier, gateway, and environment
- scope onboarding or payment session records by environment when they can touch external systems
- return environment metadata to operational UIs so the active path is visible

Avoid:

- overloading one config record with both test and prod values
- switching environment with only a frontend flag
- sharing a secret between test and production paths

## Runtime Behavior

### Testing

- use test credentials only
- route operational verification through the CLICK emulator workflow
- require passing scenarios and report generation before enabling production cutover
- do not wire production credentials into the first implementation pass

### Production

- use production credentials only
- require public callback URLs and production monitoring
- keep rollback steps ready so a bad cutover can be disabled quickly
- treat production enablement as a separate deployment milestone after testing succeeds

## UI and Ops Guidance

Operational surfaces should show:

- current environment
- whether credentials are configured for that environment
- last successful test or verification timestamp
- whether production is enabled

## Repo-Specific Note

In this workspace, supplier credential management and customer checkout are different systems.

- Supplier config belongs in vault-backed payment settings.
- Customer checkout belongs in payment execution code and app launch flows.

Do not use testing-versus-production separation as a reason to merge those two systems.
