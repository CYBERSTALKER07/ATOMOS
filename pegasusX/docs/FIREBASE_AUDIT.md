# Firebase + App Hosting — code audit

**Date:** 2026-08-18  
**Tree:** `pegasusX/`  
**Kind:** point-in-time. Firebase is an **OTP + FCM sidecar**. Spanner + pegasus JWT stay SoT. **Not** firebase init/deploy. **Not** Firestore.

**Related:** [`SURFACE_AUDITS.md`](./SURFACE_AUDITS.md) · [`DEVOPS_CICD_AUDIT.md`](./DEVOPS_CICD_AUDIT.md)

---

## 0. Verdict

```
VERDICT: PARTIAL (Auth + FCM). Hosting / App Hosting / Firestore / SQL Connect / Crashlytics / Storage / RTDB / Functions / Remote Config / App Check / Analytics: GONE
INTEGRATE Hosting/App Hosting: NO
INTEGRATE Firestore / Data Connect: NO (competing SoT)
NEXT: staging FCM_ALLOW_NOOP (do not flip without credentials). Expo tokens **rejected** at DeviceTokens. Payload-terminal iOS Expo still has no FCM (native payload-app-ios does). Not firebase deploy. Not Layer B.
```

---

## 1. What exists

`pegasusX/firebase.json` — Auth emulator **9099** + UI **4000** only. **No** `"hosting"` key. No `.firebaserc`. No `apphosting.yaml`.

| Product | Verdict | Evidence |
|---------|---------|----------|
| **Auth** | **PARTIAL** | Flag default **false** `bootstrap/bootstrap.go:327`. K8s base `FIREBASE_AUTH_ENABLED=true` (`configmap.yaml:40`). Staging/EU overlay **false**. Verifier composed at construct (`bootstrap/firebase.go` + `NewService`). Phone OTP → verify ID → **issue pegasus HS256 JWT**. Session SoT `SessionAuth` `auth/jwt.go:236-256`, mount `main.go:129`. |
| **FCM** | **PARTIAL** | Admin Messaging `notifications/fcm.go`. `InitFCM` bootstrap. Device-token persist `POST /v1/user/device-token` (`platformroutes/routes.go:37`) — JWT via `SessionAuth` (`main.go:129`), handler 401 without claims. **Android retailer/driver/payload/factory/warehouse/supplier** POST FCM tokens (`firebase-messaging-ktx` + `FirebaseMessagingService`). **iOS retailer/driver/warehouse/payload/factory/supplier** POST FCM registration tokens (FirebaseMessaging SPM + `Messaging.apnsToken`). Staging `FCM_ALLOW_NOOP=true`. |
| **Hosting / App Hosting** | **GONE** | No hosting key; Class A is GCE Ingress → `backend-go` (`infra/k8s/ingress/ingress.yaml`). Next `TAURI_BUILD=1` → `output: "export"`. Only supplier-portal has a Dockerfile. |
| **Firestore / SQL Connect / RTDB / Functions / Remote Config / App Check / Analytics / Perf / Crashlytics / Storage** | **GONE** | Zero product SDK. Media is GCS signed PUT `storage/gcs.go` + `GET /v1/media/upload-ticket`. google-services `storage_bucket` unused. |

Admin SDK (`firebase.google.com/go/v4`): custom tokens + FCM only.

---

## 2. Auth composition (re-verify)

`newLoginFirebaseVerifier` (`bootstrap/firebase.go:12-27`) builds the ID-token verifier when `FIREBASE_AUTH_ENABLED` and `FIREBASE_PROJECT_ID` are set; otherwise **nil**. Bootstrap **does** pass that instance into retailer / factory / payload / driver / warehouse `NewService` and `App.FirebaseVerifier` (`bootstrap/bootstrap.go:657`, `:673`, `:1033`, `:1051`, `:1205`, `:1343`, `:1883`). `FIREBASE_AUTH_ENABLED` is **login verifier construct only** — `main.go` does not remount a Firebase session middleware.

`id_token` on login (and factory/warehouse register) with a **nil** verifier is **503** `firebase_login_unavailable` — not a password/PIN fall-through (`retailer/auth_login.go:38-40`, same pattern driver/payload/factory/warehouse login; register `factory/auth_register.go:61-64`, `warehouse/auth_register.go:61-64`). Password/PIN login still works when the flag is off.

If a client sends Firebase ID as `Authorization`, `FirebaseAuth` is a **pass-through** and does **not** attach claims (`auth/firebase.go:192-203`). HTTP session SoT remains pegasus JWT (`SessionAuth` `auth/jwt.go:236-256`). OTP still verifies Firebase ID as login/register JSON `id_token` only.

**Retailer Android HTTP (this session):** JWT only.

```33:47:pegasusX/apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/data/local/TokenManager.kt
    /** Session JWT for HTTP/WS Bearer. Firebase ID is OTP `id_token` body only. */
    fun getPreferredToken(): String? = httpAuthorizationToken(getToken(), null)
    ...
        fun httpAuthorizationToken(sessionJwt: String?, firebaseIdToken: String?): String? {
            val jwt = sessionJwt?.trim().orEmpty()
            if (jwt.isNotEmpty()) {
                return jwt
            }
            return null
        }
```

