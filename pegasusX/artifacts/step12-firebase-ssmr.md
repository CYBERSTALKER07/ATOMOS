# Step 12 — Firebase phone OTP + FCM (SSMR)

**Date:** 2026-07-27  
**Project:** `pegasus-503013`  
**Cluster:** `pegasusx-ssmr-gke` / ns `pegasusx-ssmr`  
**Image:** `backend-go:ssmr-s12-firebase`

## Verdict

| Item | Status |
|------|--------|
| Identity Toolkit (Auth) on project | **PASS** — config present |
| Phone sign-in enabled | **PASS** — + test number for smoke |
| `FIREBASE_AUTH_ENABLED` / `FIREBASE_PROJECT_ID` | **PASS** — ConfigMap + GSM |
| Firebase ID token verifier (API) | **PASS** — log: `firebase auth verifier initialized` |
| FCM client (API + worker) | **PASS** — `FCM client online` mode=`adc` |
| SA user keys | **Blocked by org policy** — use Workload Identity ADC instead |
| Full client OTP e2e (real SMS) | **Deferred** — needs app Firebase config + billing SMS |
| FCM device send e2e | **Deferred** — needs real device registration token |

## Runtime config

```
FIREBASE_AUTH_ENABLED=true
FIREBASE_PROJECT_ID=pegasus-503013
FIREBASE_CREDENTIALS_PATH=/var/secrets/firebase/service-account.json  # stub {} → ignored
```

WI runtime SA: `ssmr-backend@pegasus-503013.iam.gserviceaccount.com`  
Roles: `firebase.admin`, `firebaseauth.admin`, `firebasecloudmessaging.admin` (+ existing Spanner/Redis/GSM).

GSM:

- `pegasusx-ssmr-firebase-project-id` = `pegasus-503013`
- `pegasusx-ssmr-firebase-auth-enabled` = `true`

## Code change (ADC / Workload Identity)

Org policy `iam.disableServiceAccountKeyCreation` blocks JSON keys.  
FCM + Admin Auth now fall back to ADC when the credentials file is missing/stub `{}`:

- `apps/backend-go/notifications/fcm.go` — `InitFCM(path, projectID, ...)`
- `apps/backend-go/auth/firebase_admin.go` — ADC path for custom tokens
- `bootstrap` wires project ID into FCM init

## Phone OTP

Identity Toolkit phone provider:

```json
{
  "enabled": true,
  "testPhoneNumbers": {
    "+998901112233": "123456"
  }
}
```

**How OTP works in product**

1. Mobile/web client uses Firebase Auth SDK `signInWithPhoneNumber` (client-side).
2. Client receives Firebase **ID token**.
3. Backend verifies ID token via `FirebaseAuth` middleware / verifier (`FIREBASE_PROJECT_ID`).
4. Backend JWT / session issuance continues as before for app APIs.

**Test number smoke (no SMS cost)**  
Use `+998901112233` / code `123456` from a Firebase-configured client, then call backend with `Authorization: Bearer <idToken>`.

**Production SMS**  
Requires:

1. Firebase Console → Authentication → Phone (billing / reCAPTCHA / app check as needed).
2. Android `google-services.json` / iOS `GoogleService-Info.plist` for project `pegasus-503013`.
3. Optional: real SHA-1 / APNs for production clients.

## FCM

- Push path: `notifications.PushBridge` → `FCMClient.SendDataMessage`.
- Client registers device tokens via platform device-token APIs (Spanner `DeviceTokens` when present).
- Live send needs a real FCM registration token from a built app.

## Logs (verified)

```
FCM client online  project_id=pegasus-503013 mode=adc
firebase auth verifier initialized  project_id=pegasus-503013
```

API health after deploy still 200 via Ingress.

## Console note

`projects:addFirebase` API returned 403 for this account, but Identity Toolkit config already exists (callback host `pegasus-503013.firebaseapp.com`).  
If Console shows the project as not fully “Firebase-enabled”, open  
https://console.firebase.google.com/project/pegasus-503013  
once as Owner to complete product linkage / Blaze if SMS requires it.

## Next

- **Step 13** Maps API key + geocode  
- Or wire mobile apps with Firebase config + register a real device token and confirm push delivery
