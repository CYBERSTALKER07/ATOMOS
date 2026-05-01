**Generatedat:** 2026-04-06T19:27:38.276Z

**Language:** uz

**Label:** O'zbekcha

**Sourcefolder:** patent-dossier/page-dossiers

**Localizationmode:** overlay-with-source-anchors

# Notes

- This file is a localized overlay for detailed page dossiers.
- Asosiy inglizcha JSON fayllar kanonik dalil manbai bo'lib qoladi; lokalizatsiya qilingan gaplar aniq UI label va texnik identifikatorlarni source anchor sifatida saqlaydi.
- Route, endpoint, file path, page ID va icon nomlari texnik anchor sifatida o'zgartirilmaydi.

**Filecount:** 59

# Entries

**Dossierfile:** driver-android-cash-collection.json

**Pageid:** android-driver-cash-collection

**Navroute:** cash_collection/{orderId}/{amountUZS}

**Platform:** android

**Role:** DRIVER

**Status:** implemented

**Shell:** driver-android-main

## Sourcefiles

- apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/offload/CashCollectionScreen.kt

**Entrytype:** page

**Localizedsummary:** "android-driver-cash-collection" yuzasi uchun haydovchi roli va android platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "android-driver-cash-collection" yuzasi android platformasida haydovchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Android driver cash-handling screen requiring explicit confirmation before delivery completion and guarding against accidental back navigation.

### Layoutoverview

- "center-stack" zonasi "center vertical stack" hududida joylashgan. Tarkibi: "Payments icon"; "COLLECT CASH heading"; "amount text"; "instructional helper text".
- "error-row" zonasi "below helper text when present" hududida joylashgan. Ko'rinish qoidasi: "visible when state.error is non-null". Tarkibi: "centered error text".
- "completion-cta" zonasi "bottom of center stack" hududida joylashgan. Tarkibi: "Cash Collected — Complete button".
- "exit-confirm-dialog" zonasi "modal overlay" hududida joylashgan. Ko'rinish qoidasi: "visible when showExitConfirm is true". Tarkibi: "Leave cash collection title"; "warning text"; "Stay button"; "Leave button".

### Controloverview

- "Cash Collected — Complete" tugmasi "completion-cta" hududida joylashgan. Uslub: "full-width primary".
- "Stay" tugmasi "exit-confirm-dialog confirm button slot" hududida joylashgan. Uslub: "text button".
- "Leave" tugmasi "exit-confirm-dialog dismiss button slot" hududida joylashgan. Uslub: "text button".

### Iconoverview

- "Payments" ikonasi "center-stack top" zonasida ishlatiladi.
- "CircularProgressIndicator" ikonasi "completion CTA while isCompleting" zonasida ishlatiladi.

### Flowoverview

**Flowid:** cash-completion

**Summary:** "cash-completion" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver reviews amount to collect".
- 2-qadam: "Driver taps Cash Collected — Complete".
- 3-qadam: "ViewModel calls collectCash".
- 4-qadam: "Route exits through onComplete when state.completed is true".

---

**Flowid:** guarded-back-navigation

**Summary:** "guarded-back-navigation" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver presses back before completion".
- 2-qadam: "BackHandler opens confirmation dialog".
- 3-qadam: "Driver chooses Stay or Leave".
- 4-qadam: "Submission state suppresses back navigation entirely".

---


### Stateoverview

- Holat: "cash collection idle state".
- Holat: "back-navigation confirmation dialog".
- Holat: "completion-in-flight state".
- Holat: "error state".

### Figureoverview

- Figura: "android cash collection screen".
- Figura: "cash collection exit-confirmation dialog".
- Figura: "cash completion CTA state".

---

**Dossierfile:** driver-android-delivery-correction.json

**Pageid:** android-driver-delivery-correction

**Navroute:** correction/{orderId}/{retailerName}

**Platform:** android

**Role:** DRIVER

**Status:** implemented

**Shell:** driver-android-main

## Sourcefiles

- apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/manifest/DeliveryCorrectionScreen.kt

**Entrytype:** page

**Localizedsummary:** "android-driver-delivery-correction" yuzasi uchun haydovchi roli va android platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "android-driver-delivery-correction" yuzasi android platformasida haydovchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Android driver reconciliation screen for editing accepted quantities, assigning rejection reasons, previewing refund impact, and submitting amended manifests.

### Layoutoverview

- "top-app-bar" zonasi "top scaffold app bar" hududida joylashgan. Tarkibi: "left: back arrow button"; "center: Verify Cargo title; optional retailer subtitle"; "right: modified-count badge when modifications exist".
- "loading-or-error-state" zonasi "center body when list unavailable" hududida joylashgan. Tarkibi: "loading state with spinner and Loading manifest text"; "error state with Inventory2 icon, failed headline, and message".
- "manifest-list" zonasi "scrollable body when loaded" hududida joylashgan. Tarkibi: "section label showing manifest item count"; "order ID mono badge"; "line item cards with product name, SKU, accepted quantity, total, and modify icon"; "reason tag on modified items".
- "modification-bottom-sheet" zonasi "modal bottom sheet" hududida joylashgan. Ko'rinish qoidasi: "visible when editingIndex targets a line item". Tarkibi: "product header"; "accepted quantity stepper"; "auto-calculated rejected text"; "rejection reason filter chips"; "adjusted line total preview"; "Apply Modification or No Changes button".
- "sticky-footer" zonasi "bottom bar" hududida joylashgan. Tarkibi: "original total row"; "animated refund delta row"; "adjusted total row"; "Submit Amendment or Confirm and Complete Delivery button".
- "confirm-dialog" zonasi "alert overlay" hududida joylashgan. Ko'rinish qoidasi: "visible when showConfirmDialog is true". Tarkibi: "Warning icon"; "Confirm Amendment title"; "modification count text"; "refund amount panel"; "Confirm Amendment button"; "Cancel button".

### Controloverview

- "Back" tugmasi "top-app-bar navigation icon" hududida joylashgan. Uslub: "icon button".
- "Modify item" tugmasi "line item card top-right" hududida joylashgan. Uslub: "small icon button".
- "Decrease accepted" tugmasi "modification-bottom-sheet stepper" hududida joylashgan. Uslub: "filled icon button".
- "Increase accepted" tugmasi "modification-bottom-sheet stepper" hududida joylashgan. Uslub: "filled icon button".
- "Rejection reason chip" tugmasi "modification-bottom-sheet" hududida joylashgan. Uslub: "filter chip".
- "Apply Modification" tugmasi "modification-bottom-sheet footer" hududida joylashgan. Uslub: "full-width primary". Ko'rinish qoidasi: "rejected quantity greater than zero".
- "No Changes" tugmasi "modification-bottom-sheet footer" hududida joylashgan. Uslub: "full-width primary". Ko'rinish qoidasi: "rejected quantity equals zero".
- "Submit Amendment" tugmasi "sticky-footer" hududida joylashgan. Uslub: "full-width error-colored button". Ko'rinish qoidasi: "state.hasModifications is true".
- "Confirm & Complete Delivery" tugmasi "sticky-footer" hududida joylashgan. Uslub: "full-width primary button". Ko'rinish qoidasi: "state.hasModifications is false".
- "Confirm Amendment" tugmasi "confirm-dialog confirm button" hududida joylashgan. Uslub: "error primary".
- "Cancel" tugmasi "confirm-dialog dismiss button" hududida joylashgan. Uslub: "text button".

### Iconoverview

- "ArrowBack" ikonasi "top-app-bar navigation icon" zonasida ishlatiladi.
- "Edit" ikonasi "line item modify control" zonasida ishlatiladi.
- "Warning" ikonasi "modified reason tag, sticky-footer submit CTA, and confirm dialog" zonasida ishlatiladi.
- "Remove" ikonasi "bottom-sheet decrement control" zonasida ishlatiladi.
- "Add" ikonasi "bottom-sheet increment control" zonasida ishlatiladi.
- "CheckCircle" ikonasi "non-modified footer CTA" zonasida ishlatiladi.
- "Inventory2" ikonasi "error state" zonasida ishlatiladi.
- "CircularProgressIndicator" ikonasi "loading state and submit CTA while submitting" zonasida ishlatiladi.

### Flowoverview

**Flowid:** modify-line-item

**Summary:** "modify-line-item" oqimi 6 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver taps edit icon on a line item card".
- 2-qadam: "Modal bottom sheet opens".
- 3-qadam: "Driver changes accepted quantity with stepper".
- 4-qadam: "Rejected quantity auto-calculates".
- 5-qadam: "Driver optionally selects rejection reason chips".
- 6-qadam: "Driver applies modification and returns to list".

---

**Flowid:** footer-summary-updates

**Summary:** "footer-summary-updates" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Any modification updates modified-count badge".
- 2-qadam: "Refund delta and adjusted total in sticky footer recompute live".
- 3-qadam: "Footer CTA changes from confirm-complete to submit-amendment mode".

---

**Flowid:** confirm-amendment

**Summary:** "confirm-amendment" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver taps Submit Amendment".
- 2-qadam: "Alert dialog summarizes modification count and refund amount".
- 3-qadam: "Driver confirms".
- 4-qadam: "ViewModel submits amendment and route completes on success".

---


### Stateoverview

- Holat: "loading manifest state".
- Holat: "error state".
- Holat: "clean manifest state".
- Holat: "modified manifest state with badges and reason tags".
- Holat: "modification bottom sheet open".
- Holat: "confirm amendment dialog".
- Holat: "submitting footer state".

### Figureoverview

- Figura: "android delivery correction full screen".
- Figura: "modified line item card".
- Figura: "quantity-edit bottom sheet with reason chips".
- Figura: "sticky footer with refund delta".
- Figura: "confirm amendment dialog".

---

**Dossierfile:** driver-android-offload-review.json

**Pageid:** android-driver-offload-review

**Navroute:** offload_review/{orderId}/{retailerName}

**Platform:** android

**Role:** DRIVER

**Status:** implemented

**Shell:** driver-android-main

## Sourcefiles

- apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/offload/OffloadReviewScreen.kt

**Entrytype:** page

**Localizedsummary:** "android-driver-offload-review" yuzasi uchun haydovchi roli va android platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "android-driver-offload-review" yuzasi android platformasida haydovchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Android driver cargo-review screen for checking accepted totals, excluding damaged units, and confirming offload before payment or cash collection routing.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "left: back icon button; OFFLOAD REVIEW monospace label; retailer name".
- "totals-bar" zonasi "below header" hududida joylashgan. Tarkibi: "original total cluster"; "adjusted total cluster with dynamic color".
- "line-item-list" zonasi "scrollable body" hududida joylashgan. Tarkibi: "status icon per line item"; "product name"; "quantity and unit price line"; "accepted total"; "rejected quantity stepper".
- "error-row" zonasi "above footer when present" hududida joylashgan. Ko'rinish qoidasi: "visible when state.error is non-null". Tarkibi: "red error text".
- "footer-cta" zonasi "bottom full-width container" hududida joylashgan. Tarkibi: "Confirm Offload or Amend and Confirm Offload button with spinner state".

### Controloverview

- "Back" tugmasi "header-left" hududida joylashgan. Uslub: "icon button".
- "Reduce rejected" tugmasi "line-item stepper" hududida joylashgan. Uslub: "icon button".
- "Increase rejected" tugmasi "line-item stepper" hududida joylashgan. Uslub: "icon button".
- "Confirm Offload" tugmasi "footer-cta" hududida joylashgan. Uslub: "full-width primary".
- "Amend & Confirm Offload" tugmasi "footer-cta" hududida joylashgan. Uslub: "full-width primary". Ko'rinish qoidasi: "state.hasExclusions is true".

### Iconoverview

- "ArrowBack" ikonasi "header-left back control" zonasida ishlatiladi.
- "CheckCircle" ikonasi "line-item row when no exclusions" zonasida ishlatiladi.
- "RemoveCircleOutline" ikonasi "line-item row when fully rejected" zonasida ishlatiladi.
- "RemoveCircle" ikonasi "stepper decrement" zonasida ishlatiladi.
- "AddCircle" ikonasi "stepper increment" zonasida ishlatiladi.
- "CircularProgressIndicator" ikonasi "footer CTA while submitting" zonasida ishlatiladi.

### Flowoverview

**Flowid:** line-item-exclusion-adjustment

**Summary:** "line-item-exclusion-adjustment" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver uses stepper controls to increase or decrease rejected quantity per line item".
- 2-qadam: "Accepted total and status coloring recompute per row".
- 3-qadam: "Adjusted total in totals bar updates".

---

**Flowid:** confirm-offload-clean

**Summary:** "confirm-offload-clean" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver leaves all rows fully accepted".
- 2-qadam: "Driver taps Confirm Offload".
- 3-qadam: "OffloadReviewViewModel confirms offload and returns result".
- 4-qadam: "Route branches to payment or cash flow based on response".

---

**Flowid:** confirm-offload-amended

**Summary:** "confirm-offload-amended" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver excludes one or more units".
- 2-qadam: "Footer label changes to Amend and Confirm Offload".
- 3-qadam: "Submission persists amended quantities before moving to downstream payment handling".

---


### Stateoverview

- Holat: "clean offload state".
- Holat: "partially rejected line-item state".
- Holat: "fully rejected line-item state".
- Holat: "submitting state".
- Holat: "error state".

### Figureoverview

- Figura: "android offload review full screen".
- Figura: "line-item stepper detail".
- Figura: "totals bar before and after exclusions".
- Figura: "submitting offload CTA".

---

**Dossierfile:** driver-android-payment-waiting.json

**Pageid:** android-driver-payment-waiting

**Navroute:** payment_waiting/{orderId}/{amountUZS}

**Platform:** android

**Role:** DRIVER

**Status:** implemented

**Shell:** driver-android-main

## Sourcefiles

- apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/offload/PaymentWaitingScreen.kt

**Entrytype:** page

**Localizedsummary:** "android-driver-payment-waiting" yuzasi uchun haydovchi roli va android platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "android-driver-payment-waiting" yuzasi android platformasida haydovchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Android driver settlement screen that waits for electronic payment completion before enabling delivery finalization.

### Layoutoverview

- "status-stack" zonasi "center vertical stack" hududida joylashgan. Tarkibi: "hourglass or check-circle icon"; "AWAITING PAYMENT or PAYMENT RECEIVED heading"; "amount text"; "credit-card icon"; "Payme label".
- "waiting-copy" zonasi "below payment method label" hududida joylashgan. Ko'rinish qoidasi: "visible when state.paymentSettled is false". Tarkibi: "Waiting for retailer to complete payment text".
- "error-row" zonasi "above completion CTA" hududida joylashgan. Ko'rinish qoidasi: "visible when state.error is non-null". Tarkibi: "centered error text".
- "completion-cta" zonasi "bottom of central stack" hududida joylashgan. Tarkibi: "Complete Delivery button with disabled state until settlement".

### Controloverview

- "Complete Delivery" tugmasi "completion-cta" hududida joylashgan. Uslub: "full-width primary, disabled until paymentSettled".

### Iconoverview

- "HourglassTop" ikonasi "status-stack when awaiting" zonasida ishlatiladi.
- "CheckCircle" ikonasi "status-stack when settled" zonasida ishlatiladi.
- "CreditCard" ikonasi "payment method indicator" zonasida ishlatiladi.
- "CircularProgressIndicator" ikonasi "completion CTA while isCompleting" zonasida ishlatiladi.

### Flowoverview

**Flowid:** waiting-to-settled

**Summary:** "waiting-to-settled" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver remains on waiting screen after offload confirmation".
- 2-qadam: "ViewModel observes payment settlement state".
- 3-qadam: "Heading, icon, and CTA state update once paymentSettled becomes true".

---

**Flowid:** complete-after-settlement

**Summary:** "complete-after-settlement" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver taps Complete Delivery once enabled".
- 2-qadam: "ViewModel completes order".
- 3-qadam: "Route exits through onComplete when state.completed becomes true".

---


### Stateoverview

- Holat: "awaiting-payment state".
- Holat: "payment-received state".
- Holat: "CTA completing state".
- Holat: "error state".

### Figureoverview

- Figura: "android awaiting payment screen".
- Figura: "android payment received screen".
- Figura: "enabled and disabled completion CTA comparison".

---

**Dossierfile:** driver-android-root-shell.json

**Pageid:** android-driver-main-shell

**Navroute:** main

**Platform:** android

**Role:** DRIVER

**Status:** implemented

**Shell:** driver-android-main

## Sourcefiles

- apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/navigation/DriverNavigation.kt
- apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/navigation/MainTabView.kt

**Entrytype:** page

**Localizedsummary:** "android-driver-main-shell" yuzasi uchun haydovchi roli va android platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "android-driver-main-shell" yuzasi android platformasida haydovchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Authenticated driver execution shell that holds the core four-tab workspace and routes into scanner, offload review, payment waiting, cash collection, and correction flows.

### Layoutoverview

- "animated-content-region" zonasi "center full-width" hududida joylashgan. Tarkibi: "HOME content"; "MAP content"; "RIDES content"; "PROFILE content".
- "bottom-stack" zonasi "bottom full-width" hududida joylashgan. Tarkibi: "optional activeRideBar slot"; "80dp NavigationBar".
- "secondary-routes" zonasi "outside root shell but in same navigation graph" hududida joylashgan. Tarkibi: "ScannerScreen"; "OffloadReviewScreen"; "PaymentWaitingScreen"; "CashCollectionScreen"; "DeliveryCorrectionScreen".

### Controloverview

- "HOME tab" tugmasi "bottom nav" hududida joylashgan. Uslub: "NavigationBarItem".
- "MAP tab" tugmasi "bottom nav" hududida joylashgan. Uslub: "NavigationBarItem".
- "RIDES tab" tugmasi "bottom nav" hududida joylashgan. Uslub: "NavigationBarItem".
- "PROFILE tab" tugmasi "bottom nav" hududida joylashgan. Uslub: "NavigationBarItem".
- "scan entry CTA" tugmasi "home content route handoff" hududida joylashgan. Uslub: "screen CTA routed to scanner".
- "active ride bar tap target" tugmasi "bottom stack above nav" hududida joylashgan. Uslub: "floating summary CTA when supplied by host content".

### Iconoverview

- "Home filled and outlined" ikonasi "bottom nav" zonasida ishlatiladi.
- "Map filled and outlined" ikonasi "bottom nav" zonasida ishlatiladi.
- "ListAlt filled and outlined" ikonasi "bottom nav" zonasida ishlatiladi.
- "Person filled and outlined" ikonasi "bottom nav" zonasida ishlatiladi.

### Flowoverview

**Flowid:** scanner-to-offload

**Summary:** "scanner-to-offload" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver taps scan entry from Home".
- 2-qadam: "Navigation pushes ScannerScreen".
- 3-qadam: "Validated QR result pops scanner".
- 4-qadam: "Navigation pushes OffloadReviewScreen with orderId and retailerName".

---

**Flowid:** offload-to-payment-or-cash

**Summary:** "offload-to-payment-or-cash" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver confirms offload".
- 2-qadam: "Navigation examines paymentMethod in response".
- 3-qadam: "Cash path routes to CashCollectionScreen".
- 4-qadam: "Card path routes to PaymentWaitingScreen".

---

**Flowid:** return-to-main-shell

**Summary:** "return-to-main-shell" oqimi 1 ta qadamdan iborat.

#### Steps

- 1-qadam: "Completion actions from payment waiting, cash collection, or correction pop back to MAIN without destroying the main workspace".

---


### Stateoverview

- Holat: "home tab active".
- Holat: "map tab active".
- Holat: "rides tab active".
- Holat: "profile tab active".
- Holat: "active ride bar present".
- Holat: "scanner route open".
- Holat: "offload review route open".
- Holat: "payment waiting route open".
- Holat: "cash collection route open".
- Holat: "correction route open".

### Figureoverview

- Figura: "driver main shell with four-tab navigation".
- Figura: "active ride bar plus bottom navigation".
- Figura: "scanner handoff sequence".
- Figura: "offload review to payment branch sequence".
- Figura: "cash collection route state".

---

**Dossierfile:** driver-android-scanner.json

**Pageid:** android-driver-scanner

**Navroute:** scanner

**Platform:** android

**Role:** DRIVER

**Status:** implemented

**Shell:** driver-android-main

## Sourcefiles

- apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/scanner/ScannerScreen.kt

**Entrytype:** page

**Localizedsummary:** "android-driver-scanner" yuzasi uchun haydovchi roli va android platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "android-driver-scanner" yuzasi android platformasida haydovchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Android driver scan-entry screen that validates retailer QR codes from a live camera preview and branches into cargo review or retry states.

### Layoutoverview

- "camera-preview" zonasi "full-screen base layer" hududida joylashgan. Tarkibi: "CameraPreview AndroidView filling the full screen".
- "bottom-scan-prompt" zonasi "bottom center above safe area while scanning" hududida joylashgan. Ko'rinish qoidasi: "visible when state.isScanning is true". Tarkibi: "rounded dark prompt card"; "QrCodeScanner icon"; "scanner prompt text".
- "validating-overlay" zonasi "full-screen modal overlay" hududida joylashgan. Ko'rinish qoidasi: "visible when state.isSubmitting is true". Tarkibi: "dark scrim"; "CircularProgressIndicator"; "Validating QR text".
- "validated-overlay" zonasi "full-screen modal overlay" hududida joylashgan. Ko'rinish qoidasi: "visible when state.validated is non-null". Tarkibi: "CheckCircle icon"; "QR Verified title"; "retailer name"; "total amount"; "item count"; "Review Cargo button"; "Scan Next filled tonal button".
- "error-overlay" zonasi "full-screen modal overlay" hududida joylashgan. Ko'rinish qoidasi: "visible when state.error is non-null". Tarkibi: "ErrorOutline icon"; "error text"; "Retry button".
- "close-control" zonasi "top-right corner" hududida joylashgan. Tarkibi: "white close icon button".

### Controloverview

- "Close scanner" tugmasi "close-control top-right" hududida joylashgan. Uslub: "icon button".
- "Review Cargo" tugmasi "validated-overlay" hududida joylashgan. Uslub: "primary button".
- "Scan Next" tugmasi "validated-overlay" hududida joylashgan. Uslub: "filled tonal button".
- "Retry" tugmasi "error-overlay" hududida joylashgan. Uslub: "primary button".

### Iconoverview

- "QrCodeScanner" ikonasi "bottom-scan-prompt" zonasida ishlatiladi.
- "CheckCircle" ikonasi "validated-overlay" zonasida ishlatiladi.
- "ErrorOutline" ikonasi "error-overlay" zonasida ishlatiladi.
- "Close" ikonasi "close-control" zonasida ishlatiladi.
- "CircularProgressIndicator" ikonasi "validating-overlay" zonasida ishlatiladi.

### Flowoverview

**Flowid:** scan-and-validate

**Summary:** "scan-and-validate" oqimi 5 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver enters scanner route".
- 2-qadam: "CameraPreview analyzes barcodes continuously".
- 3-qadam: "Detected value is handed to ScannerViewModel".
- 4-qadam: "Validated payload opens verified overlay".
- 5-qadam: "Driver taps Review Cargo to continue to offload review".

---

**Flowid:** scan-reset

**Summary:** "scan-reset" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver taps Scan Next after a successful validation or Retry after an error".
- 2-qadam: "Scanner state resets and live preview resumes".

---

**Flowid:** scanner-exit

**Summary:** "scanner-exit" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver taps top-right close button".
- 2-qadam: "Route exits through onClose".

---


### Stateoverview

- Holat: "active scan state".
- Holat: "validating overlay state".
- Holat: "validated overlay state".
- Holat: "error overlay state".

### Figureoverview

- Figura: "android scanner screen with prompt card".
- Figura: "scanner validating overlay".
- Figura: "validated overlay with review and scan-next buttons".
- Figura: "scanner error overlay".

---

**Dossierfile:** driver-android-secondary-surfaces.json

**Bundleid:** driver-android-secondary-surfaces

**Appid:** driver-app-android

**Platform:** android

**Role:** DRIVER

**Status:** implemented

**Entrytype:** bundle

**Localizedsummary:** "driver-android-secondary-surfaces" paketi "driver-app-android" ilovasi uchun 6 ta yuzani qamrab oladi.

## Surfaces

**Pageid:** android-driver-login

**Navroute:** login

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/auth/LoginScreen.kt

**Localizedsummary:** "android-driver-login" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-driver-login" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Android driver sign-in form using phone and PIN with IME management and auth loading feedback.

#### Layoutoverview

- Layout zonasi: "brand header".
- Layout zonasi: "phone and PIN text field column".
- Layout zonasi: "PIN visibility icon button".
- Layout zonasi: "login CTA and error state".

#### Controloverview

- Boshqaruv elementi: "PIN visibility icon button in PIN field trailing slot".
- Boshqaruv elementi: "Login button below fields".

#### Iconoverview

- Ikona joylashuvi: "brand mark at screen top".
- Ikona joylashuvi: "eye visibility icon".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "type phone and PIN".

---

**Summary:** Oqim quyidagicha qayd etilgan: "toggle PIN visibility".

---

**Summary:** Oqim quyidagicha qayd etilgan: "submit auth coroutine and persist session".

---

**Summary:** Oqim quyidagicha qayd etilgan: "show loading spinner during auth".

---


#### Dependencyoverview

##### Reads

- driver login API

##### Writes

- driver token store

##### Localizednotes

- O'qish: "driver login API".
- Yozish: "driver token store".

#### Stateoverview

- Holat: "idle".
- Holat: "loading".
- Holat: "error".

#### Figureoverview

- Figura: "android login screen with phone and PIN form".

#### Minifeatureoverview

- Mini-feature: "phone prefill".
- Mini-feature: "PIN field".
- Mini-feature: "visibility toggle".
- Mini-feature: "loading spinner".
- Mini-feature: "error message".

**Minifeaturecount:** 5

---

**Pageid:** android-driver-home

**Navroute:** HOME

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/home/HomeScreen.kt

**Localizedsummary:** "android-driver-home" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-driver-home" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Android driver home dashboard with status chips, vehicle card, transit control, and quick actions.

#### Layoutoverview

- Layout zonasi: "time-based greeting and status chips".
- Layout zonasi: "vehicle info card".
- Layout zonasi: "transit control card".
- Layout zonasi: "today summary band".
- Layout zonasi: "quick action row".
- Layout zonasi: "recent activity list".

#### Controloverview

- Boshqaruv elementi: "Open Map CTA".
- Boshqaruv elementi: "Scan QR CTA".

#### Iconoverview

- Ikona joylashuvi: "route-state icon".
- Ikona joylashuvi: "truck or cargo glyphs in cards".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "refresh dashboard state".

---

**Summary:** Oqim quyidagicha qayd etilgan: "jump to map".

---

**Summary:** Oqim quyidagicha qayd etilgan: "jump to scanner".

---


#### Dependencyoverview

##### Reads

- ManifestViewModel state

##### Writes


##### Localizednotes

- O'qish: "ManifestViewModel state".
- Yozish: yo'q.

#### Stateoverview

- Holat: "idle".
- Holat: "on route".
- Holat: "loading".

#### Figureoverview

- Figura: "android driver home dashboard".

#### Minifeatureoverview

- Mini-feature: "greeting".
- Mini-feature: "status chips".
- Mini-feature: "vehicle card".
- Mini-feature: "transit control".
- Mini-feature: "summary band".
- Mini-feature: "Open Map CTA".
- Mini-feature: "Scan QR CTA".
- Mini-feature: "recent activity list".

**Minifeaturecount:** 8

---

**Pageid:** android-driver-map

**Navroute:** MAP

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/map/MapScreen.kt

**Localizedsummary:** "android-driver-map" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-driver-map" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Stub placeholder reserving the future Google Maps execution surface in the Android driver stack.

#### Layoutoverview

- Layout zonasi: "centered placeholder icon".
- Layout zonasi: "stub title and explanatory subtitle".

#### Controloverview

- Boshqaruv elementi: "none; stub surface".

#### Iconoverview

- Ikona joylashuvi: "Map icon centered".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "static placeholder only".

---


#### Dependencyoverview

##### Reads


##### Writes


##### Localizednotes

- O'qish: yo'q.
- Yozish: yo'q.

#### Stateoverview

- Holat: "single stub state".

#### Figureoverview

- Figura: "stub map placeholder figure".

#### Minifeatureoverview

- Mini-feature: "map pending icon".
- Mini-feature: "placeholder messaging".

**Minifeaturecount:** 2

---

**Pageid:** android-driver-rides

**Navroute:** RIDES

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/manifest/ManifestScreen.kt

**Localizedsummary:** "android-driver-rides" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-driver-rides" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Android route manifest ledger with loading-mode reversal for physical truck packing and upcoming stop review.

#### Layoutoverview

- Layout zonasi: "UPCOMING header with pending count".
- Layout zonasi: "Loading Mode switch row".
- Layout zonasi: "ride card lazy list".
- Layout zonasi: "loading or empty states".

#### Controloverview

- Boshqaruv elementi: "Loading Mode switch in header".
- Boshqaruv elementi: "ride card tap target".

#### Iconoverview

- Ikona joylashuvi: "loading sequence badge".
- Ikona joylashuvi: "status pill".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "toggle loading mode".

---

**Summary:** Oqim quyidagicha qayd etilgan: "tap ride to focus mission".

---

**Summary:** Oqim quyidagicha qayd etilgan: "refresh manifest".

---


#### Dependencyoverview

##### Reads

- ManifestViewModel.state

##### Writes

- selected mission state

##### Localizednotes

- O'qish: "ManifestViewModel.state".
- Yozish: "selected mission state".

#### Stateoverview

- Holat: "standard order".
- Holat: "loading order".
- Holat: "empty".
- Holat: "loading".

#### Figureoverview

- Figura: "android rides manifest with loading mode switch".

#### Minifeatureoverview

- Mini-feature: "pending count badge".
- Mini-feature: "loading mode switch".
- Mini-feature: "ride cards".
- Mini-feature: "sequence badge".
- Mini-feature: "status pill".
- Mini-feature: "empty state".

**Minifeaturecount:** 6

---

**Pageid:** android-driver-profile

**Navroute:** PROFILE

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/profile/ProfileScreen.kt

**Localizedsummary:** "android-driver-profile" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-driver-profile" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Android driver profile screen with truck identity, stats, quick actions, and ride-history review.

#### Layoutoverview

- Layout zonasi: "profile title header".
- Layout zonasi: "identity card".
- Layout zonasi: "truck and completion info grid".
- Layout zonasi: "quick actions row".
- Layout zonasi: "ride history list".
- Layout zonasi: "stats section".

#### Controloverview

- Boshqaruv elementi: "Sync quick action".
- Boshqaruv elementi: "Logout quick action".
- Boshqaruv elementi: "Settings quick action".

#### Iconoverview

- Ikona joylashuvi: "initials avatar".
- Ikona joylashuvi: "quick action icons".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "sync state".

---

**Summary:** Oqim quyidagicha qayd etilgan: "logout session".

---

**Summary:** Oqim quyidagicha qayd etilgan: "review history".

---


#### Dependencyoverview

##### Reads

- ManifestViewModel driver and order stats

##### Writes

- sync or logout state

##### Localizednotes

- O'qish: "ManifestViewModel driver and order stats".
- Yozish: "sync or logout state".

#### Stateoverview

- Holat: "active".
- Holat: "idle".
- Holat: "history populated".

#### Figureoverview

- Figura: "android driver profile screen".

#### Minifeatureoverview

- Mini-feature: "identity card".
- Mini-feature: "status pill".
- Mini-feature: "truck grid".
- Mini-feature: "Sync action".
- Mini-feature: "Logout action".
- Mini-feature: "Settings action".
- Mini-feature: "history ledger".
- Mini-feature: "stats band".

**Minifeaturecount:** 8

---

**Pageid:** android-driver-correction

**Navroute:** correction/{orderId}/{retailerName}

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/manifest/DeliveryCorrectionScreen.kt

**Localizedsummary:** "android-driver-correction" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-driver-correction" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Alias dossier for the Android driver delivery-correction workflow already documented as a primary execution surface.

#### Layoutoverview

- Layout zonasi: "header app bar".
- Layout zonasi: "manifest item cards".
- Layout zonasi: "sticky summary footer".
- Layout zonasi: "correction bottom sheet and confirmation dialog overlays".

#### Controloverview

- Boshqaruv elementi: "Modify item action".
- Boshqaruv elementi: "confirm amendment action".

#### Iconoverview

- Ikona joylashuvi: "item correction glyphs".
- Ikona joylashuvi: "dialog warning icon".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "open correction editor".

---

**Summary:** Oqim quyidagicha qayd etilgan: "adjust delivered and rejected quantities".

---

**Summary:** Oqim quyidagicha qayd etilgan: "confirm amendment".

---


#### Dependencyoverview

##### Reads

- delivery correction payload

##### Writes

- correction submission

##### Localizednotes

- O'qish: "delivery correction payload".
- Yozish: "correction submission".

#### Stateoverview

- Holat: "review".
- Holat: "editing".
- Holat: "confirming".

#### Figureoverview

- Figura: "delivery correction alias figure".

#### Minifeatureoverview

