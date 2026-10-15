# Pegasus Executive Console (Native Android)

A native Android application built in **Kotlin** with **Jetpack Compose**, implementing the **Pegasus Swiss CAD Dark Design System**.

> **Operational Scope**: Dedicated handheld console for the administrator to **review and answer incoming client requests** submitted via the Pegasus website. Clients submit requests on the website only; this native app serves as the review, response, and CRM workflow hub.

---

## Key Operator Capabilities
1. **Live Review Queue**:
   - Inspect client contact info, operational role, company, and fleet network scope.
   - Review incoming messages and operational requirements.
   - Horizontal status filter chips (`ALL`, `NEW`, `IN REVIEW`, `CONTACTED`, `QUALIFIED`, `CLOSED`).
2. **In-App Answer Studio**:
   - Pre-formatted response templates (`DEMO INVITE`, `PILOT TERMS`).
   - Direct email reply transmitter connecting to `/api/admin/reply`.
   - Native email client intent launcher.
   - Automatic status transition to `CONTACTED` upon sending.
3. **Workflow & Actions**:
   - One-tap status advancement.
   - Delete record action for spam removal.

---

## Swiss CAD Design Tokens
- **Background**: `#000000` (Pitch Black)
- **Panels**: `#0D0D0E`, Cards `#141417`
- **Accents**: Electric Cyan (`#A9EBF9`), Volt Lime (`#FBFF63`), Emerald (`#8DDC96`), Alert Tangerine (`#FE5934`)
- **Shapes**: Zero-radius sharp corners (`RoundedCornerShape(0.dp)`)

---

## Build & Run

### Terminal
```bash
cd apps/enterprise-app-android
./gradlew assembleDebug
```
The output APK is at:
`app/build/outputs/apk/debug/app-debug.apk`
