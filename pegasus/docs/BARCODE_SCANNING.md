# Barcode / QR Code Scanning — Ecosystem Policy

**Last audited:** 2026-04-15
**Status:** Barcode scanning is **out of scope** for new surfaces. Existing production flows that use QR/barcode scanning are preserved; no new scanner surfaces should be built without explicit product approval.

---

## Why this doc exists

An ecosystem-wide audit was requested to confirm barcode/QR scanning is not silently spreading to surfaces that do not need it. This doc records:

1. Which apps **actively use** barcode/QR scanning today (and why).
2. Which apps had **scaffolding** (deps, plist keys, README references) that has been **removed**.
3. Which apps are **confirmed clean**.
4. How to **reinstate** scanning if a future product decision reverses this.

---

## 1. Active production scanners (PRESERVED)

These are wired into live delivery-verification flows. Removing them would break the flow. They remain in place.

| App | File(s) | Purpose |
|-----|---------|---------|
| `apps/driver-app-android` | `ui/screens/scanner/ScannerScreen.kt`, `ui/screens/scanner/ScannerViewModel.kt` | Driver QR code scan for retailer delivery confirmation. Wired via `DriverRoutes.SCANNER` and the Scan QR CTA on `HomeScreen`. Uses CameraX + ML Kit `barcode-scanning:17.3.0`. |
| `apps/driverappios` | `Views/QRScannerView.swift` (+ `QRCameraPreview`), `ViewModels/ScannerViewModel.swift` | Driver QR scan → `/v1/driver/validate-qr`. Used inside `FleetMapView` delivery sheet. AVFoundation-based. |
| `apps/driverappios` | `Views/OfflineVerifierView.swift`, `ViewModels/OfflineVerifierViewModel.swift` (`handleBarcodeScan(_:)`) | Offline manifest verifier — reads retailer token via camera when offline, queued via `OfflineDelivery` SwiftData model. |

**Do not rip these out** without a product-side decision and a replacement flow (manual token entry, NFC tap, etc.). They are the canonical delivery-proof mechanism for the DRIVER role.

---

## 2. Removed scaffolding (no production impact)

The following were speculative stubs / aspirational deps. They never served real traffic and have been removed.

| App | What was removed | Why it was safe to remove |
|-----|------------------|---------------------------|
| `apps/payload-app-android/app/build.gradle.kts` | `androidx.camera:camera-{core,camera2,lifecycle,view}:1.4.1` + `com.google.mlkit:barcode-scanning:17.3.0` | No Kotlin file in the app referenced these deps. Phase 8 "barcode scan" was aspirational. |
| `apps/retailer-app-android/app/build.gradle.kts` | `androidx.camera:camera-{camera2,lifecycle,view}:1.4.1` + `com.google.mlkit:barcode-scanning:17.3.0` | No Kotlin file referenced these deps. |
| `apps/payload-app-ios/project.yml` | `INFOPLIST_KEY_NSCameraUsageDescription: "Scan SKU barcodes during loading"` | No SwiftUI view used VisionKit / DataScanner / AVFoundation. App will not prompt for camera permission. |
| `apps/payload-app-android/README.md` | Phase 8 "Barcode scan + tablet polish" row → renamed to "Tablet polish (barcode deferred)"; removed "CameraX 1.4.1 + ML Kit barcode 17.3 (Phase 8)" architecture bullet | Documentation drift — removed feature was never built. |
| `apps/payload-app-ios/README.md` | Phase 8 "VisionKit barcode scan + tablet polish" row → renamed "Tablet polish"; removed "VisionKit DataScannerViewController (Phase 8)" architecture bullet | Same as above. |

---

## 3. Confirmed clean (no barcode/scanner anywhere)

Grepped and verified at audit time:

