# Barcode / QR Code Scanning — Ecosystem Policy

**Last audited:** 2026-06-15
**Status:** Barcode scanning is **approved for return-gate EAN on payload + warehouse roles only**. Driver QR delivery-proof scanners remain preserved. No other surfaces may add scanning without updating this doc and product sign-off.

---

## Why this doc exists

This doc records:

1. Which apps **actively use** barcode/QR scanning (and why).
2. **Approved return-gate EAN** surfaces (payload + warehouse).
3. **Per-platform stack** and client UX contract.
4. Which apps are **confirmed clean** (no scanner).
5. When a **commercial SDK** (e.g. Scandit) would make sense.

---

## 1. Driver delivery scanners (PRESERVED — QR only)

These are wired into live delivery-verification flows. Do not remove without a product decision and replacement flow.

| App | File(s) | Purpose |
|-----|---------|---------|
| `pegasusX/apps/driver-app-android` | `ui/screens/scanner/ScannerScreen.kt`, `ScannerViewModel.kt` | Driver QR scan for retailer delivery confirmation. CameraX + ML Kit `barcode-scanning:17.3.0`. |
| `pegasusX/apps/driver-app-ios` | `Views/QRScannerView.swift`, `ViewModels/ScannerViewModel.swift` | Driver QR scan → `/v1/driver/validate-qr`. AVFoundation-based. |
| `pegasusX/apps/driver-app-ios` | `Views/OfflineVerifierView.swift` | Offline manifest verifier — retailer token via camera when offline. |

**Symbology:** QR codes only (not EAN). Do not extend driver QR to other roles.

---

## 2. Approved return-gate EAN surfaces

Product-approved for **reverse logistics** (driver returns → warehouse gate). Symbologies: **EAN-8, EAN-13, GTIN-12/14** (backend normalizes via `returns.NormalizeBarcode`).

| App | Camera | Manual / wedge | API endpoints |
|-----|--------|----------------|---------------|
| `pegasusX/apps/payload-app-android` | Yes (shared `EanBarcodeScannerPreview`) | Yes | `POST /v1/returns/inbound/sessions`, `POST /v1/returns/inbound/scan`, `POST /v1/returns/inbound/confirm` |
| `pegasusX/apps/payload-app-ios` | Yes (`EANBarcodeScannerView`) | Yes | Same |
| `pegasusX/apps/payload-terminal` (Expo) | Optional (`expo-camera`) | Yes | Same |
| `pegasusX/apps/warehouse-app-android` | Yes (shared module) | Yes | Same |
| `pegasusX/apps/warehouse-app-ios` | Yes (`EANBarcodeScannerView`) | Yes | Same |
| `pegasusX/apps/warehouse-portal` | No | USB/BT keyboard wedge | Same |

**Supplier catalog EAN capture** (link retail barcode to product — not return-gate scan):

| App | Camera | Manual entry | API |
|-----|--------|--------------|-----|
| `pegasusX/apps/supplier-app-android` | Yes (shared `EanBarcodeScannerPreview`) | Yes | `POST /v1/catalog/products`, `PUT /v1/catalog/products/{id}` with `barcode` |
| `pegasusX/apps/supplier-app-ios` | Yes (`EANBarcodeScannerView`) | Yes | Same |
| `pegasusX/apps/supplier-portal` | No | Yes | Same |

**Catalog lookup (optional):** `GET /v1/catalog/barcode/{ean}`

**Idempotency:** All mutating scan/confirm calls MUST send `Idempotency-Key` header. Reuse per-app helpers (`PayloadIdempotencyKeys`, `PayloadIdempotency`, `WarehouseIdempotency`).

**Offline:** When `online == false`, enqueue scan payload to the app's offline queue (`returns/inbound/scan` endpoint). Drain on reconnect.

---

## 3. Per-platform stack (canonical versions)

Do **not** adopt a cross-platform barcode framework (no React Native Vision Camera, no ecosystem-wide ZXing wrapper, no Scandit by default).