- Mini-feature: "item cards".
- Mini-feature: "sticky footer".
- Mini-feature: "bottom sheet editor".
- Mini-feature: "confirmation dialog".

**Minifeaturecount:** 4

---


---

**Dossierfile:** driver-ios-cash-collection.json

**Pageid:** ios-driver-cash-collection

**Viewname:** CashCollectionView

**Platform:** ios

**Role:** DRIVER

**Status:** implemented

**Shell:** driver-ios-main

## Sourcefiles

- apps/driverappios/driverappios/Views/CashCollectionView.swift

**Entrytype:** page

**Localizedsummary:** "ios-driver-cash-collection" yuzasi uchun haydovchi roli va iOS platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "ios-driver-cash-collection" yuzasi iOS platformasida haydovchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Driver cash-confirmation screen used when retailer payment is collected physically before delivery completion.

### Layoutoverview

- "top-close-row" zonasi "top safe-area inset" hududida joylashgan. Tarkibi: "circular close button aligned right".
- "center-cash-stack" zonasi "center vertical stack" hududida joylashgan. Tarkibi: "banknote icon"; "Collect Cash title"; "order ID"; "amount"; "helper copy".
- "error-row" zonasi "above footer CTA when present" hududida joylashgan. Ko'rinish qoidasi: "visible when errorMessage exists". Tarkibi: "destructive error text".
- "footer-cta" zonasi "bottom full-width" hududida joylashgan. Tarkibi: "Cash Collected — Complete button with optional spinner".

### Controloverview

- "Close" tugmasi "top-close-row right" hududida joylashgan. Uslub: "icon button".
- "Cash Collected — Complete" tugmasi "footer-cta" hududida joylashgan. Uslub: "full-width primary".

### Iconoverview

- "xmark" ikonasi "top-close-row" zonasida ishlatiladi.
- "banknote.fill" ikonasi "center-cash-stack top" zonasida ishlatiladi.
- "ProgressView" ikonasi "footer CTA when completing" zonasida ishlatiladi.

### Flowoverview

**Flowid:** cancel-cash-collection

**Summary:** "cancel-cash-collection" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver taps close button".
- 2-qadam: "View exits through onCancel callback".

---

**Flowid:** collect-cash-and-complete

**Summary:** "collect-cash-and-complete" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver confirms physical collection".
- 2-qadam: "Driver taps Cash Collected — Complete".
- 3-qadam: "View calls collectCash(orderId)".
- 4-qadam: "Successful response exits through onCompleted callback".

---


### Stateoverview

- Holat: "cash collection idle state".
- Holat: "completion-in-flight state".
- Holat: "inline error state".

### Figureoverview

- Figura: "cash collection screen".
- Figura: "cash collection CTA state".
- Figura: "cash collection error state".

---

**Dossierfile:** driver-ios-delivery-correction.json

**Pageid:** ios-driver-delivery-correction

**Viewname:** DeliveryCorrectionView

**Platform:** ios

**Role:** DRIVER

**Status:** implemented

**Shell:** driver-ios-main

## Sourcefiles

- apps/driverappios/driverappios/Views/DeliveryCorrectionView.swift

**Entrytype:** page

**Localizedsummary:** "ios-driver-delivery-correction" yuzasi uchun haydovchi roli va iOS platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "ios-driver-delivery-correction" yuzasi iOS platformasida haydovchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Driver amendment screen for toggling manifest items between delivered and rejected states and calculating refund deltas before submission.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "left: Back button; Delivery Correction title; order ID"; "right: StatusPill showing rejected count or all-clear state".
- "loading-region" zonasi "center body" hududida joylashgan. Ko'rinish qoidasi: "visible when vm.isLoading is true". Tarkibi: "Loading line items progress indicator".
- "manifest-list" zonasi "scrollable body when loaded" hududida joylashgan. Tarkibi: "MANIFEST ITEMS section label"; "line item cards with sku, quantity x unit price, status pill, line total, bottom status bar".
- "summary-bar" zonasi "bottom material overlay" hududida joylashgan. Tarkibi: "original total row"; "refund delta row when refundDelta > 0"; "divider"; "adjusted total row"; "Submit Amendment or All Items Delivered CTA".
- "confirm-alert" zonasi "system alert overlay" hududida joylashgan. Ko'rinish qoidasi: "visible when showConfirmAlert is true". Tarkibi: "Confirm Amendment title"; "cancel button"; "destructive submit button"; "message with rejected count and refund delta".

### Controloverview

- "Back" tugmasi "header-left" hududida joylashgan. Uslub: "inline icon-text button".
- "line-item card tap target" tugmasi "manifest-list" hududida joylashgan. Uslub: "whole-card toggle button".
- "Submit Amendment" tugmasi "summary-bar footer" hududida joylashgan. Uslub: "full-width destructive". Ko'rinish qoidasi: "one or more items rejected".
- "All Items Delivered" tugmasi "summary-bar footer" hududida joylashgan. Uslub: "disabled muted". Ko'rinish qoidasi: "no items rejected".
- "Cancel" tugmasi "confirm-alert" hududida joylashgan. Uslub: "system alert cancel action".
- "Submit" tugmasi "confirm-alert" hududida joylashgan. Uslub: "system alert destructive action".

### Iconoverview

- "chevron.left" ikonasi "header back button" zonasida ishlatiladi.
- "StatusPill capsule" ikonasi "header-right" zonasida ishlatiladi.
- "bottom status bar on each line item card" ikonasi "line item footer" zonasida ishlatiladi.
- "ProgressView" ikonasi "loading-region" zonasida ishlatiladi.

### Flowoverview

**Flowid:** load-manifest-items

**Summary:** "load-manifest-items" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "View loads line items on task start".
- 2-qadam: "Loading state is replaced by tappable manifest cards".

---

**Flowid:** toggle-item-status

**Summary:** "toggle-item-status" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver taps a line item card".
- 2-qadam: "Item toggles between delivered and rejected status".
- 3-qadam: "Status pill, strikethrough, and bottom bar update".
- 4-qadam: "Summary bar recalculates refund delta and adjusted total".

---

**Flowid:** submit-amendment

**Summary:** "submit-amendment" oqimi 5 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver rejects one or more items".
- 2-qadam: "Driver taps Submit Amendment".
- 3-qadam: "Confirm Amendment alert appears with refund summary".
- 4-qadam: "Driver submits and page calls submitAmendment(orderId, driverId)".
- 5-qadam: "Successful submission exits through onAmended callback".

---


### Stateoverview

- Holat: "loading state".
- Holat: "all-clear manifest state".
- Holat: "mixed delivered and rejected items".
- Holat: "summary bar with refund delta".
- Holat: "confirm amendment alert".

### Figureoverview

- Figura: "delivery correction full screen".
- Figura: "line item card with rejected state".
- Figura: "summary bar with refund delta".
- Figura: "confirm amendment alert".

---

**Dossierfile:** driver-ios-offload-review.json

**Pageid:** ios-driver-offload-review

**Viewname:** OffloadReviewView

**Platform:** ios

**Role:** DRIVER

**Status:** implemented

**Shell:** driver-ios-main

## Sourcefiles

- apps/driverappios/driverappios/Views/OffloadReviewView.swift

**Entrytype:** page

**Localizedsummary:** "ios-driver-offload-review" yuzasi uchun haydovchi roli va iOS platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "ios-driver-offload-review" yuzasi iOS platformasida haydovchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Driver delivery-review screen for confirming offload, partially rejecting damaged units, and branching into payment collection flows.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "left: OFFLOAD REVIEW label; order ID in monospace"; "right: circular close button".
- "retailer-total-row" zonasi "below header" hududida joylashgan. Tarkibi: "retailer name"; "total amount".
- "line-item-list" zonasi "scrollable middle region" hududida joylashgan. Tarkibi: "product name"; "quantity x unit price line"; "minus button"; "rejected quantity counter"; "plus button".
- "error-row" zonasi "above footer CTA when present" hududida joylashgan. Ko'rinish qoidasi: "visible when errorMessage is non-null". Tarkibi: "inline destructive error text".
- "footer-cta" zonasi "bottom full-width" hududida joylashgan. Tarkibi: "Confirm Offload button with optional spinner".

### Controloverview

- "Close" tugmasi "header-right circular button" hududida joylashgan. Uslub: "icon button".
- "minus reject quantity" tugmasi "each line-item stepper" hududida joylashgan. Uslub: "icon stepper control".
- "plus reject quantity" tugmasi "each line-item stepper" hududida joylashgan. Uslub: "icon stepper control".
- "Confirm Offload" tugmasi "footer-cta" hududida joylashgan. Uslub: "full-width primary".

### Iconoverview

- "xmark" ikonasi "header-right close control" zonasida ishlatiladi.
- "minus.circle.fill" ikonasi "line-item stepper decrement" zonasida ishlatiladi.
- "plus.circle.fill" ikonasi "line-item stepper increment" zonasida ishlatiladi.
- "ProgressView" ikonasi "Confirm Offload button when submitting" zonasida ishlatiladi.

### Flowoverview

**Flowid:** quantity-rejection-adjustment

**Summary:** "quantity-rejection-adjustment" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver reviews line items".
- 2-qadam: "Driver uses plus or minus buttons to set rejected quantity per item".
- 3-qadam: "Item styling changes to delivered, partial, or fully rejected visual state".

---

**Flowid:** confirm-offload-no-rejections

**Summary:** "confirm-offload-no-rejections" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver leaves all rejected quantities at zero".
- 2-qadam: "Driver taps Confirm Offload".
- 3-qadam: "Page calls confirmOffload on fleet service".
- 4-qadam: "Successful response exits through onConfirm callback".

---

**Flowid:** confirm-offload-with-amendment

**Summary:** "confirm-offload-with-amendment" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver marks one or more rejected quantities".
- 2-qadam: "Page first calls amendOrder with derived status per line item".
- 3-qadam: "Page then calls confirmOffload".
- 4-qadam: "Workflow branches downstream based on returned payment mode".

---


### Stateoverview

- Holat: "all items delivered state".
- Holat: "partial rejection state".
- Holat: "full rejection for a line-item".
- Holat: "submitting CTA state".
- Holat: "inline error state".

### Figureoverview

- Figura: "offload review full screen".
- Figura: "line-item row with stepper controls".
- Figura: "mixed accepted and rejected quantities".
- Figura: "confirm-offload submitting state".

---

**Dossierfile:** driver-ios-payment-waiting.json

**Pageid:** ios-driver-payment-waiting

**Viewname:** PaymentWaitingView

**Platform:** ios

**Role:** DRIVER

**Status:** implemented

**Shell:** driver-ios-main

## Sourcefiles

- apps/driverappios/driverappios/Views/PaymentWaitingView.swift

**Entrytype:** page

**Localizedsummary:** "ios-driver-payment-waiting" yuzasi uchun haydovchi roli va iOS platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "ios-driver-payment-waiting" yuzasi iOS platformasida haydovchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Driver payment-settlement holding screen that waits for a websocket settlement event before enabling delivery completion.

### Layoutoverview

- "status-stack" zonasi "center vertical stack" hududida joylashgan. Tarkibi: "status icon"; "title"; "order ID"; "amount".
- "waiting-copy" zonasi "below amount when payment is unsettled" hududida joylashgan. Ko'rinish qoidasi: "visible when isSettled is false". Tarkibi: "ProgressView spinner"; "Retailer is completing payment helper text".
- "error-row" zonasi "above completion CTA when present" hududida joylashgan. Ko'rinish qoidasi: "visible when errorMessage exists". Tarkibi: "destructive error text".
- "completion-cta" zonasi "bottom full-width" hududida joylashgan. Tarkibi: "Complete Delivery button disabled until settlement".

### Controloverview

- "Complete Delivery" tugmasi "completion-cta" hududida joylashgan. Uslub: "full-width primary when settled and muted disabled button when unsettled".

### Iconoverview

- "clock.fill" ikonasi "status-stack icon when unsettled" zonasida ishlatiladi.
- "checkmark.seal.fill" ikonasi "status-stack icon when settled" zonasida ishlatiladi.
- "ProgressView" ikonasi "waiting-copy" zonasida ishlatiladi.

### Flowoverview

**Flowid:** settlement-wait-loop

**Summary:** "settlement-wait-loop" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "View opens websocket to /v1/ws/driver with driver_id and bearer token".
- 2-qadam: "Driver watches awaiting-payment state".
- 3-qadam: "Page listens for PAYMENT_SETTLED matching the current orderId".

---

**Flowid:** settlement-received

**Summary:** "settlement-received" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "PAYMENT_SETTLED websocket message arrives".
- 2-qadam: "isSettled flips to true".
- 3-qadam: "status icon changes from clock to seal".
- 4-qadam: "Complete Delivery button becomes enabled".

---

**Flowid:** complete-delivery

**Summary:** "complete-delivery" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver taps Complete Delivery after settlement".
- 2-qadam: "Page calls completeOrder".
- 3-qadam: "Successful completion exits through onCompleted callback".

---


### Stateoverview

- Holat: "awaiting payment state".
- Holat: "settled state".
- Holat: "completion-in-flight state".
- Holat: "error state".
- Holat: "websocket reconnect behavior after failure".

### Figureoverview

- Figura: "awaiting payment screen".
- Figura: "payment received screen".
- Figura: "disabled versus enabled completion CTA comparison".

---

**Dossierfile:** driver-ios-qr-scanner.json

**Pageid:** ios-driver-qr-scanner

**Viewname:** QRScannerView

**Platform:** ios

**Role:** DRIVER

**Status:** implemented

**Shell:** driver-ios-main

## Sourcefiles

- apps/driverappios/driverappios/Views/QRScannerView.swift

**Entrytype:** page

**Localizedsummary:** "ios-driver-qr-scanner" yuzasi uchun haydovchi roli va iOS platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "ios-driver-qr-scanner" yuzasi iOS platformasida haydovchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Driver scan-entry screen that overlays a QR targeting reticle on a live camera preview and routes validated scans into the offload workflow.

### Layoutoverview

- "camera-preview" zonasi "full-screen base layer" hududida joylashgan. Tarkibi: "live camera feed".
- "top-cancel-bar" zonasi "top safe-area inset" hududida joylashgan. Tarkibi: "Cancel text button aligned left".
- "center-reticle" zonasi "screen center" hududida joylashgan. Tarkibi: "rounded square targeting border"; "light inner translucent scan area".
- "processing-indicator" zonasi "bottom center above safe area" hududida joylashgan. Ko'rinish qoidasi: "visible when vm.isProcessing is true". Tarkibi: "ProgressView spinner"; "Processing text".
- "system-alert-layer" zonasi "center overlay" hududida joylashgan. Ko'rinish qoidasi: "visible for camera permission denial or validation result alerts". Tarkibi: "camera access required alert with Close button"; "scan result alert with Rescan and Close or OK buttons".

### Controloverview

- "Cancel" tugmasi "top-cancel-bar left" hududida joylashgan. Uslub: "text button".
- "Close" tugmasi "camera permission alert" hududida joylashgan. Uslub: "system alert action".
- "Rescan" tugmasi "failed scan alert" hududida joylashgan. Uslub: "system alert cancel action".
- "Close" tugmasi "failed scan alert" hududida joylashgan. Uslub: "system alert default action".
- "OK" tugmasi "successful validation alert" hududida joylashgan. Uslub: "system alert default action".

### Iconoverview

- "rounded reticle border" ikonasi "center-reticle" zonasida ishlatiladi.
- "ProgressView" ikonasi "processing-indicator" zonasida ishlatiladi.

### Flowoverview

**Flowid:** scan-and-validate

**Summary:** "scan-and-validate" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "View requests camera permission on task start".
- 2-qadam: "Driver points camera at QR code inside reticle".
- 3-qadam: "ScannerViewModel handles scanned value".
- 4-qadam: "Validated response returns through onValidated callback".

---

**Flowid:** permission-denied

**Summary:** "permission-denied" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Camera permission check fails".
- 2-qadam: "Camera Access Required alert appears".
- 3-qadam: "Driver taps Close and scanner exits via onCancel".

---

**Flowid:** scan-failure-recovery

**Summary:** "scan-failure-recovery" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Validation fails".
- 2-qadam: "Alert presents Rescan and Close choices".
- 3-qadam: "Driver either resumes scanning or exits".

---


### Stateoverview

- Holat: "live camera scan state".
- Holat: "processing state".
- Holat: "camera permission alert".
- Holat: "failed scan alert".
- Holat: "successful validation alert".

### Figureoverview

- Figura: "scanner screen with reticle".
- Figura: "scanner processing state".
- Figura: "scanner permission alert".
- Figura: "scanner failure alert".

---

**Dossierfile:** driver-ios-root-shell.json

**Pageid:** ios-driver-main-shell

**Viewname:** MainTabView

**Platform:** ios

**Role:** DRIVER

**Status:** implemented

**Shell:** driver-ios-main

## Sourcefiles

- apps/driverappios/driverappios/LabDriverApp.swift
- apps/driverappios/driverappios/Views/MainTabView.swift
- apps/driverappios/driverappios/Views/Components/ActiveRideBar.swift

**Entrytype:** page

**Localizedsummary:** "ios-driver-main-shell" yuzasi uchun haydovchi roli va iOS platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "ios-driver-main-shell" yuzasi iOS platformasida haydovchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Authenticated driver shell with tab-based execution workspace, full-screen map mode, and floating active-route summary above the tab bar.

### Layoutoverview

- "auth-gate" zonasi "root level above shell" hududida joylashgan. Tarkibi: "RootView switches between LoginView and MainTabView based on TokenStore.isAuthenticated".
- "tab-layer" zonasi "base layer when not on map" hududida joylashgan. Tarkibi: "Home tab"; "Rides tab"; "Profile tab".
- "map-mode" zonasi "full-screen replacement state" hududida joylashgan. Tarkibi: "FleetMapView with go-back closure".
- "bottom-safe-area-inset" zonasi "above tab bar" hududida joylashgan. Tarkibi: "ActiveRideBar when vm.hasActiveRoute and activeMission exist".

### Controloverview

- "Home tab" tugmasi "tab bar" hududida joylashgan. Uslub: "TabView tab item".
- "Rides tab" tugmasi "tab bar" hududida joylashgan. Uslub: "TabView tab item".
- "Profile tab" tugmasi "tab bar" hududida joylashgan. Uslub: "TabView tab item".
- "Home open-map trigger" tugmasi "home content callback" hududida joylashgan. Uslub: "screen CTA transitions to map mode".
- "ActiveRideBar" tugmasi "bottom safe-area inset" hududida joylashgan. Uslub: "floating pill CTA to map mode".
- "Map goBack" tugmasi "full-screen map mode" hududida joylashgan. Uslub: "callback-driven return control".

### Iconoverview

- "house.fill" ikonasi "Home tab" zonasida ishlatiladi.
- "list.bullet" ikonasi "Rides tab" zonasida ishlatiladi.
- "person.fill" ikonasi "Profile tab" zonasida ishlatiladi.
- "map.fill" ikonasi "Map mode logical tab target" zonasida ishlatiladi.
- "chevron.right" ikonasi "ActiveRideBar trailing affordance" zonasida ishlatiladi.

### Flowoverview

**Flowid:** home-to-map-mode

**Summary:** "home-to-map-mode" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Driver taps map CTA from HomeView".
- 2-qadam: "selectedTab switches to map with snappy animation".
- 3-qadam: "FleetMapView replaces the normal tab shell".

---

**Flowid:** active-route-drilldown

**Summary:** "active-route-drilldown" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Active route exists".
- 2-qadam: "ActiveRideBar appears above tab bar".
- 3-qadam: "Driver taps ActiveRideBar".
- 4-qadam: "Shell transitions into full-screen map mode".

---

**Flowid:** authenticated-root-branching

**Summary:** "authenticated-root-branching" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "RootView checks TokenStore.isAuthenticated".
- 2-qadam: "Authenticated drivers go directly to MainTabView".
- 3-qadam: "Unauthenticated drivers stay on LoginView".

---


### Stateoverview

- Holat: "unauthenticated login state".
- Holat: "home tab active".
- Holat: "rides tab active".
- Holat: "profile tab active".
- Holat: "active route bar visible".
- Holat: "full-screen map mode".

### Figureoverview

- Figura: "driver iOS shell with tab bar".
- Figura: "active route bar above tab bar".
- Figura: "full-screen map mode".
- Figura: "root auth-gate transition".

---

**Dossierfile:** driver-ios-secondary-surfaces.json

**Bundleid:** driver-ios-secondary-surfaces

**Appid:** driver-app-ios

**Platform:** ios

**Role:** DRIVER

**Status:** implemented

**Entrytype:** bundle

**Localizedsummary:** "driver-ios-secondary-surfaces" paketi "driver-app-ios" ilovasi uchun 9 ta yuzani qamrab oladi.

## Surfaces

**Pageid:** ios-driver-root-gate

**Viewname:** RootView

**Surfacetype:** root-gate

**Sourcefile:** apps/driverappios/driverappios/LabDriverApp.swift

**Localizedsummary:** "ios-driver-root-gate" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-driver-root-gate" yuzasi unknown-platform platformasida unknown-role roli uchun ildiz gate sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Session gate that decides between driver login flow and the protected multi-tab shell.

#### Layoutoverview

- Layout zonasi: "app bootstrap region hosting root state decision".
- Layout zonasi: "login presentation branch for unauthenticated driver".
- Layout zonasi: "main-tab branch for authenticated driver".

#### Controloverview

- Boshqaruv elementi: "no persistent buttons; state resolution presents login or shell".

#### Iconoverview

- Ikona joylashuvi: "none at gate level".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "read token and driver session state on launch".

---

**Summary:** Oqim quyidagicha qayd etilgan: "show login when token is absent or invalid".

---

**Summary:** Oqim quyidagicha qayd etilgan: "show MainTabView when session is active".

---


#### Dependencyoverview

##### Reads

- driver token store
- driver session bootstrap

##### Writes


##### Localizednotes

- O'qish: "driver token store", "driver session bootstrap".
- Yozish: yo'q.

#### Stateoverview

- Holat: "authenticated branch".
- Holat: "unauthenticated branch".

#### Figureoverview

- Figura: "route-flow figure from root gate to login or main shell".

#### Minifeatureoverview

- Mini-feature: "auth branch resolution".
- Mini-feature: "protected-shell presentation".
- Mini-feature: "guest-shell suppression".

**Minifeaturecount:** 3

---

**Pageid:** ios-driver-login

**Viewname:** LoginView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/LoginView.swift

**Localizedsummary:** "ios-driver-login" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-driver-login" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Phone and PIN sign-in screen for driver session acquisition.

#### Layoutoverview

- Layout zonasi: "brand crest and title stack".
- Layout zonasi: "phone field and PIN field cluster".
- Layout zonasi: "PIN visibility toggle inside secure entry".
- Layout zonasi: "login CTA and error message strip".

#### Controloverview

- Boshqaruv elementi: "PIN visibility eye toggle inside PIN field trailing edge".
- Boshqaruv elementi: "Login button at form footer".

#### Iconoverview

- Ikona joylashuvi: "brand disk at page top".
- Ikona joylashuvi: "eye or eye-slash in PIN field".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "input phone and PIN".

---

**Summary:** Oqim quyidagicha qayd etilgan: "toggle PIN visibility".

---

**Summary:** Oqim quyidagicha qayd etilgan: "submit DriverApi login and persist token".

---

**Summary:** Oqim quyidagicha qayd etilgan: "on success dismiss into protected shell".

---


#### Dependencyoverview

##### Reads

- DriverApi.login

##### Writes

- TokenHolder session state

##### Localizednotes

- O'qish: "DriverApi.login".
- Yozish: "TokenHolder session state".

#### Stateoverview

- Holat: "idle login form".
- Holat: "auth loading state".
- Holat: "error state".

#### Figureoverview

- Figura: "full login screen with phone and PIN fields".
- Figura: "PIN field close-up with visibility toggle".

#### Minifeatureoverview

- Mini-feature: "phone prefill".
- Mini-feature: "secure PIN entry".
- Mini-feature: "visibility toggle".
- Mini-feature: "loading disable".
- Mini-feature: "error banner".

**Minifeaturecount:** 5

---

**Pageid:** ios-driver-home

**Viewname:** HomeView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/HomeView.swift

**Localizedsummary:** "ios-driver-home" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-driver-home" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Driver dashboard summarizing mission status, truck identity, daily metrics, and quick-entry actions into execution surfaces.

#### Layoutoverview

- Layout zonasi: "dynamic greeting header".
- Layout zonasi: "truck and route status chips".
- Layout zonasi: "vehicle card and transit control card".
- Layout zonasi: "today summary metrics".
- Layout zonasi: "Open Map and quick action buttons".
- Layout zonasi: "recent activity ledger".

#### Controloverview

- Boshqaruv elementi: "Open Map CTA in transit control zone".
- Boshqaruv elementi: "Scan QR quick action button".
- Boshqaruv elementi: "View Manifest quick action button".

#### Iconoverview

- Ikona joylashuvi: "antenna or moon status icon".
- Ikona joylashuvi: "truck or route glyphs in status cards".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "load missions and driver summary on appear".

---

**Summary:** Oqim quyidagicha qayd etilgan: "pull to refresh home data".

---

**Summary:** Oqim quyidagicha qayd etilgan: "jump to map, scanner, or rides manifest".

---


#### Dependencyoverview

##### Reads

- FleetViewModel mission summary
- driver metrics
- recent activity feed

##### Writes


##### Localizednotes

- O'qish: "FleetViewModel mission summary", "driver metrics", "recent activity feed".
- Yozish: yo'q.

#### Stateoverview

- Holat: "loading home state".
- Holat: "idle off-route state".
- Holat: "on-route active state".

#### Figureoverview

- Figura: "driver home dashboard with quick action band".
- Figura: "status chip and summary card close-up".

#### Minifeatureoverview

- Mini-feature: "time-of-day greeting".
- Mini-feature: "truck plate chip".
- Mini-feature: "route-state chip".
- Mini-feature: "vehicle card".
- Mini-feature: "daily summary".
- Mini-feature: "Open Map CTA".
- Mini-feature: "Scan QR shortcut".
- Mini-feature: "View Manifest shortcut".
- Mini-feature: "recent activity list".

**Minifeaturecount:** 9

---

**Pageid:** ios-driver-map

**Viewname:** FleetMapView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/FleetMapView.swift

**Localizedsummary:** "ios-driver-map" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-driver-map" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Primary driver execution surface joining map telemetry, mission selection, QR scan initiation, payment branching, and delivery correction entry.

#### Layoutoverview

- Layout zonasi: "full-screen map region with mission markers".
- Layout zonasi: "zoom focus control cycling Me, Target, Both".
- Layout zonasi: "selected mission side or bottom detail region".
- Layout zonasi: "bottom action strip for Scan QR and Correct Delivery".
- Layout zonasi: "navigation bridge into scanner, offload, payment, cash, and correction flows".

#### Controloverview

- Boshqaruv elementi: "zoom focus cycle button over map chrome".
- Boshqaruv elementi: "Scan QR primary action in selected mission panel".
- Boshqaruv elementi: "Correct Delivery secondary action in selected mission panel".

#### Iconoverview

- Ikona joylashuvi: "mission markers on map".
- Ikona joylashuvi: "location and target glyphs in focus control".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "select mission from map marker".

---

**Summary:** Oqim quyidagicha qayd etilgan: "cycle map framing mode".

---

**Summary:** Oqim quyidagicha qayd etilgan: "launch QR scanner for selected mission".

---

**Summary:** Oqim quyidagicha qayd etilgan: "after scan branch into offload review then payment or cash collection".

---

**Summary:** Oqim quyidagicha qayd etilgan: "open correction workflow for selected mission".

---


#### Dependencyoverview

##### Reads

- TelemetryViewModel live location
- FleetViewModel missions
- geofence and QR validation payloads

##### Writes

- navigation path for execution subflows

##### Localizednotes

- O'qish: "TelemetryViewModel live location", "FleetViewModel missions", "geofence and QR validation payloads".
- Yozish: "navigation path for execution subflows".

#### Stateoverview

- Holat: "no mission selected".
- Holat: "mission previewing state".
- Holat: "active delivery state".

#### Figureoverview

- Figura: "full operational map with selected mission panel".
- Figura: "map chrome close-up with focus control and CTA band".

#### Minifeatureoverview

- Mini-feature: "live mission markers".
- Mini-feature: "mission selection".
- Mini-feature: "focus cycle control".
- Mini-feature: "selected mission detail pane".
- Mini-feature: "Scan QR CTA".
- Mini-feature: "Correct Delivery CTA".
- Mini-feature: "payment branch routing".
- Mini-feature: "cash branch routing".

**Minifeaturecount:** 8

---

**Pageid:** ios-driver-rides

**Viewname:** RidesListView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/RidesListView.swift

**Localizedsummary:** "ios-driver-rides" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-driver-rides" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Route-manifest ledger of upcoming rides with physical loading-sequence toggle.

#### Layoutoverview

- Layout zonasi: "UPCOMING header with pending count".
- Layout zonasi: "Loading Mode toggle row".
- Layout zonasi: "mission ride card list".
- Layout zonasi: "pull-to-refresh scaffold".

#### Controloverview

- Boshqaruv elementi: "Loading Mode switch in header row".
- Boshqaruv elementi: "ride card tap target to select or focus mission".

#### Iconoverview

- Ikona joylashuvi: "sequence badge when loading mode is enabled".
- Ikona joylashuvi: "status badge on ride cards".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "toggle loading mode to reverse sequence for warehouse loading".

---

**Summary:** Oqim quyidagicha qayd etilgan: "tap ride card to select mission and synchronize with map".

---

**Summary:** Oqim quyidagicha qayd etilgan: "refresh pending missions".

---


#### Dependencyoverview

##### Reads

- FleetViewModel.pendingMissions

##### Writes

- FleetViewModel.selectMission

##### Localizednotes

- O'qish: "FleetViewModel.pendingMissions".
- Yozish: "FleetViewModel.selectMission".

#### Stateoverview

- Holat: "standard route order".
- Holat: "loading-sequence order".
- Holat: "empty rides list".

#### Figureoverview

- Figura: "rides manifest screen with loading mode toggle".
- Figura: "single ride card with sequence badge".

#### Minifeatureoverview

- Mini-feature: "pending count badge".
- Mini-feature: "loading mode toggle".
- Mini-feature: "sequence badge".
- Mini-feature: "ride amount summary".
- Mini-feature: "item count summary".
- Mini-feature: "status pill".

**Minifeaturecount:** 6

---

**Pageid:** ios-driver-profile

**Viewname:** ProfileView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/ProfileView.swift

**Localizedsummary:** "ios-driver-profile" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-driver-profile" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Driver identity, truck metadata, quick operations, and ride-history review surface.

#### Layoutoverview

- Layout zonasi: "driver title header".
- Layout zonasi: "driver identity card with status pill".
- Layout zonasi: "truck and metrics info grid".
- Layout zonasi: "quick actions row".
- Layout zonasi: "ride history ledger".
- Layout zonasi: "stats section".

#### Controloverview

- Boshqaruv elementi: "Sync quick action button".
- Boshqaruv elementi: "Logout quick action button".
- Boshqaruv elementi: "Offline Verifier quick action button or sheet trigger".

#### Iconoverview

- Ikona joylashuvi: "driver initials avatar".
- Ikona joylashuvi: "status pill".
- Ikona joylashuvi: "quick action glyphs".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "sync local and live route state".

---

**Summary:** Oqim quyidagicha qayd etilgan: "open offline verifier".

---

**Summary:** Oqim quyidagicha qayd etilgan: "logout driver session".

---


#### Dependencyoverview

##### Reads

- FleetViewModel driver profile and mission history
- TelemetryService state

##### Writes

- logout session
- offline verifier sheet state

##### Localizednotes

- O'qish: "FleetViewModel driver profile and mission history", "TelemetryService state".
- Yozish: "logout session", "offline verifier sheet state".

#### Stateoverview

- Holat: "on duty".
- Holat: "idle".
- Holat: "history populated".
- Holat: "history sparse".

#### Figureoverview

- Figura: "driver profile with quick actions and stats".
- Figura: "identity card close-up".

#### Minifeatureoverview

- Mini-feature: "identity card".
- Mini-feature: "on-duty status pill".
- Mini-feature: "truck info grid".
- Mini-feature: "Sync action".
- Mini-feature: "Logout action".
- Mini-feature: "Offline Verifier access".
- Mini-feature: "ride history list".
- Mini-feature: "revenue stat".
- Mini-feature: "completed-orders stat".

**Minifeaturecount:** 9

---

**Pageid:** ios-driver-offline-verifier

**Viewname:** OfflineVerifierView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/OfflineVerifierView.swift

**Localizedsummary:** "ios-driver-offline-verifier" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-driver-offline-verifier" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Cryptographic offline verification terminal for zero-connectivity proof of delivery and fraud detection.

#### Layoutoverview

- Layout zonasi: "terminal header with protocol name".
- Layout zonasi: "protocol status band".
- Layout zonasi: "state-driven body switching among idle, syncing, ready, scanning, verified, fraud, and error cards".

#### Controloverview

