**Generatedat:** 2026-04-06

**Pageid:** android-driver-scanner

**Navroute:** scanner

**Platform:** android

**Role:** DRIVER

# Sourcefiles

- apps/driver-app-android/app/src/main/java/com/thelab/driver/ui/screens/scanner/ScannerScreen.kt

**Shell:** driver-android-main

**Status:** implemented

**Purpose:** Android driver scan-entry screen that validates retailer QR codes from a live camera preview and branches into cargo review or retry states.

# Layoutzones

**Zoneid:** camera-preview

**Position:** full-screen base layer

## Contents

- CameraPreview AndroidView filling the full screen

---

**Zoneid:** bottom-scan-prompt

**Position:** bottom center above safe area while scanning

**Visibilityrule:** visible when state.isScanning is true

## Contents

- rounded dark prompt card
- QrCodeScanner icon
- scanner prompt text

---

**Zoneid:** validating-overlay

**Position:** full-screen modal overlay

**Visibilityrule:** visible when state.isSubmitting is true

## Contents

- dark scrim
- CircularProgressIndicator
- Validating QR text

---

**Zoneid:** validated-overlay

**Position:** full-screen modal overlay

**Visibilityrule:** visible when state.validated is non-null

## Contents

- CheckCircle icon
- QR Verified title
- retailer name
- total amount
- item count
- Review Cargo button
- Scan Next filled tonal button

---

**Zoneid:** error-overlay

**Position:** full-screen modal overlay

**Visibilityrule:** visible when state.error is non-null

## Contents

- ErrorOutline icon
- error text
- Retry button

---

**Zoneid:** close-control

**Position:** top-right corner

## Contents

- white close icon button

---


# Buttonplacements

**Button:** Close scanner

**Zone:** close-control top-right

**Style:** icon button

---

**Button:** Review Cargo

**Zone:** validated-overlay

**Style:** primary button

---

**Button:** Scan Next

**Zone:** validated-overlay

**Style:** filled tonal button

---

**Button:** Retry

**Zone:** error-overlay

**Style:** primary button

---


# Iconplacements

**Icon:** QrCodeScanner

**Zone:** bottom-scan-prompt

---

**Icon:** CheckCircle

**Zone:** validated-overlay

---

**Icon:** ErrorOutline

**Zone:** error-overlay

---

**Icon:** Close

**Zone:** close-control

---

**Icon:** CircularProgressIndicator

**Zone:** validating-overlay

---


# Interactiveflows

**Flowid:** scan-and-validate

## Steps

- Driver enters scanner route
- CameraPreview analyzes barcodes continuously
- Detected value is handed to ScannerViewModel
- Validated payload opens verified overlay
- Driver taps Review Cargo to continue to offload review

---

**Flowid:** scan-reset

## Steps

- Driver taps Scan Next after a successful validation or Retry after an error
- Scanner state resets and live preview resumes

---

**Flowid:** scanner-exit

## Steps

- Driver taps top-right close button
- Route exits through onClose

---


# Statevariants

- active scan state
- validating overlay state
- validated overlay state
- error overlay state

# Figureblueprints

- android scanner screen with prompt card
- scanner validating overlay
- validated overlay with review and scan-next buttons
- scanner error overlay

