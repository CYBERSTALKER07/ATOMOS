**Generatedat:** 2026-04-06

**Pageid:** ios-driver-qr-scanner

**Viewname:** QRScannerView

**Platform:** ios

**Role:** DRIVER

# Sourcefiles

- apps/driverappios/driverappios/Views/QRScannerView.swift

**Shell:** driver-ios-main

**Status:** implemented

**Purpose:** Driver scan-entry screen that overlays a QR targeting reticle on a live camera preview and routes validated scans into the offload workflow.

# Layoutzones

**Zoneid:** camera-preview

**Position:** full-screen base layer

## Contents

- live camera feed

---

**Zoneid:** top-cancel-bar

**Position:** top safe-area inset

## Contents

- Cancel text button aligned left

---

**Zoneid:** center-reticle

**Position:** screen center

## Contents

- rounded square targeting border
- light inner translucent scan area

---

**Zoneid:** processing-indicator

**Position:** bottom center above safe area

**Visibilityrule:** visible when vm.isProcessing is true

## Contents

- ProgressView spinner
- Processing text

---

**Zoneid:** system-alert-layer

**Position:** center overlay

**Visibilityrule:** visible for camera permission denial or validation result alerts

## Contents

- camera access required alert with Close button
- scan result alert with Rescan and Close or OK buttons

---


# Buttonplacements

**Button:** Cancel

**Zone:** top-cancel-bar left

**Style:** text button

---

**Button:** Close

**Zone:** camera permission alert

**Style:** system alert action

---

**Button:** Rescan

**Zone:** failed scan alert

**Style:** system alert cancel action

---

**Button:** Close

**Zone:** failed scan alert

**Style:** system alert default action

---

**Button:** OK

**Zone:** successful validation alert

**Style:** system alert default action

---


# Iconplacements

**Icon:** rounded reticle border

**Zone:** center-reticle

---

**Icon:** ProgressView

**Zone:** processing-indicator

---


# Interactiveflows

**Flowid:** scan-and-validate

## Steps

- View requests camera permission on task start
- Driver points camera at QR code inside reticle
- ScannerViewModel handles scanned value
- Validated response returns through onValidated callback

---

**Flowid:** permission-denied

## Steps

- Camera permission check fails
- Camera Access Required alert appears
- Driver taps Close and scanner exits via onCancel

---

**Flowid:** scan-failure-recovery

## Steps

- Validation fails
- Alert presents Rescan and Close choices
- Driver either resumes scanning or exits

---


# Statevariants

- live camera scan state
- processing state
- camera permission alert
- failed scan alert
- successful validation alert

# Figureblueprints

- scanner screen with reticle
- scanner processing state
- scanner permission alert
- scanner failure alert