- Boshqaruv elementi: "Sync Route Manifest button in idle state".
- Boshqaruv elementi: "Start Scan button in ready state".

#### Iconoverview

- Ikona joylashuvi: "scanner overlay in scanning state".
- Ikona joylashuvi: "success or fraud glyphs in verification result cards".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "sync manifest hash locally".

---

**Summary:** Oqim quyidagicha qayd etilgan: "start offline scan".

---

**Summary:** Oqim quyidagicha qayd etilgan: "verify QR against manifest hash".

---

**Summary:** Oqim quyidagicha qayd etilgan: "show verified order result or fraud reason".

---


#### Dependencyoverview

##### Reads

- OfflineDeliveryStore
- AVFoundation camera feed
- SHA256Helper manifest validation

##### Writes

- offline verification state machine

##### Localizednotes

- O'qish: "OfflineDeliveryStore", "AVFoundation camera feed", "SHA256Helper manifest validation".
- Yozish: "offline verification state machine".

#### Stateoverview

- Holat: "idle".
- Holat: "syncing".
- Holat: "ready".
- Holat: "scanning".
- Holat: "verified".
- Holat: "fraud".
- Holat: "error".

#### Figureoverview

- Figura: "offline verification terminal in ready state".
- Figura: "two-panel verified versus fraud outcome figure".

#### Minifeatureoverview

- Mini-feature: "protocol status pill".
- Mini-feature: "sync progress state".
- Mini-feature: "manifest hash display".
- Mini-feature: "Start Scan CTA".
- Mini-feature: "scanner overlay".
- Mini-feature: "verified result card".
- Mini-feature: "fraud result card".
- Mini-feature: "error result card".

**Minifeaturecount:** 8

---

**Pageid:** ios-driver-mission-detail-sheet

**Viewname:** MissionDetailSheet

**Surfacetype:** overlay

**Sourcefile:** apps/driverappios/driverappios/Views/MissionDetailSheet.swift

**Localizedsummary:** "ios-driver-mission-detail-sheet" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-driver-mission-detail-sheet" yuzasi unknown-platform platformasida unknown-role roli uchun overlay sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Bottom-sheet mission inspector exposing geofence clearance, endpoint distance, payment badge, and scan or correction actions.

#### Layoutoverview

- Layout zonasi: "order header with monospaced order ID and gateway badge".
- Layout zonasi: "delivery endpoint card with coordinates and geofence state".
- Layout zonasi: "distance and proximity indicators".
- Layout zonasi: "footer action cluster".

#### Controloverview

- Boshqaruv elementi: "Delivery Correction text button".
- Boshqaruv elementi: "Scan QR primary button".

#### Iconoverview

- Ikona joylashuvi: "geofence status dot".
- Ikona joylashuvi: "gateway badge".
- Ikona joylashuvi: "location glyph in endpoint card".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "present mission detail over map".

---

**Summary:** Oqim quyidagicha qayd etilgan: "launch correction from sheet".

---

**Summary:** Oqim quyidagicha qayd etilgan: "launch scanner from sheet".

---


#### Dependencyoverview

##### Reads

- Mission object
- distance calculation
- geofence validation state

##### Writes

- scan callback
- correction callback

##### Localizednotes

- O'qish: "Mission object", "distance calculation", "geofence validation state".
- Yozish: "scan callback", "correction callback".

#### Stateoverview

- Holat: "cleared geofence".
- Holat: "fault geofence".

#### Figureoverview

- Figura: "mission detail sheet over map backdrop".
- Figura: "endpoint card close-up with geofence state".

#### Minifeatureoverview

- Mini-feature: "monospaced order ID".
- Mini-feature: "gateway badge".
- Mini-feature: "amount display".
- Mini-feature: "geofence dot".
- Mini-feature: "distance meter".
- Mini-feature: "Delivery Correction CTA".
- Mini-feature: "Scan QR CTA".

**Minifeaturecount:** 7

---

**Pageid:** ios-driver-map-marker-detail-sheet

**Viewname:** MapMarkerDetailSheet

**Surfacetype:** overlay

**Sourcefile:** apps/driverappios/driverappios/Views/Components/MapMarkerDetailSheet.swift

**Localizedsummary:** "ios-driver-map-marker-detail-sheet" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-driver-map-marker-detail-sheet" yuzasi unknown-platform platformasida unknown-role roli uchun overlay sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Compact marker drill-down for map stops, emphasizing stop identity and route semantics without entering the full mission sheet.

#### Layoutoverview

- Layout zonasi: "marker header with stop label".
- Layout zonasi: "order or stop metadata stack".
- Layout zonasi: "micro action row".

#### Controloverview

- Boshqaruv elementi: "compact dismiss or expand controls inside overlay".

#### Iconoverview

- Ikona joylashuvi: "marker glyph".
- Ikona joylashuvi: "status dot or stop-type icon".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "tap map marker to open compact stop detail".

---

**Summary:** Oqim quyidagicha qayd etilgan: "dismiss or escalate into fuller mission inspection".

---


#### Dependencyoverview

##### Reads

- selected map marker payload

##### Writes

- marker-detail dismissal or expansion state

##### Localizednotes

- O'qish: "selected map marker payload".
- Yozish: "marker-detail dismissal or expansion state".

#### Stateoverview

- Holat: "compact marker summary".
- Holat: "expanded marker context".

#### Figureoverview

- Figura: "map marker detail overlay figure".

#### Minifeatureoverview

- Mini-feature: "marker label".
- Mini-feature: "stop metadata lines".
- Mini-feature: "compact overlay".
- Mini-feature: "expand affordance".

**Minifeaturecount:** 4

---


---

**Dossierfile:** payload-auth-loading.json

**Pageid:** payload-auth-loading

**State:** authLoading

**Platform:** react-native-tablet

**Role:** PAYLOAD

**Status:** implemented

**Shell:** payload-terminal-state-shell

## Sourcefiles

- apps/payload-terminal/App.tsx

**Entrytype:** page

**Localizedsummary:** "payload-auth-loading" yuzasi uchun payload operatori roli va React Native tablet platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "payload-auth-loading" yuzasi React Native tablet platformasida payload operatori roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Payload terminal session-restore state shown while SecureStore token and worker context are being recovered at app startup.

### Layoutoverview

- "restore-center" zonasi "centered full-screen state" hududida joylashgan. Tarkibi: "restoring session text".

### Controloverview


### Iconoverview


### Flowoverview

**Flowid:** session-restore

**Summary:** "session-restore" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "App locks screen to landscape".
- 2-qadam: "App reads payloader token, name, and supplier ID from SecureStore".
- 3-qadam: "App exits authLoading into login or authenticated state".

---


### Stateoverview

- Holat: "restoring-session state".

### Figureoverview

- Figura: "payload auth restore splash state".

---

**Dossierfile:** payload-dispatch-success.json

**Pageid:** payload-dispatch-success

**State:** allSealed == true

**Platform:** react-native-tablet

**Role:** PAYLOAD

**Status:** implemented

**Shell:** payload-terminal-state-shell

## Sourcefiles

- apps/payload-terminal/App.tsx

**Entrytype:** page

**Localizedsummary:** "payload-dispatch-success" yuzasi uchun payload operatori roli va React Native tablet platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "payload-dispatch-success" yuzasi React Native tablet platformasida payload operatori roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Payload terminal dispatch-complete success state confirming manifest sealing and exposing dispatch codes before starting a new manifest.

### Layoutoverview

- "success-center" zonasi "centered body" hududida joylashgan. Tarkibi: "active truck mono label"; "Manifest Secured headline"; "Fleet Dispatched headline".
- "dispatch-code-panel" zonasi "center body below headlines" hududida joylashgan. Ko'rinish qoidasi: "visible when dispatchCodes has entries". Tarkibi: "Dispatch Codes heading"; "rows of order ID to code pairs".
- "new-manifest-action" zonasi "below code panel" hududida joylashgan. Tarkibi: "New Manifest outlined button".

### Controloverview

- "New Manifest" tugmasi "new-manifest-action" hududida joylashgan. Uslub: "outlined button".

### Iconoverview


### Flowoverview

**Flowid:** dispatch-complete-reset

**Summary:** "dispatch-complete-reset" oqimi 5 ta qadamdan iborat.

#### Steps

- 1-qadam: "All loaded orders on the truck are sealed".
- 2-qadam: "App enters success state".
- 3-qadam: "Worker reviews dispatch codes if present".
- 4-qadam: "Worker taps New Manifest".
- 5-qadam: "App clears activeTruck, allSealed, and dispatchCodes, then returns to truck selection".

---


### Stateoverview

- Holat: "success state without dispatch codes".
- Holat: "success state with dispatch code panel".

### Figureoverview

- Figura: "payload dispatch success state".
- Figura: "dispatch code panel close-up".

---

**Dossierfile:** payload-login.json

**Pageid:** payload-login

**State:** token == null

**Platform:** react-native-tablet

**Role:** PAYLOAD

**Status:** implemented

**Shell:** payload-terminal-state-shell

## Sourcefiles

- apps/payload-terminal/App.tsx

**Entrytype:** page

**Localizedsummary:** "payload-login" yuzasi uchun payload operatori roli va React Native tablet platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "payload-login" yuzasi React Native tablet platformasida payload operatori roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Payload worker sign-in state for tablet authentication via phone number and 6-digit PIN.

### Layoutoverview

- "brand-header" zonasi "top centered column" hududida joylashgan. Tarkibi: "Pegasus Payload Terminal label"; "Payloader Login headline".
- "credential-stack" zonasi "centered form column" hududida joylashgan. Tarkibi: "phone number input"; "6-digit PIN input with centered wide letter spacing"; "Sign In button".

### Controloverview

- "Sign In" tugmasi "credential-stack footer" hududida joylashgan. Uslub: "full-width filled button".

### Iconoverview


### Flowoverview

**Flowid:** payloader-login

**Summary:** "payloader-login" oqimi 5 ta qadamdan iborat.

#### Steps

- 1-qadam: "Worker enters phone number and PIN".
- 2-qadam: "Worker taps Sign In".
- 3-qadam: "App posts to /v1/auth/payloader/login".
- 4-qadam: "Successful response persists token, name, and supplier ID in SecureStore".
- 5-qadam: "App advances into truck selection state".

---


### Stateoverview

- Holat: "idle login state".
- Holat: "authenticating state".
- Holat: "login failure alert".

### Figureoverview

- Figura: "payload login state".
- Figura: "payload login authenticating CTA state".

---

**Dossierfile:** payload-manifest-workspace.json

**Pageid:** payload-manifest-workspace

**State:** token != null && activeTruck != null && allSealed == false

**Platform:** react-native-tablet

**Role:** PAYLOAD

**Status:** implemented

**Shell:** payload-terminal-state-shell

**Sourcefile:** apps/payload-terminal/App.tsx

**Entrytype:** page

**Localizedsummary:** "payload-manifest-workspace" yuzasi uchun payload operatori roli va React Native tablet platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "payload-manifest-workspace" yuzasi React Native tablet platformasida payload operatori roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Warehouse payloader tablet workspace for selecting orders on a truck, checklist scanning line items, and sealing the load for dispatch.

### Layoutoverview

- "left-pane" zonasi "fixed-width left column" hududida joylashgan. Tarkibi: "terminal header with title and active truck"; "truck toggle bar"; "scrollable order list with active and cleared states".
- "right-header" zonasi "top of right pane" hududida joylashgan. Tarkibi: "selected order ID"; "retailer ID, payment gateway, amount text line"; "truck badge chip".
- "checklist-region" zonasi "center right pane" hududida joylashgan. Tarkibi: "scrollable manifest checklist"; "tap-to-toggle checkbox control"; "brand code line"; "item label line".
- "seal-footer" zonasi "bottom of right pane" hududida joylashgan. Tarkibi: "Mark as Loaded action button".

### Controloverview

- "truck selector in left-pane toggle bar" tugmasi "left-pane truck toggle row" hududida joylashgan. Uslub: "segmented text button".
- "order selector" tugmasi "left-pane order list row" hududida joylashgan. Uslub: "list row button". Ko'rinish qoidasi: "disabled for sealed orders".
- "manifest item checkbox row" tugmasi "checklist-region" hududida joylashgan. Uslub: "full-row toggle".
- "Mark as Loaded" tugmasi "seal-footer" hududida joylashgan. Uslub: "primary footer CTA". Ko'rinish qoidasi: "enabled only when all selected-order checklist items are checked and not currently sealing".

### Iconoverview

- "text-only checkmark glyph" ikonasi "checkbox control when item is scanned" zonasida ishlatiladi.

### Flowoverview

**Flowid:** switch-active-truck

**Summary:** "switch-active-truck" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Payload operator taps truck label in left-pane toggle row".
- 2-qadam: "handleTruckSelect resets local manifest state".
- 3-qadam: "fetchManifest reloads orders and checklist for that truck".

---

**Flowid:** select-order-and-clear-items

**Summary:** "select-order-and-clear-items" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Payload operator taps order row in left pane".
- 2-qadam: "Right pane updates selected order header and checklist".
- 3-qadam: "Operator taps each checklist row to toggle scanned state".
- 4-qadam: "Checkbox fills accent color with checkmark when scanned".

---

**Flowid:** seal-order

**Summary:** "seal-order" oqimi 5 ta qadamdan iborat.

#### Steps

- 1-qadam: "All checklist items for selected order are scanned".
- 2-qadam: "Mark as Loaded becomes enabled".
- 3-qadam: "Operator taps Mark as Loaded".
- 4-qadam: "App posts to /v1/payload/seal".
- 5-qadam: "Order is added to sealedOrderIds and next remaining order is auto-selected or allSealed becomes true".

---


### Dependencyoverview

#### Reads

- /v1/payloader/trucks
- /v1/payloader/orders?vehicle_id={truckId}&state=LOADED

#### Writes

- /v1/payload/seal

#### Localizednotes

- O'qish: "/v1/payloader/trucks", "/v1/payloader/orders?vehicle_id={truckId}&state=LOADED".
- Yozish: "/v1/payload/seal".
- Offline fallback: "manifest fetch attempts SecureStore cache keyed by manifest_{truckId}".

### Stateoverview

- Holat: "manifest loading in left pane".
- Holat: "no pending orders in left pane".
- Holat: "active order row styling".
- Holat: "cleared order row styling".
- Holat: "selected order right-pane detail".
- Holat: "no selected order placeholder".
- Holat: "sealing disabled footer".
- Holat: "sealing in-progress footer".

### Figureoverview

- Figura: "full two-pane manifest workspace".
- Figura: "left-pane truck selector and order list".
- Figura: "right-pane order header".
- Figura: "checklist row with checkbox state".
- Figura: "seal footer CTA".

---

**Dossierfile:** payload-truck-selection.json

**Pageid:** payload-truck-selection

**State:** token != null && activeTruck == null

**Platform:** react-native-tablet

**Role:** PAYLOAD

**Status:** implemented

**Shell:** payload-terminal-state-shell

## Sourcefiles

- apps/payload-terminal/App.tsx

**Entrytype:** page

**Localizedsummary:** "payload-truck-selection" yuzasi uchun payload operatori roli va React Native tablet platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "payload-truck-selection" yuzasi React Native tablet platformasida payload operatori roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Payload tablet vehicle-selection state for choosing the target truck before loading a manifest.

### Layoutoverview

- "header-bar" zonasi "top full-width" hududida joylashgan. Tarkibi: "terminal title"; "worker name"; "Sign Out action".
- "selection-center" zonasi "centered body" hududida joylashgan. Tarkibi: "Select Target Vehicle label"; "vehicle card row with label, license plate, and vehicle class"; "loading-or-empty helper text".

### Controloverview

- "Sign Out" tugmasi "header-bar right" hududida joylashgan. Uslub: "text action".
- "truck card" tugmasi "selection-center" hududida joylashgan. Uslub: "card button".

### Iconoverview


### Flowoverview

**Flowid:** truck-selection

**Summary:** "truck-selection" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Authenticated worker waits for /v1/payloader/trucks to populate available vehicles".
- 2-qadam: "Worker taps a truck card".
- 3-qadam: "handleTruckSelect sets activeTruck and triggers manifest fetch".
- 4-qadam: "App transitions into manifest workspace".

---

**Flowid:** logout-from-selector

**Summary:** "logout-from-selector" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Worker taps Sign Out".
- 2-qadam: "SecureStore credentials are cleared".
- 3-qadam: "App returns to payload login state".

---


### Stateoverview

- Holat: "vehicle cards available".
- Holat: "no vehicles available".
- Holat: "loading vehicles helper text".

### Figureoverview

- Figura: "payload truck selection state".
- Figura: "payload truck card close-up".

---

**Dossierfile:** retailer-android-active-deliveries-sheet.json

**Pageid:** android-retailer-active-deliveries

**Navroute:** ActiveDeliveriesSheet

**Platform:** android

**Role:** RETAILER

**Status:** implemented

**Shell:** retailer-android-overlay

## Sourcefiles

- apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/ActiveDeliveriesSheet.kt

**Entrytype:** page

**Localizedsummary:** "android-retailer-active-deliveries" yuzasi uchun chakana savdogar roli va android platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "android-retailer-active-deliveries" yuzasi android platformasida chakana savdogar roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Android retailer active-deliveries bottom sheet listing in-progress orders with detail and QR actions.

### Layoutoverview

- "sheet-header" zonasi "top of modal bottom sheet" hududida joylashgan. Tarkibi: "Active Deliveries title"; "order count subtitle"; "Done action".
- "delivery-card-list" zonasi "sheet body" hududida joylashgan. Tarkibi: "active delivery cards with progress ring, order metadata, countdown row, and action buttons".

### Controloverview

- "Done" tugmasi "sheet-header trailing action" hududida joylashgan. Uslub: "text button".
- "Details" tugmasi "delivery card action row" hududida joylashgan. Uslub: "pill button".
- "Show QR" tugmasi "delivery card action row" hududida joylashgan. Uslub: "primary pill". Ko'rinish qoidasi: "order has delivery token".

### Iconoverview

- "progress ring" ikonasi "delivery card leading visual" zonasida ishlatiladi.
- "QrCode2" ikonasi "Show QR action or awaiting-dispatch status pill" zonasida ishlatiladi.
- "CountdownTimer" ikonasi "countdown row" zonasida ishlatiladi.

### Flowoverview

**Flowid:** delivery-sheet-review

**Summary:** "delivery-sheet-review" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer opens active deliveries sheet".
- 2-qadam: "Retailer reviews each active-delivery card".
- 3-qadam: "Retailer opens details or QR flow from a selected card".
- 4-qadam: "Retailer dismisses sheet with Done".

---


### Stateoverview

- Holat: "active deliveries sheet open".
- Holat: "delivery card with QR enabled".
- Holat: "delivery card awaiting dispatch".

### Figureoverview

- Figura: "android retailer active deliveries sheet".
- Figura: "delivery card with countdown and QR action".

---

**Dossierfile:** retailer-android-auth.json

**Pageid:** android-retailer-auth

**Navroute:** AuthScreen

**Platform:** android

**Role:** RETAILER

**Status:** implemented

**Shell:** retailer-android-auth

## Sourcefiles

- apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/auth/AuthScreen.kt

**Entrytype:** page

**Localizedsummary:** "android-retailer-auth" yuzasi uchun chakana savdogar roli va android platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "android-retailer-auth" yuzasi android platformasida chakana savdogar roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Android retailer authentication and registration screen with expandable onboarding fields, map and GPS location capture, and logistics profile collection.

### Layoutoverview

- "brand-stack" zonasi "top centered column" hududida joylashgan. Tarkibi: "black storefront icon disk"; "Pegasus title"; "Retailer Portal subtitle".
- "credential-core" zonasi "main form column" hududida joylashgan. Tarkibi: "phone field"; "password field".
- "registration-extension" zonasi "below core credentials when login mode is off" hududida joylashgan. Ko'rinish qoidasi: "visible when isLoginMode is false". Tarkibi: "store name field"; "owner name field"; "address field"; "Open Map button"; "Use GPS button"; "selected location label"; "tax ID field"; "receiving window fields"; "access type chip buttons"; "ceiling height field".
- "primary-action-region" zonasi "below form fields" hududida joylashgan. Tarkibi: "Sign In or Create Account button"; "error text when present"; "mode-toggle text button".

### Controloverview

- "Open Map" tugmasi "registration-extension location row" hududida joylashgan. Uslub: "outlined button".
- "Use GPS" tugmasi "registration-extension location row" hududida joylashgan. Uslub: "outlined button".
- "Street" tugmasi "registration-extension access row" hududida joylashgan. Uslub: "outlined chip toggle".
- "Alley" tugmasi "registration-extension access row" hududida joylashgan. Uslub: "outlined chip toggle".
- "Dock" tugmasi "registration-extension access row" hududida joylashgan. Uslub: "outlined chip toggle".
- "Sign In" tugmasi "primary-action-region" hududida joylashgan. Uslub: "full-width filled pill". Ko'rinish qoidasi: "isLoginMode true".
- "Create Account" tugmasi "primary-action-region" hududida joylashgan. Uslub: "full-width filled pill". Ko'rinish qoidasi: "isLoginMode false".
- "mode toggle" tugmasi "primary-action-region footer" hududida joylashgan. Uslub: "text button".

### Iconoverview

- "Storefront" ikonasi "brand-stack" zonasida ishlatiladi.
- "Map" ikonasi "Open Map button" zonasida ishlatiladi.
- "MyLocation" ikonasi "Use GPS button" zonasida ishlatiladi.
- "CircularProgressIndicator" ikonasi "Use GPS button when locating and primary CTA when state.isLoading" zonasida ishlatiladi.

### Flowoverview

**Flowid:** login-flow

**Summary:** "login-flow" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer enters phone and password".
- 2-qadam: "Retailer taps Sign In".
- 3-qadam: "AuthViewModel authenticates and onAuthenticated advances into the main shell".

---

**Flowid:** registration-flow

**Summary:** "registration-flow" oqimi 5 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer switches out of login mode".
- 2-qadam: "AnimatedVisibility expands onboarding fields".
- 3-qadam: "Retailer captures location by map picker or GPS".
- 4-qadam: "Retailer submits Create Account".
- 5-qadam: "AuthViewModel sends registration payload with logistics fields".

---


### Stateoverview

- Holat: "login mode".
- Holat: "registration mode".
- Holat: "GPS locating state".
- Holat: "error text state".
- Holat: "loading CTA state".
- Holat: "map picker route handoff".

### Figureoverview

- Figura: "android retailer login mode".
- Figura: "android retailer registration mode".
- Figura: "location capture row".
- Figura: "loading and error state".

---

**Dossierfile:** retailer-android-cart.json

**Pageid:** android-retailer-cart

**Navroute:** CART

**Platform:** android

**Role:** RETAILER

**Status:** implemented

**Shell:** retailer-android-root

## Sourcefiles

- apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/cart/CartScreen.kt

**Entrytype:** page

**Localizedsummary:** "android-retailer-cart" yuzasi uchun chakana savdogar roli va android platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "android-retailer-cart" yuzasi android platformasida chakana savdogar roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Android retailer cart screen with list-based basket control, checkout sheet launch, supplier-closed guard dialog, and empty-cart branch.

### Layoutoverview

- "cart-list-region" zonasi "main list when items exist" hududida joylashgan. Tarkibi: "item count header"; "Clear All text button"; "cart item cards with placeholder image, size and pack pills, price, and quantity stepper".
- "bottom-bar" zonasi "sticky bottom bar" hududida joylashgan. Ko'rinish qoidasi: "visible when cart is not empty". Tarkibi: "subtotal row"; "delivery row"; "total cluster"; "Checkout surface button".
- "checkout-sheet" zonasi "modal bottom sheet" hududida joylashgan. Ko'rinish qoidasi: "visible when uiState.showCheckout is true". Tarkibi: "CheckoutSheet overlay".
- "supplier-closed-dialog" zonasi "alert dialog" hududida joylashgan. Ko'rinish qoidasi: "visible when showSupplierClosedDialog is true". Tarkibi: "warning title"; "supplier closed message"; "I Understand, Place Order button"; "Cancel button".
- "empty-state" zonasi "center body" hududida joylashgan. Ko'rinish qoidasi: "visible when uiState.isEmpty is true". Tarkibi: "double-ring shopping cart illustration"; "empty headline"; "helper copy"; "Browse Catalog button".

### Controloverview

- "Clear All" tugmasi "cart-list-region header-right" hududida joylashgan. Uslub: "text destructive".
- "quantity decrement" tugmasi "each item stepper" hududida joylashgan. Uslub: "icon button".
- "quantity increment" tugmasi "each item stepper" hududida joylashgan. Uslub: "icon button".
- "Checkout" tugmasi "bottom-bar right" hududida joylashgan. Uslub: "filled pill surface".
- "I Understand, Place Order" tugmasi "supplier-closed-dialog confirm action" hududida joylashgan. Uslub: "filled button".
- "Browse Catalog" tugmasi "empty-state" hududida joylashgan. Uslub: "filled pill surface".

### Iconoverview

- "Eco" ikonasi "cart item placeholder" zonasida ishlatiladi.
- "Delete or Remove" ikonasi "quantity decrement control" zonasida ishlatiladi.
- "Add" ikonasi "quantity increment control" zonasida ishlatiladi.
- "ArrowForward" ikonasi "Checkout CTA trailing icon" zonasida ishlatiladi.
- "ShoppingCart" ikonasi "empty-state hero" zonasida ishlatiladi.
- "GridView" ikonasi "Browse Catalog CTA" zonasida ishlatiladi.

### Flowoverview

**Flowid:** basket-editing

**Summary:** "basket-editing" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer increments or decrements quantities from each item row".
- 2-qadam: "quantity, item total, and summary totals update".
- 3-qadam: "decrement icon changes to delete when quantity reaches one".

---

**Flowid:** checkout-gating

**Summary:** "checkout-gating" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer taps Checkout".
- 2-qadam: "if supplier is closed, alert dialog interposes".
- 3-qadam: "otherwise CheckoutSheet opens immediately".

---


### Stateoverview

- Holat: "populated cart state".
- Holat: "checkout sheet active".
- Holat: "supplier closed dialog".
- Holat: "empty cart state".
- Holat: "snackbar feedback state".

### Figureoverview

- Figura: "android retailer cart populated state".
- Figura: "cart bottom bar".
- Figura: "supplier closed dialog".
- Figura: "empty cart state".

---

**Dossierfile:** retailer-android-catalog.json

**Pageid:** android-retailer-catalog

**Navroute:** CATALOG

**Platform:** android

**Role:** RETAILER

**Status:** implemented

**Shell:** retailer-android-root

## Sourcefiles

- apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/catalog/CatalogScreen.kt

**Entrytype:** page

**Localizedsummary:** "android-retailer-catalog" yuzasi uchun chakana savdogar roli va android platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "android-retailer-catalog" yuzasi android platformasida chakana savdogar roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Android retailer catalog surface combining search-driven product discovery with a mixed-scale bento category browser.

### Layoutoverview

- "search-field" zonasi "top full-width" hududida joylashgan. Tarkibi: "outlined pill search field"; "Search icon".
- "search-results-grid" zonasi "main body when query length is at least two and results exist" hududida joylashgan. Ko'rinish qoidasi: "visible when searchQuery length >= 2 and filteredProducts not empty". Tarkibi: "two-column ProductCard grid".
- "category-bento-list" zonasi "main body when search branch inactive" hududida joylashgan. Tarkibi: "Categories header with count"; "rows of large, wide, compact, and remainder category cards".

### Controloverview

- "category card" tugmasi "category-bento-list" hududida joylashgan. Uslub: "surface tap target".
- "product card" tugmasi "search-results-grid" hududida joylashgan. Uslub: "card tap target".

### Iconoverview

- "Search" ikonasi "search-field leading edge" zonasida ishlatiladi.
- "Inventory2 or category glyph" ikonasi "category cards" zonasida ishlatiladi.

### Flowoverview

**Flowid:** category-navigation

**Summary:** "category-navigation" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer taps a category bento card".
- 2-qadam: "Catalog routes to category-specific supplier or product inventory".

---

**Flowid:** search-navigation

**Summary:** "search-navigation" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer types at least two characters".
- 2-qadam: "Catalog switches to filtered product grid".
- 3-qadam: "Retailer taps a product card".
- 4-qadam: "Catalog routes to product detail".

---


### Stateoverview

- Holat: "category bento state".
- Holat: "search results state".
- Holat: "empty search branch fallback to categories".

### Figureoverview

- Figura: "android retailer category bento layout".
- Figura: "android retailer search results grid".

---

**Dossierfile:** retailer-android-checkout-sheet.json

**Pageid:** android-retailer-checkout-sheet

**Navroute:** CheckoutSheet

**Platform:** android

**Role:** RETAILER

**Status:** implemented

**Shell:** retailer-android-overlay

## Sourcefiles

- apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/CheckoutSheet.kt

**Entrytype:** page

**Localizedsummary:** "android-retailer-checkout-sheet" yuzasi uchun chakana savdogar roli va android platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "android-retailer-checkout-sheet" yuzasi android platformasida chakana savdogar roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Android retailer checkout bottom sheet for reviewing order totals, selecting payment gateway from a split buy control, and showing processing or completion phases.

### Layoutoverview

- "sheet-header" zonasi "top of modal bottom sheet" hududida joylashgan. Tarkibi: "Order details title"; "linear progress bar in review phase".
- "review-phase" zonasi "sheet body when phase is REVIEW" hududida joylashgan. Tarkibi: "product recap card with placeholder image"; "subtotal, shipping, discount, and total rows"; "Payment Method label"; "split Buy button with payment dropdown segment".
- "processing-phase" zonasi "sheet body when phase is PROCESSING" hududida joylashgan. Tarkibi: "CircularProgressIndicator"; "Processing payment text".
- "complete-phase" zonasi "sheet body when phase is COMPLETE" hududida joylashgan. Tarkibi: "check icon"; "Payment complete text".

### Controloverview

- "Buy" tugmasi "review-phase bottom control row" hududida joylashgan. Uslub: "left segment of split CTA".
- "payment dropdown segment" tugmasi "review-phase bottom control row" hududida joylashgan. Uslub: "right segment of split CTA".
- "payment option" tugmasi "dropdown menu" hududida joylashgan. Uslub: "menu row".

### Iconoverview

- "Eco" ikonasi "review-phase placeholder image" zonasida ishlatiladi.
- "Payment" ikonasi "Buy segment" zonasida ishlatiladi.
- "KeyboardArrowDown" ikonasi "dropdown segment" zonasida ishlatiladi.
- "Check" ikonasi "complete phase and selected dropdown option" zonasida ishlatiladi.
- "CircularProgressIndicator" ikonasi "processing phase" zonasida ishlatiladi.

### Flowoverview

**Flowid:** gateway-selection

**Summary:** "gateway-selection" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer taps dropdown segment".
- 2-qadam: "DropdownMenu opens with payment options".
- 3-qadam: "Retailer selects gateway and label updates".

---

**Flowid:** buy-processing-complete

**Summary:** "buy-processing-complete" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer taps Buy".
- 2-qadam: "Sheet phase transitions from REVIEW to PROCESSING".
- 3-qadam: "On success the sheet renders COMPLETE state".

---


### Stateoverview

- Holat: "review phase".
- Holat: "dropdown open state".
- Holat: "processing phase".
- Holat: "complete phase".

### Figureoverview

- Figura: "android retailer checkout review sheet".
- Figura: "split buy and payment dropdown control".
- Figura: "processing sheet".
- Figura: "complete sheet".

---

**Dossierfile:** retailer-android-orders.json

**Pageid:** android-retailer-orders

**Navroute:** ORDERS

**Platform:** android

**Role:** RETAILER

**Status:** implemented

**Shell:** retailer-android-root

## Sourcefiles

- apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/orders/OrdersScreen.kt

**Entrytype:** page

**Localizedsummary:** "android-retailer-orders" yuzasi uchun chakana savdogar roli va android platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "android-retailer-orders" yuzasi android platformasida chakana savdogar roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Android retailer orders hub with tabbed pager content, pull-to-refresh, detail-sheet drilldown, and QR overlay access for live deliveries.

### Layoutoverview

- "tab-row" zonasi "top full-width" hududida joylashgan. Tarkibi: "Active tab with Inventory2 icon"; "Ordered tab with Receipt icon"; "AI Planned tab with AutoAwesome icon".
- "pager-region" zonasi "main body" hududida joylashgan. Tarkibi: "ActiveOrdersList"; "OrderedList"; "AiPlannedList".
- "detail-sheet" zonasi "overlay sheet" hududida joylashgan. Ko'rinish qoidasi: "visible when selectedOrder is non-null". Tarkibi: "OrderDetailSheet".
- "qr-overlay" zonasi "overlay" hududida joylashgan. Ko'rinish qoidasi: "visible when qrOrder is non-null". Tarkibi: "QROverlay".

### Controloverview

- "tab" tugmasi "tab-row" hududida joylashgan. Uslub: "tab selector".
- "Details" tugmasi "active and ordered card action rows" hududida joylashgan. Uslub: "pill button".
- "Show QR" tugmasi "active card action row" hududida joylashgan. Uslub: "primary pill".
- "Cancel" tugmasi "ordered card action row" hududida joylashgan. Uslub: "destructive pill".

