# Identifier Hard Cutover Rollback Notes (2026-04-30)

## Scope
This release removes legacy desktop and iOS identifier bridges and finalizes canonical identifiers.

Changed surfaces:
- Desktop Tauri identifiers:
  - admin: `com.pegasus.admin`
  - factory: `com.pegasus.factory`
  - warehouse: `com.pegasus.warehouse`
  - retailer: `com.pegasus.retailer` (already set prior)
- Desktop keyring bridge removal:
  - removed legacy service fallback reads/writes/deletes in all desktop `security.rs` command modules
- iOS keychain bridge removal:
  - removed legacy service fallback migration paths in driver, retailer, factory, warehouse, payload auth/token stores
- iOS API host bridge removal:
  - removed pre-hard-cut legacy API host fallback in driver, retailer, factory, warehouse, payload API clients
- iOS bundle ID hard cutover:
  - factory app moved to `com.pegasus.factory`

## Release Gates
1. Gate A (pre-prod canary, same day)
- Validate desktop sign-in, token refresh, and logout on clean install and upgraded install.
- Validate iOS sign-in, token refresh, and logout on clean install and upgraded install.
- Confirm no crash regressions in auth/bootstrap paths.

2. Gate B (production canary, 5-10%)
- Watch auth failure rate, token-missing/session-expired rate, and login retry rate for 2-4 hours.
- Verify no abnormal increase in forced re-auth loops.

3. Gate C (full rollout)
- Expand only after Gate B metrics remain within baseline tolerance.
- Keep rollback patch ready for immediate release if auth metrics degrade.

## Rollback Trigger Conditions
Rollback immediately if any of the following persists > 15 minutes:
- Auth failure rate > 2x baseline
- Token-missing/session-expired events > 2x baseline
- Login success rate drops below baseline SLO
- Support reports widespread post-upgrade forced logout loops

## Rollback Plan (Patch Release)
1. Restore previous identifiers in affected desktop/iOS project configs.
2. Re-enable legacy keychain/keyring fallback reads in auth/token stores.
3. Re-enable legacy key cleanup logic on logout.
4. Re-enable legacy API host fallback path in iOS API clients.
5. Ship patch release with rollback changes only (no unrelated edits).
6. Re-run canary gates before wider redeploy.

## Data Recovery Notes
- This cutover intentionally stops reading legacy keychain/keyring namespaces.
- Rollback re-enables legacy reads and allows prior credentials to resolve again.
- No server-side data migration is required for rollback.

## Post-Rollout Verification Checklist
- Desktop:
  - login -> authenticated session persisted -> restart app -> still authenticated
  - logout clears canonical tokens
- iOS:
  - login -> authenticated session persisted -> restart app -> still authenticated
  - token refresh succeeds without forced re-login
  - logout clears canonical tokens
- API contracts and route flows remain unchanged.