Driver Android interceptor uses `TokenHolder.httpAuthorizationToken(TokenHolder.token, null)` — `NetworkModule.kt:56`. Unused `firebaseIdToken` store removed. Factory portal `apiFetch` uses JWT (`httpAuthorizationToken(await getFactoryToken())`) — `factory-portal/lib/auth.ts:150-151`. OTP still sends Firebase ID as login **body** `id_token` only.

iOS retailer API uses pegasus JWT (`AuthManager.swift`). Payload Android HTTP uses pegasus JWT. Warehouse/supplier portals already used JWT.

---

## 3. Project ID split

| Layer | ID |
|-------|-----|
| Android `google-services.json` / iOS plists | `pegasus-503013` |
| Base ConfigMap `FIREBASE_PROJECT_ID` | `pegasus-503013` (`infra/k8s/backend-go/configmap.yaml`) |
| ExternalSecret | **no** `FIREBASE_PROJECT_ID` key |
| GSM SecretStore project | `pegasus-503013` |
| Compose SSMR/sandbox | `demo-pegasus` (emulator) |

`SPANNER_PROJECT` in the same ConfigMap is still `pegasusx-prod` — GCP Spanner project, not Firebase. Do not conflate.

---

## 4. App Hosting / Hosting — how it should **not** be done

Firebase Hosting / App Hosting skills assume `firebase.json` hosting + `apphosting.yaml` for Next SSR. This product:

- Portals: Tauri static export **or** local Next rewrite to Auth emulator (`supplier-portal/next.config.mjs:17-33`).
- API: GKE Ingress `/v1` only (does not publish `/metrics` on `/`).
- Marketing-site: Next, no k8s, no Hosting public dir.

**INTEGRATE: NO.** Do not move Class A API to Hosting. Do not put Spanner traffic behind App Hosting. Marketing static export to Hosting is a **later ops** option only if `output: "export"` — not App Hosting for portals, not this slice.

---

## 5. Best practices vs skills

| Skill | vs code |
|-------|---------|
| firebase-basics CLI | `npx firebase-tools` exists locally; no `.firebaserc`; do not `firebase init` |
| firebase-auth-basics | OTP bootstrap OK; **do not** make Firebase session SoT |
| firebase-security-rules-auditor | N/A — no Firestore/RTDB rules |
| App Hosting | **Do not add** |

---

## 6. Ranked next (code)

1. **Closed this session:** HTTP Bearer is pegasus JWT. Factory portal no longer prefers Firebase ID. Driver unused `firebaseIdToken` store removed. Retailer no longer persists Firebase ID for HTTP.
2. **Closed this session:** `FirebaseAuth` does not attach claims from `Authorization` (`auth/firebase.go:192-203`). Login OTP `id_token` body stays.
3. Staging / EU overlay `FCM_ALLOW_NOOP=true` — do not flip without live Firebase credentials (not Layer B keys from this slice).
4. **Closed this session:** unused no-op `FirebaseAuth` route wraps removed (`ProtectMutations` `auth/route_guard.go:11-22`; factory/warehouse/payload/returns/telemetry/WS always RequireRole / JWT). `FirebaseAuth()` kept + tests. Flag still builds login verifier (`bootstrap/firebase.go:12-13`).
5. **Closed:** `FirebaseVerifier` is composed at construct; nil verifier + `id_token` → 503 `firebase_login_unavailable`.
6. **Closed this session:** native FCM leftover row (iOS + Android role apps).
7. **Closed this session:** unused portal `getFirebaseIdToken` exports removed (factory/warehouse/supplier/retailer-desktop `lib/firebase.ts`). OTP still `verifyPhoneOtp`.
8. **Closed this session:** unused portal `exchangeCustomToken` removed (same four `lib/firebase.ts`). Login persists JWT only (`factory-portal/app/auth/login/page.tsx:71`). Native Android/iOS still exchange custom tokens for the Firebase SDK, not HTTP.
9. **Closed this session:** payload-terminal no longer stores/POSTs login `firebase_token` as an FCM device token (`authSession.ts:19-28`, `pushRegistration.ts:29-31`). Mint for native payload apps stays (`payload/auth_login.go:215`).
10. **Closed this session:** Expo push tokens are not stored as FCM. `POST /v1/user/device-token` returns **422** `not_fcm_registration_token` for `platform=expo` or `ExponentPushToken[` — `platform/handlers.go:201-203`, `IsFCMRegistrationToken` `:248-261`. ListTokens skips leftover Expo-shaped rows — `platform/repository.go:207-209`. Payload-terminal uses `getDevicePushTokenAsync` and POSTs **Android FCM only** — `fcmDeviceToken.ts:12-25`, `pushRegistration.ts:41-45`. iOS Expo APNs is skipped (native `payload-app-ios` POSTs FCM). `app.json` has no `googleServicesFile`; Android Expo FCM POST is best-effort, not a live FCM claim.

Residual: staging / EU overlay `FCM_ALLOW_NOOP=true` — do not flip without live Firebase credentials. Payload-terminal iOS Expo has no Firebase Messaging. Not Firestore. Not Hosting. Not Layer B keys.