### Iconoverview

- "Inventory2" ikonasi "Active tab and empty state" zonasida ishlatiladi.
- "Receipt" ikonasi "Ordered tab and empty state" zonasida ishlatiladi.
- "AutoAwesome" ikonasi "AI Planned tab and empty state" zonasida ishlatiladi.
- "QrCode2" ikonasi "Show QR action" zonasida ishlatiladi.

### Flowoverview

**Flowid:** tabbed-order-review

**Summary:** "tabbed-order-review" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer switches between tabs in TabRow".
- 2-qadam: "HorizontalPager swaps the associated list view".

---

**Flowid:** order-drilldown-and-qr

**Summary:** "order-drilldown-and-qr" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer opens OrderDetailSheet from a card".
- 2-qadam: "Retailer may open QROverlay for dispatch-ready orders".
- 3-qadam: "Pending orders can also be cancelled from Ordered cards or sheet actions".

---


### Stateoverview

- Holat: "active list".
- Holat: "ordered list".
- Holat: "AI planned list".
- Holat: "pull-to-refresh state".
- Holat: "detail sheet open".
- Holat: "QR overlay visible".
- Holat: "empty lists".

### Figureoverview

- Figura: "android retailer orders active tab".
- Figura: "android retailer orders ordered tab".
- Figura: "AI planned forecast card".
- Figura: "orders QR overlay".

---

**Dossierfile:** retailer-android-payment-sheet.json

**Pageid:** android-retailer-payment-sheet

**Navroute:** DeliveryPaymentSheet

**Platform:** android

**Role:** RETAILER

**Status:** implemented

**Shell:** retailer-android-overlay

## Sourcefiles

- apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/DeliveryPaymentSheet.kt

**Entrytype:** page

**Localizedsummary:** "android-retailer-payment-sheet" yuzasi uchun chakana savdogar roli va android platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "android-retailer-payment-sheet" yuzasi android platformasida chakana savdogar roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Android retailer payment-required bottom sheet for choosing payment path after delivery, waiting for cash confirmation or card settlement, and resolving success or failure.

### Layoutoverview

- "choose-phase" zonasi "sheet body when phase is CHOOSE" hududida joylashgan. Tarkibi: "payments icon disk"; "amount due stack with optional struck original amount"; "cash option row"; "card gateway option rows".
- "processing-phase" zonasi "sheet body when phase is PROCESSING" hududida joylashgan. Tarkibi: "progress indicator"; "Processing headline"; "connection helper text".
- "cash-pending-phase" zonasi "sheet body when phase is CASH_PENDING" hududida joylashgan. Tarkibi: "cash icon disk"; "Cash Collection Pending headline"; "amount text"; "waiting chip with progress indicator".
- "success-phase" zonasi "sheet body when phase is SUCCESS" hududida joylashgan. Tarkibi: "success icon disk"; "Payment Complete headline"; "amount text"; "Done button".
- "failed-phase" zonasi "sheet body when phase is FAILED" hududida joylashgan. Tarkibi: "failure icon disk"; "Payment Failed headline"; "error message"; "Retry button"; "Cancel outlined button".

### Controloverview

- "Cash on Delivery" tugmasi "choose-phase" hududida joylashgan. Uslub: "option row".
- "card gateway option" tugmasi "choose-phase" hududida joylashgan. Uslub: "option row".
- "Done" tugmasi "success phase footer" hududida joylashgan. Uslub: "full-width primary".
- "Retry" tugmasi "failed phase footer" hududida joylashgan. Uslub: "full-width primary".
- "Cancel" tugmasi "failed phase footer" hududida joylashgan. Uslub: "full-width outlined".

### Iconoverview

- "Payments" ikonasi "choose-phase hero" zonasida ishlatiladi.
- "LocalAtm" ikonasi "cash option and cash-pending hero" zonasida ishlatiladi.
- "CreditCard" ikonasi "card option rows" zonasida ishlatiladi.
- "Check" ikonasi "success hero" zonasida ishlatiladi.
- "Close" ikonasi "failed hero" zonasida ishlatiladi.
- "CircularProgressIndicator" ikonasi "processing and cash-pending states" zonasida ishlatiladi.

### Flowoverview

**Flowid:** cash-route

**Summary:** "cash-route" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer chooses cash option".
- 2-qadam: "Sheet enters cash-pending state".
- 3-qadam: "Retailer waits for driver-side confirmation".

---

**Flowid:** card-route

**Summary:** "card-route" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer chooses card gateway row".
- 2-qadam: "Sheet enters processing state".
- 3-qadam: "External or backend-driven payment settlement updates the phase to success or failed".

---

**Flowid:** failure-recovery

**Summary:** "failure-recovery" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Sheet enters FAILED state".
- 2-qadam: "Retailer retries or dismisses".

---


### Stateoverview

- Holat: "choose phase".
- Holat: "processing phase".
- Holat: "cash pending phase".
- Holat: "success phase".
- Holat: "failed phase".

### Figureoverview

- Figura: "android retailer payment choose phase".
- Figura: "android retailer payment cash pending phase".
- Figura: "android retailer payment success phase".
- Figura: "android retailer payment failed phase".

---

**Dossierfile:** retailer-android-root-shell.json

**Pageid:** android-retailer-root-shell

**Navroute:** RetailerNavigation

**Platform:** android

**Role:** RETAILER

**Status:** implemented

**Shell:** retailer-android-root

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/navigation/RetailerNavigation.kt

**Entrytype:** page

**Localizedsummary:** "android-retailer-root-shell" yuzasi uchun chakana savdogar roli va android platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "android-retailer-root-shell" yuzasi android platformasida chakana savdogar roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Authenticated retailer shell that anchors primary navigation, active-order visibility, and global payment, QR, detail, and sidebar overlays.

### Layoutoverview

- "top-bar" zonasi "top full-width" hududida joylashgan. Tarkibi: "left: avatar circle button"; "center: Pegasus title"; "right: cart icon button with badge; notification icon button with badge".
- "content-navhost" zonasi "center full-width" hududida joylashgan. Tarkibi: "HOME dashboard"; "CATALOG"; "ORDERS"; "PROFILE"; "SUPPLIERS"; "CART"; "ANALYTICS"; "AUTO_ORDER"; "PRODUCT_DETAIL"; "CATEGORY_SUPPLIERS"; "SUPPLIER_CATEGORY_CATALOG".
- "bottom-stack" zonasi "bottom full-width" hududida joylashgan. Tarkibi: "FloatingActiveOrdersBar"; "LabBottomBar".
- "global-overlays" zonasi "above shell content" hududida joylashgan. Tarkibi: "ActiveDeliveriesSheet"; "OrderDetailSheet"; "QROverlay"; "SidebarMenu"; "DeliveryPaymentSheet".

### Controloverview

- "avatar button" tugmasi "top-bar-left" hududida joylashgan. Uslub: "circular filled button".
- "cart button" tugmasi "top-bar-right" hududida joylashgan. Uslub: "icon button with badge".
- "notification button" tugmasi "top-bar-right" hududida joylashgan. Uslub: "icon button with badge".
- "bottom nav tabs" tugmasi "bottom-stack lower row" hududida joylashgan. Uslub: "NavigationBarItem set of five".
- "floating active orders bar" tugmasi "bottom-stack upper row" hududida joylashgan. Uslub: "full-width floating card CTA".
- "sidebar menu rows" tugmasi "sidebar overlay vertical list" hududida joylashgan. Uslub: "full-width menu row buttons".

### Iconoverview

- "Outlined.ShoppingCart" ikonasi "top-bar cart action" zonasida ishlatiladi.
- "Outlined.Notifications" ikonasi "top-bar notification action" zonasida ishlatiladi.
- "Home/Store/Inventory2/AccountCircle/Person" ikonasi "bottom nav" zonasida ishlatiladi.
- "LocalShipping" ikonasi "floating active orders progress ring center" zonasida ishlatiladi.
- "KeyboardArrowUp" ikonasi "floating active orders expand affordance" zonasida ishlatiladi.
- "GridView/BarChart/Insights/AutoAwesome/Inbox/Person/Settings/ExitToApp" ikonasi "sidebar rows" zonasida ishlatiladi.

### Flowoverview

**Flowid:** global-order-attention

**Summary:** "global-order-attention" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer sees active-order summary in floating bar".
- 2-qadam: "Retailer taps floating bar".
- 3-qadam: "ActiveDeliveriesSheet opens".
- 4-qadam: "Retailer can drill into detail or QR overlay".

---

**Flowid:** websocket-payment-resolution

**Summary:** "websocket-payment-resolution" oqimi 5 ta qadamdan iborat.

#### Steps

- 1-qadam: "NavigationViewModel receives PAYMENT_REQUIRED event".
- 2-qadam: "DeliveryPaymentSheet renders".
- 3-qadam: "Retailer chooses cash or card gateway".
- 4-qadam: "Card path deep-links to external payment app or cash path enters pending state".
- 5-qadam: "ORDER_COMPLETED or failure events drive final phase".

---

**Flowid:** sidebar-navigation

**Summary:** "sidebar-navigation" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer taps avatar".
- 2-qadam: "SidebarMenu slides in from left".
- 3-qadam: "Retailer chooses dashboard, procurement, insights, auto-order, or AI predictions".
- 4-qadam: "Shell navigates or dismisses accordingly".

---


### Stateoverview

- Holat: "base shell with no overlays".
- Holat: "floating active orders visible".
- Holat: "sidebar open with scrim".
- Holat: "active deliveries sheet open".
- Holat: "order detail sheet open".
- Holat: "QR overlay open".
- Holat: "payment sheet choose phase".
- Holat: "payment sheet processing or failed or success phase".

### Figureoverview

- Figura: "full retailer shell".
- Figura: "top-bar control cluster".
- Figura: "bottom-stack with floating active orders bar".
- Figura: "sidebar overlay state".
- Figura: "payment sheet state".

---

**Dossierfile:** retailer-android-secondary-surfaces.json

**Bundleid:** retailer-android-secondary-surfaces

**Appid:** retailer-app-android

**Platform:** android

**Role:** RETAILER

**Status:** implemented

**Entrytype:** bundle

**Localizedsummary:** "retailer-android-secondary-surfaces" paketi "retailer-app-android" ilovasi uchun 12 ta yuzani qamrab oladi.

## Surfaces

**Pageid:** android-retailer-location-picker

**Navroute:** LocationPickerScreen

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/auth/LocationPickerScreen.kt

**Localizedsummary:** "android-retailer-location-picker" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-retailer-location-picker" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Signup or profile location selector using a map-centered pin and confirm affordance.

#### Layoutoverview

- Layout zonasi: "top app bar".
- Layout zonasi: "map canvas".
- Layout zonasi: "center pin indicator".
- Layout zonasi: "address label".
- Layout zonasi: "Confirm Location footer".

#### Controloverview

- Boshqaruv elementi: "close or back control".
- Boshqaruv elementi: "Confirm Location button".

#### Iconoverview

- Ikona joylashuvi: "mappin or location indicator at center".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "pan map under fixed pin".

---

**Summary:** Oqim quyidagicha qayd etilgan: "reverse geocode displayed address".

---

**Summary:** Oqim quyidagicha qayd etilgan: "confirm selected coordinates".

---


#### Dependencyoverview

##### Reads

- map and geocoder state

##### Writes

- selected location result

##### Localizednotes

- O'qish: "map and geocoder state".
- Yozish: "selected location result".

#### Stateoverview

- Holat: "default map state".
- Holat: "confirm-ready state".

#### Figureoverview

- Figura: "android location picker with centered pin".

#### Minifeatureoverview

- Mini-feature: "map pin".
- Mini-feature: "address label".
- Mini-feature: "confirm CTA".

**Minifeaturecount:** 3

---

**Pageid:** android-retailer-home

**Navroute:** HOME

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/dashboard/DashboardScreen.kt

**Localizedsummary:** "android-retailer-home" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-retailer-home" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Retailer dashboard of service-entry tiles, reorder intelligence, and date-range driven spend snapshots.

#### Layoutoverview

- Layout zonasi: "service-tile grid".
- Layout zonasi: "pull-to-refresh scaffold".
- Layout zonasi: "reorder strip".
- Layout zonasi: "date-range buttons".
- Layout zonasi: "summary cards".

#### Controloverview

- Boshqaruv elementi: "service tiles as tap targets".
- Boshqaruv elementi: "date-range segmented buttons".

#### Iconoverview

- Ikona joylashuvi: "service icons inside M3 cards".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "tap into catalog, orders, procurement, inbox, insights".

---

**Summary:** Oqim quyidagicha qayd etilgan: "refresh dashboard".

---

**Summary:** Oqim quyidagicha qayd etilgan: "change analytics date range".

---


#### Dependencyoverview

##### Reads

- orders count
- reorder products
- AI demand forecasts

##### Writes


##### Localizednotes

- O'qish: "orders count", "reorder products", "AI demand forecasts".
- Yozish: yo'q.

#### Stateoverview

- Holat: "normal dashboard".
- Holat: "refreshing".
- Holat: "sparse data".

#### Figureoverview

- Figura: "android retailer dashboard tile grid".

#### Minifeatureoverview

- Mini-feature: "service tiles".
- Mini-feature: "reorder strip".
- Mini-feature: "date-range filter".
- Mini-feature: "pull refresh".
- Mini-feature: "summary cards".

**Minifeaturecount:** 5

---

**Pageid:** android-retailer-category-suppliers

**Navroute:** CATEGORY_SUPPLIERS/{categoryId}/{categoryName}

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/catalog/CategorySuppliersScreen.kt

**Localizedsummary:** "android-retailer-category-suppliers" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-retailer-category-suppliers" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Category-scoped supplier browser for narrowing supplier selection before catalog exploration.

#### Layoutoverview

- Layout zonasi: "top app bar with category name".
- Layout zonasi: "supplier row list".
- Layout zonasi: "empty state".

#### Controloverview

- Boshqaruv elementi: "back button".
- Boshqaruv elementi: "supplier row tap target".

#### Iconoverview

- Ikona joylashuvi: "supplier avatar initials".
- Ikona joylashuvi: "chevron row affordance".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "return to catalog".

---

**Summary:** Oqim quyidagicha qayd etilgan: "open selected supplier catalog".

---


#### Dependencyoverview

##### Reads

- suppliers by category

##### Writes


##### Localizednotes

- O'qish: "suppliers by category".
- Yozish: yo'q.

#### Stateoverview

- Holat: "list populated".
- Holat: "empty".

#### Figureoverview

- Figura: "category suppliers list".

#### Minifeatureoverview

- Mini-feature: "back nav".
- Mini-feature: "supplier rows".
- Mini-feature: "status badge".
- Mini-feature: "empty state".

**Minifeaturecount:** 4

---

**Pageid:** android-retailer-supplier-catalog

**Navroute:** SUPPLIER_CATEGORY_CATALOG/{supplierId}/{supplierName}/{supplierCategory}/{supplierIsActive}

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/suppliers/SupplierCatalogScreen.kt

**Localizedsummary:** "android-retailer-supplier-catalog" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-retailer-supplier-catalog" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier-specific catalog grouped by product category with top-bar supplier identity and availability badge.

#### Layoutoverview

- Layout zonasi: "top app bar with supplier name and category".
- Layout zonasi: "OPEN or CLOSED badge".
- Layout zonasi: "grouped product list".
- Layout zonasi: "category headers".

#### Controloverview

- Boshqaruv elementi: "back button".
- Boshqaruv elementi: "product row tap target".

#### Iconoverview

- Ikona joylashuvi: "availability status dot".
- Ikona joylashuvi: "back icon".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "return to supplier list".

---

**Summary:** Oqim quyidagicha qayd etilgan: "open product detail from grouped list".

---


#### Dependencyoverview

##### Reads

- supplier products grouped by category

##### Writes


##### Localizednotes

- O'qish: "supplier products grouped by category".
- Yozish: yo'q.

#### Stateoverview

- Holat: "supplier open".
- Holat: "supplier closed".
- Holat: "empty catalog".

#### Figureoverview

- Figura: "supplier catalog grouped by category".

#### Minifeatureoverview

- Mini-feature: "supplier title".
- Mini-feature: "category subtitle".
- Mini-feature: "availability badge".
- Mini-feature: "grouped list".

**Minifeaturecount:** 4

---

**Pageid:** android-retailer-product-detail

**Navroute:** PRODUCT_DETAIL/{productId}

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/product/ProductDetailScreen.kt

**Localizedsummary:** "android-retailer-product-detail" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-retailer-product-detail" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Retailer product inspector with variant choice, quantity control, per-variant auto-order toggle, and fixed add-to-cart bar.

#### Layoutoverview

- Layout zonasi: "hero image region".
- Layout zonasi: "product info section".
- Layout zonasi: "variant selector".
- Layout zonasi: "quantity stepper".
- Layout zonasi: "nutrition or metadata section".
- Layout zonasi: "fixed bottom Add to Cart bar".

#### Controloverview

- Boshqaruv elementi: "variant chips".
- Boshqaruv elementi: "quantity plus and minus controls".
- Boshqaruv elementi: "auto-order toggle".
- Boshqaruv elementi: "Add to Cart button".

#### Iconoverview

- Ikona joylashuvi: "placeholder leaf icon".
- Ikona joylashuvi: "toggle or snackbar icons".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "switch variants".

---

**Summary:** Oqim quyidagicha qayd etilgan: "adjust quantity".

---

**Summary:** Oqim quyidagicha qayd etilgan: "toggle auto-order with history or fresh dialog".

---

**Summary:** Oqim quyidagicha qayd etilgan: "add selected configuration to cart".

---


#### Dependencyoverview

##### Reads

- product detail payload
- variant auto-order settings

##### Writes

- cart mutation
- auto-order settings update

##### Localizednotes

- O'qish: "product detail payload", "variant auto-order settings".
- Yozish: "cart mutation", "auto-order settings update".

#### Stateoverview

- Holat: "product loaded".
- Holat: "placeholder image".
- Holat: "auto-order dialog open".

#### Figureoverview

- Figura: "product detail screen with bottom add-to-cart bar".

#### Minifeatureoverview

- Mini-feature: "hero image".
- Mini-feature: "variant chips".
- Mini-feature: "quantity stepper".
- Mini-feature: "auto-order toggle".
- Mini-feature: "Add to Cart bar".
- Mini-feature: "history or fresh dialog".

**Minifeaturecount:** 6

---

**Pageid:** android-retailer-analytics

**Navroute:** ANALYTICS

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/analytics/AnalyticsScreen.kt

**Localizedsummary:** "android-retailer-analytics" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-retailer-analytics" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Retailer expense and supplier-spend analytics dashboard with date-range filters and charts.

#### Layoutoverview

- Layout zonasi: "date-range chip row".
- Layout zonasi: "KPI cards".
- Layout zonasi: "expense line chart".
- Layout zonasi: "top suppliers chart".
- Layout zonasi: "top products table".

#### Controloverview

- Boshqaruv elementi: "date-range filter chips".

#### Iconoverview

- Ikona joylashuvi: "chart glyphs inside KPI cards".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "change date range".

---

**Summary:** Oqim quyidagicha qayd etilgan: "refresh analytics dataset".

---


#### Dependencyoverview

##### Reads

- retailer analytics endpoint

##### Writes


##### Localizednotes

- O'qish: "retailer analytics endpoint".
- Yozish: yo'q.

#### Stateoverview

- Holat: "chart populated".
- Holat: "empty analytics".
- Holat: "refreshing".

#### Figureoverview

- Figura: "analytics screen with range chips and charts".

#### Minifeatureoverview

- Mini-feature: "range chips".
- Mini-feature: "KPI cards".
- Mini-feature: "line chart".
- Mini-feature: "supplier chart".
- Mini-feature: "products table".

**Minifeaturecount:** 5

---

**Pageid:** android-retailer-auto-order

**Navroute:** AUTO_ORDER

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/autoorder/AutoOrderScreen.kt

**Localizedsummary:** "android-retailer-auto-order" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-retailer-auto-order" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Hierarchy-based auto-order settings page covering global, supplier, category, and product enablement.

#### Layoutoverview

- Layout zonasi: "sparkles header card".
- Layout zonasi: "settings list".
- Layout zonasi: "toggle rows".
- Layout zonasi: "confirmation dialog".

#### Controloverview

- Boshqaruv elementi: "toggle switches for each scope".
- Boshqaruv elementi: "Use History action".
- Boshqaruv elementi: "Start Fresh action".
- Boshqaruv elementi: "Cancel action".

#### Iconoverview

- Ikona joylashuvi: "sparkles icon".
- Ikona joylashuvi: "scope icons in rows".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "toggle auto-order at any scope".

---

**Summary:** Oqim quyidagicha qayd etilgan: "open dialog when enabling".

---

**Summary:** Oqim quyidagicha qayd etilgan: "persist selection".

---


#### Dependencyoverview

##### Reads

- auto-order settings

##### Writes

- auto-order enable or disable endpoints

##### Localizednotes

- O'qish: "auto-order settings".
- Yozish: "auto-order enable or disable endpoints".

#### Stateoverview

- Holat: "all disabled".
- Holat: "mixed enabled".
- Holat: "enable dialog open".

#### Figureoverview

- Figura: "auto-order settings hierarchy screen".

#### Minifeatureoverview

- Mini-feature: "global toggle".
- Mini-feature: "supplier toggle".
- Mini-feature: "category toggle".
- Mini-feature: "product toggle".
- Mini-feature: "enable dialog".

**Minifeaturecount:** 5

---

**Pageid:** android-retailer-profile

**Navroute:** PROFILE

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/profile/ProfileScreen.kt

**Localizedsummary:** "android-retailer-profile" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-retailer-profile" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Retailer profile, support, company settings, and global auto-order governance surface.

#### Layoutoverview

- Layout zonasi: "profile header card".
- Layout zonasi: "stats row".
- Layout zonasi: "auto-order card".
- Layout zonasi: "settings sections".
- Layout zonasi: "support rows".

#### Controloverview

- Boshqaruv elementi: "global auto-order switch".
- Boshqaruv elementi: "settings row tap targets".
- Boshqaruv elementi: "logout action".

#### Iconoverview

- Ikona joylashuvi: "avatar initials".
- Ikona joylashuvi: "settings row icons".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "toggle global auto-order".

---

**Summary:** Oqim quyidagicha qayd etilgan: "open history or fresh dialog".

---

**Summary:** Oqim quyidagicha qayd etilgan: "navigate into settings items".

---

**Summary:** Oqim quyidagicha qayd etilgan: "logout".

---


#### Dependencyoverview

##### Reads

- retailer profile endpoint

##### Writes

- global auto-order endpoint

##### Localizednotes

- O'qish: "retailer profile endpoint".
- Yozish: "global auto-order endpoint".

#### Stateoverview

- Holat: "normal profile".
- Holat: "dialog open".

#### Figureoverview

- Figura: "profile screen with auto-order card".

#### Minifeatureoverview

- Mini-feature: "profile card".
- Mini-feature: "status pill".
- Mini-feature: "stats row".
- Mini-feature: "global auto-order toggle".
- Mini-feature: "settings rows".
- Mini-feature: "logout".

**Minifeaturecount:** 6

---

**Pageid:** android-retailer-suppliers

**Navroute:** SUPPLIERS

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/suppliers/MySuppliersScreen.kt

**Localizedsummary:** "android-retailer-suppliers" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-retailer-suppliers" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Favorite suppliers grid with pull-to-refresh and retry-capable empty or error fallback.

#### Layoutoverview

- Layout zonasi: "supplier card grid".
- Layout zonasi: "pull refresh scaffold".
- Layout zonasi: "empty or retry state".

#### Controloverview

- Boshqaruv elementi: "supplier card tap target".
- Boshqaruv elementi: "Retry button in fallback state".

#### Iconoverview

- Ikona joylashuvi: "supplier initials tile".
- Ikona joylashuvi: "empty-state icon".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "refresh supplier list".

---

**Summary:** Oqim quyidagicha qayd etilgan: "open supplier catalog from card".

---

**Summary:** Oqim quyidagicha qayd etilgan: "retry after failure".

---


#### Dependencyoverview

##### Reads

- favorite suppliers endpoint

##### Writes


##### Localizednotes

- O'qish: "favorite suppliers endpoint".
- Yozish: yo'q.

#### Stateoverview

- Holat: "grid populated".
- Holat: "empty".
- Holat: "error".

#### Figureoverview

- Figura: "favorite suppliers grid".

#### Minifeatureoverview

- Mini-feature: "grid view".
- Mini-feature: "pull refresh".
- Mini-feature: "retry action".
- Mini-feature: "order count badge".
- Mini-feature: "auto-order badge".

**Minifeaturecount:** 5

---

**Pageid:** android-retailer-order-detail-sheet

**Navroute:** OrderDetailSheet

**Surfacetype:** overlay

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/OrderDetailSheet.kt

**Localizedsummary:** "android-retailer-order-detail-sheet" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-retailer-order-detail-sheet" yuzasi unknown-platform platformasida unknown-role roli uchun overlay sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Bottom-sheet order drill-down showing status, items, amounts, and terminal actions tied to an active order.

#### Layoutoverview

- Layout zonasi: "sheet header".
- Layout zonasi: "order metadata section".
- Layout zonasi: "line-item list".
- Layout zonasi: "footer action row".

#### Controloverview

- Boshqaruv elementi: "sheet close affordance".
- Boshqaruv elementi: "status-specific footer actions".

#### Iconoverview

- Ikona joylashuvi: "status icon or ring".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "open from order card".

---

**Summary:** Oqim quyidagicha qayd etilgan: "inspect line items".

---

**Summary:** Oqim quyidagicha qayd etilgan: "launch payment or QR step when relevant".

---


#### Dependencyoverview

##### Reads

- selected order payload

##### Writes

- order action callbacks

##### Localizednotes

- O'qish: "selected order payload".
- Yozish: "order action callbacks".

#### Stateoverview

- Holat: "standard order review".
- Holat: "actionable order state".

#### Figureoverview

- Figura: "order detail bottom sheet".

#### Minifeatureoverview

- Mini-feature: "sheet header".
- Mini-feature: "item list".
- Mini-feature: "footer actions".
- Mini-feature: "status indicator".

**Minifeaturecount:** 4

---

**Pageid:** android-retailer-qr-overlay

**Navroute:** QROverlay

**Surfacetype:** overlay

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/QROverlay.kt

**Localizedsummary:** "android-retailer-qr-overlay" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-retailer-qr-overlay" yuzasi unknown-platform platformasida unknown-role roli uchun overlay sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Retailer QR verification overlay for delivery acceptance and handoff confirmation.

#### Layoutoverview

- Layout zonasi: "camera or QR display frame".
- Layout zonasi: "instruction label".
- Layout zonasi: "dismiss region".

#### Controloverview

- Boshqaruv elementi: "dismiss control".
- Boshqaruv elementi: "confirm or continue action if present".

#### Iconoverview

- Ikona joylashuvi: "QR framing corners".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "show QR token for driver scan or scan driver token depending on state".

---

**Summary:** Oqim quyidagicha qayd etilgan: "dismiss overlay".

---


#### Dependencyoverview

##### Reads

- QR payload state

##### Writes

- verification callback

##### Localizednotes

- O'qish: "QR payload state".
- Yozish: "verification callback".

#### Stateoverview

- Holat: "display mode".
- Holat: "scan mode".

#### Figureoverview

- Figura: "QR overlay figure".

#### Minifeatureoverview

- Mini-feature: "QR frame".
- Mini-feature: "instruction text".
- Mini-feature: "dismiss control".

**Minifeaturecount:** 3

---

**Pageid:** android-retailer-sidebar-menu

**Navroute:** SidebarMenu

**Surfacetype:** overlay

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/SidebarMenu.kt

**Localizedsummary:** "android-retailer-sidebar-menu" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "android-retailer-sidebar-menu" yuzasi unknown-platform platformasida unknown-role roli uchun overlay sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Drawer-style navigation overlay exposing secondary retailer destinations and profile context.

#### Layoutoverview

- Layout zonasi: "avatar header".
- Layout zonasi: "menu rows".
- Layout zonasi: "footer utility area".

#### Controloverview

- Boshqaruv elementi: "menu row tap targets".
- Boshqaruv elementi: "dismiss touch scrim".

#### Iconoverview

- Ikona joylashuvi: "row icons".
- Ikona joylashuvi: "avatar initials".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "open from shell menu trigger".

---

**Summary:** Oqim quyidagicha qayd etilgan: "navigate to selected destination".

---

**Summary:** Oqim quyidagicha qayd etilgan: "dismiss drawer".

---


#### Dependencyoverview

##### Reads

- retailer identity summary

##### Writes

- navigation state

##### Localizednotes

- O'qish: "retailer identity summary".
- Yozish: "navigation state".

#### Stateoverview

- Holat: "open drawer".

#### Figureoverview

- Figura: "sidebar drawer overlay".

#### Minifeatureoverview

- Mini-feature: "avatar header".
- Mini-feature: "menu rows".
- Mini-feature: "icon stack".
- Mini-feature: "dismiss scrim".

**Minifeaturecount:** 4

---


---

**Dossierfile:** retailer-ios-active-deliveries.json

**Pageid:** ios-retailer-active-deliveries

**Viewname:** ActiveDeliveriesView

**Platform:** ios

**Role:** RETAILER

**Status:** implemented

**Shell:** retailer-ios-overlay

## Sourcefiles

- apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ActiveDeliveriesView.swift

**Entrytype:** page

**Localizedsummary:** "ios-retailer-active-deliveries" yuzasi uchun chakana savdogar roli va iOS platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "ios-retailer-active-deliveries" yuzasi iOS platformasida chakana savdogar roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Retailer active-delivery monitor showing only live orders, detail-sheet drilldown, and QR handoff from a dedicated delivery surface.

### Layoutoverview

- "delivery-scroll-region" zonasi "main body" hududida joylashgan. Tarkibi: "active delivery card list".
- "empty-state" zonasi "center body" hududida joylashgan. Ko'rinish qoidasi: "visible when orders list is empty". Tarkibi: "shippingbox icon disk"; "No Active Orders headline"; "helper copy".
- "detail-sheet" zonasi "bottom sheet" hududida joylashgan. Ko'rinish qoidasi: "visible when selectedOrder is non-null". Tarkibi: "OrderDetailSheet at 75 percent height".
- "qr-overlay" zonasi "full-screen overlay" hududida joylashgan. Ko'rinish qoidasi: "visible when qrOverlayOrder is non-null and status has delivery token". Tarkibi: "QROverlay".

### Controloverview

- "Details" tugmasi "delivery card action row" hududida joylashgan. Uslub: "neutral pill button".
- "Show QR" tugmasi "delivery card action row" hududida joylashgan. Uslub: "accent pill button". Ko'rinish qoidasi: "order has delivery token".
- "Awaiting Dispatch" tugmasi "delivery card action row" hududida joylashgan. Uslub: "disabled status pill". Ko'rinish qoidasi: "order lacks delivery token".

### Iconoverview

- "shippingbox or qrcode or clock" ikonasi "delivery cards and empty state" zonasida ishlatiladi.

### Flowoverview

**Flowid:** active-delivery-review

**Summary:** "active-delivery-review" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "View loads retailer orders and filters them to active statuses".
- 2-qadam: "Retailer taps Details for a selected order".
- 3-qadam: "Retailer may alternatively tap Show QR for token-enabled orders".

---


### Stateoverview

- Holat: "loading state".
- Holat: "empty state".
- Holat: "active deliveries list".
- Holat: "detail sheet open".
- Holat: "QR overlay open".

### Figureoverview

- Figura: "retailer iOS active deliveries list".
- Figura: "delivery card close-up".
- Figura: "active deliveries QR overlay".

---

**Dossierfile:** retailer-ios-cart.json

**Pageid:** ios-retailer-cart

**Viewname:** CartView

**Platform:** ios

**Role:** RETAILER

**Status:** implemented

**Shell:** retailer-ios-root

## Sourcefiles

- apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/CartView.swift

**Entrytype:** page

**Localizedsummary:** "ios-retailer-cart" yuzasi uchun chakana savdogar roli va iOS platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "ios-retailer-cart" yuzasi iOS platformasida chakana savdogar roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Retailer basket-management screen with item-level quantity control, destructive removal, summary footer, and full-screen checkout handoff.

### Layoutoverview

- "cart-list-region" zonasi "scroll body when cart has items" hududida joylashgan. Tarkibi: "cart count header"; "Clear All button"; "cart item cards with image placeholder, product metadata, total price, quantity stepper, and delete affordance".
- "bottom-bar" zonasi "sticky bottom summary bar" hududida joylashgan. Ko'rinish qoidasi: "visible when cart is not empty". Tarkibi: "subtotal row"; "delivery row"; "total cluster"; "Checkout pill button".
- "empty-state" zonasi "center body" hududida joylashgan. Ko'rinish qoidasi: "visible when cart.isEmpty is true". Tarkibi: "double-ring cart illustration"; "empty headline"; "helper copy"; "Browse Catalog button".
- "checkout-cover" zonasi "full-screen cover" hududida joylashgan. Ko'rinish qoidasi: "visible when showCheckout is true". Tarkibi: "CheckoutView full-screen modal".