| Platform | Stack | Notes |
|----------|-------|-------|
| **Android** | `androidx.camera:camera-*:1.4.1` + `com.google.mlkit:barcode-scanning:17.3.0` | Shared module: `pegasusX/packages/mobile-android-barcode-scanner` |
| **iOS** | VisionKit `DataScannerViewController` (iOS 16+, preferred) | Shared source: `pegasusX/packages/mobile-ios-barcode/EANBarcodeScannerView.swift` |
| **iOS fallback** | AVFoundation `.ean13` / `.ean8` metadata | Used when DataScanner unavailable |
| **Expo** | `expo-camera` `CameraView` + `barcodeScannerSettings.barcodeTypes: ['ean13','ean8']` | Requires dev-client rebuild |
| **Web** | `<input>` + hardware wedge (scanner acts as keyboard) | No camera dependency |

---

## 4. Client scan UX contract (normative)

All return-gate EAN surfaces MUST implement:

1. **Debounce** duplicate reads (~1.5s for the same code).
2. **Haptic / vibrate** on successful decode.
3. **Auto-submit** to `POST /v1/returns/inbound/scan` after decode (keep manual text field as fallback).
4. **Idempotency-Key** on every scan and confirm mutation.
5. **Offline enqueue** when network is unavailable (payload apps).
6. **EAN-first filtering** — prefer EAN-8/EAN-13 over other symbologies.

---

## 5. Confirmed clean (no barcode/scanner)

- `apps/retailer-app-android/`, `apps/retailer-app-ios/`, `apps/retailer-app-desktop/`
- `apps/factory-app-android/`, `apps/factory-app-ios/`, `apps/factory-portal/`

---

## 6. When Scandit (or similar commercial SDK) makes sense

**Consider a commercial SDK when:**

- Labels are damaged, partial, or unreadable under phone cameras in production lighting.
- Scan SLA requires sub-200ms decode at high volume (dedicated scan lanes).
- Symbologies beyond EAN are required (Code128 pallets, GS1 DataMatrix, PDF417).
- Integrating dedicated Zebra/Honeywell scan engines with vendor SDKs.

**Not justified for:**

- Standard indoor warehouse gate EAN scanning with phone/tablet cameras under normal lighting.
- The current return-gate MVP (EAN-8/13 on retail product labels).

---

## 7. How to add scanning to an approved surface

### Android

1. Add `implementation(project(":barcode-scanner"))` in app `build.gradle.kts` and `include` the module in `settings.gradle.kts`.
2. Add `CAMERA` permission to `AndroidManifest.xml`.
3. Use `@Composable EanBarcodeScannerPreview(onBarcode, modifier, enabled)` from the shared module.
4. Wire scan handler to `POST /v1/returns/inbound/scan` with idempotency key.

### iOS

1. Add `INFOPLIST_KEY_NSCameraUsageDescription` in `project.yml`.
2. Add `pegasusX/packages/mobile-ios-barcode/EANBarcodeScannerView.swift` to the app target.
3. Embed `EANBarcodeScannerView` in the inbound returns screen; on decode call scan API.

### Expo (`payload-terminal`)

1. `npx expo install expo-camera`
2. Add `expo-camera` plugin in `app.json` with camera permission string.
3. Use `CameraView` with `barcodeScannerSettings` and `onBarcodeScanned`.
4. Rebuild dev client.

### Web (`warehouse-portal`)

1. Auto-focus scan input on load.
2. Submit on Enter (wedge scanners append Enter after barcode).
3. No camera library.

---

## 8. Policy for new work

- **Do not** add barcode scanning to surfaces not listed in sections 1–2 without updating this doc AND product sign-off.
- **Do not** add Scandit or cross-platform barcode abstractions without the criteria in section 6.
- Driver apps keep QR scanners — do not change them to EAN or extend QR to new roles.
- New `mlkit:barcode-scanning`, `androidx.camera:*`, `expo-camera`, or VisionKit usage outside approved apps is a policy violation.

---

## 9. Decision log

| Date | Decision | Owner |
|------|----------|-------|
| 2026-04-15 | Remove unused barcode scaffolding from payload-app (Android/iOS), retailer-app-android; preserve active driver-app scanners. | Product |
| 2026-06-15 | Approve return-gate EAN on payload + warehouse surfaces; standardize per-platform native stacks; driver QR unchanged. | Product |
| 2026-06-17 | Approve supplier native catalog EAN capture (camera + manual) on create/edit product. | Product |
