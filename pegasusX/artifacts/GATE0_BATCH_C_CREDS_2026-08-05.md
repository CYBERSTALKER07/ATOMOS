# Gate-0 Batch C — credentials / Firebase / Maps / Global Pay

**Project:** `pegasus-503013`  
**Date:** 2026-08-05

## Secret Manager (SSMR) — present

| Secret | Versions | Notes |
|--------|----------|--------|
| `pegasusx-ssmr-google-maps-api-key` | 2 enabled | Length ~39 bytes (key-shaped) |
| `pegasusx-ssmr-jwt-secret` | 1 enabled | Present |
| `pegasusx-ssmr-global-pay-webhook-secret` | 1 enabled | Present |
| `pegasusx-ssmr-global-pay-password` | 2 enabled | Length 32 bytes — **treat as candidate real merchant password; card SUCCESS still needs live e2e** |
| `pegasusx-ssmr-global-pay-username` | present | Listed |
| `pegasusx-ssmr-global-pay-service-id` | present | Listed |

ExternalSecret wiring already maps these into `backend-go-secrets` (SSMR overlay).

## Firebase client configs

| Platform | Status |
|----------|--------|
| Android `google-services.json` | Present for all 6 apps |
| iOS `GoogleService-Info.plist` | Present for all 6 apps |
| iOS `aps-environment` entitlements | **Added** (development) for all 6 apps + XcodeGen `CODE_SIGN_ENTITLEMENTS` where `project.yml` exists |
| Driver iOS background | `location` + `remote-notification` added to `UIBackgroundModes` |
| Driver Android FCM | **`DriverFirebaseMessagingService`** registered in manifest |
| Retailer / Payload Android FCM | Already declared |
| Android debug SHA-1 | **Registered** on all 6 Firebase Android apps (`DD13…6759` from `~/.android/debug.keystore`) |
| Real Phone SMS proof | Still **owner ops** — send one OTP on device/emulator after Blaze billing confirmed; configs + SHA ≠ live SMS until tested |
| Identity Toolkit API | Enabled (`identitytoolkit.googleapis.com`) |

## Maps

GSM key present and referenced by ExternalSecret (`google-maps-api-key`). No code change required for Gate-0.

## Global Pay card SUCCESS

Credentials exist in GSM (username / password / service-id / webhook secret). Remaining proof (not automatable without merchant portal + live card):

1. Confirm merchant password is the live staging password (not a stub).
2. Register webhook URL in Global Pay portal → SSMR ingress.
3. Run SSMR e2e until `PX_E2E_PAYMENT_CARD_SUCCESS_OK` (cash fallback marker already green).

## Artifact Registry (image pins)

| Image | Digest / tag used in prod overlay |
|-------|-----------------------------------|
| backend-go | `sha256:a5b31a6…` (`ssmr-substance-gate-a66868b8-084112`) |
| ai-worker | `sha256:9f05b7d8…` (`ssmr-4a0796fd-glibc`) |
| optimizer-core | **Not published** — prod overlay scales replicas to 0 until a real image lands. SoT: [`docs/OPTIMIZER_AND_ROUTING_RUNTIME.md`](../docs/OPTIMIZER_AND_ROUTING_RUNTIME.md) |

Repo: `asia-south1-docker.pkg.dev/pegasus-503013/pegasusx-ssmr-images`