### Controloverview

- "Clear All" tugmasi "cart-list-region header-right" hududida joylashgan. Uslub: "text destructive".
- "quantity stepper decrement" tugmasi "each cart item card" hududida joylashgan. Uslub: "stepper button".
- "quantity stepper increment" tugmasi "each cart item card" hududida joylashgan. Uslub: "stepper button".
- "delete" tugmasi "cart item trailing overlay" hududida joylashgan. Uslub: "small destructive icon button".
- "Checkout" tugmasi "bottom-bar right" hududida joylashgan. Uslub: "accent pill CTA".
- "Browse Catalog" tugmasi "empty-state" hududida joylashgan. Uslub: "accent pill CTA".

### Iconoverview

- "leaf.fill" ikonasi "cart item image placeholder" zonasida ishlatiladi.
- "trash" ikonasi "cart item delete overlay and swipe action" zonasida ishlatiladi.
- "arrow.right" ikonasi "Checkout CTA trailing edge" zonasida ishlatiladi.
- "square.grid.2x2" ikonasi "Browse Catalog button" zonasida ishlatiladi.
- "cart" ikonasi "empty-state illustration" zonasida ishlatiladi.

### Flowoverview

**Flowid:** quantity-adjustment

**Summary:** "quantity-adjustment" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer uses quantity stepper on a cart item".
- 2-qadam: "CartManager updates item quantity".
- 3-qadam: "line totals and bottom summary recompute".

---

**Flowid:** item-removal

**Summary:** "item-removal" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer taps trailing delete control or swipe action".
- 2-qadam: "CartManager removes item with animated transition".

---

**Flowid:** checkout-handoff

**Summary:** "checkout-handoff" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer taps Checkout".
- 2-qadam: "CartView presents CheckoutView as a full-screen cover".

---


### Stateoverview

- Holat: "populated cart state".
- Holat: "empty cart state".
- Holat: "quantity update state".
- Holat: "full-screen checkout cover active".

### Figureoverview

- Figura: "retailer iOS cart populated state".
- Figura: "cart item row with quantity stepper and delete control".
- Figura: "cart bottom summary bar".
- Figura: "empty cart state".

---

**Dossierfile:** retailer-ios-catalog.json

**Pageid:** ios-retailer-catalog

**Viewname:** CatalogView

**Platform:** ios

**Role:** RETAILER

**Status:** implemented

**Shell:** retailer-ios-root

## Sourcefiles

- apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/CatalogView.swift

**Entrytype:** page

**Localizedsummary:** "ios-retailer-catalog" yuzasi uchun chakana savdogar roli va iOS platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "ios-retailer-catalog" yuzasi iOS platformasida chakana savdogar roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Retailer category-browse and product-search screen using a bento-grid catalog overview and product-grid search results.

### Layoutoverview

- "search-bar" zonasi "top full-width" hududida joylashgan. Tarkibi: "magnifying glass icon"; "search text field"; "clear-search button when text is non-empty".
- "category-browse-region" zonasi "scroll body when search is empty" hududida joylashgan. Ko'rinish qoidasi: "visible when searchText is empty". Tarkibi: "Categories header row with count"; "mixed-size bento category cards"; "remaining categories two-column grid".
- "search-results-grid" zonasi "scroll body when search has results" hududida joylashgan. Ko'rinish qoidasi: "visible when searchText is non-empty and filteredProducts is non-empty". Tarkibi: "two-column ProductCardView grid".
- "no-results-state" zonasi "center body" hududida joylashgan. Ko'rinish qoidasi: "visible when searchText is non-empty and filteredProducts is empty". Tarkibi: "search icon disk"; "No Results headline"; "query-specific helper text".

### Controloverview

- "clear search" tugmasi "search-bar trailing edge" hududida joylashgan. Uslub: "icon button".
- "bento category card" tugmasi "category-browse-region" hududida joylashgan. Uslub: "card tap target".
- "product card" tugmasi "search-results-grid" hududida joylashgan. Uslub: "card tap target".

### Iconoverview

- "magnifyingglass" ikonasi "search-bar leading edge" zonasida ishlatiladi.
- "xmark.circle.fill" ikonasi "search-bar trailing clear button" zonasida ishlatiladi.
- "category icon glyph" ikonasi "bento cards" zonasida ishlatiladi.
- "magnifyingglass" ikonasi "no-results-state hero icon" zonasida ishlatiladi.

### Flowoverview

**Flowid:** category-browse

**Summary:** "category-browse" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer lands on category overview".
- 2-qadam: "Retailer taps a bento category card".
- 3-qadam: "Navigation pushes CategorySuppliersView".

---

**Flowid:** product-search

**Summary:** "product-search" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer types into search bar".
- 2-qadam: "Catalog switches from category bento layout to filtered product grid".
- 3-qadam: "Retailer taps a product card".
- 4-qadam: "Navigation pushes ProductDetailView".

---


### Stateoverview

- Holat: "loading skeleton grid".
- Holat: "category bento state".
- Holat: "search results state".
- Holat: "no-results state".
- Holat: "failed-load alert".

### Figureoverview

- Figura: "retailer iOS catalog bento grid".
- Figura: "search bar close-up".
- Figura: "product search results grid".
- Figura: "no-results state".

---

**Dossierfile:** retailer-ios-checkout.json

**Pageid:** ios-retailer-checkout

**Viewname:** CheckoutView

**Platform:** ios

**Role:** RETAILER

**Status:** implemented

**Shell:** retailer-ios-modal

## Sourcefiles

- apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/CheckoutView.swift

**Entrytype:** page

**Localizedsummary:** "ios-retailer-checkout" yuzasi uchun chakana savdogar roli va iOS platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "ios-retailer-checkout" yuzasi iOS platformasida chakana savdogar roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Retailer order-finalization screen with cart recap, payment-method selection, supplier-closed confirmation, offline retry fallback, and success state.

### Layoutoverview

- "toolbar" zonasi "top navigation bar" hududida joylashgan. Tarkibi: "Checkout title"; "circular xmark dismiss button".
- "checkout-scroll-stack" zonasi "main scroll body while showSuccess is false" hududida joylashgan. Tarkibi: "Cart card with line items and quantity steppers"; "Payment card with change button"; "Summary card with subtotal, delivery, and total".
- "submit-bar" zonasi "bottom sticky region" hududida joylashgan. Ko'rinish qoidasi: "visible when showSuccess is false". Tarkibi: "Place Order button".
- "payment-picker-sheet" zonasi "modal sheet" hududida joylashgan. Ko'rinish qoidasi: "visible when showPaymentPicker is true". Tarkibi: "payment method list rows"; "selected checkmark".
- "success-state" zonasi "full-screen success replacement" hududida joylashgan. Ko'rinish qoidasi: "visible when showSuccess is true". Tarkibi: "success icon cluster"; "Order Placed headline"; "supporting copy"; "Done button".

### Controloverview

- "dismiss" tugmasi "toolbar trailing edge" hududida joylashgan. Uslub: "circular icon button".
- "Change" tugmasi "payment card trailing edge" hududida joylashgan. Uslub: "text button".
- "Place Order" tugmasi "submit-bar" hududida joylashgan. Uslub: "full-width primary".
- "payment method row" tugmasi "payment-picker-sheet" hududida joylashgan. Uslub: "list row".
- "I Understand, Place Order" tugmasi "supplier-closed confirmation dialog" hududida joylashgan. Uslub: "confirm action".
- "Done" tugmasi "success-state footer" hududida joylashgan. Uslub: "full-width primary".

### Iconoverview

- "xmark" ikonasi "toolbar dismiss control" zonasida ishlatiladi.
- "cart.fill" ikonasi "cart section header" zonasida ishlatiladi.
- "creditcard.fill" ikonasi "payment section header" zonasida ishlatiladi.
- "creditcard or wallet.pass or banknote" ikonasi "payment picker rows" zonasida ishlatiladi.
- "checkmark.circle.fill" ikonasi "success-state hero" zonasida ishlatiladi.

### Flowoverview

**Flowid:** payment-method-selection

**Summary:** "payment-method-selection" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer taps Change in payment card".
- 2-qadam: "Payment picker sheet opens".
- 3-qadam: "Retailer selects Click, Payme, Global Pay, or Cash on Delivery".
- 4-qadam: "selectedPayment updates and sheet dismisses".

---

**Flowid:** order-submission

**Summary:** "order-submission" oqimi 5 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer taps Place Order".
- 2-qadam: "If supplier is inactive, confirmation dialog interposes".
- 3-qadam: "Checkout posts to /v1/checkout/unified with gateway-mapped code".
- 4-qadam: "Success clears cart and shows success state".
- 5-qadam: "Failure stores PendingOrder for retry and shows alert".

---


### Stateoverview

- Holat: "review state".
- Holat: "payment picker sheet".
- Holat: "supplier closed confirmation dialog".
- Holat: "error alert".
- Holat: "submitting state".
- Holat: "success replacement state".

### Figureoverview

- Figura: "retailer iOS checkout review state".
- Figura: "payment picker sheet".
- Figura: "supplier closed confirmation dialog".
- Figura: "checkout success state".

---

**Dossierfile:** retailer-ios-delivery-payment-sheet.json

**Pageid:** ios-retailer-payment-sheet

**Viewname:** DeliveryPaymentSheetView

**Platform:** ios

**Role:** RETAILER

**Status:** implemented

**Shell:** retailer-ios-overlay

## Sourcefiles

- apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/DeliveryPaymentSheetView.swift

**Entrytype:** page

**Localizedsummary:** "ios-retailer-payment-sheet" yuzasi uchun chakana savdogar roli va iOS platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "ios-retailer-payment-sheet" yuzasi iOS platformasida chakana savdogar roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Retailer payment-required overlay for choosing cash or card gateways after offload, waiting for settlement, and confirming successful completion.

### Layoutoverview

- "phase-container" zonasi "sheet body" hududida joylashgan. Tarkibi: "choose content"; "processing content"; "cash pending content"; "success content"; "failed content".
- "choose-phase" zonasi "sheet body when phase is choose" hududida joylashgan. Ko'rinish qoidasi: "visible when phase equals choose". Tarkibi: "warning icon disk"; "amount due stack with optional struck original amount"; "payment method choice list"; "cash option button"; "one or more card gateway option buttons".
- "processing-phase" zonasi "sheet body when phase is processing" hududida joylashgan. Tarkibi: "ProgressView"; "Processing headline"; "connecting helper text".
- "cash-pending-phase" zonasi "sheet body when phase is cashPending" hududida joylashgan. Tarkibi: "banknote icon disk"; "Cash Collection Pending headline"; "amount text"; "waiting pill with progress indicator".
- "success-or-failure-phase" zonasi "sheet body when phase is success or failed" hududida joylashgan. Tarkibi: "success checkmark or failure xmark disk"; "result headline"; "amount or error message"; "Done or Retry and Cancel actions".

### Controloverview

- "Close" tugmasi "navigation bar cancellation action" hududida joylashgan. Uslub: "text button". Ko'rinish qoidasi: "phase is choose or failed".
- "Cash on Delivery option" tugmasi "choose-phase" hududida joylashgan. Uslub: "full-width option row".
- "card gateway option" tugmasi "choose-phase" hududida joylashgan. Uslub: "full-width option row".
- "Done" tugmasi "success phase footer" hududida joylashgan. Uslub: "full-width primary".
- "Try Again" tugmasi "failed phase footer" hududida joylashgan. Uslub: "full-width primary".
- "Cancel" tugmasi "failed phase footer" hududida joylashgan. Uslub: "text button".

### Iconoverview

- "banknote.fill" ikonasi "choose and cash-pending hero disks" zonasida ishlatiladi.
- "creditcard.fill" ikonasi "card gateway option rows" zonasida ishlatiladi.
- "checkmark.circle.fill" ikonasi "success phase hero" zonasida ishlatiladi.
- "xmark.circle.fill" ikonasi "failed phase hero" zonasida ishlatiladi.
- "ProgressView" ikonasi "processing and cash-pending phases" zonasida ishlatiladi.
- "chevron.right" ikonasi "payment option rows" zonasida ishlatiladi.

### Flowoverview

**Flowid:** cash-selection

**Summary:** "cash-selection" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer chooses Cash on Delivery".
- 2-qadam: "Sheet enters processing then cashPending phase".
- 3-qadam: "Sheet listens for driver confirmation or websocket completion".

---

**Flowid:** card-selection

**Summary:** "card-selection" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer chooses Click, Payme, or Global Pay".
- 2-qadam: "Sheet posts checkout request".
- 3-qadam: "External payment URL opens when available".
- 4-qadam: "Sheet remains in processing until paymentSettled or orderCompleted websocket event arrives".

---

**Flowid:** failure-retry

**Summary:** "failure-retry" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Payment attempt fails".
- 2-qadam: "Sheet renders failed phase".
- 3-qadam: "Retailer retries or cancels".

---


### Stateoverview

- Holat: "choose phase".
- Holat: "processing phase".
- Holat: "cash pending phase".
- Holat: "success phase".
- Holat: "failed phase".

### Figureoverview

- Figura: "retailer iOS payment choose phase".
- Figura: "cash pending phase".
- Figura: "processing phase".
- Figura: "success and failure states".

---

**Dossierfile:** retailer-ios-login.json

**Pageid:** ios-retailer-login

**Viewname:** LoginView

**Platform:** ios

**Role:** RETAILER

**Status:** implemented

**Shell:** retailer-ios-auth

## Sourcefiles

- apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/LoginView.swift

**Entrytype:** page

**Localizedsummary:** "ios-retailer-login" yuzasi uchun chakana savdogar roli va iOS platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "ios-retailer-login" yuzasi iOS platformasida chakana savdogar roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Retailer authentication and registration screen combining login, store onboarding, map-based location capture, and logistics intake fields.

### Layoutoverview

- "brand-stack" zonasi "top centered column" hududida joylashgan. Tarkibi: "gradient storefront logo disk"; "Pegasus title"; "Retailer Portal subtitle".
- "credential-core" zonasi "main form column" hududida joylashgan. Tarkibi: "phone field"; "password field".
- "registration-extension" zonasi "below core credentials when sign-up mode active" hududida joylashgan. Ko'rinish qoidasi: "visible when isLoginMode is false". Tarkibi: "store name field"; "owner name field"; "store address field"; "Open Map button"; "Share Location button"; "selected location label"; "tax ID field"; "receiving window open and close fields"; "loading access type chip row"; "ceiling height field".
- "error-row" zonasi "below form fields when auth error exists" hududida joylashgan. Ko'rinish qoidasi: "visible when auth.errorMessage is non-null". Tarkibi: "warning icon"; "error text".
- "primary-action-region" zonasi "below form stack" hududida joylashgan. Tarkibi: "Sign In or Create Account button"; "mode toggle link".

### Controloverview

- "Open Map" tugmasi "registration-extension location row" hududida joylashgan. Uslub: "outlined pill".
- "Share Location" tugmasi "registration-extension location row" hududida joylashgan. Uslub: "outlined pill".
- "Street" tugmasi "registration-extension access type row" hududida joylashgan. Uslub: "chip toggle".
- "Alley" tugmasi "registration-extension access type row" hududida joylashgan. Uslub: "chip toggle".
- "Dock" tugmasi "registration-extension access type row" hududida joylashgan. Uslub: "chip toggle".
- "Sign In" tugmasi "primary-action-region" hududida joylashgan. Uslub: "full-width gradient CTA". Ko'rinish qoidasi: "isLoginMode true".
- "Create Account" tugmasi "primary-action-region" hududida joylashgan. Uslub: "full-width gradient CTA". Ko'rinish qoidasi: "isLoginMode false".
- "mode toggle link" tugmasi "primary-action-region footer" hududida joylashgan. Uslub: "text button".

### Iconoverview

- "storefront.fill" ikonasi "brand-stack logo disk" zonasida ishlatiladi.
- "map" ikonasi "Open Map button" zonasida ishlatiladi.
- "location.fill" ikonasi "Share Location button" zonasida ishlatiladi.
- "arrow.right" ikonasi "primary CTA trailing edge when not loading" zonasida ishlatiladi.
- "ProgressView" ikonasi "primary CTA when auth.isLoading" zonasida ishlatiladi.
- "exclamationmark.triangle.fill" ikonasi "error-row" zonasida ishlatiladi.

### Flowoverview

**Flowid:** retailer-login

**Summary:** "retailer-login" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer enters phone and password".
- 2-qadam: "Retailer taps Sign In".
- 3-qadam: "AuthManager login executes and authenticated state transitions into the app shell".

---

**Flowid:** retailer-registration

**Summary:** "retailer-registration" oqimi 5 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer toggles to sign-up mode".
- 2-qadam: "Additional onboarding fields animate into view".
- 3-qadam: "Retailer may open map picker or share GPS location".
- 4-qadam: "Retailer submits Create Account".
- 5-qadam: "AuthManager register executes with location and logistics metadata".

---

**Flowid:** location-capture

**Summary:** "location-capture" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer opens map picker or uses current location".
- 2-qadam: "Latitude and longitude populate state".
- 3-qadam: "Selected location label renders below the location row".

---


### Stateoverview

- Holat: "login mode".
- Holat: "registration mode".
- Holat: "GPS locating state".
- Holat: "error state".
- Holat: "submitting state".
- Holat: "map picker route handoff".

### Figureoverview

- Figura: "retailer iOS login mode".
- Figura: "retailer iOS registration mode".
- Figura: "location capture row".
- Figura: "error and submitting CTA state".

---

**Dossierfile:** retailer-ios-orders.json

**Pageid:** ios-retailer-orders

**Viewname:** OrdersView

**Platform:** ios

**Role:** RETAILER

**Status:** implemented

**Shell:** retailer-ios-root

## Sourcefiles

- apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/OrdersView.swift

**Entrytype:** page

**Localizedsummary:** "ios-retailer-orders" yuzasi uchun chakana savdogar roli va iOS platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "ios-retailer-orders" yuzasi iOS platformasida chakana savdogar roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Retailer order-tracking hub with active, pending, and AI-planned tabs, detail-sheet drilldown, and QR overlay access for dispatched orders.

### Layoutoverview

- "top-tabs" zonasi "top full-width" hududida joylashgan. Tarkibi: "Active tab with count badge"; "Pending tab with count badge"; "AI Planned tab with count badge".
- "tab-content-pager" zonasi "main body" hududida joylashgan. Tarkibi: "active order card list"; "pending order card list"; "AI planned forecast list".
- "detail-sheet" zonasi "bottom sheet" hududida joylashgan. Ko'rinish qoidasi: "visible when selectedOrder is non-null". Tarkibi: "OrderDetailSheet at 75 percent height".
- "qr-overlay" zonasi "full-screen overlay" hududida joylashgan. Ko'rinish qoidasi: "visible when qrOverlayOrder is non-null and status has delivery token". Tarkibi: "QROverlay over current tab content".

### Controloverview

- "tab selector" tugmasi "top-tabs" hududida joylashgan. Uslub: "tab button".
- "Details" tugmasi "active and pending card action row" hududida joylashgan. Uslub: "pill button".
- "Show QR" tugmasi "active order card action row" hududida joylashgan. Uslub: "accent pill button". Ko'rinish qoidasi: "order has delivery token".
- "Pre-Order" tugmasi "AI planned card trailing action" hududida joylashgan. Uslub: "accent pill button".
- "View" tugmasi "pending order card trailing action" hududida joylashgan. Uslub: "neutral pill button".

### Iconoverview

- "bolt.fill" ikonasi "Active tab" zonasida ishlatiladi.
- "clock.fill" ikonasi "Pending tab" zonasida ishlatiladi.
- "sparkles" ikonasi "AI Planned tab" zonasida ishlatiladi.
- "shippingbox.fill or clock.fill or qrcode" ikonasi "order cards and action pills" zonasida ishlatiladi.

### Flowoverview

**Flowid:** tabbed-order-navigation

**Summary:** "tabbed-order-navigation" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer switches between Active, Pending, and AI Planned tabs".
- 2-qadam: "TabView page content swaps without index dots".

---

**Flowid:** order-drilldown

**Summary:** "order-drilldown" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer taps Details or View on an order card".
- 2-qadam: "OrderDetailSheet opens with logistics, line items, totals, and QR content when available".

---

**Flowid:** qr-surface

**Summary:** "qr-surface" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer taps Show QR on an active order".
- 2-qadam: "QROverlay appears over the orders interface".

---


### Stateoverview

- Holat: "active tab populated".
- Holat: "pending tab populated".
- Holat: "AI planned tab populated".
- Holat: "empty tab states".
- Holat: "loading state".
- Holat: "detail sheet open".
- Holat: "QR overlay visible".

### Figureoverview

- Figura: "retailer iOS orders active tab".
- Figura: "retailer iOS orders pending tab".
- Figura: "AI planned forecast card".
- Figura: "orders QR overlay".

---

**Dossierfile:** retailer-ios-root-shell.json

**Pageid:** ios-retailer-root-shell

**Viewname:** ContentView

**Platform:** ios

**Role:** RETAILER

**Status:** implemented

**Shell:** retailer-ios-root

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/ContentView.swift

**Entrytype:** page

**Localizedsummary:** "ios-retailer-root-shell" yuzasi uchun chakana savdogar roli va iOS platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "ios-retailer-root-shell" yuzasi iOS platformasida chakana savdogar roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Authenticated retailer shell for tab navigation, toolbar-driven controls, floating active-order summary, and modal or sheet-based operational flows.

### Layoutoverview

- "tab-layer" zonasi "full-screen base layer" hududida joylashgan. Tarkibi: "Home tab"; "Catalog tab"; "Orders tab"; "Profile tab"; "Suppliers tab".
- "toolbar" zonasi "top navigation bar within each tab" hududida joylashgan. Tarkibi: "left: circular avatar/menu button"; "center: leaf icon plus Pegasus wordmark"; "right: cart button with count badge; notification bell with count badge".
- "floating-summary" zonasi "bottom above tab bar" hududida joylashgan. Ko'rinish qoidasi: "visible on home, orders, and suppliers tabs only". Tarkibi: "FloatingActiveOrdersBar".
- "sheet-and-overlay-layer" zonasi "above base layer" hududida joylashgan. Tarkibi: "SidebarMenu"; "BottomSheetOverlay containing ActiveDeliveriesView"; "DeliveryPaymentSheetView"; "FutureDemandView sheet"; "AutoOrderView sheet"; "CartView sheet"; "InsightsView sheet"; "ProfileView sheet".

### Controloverview

- "avatar/menu button" tugmasi "toolbar-left" hududida joylashgan. Uslub: "circular gradient button".
- "cart button" tugmasi "toolbar-right" hududida joylashgan. Uslub: "icon button with numeric badge".
- "notification button" tugmasi "toolbar-right" hududida joylashgan. Uslub: "icon button with numeric badge".
- "floating active orders bar" tugmasi "bottom floating layer" hududida joylashgan. Uslub: "pill-like full-width CTA".
- "sidebar rows" tugmasi "sidebar vertical stack" hududida joylashgan. Uslub: "plain button rows with icon tile and chevron".
- "Done toolbar actions" tugmasi "sheet top-right confirmation slot" hududida joylashgan. Uslub: "text confirmation button".

### Iconoverview

- "house / square.grid.2x2 / shippingbox / person.circle / building.2" ikonasi "tab bar" zonasida ishlatiladi.
- "leaf.fill" ikonasi "toolbar center brand mark" zonasida ishlatiladi.
- "cart" ikonasi "toolbar cart action" zonasida ishlatiladi.
- "bell" ikonasi "toolbar notification action" zonasida ishlatiladi.
- "chevron.up" ikonasi "floating active orders bar expand affordance" zonasida ishlatiladi.
- "square.grid.2x2 / chart.bar / chart.line.uptrend.xyaxis / wand.and.stars / sparkles / tray / person / gearshape / rectangle.portrait.and.arrow.right" ikonasi "sidebar rows" zonasida ishlatiladi.

### Flowoverview

**Flowid:** active-orders-drilldown

**Summary:** "active-orders-drilldown" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "FloatingActiveOrdersBar appears when active orders exist".
- 2-qadam: "Retailer taps floating bar".
- 3-qadam: "BottomSheetOverlay presents ActiveDeliveriesView".
- 4-qadam: "Retailer navigates to delivery detail or payment sheet based on current order state".

---

**Flowid:** sidebar-mode-switching

**Summary:** "sidebar-mode-switching" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Retailer taps avatar button".
- 2-qadam: "SidebarMenu animates in from left".
- 3-qadam: "Retailer selects dashboard, procurement, insights, auto-order, AI predictions, inbox, profile, or settings".
- 4-qadam: "Shell switches tab or opens target sheet".

---

**Flowid:** payment-event-presentation

**Summary:** "payment-event-presentation" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "RetailerWebSocket sets paymentEvent".
- 2-qadam: "DeliveryPaymentSheetView presents as large sheet".
- 3-qadam: "Retailer resolves payment".
- 4-qadam: "Sheet dismiss triggers active-order reload".

---


### Stateoverview

- Holat: "base tab shell".
- Holat: "floating active orders visible".
- Holat: "sidebar open with dimmed background".
- Holat: "active deliveries bottom sheet open".
- Holat: "payment sheet open".
- Holat: "future demand sheet open".
- Holat: "cart sheet open".
- Holat: "insights sheet open".

### Figureoverview

- Figura: "full iOS retailer shell".
- Figura: "toolbar control cluster".
- Figura: "floating active orders bar".
- Figura: "sidebar overlay state".
- Figura: "payment sheet state".

---

**Dossierfile:** retailer-ios-secondary-surfaces.json

**Bundleid:** retailer-ios-secondary-surfaces

**Appid:** retailer-app-ios

**Platform:** ios

**Role:** RETAILER

**Status:** implemented

**Entrytype:** bundle

**Localizedsummary:** "retailer-ios-secondary-surfaces" paketi "retailer-app-ios" ilovasi uchun 17 ta yuzani qamrab oladi.

## Surfaces

**Pageid:** ios-retailer-home

**Viewname:** DashboardView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/DashboardView.swift

**Localizedsummary:** "ios-retailer-home" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-home" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Retailer service dashboard with breadbox-style tiles, quick reorder, and AI forecast highlights.

#### Layoutoverview

- Layout zonasi: "hero service tile grid".
- Layout zonasi: "quick reorder section".
- Layout zonasi: "AI prediction cards".
- Layout zonasi: "refresh scaffold".

#### Controloverview

- Boshqaruv elementi: "service tiles as navigation targets".

#### Iconoverview

- Ikona joylashuvi: "service icons in tiles".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "tap into catalog, orders, procurement, inbox, insights, history, search, profile".

---

**Summary:** Oqim quyidagicha qayd etilgan: "refresh dashboard data".

---


#### Dependencyoverview

##### Reads

- orders summary
- reorder products
- AI demand forecasts

##### Writes


##### Localizednotes

- O'qish: "orders summary", "reorder products", "AI demand forecasts".
- Yozish: yo'q.

#### Stateoverview

- Holat: "normal dashboard".
- Holat: "refreshing".

#### Figureoverview

- Figura: "retailer iOS dashboard tile grid".

#### Minifeatureoverview

- Mini-feature: "service grid".
- Mini-feature: "quick reorder strip".
- Mini-feature: "AI cards".
- Mini-feature: "pull refresh".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-category-suppliers

**Viewname:** CategorySuppliersView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/CategorySuppliersView.swift

**Localizedsummary:** "ios-retailer-category-suppliers" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-category-suppliers" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier list filtered by category for drill-down browsing.

#### Layoutoverview

- Layout zonasi: "navigation header".
- Layout zonasi: "supplier rows".
- Layout zonasi: "empty-state region".

#### Controloverview

- Boshqaruv elementi: "supplier row tap target".

#### Iconoverview

- Ikona joylashuvi: "supplier initials tiles".
- Ikona joylashuvi: "chevron affordance".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "select supplier to enter supplier products".

---


#### Dependencyoverview

##### Reads

- suppliers by category endpoint

##### Writes


##### Localizednotes

- O'qish: "suppliers by category endpoint".
- Yozish: yo'q.

#### Stateoverview

- Holat: "rows present".
- Holat: "empty".

#### Figureoverview

- Figura: "category supplier list".

#### Minifeatureoverview

- Mini-feature: "supplier rows".
- Mini-feature: "status badge".
- Mini-feature: "empty state".

**Minifeaturecount:** 3

---

**Pageid:** ios-retailer-my-suppliers

**Viewname:** MySuppliersView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/MySuppliersView.swift

**Localizedsummary:** "ios-retailer-my-suppliers" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-my-suppliers" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Favorite supplier gallery with search, refresh, order counts, and auto-order badges.

#### Layoutoverview

- Layout zonasi: "search field".
- Layout zonasi: "supplier card grid".
- Layout zonasi: "empty-state region".

#### Controloverview

- Boshqaruv elementi: "supplier card tap target".

#### Iconoverview

- Ikona joylashuvi: "avatar initials".
- Ikona joylashuvi: "auto-order badge".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "search favorite suppliers".

---

**Summary:** Oqim quyidagicha qayd etilgan: "refresh supplier grid".

---

**Summary:** Oqim quyidagicha qayd etilgan: "open supplier products".

---


#### Dependencyoverview

##### Reads

- favorite suppliers

##### Writes


##### Localizednotes

- O'qish: "favorite suppliers".
- Yozish: yo'q.

#### Stateoverview

- Holat: "grid populated".
- Holat: "empty".

#### Figureoverview

- Figura: "favorite supplier grid with badges".

#### Minifeatureoverview

- Mini-feature: "search".
- Mini-feature: "grid".
- Mini-feature: "order count badge".
- Mini-feature: "auto-order badge".
- Mini-feature: "refresh".

**Minifeaturecount:** 5

---

**Pageid:** ios-retailer-supplier-products

**Viewname:** SupplierProductsView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/SupplierProductsView.swift

**Localizedsummary:** "ios-retailer-supplier-products" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-supplier-products" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier catalog grouped by category with supplier follow-state and supplier-level auto-order control.

#### Layoutoverview

- Layout zonasi: "supplier header card".
- Layout zonasi: "Add or Remove Supplier control".
- Layout zonasi: "supplier auto-order toggle".
- Layout zonasi: "category-grouped product sections".

#### Controloverview

- Boshqaruv elementi: "Add or Remove Supplier button".
- Boshqaruv elementi: "supplier auto-order toggle".
- Boshqaruv elementi: "product row tap target".

#### Iconoverview

- Ikona joylashuvi: "supplier avatar initials".
- Ikona joylashuvi: "OPEN or CLOSED badge".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "favorite or unfavorite supplier".

---

**Summary:** Oqim quyidagicha qayd etilgan: "toggle supplier auto-order".

---

**Summary:** Oqim quyidagicha qayd etilgan: "open product detail".

---


#### Dependencyoverview

##### Reads

- supplier products endpoint

##### Writes

- favorite supplier endpoint
- supplier auto-order endpoint

##### Localizednotes

- O'qish: "supplier products endpoint".
- Yozish: "favorite supplier endpoint", "supplier auto-order endpoint".

#### Stateoverview

- Holat: "supplier open".
- Holat: "supplier closed".

#### Figureoverview

- Figura: "supplier products screen with header card".

#### Minifeatureoverview

- Mini-feature: "supplier header".
- Mini-feature: "favorite button".
- Mini-feature: "auto-order toggle".
- Mini-feature: "status badge".
- Mini-feature: "grouped products".

**Minifeaturecount:** 5

---

**Pageid:** ios-retailer-product-detail

**Viewname:** ProductDetailView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ProductDetailView.swift

**Localizedsummary:** "ios-retailer-product-detail" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-product-detail" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Product detail inspector with imagery, quantity selection, variant logic, and add-to-cart flow.

#### Layoutoverview

- Layout zonasi: "hero image area".
- Layout zonasi: "product info stack".
- Layout zonasi: "variant selector".
- Layout zonasi: "quantity controls".
- Layout zonasi: "bottom add-to-cart action area".

#### Controloverview

- Boshqaruv elementi: "variant chip buttons".
- Boshqaruv elementi: "quantity plus and minus".
- Boshqaruv elementi: "Add to Cart button".

#### Iconoverview

- Ikona joylashuvi: "placeholder image glyph".
- Ikona joylashuvi: "nutrition or metadata icons".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "select variant".

---

**Summary:** Oqim quyidagicha qayd etilgan: "adjust quantity".

---

**Summary:** Oqim quyidagicha qayd etilgan: "add product to cart".

---


#### Dependencyoverview

##### Reads

- product detail payload

##### Writes

- cart state mutation

##### Localizednotes

- O'qish: "product detail payload".
- Yozish: "cart state mutation".

#### Stateoverview

- Holat: "image present".
- Holat: "placeholder image".

#### Figureoverview

- Figura: "product detail screen with bottom CTA".

#### Minifeatureoverview

- Mini-feature: "hero image".
- Mini-feature: "variant chips".
- Mini-feature: "quantity stepper".
- Mini-feature: "Add to Cart CTA".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-category-products

