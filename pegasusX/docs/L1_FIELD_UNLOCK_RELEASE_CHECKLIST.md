# L1 Field Unlock — Release Checklist

## Global Pay

- [ ] Real staging `service_id` / username / **password** in GSM (`pegasusx-ssmr-global-pay-*`)
- [ ] Webhook registered: `https://api-ssmr.pegasusx.app/v1/webhooks/global-pay`
- [ ] ESO sync + API/worker restart
- [ ] Smokecheck prints `PX_E2E_PAYMENT_CARD_SUCCESS_OK` (not only `PX_E2E_PAYMENT_CASH_FALLBACK_OK`)
- [ ] `ValidateProductionProfile` does not trip on `doc-*` stubs

## Firebase OTP

- [ ] Blaze + Phone Auth enabled
- [ ] Android SHA-1/SHA-256 registered (debug + release) per `applicationId`
- [ ] APNs key for iOS if FCM required
- [ ] `FIREBASE_AUTH_ENABLED=true` on SSMR with SA mounted
- [ ] **Release builds:** `firebase.auth.emulator` must stay `false` in `local.properties` (debug-only)
- [ ] iOS release: no `FIREBASE_AUTH_EMULATOR_HOST` in scheme env
- [ ] OTP login proven on one Android + one iOS app against `https://api-ssmr.pegasusx.app`

## Do not

- Flip `PEGASUSX_ENV=production` until card SUCCESS + OTP green
- Ship release APK/IPA with auth emulator enabled