- `apps/factory-app-android/` — no `camera|barcode|mlkit|scanner|QrCode` matches.
- `apps/factory-app-ios/` — no `Camera|Barcode|VisionKit|DataScanner|QRScanner` matches.
- `apps/factory-portal/` — only minified vendor chunks (Firebase TOTP QR URL generator, `@nodelib/fs.scandir`, AppKit/Foundation generated headers). No product code path.
- `apps/payload-terminal/` (Expo) — no `expo-camera` / `expo-barcode` / `vision-camera` in `package.json`.
- `apps/warehouse-app-android/`, `apps/warehouse-app-ios/`, `apps/warehouse-portal/` — no matches.
- `apps/retailer-app-ios/`, `apps/retailer-app-desktop/` — no matches.

**Factory admin (the unified Factory Admin + Payloader surface)** was explicitly re-verified per request and is 100% clean on both Android and iOS.

---

## 4. How to reinstate scanning on a specific surface

If product reverses this decision for an app, follow the steps for that platform.

### Android (e.g., `payload-app-android`, `retailer-app-android`)

1. Add to `app/build.gradle.kts` in the `dependencies` block:
   ```kotlin
   implementation("androidx.camera:camera-core:1.4.1")
   implementation("androidx.camera:camera-camera2:1.4.1")
   implementation("androidx.camera:camera-lifecycle:1.4.1")
   implementation("androidx.camera:camera-view:1.4.1")
   implementation("com.google.mlkit:barcode-scanning:17.3.0")
   ```
2. Add to `app/src/main/AndroidManifest.xml`:
   ```xml
   <uses-permission android:name="android.permission.CAMERA" />
   <uses-feature android:name="android.hardware.camera" android:required="false" />
   ```
3. Request camera permission at runtime (`Manifest.permission.CAMERA`) before launching the scanner screen.
4. Copy the scanner screen template from `apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/scanner/` (ScannerScreen.kt + ScannerViewModel.kt) and adapt package paths.
5. Wire the route in the app's `*Navigation.kt` and add the CTA on the home screen.
6. `./gradlew :app:assembleDebug` to verify.

### iOS (e.g., `payload-app-ios`)

1. In `project.yml` under `settings.base`, add:
   ```yaml
   INFOPLIST_KEY_NSCameraUsageDescription: "<user-facing reason>"
   ```
2. Run `xcodegen generate` from the app directory.
3. Copy `apps/driverappios/driverappios/Views/QRScannerView.swift` (includes `QRCameraPreview` UIViewRepresentable) into the target's `Views/` folder.
4. Copy `apps/driverappios/driverappios/ViewModels/ScannerViewModel.swift` (AVFoundation permission check + handleScan).
5. For VisionKit DataScanner (iOS 16+), prefer `DataScannerViewController` via a UIViewControllerRepresentable wrapper — simpler than AVFoundation but iOS 16+ only and requires `com.apple.developer.visionkit` entitlement indirectly handled by Xcode.
6. Build via Xcode or `xcodebuild -scheme <app> -destination 'generic/platform=iOS'`.

### Expo (`payload-terminal`)

1. `npx expo install expo-camera`
2. In `app.json` add `"plugins": [["expo-camera", { "cameraPermission": "<user-facing reason>" }]]`.
3. Use the `<CameraView>` component with `barcodeScannerSettings={{ barcodeTypes: ['qr', 'ean13', ...] }}` and an `onBarcodeScanned` handler.
4. Rebuild the dev client (`npx expo prebuild --clean && npx expo run:ios`).

---

## 5. Policy for new work

- **Do not** add `mlkit:barcode-scanning`, `androidx.camera:*`, `AVCaptureMetadataOutput`, `DataScannerViewController`, `expo-camera`, or any QR/barcode scanning dependency to a surface without updating this doc AND getting product sign-off.
- **Do not** re-enable Phase 8 in any payload-app README as "barcode scan". Phase 8 is tablet polish only until further notice.
- The driver apps keep their scanners — do not extend QR scanning to new roles opportunistically.

---

## 6. Contact / decision log

| Date | Decision | Owner |
|------|----------|-------|
| 2026-04-15 | Remove unused barcode scaffolding from payload-app (Android/iOS), retailer-app-android; preserve active driver-app scanners. | Boss (product) |