**Viewname:** CategoryProductsView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/CategoryProductsView.swift

**Localizedsummary:** "ios-retailer-category-products" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-category-products" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Category-scoped products grouped by supplier with inline per-product auto-order controls.

#### Layoutoverview

- Layout zonasi: "category header".
- Layout zonasi: "collapsible supplier sections".
- Layout zonasi: "product rows with quantity and toggle controls".

#### Controloverview

- Boshqaruv elementi: "supplier group expand or collapse".
- Boshqaruv elementi: "product auto-order toggle".
- Boshqaruv elementi: "product row tap target".

#### Iconoverview

- Ikona joylashuvi: "section chevrons".
- Ikona joylashuvi: "auto-order badge".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "expand supplier group".

---

**Summary:** Oqim quyidagicha qayd etilgan: "toggle product auto-order".

---

**Summary:** Oqim quyidagicha qayd etilgan: "open product detail".

---


#### Dependencyoverview

##### Reads

- products by category

##### Writes

- product auto-order endpoint

##### Localizednotes

- O'qish: "products by category".
- Yozish: "product auto-order endpoint".

#### Stateoverview

- Holat: "collapsed groups".
- Holat: "expanded groups".

#### Figureoverview

- Figura: "category products grouped by supplier".

#### Minifeatureoverview

- Mini-feature: "group headers".
- Mini-feature: "expand collapse".
- Mini-feature: "product toggle".
- Mini-feature: "quantity adjuster".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-active-order

**Viewname:** ActiveOrderView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ActiveOrderView.swift

**Localizedsummary:** "ios-retailer-active-order" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-active-order" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Active-order monitor for in-transit orders with live-state emphasis.

#### Layoutoverview

- Layout zonasi: "active order card list".
- Layout zonasi: "status emphasis band".
- Layout zonasi: "refresh scaffold".

#### Controloverview

- Boshqaruv elementi: "order card tap target".

#### Iconoverview

- Ikona joylashuvi: "live indicator dot".
- Ikona joylashuvi: "status badge".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "refresh active orders".

---

**Summary:** Oqim quyidagicha qayd etilgan: "open selected order detail".

---


#### Dependencyoverview

##### Reads

- orders filtered by IN_TRANSIT

##### Writes


##### Localizednotes

- O'qish: "orders filtered by IN_TRANSIT".
- Yozish: yo'q.

#### Stateoverview

- Holat: "active orders present".
- Holat: "no active orders".

#### Figureoverview

- Figura: "active order list".

#### Minifeatureoverview

- Mini-feature: "live dot".
- Mini-feature: "order cards".
- Mini-feature: "status pill".
- Mini-feature: "refresh".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-arrival

**Viewname:** ArrivalView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ArrivalView.swift

**Localizedsummary:** "ios-retailer-arrival" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-arrival" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Arrival-state order list emphasizing imminent handoff and manual arrival confirmation.

#### Layoutoverview

- Layout zonasi: "arrival order cards".
- Layout zonasi: "ETA countdown label".
- Layout zonasi: "acknowledge action row".

#### Controloverview

- Boshqaruv elementi: "manual arrival confirm button".
- Boshqaruv elementi: "order card tap target".

#### Iconoverview

- Ikona joylashuvi: "arrival arrow icon".
- Ikona joylashuvi: "status indicator".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "acknowledge arrival".

---

**Summary:** Oqim quyidagicha qayd etilgan: "open selected order".

---


#### Dependencyoverview

##### Reads

- orders filtered by ARRIVED

##### Writes

- confirm arrival endpoint

##### Localizednotes

- O'qish: "orders filtered by ARRIVED".
- Yozish: "confirm arrival endpoint".

#### Stateoverview

- Holat: "arrival cards present".
- Holat: "no arrivals".

#### Figureoverview

- Figura: "arrival card list with confirm action".

#### Minifeatureoverview

- Mini-feature: "ETA countdown".
- Mini-feature: "confirm arrival action".
- Mini-feature: "arrival icon".
- Mini-feature: "order cards".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-future-demand

**Viewname:** FutureDemandView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/FutureDemandView.swift

**Localizedsummary:** "ios-retailer-future-demand" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-future-demand" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** AI demand forecast gallery with confidence indicators and procurement handoff.

#### Layoutoverview

- Layout zonasi: "forecast header card".
- Layout zonasi: "stats strip".
- Layout zonasi: "forecast card list".
- Layout zonasi: "close control".

#### Controloverview

- Boshqaruv elementi: "close button".
- Boshqaruv elementi: "drill-down to procurement control".

#### Iconoverview

- Ikona joylashuvi: "sparkles icon".
- Ikona joylashuvi: "confidence ring graphics".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "review predicted quantities and confidence".

---

**Summary:** Oqim quyidagicha qayd etilgan: "move into procurement workflow".

---


#### Dependencyoverview

##### Reads

- AI demand forecasts

##### Writes


##### Localizednotes

- O'qish: "AI demand forecasts".
- Yozish: yo'q.

#### Stateoverview

- Holat: "forecasts present".
- Holat: "empty forecasts".

#### Figureoverview

- Figura: "future demand modal with confidence rings".

#### Minifeatureoverview

- Mini-feature: "sparkles header".
- Mini-feature: "confidence rings".
- Mini-feature: "stats strip".
- Mini-feature: "forecast cards".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-auto-order

**Viewname:** AutoOrderView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/AutoOrderView.swift

**Localizedsummary:** "ios-retailer-auto-order" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-auto-order" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Hierarchical auto-order governance for supplier, category, and product scopes.

#### Layoutoverview

- Layout zonasi: "header".
- Layout zonasi: "suggestion cards".
- Layout zonasi: "checkbox or toggle rows".
- Layout zonasi: "bulk action bar".

#### Controloverview

- Boshqaruv elementi: "row toggles".
- Boshqaruv elementi: "Select All".
- Boshqaruv elementi: "Deselect All".
- Boshqaruv elementi: "submit action".

#### Iconoverview

- Ikona joylashuvi: "scope icons and checkmarks".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "select targets for auto-order".

---

**Summary:** Oqim quyidagicha qayd etilgan: "choose history or fresh behavior".

---

**Summary:** Oqim quyidagicha qayd etilgan: "submit updated settings".

---


#### Dependencyoverview

##### Reads

- auto-order settings

##### Writes

- auto-order action endpoints

##### Localizednotes

- O'qish: "auto-order settings".
- Yozish: "auto-order action endpoints".

#### Stateoverview

- Holat: "mixed toggles".
- Holat: "all off".
- Holat: "selection pending".

#### Figureoverview

- Figura: "auto-order hierarchy screen".

#### Minifeatureoverview

- Mini-feature: "global or scoped toggles".
- Mini-feature: "bulk select".
- Mini-feature: "submit".
- Mini-feature: "suggestion cards".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-insights

**Viewname:** InsightsView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/InsightsView.swift

**Localizedsummary:** "ios-retailer-insights" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-insights" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Expense analytics dashboard with selectable windows, KPI cards, charts, and top supplier or product breakdowns.

#### Layoutoverview

- Layout zonasi: "date-range filter row".
- Layout zonasi: "KPI cards".
- Layout zonasi: "chart region".
- Layout zonasi: "supplier and product expense tables".

#### Controloverview

- Boshqaruv elementi: "range buttons".

#### Iconoverview

- Ikona joylashuvi: "chart accents and analytics icons".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "change time horizon".

---

**Summary:** Oqim quyidagicha qayd etilgan: "refresh insights dataset".

---


#### Dependencyoverview

##### Reads

- retailer analytics endpoint

##### Writes


##### Localizednotes

- O'qish: "retailer analytics endpoint".
- Yozish: yo'q.

#### Stateoverview

- Holat: "analytics loaded".
- Holat: "empty analytics".

#### Figureoverview

- Figura: "insights dashboard with chart".

#### Minifeatureoverview

- Mini-feature: "range filters".
- Mini-feature: "KPI cards".
- Mini-feature: "expense chart".
- Mini-feature: "supplier table".
- Mini-feature: "product table".

**Minifeaturecount:** 5

---

**Pageid:** ios-retailer-profile

**Viewname:** ProfileView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ProfileView.swift

**Localizedsummary:** "ios-retailer-profile" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-profile" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Retailer account hub combining profile identity, settings links, support, and global empathy-engine auto-order toggles.

#### Layoutoverview

- Layout zonasi: "gradient header card".
- Layout zonasi: "stats row".
- Layout zonasi: "order history link".
- Layout zonasi: "empathy engine toggle card".
- Layout zonasi: "company and support sections".

#### Controloverview

- Boshqaruv elementi: "global auto-order toggle".
- Boshqaruv elementi: "settings row links".
- Boshqaruv elementi: "logout action".

#### Iconoverview

- Ikona joylashuvi: "avatar initials".
- Ikona joylashuvi: "settings row icons".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "toggle global auto-order with history or fresh branching".

---

**Summary:** Oqim quyidagicha qayd etilgan: "open settings links".

---

**Summary:** Oqim quyidagicha qayd etilgan: "logout".

---


#### Dependencyoverview

##### Reads

- retailer profile

##### Writes

- global auto-order endpoint

##### Localizednotes

- O'qish: "retailer profile".
- Yozish: "global auto-order endpoint".

#### Stateoverview

- Holat: "profile loaded".
- Holat: "toggle dialog open".

#### Figureoverview

- Figura: "profile page with gradient header".

#### Minifeatureoverview

- Mini-feature: "gradient profile header".
- Mini-feature: "stats row".
- Mini-feature: "history link".
- Mini-feature: "global auto-order".
- Mini-feature: "settings sections".
- Mini-feature: "logout".

**Minifeaturecount:** 6

---

**Pageid:** ios-retailer-procurement

**Viewname:** ProcurementView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ProcurementView.swift

**Localizedsummary:** "ios-retailer-procurement" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-procurement" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** AI-assisted procurement composer where predicted line items can be accepted, edited, and submitted as an order basket.

#### Layoutoverview

- Layout zonasi: "header card".
- Layout zonasi: "suggested items list".
- Layout zonasi: "quantity spinners".
- Layout zonasi: "bulk action bar".

#### Controloverview

- Boshqaruv elementi: "item checkbox".
- Boshqaruv elementi: "quantity controls".
- Boshqaruv elementi: "Select All".
- Boshqaruv elementi: "Deselect All".
- Boshqaruv elementi: "Submit Order".

#### Iconoverview

- Ikona joylashuvi: "confidence or suggestion icons".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "toggle predicted items".

---

**Summary:** Oqim quyidagicha qayd etilgan: "manually adjust quantities".

---

**Summary:** Oqim quyidagicha qayd etilgan: "submit procurement order".

---


#### Dependencyoverview

##### Reads

- AI demand forecasts

##### Writes

- procurement order endpoint

##### Localizednotes

- O'qish: "AI demand forecasts".
- Yozish: "procurement order endpoint".

#### Stateoverview

- Holat: "suggestions present".
- Holat: "none selected".
- Holat: "submit pending".

#### Figureoverview

- Figura: "procurement suggestion screen".

#### Minifeatureoverview

- Mini-feature: "suggestion list".
- Mini-feature: "checkboxes".
- Mini-feature: "quantity spinners".
- Mini-feature: "bulk actions".
- Mini-feature: "Submit Order".

**Minifeaturecount:** 5

---

**Pageid:** ios-retailer-inbox

**Viewname:** InboxView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/InboxView.swift

**Localizedsummary:** "ios-retailer-inbox" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-inbox" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Operational inbox of arriving, loaded, and in-transit orders for attention routing.

#### Layoutoverview

- Layout zonasi: "order feed list".
- Layout zonasi: "status pills".
- Layout zonasi: "ETA countdown or timing labels".

#### Controloverview

- Boshqaruv elementi: "order card tap target".

#### Iconoverview

- Ikona joylashuvi: "supplier badges".
- Ikona joylashuvi: "status pills".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "refresh inbox feed".

---

**Summary:** Oqim quyidagicha qayd etilgan: "open order from feed".

---


#### Dependencyoverview

##### Reads

- orders filtered for transit statuses

##### Writes


##### Localizednotes

- O'qish: "orders filtered for transit statuses".
- Yozish: yo'q.

#### Stateoverview

- Holat: "feed populated".
- Holat: "empty feed".

#### Figureoverview

- Figura: "inbox feed with status pills".

#### Minifeatureoverview

- Mini-feature: "status filter logic".
- Mini-feature: "feed cards".
- Mini-feature: "ETA labels".
- Mini-feature: "supplier badge".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-history

**Viewname:** HistoryView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/HistoryView.swift

**Localizedsummary:** "ios-retailer-history" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-history" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Historical order browser with status chip filters and drill-down entry.

#### Layoutoverview

- Layout zonasi: "status chip scroller".
- Layout zonasi: "order history card list".

#### Controloverview

- Boshqaruv elementi: "status chips".
- Boshqaruv elementi: "order card tap target".

#### Iconoverview

- Ikona joylashuvi: "status chip accents".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "filter by status".

---

**Summary:** Oqim quyidagicha qayd etilgan: "refresh history".

---

**Summary:** Oqim quyidagicha qayd etilgan: "open specific order".

---


#### Dependencyoverview

##### Reads

- orders by status

##### Writes


##### Localizednotes

- O'qish: "orders by status".
- Yozish: yo'q.

#### Stateoverview

- Holat: "all orders".
- Holat: "filtered subset".
- Holat: "empty filter result".

#### Figureoverview

- Figura: "history screen with status chips".

#### Minifeatureoverview

- Mini-feature: "status chips".
- Mini-feature: "history list".
- Mini-feature: "refresh".

**Minifeaturecount:** 3

---

**Pageid:** ios-retailer-search

**Viewname:** SearchView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/SearchView.swift

**Localizedsummary:** "ios-retailer-search" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-search" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Global product search across suppliers and categories.

#### Layoutoverview

- Layout zonasi: "search bar".
- Layout zonasi: "clear affordance".
- Layout zonasi: "empty search prompt".
- Layout zonasi: "result card grid".

#### Controloverview

- Boshqaruv elementi: "clear search button".
- Boshqaruv elementi: "product card tap target".

#### Iconoverview

- Ikona joylashuvi: "magnifying glass".
- Ikona joylashuvi: "clear xmark".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "type search query".

---

**Summary:** Oqim quyidagicha qayd etilgan: "clear query".

---

**Summary:** Oqim quyidagicha qayd etilgan: "open selected product detail".

---


#### Dependencyoverview

##### Reads

- products search endpoint

##### Writes


##### Localizednotes

- O'qish: "products search endpoint".
- Yozish: yo'q.

#### Stateoverview

- Holat: "empty query".
- Holat: "results shown".
- Holat: "no results".

#### Figureoverview

- Figura: "global product search screen".

#### Minifeatureoverview

- Mini-feature: "search bar".
- Mini-feature: "clear button".
- Mini-feature: "result grid".
- Mini-feature: "empty prompt".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-location-picker

**Viewname:** LocationPickerView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/LocationPickerView.swift

**Localizedsummary:** "ios-retailer-location-picker" yuzasi uchun unknown-role roli va unknown-platform platformasidagi lokalizatsiya qilingan ko'rinish.

### Localized

**Purpose:** "ios-retailer-location-picker" yuzasi unknown-platform platformasida unknown-role roli uchun ekran sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** MapKit location picker for retailer signup and location updates.

#### Layoutoverview

- Layout zonasi: "map view".
- Layout zonasi: "center pin".
- Layout zonasi: "address label".
- Layout zonasi: "close control".
- Layout zonasi: "Confirm Location button".

#### Controloverview

- Boshqaruv elementi: "close xmark button".
- Boshqaruv elementi: "Confirm Location button".

#### Iconoverview

- Ikona joylashuvi: "mappin circle and arrowtriangle center glyph".

#### Flowoverview

**Summary:** Oqim quyidagicha qayd etilgan: "move map under fixed pin".

---

**Summary:** Oqim quyidagicha qayd etilgan: "resolve address".

---

**Summary:** Oqim quyidagicha qayd etilgan: "confirm chosen location".

---


#### Dependencyoverview

##### Reads

- MapKit reverse geocoder state

##### Writes

- selected location callback

##### Localizednotes

- O'qish: "MapKit reverse geocoder state".
- Yozish: "selected location callback".

#### Stateoverview

- Holat: "default location".
- Holat: "adjusted location".

#### Figureoverview

- Figura: "iOS location picker with centered pin".

#### Minifeatureoverview

- Mini-feature: "map pin".
- Mini-feature: "address label".
- Mini-feature: "close button".
- Mini-feature: "confirm CTA".

**Minifeaturecount:** 4

---


---

**Dossierfile:** supplier-orders.json

**Pageid:** web-supplier-orders

**Route:** /supplier/orders

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/orders/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-orders" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-orders" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier order-lifecycle command page for approval, reassignment, search, filtering, history viewing, and order inspection.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "left: headline: Orders; subtitle: Manage the full order lifecycle — approval, dispatch, tracking, and history"; "right: Refresh button".
- "tab-row" zonasi "below divider" hududida joylashgan. Tarkibi: "Active tab button with orders icon and count badge"; "Scheduled tab button with schedule icon and count badge when active".
- "filter-row" zonasi "below tabs" hududida joylashgan. Tarkibi: "search input for order ID or retailer"; "state filter select"; "History toggle chip with ledger icon".
- "bulk-action-bar" zonasi "conditional row above table" hududida joylashgan. Ko'rinish qoidasi: "visible when one or more rows are selected". Tarkibi: "left: selected count"; "right: Reassign Truck button when selection is eligible; Clear button".
- "table-region" zonasi "primary content card" hududida joylashgan. Tarkibi: "select-all checkbox header"; "order ID column"; "retailer column"; "state badge column"; "truck or route column"; "delivery date column"; "items column"; "amount column"; "payment column"; "created timestamp column"; "actions column".

### Controloverview

- "Refresh" tugmasi "header-right" hududida joylashgan. Uslub: "secondary". Ikona: "returns".
- "Active tab" tugmasi "tab-row-left" hududida joylashgan. Uslub: "tab". Ikona: "orders".
- "Scheduled tab" tugmasi "tab-row-left" hududida joylashgan. Uslub: "tab". Ikona: "schedule".
- "History" tugmasi "filter-row-right" hududida joylashgan. Uslub: "chip-toggle". Ikona: "ledger".
- "Approve" tugmasi "table-row-actions" hududida joylashgan. Uslub: "primary". Ko'rinish qoidasi: "active tab and order state is PENDING".
- "Reject" tugmasi "table-row-actions" hududida joylashgan. Uslub: "outline-danger". Ko'rinish qoidasi: "active tab and order state is PENDING".
- "Reassign" tugmasi "table-row-actions" hududida joylashgan. Uslub: "secondary". Ko'rinish qoidasi: "active tab and order state is PENDING or LOADED and route_id exists".
- "Reassign Truck" tugmasi "bulk-action-bar-right" hududida joylashgan. Uslub: "primary". Ko'rinish qoidasi: "selected rows all eligible for reassignment".
- "Clear" tugmasi "bulk-action-bar-right" hududida joylashgan. Uslub: "ghost".
- "Cancel" tugmasi "reassign-dialog-footer-left" hududida joylashgan. Uslub: "ghost".
- "Reassign N Order(s)" tugmasi "reassign-dialog-footer-right" hududida joylashgan. Uslub: "primary".
- "Reassign to Different Truck" tugmasi "detail-drawer-footer" hududida joylashgan. Uslub: "secondary". Ko'rinish qoidasi: "drawer open and order has route_id and state is PENDING or LOADED".

### Iconoverview

- "returns" ikonasi "header-right refresh button" zonasida ishlatiladi.
- "orders" ikonasi "Active tab button" zonasida ishlatiladi.
- "schedule" ikonasi "Scheduled tab button" zonasida ishlatiladi.
- "ledger" ikonasi "History filter chip" zonasida ishlatiladi.
- "StatusBadge" ikonasi "state column and detail drawer" zonasida ishlatiladi.

### Flowoverview

**Flowid:** approve-pending-order

**Summary:** "approve-pending-order" oqimi 5 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier stays on Active tab".
- 2-qadam: "Clicks Approve in row actions".
- 3-qadam: "Page posts to /v1/supplier/orders/vet".
- 4-qadam: "Toast shows result".
- 5-qadam: "Orders reload".

---

**Flowid:** reject-pending-order

**Summary:** "reject-pending-order" oqimi 5 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier clicks Reject in row actions".
- 2-qadam: "Inline reason input appears in actions column".
- 3-qadam: "Supplier types reason and confirms Reject".
- 4-qadam: "Page posts to /v1/supplier/orders/vet with decision REJECTED".
- 5-qadam: "Toast shows result and rows reload".

---

**Flowid:** single-or-bulk-reassign

**Summary:** "single-or-bulk-reassign" oqimi 7 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier clicks Reassign in a row or selects multiple eligible rows".
- 2-qadam: "Dialog opens".
- 3-qadam: "Supplier selects target truck".
- 4-qadam: "Capacity metrics and capacity bar render".
- 5-qadam: "Supplier confirms reassignment".
- 6-qadam: "Page posts to /v1/fleet/reassign".
- 7-qadam: "Toast summarizes reassign or conflict results".

---

**Flowid:** row-inspection

**Summary:** "row-inspection" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier clicks any table row".
- 2-qadam: "Order detail drawer opens from right".
- 3-qadam: "Drawer shows ID, status, retailer, payment, assignment, timestamps, and optional reassign CTA".

---


### Dependencyoverview

#### Reads

- /v1/supplier/orders
- /v1/fleet/active
- /v1/fleet/capacity

#### Writes

- /v1/supplier/orders/vet
- /v1/fleet/reassign

#### Localizednotes

- O'qish: "/v1/supplier/orders", "/v1/fleet/active", "/v1/fleet/capacity".
- Yozish: "/v1/supplier/orders/vet", "/v1/fleet/reassign".
- Yangilash modeli: "manual refresh plus 30-second polling when not in history mode".

### Stateoverview

- Holat: "loading skeleton rows".
- Holat: "empty state by active or scheduled or history context".
- Holat: "normal data table".
- Holat: "selected rows accent-soft highlighting".
- Holat: "reject inline-input mode".
- Holat: "detail drawer open".
- Holat: "reassignment dialog open".

### Figureoverview

- Figura: "full-page command view with tab row and filter row".
- Figura: "table row with approve and reject controls".
- Figura: "bulk-action bar with selected rows and reassign CTA".
- Figura: "reassignment dialog with capacity bar".
- Figura: "order detail drawer".

---

**Dossierfile:** web-supplier-analytics-demand.json

**Pageid:** web-supplier-analytics-demand

**Route:** /supplier/analytics/demand

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/analytics/demand/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-analytics-demand" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-analytics-demand" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier advanced demand-forecast page comparing predicted versus actual order volume over time and listing upcoming AI-planned order line items.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "circular back button linking to analytics hub"; "headline: AI Demand Analytics"; "subtitle describing predicted versus actual volume over a 30-day window".
- "kpi-row" zonasi "below header" hududida joylashgan. Tarkibi: "Prediction Accuracy card"; "Upcoming AI Orders card"; "Data Points card".
- "chart-card" zonasi "below KPI row" hududida joylashgan. Tarkibi: "dual-axis line chart"; "legend"; "tooltip-driven predicted and actual value inspection"; "empty chart message when time series is absent".
- "upcoming-orders-card" zonasi "bottom full-width" hududida joylashgan. Tarkibi: "table header"; "upcoming AI-planned order rows with date, retailer, SKU, product, predicted quantity"; "pagination controls or empty-state message".

### Controloverview

- "Back to analytics" tugmasi "header far-left circular control" hududida joylashgan. Uslub: "round surface button".

### Iconoverview

- "left-arrow glyph" ikonasi "header back button" zonasida ishlatiladi.

### Flowoverview

**Flowid:** demand-history-bootstrap

**Summary:** "demand-history-bootstrap" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page reads supplier token from cookie".
- 2-qadam: "If token is absent, page renders supplier-credentials-required error state".
- 3-qadam: "Otherwise page fetches /v1/supplier/analytics/demand/history and binds time series plus upcoming rows".

---

**Flowid:** chart-review

**Summary:** "chart-review" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier inspects dual-axis lines for predicted and actual value and quantity".
- 2-qadam: "Tooltip exposes exact UZS and quantity values for a selected date".

---

**Flowid:** upcoming-order-review

**Summary:** "upcoming-order-review" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier reviews paginated upcoming AI-planned order rows".
- 2-qadam: "Shared pagination controls advance through the upcoming dataset".

---


### Dependencyoverview

#### Reads

- /v1/supplier/analytics/demand/history

#### Writes


#### Localizednotes

- O'qish: "/v1/supplier/analytics/demand/history".
- Yozish: yo'q.
- Yangilash modeli: "single fetch on mount with local pagination over the upcoming rows".

### Stateoverview

- Holat: "page loading spinner".
- Holat: "unauthorized error card".
- Holat: "history error card".
- Holat: "chart with time-series data".
- Holat: "chart empty state with no time-series data available".
- Holat: "upcoming rows table with pagination".
- Holat: "upcoming rows empty message".

### Figureoverview

- Figura: "full advanced demand analytics page with header, KPI row, chart, and table".
- Figura: "chart card close-up showing four line series and legend".
- Figura: "upcoming orders table close-up with pagination footer".

---

**Dossierfile:** web-supplier-analytics.json

**Pageid:** web-supplier-analytics

**Route:** /supplier/analytics

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/analytics/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-analytics" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-analytics" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier analytics hub presenting financial velocity, AI demand highlights, and deep links into advanced forecast review and dispatch operations.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "headline: Analytics"; "subtitle describing financial overview and operational intelligence"; "Demand Forecast CTA"; "Dispatch Room CTA".
- "ai-demand-card" zonasi "below header when demand predictions exist" hududida joylashgan. Ko'rinish qoidasi: "visible when demand prediction_count is greater than zero". Tarkibi: "analytics avatar circle"; "prediction count chip"; "three-column forecast metrics"; "forecasted SKU pills"; "View Advanced Analytics CTA".
- "kpi-grid" zonasi "below demand card or directly below header" hududida joylashgan. Tarkibi: "Gross Volume card"; "Total Pallets card"; "Avg Velocity per SKU card"; "Top SKU card".
- "velocity-chart-region" zonasi "below KPI grid" hududida joylashgan. Tarkibi: "VelocityChart component spanning page width".
- "sku-breakdown-table" zonasi "bottom full-width card when velocity data exists" hududida joylashgan. Tarkibi: "table with SKU ID, pallets, volume, and share bar".

### Controloverview

- "Demand Forecast" tugmasi "header top-right" hududida joylashgan. Uslub: "soft accent link button".
- "Dispatch Room" tugmasi "header top-right" hududida joylashgan. Uslub: "filled accent link button".
- "View Advanced Analytics" tugmasi "ai-demand-card footer" hududida joylashgan. Uslub: "pill button". Ko'rinish qoidasi: "visible when demand card is rendered".

### Iconoverview

- "error" ikonasi "error card" zonasida ishlatiladi.
- "analytics" ikonasi "Demand Forecast header CTA and AI demand card avatar" zonasida ishlatiladi.
- "orders" ikonasi "Dispatch Room header CTA" zonasida ishlatiladi.
- "arrow_forward" ikonasi "View Advanced Analytics CTA trailing icon" zonasida ishlatiladi.

### Flowoverview

**Flowid:** analytics-bootstrap

**Summary:** "analytics-bootstrap" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page fetches /v1/supplier/analytics/velocity and /v1/supplier/analytics/demand/today in parallel using apiFetch".
- 2-qadam: "Loading skeletons occupy the KPI and chart regions until both requests settle".
- 3-qadam: "Velocity metrics derive from the returned data array and top SKU is computed client-side".

---

**Flowid:** forecast-escalation

**Summary:** "forecast-escalation" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier selects Demand Forecast in the header or View Advanced Analytics in the AI demand card".
- 2-qadam: "Navigation moves to /supplier/analytics/demand for deeper analysis".

---

**Flowid:** dispatch-escalation

**Summary:** "dispatch-escalation" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier selects Dispatch Room".
- 2-qadam: "Navigation leaves analytics and returns to the operational command surface linked by the app root".

---


### Dependencyoverview

#### Reads

- /v1/supplier/analytics/velocity
- /v1/supplier/analytics/demand/today

#### Writes


#### Localizednotes

- O'qish: "/v1/supplier/analytics/velocity", "/v1/supplier/analytics/demand/today".
- Yozish: yo'q.
- Yangilash modeli: "single load on mount with computed metrics derived in memory".

### Stateoverview

- Holat: "skeleton loading state".
- Holat: "error card state".
- Holat: "analytics hub with AI demand card visible".
- Holat: "analytics hub without AI demand card".
- Holat: "velocity chart with SKU breakdown table".
- Holat: "analytics hub with no velocity rows and no bottom table".

### Figureoverview

- Figura: "full analytics hub with AI demand card, KPI grid, chart, and breakdown table".
- Figura: "AI demand card close-up with metric triplet and forecast pills".
- Figura: "SKU breakdown row with share bar".

---

**Dossierfile:** web-supplier-catalog.json

**Pageid:** web-supplier-catalog

**Route:** /supplier/catalog

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/catalog/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-catalog" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-catalog" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier catalog control page combining product creation, promotion creation, category filtering, product-ledger review, status toggling, and modal-based product editing.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "headline: Inventory Control"; "subtitle: Catalog Injection, Product Ledger and Promotional Routing".
- "kpi-strip" zonasi "below header" hududida joylashgan. Tarkibi: "Total SKUs card"; "Active card"; "Inactive card"; "Catalog Value card".
- "category-filter-row" zonasi "below KPI strip when operating categories exist" hududida joylashgan. Tarkibi: "All chip with total count"; "category chips with per-category counts".
- "dual-form-region" zonasi "two-column band" hududida joylashgan. Tarkibi: "SupplierProductForm on left"; "SupplierPromotionForm on right".
- "product-ledger-table" zonasi "bottom full-width ledger card" hududida joylashgan. Tarkibi: "ledger header with product count chip"; "product table with image cell, category chip, price, VU, block, MOQ-step, status, truncated SKU, actions"; "empty state when no products match filter".
- "edit-product-modal" zonasi "centered modal overlay" hududida joylashgan. Ko'rinish qoidasi: "visible when editProduct exists". Tarkibi: "modal header with title and close button"; "optional error banner"; "name input"; "description textarea"; "base price input"; "MOQ, Step, Units-per-Block triplet"; "image preview and file picker"; "Cancel and Save Changes footer buttons".

### Controloverview

- "All" tugmasi "category-filter-row" hududida joylashgan. Uslub: "chip".
- "Category chip" tugmasi "category-filter-row" hududida joylashgan. Uslub: "chip".
- "Edit" tugmasi "product-ledger actions column" hududida joylashgan. Uslub: "soft accent small".
- "Deactivate" tugmasi "product-ledger actions column" hududida joylashgan. Uslub: "soft danger small". Ko'rinish qoidasi: "product is active".
- "Activate" tugmasi "product-ledger actions column" hududida joylashgan. Uslub: "soft success small". Ko'rinish qoidasi: "product is inactive".
- "Close" tugmasi "edit-product-modal header-right" hududida joylashgan. Uslub: "small muted".
- "Cancel" tugmasi "edit-product-modal footer-left" hududida joylashgan. Uslub: "pill muted".
- "Save Changes" tugmasi "edit-product-modal footer-right" hududida joylashgan. Uslub: "pill primary".

### Iconoverview

- "image placeholder glyph" ikonasi "product table image cell when image_url missing" zonasida ishlatiladi.
- "catalog" ikonasi "ledger empty state" zonasida ishlatiladi.
- "status badge chip" ikonasi "status column" zonasida ishlatiladi.

### Flowoverview

**Flowid:** catalog-bootstrap

**Summary:** "catalog-bootstrap" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page fetches /v1/supplier/products and /v1/supplier/profile in parallel".
- 2-qadam: "If operating categories are present, page also maps them against /v1/catalog/platform-categories".
- 3-qadam: "KPI strip and filtered ledger compute from catalog payload".

---

**Flowid:** category-filtering

**Summary:** "category-filtering" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier clicks All or a category chip".
- 2-qadam: "Filtered ledger, active count, inactive count, and catalog value recompute client-side".

---

**Flowid:** product-editing

**Summary:** "product-editing" oqimi 6 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier clicks Edit in ledger row".
- 2-qadam: "Edit modal opens prefilled with product data".
- 3-qadam: "Supplier may replace image, update quantities, price, and copy".
- 4-qadam: "Page optionally obtains upload ticket from /v1/supplier/products/upload-ticket and uploads image".
- 5-qadam: "Page puts updated payload to /v1/supplier/products/{sku_id}".
- 6-qadam: "Catalog reloads".

---

**Flowid:** status-toggle

**Summary:** "status-toggle" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier clicks Activate or Deactivate in row action cluster".
- 2-qadam: "Page updates is_active through /v1/supplier/products/{sku_id}".
- 3-qadam: "Ledger refreshes".

---


### Dependencyoverview

#### Reads

- /v1/supplier/products
- /v1/supplier/profile
- /v1/catalog/platform-categories

#### Writes

- /v1/supplier/products/{sku_id}
- /v1/supplier/products/upload-ticket

#### Localizednotes

- O'qish: "/v1/supplier/products", "/v1/supplier/profile", "/v1/catalog/platform-categories".
- Yozish: "/v1/supplier/products/{sku_id}", "/v1/supplier/products/upload-ticket".
- Yangilash modeli: "initial fetch on mount plus targeted reload after edit and status-toggle actions".

### Stateoverview

- Holat: "page spinner loading state".
- Holat: "error card state".
- Holat: "ledger empty state with forms still visible".
- Holat: "ledger with image thumbnails and category chips".
- Holat: "row-level toggle pending state".
- Holat: "edit-product modal open".
- Holat: "edit-product modal with upload preview".
- Holat: "edit-product modal saving state".

### Figureoverview

- Figura: "full catalog page with KPI strip, filters, two-column forms, and ledger".
- Figura: "product-ledger table close-up".
- Figura: "edit-product modal".
- Figura: "ledger row with image placeholder and activation toggle".

---

**Dossierfile:** web-supplier-crm.json

**Pageid:** web-supplier-crm

**Route:** /supplier/crm

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/crm/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-crm" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-crm" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier retailer-relationship page for tracking retailer lifetime value, order history, and contact detail in a table-plus-drawer CRM workspace.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "headline: Retailer CRM"; "subtitle describing retailer relationship and lifetime-value tracking".
- "kpi-grid" zonasi "below header" hududida joylashgan. Tarkibi: "Total Retailers card"; "Active card"; "Total Lifetime Value card".
- "retailer-ledger" zonasi "main card region" hududida joylashgan. Tarkibi: "loading spinner or CRM empty state"; "table of retailers with avatar initials, lifetime value, order count, last order date, and status chip"; "pagination controls".
- "retailer-detail-drawer" zonasi "slide-out side drawer" hududida joylashgan. Ko'rinish qoidasi: "visible when slideOpen is true". Tarkibi: "retailer initials tile"; "status chip"; "contact links for phone and email"; "lifetime value and total orders KPI cards"; "order ledger list with state dot, item count, amount, and date".

### Controloverview

- "Retailer row" tugmasi "retailer-ledger body rows" hududida joylashgan. Uslub: "full-row tap target opening detail drawer".

### Iconoverview

- "crm" ikonasi "CRM empty state" zonasida ishlatiladi.
- "phone" ikonasi "detail drawer contact row" zonasida ishlatiladi.
- "email" ikonasi "detail drawer contact row" zonasida ishlatiladi.

### Flowoverview

**Flowid:** crm-bootstrap

**Summary:** "crm-bootstrap" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page reads supplier token and fetches /v1/supplier/crm/retailers".
- 2-qadam: "Returned retailer rows populate KPI cards and paginated ledger".

---

**Flowid:** detail-drawer-open

**Summary:** "detail-drawer-open" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier clicks a retailer row".
- 2-qadam: "Drawer opens immediately in loading state".
- 3-qadam: "Page requests /v1/supplier/crm/retailers/{retailer_id}".
- 4-qadam: "If detail fetch fails, page falls back to a synthesized order-history sample based on the selected base record".

---

**Flowid:** contact-escalation

**Summary:** "contact-escalation" oqimi 1 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier can use tel: and mailto: links in the drawer contact section to escalate directly to retailer communications".

---


### Dependencyoverview

#### Reads

- /v1/supplier/crm/retailers
- /v1/supplier/crm/retailers/{retailer_id}

#### Writes


#### Localizednotes

- O'qish: "/v1/supplier/crm/retailers", "/v1/supplier/crm/retailers/{retailer_id}".
- Yozish: yo'q.
- Yangilash modeli: "single fetch for retailer list plus on-demand detail fetch when a row is opened".

### Stateoverview

- Holat: "table loading spinner state".
- Holat: "CRM empty state".
- Holat: "retailer table with pagination".
- Holat: "detail drawer loading spinner".
- Holat: "detail drawer with contact and order ledger".
- Holat: "detail drawer empty order ledger".

### Figureoverview

- Figura: "full CRM page with KPI grid and retailer ledger".
- Figura: "retailer detail drawer over table background".
- Figura: "drawer order-ledger close-up with state dots and amount column".

---

**Dossierfile:** web-supplier-dashboard.json

**Pageid:** web-supplier-dashboard

**Route:** /supplier/dashboard

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/dashboard/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-dashboard" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-dashboard" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier analytics landing page combining forward-demand intelligence, SKU velocity summaries, financial KPIs, and volume-share tables.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "left: headline: Analytics; subtitle: Financial overview and operational intelligence"; "right: Dispatch Control Room link button with clipboard-style glyph".
- "future-demand-card" zonasi "below header when predictions exist" hududida joylashgan. Ko'rinish qoidasi: "visible when demand payload exists and prediction_count > 0". Tarkibi: "lightning icon badge"; "AI Future Demand headline and helper copy"; "prediction-count status chip"; "three-metric strip for retailers, pallets, forecast value"; "forecast item pills"; "View Advanced Analytics CTA".
- "kpi-grid" zonasi "below future-demand card" hududida joylashgan. Tarkibi: "Gross Volume card"; "Total Pallets Moved card"; "Avg Velocity per SKU card"; "Top Performing SKU card".
- "velocity-chart" zonasi "mid-page primary visualization" hududida joylashgan. Tarkibi: "VelocityChart component showing SKU volume performance".
- "sku-breakdown-table" zonasi "bottom card" hududida joylashgan. Ko'rinish qoidasi: "visible when velocityData length > 0". Tarkibi: "SKU ID column"; "pallet count column"; "gross volume column"; "share column with right-aligned progress bar and percentage text".

### Controloverview

- "Dispatch Control Room" tugmasi "header-right" hududida joylashgan. Uslub: "filled pill link". Ikona: "manifest clipboard glyph".
- "View Advanced Analytics" tugmasi "future-demand-card footer" hududida joylashgan. Uslub: "filled rounded CTA". Ikona: "right arrow".

### Iconoverview

- "manifest clipboard glyph" ikonasi "header-right dispatch link" zonasida ishlatiladi.
- "lightning bolt" ikonasi "future-demand-card leading badge" zonasida ishlatiladi.
- "right arrow" ikonasi "future-demand-card CTA trailing edge" zonasida ishlatiladi.
- "progress fill bar" ikonasi "share column in sku-breakdown-table" zonasida ishlatiladi.

### Flowoverview

**Flowid:** analytics-bootstrap

**Summary:** "analytics-bootstrap" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page reads supplier token from cookie".
- 2-qadam: "Page requests /v1/supplier/analytics/velocity and /v1/supplier/analytics/demand/today in parallel".
- 3-qadam: "KPI cards, chart, demand card, and table compute derived totals from returned payloads".
- 4-qadam: "If demand payload is absent, only analytics baseline regions remain".

---

**Flowid:** advanced-demand-drilldown

**Summary:** "advanced-demand-drilldown" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier reviews AI demand summary card".
- 2-qadam: "Supplier clicks View Advanced Analytics".
- 3-qadam: "Page routes to /supplier/analytics/demand".

---

**Flowid:** dispatch-linkout

**Summary:** "dispatch-linkout" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier uses header CTA".
- 2-qadam: "Page routes to /supplier/manifests legacy dispatch surface".

---


### Dependencyoverview

#### Reads

- /v1/supplier/analytics/velocity
- /v1/supplier/analytics/demand/today

#### Writes


#### Localizednotes

- O'qish: "/v1/supplier/analytics/velocity", "/v1/supplier/analytics/demand/today".
- Yozish: yo'q.
- Yangilash modeli: "single fetch on mount; data remains static until navigation or reload".

### Stateoverview

- Holat: "page-level loading skeleton with placeholder header, KPI cards, and chart block".
- Holat: "unauthorized error card".
- Holat: "analytics-only state with no demand card".
- Holat: "full intelligence state with demand card and forecast pills".
- Holat: "table hidden when velocityData is empty".

### Figureoverview

- Figura: "full analytics dashboard with future-demand card and KPI grid".
- Figura: "AI Future Demand card close-up with forecast pills".
- Figura: "velocity chart region".
- Figura: "SKU breakdown table with share bar column".

---

**Dossierfile:** web-supplier-depot-reconciliation.json

**Pageid:** web-supplier-depot-reconciliation

**Route:** /supplier/depot-reconciliation

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/depot-reconciliation/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-depot-reconciliation" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-depot-reconciliation" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier depot-reconciliation page for processing quarantined returned loads by vehicle, order, and individual line item, with restock and write-off actions.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "headline: Depot Reconciliation"; "subtitle describing returned loads awaiting restock or write-off"; "Refresh button".
- "vehicle-card-stack" zonasi "main vertical stack" hududida joylashgan. Tarkibi: "one vehicle card per quarantined vehicle with header summary and nested order sections".
- "vehicle-card-header" zonasi "top of each vehicle card" hududida joylashgan. Tarkibi: "fleet icon avatar"; "vehicle class, driver name, route identifier, and order count"; "Restock All button"; "Write Off All button".
- "order-section" zonasi "within each vehicle card" hududida joylashgan. Tarkibi: "order short ID and retailer name"; "quarantine pill"; "item table with product, quantity, unit price, and per-item actions".

### Controloverview

- "Retry" tugmasi "error fallback state" hududida joylashgan. Uslub: "secondary button".
- "Refresh" tugmasi "header top-right" hududida joylashgan. Uslub: "outline button with leading icon".
- "Restock All" tugmasi "vehicle-card-header action cluster" hududida joylashgan. Uslub: "secondary small button".
- "Write Off All" tugmasi "vehicle-card-header action cluster" hududida joylashgan. Uslub: "outline danger small button".
- "Restock" tugmasi "item row actions" hududida joylashgan. Uslub: "secondary small button".
- "Write Off" tugmasi "item row actions" hududida joylashgan. Uslub: "outline danger small button".

### Iconoverview

- "error" ikonasi "error fallback" zonasida ishlatiladi.
- "warehouse" ikonasi "empty state" zonasida ishlatiladi.
- "refresh" ikonasi "Refresh button" zonasida ishlatiladi.
- "fleet" ikonasi "vehicle-card header avatar" zonasida ishlatiladi.

### Flowoverview

**Flowid:** quarantine-bootstrap

**Summary:** "quarantine-bootstrap" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page reads supplier token and fetches /v1/supplier/quarantine-stock".
- 2-qadam: "Loading placeholders occupy the card stack until quarantine vehicles resolve".

---

**Flowid:** bulk-vehicle-reconciliation

**Summary:** "bulk-vehicle-reconciliation" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier presses Restock All or Write Off All in a vehicle card header".
- 2-qadam: "All line_item_ids for that vehicle are posted to /v1/inventory/reconcile-returns with the selected action".
- 3-qadam: "Toast confirms success and vehicle card stack reloads".

---

**Flowid:** item-level-reconciliation

**Summary:** "item-level-reconciliation" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier presses Restock or Write Off in a line-item row".
- 2-qadam: "Single line_item_id is posted to /v1/inventory/reconcile-returns".
- 3-qadam: "Row and enclosing vehicle dataset refresh after reconciliation".

---


### Dependencyoverview

#### Reads

- /v1/supplier/quarantine-stock

#### Writes

- /v1/inventory/reconcile-returns

#### Localizednotes

- O'qish: "/v1/supplier/quarantine-stock".
- Yozish: "/v1/inventory/reconcile-returns".
- Yangilash modeli: "load on mount, manual refresh button, and automatic reload after reconciliation actions".

### Stateoverview

- Holat: "loading skeleton stack".
- Holat: "error fallback with retry button".
- Holat: "empty state with no quarantine stock".
- Holat: "vehicle stack with nested order sections".
- Holat: "action-in-progress state disabling reconciliation buttons".

### Figureoverview

- Figura: "full depot reconciliation page with stacked vehicle cards".
- Figura: "single vehicle card header with bulk action buttons".
- Figura: "order section close-up with quarantine pill and per-item restock and write-off controls".

---

**Dossierfile:** web-supplier-dispatch.json

**Pageid:** web-supplier-dispatch

**Route:** /supplier/dispatch

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented-as-redirect

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/dispatch/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-dispatch" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-dispatch" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Legacy supplier dispatch route that immediately redirects to the canonical supplier orders surface rather than rendering a dedicated dispatch page.

### Layoutoverview

- "redirect-guard" zonasi "server-side route handler" hududida joylashgan. Tarkibi: "no persisted UI; redirect('/supplier/orders') executes during route resolution".

### Controloverview


### Iconoverview


### Flowoverview

**Flowid:** route-alias-redirect

**Summary:** "route-alias-redirect" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier navigates to /supplier/dispatch".
- 2-qadam: "Next.js redirect executes immediately".
- 3-qadam: "Browser lands on /supplier/orders where dispatch actions are actually performed".

---


### Dependencyoverview

#### Reads


#### Writes


#### Localizednotes

- O'qish: yo'q.
- Yozish: yo'q.
- Yangilash modeli: "no local data fetch; route delegates to /supplier/orders".

### Stateoverview

- Holat: "immediate server redirect with no rendered intermediate page".

### Figureoverview

- Figura: "route-flow figure showing /supplier/dispatch aliasing into /supplier/orders".

---

**Dossierfile:** web-supplier-fleet.json

**Pageid:** web-supplier-fleet

**Route:** /supplier/fleet

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/fleet/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-fleet" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-fleet" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier fleet command center for driver provisioning, vehicle registration, assignment control, capacity visibility, and credential reveal for newly provisioned operators.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "left: back link to supplier dashboard; headline: Fleet Management; subtitle: Provision drivers, register vehicles, manage fleet capacity"; "right: conditional + Add Driver CTA when drivers tab active; conditional + Add Vehicle CTA when vehicles tab active".
- "tab-selector" zonasi "below header" hududida joylashgan. Tarkibi: "Drivers tab pill with count"; "Vehicles tab pill with count".
- "kpi-row" zonasi "below tab-selector" hududida joylashgan. Tarkibi: "driver metrics row when drivers tab active"; "vehicle metrics row when vehicles tab active".
- "primary-table-region" zonasi "main content card" hududida joylashgan. Tarkibi: "vehicle table with class, label, plate, capacity, assigned driver, status, actions when vehicles tab active"; "driver table with clickable rows, phone, type badge, assignment select, status when drivers tab active".
- "driver-add-drawer" zonasi "right slide-out overlay" hududida joylashgan. Ko'rinish qoidasi: "open when showAdd is true". Tarkibi: "name input"; "phone input"; "driver-type chip toggle"; "assign-vehicle select"; "license-plate input"; "inline error copy"; "Provision Driver button".
- "pin-reveal-modal" zonasi "centered modal overlay" hududida joylashgan. Ko'rinish qoidasi: "visible when createdPin exists". Tarkibi: "success icon disk"; "driver identity text"; "dashed login-pin panel"; "warning banner instructing copy-once behavior"; "Done button".
- "driver-detail-drawer" zonasi "right slide-out overlay" hududida joylashgan. Ko'rinish qoidasi: "open when selectedDriver exists". Tarkibi: "initial avatar circle"; "driver name and phone"; "type badge"; "detail cell grid"; "optional current-location row".
- "vehicle-add-drawer" zonasi "right slide-out overlay" hududida joylashgan. Ko'rinish qoidasi: "open when showAddVehicle is true". Tarkibi: "class-versus-dimensions mode toggle"; "vehicle-class select"; "computed capacity readout"; "dimension inputs when LxWxH mode selected"; "label input"; "license plate input"; "Register Vehicle button".

### Controloverview

- "+ Add Driver" tugmasi "header-right" hududida joylashgan. Uslub: "primary". Ko'rinish qoidasi: "drivers tab active".
- "+ Add Vehicle" tugmasi "header-right" hududida joylashgan. Uslub: "primary". Ko'rinish qoidasi: "vehicles tab active".
- "Drivers tab" tugmasi "tab-selector" hududida joylashgan. Uslub: "segmented tab".
- "Vehicles tab" tugmasi "tab-selector" hududida joylashgan. Uslub: "segmented tab".
- "Deactivate" tugmasi "vehicle row action cluster" hududida joylashgan. Uslub: "ghost-danger". Ko'rinish qoidasi: "vehicle is active".
- "Clear Returns" tugmasi "vehicle row action cluster" hududida joylashgan. Uslub: "outline-warning". Ko'rinish qoidasi: "vehicle active, assigned, and pending returns exist".
- "Provision Driver" tugmasi "driver-add-drawer footer" hududida joylashgan. Uslub: "full-width primary".
- "Done" tugmasi "pin-reveal-modal footer" hududida joylashgan. Uslub: "full-width primary".
- "Class" tugmasi "vehicle-add-drawer top-right mode toggle" hududida joylashgan. Uslub: "segmented button".
- "LxWxH" tugmasi "vehicle-add-drawer top-right mode toggle" hududida joylashgan. Uslub: "segmented button".
- "Register Vehicle" tugmasi "vehicle-add-drawer footer" hududida joylashgan. Uslub: "full-width primary".

### Iconoverview

- "warning" ikonasi "pin-reveal-modal warning banner" zonasida ishlatiladi.
- "success checkmark disk" ikonasi "pin-reveal-modal top" zonasida ishlatiladi.
- "driver initial avatar" ikonasi "driver-detail-drawer header" zonasida ishlatiladi.
- "driver-type badge" ikonasi "driver rows and driver-detail-drawer header" zonasida ishlatiladi.

### Flowoverview

**Flowid:** driver-provisioning

**Summary:** "driver-provisioning" oqimi 5 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier stays on Drivers tab".
- 2-qadam: "Clicks + Add Driver".
- 3-qadam: "Completes drawer form and optionally preassigns vehicle".
- 4-qadam: "Page posts to /v1/supplier/fleet/drivers".
- 5-qadam: "Drawer closes and one-time PIN modal appears".

---

**Flowid:** vehicle-registration

**Summary:** "vehicle-registration" oqimi 5 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier switches to Vehicles tab".
- 2-qadam: "Clicks + Add Vehicle".
- 3-qadam: "Supplier either keeps class mode or enters dimensions for computed VU".
- 4-qadam: "Page posts to /v1/supplier/fleet/vehicles".
- 5-qadam: "Vehicle list reloads".

---

**Flowid:** assignment-control

**Summary:** "assignment-control" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier changes assignment select inside a driver row".
- 2-qadam: "Page patches /v1/supplier/fleet/drivers/{driverId}/assign-vehicle".
- 3-qadam: "Drivers and vehicles refresh to reflect occupancy state".

---

**Flowid:** driver-inspection

**Summary:** "driver-inspection" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier clicks a driver row".
- 2-qadam: "Page requests /v1/supplier/fleet/drivers/{id}".
- 3-qadam: "Right-side drawer opens with identity, stats, and current location when present".

---


### Dependencyoverview

#### Reads

- /v1/supplier/fleet/drivers
- /v1/supplier/fleet/vehicles
- /v1/supplier/fleet/capacity
- /v1/supplier/fleet/drivers/{id}

#### Writes

- /v1/supplier/fleet/drivers
- /v1/supplier/fleet/vehicles
- /v1/supplier/fleet/drivers/{driverId}/assign-vehicle
- /v1/supplier/fleet/vehicles/{vehicleId}
- /v1/vehicle/{vehicleId}/clear-returns

#### Localizednotes

- O'qish: "/v1/supplier/fleet/drivers", "/v1/supplier/fleet/vehicles", "/v1/supplier/fleet/capacity", "/v1/supplier/fleet/drivers/{id}".
- Yozish: "/v1/supplier/fleet/drivers", "/v1/supplier/fleet/vehicles", "/v1/supplier/fleet/drivers/{driverId}/assign-vehicle", "/v1/supplier/fleet/vehicles/{vehicleId}", "/v1/vehicle/{vehicleId}/clear-returns".
- Yangilash modeli: "initial fetch on mount followed by targeted reloads after create, assign, deactivate, and clear-return actions".

### Stateoverview

- Holat: "drivers-tab loading spinner".
- Holat: "drivers empty state".
- Holat: "vehicles empty state".
- Holat: "drivers table with assignment select".
- Holat: "vehicles table with action chips".
- Holat: "add-driver drawer open".
- Holat: "add-vehicle drawer open in class mode".
- Holat: "add-vehicle drawer open in dimension mode with computed VU".
- Holat: "PIN reveal modal".
- Holat: "driver detail drawer open".

### Figureoverview

- Figura: "fleet page on drivers tab with KPI row and assignment table".
- Figura: "fleet page on vehicles tab with capacity metrics and vehicle table".
- Figura: "add-driver drawer".
- Figura: "PIN reveal modal".
- Figura: "driver detail drawer".
- Figura: "add-vehicle drawer in dimensions mode".

---

**Dossierfile:** web-supplier-inventory.json

**Pageid:** web-supplier-inventory

**Route:** /supplier/inventory

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/inventory/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-inventory" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-inventory" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier stock-control page for quantity adjustments, low-stock visibility, and immutable audit-log inspection.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "headline: Inventory Management"; "subtitle: Stock levels, replenishment controls, and audit trail".
- "tab-switcher" zonasi "below header" hududida joylashgan. Tarkibi: "Stock Levels segmented chip"; "Audit Log segmented chip".
- "stock-card" zonasi "primary card when stock tab active" hududida joylashgan. Tarkibi: "SKU count label"; "Refresh button"; "inventory table with product, sku, stock, action columns"; "inline adjustment editor replacing action cell when row is in adjust mode"; "pagination controls".
- "audit-card" zonasi "primary card when audit tab active" hududida joylashgan. Tarkibi: "Last 100 adjustments label"; "audit table with product, prev, delta, new, reason, date columns"; "pagination controls".

### Controloverview

- "Stock Levels" tugmasi "tab-switcher" hududida joylashgan. Uslub: "segmented chip".
- "Audit Log" tugmasi "tab-switcher" hududida joylashgan. Uslub: "segmented chip".
- "Refresh" tugmasi "stock-card header-right" hududida joylashgan. Uslub: "outline".
- "Adjust" tugmasi "stock row action cell" hududida joylashgan. Uslub: "outline small".
- "Apply" tugmasi "inline adjustment editor" hududida joylashgan. Uslub: "primary small".
- "Cancel" tugmasi "inline adjustment editor" hududida joylashgan. Uslub: "text small".

### Iconoverview

- "inventory" ikonasi "stock empty state" zonasida ishlatiladi.
- "ledger" ikonasi "audit empty state" zonasida ishlatiladi.
- "reason chip" ikonasi "audit reason column" zonasida ishlatiladi.

### Flowoverview

**Flowid:** inventory-bootstrap

**Summary:** "inventory-bootstrap" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page reads token via useToken".
- 2-qadam: "Page requests /v1/supplier/inventory and /v1/supplier/inventory/audit".
- 3-qadam: "Stock tab renders by default".

---

**Flowid:** quantity-adjustment

**Summary:** "quantity-adjustment" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier clicks Adjust on a row".
- 2-qadam: "Action cell expands into delta input, reason select, Apply, and Cancel controls".
- 3-qadam: "Page patches /v1/supplier/inventory with adjustment payload".
- 4-qadam: "Page refreshes both stock and audit datasets".

---

**Flowid:** audit-review

**Summary:** "audit-review" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier switches to Audit Log tab".
- 2-qadam: "Page displays signed delta values, reason chip, and timestamped adjustments".

---


### Dependencyoverview

#### Reads

- /v1/supplier/inventory
- /v1/supplier/inventory/audit

#### Writes

- /v1/supplier/inventory

#### Localizednotes

- O'qish: "/v1/supplier/inventory", "/v1/supplier/inventory/audit".
- Yozish: "/v1/supplier/inventory".
- Yangilash modeli: "load on mount plus manual refresh button and automatic re-fetch after successful adjustments".

### Stateoverview

- Holat: "unauthorized supplier-required card".
- Holat: "stock-tab loading state".
- Holat: "stock empty state".
- Holat: "normal stock table".
- Holat: "row-level inline adjustment mode".
- Holat: "audit empty state".
- Holat: "audit table with positive and negative delta coloring".

### Figureoverview

- Figura: "inventory page on stock tab".
- Figura: "stock row in inline adjust mode".
- Figura: "inventory page on audit tab".
- Figura: "audit table close-up with reason chip and signed delta".

---

**Dossierfile:** web-supplier-login.json

**Pageid:** web-auth-login

**Route:** /auth/login

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Sourcefile:** apps/admin-portal/app/auth/login/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-auth-login" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-auth-login" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier portal sign-in page with credential form, stale-cookie clearing, inline error handling, and password visibility toggle.

### Layoutoverview

- "mobile-brand-strip" zonasi "top on mobile only" hududida joylashgan. Tarkibi: "brand icon tile"; "Pegasus Hub title"; "Supplier Operations Portal subtitle".
- "login-card" zonasi "central card" hududida joylashgan. Tarkibi: "Sign in headline"; "supporting subtitle"; "optional inline error alert"; "email field"; "password field with show-hide button"; "primary Sign In button"; "link to create account".
- "mobile-footer-copy" zonasi "bottom on mobile only" hududida joylashgan. Tarkibi: "Pegasus copyright text".

### Controloverview

- "password visibility toggle" tugmasi "inside password field trailing edge" hududida joylashgan. Uslub: "icon button".
- "Sign In" tugmasi "login-card footer" hududida joylashgan. Uslub: "full-width primary CTA".
- "Create account link" tugmasi "below primary CTA" hududida joylashgan. Uslub: "inline text link".

### Iconoverview

- "brand warehouse glyph" ikonasi "mobile-brand-strip" zonasida ishlatiladi.
- "error alert icon" ikonasi "inline alert row" zonasida ishlatiladi.
- "eye or eye-off glyph" ikonasi "password visibility toggle" zonasida ishlatiladi.
- "spinner" ikonasi "Sign In button loading state" zonasida ishlatiladi.

### Flowoverview

**Flowid:** credential-login

**Summary:** "credential-login" oqimi 5 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page clears stale auth cookies on mount".
- 2-qadam: "User enters email and password".
- 3-qadam: "User submits Sign In".
- 4-qadam: "Page posts to /v1/auth/admin/login".
- 5-qadam: "Successful response writes cookies and routes user to /".

---

**Flowid:** password-peek

**Summary:** "password-peek" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "User taps trailing password eye control".
- 2-qadam: "Password field switches between masked and plain text state".

---


### Stateoverview

- Holat: "idle form".
- Holat: "inline error state".
- Holat: "submitting state with spinner".

### Figureoverview

- Figura: "full login page".
- Figura: "login card close-up".
- Figura: "password field with visibility toggle".
- Figura: "error alert state".
- Figura: "submitting CTA state".

---

**Dossierfile:** web-supplier-manifests.json

**Pageid:** web-supplier-manifests

**Route:** /supplier/manifests

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented-as-redirect

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/manifests/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-manifests" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-manifests" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Legacy supplier manifests route that immediately redirects to the canonical supplier orders surface instead of rendering standalone UI.

### Layoutoverview

- "redirect-guard" zonasi "server-side route handler" hududida joylashgan. Tarkibi: "no persisted UI; redirect('/supplier/orders') executes during route resolution".

### Controloverview


### Iconoverview


### Flowoverview

**Flowid:** route-alias-redirect

**Summary:** "route-alias-redirect" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier navigates to /supplier/manifests".
- 2-qadam: "Next.js redirect executes immediately".
- 3-qadam: "Browser lands on /supplier/orders where the actual manifest and order workflow resides".

---


### Dependencyoverview

#### Reads


#### Writes


#### Localizednotes

- O'qish: yo'q.
- Yozish: yo'q.
- Yangilash modeli: "no local data fetch; route delegates to /supplier/orders".

### Stateoverview

- Holat: "immediate server redirect with no rendered intermediate page".

### Figureoverview

- Figura: "route-flow figure showing /supplier/manifests aliasing into /supplier/orders".

---

**Dossierfile:** web-supplier-onboarding.json

**Pageid:** web-supplier-onboarding

**Route:** /supplier/onboarding

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented-as-redirect

**Shell:** none

**Sourcefile:** apps/admin-portal/app/supplier/onboarding/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-onboarding" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-onboarding" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Deprecated supplier onboarding route retained as a transitional redirect because onboarding is now fully embedded into the registration wizard at /auth/register.

### Layoutoverview

- "redirect-indicator" zonasi "centered full-screen state" hududida joylashgan. Tarkibi: "spinner glyph"; "status text: Redirecting to dashboard…".

### Controloverview


### Iconoverview

- "spinner glyph" ikonasi "centered redirect-indicator" zonasida ishlatiladi.

### Flowoverview

**Flowid:** conditional-redirect

**Summary:** "conditional-redirect" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Client effect reads supplier token from cookie".
- 2-qadam: "If token is absent, router.replace('/auth/register') executes".
- 3-qadam: "If token is present, router.replace('/supplier/dashboard') executes".

---


### Dependencyoverview

#### Reads


#### Writes


#### Localizednotes

- O'qish: yo'q.
- Yozish: yo'q.
- Yangilash modeli: "no network fetch; client-side token check determines redirect target".

### Stateoverview

- Holat: "transient redirect indicator before navigation resolves".
- Holat: "redirect to registration wizard when unauthenticated".
- Holat: "redirect to supplier dashboard when authenticated".

### Figureoverview

- Figura: "full-screen redirect indicator with spinner and redirect text".
- Figura: "route-flow figure showing /supplier/onboarding branching to /auth/register or /supplier/dashboard".

---

**Dossierfile:** web-supplier-payment-config.json

**Pageid:** web-supplier-payment-config

**Route:** /supplier/payment-config

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/payment-config/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-payment-config" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-payment-config" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier gateway-credential administration page for Click, Payme, and Global Pay, supporting manual onboarding, activation-state review, update, and deactivation.

### Layoutoverview

- "header" zonasi "top constrained column" hududida joylashgan. Tarkibi: "headline: Payment Gateways"; "subtitle explaining Click, Payme, and Global Pay configuration".
- "toast-region" zonasi "below header when toast exists" hududida joylashgan. Ko'rinish qoidasi: "visible when toast state is non-null". Tarkibi: "success or error colored banner"; "leading status icon"; "toast message".
- "provider-stack" zonasi "vertical list of provider cards" hududida joylashgan. Tarkibi: "one card per gateway capability"; "icon tile"; "display name"; "active or not-configured chip"; "merchant or service preview text when configured"; "connect placeholder badge or button"; "manual-setup or update button"; "optional deactivate button".
- "expanded-manual-form" zonasi "inside provider card below divider" hududida joylashgan. Ko'rinish qoidasi: "visible when expandedGateway matches card gateway". Tarkibi: "shield-led security hint"; "merchant-id input"; "optional service-id input"; "secret-key password input"; "per-field helper copy"; "Cancel and Save or Update Configuration buttons".
- "empty-state" zonasi "center card" hududida joylashgan. Ko'rinish qoidasi: "visible when no capabilities and no configs are returned". Tarkibi: "payment icon"; "No payment gateways available headline"; "administrator-support body copy".

### Controloverview

- "Connect" tugmasi "provider-card action cluster" hududida joylashgan. Uslub: "primary small". Ko'rinish qoidasi: "provider supports redirect onboarding".
- "Connect coming soon badge" tugmasi "provider-card action cluster" hududida joylashgan. Uslub: "disabled status badge". Ko'rinish qoidasi: "manual-only provider".
- "Manual setup" tugmasi "provider-card action cluster" hududida joylashgan. Uslub: "outline or primary small". Ko'rinish qoidasi: "provider not configured".
- "Update" tugmasi "provider-card action cluster" hududida joylashgan. Uslub: "outline or primary small". Ko'rinish qoidasi: "provider configured".
- "Deactivate" tugmasi "provider-card action cluster" hududida joylashgan. Uslub: "danger-soft small". Ko'rinish qoidasi: "config active".
- "Cancel" tugmasi "expanded-manual-form footer-left" hududida joylashgan. Uslub: "outline".
- "Save Configuration" tugmasi "expanded-manual-form footer-right" hududida joylashgan. Uslub: "primary". Ko'rinish qoidasi: "new config".
- "Update Configuration" tugmasi "expanded-manual-form footer-right" hududida joylashgan. Uslub: "primary". Ko'rinish qoidasi: "editing existing config".

### Iconoverview

- "gateway svg badge" ikonasi "provider-card leading tile" zonasida ishlatiladi.
- "check-circle" ikonasi "active status chip and success toast" zonasida ishlatiladi.
- "x-circle" ikonasi "error toast" zonasida ishlatiladi.
- "clock" ikonasi "not-configured chip" zonasida ishlatiladi.
- "shield" ikonasi "manual-form helper lines" zonasida ishlatiladi.
- "key-round" ikonasi "manual-setup toggle button" zonasida ishlatiladi.
- "chevron-down or chevron-up" ikonasi "manual-setup toggle button trailing edge" zonasida ishlatiladi.
- "link2" ikonasi "connect action" zonasida ishlatiladi.

### Flowoverview

**Flowid:** gateway-bootstrap

**Summary:** "gateway-bootstrap" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page requests /v1/supplier/payment-config".
- 2-qadam: "Configured gateways and provider capabilities render as stacked cards".
- 3-qadam: "Merchant and service previews appear without secret prefill".

---

**Flowid:** manual-credential-save

**Summary:** "manual-credential-save" oqimi 5 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier expands Manual setup or Update on a gateway card".
- 2-qadam: "Page seeds merchant and service values from existing config when present".
- 3-qadam: "Supplier enters required fields".
- 4-qadam: "Page posts to /v1/supplier/payment-config".
- 5-qadam: "Success toast appears and cards reload".

---

**Flowid:** gateway-deactivation

**Summary:** "gateway-deactivation" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier clicks Deactivate on an active gateway card".
- 2-qadam: "Page deletes through /v1/supplier/payment-config with config_id payload".
- 3-qadam: "Success toast appears and list refreshes".

---


### Dependencyoverview

#### Reads

- /v1/supplier/payment-config

#### Writes

- /v1/supplier/payment-config

#### Localizednotes

- O'qish: "/v1/supplier/payment-config".
- Yozish: "/v1/supplier/payment-config".
- Yangilash modeli: "initial fetch on mount plus reload after save and deactivate operations".

### Stateoverview

- Holat: "loading spinner row".
- Holat: "provider stack with all cards collapsed".
- Holat: "expanded manual form for Click".
- Holat: "expanded manual form for Payme".
- Holat: "expanded manual form for Global Pay with service-id helper text".
- Holat: "success toast state".
- Holat: "error toast state".
- Holat: "no-capabilities empty state".

### Figureoverview

- Figura: "full payment gateway stack".
- Figura: "configured gateway card close-up".
- Figura: "expanded Global Pay manual form".
- Figura: "success toast over gateway stack".

---

**Dossierfile:** web-supplier-pricing.json

**Pageid:** web-supplier-pricing

**Route:** /supplier/pricing

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/pricing/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-pricing" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-pricing" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier pricing-engine page for composing volume-discount rules and auditing the currently active pricing rule ledger.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "headline: Pricing Engine"; "subtitle: B2B Volume Discount Rules — Upsert and Manage".
- "primary-grid" zonasi "two-column workspace with wider table column" hududida joylashgan. Tarkibi: "left form panel for new pricing rule composition"; "right rules ledger card with table or empty state".
- "rule-form-panel" zonasi "left column" hududida joylashgan. Tarkibi: "SKU selector or fallback text input"; "Min Pallets numeric input"; "Discount percent numeric input with helper copy"; "Target Retailer Tier chip row"; "Valid Until datetime input"; "Tier ID input"; "Lock Pricing Rule CTA".
- "rules-ledger-card" zonasi "right column" hududida joylashgan. Tarkibi: "rules count header"; "loading message or pricing empty state"; "rules table with SKU, pallet threshold, discount chip, retailer tier, expiry, status, and actions".

### Controloverview

- "Target Retailer Tier chip" tugmasi "rule-form-panel target tier row" hududida joylashgan. Uslub: "chip toggle".
- "Lock Pricing Rule" tugmasi "rule-form-panel footer" hududida joylashgan. Uslub: "full-width primary button".
- "Deactivate" tugmasi "rules-ledger actions column" hududida joylashgan. Uslub: "small danger-tinted button". Ko'rinish qoidasi: "rule is active".

### Iconoverview

- "pricing" ikonasi "rules empty state" zonasida ishlatiladi.

### Flowoverview

**Flowid:** pricing-bootstrap

**Summary:** "pricing-bootstrap" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page obtains supplier token".
- 2-qadam: "Page fetches /v1/supplier/pricing/rules and /v1/supplier/products in parallel".
- 3-qadam: "SKU selector options and rules ledger render from those responses".

---

**Flowid:** rule-composition

**Summary:** "rule-composition" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier chooses SKU, pallet threshold, discount percent, retailer tier, expiry, and optional tier ID".
- 2-qadam: "Submit generates UUID when tier_id is blank".
- 3-qadam: "Page posts the assembled rule to /v1/supplier/pricing/rules".
- 4-qadam: "Success resets the form and refreshes the rules ledger".

---

**Flowid:** rule-deactivation

**Summary:** "rule-deactivation" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier presses Deactivate on an active rule row".
- 2-qadam: "Page deletes /v1/supplier/pricing/rules/{tier_id}".
- 3-qadam: "Row status transitions from active to inactive after refresh".

---


### Dependencyoverview

#### Reads

- /v1/supplier/pricing/rules
- /v1/supplier/products

#### Writes

- /v1/supplier/pricing/rules
- /v1/supplier/pricing/rules/{tier_id}

#### Localizednotes

- O'qish: "/v1/supplier/pricing/rules", "/v1/supplier/products".
- Yozish: "/v1/supplier/pricing/rules", "/v1/supplier/pricing/rules/{tier_id}".
- Yangilash modeli: "load on mount and reload after successful create or deactivate actions".

### Stateoverview

- Holat: "rules loading message state".
- Holat: "empty rules ledger state".
- Holat: "form with product select populated".
- Holat: "form submit in locking state".
- Holat: "rules table with active and inactive badges".
- Holat: "row-level deactivation pending state".

### Figureoverview

- Figura: "full pricing-engine page with form panel and rules table".
- Figura: "rule composition form close-up showing tier chips and lock CTA".
- Figura: "rules ledger row showing discount chip, expiry, and deactivate action".

---

**Dossierfile:** web-supplier-product-detail.json

**Pageid:** web-supplier-product-detail

**Route:** /supplier/products/[sku_id]

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/products/[sku_id]/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-product-detail" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-product-detail" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier per-SKU detail workspace for inspecting metadata, editing commercial fields, and adjusting logistics constraints and activation status.

### Layoutoverview

- "back-nav" zonasi "top-left above header" hududida joylashgan. Tarkibi: "Back to Products text button with arrow icon".
- "header-row" zonasi "top full-width below back-nav" hududida joylashgan. Tarkibi: "thumbnail block"; "title, status pill, category label, and SKU metadata"; "action cluster with activate or deactivate button and edit-mode controls".
- "save-message" zonasi "below header when saveMsg exists" hududida joylashgan. Ko'rinish qoidasi: "visible after save or no-op response". Tarkibi: "success or error message banner".
- "detail-grid" zonasi "two-column main region" hududida joylashgan. Tarkibi: "Product Details card with name, description, image URL, and base price"; "Logistics and Ordering card with MOQ, step size, block settings, volumetric unit, dimensions, and created date".

### Controloverview

- "Back to Products" tugmasi "back-nav" hududida joylashgan. Uslub: "text button with leading icon".
- "Deactivate" tugmasi "header-row action cluster" hududida joylashgan. Uslub: "outline button". Ko'rinish qoidasi: "product is active".
- "Activate" tugmasi "header-row action cluster" hududida joylashgan. Uslub: "outline button". Ko'rinish qoidasi: "product is inactive".
- "Edit Product" tugmasi "header-row action cluster" hududida joylashgan. Uslub: "primary button". Ko'rinish qoidasi: "editing is false".
- "Cancel" tugmasi "header-row action cluster" hududida joylashgan. Uslub: "outline button". Ko'rinish qoidasi: "editing is true".
- "Save Changes" tugmasi "header-row action cluster" hududida joylashgan. Uslub: "primary button". Ko'rinish qoidasi: "editing is true".

### Iconoverview

- "arrow_back" ikonasi "Back to Products control" zonasida ishlatiladi.
- "image" ikonasi "thumbnail placeholder when product image is absent" zonasida ishlatiladi.

### Flowoverview

**Flowid:** detail-bootstrap

**Summary:** "detail-bootstrap" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page reads sku_id from route params and supplier token from auth context".
- 2-qadam: "Page requests /v1/supplier/products/{sku_id}".
- 3-qadam: "Fetched data populates the read-only detail view and edit draft state".

---

**Flowid:** edit-session

**Summary:** "edit-session" oqimi 5 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier presses Edit Product".
- 2-qadam: "All editable fields switch to inputs or textarea controls".
- 3-qadam: "Supplier modifies commercial or logistics values".
- 4-qadam: "Page computes a diff against original product state".
- 5-qadam: "PUT request to /v1/supplier/products/{sku_id} persists changed fields only".

---

**Flowid:** activation-toggle

**Summary:** "activation-toggle" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier presses Activate or Deactivate in the header action cluster".
- 2-qadam: "Page submits an is_active update to /v1/supplier/products/{sku_id}".
- 3-qadam: "Detail view reloads and the status pill flips".

---

**Flowid:** save-feedback

**Summary:** "save-feedback" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "After save, page emits a success or error banner beneath the header".
- 2-qadam: "No-change saves collapse edit mode with informational confirmation".

---


### Dependencyoverview

#### Reads

- /v1/supplier/products/{sku_id}

#### Writes

- /v1/supplier/products/{sku_id}

#### Localizednotes

- O'qish: "/v1/supplier/products/{sku_id}".
- Yozish: "/v1/supplier/products/{sku_id}".
- Yangilash modeli: "load on mount and reload after successful save or activation changes".

### Stateoverview

- Holat: "page loading spinner".
- Holat: "error or not-found card with back button".
- Holat: "default read-only detail mode".
- Holat: "edit mode with form controls in both cards".
- Holat: "saving state with disabled buttons".
- Holat: "success message banner".
- Holat: "error message banner".

### Figureoverview

- Figura: "full product-detail page in read-only mode".
- Figura: "product-detail page in edit mode with both cards active".
- Figura: "header close-up showing thumbnail, status pill, SKU, and action cluster".
- Figura: "logistics card close-up showing MOQ, step size, units-per-block, and dimensions".

---

**Dossierfile:** web-supplier-products.json

**Pageid:** web-supplier-products

**Route:** /supplier/products

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/products/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-products" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-products" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier product-portfolio page for SKU search, category filtering, activation toggling, and entry into per-product detail editing.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "headline: My Products"; "subtitle with registered SKU count"; "Add Product CTA linking to supplier catalog".
- "kpi-strip" zonasi "below header" hududida joylashgan. Tarkibi: "Total SKUs card"; "Active card"; "Inactive card"; "Catalog Value card".
- "control-row" zonasi "below KPI strip" hududida joylashgan. Tarkibi: "left search input with embedded search icon"; "right Refresh button".
- "category-chip-row" zonasi "below control row when categoryOptions exist" hududida joylashgan. Tarkibi: "All chip with total count"; "per-category chips with counts".
- "product-grid" zonasi "main content grid" hududida joylashgan. Tarkibi: "responsive product cards with image region, status badge, category pill, title, description, price block, activation icon button, and SKU footer"; "empty state when no products match filters".

### Controloverview

- "Add Product" tugmasi "header top-right" hududida joylashgan. Uslub: "accent filled link button".
- "Refresh" tugmasi "control-row right" hududida joylashgan. Uslub: "outline button with refresh icon".
- "All" tugmasi "category-chip-row" hududida joylashgan. Uslub: "filter chip".
- "Category chip" tugmasi "category-chip-row" hududida joylashgan. Uslub: "filter chip".
- "Deactivate" tugmasi "product card bottom-right icon button" hududida joylashgan. Uslub: "round danger-tinted icon button". Ko'rinish qoidasi: "product is active".
- "Activate" tugmasi "product card bottom-right icon button" hududida joylashgan. Uslub: "round success-tinted icon button". Ko'rinish qoidasi: "product is inactive".

### Iconoverview

- "add" ikonasi "Add Product CTA leading icon" zonasida ishlatiladi.
- "search" ikonasi "search field left inset" zonasida ishlatiladi.
- "refresh" ikonasi "Refresh button leading icon" zonasida ishlatiladi.
- "image placeholder glyph" ikonasi "product card media region when image_url missing" zonasida ishlatiladi.
- "visibility_off" ikonasi "active product toggle button" zonasida ishlatiladi.
- "visibility" ikonasi "inactive product toggle button" zonasida ishlatiladi.
- "catalog" ikonasi "empty state" zonasida ishlatiladi.

### Flowoverview

**Flowid:** products-bootstrap

**Summary:** "products-bootstrap" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page reads supplier token".
- 2-qadam: "Page fetches /v1/supplier/products and /v1/supplier/profile in parallel".
- 3-qadam: "When operating categories exist, page maps them against /v1/catalog/platform-categories".
- 4-qadam: "KPI strip and filtered grid derive from the resulting product dataset".

---

**Flowid:** search-and-filter

**Summary:** "search-and-filter" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier types in the search field or taps a category chip".
- 2-qadam: "Grid filters client-side by category, SKU, product name, and description".
- 3-qadam: "KPI totals recompute from the filtered list".

---

**Flowid:** product-detail-navigation

**Summary:** "product-detail-navigation" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier clicks a product card".
- 2-qadam: "Router pushes to /supplier/products/{sku_id}".
- 3-qadam: "Per-product detail workspace opens".

---

**Flowid:** activation-toggle

**Summary:** "activation-toggle" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier presses the card-level activation icon button".
- 2-qadam: "Page puts new is_active value to /v1/supplier/products/{sku_id}".
- 3-qadam: "Products dataset reloads and card opacity/status badge update".

---


### Dependencyoverview

#### Reads

- /v1/supplier/products
- /v1/supplier/profile
- /v1/catalog/platform-categories

#### Writes

- /v1/supplier/products/{sku_id}

#### Localizednotes

- O'qish: "/v1/supplier/products", "/v1/supplier/profile", "/v1/catalog/platform-categories".
- Yozish: "/v1/supplier/products/{sku_id}".
- Yangilash modeli: "load on mount, manual refresh button, and automatic reload after activation changes".

### Stateoverview

- Holat: "page loading spinner".
- Holat: "error card state".
- Holat: "grid empty state with no products yet".
- Holat: "grid empty state with no search matches".
- Holat: "filtered product grid with mixed active and inactive cards".
- Holat: "row-level activation toggle pending spinner".

### Figureoverview

- Figura: "full products page with header, KPI strip, filters, and grid".
- Figura: "single product card showing status badge and activation icon button".
- Figura: "empty-state view with search field and refresh button retained".

---

**Dossierfile:** web-supplier-profile.json

**Pageid:** web-supplier-profile

**Route:** /supplier/profile

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/profile/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-profile" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-profile" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier profile and operations-account page for managing identity, warehouse, banking, category, and shift-status data in a sectioned desktop form layout.

### Layoutoverview

- "error-banner" zonasi "top of page when error exists" hududida joylashgan. Ko'rinish qoidasi: "visible when an error is present". Tarkibi: "error icon"; "error text".
- "hero-header" zonasi "top full-width card" hududida joylashgan. Tarkibi: "circular warehouse avatar"; "supplier name, configuration status, category-email-phone summary"; "edit action cluster".
- "company-details-section" zonasi "below hero header" hududida joylashgan. Tarkibi: "section title with accent border"; "two-column card containing company and billing fields".
- "warehouse-section" zonasi "below company details" hududida joylashgan. Tarkibi: "warehouse address field"; "latitude and longitude read-only fields".
- "banking-section" zonasi "below warehouse section" hududida joylashgan. Tarkibi: "bank name field"; "account number field"; "card number field"; "payment gateway field".
- "operating-categories-section" zonasi "below banking when categories exist" hududida joylashgan. Ko'rinish qoidasi: "visible when operating_categories is non-empty". Tarkibi: "section title"; "category chips".
- "shift-status-section" zonasi "bottom card" hududida joylashgan. Tarkibi: "status dot"; "shift-state text"; "manual override label when manual_off_shift is true".

### Controloverview

- "Retry" tugmasi "error fallback screen" hududida joylashgan. Uslub: "secondary button". Ko'rinish qoidasi: "profile failed to load and profile is absent".
- "Edit Profile" tugmasi "hero-header top-right" hududida joylashgan. Uslub: "primary button with leading edit icon". Ko'rinish qoidasi: "editing is false".
- "Cancel" tugmasi "hero-header top-right" hududida joylashgan. Uslub: "outline button". Ko'rinish qoidasi: "editing is true".
- "Save" tugmasi "hero-header top-right" hududida joylashgan. Uslub: "primary button". Ko'rinish qoidasi: "editing is true".

### Iconoverview

- "error" ikonasi "error banner and error fallback" zonasida ishlatiladi.
- "warehouse" ikonasi "hero avatar" zonasida ishlatiladi.
- "verified" ikonasi "configuration status line in hero header" zonasida ishlatiladi.
- "edit" ikonasi "Edit Profile button" zonasida ishlatiladi.

### Flowoverview

**Flowid:** profile-bootstrap

**Summary:** "profile-bootstrap" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page requests /v1/supplier/profile through apiFetch".
- 2-qadam: "Skeleton placeholders render until the profile payload resolves".
- 3-qadam: "Resolved data populates hero, section cards, and status chips".

---

**Flowid:** edit-initialization

**Summary:** "edit-initialization" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier presses Edit Profile".
- 2-qadam: "Page copies editable fields into a draft object".
- 3-qadam: "Read-only display cells switch to outline inputs".

---

**Flowid:** profile-save

**Summary:** "profile-save" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page diffs the draft against the fetched profile".
- 2-qadam: "Changed fields only are submitted in a PUT request to /v1/supplier/profile".
- 3-qadam: "Profile reloads after save and the page returns to read-only mode".

---

**Flowid:** edit-cancel

**Summary:** "edit-cancel" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier presses Cancel during editing".
- 2-qadam: "Draft state clears and read-only cards restore without network writes".

---


### Dependencyoverview

#### Reads

- /v1/supplier/profile

#### Writes

- /v1/supplier/profile

#### Localizednotes

- O'qish: "/v1/supplier/profile".
- Yozish: "/v1/supplier/profile".
- Yangilash modeli: "load on mount, retry on failure, and reload after successful profile updates".

### Stateoverview

- Holat: "skeleton loading state".
- Holat: "hard error fallback with retry button".
- Holat: "inline error banner above populated profile".
- Holat: "default read-only profile sections".
- Holat: "edit mode with outlined field inputs".
- Holat: "saving state with disabled save button".
- Holat: "operating categories section visible".
- Holat: "manual off-shift marker visible".

### Figureoverview

- Figura: "full supplier profile page with hero header and stacked sections".
- Figura: "hero header close-up showing avatar, configuration badge, and edit controls".
- Figura: "company-details section in edit mode with input fields".
- Figura: "shift-status card with active or off-shift indicator".

---

**Dossierfile:** web-supplier-register.json

**Pageid:** web-auth-register

**Route:** /auth/register

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Sourcefile:** apps/admin-portal/app/auth/register/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-auth-register" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-auth-register" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Four-step supplier registration wizard combining account identity, warehouse location, business and fleet profile, category selection, and payment gateway preference.

### Layoutoverview

- "mobile-brand-strip" zonasi "top on mobile only" hududida joylashgan. Tarkibi: "brand icon tile"; "Pegasus Hub title"; "Supplier Registration subtitle".
- "step-indicator" zonasi "above main card" hududida joylashgan. Tarkibi: "Account step node"; "Location step node"; "Business step node"; "Payments step node".
- "wizard-card" zonasi "central card" hududida joylashgan. Tarkibi: "step headline and subtitle"; "optional inline error alert"; "step-specific form body"; "Back and Continue/Create Account buttons"; "Sign in link on step 1".

### Controloverview

- "Locate" tugmasi "step 2 location field trailing action" hududida joylashgan. Uslub: "secondary inline CTA".
- "category chips" tugmasi "step 3 category grid" hududida joylashgan. Uslub: "multi-select chip".
- "cold-chain toggle" tugmasi "step 3 fleet profile card" hududida joylashgan. Uslub: "switch-like toggle".
- "payment gateway rows" tugmasi "step 4 payment list" hududida joylashgan. Uslub: "full-width selectable card rows".
- "Back" tugmasi "wizard footer left" hududida joylashgan. Uslub: "outline CTA when step > 0".
- "Continue to next step" tugmasi "wizard footer primary slot" hududida joylashgan. Uslub: "full-width primary CTA on non-final steps".
- "Create Supplier Account" tugmasi "wizard footer primary slot" hududida joylashgan. Uslub: "full-width primary CTA on final step".
- "Sign in link" tugmasi "step 1 footer text" hududida joylashgan. Uslub: "inline text link".

### Iconoverview

- "brand warehouse glyph" ikonasi "mobile-brand-strip" zonasida ishlatiladi.
- "step icons and checkmark states" ikonasi "step indicator nodes" zonasida ishlatiladi.
- "Locate spinner or location glyph" ikonasi "step 2 Locate button" zonasida ishlatiladi.
- "category checkmark" ikonasi "selected category chips" zonasida ishlatiladi.
- "gateway icons" ikonasi "step 4 payment gateway rows" zonasida ishlatiladi.
- "spinner" ikonasi "Create Supplier Account submitting state" zonasida ishlatiladi.

### Flowoverview

**Flowid:** wizard-progression

**Summary:** "wizard-progression" oqimi 6 ta qadamdan iborat.

#### Steps

- 1-qadam: "User completes account step".
- 2-qadam: "User advances to location step".
- 3-qadam: "User advances to business step with category selection".
- 4-qadam: "User advances to payment step".
- 5-qadam: "User submits registration".
- 6-qadam: "On success cookies are written and user is routed to /supplier/dashboard".

---

**Flowid:** geolocation-capture

**Summary:** "geolocation-capture" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "User presses Locate on step 2".
- 2-qadam: "Browser geolocation retrieves coordinates".
- 3-qadam: "Page reverse-geocodes via Nominatim when possible".
- 4-qadam: "Address and lat/lng fields are populated".

---

**Flowid:** category-and-gateway-selection

**Summary:** "category-and-gateway-selection" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "User filters or browses category chips".
- 2-qadam: "User selects one or more categories".
- 3-qadam: "User selects one payment gateway card row as active".

---


### Stateoverview

- Holat: "step 1 account fields".
- Holat: "step 2 location fields".
- Holat: "step 3 business and category selection".
- Holat: "step 4 payment gateway selection".
- Holat: "inline validation error state".
- Holat: "Create Account submitting state".

### Figureoverview

- Figura: "full registration wizard step 1".
- Figura: "step indicator close-up".
- Figura: "location step with Locate action".
- Figura: "business step category grid".
- Figura: "payment step gateway rows".
- Figura: "final submitting state".

---

**Dossierfile:** web-supplier-returns.json

**Pageid:** web-supplier-returns

**Route:** /supplier/returns

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/returns/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-returns" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-returns" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier dispute-resolution page for reviewing rejected or damaged line items and resolving them as write-offs or returns to stock.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "headline: Dispute and Returns"; "subtitle describing write-off versus return-to-stock resolution intent".
- "summary-strip" zonasi "below header" hududida joylashgan. Tarkibi: "Open Returns metric card"; "Total Damage Value metric card".
- "returns-card-header" zonasi "top of main card" hududida joylashgan. Tarkibi: "Damaged and Rejected Items label"; "Refresh button".
- "returns-ledger" zonasi "main card body" hududida joylashgan. Tarkibi: "loading message or empty state"; "paginated return-item rows with retailer, quantity, value, order reference, and action cluster".
- "inline-resolution-cluster" zonasi "row action area when resolvingId matches row" hududida joylashgan. Ko'rinish qoidasi: "visible for the selected line item". Tarkibi: "resolution select"; "notes input"; "Resolve button"; "dismiss x control".

### Controloverview

- "Refresh" tugmasi "returns-card-header right" hududida joylashgan. Uslub: "outline button".
- "Resolve" tugmasi "row action column" hududida joylashgan. Uslub: "outline small button".
- "Resolve" tugmasi "inline-resolution-cluster" hududida joylashgan. Uslub: "small primary button". Ko'rinish qoidasi: "resolution editor is open".
- "x" tugmasi "inline-resolution-cluster trailing edge" hududida joylashgan. Uslub: "small text dismiss control". Ko'rinish qoidasi: "resolution editor is open".

### Iconoverview

- "returns" ikonasi "empty state" zonasida ishlatiladi.

### Flowoverview

**Flowid:** returns-bootstrap

**Summary:** "returns-bootstrap" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page reads supplier token".
- 2-qadam: "Page requests /v1/supplier/returns".
- 3-qadam: "Summary cards derive from returned line items".

---

**Flowid:** resolution-open

**Summary:** "resolution-open" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier presses Resolve on a row".
- 2-qadam: "Action cell expands into resolution select, notes field, confirm control, and dismiss control".

---

**Flowid:** resolution-submit

**Summary:** "resolution-submit" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier selects WRITE_OFF or RETURN_TO_STOCK and optionally enters notes".
- 2-qadam: "Page posts to /v1/supplier/returns/resolve with line_item_id, resolution, and notes".
- 3-qadam: "Successful resolution clears the inline editor and reloads the ledger".

---

**Flowid:** pagination-review

**Summary:** "pagination-review" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier moves through paginated rows using shared pagination controls".
- 2-qadam: "Visible rows update while metric cards remain global".

---


### Dependencyoverview

#### Reads

- /v1/supplier/returns

#### Writes

- /v1/supplier/returns/resolve

#### Localizednotes

- O'qish: "/v1/supplier/returns".
- Yozish: "/v1/supplier/returns/resolve".
- Yangilash modeli: "load on mount, manual refresh button, and automatic reload after resolution".

### Stateoverview

- Holat: "unauthorized supplier-required card".
- Holat: "returns loading state".
- Holat: "returns empty state".
- Holat: "returns ledger list state".
- Holat: "inline resolution editor open".
- Holat: "row-level resolution submit pending state".

### Figureoverview

- Figura: "full returns page with summary cards and paginated ledger".
- Figura: "return row in inline resolution mode with dropdown and notes field".
- Figura: "empty-state view with refresh control".

---

**Dossierfile:** web-supplier-settings.json

**Pageid:** web-supplier-settings

**Route:** /supplier/settings

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/settings/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-settings" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-settings" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier operational-settings page for controlling manual off-shift status and day-by-day business-hour windows backed by the shared supplier-shift context.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "headline: Settings"; "subtitle describing business hours and shift availability".
- "shift-status-card" zonasi "first card below header" hududida joylashgan. Tarkibi: "section title and explanatory copy"; "manual off-shift toggle pill"; "effective OPEN or CLOSED status label".
- "business-hours-card" zonasi "second card below shift-status" hududida joylashgan. Tarkibi: "section title and scheduling guidance"; "seven day rows each with enable checkbox, day label, and open-close time controls or Closed text".
- "save-row" zonasi "bottom action band" hududida joylashgan. Tarkibi: "Save Changes button"; "saved-success or failed-save status text".

### Controloverview

- "ON SHIFT / OFF SHIFT toggle" tugmasi "shift-status-card" hududida joylashgan. Uslub: "pill toggle button".
- "Day enabled checkbox" tugmasi "business-hours-card day row" hududida joylashgan. Uslub: "checkbox control".
- "Save Changes" tugmasi "save-row left" hududida joylashgan. Uslub: "primary button".

### Iconoverview

- "spinner ring" ikonasi "loading state" zonasida ishlatiladi.
- "status dot" ikonasi "manual off-shift toggle pill" zonasida ishlatiladi.

### Flowoverview

**Flowid:** settings-bootstrap

**Summary:** "settings-bootstrap" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page reads shared supplier shift context from useSupplierShift".
- 2-qadam: "Context bootstraps from /v1/supplier/profile and exposes manual_off_shift, is_active, and operating_schedule".
- 3-qadam: "When the hook finishes loading, local form state mirrors the shared shift state".

---

**Flowid:** schedule-editing

**Summary:** "schedule-editing" oqimi 3 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier toggles day checkboxes to enable or disable days".
- 2-qadam: "Enabled days expose open and close time inputs".
- 3-qadam: "Changing a time updates local schedule state only until save".

---

**Flowid:** shift-save

**Summary:** "shift-save" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier presses Save Changes".
- 2-qadam: "Page assembles final enabled-day schedule and manual off-shift value".
- 3-qadam: "Shared hook patches /v1/supplier/shift with manual_off_shift and operating_schedule".
- 4-qadam: "Save status indicator reports success or failure".

---


### Dependencyoverview

#### Reads

- /v1/supplier/profile

#### Writes

- /v1/supplier/shift

#### Localizednotes

- O'qish: "/v1/supplier/profile".
- Yozish: "/v1/supplier/shift".
- Yangilash modeli: "shared-context bootstrap on mount plus explicit save action for schedule changes".

### Stateoverview

- Holat: "settings loading spinner state".
- Holat: "default shift and schedule form state".
- Holat: "days enabled with time pickers".
- Holat: "days disabled showing Closed text".
- Holat: "save in progress state".
- Holat: "saved successfully message".
- Holat: "failed save message".

### Figureoverview

- Figura: "full settings page with shift card, hours card, and save row".
- Figura: "shift-status card close-up with manual off-shift toggle and effective status label".
- Figura: "business-hours rows showing enabled and disabled day variants".

---

**Dossierfile:** web-supplier-staff.json

**Pageid:** web-supplier-staff

**Route:** /supplier/staff

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Shell:** admin-shell

**Sourcefile:** apps/admin-portal/app/supplier/staff/page.tsx

**Entrytype:** page

**Localizedsummary:** "web-supplier-staff" yuzasi uchun ta'minotchi roli va web platformasidagi lokalizatsiya qilingan ko'rinish.

## Localized

**Purpose:** "web-supplier-staff" yuzasi web platformasida ta'minotchi roli uchun sahifa sifatida hujjatlashtirilgan; asl inglizcha maqsad tavsifi purposeSourceAnchor maydonida saqlangan.

**Purposesourceanchor:** Supplier warehouse-staff management page for provisioning payloader accounts, listing worker credentials, and revealing one-time login PINs.

### Layoutoverview

- "header" zonasi "top full-width" hududida joylashgan. Tarkibi: "back link to supplier dashboard"; "headline: Warehouse Staff"; "subtitle describing payloader provisioning"; "Provision Worker CTA".
- "kpi-row" zonasi "below header" hududida joylashgan. Tarkibi: "Total Workers card"; "Active card"; "Inactive card".
- "worker-ledger" zonasi "main card region" hududida joylashgan. Tarkibi: "loading spinner or empty message"; "worker table with name, phone, worker ID, provision date, and status chip"; "pagination controls".
- "provision-drawer" zonasi "slide-out drawer" hududida joylashgan. Ko'rinish qoidasi: "visible when showAdd is true". Tarkibi: "name input"; "phone input"; "error text when form invalid or request fails"; "Provision Worker and Generate PIN CTA".
- "pin-reveal-modal" zonasi "center overlay modal" hududida joylashgan. Ko'rinish qoidasi: "visible when createdPin exists". Tarkibi: "success glyph"; "worker name and phone text"; "dashed PIN reveal panel"; "warning helper banner"; "Done button".

### Controloverview

- "Supplier Dashboard" tugmasi "header top-left" hududida joylashgan. Uslub: "text link".
- "+ Provision Worker" tugmasi "header top-right" hududida joylashgan. Uslub: "primary button".
- "Provision Worker and Generate PIN" tugmasi "provision-drawer footer" hududida joylashgan. Uslub: "full-width primary button".
- "Done" tugmasi "pin-reveal-modal footer" hududida joylashgan. Uslub: "full-width primary button".

### Iconoverview

- "warning" ikonasi "PIN reveal warning banner" zonasida ishlatiladi.
- "success checkmark glyph" ikonasi "PIN reveal modal hero circle" zonasida ishlatiladi.

### Flowoverview

**Flowid:** staff-bootstrap

**Summary:** "staff-bootstrap" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Page fetches /v1/supplier/staff/payloader on mount".
- 2-qadam: "Worker rows populate KPI counts and the paginated table".

---

**Flowid:** worker-provisioning

**Summary:** "worker-provisioning" oqimi 4 ta qadamdan iborat.

#### Steps

- 1-qadam: "Supplier opens the provision drawer".
- 2-qadam: "Supplier enters worker name and phone".
- 3-qadam: "Page posts to /v1/supplier/staff/payloader".
- 4-qadam: "On success, drawer closes, worker table refreshes, and the one-time PIN reveal modal opens".

---

**Flowid:** pin-disclosure

**Summary:** "pin-disclosure" oqimi 2 ta qadamdan iborat.

#### Steps

- 1-qadam: "Modal displays generated login PIN exactly once".
- 2-qadam: "Supplier acknowledges via Done and the PIN overlay is dismissed".

---


### Dependencyoverview

#### Reads

- /v1/supplier/staff/payloader

#### Writes

- /v1/supplier/staff/payloader

#### Localizednotes

- O'qish: "/v1/supplier/staff/payloader".
- Yozish: "/v1/supplier/staff/payloader".
- Yangilash modeli: "load on mount and reload after successful worker provisioning".

### Stateoverview

- Holat: "table loading spinner state".
- Holat: "empty worker roster state".
- Holat: "worker table with pagination".
- Holat: "provision drawer open".
- Holat: "provision drawer validation error".
- Holat: "PIN reveal modal visible".

### Figureoverview

- Figura: "full staff page with KPI row and worker table".
- Figura: "provision drawer with name and phone fields".
- Figura: "PIN reveal modal with dashed PIN panel and warning banner".

---


