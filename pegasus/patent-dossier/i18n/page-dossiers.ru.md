**Generatedat:** 2026-04-06T19:27:38.262Z

**Language:** ru

**Label:** Русский

**Sourcefolder:** patent-dossier/page-dossiers

**Localizationmode:** overlay-with-source-anchors

# Notes

- This file is a localized overlay for detailed page dossiers.
- Source English JSON remains the canonical evidence record; localized sentences preserve direct source anchors where exact UI labels should remain unchanged.
- Routes, endpoints, file paths, page IDs, and icon identifiers are intentionally preserved as technical anchors.

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

**Localizedsummary:** Локализованный обзор поверхности "android-driver-cash-collection" для роли водитель на платформе android.

## Localized

**Purpose:** Поверхность "android-driver-cash-collection" представляет страница для роли водитель на платформе android; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Android driver cash-handling screen requiring explicit confirmation before delivery completion and guarding against accidental back navigation.

### Layoutoverview

- Зона "center-stack" расположена в области "center vertical stack". Содержимое: "Payments icon"; "COLLECT CASH heading"; "amount text"; "instructional helper text".
- Зона "error-row" расположена в области "below helper text when present". Правило видимости: "visible when state.error is non-null". Содержимое: "centered error text".
- Зона "completion-cta" расположена в области "bottom of center stack". Содержимое: "Cash Collected — Complete button".
- Зона "exit-confirm-dialog" расположена в области "modal overlay". Правило видимости: "visible when showExitConfirm is true". Содержимое: "Leave cash collection title"; "warning text"; "Stay button"; "Leave button".

### Controloverview

- Кнопка "Cash Collected — Complete" расположена в "completion-cta". Стиль: "full-width primary".
- Кнопка "Stay" расположена в "exit-confirm-dialog confirm button slot". Стиль: "text button".
- Кнопка "Leave" расположена в "exit-confirm-dialog dismiss button slot". Стиль: "text button".

### Iconoverview

- Иконка "Payments" используется в зоне "center-stack top".
- Иконка "CircularProgressIndicator" используется в зоне "completion CTA while isCompleting".

### Flowoverview

**Flowid:** cash-completion

**Summary:** Поток "cash-completion" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Driver reviews amount to collect".
- Шаг 2: "Driver taps Cash Collected — Complete".
- Шаг 3: "ViewModel calls collectCash".
- Шаг 4: "Route exits through onComplete when state.completed is true".

---

**Flowid:** guarded-back-navigation

**Summary:** Поток "guarded-back-navigation" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Driver presses back before completion".
- Шаг 2: "BackHandler opens confirmation dialog".
- Шаг 3: "Driver chooses Stay or Leave".
- Шаг 4: "Submission state suppresses back navigation entirely".

---


### Stateoverview

- Состояние: "cash collection idle state".
- Состояние: "back-navigation confirmation dialog".
- Состояние: "completion-in-flight state".
- Состояние: "error state".

### Figureoverview

- Фигура: "android cash collection screen".
- Фигура: "cash collection exit-confirmation dialog".
- Фигура: "cash completion CTA state".

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

**Localizedsummary:** Локализованный обзор поверхности "android-driver-delivery-correction" для роли водитель на платформе android.

## Localized

**Purpose:** Поверхность "android-driver-delivery-correction" представляет страница для роли водитель на платформе android; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Android driver reconciliation screen for editing accepted quantities, assigning rejection reasons, previewing refund impact, and submitting amended manifests.

### Layoutoverview

- Зона "top-app-bar" расположена в области "top scaffold app bar". Содержимое: "left: back arrow button"; "center: Verify Cargo title; optional retailer subtitle"; "right: modified-count badge when modifications exist".
- Зона "loading-or-error-state" расположена в области "center body when list unavailable". Содержимое: "loading state with spinner and Loading manifest text"; "error state with Inventory2 icon, failed headline, and message".
- Зона "manifest-list" расположена в области "scrollable body when loaded". Содержимое: "section label showing manifest item count"; "order ID mono badge"; "line item cards with product name, SKU, accepted quantity, total, and modify icon"; "reason tag on modified items".
- Зона "modification-bottom-sheet" расположена в области "modal bottom sheet". Правило видимости: "visible when editingIndex targets a line item". Содержимое: "product header"; "accepted quantity stepper"; "auto-calculated rejected text"; "rejection reason filter chips"; "adjusted line total preview"; "Apply Modification or No Changes button".
- Зона "sticky-footer" расположена в области "bottom bar". Содержимое: "original total row"; "animated refund delta row"; "adjusted total row"; "Submit Amendment or Confirm and Complete Delivery button".
- Зона "confirm-dialog" расположена в области "alert overlay". Правило видимости: "visible when showConfirmDialog is true". Содержимое: "Warning icon"; "Confirm Amendment title"; "modification count text"; "refund amount panel"; "Confirm Amendment button"; "Cancel button".

### Controloverview

- Кнопка "Back" расположена в "top-app-bar navigation icon". Стиль: "icon button".
- Кнопка "Modify item" расположена в "line item card top-right". Стиль: "small icon button".
- Кнопка "Decrease accepted" расположена в "modification-bottom-sheet stepper". Стиль: "filled icon button".
- Кнопка "Increase accepted" расположена в "modification-bottom-sheet stepper". Стиль: "filled icon button".
- Кнопка "Rejection reason chip" расположена в "modification-bottom-sheet". Стиль: "filter chip".
- Кнопка "Apply Modification" расположена в "modification-bottom-sheet footer". Стиль: "full-width primary". Правило видимости: "rejected quantity greater than zero".
- Кнопка "No Changes" расположена в "modification-bottom-sheet footer". Стиль: "full-width primary". Правило видимости: "rejected quantity equals zero".
- Кнопка "Submit Amendment" расположена в "sticky-footer". Стиль: "full-width error-colored button". Правило видимости: "state.hasModifications is true".
- Кнопка "Confirm & Complete Delivery" расположена в "sticky-footer". Стиль: "full-width primary button". Правило видимости: "state.hasModifications is false".
- Кнопка "Confirm Amendment" расположена в "confirm-dialog confirm button". Стиль: "error primary".
- Кнопка "Cancel" расположена в "confirm-dialog dismiss button". Стиль: "text button".

### Iconoverview

- Иконка "ArrowBack" используется в зоне "top-app-bar navigation icon".
- Иконка "Edit" используется в зоне "line item modify control".
- Иконка "Warning" используется в зоне "modified reason tag, sticky-footer submit CTA, and confirm dialog".
- Иконка "Remove" используется в зоне "bottom-sheet decrement control".
- Иконка "Add" используется в зоне "bottom-sheet increment control".
- Иконка "CheckCircle" используется в зоне "non-modified footer CTA".
- Иконка "Inventory2" используется в зоне "error state".
- Иконка "CircularProgressIndicator" используется в зоне "loading state and submit CTA while submitting".

### Flowoverview

**Flowid:** modify-line-item

**Summary:** Поток "modify-line-item" содержит 6 шаг(а/ов).

#### Steps

- Шаг 1: "Driver taps edit icon on a line item card".
- Шаг 2: "Modal bottom sheet opens".
- Шаг 3: "Driver changes accepted quantity with stepper".
- Шаг 4: "Rejected quantity auto-calculates".
- Шаг 5: "Driver optionally selects rejection reason chips".
- Шаг 6: "Driver applies modification and returns to list".

---

**Flowid:** footer-summary-updates

**Summary:** Поток "footer-summary-updates" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Any modification updates modified-count badge".
- Шаг 2: "Refund delta and adjusted total in sticky footer recompute live".
- Шаг 3: "Footer CTA changes from confirm-complete to submit-amendment mode".

---

**Flowid:** confirm-amendment

**Summary:** Поток "confirm-amendment" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Driver taps Submit Amendment".
- Шаг 2: "Alert dialog summarizes modification count and refund amount".
- Шаг 3: "Driver confirms".
- Шаг 4: "ViewModel submits amendment and route completes on success".

---


### Stateoverview

- Состояние: "loading manifest state".
- Состояние: "error state".
- Состояние: "clean manifest state".
- Состояние: "modified manifest state with badges and reason tags".
- Состояние: "modification bottom sheet open".
- Состояние: "confirm amendment dialog".
- Состояние: "submitting footer state".

### Figureoverview

- Фигура: "android delivery correction full screen".
- Фигура: "modified line item card".
- Фигура: "quantity-edit bottom sheet with reason chips".
- Фигура: "sticky footer with refund delta".
- Фигура: "confirm amendment dialog".

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

**Localizedsummary:** Локализованный обзор поверхности "android-driver-offload-review" для роли водитель на платформе android.

## Localized

**Purpose:** Поверхность "android-driver-offload-review" представляет страница для роли водитель на платформе android; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Android driver cargo-review screen for checking accepted totals, excluding damaged units, and confirming offload before payment or cash collection routing.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "left: back icon button; OFFLOAD REVIEW monospace label; retailer name".
- Зона "totals-bar" расположена в области "below header". Содержимое: "original total cluster"; "adjusted total cluster with dynamic color".
- Зона "line-item-list" расположена в области "scrollable body". Содержимое: "status icon per line item"; "product name"; "quantity and unit price line"; "accepted total"; "rejected quantity stepper".
- Зона "error-row" расположена в области "above footer when present". Правило видимости: "visible when state.error is non-null". Содержимое: "red error text".
- Зона "footer-cta" расположена в области "bottom full-width container". Содержимое: "Confirm Offload or Amend and Confirm Offload button with spinner state".

### Controloverview

- Кнопка "Back" расположена в "header-left". Стиль: "icon button".
- Кнопка "Reduce rejected" расположена в "line-item stepper". Стиль: "icon button".
- Кнопка "Increase rejected" расположена в "line-item stepper". Стиль: "icon button".
- Кнопка "Confirm Offload" расположена в "footer-cta". Стиль: "full-width primary".
- Кнопка "Amend & Confirm Offload" расположена в "footer-cta". Стиль: "full-width primary". Правило видимости: "state.hasExclusions is true".

### Iconoverview

- Иконка "ArrowBack" используется в зоне "header-left back control".
- Иконка "CheckCircle" используется в зоне "line-item row when no exclusions".
- Иконка "RemoveCircleOutline" используется в зоне "line-item row when fully rejected".
- Иконка "RemoveCircle" используется в зоне "stepper decrement".
- Иконка "AddCircle" используется в зоне "stepper increment".
- Иконка "CircularProgressIndicator" используется в зоне "footer CTA while submitting".

### Flowoverview

**Flowid:** line-item-exclusion-adjustment

**Summary:** Поток "line-item-exclusion-adjustment" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Driver uses stepper controls to increase or decrease rejected quantity per line item".
- Шаг 2: "Accepted total and status coloring recompute per row".
- Шаг 3: "Adjusted total in totals bar updates".

---

**Flowid:** confirm-offload-clean

**Summary:** Поток "confirm-offload-clean" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Driver leaves all rows fully accepted".
- Шаг 2: "Driver taps Confirm Offload".
- Шаг 3: "OffloadReviewViewModel confirms offload and returns result".
- Шаг 4: "Route branches to payment or cash flow based on response".

---

**Flowid:** confirm-offload-amended

**Summary:** Поток "confirm-offload-amended" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Driver excludes one or more units".
- Шаг 2: "Footer label changes to Amend and Confirm Offload".
- Шаг 3: "Submission persists amended quantities before moving to downstream payment handling".

---


### Stateoverview

- Состояние: "clean offload state".
- Состояние: "partially rejected line-item state".
- Состояние: "fully rejected line-item state".
- Состояние: "submitting state".
- Состояние: "error state".

### Figureoverview

- Фигура: "android offload review full screen".
- Фигура: "line-item stepper detail".
- Фигура: "totals bar before and after exclusions".
- Фигура: "submitting offload CTA".

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

**Localizedsummary:** Локализованный обзор поверхности "android-driver-payment-waiting" для роли водитель на платформе android.

## Localized

**Purpose:** Поверхность "android-driver-payment-waiting" представляет страница для роли водитель на платформе android; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Android driver settlement screen that waits for electronic payment completion before enabling delivery finalization.

### Layoutoverview

- Зона "status-stack" расположена в области "center vertical stack". Содержимое: "hourglass or check-circle icon"; "AWAITING PAYMENT or PAYMENT RECEIVED heading"; "amount text"; "credit-card icon"; "Payme label".
- Зона "waiting-copy" расположена в области "below payment method label". Правило видимости: "visible when state.paymentSettled is false". Содержимое: "Waiting for retailer to complete payment text".
- Зона "error-row" расположена в области "above completion CTA". Правило видимости: "visible when state.error is non-null". Содержимое: "centered error text".
- Зона "completion-cta" расположена в области "bottom of central stack". Содержимое: "Complete Delivery button with disabled state until settlement".

### Controloverview

- Кнопка "Complete Delivery" расположена в "completion-cta". Стиль: "full-width primary, disabled until paymentSettled".

### Iconoverview

- Иконка "HourglassTop" используется в зоне "status-stack when awaiting".
- Иконка "CheckCircle" используется в зоне "status-stack when settled".
- Иконка "CreditCard" используется в зоне "payment method indicator".
- Иконка "CircularProgressIndicator" используется в зоне "completion CTA while isCompleting".

### Flowoverview

**Flowid:** waiting-to-settled

**Summary:** Поток "waiting-to-settled" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Driver remains on waiting screen after offload confirmation".
- Шаг 2: "ViewModel observes payment settlement state".
- Шаг 3: "Heading, icon, and CTA state update once paymentSettled becomes true".

---

**Flowid:** complete-after-settlement

**Summary:** Поток "complete-after-settlement" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Driver taps Complete Delivery once enabled".
- Шаг 2: "ViewModel completes order".
- Шаг 3: "Route exits through onComplete when state.completed becomes true".

---


### Stateoverview

- Состояние: "awaiting-payment state".
- Состояние: "payment-received state".
- Состояние: "CTA completing state".
- Состояние: "error state".

### Figureoverview

- Фигура: "android awaiting payment screen".
- Фигура: "android payment received screen".
- Фигура: "enabled and disabled completion CTA comparison".

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

**Localizedsummary:** Локализованный обзор поверхности "android-driver-main-shell" для роли водитель на платформе android.

## Localized

**Purpose:** Поверхность "android-driver-main-shell" представляет страница для роли водитель на платформе android; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Authenticated driver execution shell that holds the core four-tab workspace and routes into scanner, offload review, payment waiting, cash collection, and correction flows.

### Layoutoverview

- Зона "animated-content-region" расположена в области "center full-width". Содержимое: "HOME content"; "MAP content"; "RIDES content"; "PROFILE content".
- Зона "bottom-stack" расположена в области "bottom full-width". Содержимое: "optional activeRideBar slot"; "80dp NavigationBar".
- Зона "secondary-routes" расположена в области "outside root shell but in same navigation graph". Содержимое: "ScannerScreen"; "OffloadReviewScreen"; "PaymentWaitingScreen"; "CashCollectionScreen"; "DeliveryCorrectionScreen".

### Controloverview

- Кнопка "HOME tab" расположена в "bottom nav". Стиль: "NavigationBarItem".
- Кнопка "MAP tab" расположена в "bottom nav". Стиль: "NavigationBarItem".
- Кнопка "RIDES tab" расположена в "bottom nav". Стиль: "NavigationBarItem".
- Кнопка "PROFILE tab" расположена в "bottom nav". Стиль: "NavigationBarItem".
- Кнопка "scan entry CTA" расположена в "home content route handoff". Стиль: "screen CTA routed to scanner".
- Кнопка "active ride bar tap target" расположена в "bottom stack above nav". Стиль: "floating summary CTA when supplied by host content".

### Iconoverview

- Иконка "Home filled and outlined" используется в зоне "bottom nav".
- Иконка "Map filled and outlined" используется в зоне "bottom nav".
- Иконка "ListAlt filled and outlined" используется в зоне "bottom nav".
- Иконка "Person filled and outlined" используется в зоне "bottom nav".

### Flowoverview

**Flowid:** scanner-to-offload

**Summary:** Поток "scanner-to-offload" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Driver taps scan entry from Home".
- Шаг 2: "Navigation pushes ScannerScreen".
- Шаг 3: "Validated QR result pops scanner".
- Шаг 4: "Navigation pushes OffloadReviewScreen with orderId and retailerName".

---

**Flowid:** offload-to-payment-or-cash

**Summary:** Поток "offload-to-payment-or-cash" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Driver confirms offload".
- Шаг 2: "Navigation examines paymentMethod in response".
- Шаг 3: "Cash path routes to CashCollectionScreen".
- Шаг 4: "Card path routes to PaymentWaitingScreen".

---

**Flowid:** return-to-main-shell

**Summary:** Поток "return-to-main-shell" содержит 1 шаг(а/ов).

#### Steps

- Шаг 1: "Completion actions from payment waiting, cash collection, or correction pop back to MAIN without destroying the main workspace".

---


### Stateoverview

- Состояние: "home tab active".
- Состояние: "map tab active".
- Состояние: "rides tab active".
- Состояние: "profile tab active".
- Состояние: "active ride bar present".
- Состояние: "scanner route open".
- Состояние: "offload review route open".
- Состояние: "payment waiting route open".
- Состояние: "cash collection route open".
- Состояние: "correction route open".

### Figureoverview

- Фигура: "driver main shell with four-tab navigation".
- Фигура: "active ride bar plus bottom navigation".
- Фигура: "scanner handoff sequence".
- Фигура: "offload review to payment branch sequence".
- Фигура: "cash collection route state".

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

**Localizedsummary:** Локализованный обзор поверхности "android-driver-scanner" для роли водитель на платформе android.

## Localized

**Purpose:** Поверхность "android-driver-scanner" представляет страница для роли водитель на платформе android; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Android driver scan-entry screen that validates retailer QR codes from a live camera preview and branches into cargo review or retry states.

### Layoutoverview

- Зона "camera-preview" расположена в области "full-screen base layer". Содержимое: "CameraPreview AndroidView filling the full screen".
- Зона "bottom-scan-prompt" расположена в области "bottom center above safe area while scanning". Правило видимости: "visible when state.isScanning is true". Содержимое: "rounded dark prompt card"; "QrCodeScanner icon"; "scanner prompt text".
- Зона "validating-overlay" расположена в области "full-screen modal overlay". Правило видимости: "visible when state.isSubmitting is true". Содержимое: "dark scrim"; "CircularProgressIndicator"; "Validating QR text".
- Зона "validated-overlay" расположена в области "full-screen modal overlay". Правило видимости: "visible when state.validated is non-null". Содержимое: "CheckCircle icon"; "QR Verified title"; "retailer name"; "total amount"; "item count"; "Review Cargo button"; "Scan Next filled tonal button".
- Зона "error-overlay" расположена в области "full-screen modal overlay". Правило видимости: "visible when state.error is non-null". Содержимое: "ErrorOutline icon"; "error text"; "Retry button".
- Зона "close-control" расположена в области "top-right corner". Содержимое: "white close icon button".

### Controloverview

- Кнопка "Close scanner" расположена в "close-control top-right". Стиль: "icon button".
- Кнопка "Review Cargo" расположена в "validated-overlay". Стиль: "primary button".
- Кнопка "Scan Next" расположена в "validated-overlay". Стиль: "filled tonal button".
- Кнопка "Retry" расположена в "error-overlay". Стиль: "primary button".

### Iconoverview

- Иконка "QrCodeScanner" используется в зоне "bottom-scan-prompt".
- Иконка "CheckCircle" используется в зоне "validated-overlay".
- Иконка "ErrorOutline" используется в зоне "error-overlay".
- Иконка "Close" используется в зоне "close-control".
- Иконка "CircularProgressIndicator" используется в зоне "validating-overlay".

### Flowoverview

**Flowid:** scan-and-validate

**Summary:** Поток "scan-and-validate" содержит 5 шаг(а/ов).

#### Steps

- Шаг 1: "Driver enters scanner route".
- Шаг 2: "CameraPreview analyzes barcodes continuously".
- Шаг 3: "Detected value is handed to ScannerViewModel".
- Шаг 4: "Validated payload opens verified overlay".
- Шаг 5: "Driver taps Review Cargo to continue to offload review".

---

**Flowid:** scan-reset

**Summary:** Поток "scan-reset" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Driver taps Scan Next after a successful validation or Retry after an error".
- Шаг 2: "Scanner state resets and live preview resumes".

---

**Flowid:** scanner-exit

**Summary:** Поток "scanner-exit" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Driver taps top-right close button".
- Шаг 2: "Route exits through onClose".

---


### Stateoverview

- Состояние: "active scan state".
- Состояние: "validating overlay state".
- Состояние: "validated overlay state".
- Состояние: "error overlay state".

### Figureoverview

- Фигура: "android scanner screen with prompt card".
- Фигура: "scanner validating overlay".
- Фигура: "validated overlay with review and scan-next buttons".
- Фигура: "scanner error overlay".

---

**Dossierfile:** driver-android-secondary-surfaces.json

**Bundleid:** driver-android-secondary-surfaces

**Appid:** driver-app-android

**Platform:** android

**Role:** DRIVER

**Status:** implemented

**Entrytype:** bundle

**Localizedsummary:** Локализованный пакет "driver-android-secondary-surfaces" охватывает 6 поверхностей приложения "driver-app-android".

## Surfaces

**Pageid:** android-driver-login

**Navroute:** login

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/auth/LoginScreen.kt

**Localizedsummary:** Локализованный обзор поверхности "android-driver-login" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-driver-login" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Android driver sign-in form using phone and PIN with IME management and auth loading feedback.

#### Layoutoverview

- Зона макета: "brand header".
- Зона макета: "phone and PIN text field column".
- Зона макета: "PIN visibility icon button".
- Зона макета: "login CTA and error state".

#### Controloverview

- Элемент управления: "PIN visibility icon button in PIN field trailing slot".
- Элемент управления: "Login button below fields".

#### Iconoverview

- Иконографическая привязка: "brand mark at screen top".
- Иконографическая привязка: "eye visibility icon".

#### Flowoverview

**Summary:** Поток фиксируется как: "type phone and PIN".

---

**Summary:** Поток фиксируется как: "toggle PIN visibility".

---

**Summary:** Поток фиксируется как: "submit auth coroutine and persist session".

---

**Summary:** Поток фиксируется как: "show loading spinner during auth".

---


#### Dependencyoverview

##### Reads

- driver login API

##### Writes

- driver token store

##### Localizednotes

- Чтение: "driver login API".
- Запись: "driver token store".

#### Stateoverview

- Состояние: "idle".
- Состояние: "loading".
- Состояние: "error".

#### Figureoverview

- Фигура: "android login screen with phone and PIN form".

#### Minifeatureoverview

- Минифункция: "phone prefill".
- Минифункция: "PIN field".
- Минифункция: "visibility toggle".
- Минифункция: "loading spinner".
- Минифункция: "error message".

**Minifeaturecount:** 5

---

**Pageid:** android-driver-home

**Navroute:** HOME

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/home/HomeScreen.kt

**Localizedsummary:** Локализованный обзор поверхности "android-driver-home" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-driver-home" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Android driver home dashboard with status chips, vehicle card, transit control, and quick actions.

#### Layoutoverview

- Зона макета: "time-based greeting and status chips".
- Зона макета: "vehicle info card".
- Зона макета: "transit control card".
- Зона макета: "today summary band".
- Зона макета: "quick action row".
- Зона макета: "recent activity list".

#### Controloverview

- Элемент управления: "Open Map CTA".
- Элемент управления: "Scan QR CTA".

#### Iconoverview

- Иконографическая привязка: "route-state icon".
- Иконографическая привязка: "truck or cargo glyphs in cards".

#### Flowoverview

**Summary:** Поток фиксируется как: "refresh dashboard state".

---

**Summary:** Поток фиксируется как: "jump to map".

---

**Summary:** Поток фиксируется как: "jump to scanner".

---


#### Dependencyoverview

##### Reads

- ManifestViewModel state

##### Writes


##### Localizednotes

- Чтение: "ManifestViewModel state".
- Запись: нет.

#### Stateoverview

- Состояние: "idle".
- Состояние: "on route".
- Состояние: "loading".

#### Figureoverview

- Фигура: "android driver home dashboard".

#### Minifeatureoverview

- Минифункция: "greeting".
- Минифункция: "status chips".
- Минифункция: "vehicle card".
- Минифункция: "transit control".
- Минифункция: "summary band".
- Минифункция: "Open Map CTA".
- Минифункция: "Scan QR CTA".
- Минифункция: "recent activity list".

**Minifeaturecount:** 8

---

**Pageid:** android-driver-map

**Navroute:** MAP

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/map/MapScreen.kt

**Localizedsummary:** Локализованный обзор поверхности "android-driver-map" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-driver-map" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Stub placeholder reserving the future Google Maps execution surface in the Android driver stack.

#### Layoutoverview

- Зона макета: "centered placeholder icon".
- Зона макета: "stub title and explanatory subtitle".

#### Controloverview

- Элемент управления: "none; stub surface".

#### Iconoverview

- Иконографическая привязка: "Map icon centered".

#### Flowoverview

**Summary:** Поток фиксируется как: "static placeholder only".

---


#### Dependencyoverview

##### Reads


##### Writes


##### Localizednotes

- Чтение: нет.
- Запись: нет.

#### Stateoverview

- Состояние: "single stub state".

#### Figureoverview

- Фигура: "stub map placeholder figure".

#### Minifeatureoverview

- Минифункция: "map pending icon".
- Минифункция: "placeholder messaging".

**Minifeaturecount:** 2

---

**Pageid:** android-driver-rides

**Navroute:** RIDES

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/manifest/ManifestScreen.kt

**Localizedsummary:** Локализованный обзор поверхности "android-driver-rides" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-driver-rides" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Android route manifest ledger with loading-mode reversal for physical truck packing and upcoming stop review.

#### Layoutoverview

- Зона макета: "UPCOMING header with pending count".
- Зона макета: "Loading Mode switch row".
- Зона макета: "ride card lazy list".
- Зона макета: "loading or empty states".

#### Controloverview

- Элемент управления: "Loading Mode switch in header".
- Элемент управления: "ride card tap target".

#### Iconoverview

- Иконографическая привязка: "loading sequence badge".
- Иконографическая привязка: "status pill".

#### Flowoverview

**Summary:** Поток фиксируется как: "toggle loading mode".

---

**Summary:** Поток фиксируется как: "tap ride to focus mission".

---

**Summary:** Поток фиксируется как: "refresh manifest".

---


#### Dependencyoverview

##### Reads

- ManifestViewModel.state

##### Writes

- selected mission state

##### Localizednotes

- Чтение: "ManifestViewModel.state".
- Запись: "selected mission state".

#### Stateoverview

- Состояние: "standard order".
- Состояние: "loading order".
- Состояние: "empty".
- Состояние: "loading".

#### Figureoverview

- Фигура: "android rides manifest with loading mode switch".

#### Minifeatureoverview

- Минифункция: "pending count badge".
- Минифункция: "loading mode switch".
- Минифункция: "ride cards".
- Минифункция: "sequence badge".
- Минифункция: "status pill".
- Минифункция: "empty state".

**Minifeaturecount:** 6

---

**Pageid:** android-driver-profile

**Navroute:** PROFILE

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/profile/ProfileScreen.kt

**Localizedsummary:** Локализованный обзор поверхности "android-driver-profile" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-driver-profile" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Android driver profile screen with truck identity, stats, quick actions, and ride-history review.

#### Layoutoverview

- Зона макета: "profile title header".
- Зона макета: "identity card".
- Зона макета: "truck and completion info grid".
- Зона макета: "quick actions row".
- Зона макета: "ride history list".
- Зона макета: "stats section".

#### Controloverview

- Элемент управления: "Sync quick action".
- Элемент управления: "Logout quick action".
- Элемент управления: "Settings quick action".

#### Iconoverview

- Иконографическая привязка: "initials avatar".
- Иконографическая привязка: "quick action icons".

#### Flowoverview

**Summary:** Поток фиксируется как: "sync state".

---

**Summary:** Поток фиксируется как: "logout session".

---

**Summary:** Поток фиксируется как: "review history".

---


#### Dependencyoverview

##### Reads

- ManifestViewModel driver and order stats

##### Writes

- sync or logout state

##### Localizednotes

- Чтение: "ManifestViewModel driver and order stats".
- Запись: "sync or logout state".

#### Stateoverview

- Состояние: "active".
- Состояние: "idle".
- Состояние: "history populated".

#### Figureoverview

- Фигура: "android driver profile screen".

#### Minifeatureoverview

- Минифункция: "identity card".
- Минифункция: "status pill".
- Минифункция: "truck grid".
- Минифункция: "Sync action".
- Минифункция: "Logout action".
- Минифункция: "Settings action".
- Минифункция: "history ledger".
- Минифункция: "stats band".

**Minifeaturecount:** 8

---

**Pageid:** android-driver-correction

**Navroute:** correction/{orderId}/{retailerName}

**Surfacetype:** screen

**Sourcefile:** apps/driver-app-android/app/src/main/java/com/pegasus/driver/ui/screens/manifest/DeliveryCorrectionScreen.kt

**Localizedsummary:** Локализованный обзор поверхности "android-driver-correction" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-driver-correction" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Alias dossier for the Android driver delivery-correction workflow already documented as a primary execution surface.

#### Layoutoverview

- Зона макета: "header app bar".
- Зона макета: "manifest item cards".
- Зона макета: "sticky summary footer".
- Зона макета: "correction bottom sheet and confirmation dialog overlays".

#### Controloverview

- Элемент управления: "Modify item action".
- Элемент управления: "confirm amendment action".

#### Iconoverview

- Иконографическая привязка: "item correction glyphs".
- Иконографическая привязка: "dialog warning icon".

#### Flowoverview

**Summary:** Поток фиксируется как: "open correction editor".

---

**Summary:** Поток фиксируется как: "adjust delivered and rejected quantities".

---

**Summary:** Поток фиксируется как: "confirm amendment".

---


#### Dependencyoverview

##### Reads

- delivery correction payload

##### Writes

- correction submission

##### Localizednotes

- Чтение: "delivery correction payload".
- Запись: "correction submission".

#### Stateoverview

- Состояние: "review".
- Состояние: "editing".
- Состояние: "confirming".

#### Figureoverview

- Фигура: "delivery correction alias figure".

#### Minifeatureoverview

- Минифункция: "item cards".
- Минифункция: "sticky footer".
- Минифункция: "bottom sheet editor".
- Минифункция: "confirmation dialog".

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

**Localizedsummary:** Локализованный обзор поверхности "ios-driver-cash-collection" для роли водитель на платформе iOS.

## Localized

**Purpose:** Поверхность "ios-driver-cash-collection" представляет страница для роли водитель на платформе iOS; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Driver cash-confirmation screen used when retailer payment is collected physically before delivery completion.

### Layoutoverview

- Зона "top-close-row" расположена в области "top safe-area inset". Содержимое: "circular close button aligned right".
- Зона "center-cash-stack" расположена в области "center vertical stack". Содержимое: "banknote icon"; "Collect Cash title"; "order ID"; "amount"; "helper copy".
- Зона "error-row" расположена в области "above footer CTA when present". Правило видимости: "visible when errorMessage exists". Содержимое: "destructive error text".
- Зона "footer-cta" расположена в области "bottom full-width". Содержимое: "Cash Collected — Complete button with optional spinner".

### Controloverview

- Кнопка "Close" расположена в "top-close-row right". Стиль: "icon button".
- Кнопка "Cash Collected — Complete" расположена в "footer-cta". Стиль: "full-width primary".

### Iconoverview

- Иконка "xmark" используется в зоне "top-close-row".
- Иконка "banknote.fill" используется в зоне "center-cash-stack top".
- Иконка "ProgressView" используется в зоне "footer CTA when completing".

### Flowoverview

**Flowid:** cancel-cash-collection

**Summary:** Поток "cancel-cash-collection" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Driver taps close button".
- Шаг 2: "View exits through onCancel callback".

---

**Flowid:** collect-cash-and-complete

**Summary:** Поток "collect-cash-and-complete" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Driver confirms physical collection".
- Шаг 2: "Driver taps Cash Collected — Complete".
- Шаг 3: "View calls collectCash(orderId)".
- Шаг 4: "Successful response exits through onCompleted callback".

---


### Stateoverview

- Состояние: "cash collection idle state".
- Состояние: "completion-in-flight state".
- Состояние: "inline error state".

### Figureoverview

- Фигура: "cash collection screen".
- Фигура: "cash collection CTA state".
- Фигура: "cash collection error state".

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

**Localizedsummary:** Локализованный обзор поверхности "ios-driver-delivery-correction" для роли водитель на платформе iOS.

## Localized

**Purpose:** Поверхность "ios-driver-delivery-correction" представляет страница для роли водитель на платформе iOS; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Driver amendment screen for toggling manifest items between delivered and rejected states and calculating refund deltas before submission.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "left: Back button; Delivery Correction title; order ID"; "right: StatusPill showing rejected count or all-clear state".
- Зона "loading-region" расположена в области "center body". Правило видимости: "visible when vm.isLoading is true". Содержимое: "Loading line items progress indicator".
- Зона "manifest-list" расположена в области "scrollable body when loaded". Содержимое: "MANIFEST ITEMS section label"; "line item cards with sku, quantity x unit price, status pill, line total, bottom status bar".
- Зона "summary-bar" расположена в области "bottom material overlay". Содержимое: "original total row"; "refund delta row when refundDelta > 0"; "divider"; "adjusted total row"; "Submit Amendment or All Items Delivered CTA".
- Зона "confirm-alert" расположена в области "system alert overlay". Правило видимости: "visible when showConfirmAlert is true". Содержимое: "Confirm Amendment title"; "cancel button"; "destructive submit button"; "message with rejected count and refund delta".

### Controloverview

- Кнопка "Back" расположена в "header-left". Стиль: "inline icon-text button".
- Кнопка "line-item card tap target" расположена в "manifest-list". Стиль: "whole-card toggle button".
- Кнопка "Submit Amendment" расположена в "summary-bar footer". Стиль: "full-width destructive". Правило видимости: "one or more items rejected".
- Кнопка "All Items Delivered" расположена в "summary-bar footer". Стиль: "disabled muted". Правило видимости: "no items rejected".
- Кнопка "Cancel" расположена в "confirm-alert". Стиль: "system alert cancel action".
- Кнопка "Submit" расположена в "confirm-alert". Стиль: "system alert destructive action".

### Iconoverview

- Иконка "chevron.left" используется в зоне "header back button".
- Иконка "StatusPill capsule" используется в зоне "header-right".
- Иконка "bottom status bar on each line item card" используется в зоне "line item footer".
- Иконка "ProgressView" используется в зоне "loading-region".

### Flowoverview

**Flowid:** load-manifest-items

**Summary:** Поток "load-manifest-items" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "View loads line items on task start".
- Шаг 2: "Loading state is replaced by tappable manifest cards".

---

**Flowid:** toggle-item-status

**Summary:** Поток "toggle-item-status" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Driver taps a line item card".
- Шаг 2: "Item toggles between delivered and rejected status".
- Шаг 3: "Status pill, strikethrough, and bottom bar update".
- Шаг 4: "Summary bar recalculates refund delta and adjusted total".

---

**Flowid:** submit-amendment

**Summary:** Поток "submit-amendment" содержит 5 шаг(а/ов).

#### Steps

- Шаг 1: "Driver rejects one or more items".
- Шаг 2: "Driver taps Submit Amendment".
- Шаг 3: "Confirm Amendment alert appears with refund summary".
- Шаг 4: "Driver submits and page calls submitAmendment(orderId, driverId)".
- Шаг 5: "Successful submission exits through onAmended callback".

---


### Stateoverview

- Состояние: "loading state".
- Состояние: "all-clear manifest state".
- Состояние: "mixed delivered and rejected items".
- Состояние: "summary bar with refund delta".
- Состояние: "confirm amendment alert".

### Figureoverview

- Фигура: "delivery correction full screen".
- Фигура: "line item card with rejected state".
- Фигура: "summary bar with refund delta".
- Фигура: "confirm amendment alert".

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

**Localizedsummary:** Локализованный обзор поверхности "ios-driver-offload-review" для роли водитель на платформе iOS.

## Localized

**Purpose:** Поверхность "ios-driver-offload-review" представляет страница для роли водитель на платформе iOS; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Driver delivery-review screen for confirming offload, partially rejecting damaged units, and branching into payment collection flows.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "left: OFFLOAD REVIEW label; order ID in monospace"; "right: circular close button".
- Зона "retailer-total-row" расположена в области "below header". Содержимое: "retailer name"; "total amount".
- Зона "line-item-list" расположена в области "scrollable middle region". Содержимое: "product name"; "quantity x unit price line"; "minus button"; "rejected quantity counter"; "plus button".
- Зона "error-row" расположена в области "above footer CTA when present". Правило видимости: "visible when errorMessage is non-null". Содержимое: "inline destructive error text".
- Зона "footer-cta" расположена в области "bottom full-width". Содержимое: "Confirm Offload button with optional spinner".

### Controloverview

- Кнопка "Close" расположена в "header-right circular button". Стиль: "icon button".
- Кнопка "minus reject quantity" расположена в "each line-item stepper". Стиль: "icon stepper control".
- Кнопка "plus reject quantity" расположена в "each line-item stepper". Стиль: "icon stepper control".
- Кнопка "Confirm Offload" расположена в "footer-cta". Стиль: "full-width primary".

### Iconoverview

- Иконка "xmark" используется в зоне "header-right close control".
- Иконка "minus.circle.fill" используется в зоне "line-item stepper decrement".
- Иконка "plus.circle.fill" используется в зоне "line-item stepper increment".
- Иконка "ProgressView" используется в зоне "Confirm Offload button when submitting".

### Flowoverview

**Flowid:** quantity-rejection-adjustment

**Summary:** Поток "quantity-rejection-adjustment" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Driver reviews line items".
- Шаг 2: "Driver uses plus or minus buttons to set rejected quantity per item".
- Шаг 3: "Item styling changes to delivered, partial, or fully rejected visual state".

---

**Flowid:** confirm-offload-no-rejections

**Summary:** Поток "confirm-offload-no-rejections" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Driver leaves all rejected quantities at zero".
- Шаг 2: "Driver taps Confirm Offload".
- Шаг 3: "Page calls confirmOffload on fleet service".
- Шаг 4: "Successful response exits through onConfirm callback".

---

**Flowid:** confirm-offload-with-amendment

**Summary:** Поток "confirm-offload-with-amendment" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Driver marks one or more rejected quantities".
- Шаг 2: "Page first calls amendOrder with derived status per line item".
- Шаг 3: "Page then calls confirmOffload".
- Шаг 4: "Workflow branches downstream based on returned payment mode".

---


### Stateoverview

- Состояние: "all items delivered state".
- Состояние: "partial rejection state".
- Состояние: "full rejection for a line-item".
- Состояние: "submitting CTA state".
- Состояние: "inline error state".

### Figureoverview

- Фигура: "offload review full screen".
- Фигура: "line-item row with stepper controls".
- Фигура: "mixed accepted and rejected quantities".
- Фигура: "confirm-offload submitting state".

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

**Localizedsummary:** Локализованный обзор поверхности "ios-driver-payment-waiting" для роли водитель на платформе iOS.

## Localized

**Purpose:** Поверхность "ios-driver-payment-waiting" представляет страница для роли водитель на платформе iOS; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Driver payment-settlement holding screen that waits for a websocket settlement event before enabling delivery completion.

### Layoutoverview

- Зона "status-stack" расположена в области "center vertical stack". Содержимое: "status icon"; "title"; "order ID"; "amount".
- Зона "waiting-copy" расположена в области "below amount when payment is unsettled". Правило видимости: "visible when isSettled is false". Содержимое: "ProgressView spinner"; "Retailer is completing payment helper text".
- Зона "error-row" расположена в области "above completion CTA when present". Правило видимости: "visible when errorMessage exists". Содержимое: "destructive error text".
- Зона "completion-cta" расположена в области "bottom full-width". Содержимое: "Complete Delivery button disabled until settlement".

### Controloverview

- Кнопка "Complete Delivery" расположена в "completion-cta". Стиль: "full-width primary when settled and muted disabled button when unsettled".

### Iconoverview

- Иконка "clock.fill" используется в зоне "status-stack icon when unsettled".
- Иконка "checkmark.seal.fill" используется в зоне "status-stack icon when settled".
- Иконка "ProgressView" используется в зоне "waiting-copy".

### Flowoverview

**Flowid:** settlement-wait-loop

**Summary:** Поток "settlement-wait-loop" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "View opens websocket to /v1/ws/driver with driver_id and bearer token".
- Шаг 2: "Driver watches awaiting-payment state".
- Шаг 3: "Page listens for PAYMENT_SETTLED matching the current orderId".

---

**Flowid:** settlement-received

**Summary:** Поток "settlement-received" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "PAYMENT_SETTLED websocket message arrives".
- Шаг 2: "isSettled flips to true".
- Шаг 3: "status icon changes from clock to seal".
- Шаг 4: "Complete Delivery button becomes enabled".

---

**Flowid:** complete-delivery

**Summary:** Поток "complete-delivery" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Driver taps Complete Delivery after settlement".
- Шаг 2: "Page calls completeOrder".
- Шаг 3: "Successful completion exits through onCompleted callback".

---


### Stateoverview

- Состояние: "awaiting payment state".
- Состояние: "settled state".
- Состояние: "completion-in-flight state".
- Состояние: "error state".
- Состояние: "websocket reconnect behavior after failure".

### Figureoverview

- Фигура: "awaiting payment screen".
- Фигура: "payment received screen".
- Фигура: "disabled versus enabled completion CTA comparison".

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

**Localizedsummary:** Локализованный обзор поверхности "ios-driver-qr-scanner" для роли водитель на платформе iOS.

## Localized

**Purpose:** Поверхность "ios-driver-qr-scanner" представляет страница для роли водитель на платформе iOS; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Driver scan-entry screen that overlays a QR targeting reticle on a live camera preview and routes validated scans into the offload workflow.

### Layoutoverview

- Зона "camera-preview" расположена в области "full-screen base layer". Содержимое: "live camera feed".
- Зона "top-cancel-bar" расположена в области "top safe-area inset". Содержимое: "Cancel text button aligned left".
- Зона "center-reticle" расположена в области "screen center". Содержимое: "rounded square targeting border"; "light inner translucent scan area".
- Зона "processing-indicator" расположена в области "bottom center above safe area". Правило видимости: "visible when vm.isProcessing is true". Содержимое: "ProgressView spinner"; "Processing text".
- Зона "system-alert-layer" расположена в области "center overlay". Правило видимости: "visible for camera permission denial or validation result alerts". Содержимое: "camera access required alert with Close button"; "scan result alert with Rescan and Close or OK buttons".

### Controloverview

- Кнопка "Cancel" расположена в "top-cancel-bar left". Стиль: "text button".
- Кнопка "Close" расположена в "camera permission alert". Стиль: "system alert action".
- Кнопка "Rescan" расположена в "failed scan alert". Стиль: "system alert cancel action".
- Кнопка "Close" расположена в "failed scan alert". Стиль: "system alert default action".
- Кнопка "OK" расположена в "successful validation alert". Стиль: "system alert default action".

### Iconoverview

- Иконка "rounded reticle border" используется в зоне "center-reticle".
- Иконка "ProgressView" используется в зоне "processing-indicator".

### Flowoverview

**Flowid:** scan-and-validate

**Summary:** Поток "scan-and-validate" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "View requests camera permission on task start".
- Шаг 2: "Driver points camera at QR code inside reticle".
- Шаг 3: "ScannerViewModel handles scanned value".
- Шаг 4: "Validated response returns through onValidated callback".

---

**Flowid:** permission-denied

**Summary:** Поток "permission-denied" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Camera permission check fails".
- Шаг 2: "Camera Access Required alert appears".
- Шаг 3: "Driver taps Close and scanner exits via onCancel".

---

**Flowid:** scan-failure-recovery

**Summary:** Поток "scan-failure-recovery" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Validation fails".
- Шаг 2: "Alert presents Rescan and Close choices".
- Шаг 3: "Driver either resumes scanning or exits".

---


### Stateoverview

- Состояние: "live camera scan state".
- Состояние: "processing state".
- Состояние: "camera permission alert".
- Состояние: "failed scan alert".
- Состояние: "successful validation alert".

### Figureoverview

- Фигура: "scanner screen with reticle".
- Фигура: "scanner processing state".
- Фигура: "scanner permission alert".
- Фигура: "scanner failure alert".

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

**Localizedsummary:** Локализованный обзор поверхности "ios-driver-main-shell" для роли водитель на платформе iOS.

## Localized

**Purpose:** Поверхность "ios-driver-main-shell" представляет страница для роли водитель на платформе iOS; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Authenticated driver shell with tab-based execution workspace, full-screen map mode, and floating active-route summary above the tab bar.

### Layoutoverview

- Зона "auth-gate" расположена в области "root level above shell". Содержимое: "RootView switches between LoginView and MainTabView based on TokenStore.isAuthenticated".
- Зона "tab-layer" расположена в области "base layer when not on map". Содержимое: "Home tab"; "Rides tab"; "Profile tab".
- Зона "map-mode" расположена в области "full-screen replacement state". Содержимое: "FleetMapView with go-back closure".
- Зона "bottom-safe-area-inset" расположена в области "above tab bar". Содержимое: "ActiveRideBar when vm.hasActiveRoute and activeMission exist".

### Controloverview

- Кнопка "Home tab" расположена в "tab bar". Стиль: "TabView tab item".
- Кнопка "Rides tab" расположена в "tab bar". Стиль: "TabView tab item".
- Кнопка "Profile tab" расположена в "tab bar". Стиль: "TabView tab item".
- Кнопка "Home open-map trigger" расположена в "home content callback". Стиль: "screen CTA transitions to map mode".
- Кнопка "ActiveRideBar" расположена в "bottom safe-area inset". Стиль: "floating pill CTA to map mode".
- Кнопка "Map goBack" расположена в "full-screen map mode". Стиль: "callback-driven return control".

### Iconoverview

- Иконка "house.fill" используется в зоне "Home tab".
- Иконка "list.bullet" используется в зоне "Rides tab".
- Иконка "person.fill" используется в зоне "Profile tab".
- Иконка "map.fill" используется в зоне "Map mode logical tab target".
- Иконка "chevron.right" используется в зоне "ActiveRideBar trailing affordance".

### Flowoverview

**Flowid:** home-to-map-mode

**Summary:** Поток "home-to-map-mode" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Driver taps map CTA from HomeView".
- Шаг 2: "selectedTab switches to map with snappy animation".
- Шаг 3: "FleetMapView replaces the normal tab shell".

---

**Flowid:** active-route-drilldown

**Summary:** Поток "active-route-drilldown" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Active route exists".
- Шаг 2: "ActiveRideBar appears above tab bar".
- Шаг 3: "Driver taps ActiveRideBar".
- Шаг 4: "Shell transitions into full-screen map mode".

---

**Flowid:** authenticated-root-branching

**Summary:** Поток "authenticated-root-branching" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "RootView checks TokenStore.isAuthenticated".
- Шаг 2: "Authenticated drivers go directly to MainTabView".
- Шаг 3: "Unauthenticated drivers stay on LoginView".

---


### Stateoverview

- Состояние: "unauthenticated login state".
- Состояние: "home tab active".
- Состояние: "rides tab active".
- Состояние: "profile tab active".
- Состояние: "active route bar visible".
- Состояние: "full-screen map mode".

### Figureoverview

- Фигура: "driver iOS shell with tab bar".
- Фигура: "active route bar above tab bar".
- Фигура: "full-screen map mode".
- Фигура: "root auth-gate transition".

---

**Dossierfile:** driver-ios-secondary-surfaces.json

**Bundleid:** driver-ios-secondary-surfaces

**Appid:** driver-app-ios

**Platform:** ios

**Role:** DRIVER

**Status:** implemented

**Entrytype:** bundle

**Localizedsummary:** Локализованный пакет "driver-ios-secondary-surfaces" охватывает 9 поверхностей приложения "driver-app-ios".

## Surfaces

**Pageid:** ios-driver-root-gate

**Viewname:** RootView

**Surfacetype:** root-gate

**Sourcefile:** apps/driverappios/driverappios/LabDriverApp.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-driver-root-gate" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-driver-root-gate" представляет корневой шлюз для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Session gate that decides between driver login flow and the protected multi-tab shell.

#### Layoutoverview

- Зона макета: "app bootstrap region hosting root state decision".
- Зона макета: "login presentation branch for unauthenticated driver".
- Зона макета: "main-tab branch for authenticated driver".

#### Controloverview

- Элемент управления: "no persistent buttons; state resolution presents login or shell".

#### Iconoverview

- Иконографическая привязка: "none at gate level".

#### Flowoverview

**Summary:** Поток фиксируется как: "read token and driver session state on launch".

---

**Summary:** Поток фиксируется как: "show login when token is absent or invalid".

---

**Summary:** Поток фиксируется как: "show MainTabView when session is active".

---


#### Dependencyoverview

##### Reads

- driver token store
- driver session bootstrap

##### Writes


##### Localizednotes

- Чтение: "driver token store", "driver session bootstrap".
- Запись: нет.

#### Stateoverview

- Состояние: "authenticated branch".
- Состояние: "unauthenticated branch".

#### Figureoverview

- Фигура: "route-flow figure from root gate to login or main shell".

#### Minifeatureoverview

- Минифункция: "auth branch resolution".
- Минифункция: "protected-shell presentation".
- Минифункция: "guest-shell suppression".

**Minifeaturecount:** 3

---

**Pageid:** ios-driver-login

**Viewname:** LoginView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/LoginView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-driver-login" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-driver-login" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Phone and PIN sign-in screen for driver session acquisition.

#### Layoutoverview

- Зона макета: "brand crest and title stack".
- Зона макета: "phone field and PIN field cluster".
- Зона макета: "PIN visibility toggle inside secure entry".
- Зона макета: "login CTA and error message strip".

#### Controloverview

- Элемент управления: "PIN visibility eye toggle inside PIN field trailing edge".
- Элемент управления: "Login button at form footer".

#### Iconoverview

- Иконографическая привязка: "brand disk at page top".
- Иконографическая привязка: "eye or eye-slash in PIN field".

#### Flowoverview

**Summary:** Поток фиксируется как: "input phone and PIN".

---

**Summary:** Поток фиксируется как: "toggle PIN visibility".

---

**Summary:** Поток фиксируется как: "submit DriverApi login and persist token".

---

**Summary:** Поток фиксируется как: "on success dismiss into protected shell".

---


#### Dependencyoverview

##### Reads

- DriverApi.login

##### Writes

- TokenHolder session state

##### Localizednotes

- Чтение: "DriverApi.login".
- Запись: "TokenHolder session state".

#### Stateoverview

- Состояние: "idle login form".
- Состояние: "auth loading state".
- Состояние: "error state".

#### Figureoverview

- Фигура: "full login screen with phone and PIN fields".
- Фигура: "PIN field close-up with visibility toggle".

#### Minifeatureoverview

- Минифункция: "phone prefill".
- Минифункция: "secure PIN entry".
- Минифункция: "visibility toggle".
- Минифункция: "loading disable".
- Минифункция: "error banner".

**Minifeaturecount:** 5

---

**Pageid:** ios-driver-home

**Viewname:** HomeView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/HomeView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-driver-home" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-driver-home" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Driver dashboard summarizing mission status, truck identity, daily metrics, and quick-entry actions into execution surfaces.

#### Layoutoverview

- Зона макета: "dynamic greeting header".
- Зона макета: "truck and route status chips".
- Зона макета: "vehicle card and transit control card".
- Зона макета: "today summary metrics".
- Зона макета: "Open Map and quick action buttons".
- Зона макета: "recent activity ledger".

#### Controloverview

- Элемент управления: "Open Map CTA in transit control zone".
- Элемент управления: "Scan QR quick action button".
- Элемент управления: "View Manifest quick action button".

#### Iconoverview

- Иконографическая привязка: "antenna or moon status icon".
- Иконографическая привязка: "truck or route glyphs in status cards".

#### Flowoverview

**Summary:** Поток фиксируется как: "load missions and driver summary on appear".

---

**Summary:** Поток фиксируется как: "pull to refresh home data".

---

**Summary:** Поток фиксируется как: "jump to map, scanner, or rides manifest".

---


#### Dependencyoverview

##### Reads

- FleetViewModel mission summary
- driver metrics
- recent activity feed

##### Writes


##### Localizednotes

- Чтение: "FleetViewModel mission summary", "driver metrics", "recent activity feed".
- Запись: нет.

#### Stateoverview

- Состояние: "loading home state".
- Состояние: "idle off-route state".
- Состояние: "on-route active state".

#### Figureoverview

- Фигура: "driver home dashboard with quick action band".
- Фигура: "status chip and summary card close-up".

#### Minifeatureoverview

- Минифункция: "time-of-day greeting".
- Минифункция: "truck plate chip".
- Минифункция: "route-state chip".
- Минифункция: "vehicle card".
- Минифункция: "daily summary".
- Минифункция: "Open Map CTA".
- Минифункция: "Scan QR shortcut".
- Минифункция: "View Manifest shortcut".
- Минифункция: "recent activity list".

**Minifeaturecount:** 9

---

**Pageid:** ios-driver-map

**Viewname:** FleetMapView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/FleetMapView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-driver-map" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-driver-map" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Primary driver execution surface joining map telemetry, mission selection, QR scan initiation, payment branching, and delivery correction entry.

#### Layoutoverview

- Зона макета: "full-screen map region with mission markers".
- Зона макета: "zoom focus control cycling Me, Target, Both".
- Зона макета: "selected mission side or bottom detail region".
- Зона макета: "bottom action strip for Scan QR and Correct Delivery".
- Зона макета: "navigation bridge into scanner, offload, payment, cash, and correction flows".

#### Controloverview

- Элемент управления: "zoom focus cycle button over map chrome".
- Элемент управления: "Scan QR primary action in selected mission panel".
- Элемент управления: "Correct Delivery secondary action in selected mission panel".

#### Iconoverview

- Иконографическая привязка: "mission markers on map".
- Иконографическая привязка: "location and target glyphs in focus control".

#### Flowoverview

**Summary:** Поток фиксируется как: "select mission from map marker".

---

**Summary:** Поток фиксируется как: "cycle map framing mode".

---

**Summary:** Поток фиксируется как: "launch QR scanner for selected mission".

---

**Summary:** Поток фиксируется как: "after scan branch into offload review then payment or cash collection".

---

**Summary:** Поток фиксируется как: "open correction workflow for selected mission".

---


#### Dependencyoverview

##### Reads

- TelemetryViewModel live location
- FleetViewModel missions
- geofence and QR validation payloads

##### Writes

- navigation path for execution subflows

##### Localizednotes

- Чтение: "TelemetryViewModel live location", "FleetViewModel missions", "geofence and QR validation payloads".
- Запись: "navigation path for execution subflows".

#### Stateoverview

- Состояние: "no mission selected".
- Состояние: "mission previewing state".
- Состояние: "active delivery state".

#### Figureoverview

- Фигура: "full operational map with selected mission panel".
- Фигура: "map chrome close-up with focus control and CTA band".

#### Minifeatureoverview

- Минифункция: "live mission markers".
- Минифункция: "mission selection".
- Минифункция: "focus cycle control".
- Минифункция: "selected mission detail pane".
- Минифункция: "Scan QR CTA".
- Минифункция: "Correct Delivery CTA".
- Минифункция: "payment branch routing".
- Минифункция: "cash branch routing".

**Minifeaturecount:** 8

---

**Pageid:** ios-driver-rides

**Viewname:** RidesListView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/RidesListView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-driver-rides" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-driver-rides" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Route-manifest ledger of upcoming rides with physical loading-sequence toggle.

#### Layoutoverview

- Зона макета: "UPCOMING header with pending count".
- Зона макета: "Loading Mode toggle row".
- Зона макета: "mission ride card list".
- Зона макета: "pull-to-refresh scaffold".

#### Controloverview

- Элемент управления: "Loading Mode switch in header row".
- Элемент управления: "ride card tap target to select or focus mission".

#### Iconoverview

- Иконографическая привязка: "sequence badge when loading mode is enabled".
- Иконографическая привязка: "status badge on ride cards".

#### Flowoverview

**Summary:** Поток фиксируется как: "toggle loading mode to reverse sequence for warehouse loading".

---

**Summary:** Поток фиксируется как: "tap ride card to select mission and synchronize with map".

---

**Summary:** Поток фиксируется как: "refresh pending missions".

---


#### Dependencyoverview

##### Reads

- FleetViewModel.pendingMissions

##### Writes

- FleetViewModel.selectMission

##### Localizednotes

- Чтение: "FleetViewModel.pendingMissions".
- Запись: "FleetViewModel.selectMission".

#### Stateoverview

- Состояние: "standard route order".
- Состояние: "loading-sequence order".
- Состояние: "empty rides list".

#### Figureoverview

- Фигура: "rides manifest screen with loading mode toggle".
- Фигура: "single ride card with sequence badge".

#### Minifeatureoverview

- Минифункция: "pending count badge".
- Минифункция: "loading mode toggle".
- Минифункция: "sequence badge".
- Минифункция: "ride amount summary".
- Минифункция: "item count summary".
- Минифункция: "status pill".

**Minifeaturecount:** 6

---

**Pageid:** ios-driver-profile

**Viewname:** ProfileView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/ProfileView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-driver-profile" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-driver-profile" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Driver identity, truck metadata, quick operations, and ride-history review surface.

#### Layoutoverview

- Зона макета: "driver title header".
- Зона макета: "driver identity card with status pill".
- Зона макета: "truck and metrics info grid".
- Зона макета: "quick actions row".
- Зона макета: "ride history ledger".
- Зона макета: "stats section".

#### Controloverview

- Элемент управления: "Sync quick action button".
- Элемент управления: "Logout quick action button".
- Элемент управления: "Offline Verifier quick action button or sheet trigger".

#### Iconoverview

- Иконографическая привязка: "driver initials avatar".
- Иконографическая привязка: "status pill".
- Иконографическая привязка: "quick action glyphs".

#### Flowoverview

**Summary:** Поток фиксируется как: "sync local and live route state".

---

**Summary:** Поток фиксируется как: "open offline verifier".

---

**Summary:** Поток фиксируется как: "logout driver session".

---


#### Dependencyoverview

##### Reads

- FleetViewModel driver profile and mission history
- TelemetryService state

##### Writes

- logout session
- offline verifier sheet state

##### Localizednotes

- Чтение: "FleetViewModel driver profile and mission history", "TelemetryService state".
- Запись: "logout session", "offline verifier sheet state".

#### Stateoverview

- Состояние: "on duty".
- Состояние: "idle".
- Состояние: "history populated".
- Состояние: "history sparse".

#### Figureoverview

- Фигура: "driver profile with quick actions and stats".
- Фигура: "identity card close-up".

#### Minifeatureoverview

- Минифункция: "identity card".
- Минифункция: "on-duty status pill".
- Минифункция: "truck info grid".
- Минифункция: "Sync action".
- Минифункция: "Logout action".
- Минифункция: "Offline Verifier access".
- Минифункция: "ride history list".
- Минифункция: "revenue stat".
- Минифункция: "completed-orders stat".

**Minifeaturecount:** 9

---

**Pageid:** ios-driver-offline-verifier

**Viewname:** OfflineVerifierView

**Surfacetype:** screen

**Sourcefile:** apps/driverappios/driverappios/Views/OfflineVerifierView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-driver-offline-verifier" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-driver-offline-verifier" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Cryptographic offline verification terminal for zero-connectivity proof of delivery and fraud detection.

#### Layoutoverview

- Зона макета: "terminal header with protocol name".
- Зона макета: "protocol status band".
- Зона макета: "state-driven body switching among idle, syncing, ready, scanning, verified, fraud, and error cards".

#### Controloverview

- Элемент управления: "Sync Route Manifest button in idle state".
- Элемент управления: "Start Scan button in ready state".

#### Iconoverview

- Иконографическая привязка: "scanner overlay in scanning state".
- Иконографическая привязка: "success or fraud glyphs in verification result cards".

#### Flowoverview

**Summary:** Поток фиксируется как: "sync manifest hash locally".

---

**Summary:** Поток фиксируется как: "start offline scan".

---

**Summary:** Поток фиксируется как: "verify QR against manifest hash".

---

**Summary:** Поток фиксируется как: "show verified order result or fraud reason".

---


#### Dependencyoverview

##### Reads

- OfflineDeliveryStore
- AVFoundation camera feed
- SHA256Helper manifest validation

##### Writes

- offline verification state machine

##### Localizednotes

- Чтение: "OfflineDeliveryStore", "AVFoundation camera feed", "SHA256Helper manifest validation".
- Запись: "offline verification state machine".

#### Stateoverview

- Состояние: "idle".
- Состояние: "syncing".
- Состояние: "ready".
- Состояние: "scanning".
- Состояние: "verified".
- Состояние: "fraud".
- Состояние: "error".

#### Figureoverview

- Фигура: "offline verification terminal in ready state".
- Фигура: "two-panel verified versus fraud outcome figure".

#### Minifeatureoverview

- Минифункция: "protocol status pill".
- Минифункция: "sync progress state".
- Минифункция: "manifest hash display".
- Минифункция: "Start Scan CTA".
- Минифункция: "scanner overlay".
- Минифункция: "verified result card".
- Минифункция: "fraud result card".
- Минифункция: "error result card".

**Minifeaturecount:** 8

---

**Pageid:** ios-driver-mission-detail-sheet

**Viewname:** MissionDetailSheet

**Surfacetype:** overlay

**Sourcefile:** apps/driverappios/driverappios/Views/MissionDetailSheet.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-driver-mission-detail-sheet" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-driver-mission-detail-sheet" представляет оверлей для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Bottom-sheet mission inspector exposing geofence clearance, endpoint distance, payment badge, and scan or correction actions.

#### Layoutoverview

- Зона макета: "order header with monospaced order ID and gateway badge".
- Зона макета: "delivery endpoint card with coordinates and geofence state".
- Зона макета: "distance and proximity indicators".
- Зона макета: "footer action cluster".

#### Controloverview

- Элемент управления: "Delivery Correction text button".
- Элемент управления: "Scan QR primary button".

#### Iconoverview

- Иконографическая привязка: "geofence status dot".
- Иконографическая привязка: "gateway badge".
- Иконографическая привязка: "location glyph in endpoint card".

#### Flowoverview

**Summary:** Поток фиксируется как: "present mission detail over map".

---

**Summary:** Поток фиксируется как: "launch correction from sheet".

---

**Summary:** Поток фиксируется как: "launch scanner from sheet".

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

- Чтение: "Mission object", "distance calculation", "geofence validation state".
- Запись: "scan callback", "correction callback".

#### Stateoverview

- Состояние: "cleared geofence".
- Состояние: "fault geofence".

#### Figureoverview

- Фигура: "mission detail sheet over map backdrop".
- Фигура: "endpoint card close-up with geofence state".

#### Minifeatureoverview

- Минифункция: "monospaced order ID".
- Минифункция: "gateway badge".
- Минифункция: "amount display".
- Минифункция: "geofence dot".
- Минифункция: "distance meter".
- Минифункция: "Delivery Correction CTA".
- Минифункция: "Scan QR CTA".

**Minifeaturecount:** 7

---

**Pageid:** ios-driver-map-marker-detail-sheet

**Viewname:** MapMarkerDetailSheet

**Surfacetype:** overlay

**Sourcefile:** apps/driverappios/driverappios/Views/Components/MapMarkerDetailSheet.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-driver-map-marker-detail-sheet" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-driver-map-marker-detail-sheet" представляет оверлей для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Compact marker drill-down for map stops, emphasizing stop identity and route semantics without entering the full mission sheet.

#### Layoutoverview

- Зона макета: "marker header with stop label".
- Зона макета: "order or stop metadata stack".
- Зона макета: "micro action row".

#### Controloverview

- Элемент управления: "compact dismiss or expand controls inside overlay".

#### Iconoverview

- Иконографическая привязка: "marker glyph".
- Иконографическая привязка: "status dot or stop-type icon".

#### Flowoverview

**Summary:** Поток фиксируется как: "tap map marker to open compact stop detail".

---

**Summary:** Поток фиксируется как: "dismiss or escalate into fuller mission inspection".

---


#### Dependencyoverview

##### Reads

- selected map marker payload

##### Writes

- marker-detail dismissal or expansion state

##### Localizednotes

- Чтение: "selected map marker payload".
- Запись: "marker-detail dismissal or expansion state".

#### Stateoverview

- Состояние: "compact marker summary".
- Состояние: "expanded marker context".

#### Figureoverview

- Фигура: "map marker detail overlay figure".

#### Minifeatureoverview

- Минифункция: "marker label".
- Минифункция: "stop metadata lines".
- Минифункция: "compact overlay".
- Минифункция: "expand affordance".

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

**Localizedsummary:** Локализованный обзор поверхности "payload-auth-loading" для роли payload-оператор на платформе React Native tablet.

## Localized

**Purpose:** Поверхность "payload-auth-loading" представляет страница для роли payload-оператор на платформе React Native tablet; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Payload terminal session-restore state shown while SecureStore token and worker context are being recovered at app startup.

### Layoutoverview

- Зона "restore-center" расположена в области "centered full-screen state". Содержимое: "restoring session text".

### Controloverview


### Iconoverview


### Flowoverview

**Flowid:** session-restore

**Summary:** Поток "session-restore" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "App locks screen to landscape".
- Шаг 2: "App reads payloader token, name, and supplier ID from SecureStore".
- Шаг 3: "App exits authLoading into login or authenticated state".

---


### Stateoverview

- Состояние: "restoring-session state".

### Figureoverview

- Фигура: "payload auth restore splash state".

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

**Localizedsummary:** Локализованный обзор поверхности "payload-dispatch-success" для роли payload-оператор на платформе React Native tablet.

## Localized

**Purpose:** Поверхность "payload-dispatch-success" представляет страница для роли payload-оператор на платформе React Native tablet; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Payload terminal dispatch-complete success state confirming manifest sealing and exposing dispatch codes before starting a new manifest.

### Layoutoverview

- Зона "success-center" расположена в области "centered body". Содержимое: "active truck mono label"; "Manifest Secured headline"; "Fleet Dispatched headline".
- Зона "dispatch-code-panel" расположена в области "center body below headlines". Правило видимости: "visible when dispatchCodes has entries". Содержимое: "Dispatch Codes heading"; "rows of order ID to code pairs".
- Зона "new-manifest-action" расположена в области "below code panel". Содержимое: "New Manifest outlined button".

### Controloverview

- Кнопка "New Manifest" расположена в "new-manifest-action". Стиль: "outlined button".

### Iconoverview


### Flowoverview

**Flowid:** dispatch-complete-reset

**Summary:** Поток "dispatch-complete-reset" содержит 5 шаг(а/ов).

#### Steps

- Шаг 1: "All loaded orders on the truck are sealed".
- Шаг 2: "App enters success state".
- Шаг 3: "Worker reviews dispatch codes if present".
- Шаг 4: "Worker taps New Manifest".
- Шаг 5: "App clears activeTruck, allSealed, and dispatchCodes, then returns to truck selection".

---


### Stateoverview

- Состояние: "success state without dispatch codes".
- Состояние: "success state with dispatch code panel".

### Figureoverview

- Фигура: "payload dispatch success state".
- Фигура: "dispatch code panel close-up".

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

**Localizedsummary:** Локализованный обзор поверхности "payload-login" для роли payload-оператор на платформе React Native tablet.

## Localized

**Purpose:** Поверхность "payload-login" представляет страница для роли payload-оператор на платформе React Native tablet; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Payload worker sign-in state for tablet authentication via phone number and 6-digit PIN.

### Layoutoverview

- Зона "brand-header" расположена в области "top centered column". Содержимое: "Pegasus Payload Terminal label"; "Payloader Login headline".
- Зона "credential-stack" расположена в области "centered form column". Содержимое: "phone number input"; "6-digit PIN input with centered wide letter spacing"; "Sign In button".

### Controloverview

- Кнопка "Sign In" расположена в "credential-stack footer". Стиль: "full-width filled button".

### Iconoverview


### Flowoverview

**Flowid:** payloader-login

**Summary:** Поток "payloader-login" содержит 5 шаг(а/ов).

#### Steps

- Шаг 1: "Worker enters phone number and PIN".
- Шаг 2: "Worker taps Sign In".
- Шаг 3: "App posts to /v1/auth/payloader/login".
- Шаг 4: "Successful response persists token, name, and supplier ID in SecureStore".
- Шаг 5: "App advances into truck selection state".

---


### Stateoverview

- Состояние: "idle login state".
- Состояние: "authenticating state".
- Состояние: "login failure alert".

### Figureoverview

- Фигура: "payload login state".
- Фигура: "payload login authenticating CTA state".

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

**Localizedsummary:** Локализованный обзор поверхности "payload-manifest-workspace" для роли payload-оператор на платформе React Native tablet.

## Localized

**Purpose:** Поверхность "payload-manifest-workspace" представляет страница для роли payload-оператор на платформе React Native tablet; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Warehouse payloader tablet workspace for selecting orders on a truck, checklist scanning line items, and sealing the load for dispatch.

### Layoutoverview

- Зона "left-pane" расположена в области "fixed-width left column". Содержимое: "terminal header with title and active truck"; "truck toggle bar"; "scrollable order list with active and cleared states".
- Зона "right-header" расположена в области "top of right pane". Содержимое: "selected order ID"; "retailer ID, payment gateway, amount text line"; "truck badge chip".
- Зона "checklist-region" расположена в области "center right pane". Содержимое: "scrollable manifest checklist"; "tap-to-toggle checkbox control"; "brand code line"; "item label line".
- Зона "seal-footer" расположена в области "bottom of right pane". Содержимое: "Mark as Loaded action button".

### Controloverview

- Кнопка "truck selector in left-pane toggle bar" расположена в "left-pane truck toggle row". Стиль: "segmented text button".
- Кнопка "order selector" расположена в "left-pane order list row". Стиль: "list row button". Правило видимости: "disabled for sealed orders".
- Кнопка "manifest item checkbox row" расположена в "checklist-region". Стиль: "full-row toggle".
- Кнопка "Mark as Loaded" расположена в "seal-footer". Стиль: "primary footer CTA". Правило видимости: "enabled only when all selected-order checklist items are checked and not currently sealing".

### Iconoverview

- Иконка "text-only checkmark glyph" используется в зоне "checkbox control when item is scanned".

### Flowoverview

**Flowid:** switch-active-truck

**Summary:** Поток "switch-active-truck" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Payload operator taps truck label in left-pane toggle row".
- Шаг 2: "handleTruckSelect resets local manifest state".
- Шаг 3: "fetchManifest reloads orders and checklist for that truck".

---

**Flowid:** select-order-and-clear-items

**Summary:** Поток "select-order-and-clear-items" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Payload operator taps order row in left pane".
- Шаг 2: "Right pane updates selected order header and checklist".
- Шаг 3: "Operator taps each checklist row to toggle scanned state".
- Шаг 4: "Checkbox fills accent color with checkmark when scanned".

---

**Flowid:** seal-order

**Summary:** Поток "seal-order" содержит 5 шаг(а/ов).

#### Steps

- Шаг 1: "All checklist items for selected order are scanned".
- Шаг 2: "Mark as Loaded becomes enabled".
- Шаг 3: "Operator taps Mark as Loaded".
- Шаг 4: "App posts to /v1/payload/seal".
- Шаг 5: "Order is added to sealedOrderIds and next remaining order is auto-selected or allSealed becomes true".

---


### Dependencyoverview

#### Reads

- /v1/payloader/trucks
- /v1/payloader/orders?vehicle_id={truckId}&state=LOADED

#### Writes

- /v1/payload/seal

#### Localizednotes

- Чтение: "/v1/payloader/trucks", "/v1/payloader/orders?vehicle_id={truckId}&state=LOADED".
- Запись: "/v1/payload/seal".
- Офлайн-фолбэк: "manifest fetch attempts SecureStore cache keyed by manifest_{truckId}".

### Stateoverview

- Состояние: "manifest loading in left pane".
- Состояние: "no pending orders in left pane".
- Состояние: "active order row styling".
- Состояние: "cleared order row styling".
- Состояние: "selected order right-pane detail".
- Состояние: "no selected order placeholder".
- Состояние: "sealing disabled footer".
- Состояние: "sealing in-progress footer".

### Figureoverview

- Фигура: "full two-pane manifest workspace".
- Фигура: "left-pane truck selector and order list".
- Фигура: "right-pane order header".
- Фигура: "checklist row with checkbox state".
- Фигура: "seal footer CTA".

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

**Localizedsummary:** Локализованный обзор поверхности "payload-truck-selection" для роли payload-оператор на платформе React Native tablet.

## Localized

**Purpose:** Поверхность "payload-truck-selection" представляет страница для роли payload-оператор на платформе React Native tablet; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Payload tablet vehicle-selection state for choosing the target truck before loading a manifest.

### Layoutoverview

- Зона "header-bar" расположена в области "top full-width". Содержимое: "terminal title"; "worker name"; "Sign Out action".
- Зона "selection-center" расположена в области "centered body". Содержимое: "Select Target Vehicle label"; "vehicle card row with label, license plate, and vehicle class"; "loading-or-empty helper text".

### Controloverview

- Кнопка "Sign Out" расположена в "header-bar right". Стиль: "text action".
- Кнопка "truck card" расположена в "selection-center". Стиль: "card button".

### Iconoverview


### Flowoverview

**Flowid:** truck-selection

**Summary:** Поток "truck-selection" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Authenticated worker waits for /v1/payloader/trucks to populate available vehicles".
- Шаг 2: "Worker taps a truck card".
- Шаг 3: "handleTruckSelect sets activeTruck and triggers manifest fetch".
- Шаг 4: "App transitions into manifest workspace".

---

**Flowid:** logout-from-selector

**Summary:** Поток "logout-from-selector" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Worker taps Sign Out".
- Шаг 2: "SecureStore credentials are cleared".
- Шаг 3: "App returns to payload login state".

---


### Stateoverview

- Состояние: "vehicle cards available".
- Состояние: "no vehicles available".
- Состояние: "loading vehicles helper text".

### Figureoverview

- Фигура: "payload truck selection state".
- Фигура: "payload truck card close-up".

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

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-active-deliveries" для роли ритейлер на платформе android.

## Localized

**Purpose:** Поверхность "android-retailer-active-deliveries" представляет страница для роли ритейлер на платформе android; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Android retailer active-deliveries bottom sheet listing in-progress orders with detail and QR actions.

### Layoutoverview

- Зона "sheet-header" расположена в области "top of modal bottom sheet". Содержимое: "Active Deliveries title"; "order count subtitle"; "Done action".
- Зона "delivery-card-list" расположена в области "sheet body". Содержимое: "active delivery cards with progress ring, order metadata, countdown row, and action buttons".

### Controloverview

- Кнопка "Done" расположена в "sheet-header trailing action". Стиль: "text button".
- Кнопка "Details" расположена в "delivery card action row". Стиль: "pill button".
- Кнопка "Show QR" расположена в "delivery card action row". Стиль: "primary pill". Правило видимости: "order has delivery token".

### Iconoverview

- Иконка "progress ring" используется в зоне "delivery card leading visual".
- Иконка "QrCode2" используется в зоне "Show QR action or awaiting-dispatch status pill".
- Иконка "CountdownTimer" используется в зоне "countdown row".

### Flowoverview

**Flowid:** delivery-sheet-review

**Summary:** Поток "delivery-sheet-review" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer opens active deliveries sheet".
- Шаг 2: "Retailer reviews each active-delivery card".
- Шаг 3: "Retailer opens details or QR flow from a selected card".
- Шаг 4: "Retailer dismisses sheet with Done".

---


### Stateoverview

- Состояние: "active deliveries sheet open".
- Состояние: "delivery card with QR enabled".
- Состояние: "delivery card awaiting dispatch".

### Figureoverview

- Фигура: "android retailer active deliveries sheet".
- Фигура: "delivery card with countdown and QR action".

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

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-auth" для роли ритейлер на платформе android.

## Localized

**Purpose:** Поверхность "android-retailer-auth" представляет страница для роли ритейлер на платформе android; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Android retailer authentication and registration screen with expandable onboarding fields, map and GPS location capture, and logistics profile collection.

### Layoutoverview

- Зона "brand-stack" расположена в области "top centered column". Содержимое: "black storefront icon disk"; "Pegasus title"; "Retailer Portal subtitle".
- Зона "credential-core" расположена в области "main form column". Содержимое: "phone field"; "password field".
- Зона "registration-extension" расположена в области "below core credentials when login mode is off". Правило видимости: "visible when isLoginMode is false". Содержимое: "store name field"; "owner name field"; "address field"; "Open Map button"; "Use GPS button"; "selected location label"; "tax ID field"; "receiving window fields"; "access type chip buttons"; "ceiling height field".
- Зона "primary-action-region" расположена в области "below form fields". Содержимое: "Sign In or Create Account button"; "error text when present"; "mode-toggle text button".

### Controloverview

- Кнопка "Open Map" расположена в "registration-extension location row". Стиль: "outlined button".
- Кнопка "Use GPS" расположена в "registration-extension location row". Стиль: "outlined button".
- Кнопка "Street" расположена в "registration-extension access row". Стиль: "outlined chip toggle".
- Кнопка "Alley" расположена в "registration-extension access row". Стиль: "outlined chip toggle".
- Кнопка "Dock" расположена в "registration-extension access row". Стиль: "outlined chip toggle".
- Кнопка "Sign In" расположена в "primary-action-region". Стиль: "full-width filled pill". Правило видимости: "isLoginMode true".
- Кнопка "Create Account" расположена в "primary-action-region". Стиль: "full-width filled pill". Правило видимости: "isLoginMode false".
- Кнопка "mode toggle" расположена в "primary-action-region footer". Стиль: "text button".

### Iconoverview

- Иконка "Storefront" используется в зоне "brand-stack".
- Иконка "Map" используется в зоне "Open Map button".
- Иконка "MyLocation" используется в зоне "Use GPS button".
- Иконка "CircularProgressIndicator" используется в зоне "Use GPS button when locating and primary CTA when state.isLoading".

### Flowoverview

**Flowid:** login-flow

**Summary:** Поток "login-flow" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer enters phone and password".
- Шаг 2: "Retailer taps Sign In".
- Шаг 3: "AuthViewModel authenticates and onAuthenticated advances into the main shell".

---

**Flowid:** registration-flow

**Summary:** Поток "registration-flow" содержит 5 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer switches out of login mode".
- Шаг 2: "AnimatedVisibility expands onboarding fields".
- Шаг 3: "Retailer captures location by map picker or GPS".
- Шаг 4: "Retailer submits Create Account".
- Шаг 5: "AuthViewModel sends registration payload with logistics fields".

---


### Stateoverview

- Состояние: "login mode".
- Состояние: "registration mode".
- Состояние: "GPS locating state".
- Состояние: "error text state".
- Состояние: "loading CTA state".
- Состояние: "map picker route handoff".

### Figureoverview

- Фигура: "android retailer login mode".
- Фигура: "android retailer registration mode".
- Фигура: "location capture row".
- Фигура: "loading and error state".

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

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-cart" для роли ритейлер на платформе android.

## Localized

**Purpose:** Поверхность "android-retailer-cart" представляет страница для роли ритейлер на платформе android; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Android retailer cart screen with list-based basket control, checkout sheet launch, supplier-closed guard dialog, and empty-cart branch.

### Layoutoverview

- Зона "cart-list-region" расположена в области "main list when items exist". Содержимое: "item count header"; "Clear All text button"; "cart item cards with placeholder image, size and pack pills, price, and quantity stepper".
- Зона "bottom-bar" расположена в области "sticky bottom bar". Правило видимости: "visible when cart is not empty". Содержимое: "subtotal row"; "delivery row"; "total cluster"; "Checkout surface button".
- Зона "checkout-sheet" расположена в области "modal bottom sheet". Правило видимости: "visible when uiState.showCheckout is true". Содержимое: "CheckoutSheet overlay".
- Зона "supplier-closed-dialog" расположена в области "alert dialog". Правило видимости: "visible when showSupplierClosedDialog is true". Содержимое: "warning title"; "supplier closed message"; "I Understand, Place Order button"; "Cancel button".
- Зона "empty-state" расположена в области "center body". Правило видимости: "visible when uiState.isEmpty is true". Содержимое: "double-ring shopping cart illustration"; "empty headline"; "helper copy"; "Browse Catalog button".

### Controloverview

- Кнопка "Clear All" расположена в "cart-list-region header-right". Стиль: "text destructive".
- Кнопка "quantity decrement" расположена в "each item stepper". Стиль: "icon button".
- Кнопка "quantity increment" расположена в "each item stepper". Стиль: "icon button".
- Кнопка "Checkout" расположена в "bottom-bar right". Стиль: "filled pill surface".
- Кнопка "I Understand, Place Order" расположена в "supplier-closed-dialog confirm action". Стиль: "filled button".
- Кнопка "Browse Catalog" расположена в "empty-state". Стиль: "filled pill surface".

### Iconoverview

- Иконка "Eco" используется в зоне "cart item placeholder".
- Иконка "Delete or Remove" используется в зоне "quantity decrement control".
- Иконка "Add" используется в зоне "quantity increment control".
- Иконка "ArrowForward" используется в зоне "Checkout CTA trailing icon".
- Иконка "ShoppingCart" используется в зоне "empty-state hero".
- Иконка "GridView" используется в зоне "Browse Catalog CTA".

### Flowoverview

**Flowid:** basket-editing

**Summary:** Поток "basket-editing" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer increments or decrements quantities from each item row".
- Шаг 2: "quantity, item total, and summary totals update".
- Шаг 3: "decrement icon changes to delete when quantity reaches one".

---

**Flowid:** checkout-gating

**Summary:** Поток "checkout-gating" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer taps Checkout".
- Шаг 2: "if supplier is closed, alert dialog interposes".
- Шаг 3: "otherwise CheckoutSheet opens immediately".

---


### Stateoverview

- Состояние: "populated cart state".
- Состояние: "checkout sheet active".
- Состояние: "supplier closed dialog".
- Состояние: "empty cart state".
- Состояние: "snackbar feedback state".

### Figureoverview

- Фигура: "android retailer cart populated state".
- Фигура: "cart bottom bar".
- Фигура: "supplier closed dialog".
- Фигура: "empty cart state".

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

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-catalog" для роли ритейлер на платформе android.

## Localized

**Purpose:** Поверхность "android-retailer-catalog" представляет страница для роли ритейлер на платформе android; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Android retailer catalog surface combining search-driven product discovery with a mixed-scale bento category browser.

### Layoutoverview

- Зона "search-field" расположена в области "top full-width". Содержимое: "outlined pill search field"; "Search icon".
- Зона "search-results-grid" расположена в области "main body when query length is at least two and results exist". Правило видимости: "visible when searchQuery length >= 2 and filteredProducts not empty". Содержимое: "two-column ProductCard grid".
- Зона "category-bento-list" расположена в области "main body when search branch inactive". Содержимое: "Categories header with count"; "rows of large, wide, compact, and remainder category cards".

### Controloverview

- Кнопка "category card" расположена в "category-bento-list". Стиль: "surface tap target".
- Кнопка "product card" расположена в "search-results-grid". Стиль: "card tap target".

### Iconoverview

- Иконка "Search" используется в зоне "search-field leading edge".
- Иконка "Inventory2 or category glyph" используется в зоне "category cards".

### Flowoverview

**Flowid:** category-navigation

**Summary:** Поток "category-navigation" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer taps a category bento card".
- Шаг 2: "Catalog routes to category-specific supplier or product inventory".

---

**Flowid:** search-navigation

**Summary:** Поток "search-navigation" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer types at least two characters".
- Шаг 2: "Catalog switches to filtered product grid".
- Шаг 3: "Retailer taps a product card".
- Шаг 4: "Catalog routes to product detail".

---


### Stateoverview

- Состояние: "category bento state".
- Состояние: "search results state".
- Состояние: "empty search branch fallback to categories".

### Figureoverview

- Фигура: "android retailer category bento layout".
- Фигура: "android retailer search results grid".

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

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-checkout-sheet" для роли ритейлер на платформе android.

## Localized

**Purpose:** Поверхность "android-retailer-checkout-sheet" представляет страница для роли ритейлер на платформе android; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Android retailer checkout bottom sheet for reviewing order totals, selecting payment gateway from a split buy control, and showing processing or completion phases.

### Layoutoverview

- Зона "sheet-header" расположена в области "top of modal bottom sheet". Содержимое: "Order details title"; "linear progress bar in review phase".
- Зона "review-phase" расположена в области "sheet body when phase is REVIEW". Содержимое: "product recap card with placeholder image"; "subtotal, shipping, discount, and total rows"; "Payment Method label"; "split Buy button with payment dropdown segment".
- Зона "processing-phase" расположена в области "sheet body when phase is PROCESSING". Содержимое: "CircularProgressIndicator"; "Processing payment text".
- Зона "complete-phase" расположена в области "sheet body when phase is COMPLETE". Содержимое: "check icon"; "Payment complete text".

### Controloverview

- Кнопка "Buy" расположена в "review-phase bottom control row". Стиль: "left segment of split CTA".
- Кнопка "payment dropdown segment" расположена в "review-phase bottom control row". Стиль: "right segment of split CTA".
- Кнопка "payment option" расположена в "dropdown menu". Стиль: "menu row".

### Iconoverview

- Иконка "Eco" используется в зоне "review-phase placeholder image".
- Иконка "Payment" используется в зоне "Buy segment".
- Иконка "KeyboardArrowDown" используется в зоне "dropdown segment".
- Иконка "Check" используется в зоне "complete phase and selected dropdown option".
- Иконка "CircularProgressIndicator" используется в зоне "processing phase".

### Flowoverview

**Flowid:** gateway-selection

**Summary:** Поток "gateway-selection" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer taps dropdown segment".
- Шаг 2: "DropdownMenu opens with payment options".
- Шаг 3: "Retailer selects gateway and label updates".

---

**Flowid:** buy-processing-complete

**Summary:** Поток "buy-processing-complete" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer taps Buy".
- Шаг 2: "Sheet phase transitions from REVIEW to PROCESSING".
- Шаг 3: "On success the sheet renders COMPLETE state".

---


### Stateoverview

- Состояние: "review phase".
- Состояние: "dropdown open state".
- Состояние: "processing phase".
- Состояние: "complete phase".

### Figureoverview

- Фигура: "android retailer checkout review sheet".
- Фигура: "split buy and payment dropdown control".
- Фигура: "processing sheet".
- Фигура: "complete sheet".

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

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-orders" для роли ритейлер на платформе android.

## Localized

**Purpose:** Поверхность "android-retailer-orders" представляет страница для роли ритейлер на платформе android; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Android retailer orders hub with tabbed pager content, pull-to-refresh, detail-sheet drilldown, and QR overlay access for live deliveries.

### Layoutoverview

- Зона "tab-row" расположена в области "top full-width". Содержимое: "Active tab with Inventory2 icon"; "Ordered tab with Receipt icon"; "AI Planned tab with AutoAwesome icon".
- Зона "pager-region" расположена в области "main body". Содержимое: "ActiveOrdersList"; "OrderedList"; "AiPlannedList".
- Зона "detail-sheet" расположена в области "overlay sheet". Правило видимости: "visible when selectedOrder is non-null". Содержимое: "OrderDetailSheet".
- Зона "qr-overlay" расположена в области "overlay". Правило видимости: "visible when qrOrder is non-null". Содержимое: "QROverlay".

### Controloverview

- Кнопка "tab" расположена в "tab-row". Стиль: "tab selector".
- Кнопка "Details" расположена в "active and ordered card action rows". Стиль: "pill button".
- Кнопка "Show QR" расположена в "active card action row". Стиль: "primary pill".
- Кнопка "Cancel" расположена в "ordered card action row". Стиль: "destructive pill".

### Iconoverview

- Иконка "Inventory2" используется в зоне "Active tab and empty state".
- Иконка "Receipt" используется в зоне "Ordered tab and empty state".
- Иконка "AutoAwesome" используется в зоне "AI Planned tab and empty state".
- Иконка "QrCode2" используется в зоне "Show QR action".

### Flowoverview

**Flowid:** tabbed-order-review

**Summary:** Поток "tabbed-order-review" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer switches between tabs in TabRow".
- Шаг 2: "HorizontalPager swaps the associated list view".

---

**Flowid:** order-drilldown-and-qr

**Summary:** Поток "order-drilldown-and-qr" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer opens OrderDetailSheet from a card".
- Шаг 2: "Retailer may open QROverlay for dispatch-ready orders".
- Шаг 3: "Pending orders can also be cancelled from Ordered cards or sheet actions".

---


### Stateoverview

- Состояние: "active list".
- Состояние: "ordered list".
- Состояние: "AI planned list".
- Состояние: "pull-to-refresh state".
- Состояние: "detail sheet open".
- Состояние: "QR overlay visible".
- Состояние: "empty lists".

### Figureoverview

- Фигура: "android retailer orders active tab".
- Фигура: "android retailer orders ordered tab".
- Фигура: "AI planned forecast card".
- Фигура: "orders QR overlay".

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

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-payment-sheet" для роли ритейлер на платформе android.

## Localized

**Purpose:** Поверхность "android-retailer-payment-sheet" представляет страница для роли ритейлер на платформе android; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Android retailer payment-required bottom sheet for choosing payment path after delivery, waiting for cash confirmation or card settlement, and resolving success or failure.

### Layoutoverview

- Зона "choose-phase" расположена в области "sheet body when phase is CHOOSE". Содержимое: "payments icon disk"; "amount due stack with optional struck original amount"; "cash option row"; "card gateway option rows".
- Зона "processing-phase" расположена в области "sheet body when phase is PROCESSING". Содержимое: "progress indicator"; "Processing headline"; "connection helper text".
- Зона "cash-pending-phase" расположена в области "sheet body when phase is CASH_PENDING". Содержимое: "cash icon disk"; "Cash Collection Pending headline"; "amount text"; "waiting chip with progress indicator".
- Зона "success-phase" расположена в области "sheet body when phase is SUCCESS". Содержимое: "success icon disk"; "Payment Complete headline"; "amount text"; "Done button".
- Зона "failed-phase" расположена в области "sheet body when phase is FAILED". Содержимое: "failure icon disk"; "Payment Failed headline"; "error message"; "Retry button"; "Cancel outlined button".

### Controloverview

- Кнопка "Cash on Delivery" расположена в "choose-phase". Стиль: "option row".
- Кнопка "card gateway option" расположена в "choose-phase". Стиль: "option row".
- Кнопка "Done" расположена в "success phase footer". Стиль: "full-width primary".
- Кнопка "Retry" расположена в "failed phase footer". Стиль: "full-width primary".
- Кнопка "Cancel" расположена в "failed phase footer". Стиль: "full-width outlined".

### Iconoverview

- Иконка "Payments" используется в зоне "choose-phase hero".
- Иконка "LocalAtm" используется в зоне "cash option and cash-pending hero".
- Иконка "CreditCard" используется в зоне "card option rows".
- Иконка "Check" используется в зоне "success hero".
- Иконка "Close" используется в зоне "failed hero".
- Иконка "CircularProgressIndicator" используется в зоне "processing and cash-pending states".

### Flowoverview

**Flowid:** cash-route

**Summary:** Поток "cash-route" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer chooses cash option".
- Шаг 2: "Sheet enters cash-pending state".
- Шаг 3: "Retailer waits for driver-side confirmation".

---

**Flowid:** card-route

**Summary:** Поток "card-route" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer chooses card gateway row".
- Шаг 2: "Sheet enters processing state".
- Шаг 3: "External or backend-driven payment settlement updates the phase to success or failed".

---

**Flowid:** failure-recovery

**Summary:** Поток "failure-recovery" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Sheet enters FAILED state".
- Шаг 2: "Retailer retries or dismisses".

---


### Stateoverview

- Состояние: "choose phase".
- Состояние: "processing phase".
- Состояние: "cash pending phase".
- Состояние: "success phase".
- Состояние: "failed phase".

### Figureoverview

- Фигура: "android retailer payment choose phase".
- Фигура: "android retailer payment cash pending phase".
- Фигура: "android retailer payment success phase".
- Фигура: "android retailer payment failed phase".

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

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-root-shell" для роли ритейлер на платформе android.

## Localized

**Purpose:** Поверхность "android-retailer-root-shell" представляет страница для роли ритейлер на платформе android; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Authenticated retailer shell that anchors primary navigation, active-order visibility, and global payment, QR, detail, and sidebar overlays.

### Layoutoverview

- Зона "top-bar" расположена в области "top full-width". Содержимое: "left: avatar circle button"; "center: Pegasus title"; "right: cart icon button with badge; notification icon button with badge".
- Зона "content-navhost" расположена в области "center full-width". Содержимое: "HOME dashboard"; "CATALOG"; "ORDERS"; "PROFILE"; "SUPPLIERS"; "CART"; "ANALYTICS"; "AUTO_ORDER"; "PRODUCT_DETAIL"; "CATEGORY_SUPPLIERS"; "SUPPLIER_CATEGORY_CATALOG".
- Зона "bottom-stack" расположена в области "bottom full-width". Содержимое: "FloatingActiveOrdersBar"; "LabBottomBar".
- Зона "global-overlays" расположена в области "above shell content". Содержимое: "ActiveDeliveriesSheet"; "OrderDetailSheet"; "QROverlay"; "SidebarMenu"; "DeliveryPaymentSheet".

### Controloverview

- Кнопка "avatar button" расположена в "top-bar-left". Стиль: "circular filled button".
- Кнопка "cart button" расположена в "top-bar-right". Стиль: "icon button with badge".
- Кнопка "notification button" расположена в "top-bar-right". Стиль: "icon button with badge".
- Кнопка "bottom nav tabs" расположена в "bottom-stack lower row". Стиль: "NavigationBarItem set of five".
- Кнопка "floating active orders bar" расположена в "bottom-stack upper row". Стиль: "full-width floating card CTA".
- Кнопка "sidebar menu rows" расположена в "sidebar overlay vertical list". Стиль: "full-width menu row buttons".

### Iconoverview

- Иконка "Outlined.ShoppingCart" используется в зоне "top-bar cart action".
- Иконка "Outlined.Notifications" используется в зоне "top-bar notification action".
- Иконка "Home/Store/Inventory2/AccountCircle/Person" используется в зоне "bottom nav".
- Иконка "LocalShipping" используется в зоне "floating active orders progress ring center".
- Иконка "KeyboardArrowUp" используется в зоне "floating active orders expand affordance".
- Иконка "GridView/BarChart/Insights/AutoAwesome/Inbox/Person/Settings/ExitToApp" используется в зоне "sidebar rows".

### Flowoverview

**Flowid:** global-order-attention

**Summary:** Поток "global-order-attention" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer sees active-order summary in floating bar".
- Шаг 2: "Retailer taps floating bar".
- Шаг 3: "ActiveDeliveriesSheet opens".
- Шаг 4: "Retailer can drill into detail or QR overlay".

---

**Flowid:** websocket-payment-resolution

**Summary:** Поток "websocket-payment-resolution" содержит 5 шаг(а/ов).

#### Steps

- Шаг 1: "NavigationViewModel receives PAYMENT_REQUIRED event".
- Шаг 2: "DeliveryPaymentSheet renders".
- Шаг 3: "Retailer chooses cash or card gateway".
- Шаг 4: "Card path deep-links to external payment app or cash path enters pending state".
- Шаг 5: "ORDER_COMPLETED or failure events drive final phase".

---

**Flowid:** sidebar-navigation

**Summary:** Поток "sidebar-navigation" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer taps avatar".
- Шаг 2: "SidebarMenu slides in from left".
- Шаг 3: "Retailer chooses dashboard, procurement, insights, auto-order, or AI predictions".
- Шаг 4: "Shell navigates or dismisses accordingly".

---


### Stateoverview

- Состояние: "base shell with no overlays".
- Состояние: "floating active orders visible".
- Состояние: "sidebar open with scrim".
- Состояние: "active deliveries sheet open".
- Состояние: "order detail sheet open".
- Состояние: "QR overlay open".
- Состояние: "payment sheet choose phase".
- Состояние: "payment sheet processing or failed or success phase".

### Figureoverview

- Фигура: "full retailer shell".
- Фигура: "top-bar control cluster".
- Фигура: "bottom-stack with floating active orders bar".
- Фигура: "sidebar overlay state".
- Фигура: "payment sheet state".

---

**Dossierfile:** retailer-android-secondary-surfaces.json

**Bundleid:** retailer-android-secondary-surfaces

**Appid:** retailer-app-android

**Platform:** android

**Role:** RETAILER

**Status:** implemented

**Entrytype:** bundle

**Localizedsummary:** Локализованный пакет "retailer-android-secondary-surfaces" охватывает 12 поверхностей приложения "retailer-app-android".

## Surfaces

**Pageid:** android-retailer-location-picker

**Navroute:** LocationPickerScreen

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/auth/LocationPickerScreen.kt

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-location-picker" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-retailer-location-picker" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Signup or profile location selector using a map-centered pin and confirm affordance.

#### Layoutoverview

- Зона макета: "top app bar".
- Зона макета: "map canvas".
- Зона макета: "center pin indicator".
- Зона макета: "address label".
- Зона макета: "Confirm Location footer".

#### Controloverview

- Элемент управления: "close or back control".
- Элемент управления: "Confirm Location button".

#### Iconoverview

- Иконографическая привязка: "mappin or location indicator at center".

#### Flowoverview

**Summary:** Поток фиксируется как: "pan map under fixed pin".

---

**Summary:** Поток фиксируется как: "reverse geocode displayed address".

---

**Summary:** Поток фиксируется как: "confirm selected coordinates".

---


#### Dependencyoverview

##### Reads

- map and geocoder state

##### Writes

- selected location result

##### Localizednotes

- Чтение: "map and geocoder state".
- Запись: "selected location result".

#### Stateoverview

- Состояние: "default map state".
- Состояние: "confirm-ready state".

#### Figureoverview

- Фигура: "android location picker with centered pin".

#### Minifeatureoverview

- Минифункция: "map pin".
- Минифункция: "address label".
- Минифункция: "confirm CTA".

**Minifeaturecount:** 3

---

**Pageid:** android-retailer-home

**Navroute:** HOME

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/dashboard/DashboardScreen.kt

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-home" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-retailer-home" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Retailer dashboard of service-entry tiles, reorder intelligence, and date-range driven spend snapshots.

#### Layoutoverview

- Зона макета: "service-tile grid".
- Зона макета: "pull-to-refresh scaffold".
- Зона макета: "reorder strip".
- Зона макета: "date-range buttons".
- Зона макета: "summary cards".

#### Controloverview

- Элемент управления: "service tiles as tap targets".
- Элемент управления: "date-range segmented buttons".

#### Iconoverview

- Иконографическая привязка: "service icons inside M3 cards".

#### Flowoverview

**Summary:** Поток фиксируется как: "tap into catalog, orders, procurement, inbox, insights".

---

**Summary:** Поток фиксируется как: "refresh dashboard".

---

**Summary:** Поток фиксируется как: "change analytics date range".

---


#### Dependencyoverview

##### Reads

- orders count
- reorder products
- AI demand forecasts

##### Writes


##### Localizednotes

- Чтение: "orders count", "reorder products", "AI demand forecasts".
- Запись: нет.

#### Stateoverview

- Состояние: "normal dashboard".
- Состояние: "refreshing".
- Состояние: "sparse data".

#### Figureoverview

- Фигура: "android retailer dashboard tile grid".

#### Minifeatureoverview

- Минифункция: "service tiles".
- Минифункция: "reorder strip".
- Минифункция: "date-range filter".
- Минифункция: "pull refresh".
- Минифункция: "summary cards".

**Minifeaturecount:** 5

---

**Pageid:** android-retailer-category-suppliers

**Navroute:** CATEGORY_SUPPLIERS/{categoryId}/{categoryName}

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/catalog/CategorySuppliersScreen.kt

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-category-suppliers" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-retailer-category-suppliers" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Category-scoped supplier browser for narrowing supplier selection before catalog exploration.

#### Layoutoverview

- Зона макета: "top app bar with category name".
- Зона макета: "supplier row list".
- Зона макета: "empty state".

#### Controloverview

- Элемент управления: "back button".
- Элемент управления: "supplier row tap target".

#### Iconoverview

- Иконографическая привязка: "supplier avatar initials".
- Иконографическая привязка: "chevron row affordance".

#### Flowoverview

**Summary:** Поток фиксируется как: "return to catalog".

---

**Summary:** Поток фиксируется как: "open selected supplier catalog".

---


#### Dependencyoverview

##### Reads

- suppliers by category

##### Writes


##### Localizednotes

- Чтение: "suppliers by category".
- Запись: нет.

#### Stateoverview

- Состояние: "list populated".
- Состояние: "empty".

#### Figureoverview

- Фигура: "category suppliers list".

#### Minifeatureoverview

- Минифункция: "back nav".
- Минифункция: "supplier rows".
- Минифункция: "status badge".
- Минифункция: "empty state".

**Minifeaturecount:** 4

---

**Pageid:** android-retailer-supplier-catalog

**Navroute:** SUPPLIER_CATEGORY_CATALOG/{supplierId}/{supplierName}/{supplierCategory}/{supplierIsActive}

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/suppliers/SupplierCatalogScreen.kt

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-supplier-catalog" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-retailer-supplier-catalog" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier-specific catalog grouped by product category with top-bar supplier identity and availability badge.

#### Layoutoverview

- Зона макета: "top app bar with supplier name and category".
- Зона макета: "OPEN or CLOSED badge".
- Зона макета: "grouped product list".
- Зона макета: "category headers".

#### Controloverview

- Элемент управления: "back button".
- Элемент управления: "product row tap target".

#### Iconoverview

- Иконографическая привязка: "availability status dot".
- Иконографическая привязка: "back icon".

#### Flowoverview

**Summary:** Поток фиксируется как: "return to supplier list".

---

**Summary:** Поток фиксируется как: "open product detail from grouped list".

---


#### Dependencyoverview

##### Reads

- supplier products grouped by category

##### Writes


##### Localizednotes

- Чтение: "supplier products grouped by category".
- Запись: нет.

#### Stateoverview

- Состояние: "supplier open".
- Состояние: "supplier closed".
- Состояние: "empty catalog".

#### Figureoverview

- Фигура: "supplier catalog grouped by category".

#### Minifeatureoverview

- Минифункция: "supplier title".
- Минифункция: "category subtitle".
- Минифункция: "availability badge".
- Минифункция: "grouped list".

**Minifeaturecount:** 4

---

**Pageid:** android-retailer-product-detail

**Navroute:** PRODUCT_DETAIL/{productId}

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/product/ProductDetailScreen.kt

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-product-detail" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-retailer-product-detail" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Retailer product inspector with variant choice, quantity control, per-variant auto-order toggle, and fixed add-to-cart bar.

#### Layoutoverview

- Зона макета: "hero image region".
- Зона макета: "product info section".
- Зона макета: "variant selector".
- Зона макета: "quantity stepper".
- Зона макета: "nutrition or metadata section".
- Зона макета: "fixed bottom Add to Cart bar".

#### Controloverview

- Элемент управления: "variant chips".
- Элемент управления: "quantity plus and minus controls".
- Элемент управления: "auto-order toggle".
- Элемент управления: "Add to Cart button".

#### Iconoverview

- Иконографическая привязка: "placeholder leaf icon".
- Иконографическая привязка: "toggle or snackbar icons".

#### Flowoverview

**Summary:** Поток фиксируется как: "switch variants".

---

**Summary:** Поток фиксируется как: "adjust quantity".

---

**Summary:** Поток фиксируется как: "toggle auto-order with history or fresh dialog".

---

**Summary:** Поток фиксируется как: "add selected configuration to cart".

---


#### Dependencyoverview

##### Reads

- product detail payload
- variant auto-order settings

##### Writes

- cart mutation
- auto-order settings update

##### Localizednotes

- Чтение: "product detail payload", "variant auto-order settings".
- Запись: "cart mutation", "auto-order settings update".

#### Stateoverview

- Состояние: "product loaded".
- Состояние: "placeholder image".
- Состояние: "auto-order dialog open".

#### Figureoverview

- Фигура: "product detail screen with bottom add-to-cart bar".

#### Minifeatureoverview

- Минифункция: "hero image".
- Минифункция: "variant chips".
- Минифункция: "quantity stepper".
- Минифункция: "auto-order toggle".
- Минифункция: "Add to Cart bar".
- Минифункция: "history or fresh dialog".

**Minifeaturecount:** 6

---

**Pageid:** android-retailer-analytics

**Navroute:** ANALYTICS

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/analytics/AnalyticsScreen.kt

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-analytics" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-retailer-analytics" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Retailer expense and supplier-spend analytics dashboard with date-range filters and charts.

#### Layoutoverview

- Зона макета: "date-range chip row".
- Зона макета: "KPI cards".
- Зона макета: "expense line chart".
- Зона макета: "top suppliers chart".
- Зона макета: "top products table".

#### Controloverview

- Элемент управления: "date-range filter chips".

#### Iconoverview

- Иконографическая привязка: "chart glyphs inside KPI cards".

#### Flowoverview

**Summary:** Поток фиксируется как: "change date range".

---

**Summary:** Поток фиксируется как: "refresh analytics dataset".

---


#### Dependencyoverview

##### Reads

- retailer analytics endpoint

##### Writes


##### Localizednotes

- Чтение: "retailer analytics endpoint".
- Запись: нет.

#### Stateoverview

- Состояние: "chart populated".
- Состояние: "empty analytics".
- Состояние: "refreshing".

#### Figureoverview

- Фигура: "analytics screen with range chips and charts".

#### Minifeatureoverview

- Минифункция: "range chips".
- Минифункция: "KPI cards".
- Минифункция: "line chart".
- Минифункция: "supplier chart".
- Минифункция: "products table".

**Minifeaturecount:** 5

---

**Pageid:** android-retailer-auto-order

**Navroute:** AUTO_ORDER

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/autoorder/AutoOrderScreen.kt

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-auto-order" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-retailer-auto-order" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Hierarchy-based auto-order settings page covering global, supplier, category, and product enablement.

#### Layoutoverview

- Зона макета: "sparkles header card".
- Зона макета: "settings list".
- Зона макета: "toggle rows".
- Зона макета: "confirmation dialog".

#### Controloverview

- Элемент управления: "toggle switches for each scope".
- Элемент управления: "Use History action".
- Элемент управления: "Start Fresh action".
- Элемент управления: "Cancel action".

#### Iconoverview

- Иконографическая привязка: "sparkles icon".
- Иконографическая привязка: "scope icons in rows".

#### Flowoverview

**Summary:** Поток фиксируется как: "toggle auto-order at any scope".

---

**Summary:** Поток фиксируется как: "open dialog when enabling".

---

**Summary:** Поток фиксируется как: "persist selection".

---


#### Dependencyoverview

##### Reads

- auto-order settings

##### Writes

- auto-order enable or disable endpoints

##### Localizednotes

- Чтение: "auto-order settings".
- Запись: "auto-order enable or disable endpoints".

#### Stateoverview

- Состояние: "all disabled".
- Состояние: "mixed enabled".
- Состояние: "enable dialog open".

#### Figureoverview

- Фигура: "auto-order settings hierarchy screen".

#### Minifeatureoverview

- Минифункция: "global toggle".
- Минифункция: "supplier toggle".
- Минифункция: "category toggle".
- Минифункция: "product toggle".
- Минифункция: "enable dialog".

**Minifeaturecount:** 5

---

**Pageid:** android-retailer-profile

**Navroute:** PROFILE

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/profile/ProfileScreen.kt

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-profile" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-retailer-profile" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Retailer profile, support, company settings, and global auto-order governance surface.

#### Layoutoverview

- Зона макета: "profile header card".
- Зона макета: "stats row".
- Зона макета: "auto-order card".
- Зона макета: "settings sections".
- Зона макета: "support rows".

#### Controloverview

- Элемент управления: "global auto-order switch".
- Элемент управления: "settings row tap targets".
- Элемент управления: "logout action".

#### Iconoverview

- Иконографическая привязка: "avatar initials".
- Иконографическая привязка: "settings row icons".

#### Flowoverview

**Summary:** Поток фиксируется как: "toggle global auto-order".

---

**Summary:** Поток фиксируется как: "open history or fresh dialog".

---

**Summary:** Поток фиксируется как: "navigate into settings items".

---

**Summary:** Поток фиксируется как: "logout".

---


#### Dependencyoverview

##### Reads

- retailer profile endpoint

##### Writes

- global auto-order endpoint

##### Localizednotes

- Чтение: "retailer profile endpoint".
- Запись: "global auto-order endpoint".

#### Stateoverview

- Состояние: "normal profile".
- Состояние: "dialog open".

#### Figureoverview

- Фигура: "profile screen with auto-order card".

#### Minifeatureoverview

- Минифункция: "profile card".
- Минифункция: "status pill".
- Минифункция: "stats row".
- Минифункция: "global auto-order toggle".
- Минифункция: "settings rows".
- Минифункция: "logout".

**Minifeaturecount:** 6

---

**Pageid:** android-retailer-suppliers

**Navroute:** SUPPLIERS

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/screens/suppliers/MySuppliersScreen.kt

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-suppliers" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-retailer-suppliers" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Favorite suppliers grid with pull-to-refresh and retry-capable empty or error fallback.

#### Layoutoverview

- Зона макета: "supplier card grid".
- Зона макета: "pull refresh scaffold".
- Зона макета: "empty or retry state".

#### Controloverview

- Элемент управления: "supplier card tap target".
- Элемент управления: "Retry button in fallback state".

#### Iconoverview

- Иконографическая привязка: "supplier initials tile".
- Иконографическая привязка: "empty-state icon".

#### Flowoverview

**Summary:** Поток фиксируется как: "refresh supplier list".

---

**Summary:** Поток фиксируется как: "open supplier catalog from card".

---

**Summary:** Поток фиксируется как: "retry after failure".

---


#### Dependencyoverview

##### Reads

- favorite suppliers endpoint

##### Writes


##### Localizednotes

- Чтение: "favorite suppliers endpoint".
- Запись: нет.

#### Stateoverview

- Состояние: "grid populated".
- Состояние: "empty".
- Состояние: "error".

#### Figureoverview

- Фигура: "favorite suppliers grid".

#### Minifeatureoverview

- Минифункция: "grid view".
- Минифункция: "pull refresh".
- Минифункция: "retry action".
- Минифункция: "order count badge".
- Минифункция: "auto-order badge".

**Minifeaturecount:** 5

---

**Pageid:** android-retailer-order-detail-sheet

**Navroute:** OrderDetailSheet

**Surfacetype:** overlay

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/OrderDetailSheet.kt

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-order-detail-sheet" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-retailer-order-detail-sheet" представляет оверлей для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Bottom-sheet order drill-down showing status, items, amounts, and terminal actions tied to an active order.

#### Layoutoverview

- Зона макета: "sheet header".
- Зона макета: "order metadata section".
- Зона макета: "line-item list".
- Зона макета: "footer action row".

#### Controloverview

- Элемент управления: "sheet close affordance".
- Элемент управления: "status-specific footer actions".

#### Iconoverview

- Иконографическая привязка: "status icon or ring".

#### Flowoverview

**Summary:** Поток фиксируется как: "open from order card".

---

**Summary:** Поток фиксируется как: "inspect line items".

---

**Summary:** Поток фиксируется как: "launch payment or QR step when relevant".

---


#### Dependencyoverview

##### Reads

- selected order payload

##### Writes

- order action callbacks

##### Localizednotes

- Чтение: "selected order payload".
- Запись: "order action callbacks".

#### Stateoverview

- Состояние: "standard order review".
- Состояние: "actionable order state".

#### Figureoverview

- Фигура: "order detail bottom sheet".

#### Minifeatureoverview

- Минифункция: "sheet header".
- Минифункция: "item list".
- Минифункция: "footer actions".
- Минифункция: "status indicator".

**Minifeaturecount:** 4

---

**Pageid:** android-retailer-qr-overlay

**Navroute:** QROverlay

**Surfacetype:** overlay

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/QROverlay.kt

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-qr-overlay" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-retailer-qr-overlay" представляет оверлей для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Retailer QR verification overlay for delivery acceptance and handoff confirmation.

#### Layoutoverview

- Зона макета: "camera or QR display frame".
- Зона макета: "instruction label".
- Зона макета: "dismiss region".

#### Controloverview

- Элемент управления: "dismiss control".
- Элемент управления: "confirm or continue action if present".

#### Iconoverview

- Иконографическая привязка: "QR framing corners".

#### Flowoverview

**Summary:** Поток фиксируется как: "show QR token for driver scan or scan driver token depending on state".

---

**Summary:** Поток фиксируется как: "dismiss overlay".

---


#### Dependencyoverview

##### Reads

- QR payload state

##### Writes

- verification callback

##### Localizednotes

- Чтение: "QR payload state".
- Запись: "verification callback".

#### Stateoverview

- Состояние: "display mode".
- Состояние: "scan mode".

#### Figureoverview

- Фигура: "QR overlay figure".

#### Minifeatureoverview

- Минифункция: "QR frame".
- Минифункция: "instruction text".
- Минифункция: "dismiss control".

**Minifeaturecount:** 3

---

**Pageid:** android-retailer-sidebar-menu

**Navroute:** SidebarMenu

**Surfacetype:** overlay

**Sourcefile:** apps/retailer-app-android/app/src/main/java/com/pegasus/retailer/ui/components/SidebarMenu.kt

**Localizedsummary:** Локализованный обзор поверхности "android-retailer-sidebar-menu" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "android-retailer-sidebar-menu" представляет оверлей для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Drawer-style navigation overlay exposing secondary retailer destinations and profile context.

#### Layoutoverview

- Зона макета: "avatar header".
- Зона макета: "menu rows".
- Зона макета: "footer utility area".

#### Controloverview

- Элемент управления: "menu row tap targets".
- Элемент управления: "dismiss touch scrim".

#### Iconoverview

- Иконографическая привязка: "row icons".
- Иконографическая привязка: "avatar initials".

#### Flowoverview

**Summary:** Поток фиксируется как: "open from shell menu trigger".

---

**Summary:** Поток фиксируется как: "navigate to selected destination".

---

**Summary:** Поток фиксируется как: "dismiss drawer".

---


#### Dependencyoverview

##### Reads

- retailer identity summary

##### Writes

- navigation state

##### Localizednotes

- Чтение: "retailer identity summary".
- Запись: "navigation state".

#### Stateoverview

- Состояние: "open drawer".

#### Figureoverview

- Фигура: "sidebar drawer overlay".

#### Minifeatureoverview

- Минифункция: "avatar header".
- Минифункция: "menu rows".
- Минифункция: "icon stack".
- Минифункция: "dismiss scrim".

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

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-active-deliveries" для роли ритейлер на платформе iOS.

## Localized

**Purpose:** Поверхность "ios-retailer-active-deliveries" представляет страница для роли ритейлер на платформе iOS; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Retailer active-delivery monitor showing only live orders, detail-sheet drilldown, and QR handoff from a dedicated delivery surface.

### Layoutoverview

- Зона "delivery-scroll-region" расположена в области "main body". Содержимое: "active delivery card list".
- Зона "empty-state" расположена в области "center body". Правило видимости: "visible when orders list is empty". Содержимое: "shippingbox icon disk"; "No Active Orders headline"; "helper copy".
- Зона "detail-sheet" расположена в области "bottom sheet". Правило видимости: "visible when selectedOrder is non-null". Содержимое: "OrderDetailSheet at 75 percent height".
- Зона "qr-overlay" расположена в области "full-screen overlay". Правило видимости: "visible when qrOverlayOrder is non-null and status has delivery token". Содержимое: "QROverlay".

### Controloverview

- Кнопка "Details" расположена в "delivery card action row". Стиль: "neutral pill button".
- Кнопка "Show QR" расположена в "delivery card action row". Стиль: "accent pill button". Правило видимости: "order has delivery token".
- Кнопка "Awaiting Dispatch" расположена в "delivery card action row". Стиль: "disabled status pill". Правило видимости: "order lacks delivery token".

### Iconoverview

- Иконка "shippingbox or qrcode or clock" используется в зоне "delivery cards and empty state".

### Flowoverview

**Flowid:** active-delivery-review

**Summary:** Поток "active-delivery-review" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "View loads retailer orders and filters them to active statuses".
- Шаг 2: "Retailer taps Details for a selected order".
- Шаг 3: "Retailer may alternatively tap Show QR for token-enabled orders".

---


### Stateoverview

- Состояние: "loading state".
- Состояние: "empty state".
- Состояние: "active deliveries list".
- Состояние: "detail sheet open".
- Состояние: "QR overlay open".

### Figureoverview

- Фигура: "retailer iOS active deliveries list".
- Фигура: "delivery card close-up".
- Фигура: "active deliveries QR overlay".

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

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-cart" для роли ритейлер на платформе iOS.

## Localized

**Purpose:** Поверхность "ios-retailer-cart" представляет страница для роли ритейлер на платформе iOS; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Retailer basket-management screen with item-level quantity control, destructive removal, summary footer, and full-screen checkout handoff.

### Layoutoverview

- Зона "cart-list-region" расположена в области "scroll body when cart has items". Содержимое: "cart count header"; "Clear All button"; "cart item cards with image placeholder, product metadata, total price, quantity stepper, and delete affordance".
- Зона "bottom-bar" расположена в области "sticky bottom summary bar". Правило видимости: "visible when cart is not empty". Содержимое: "subtotal row"; "delivery row"; "total cluster"; "Checkout pill button".
- Зона "empty-state" расположена в области "center body". Правило видимости: "visible when cart.isEmpty is true". Содержимое: "double-ring cart illustration"; "empty headline"; "helper copy"; "Browse Catalog button".
- Зона "checkout-cover" расположена в области "full-screen cover". Правило видимости: "visible when showCheckout is true". Содержимое: "CheckoutView full-screen modal".

### Controloverview

- Кнопка "Clear All" расположена в "cart-list-region header-right". Стиль: "text destructive".
- Кнопка "quantity stepper decrement" расположена в "each cart item card". Стиль: "stepper button".
- Кнопка "quantity stepper increment" расположена в "each cart item card". Стиль: "stepper button".
- Кнопка "delete" расположена в "cart item trailing overlay". Стиль: "small destructive icon button".
- Кнопка "Checkout" расположена в "bottom-bar right". Стиль: "accent pill CTA".
- Кнопка "Browse Catalog" расположена в "empty-state". Стиль: "accent pill CTA".

### Iconoverview

- Иконка "leaf.fill" используется в зоне "cart item image placeholder".
- Иконка "trash" используется в зоне "cart item delete overlay and swipe action".
- Иконка "arrow.right" используется в зоне "Checkout CTA trailing edge".
- Иконка "square.grid.2x2" используется в зоне "Browse Catalog button".
- Иконка "cart" используется в зоне "empty-state illustration".

### Flowoverview

**Flowid:** quantity-adjustment

**Summary:** Поток "quantity-adjustment" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer uses quantity stepper on a cart item".
- Шаг 2: "CartManager updates item quantity".
- Шаг 3: "line totals and bottom summary recompute".

---

**Flowid:** item-removal

**Summary:** Поток "item-removal" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer taps trailing delete control or swipe action".
- Шаг 2: "CartManager removes item with animated transition".

---

**Flowid:** checkout-handoff

**Summary:** Поток "checkout-handoff" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer taps Checkout".
- Шаг 2: "CartView presents CheckoutView as a full-screen cover".

---


### Stateoverview

- Состояние: "populated cart state".
- Состояние: "empty cart state".
- Состояние: "quantity update state".
- Состояние: "full-screen checkout cover active".

### Figureoverview

- Фигура: "retailer iOS cart populated state".
- Фигура: "cart item row with quantity stepper and delete control".
- Фигура: "cart bottom summary bar".
- Фигура: "empty cart state".

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

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-catalog" для роли ритейлер на платформе iOS.

## Localized

**Purpose:** Поверхность "ios-retailer-catalog" представляет страница для роли ритейлер на платформе iOS; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Retailer category-browse and product-search screen using a bento-grid catalog overview and product-grid search results.

### Layoutoverview

- Зона "search-bar" расположена в области "top full-width". Содержимое: "magnifying glass icon"; "search text field"; "clear-search button when text is non-empty".
- Зона "category-browse-region" расположена в области "scroll body when search is empty". Правило видимости: "visible when searchText is empty". Содержимое: "Categories header row with count"; "mixed-size bento category cards"; "remaining categories two-column grid".
- Зона "search-results-grid" расположена в области "scroll body when search has results". Правило видимости: "visible when searchText is non-empty and filteredProducts is non-empty". Содержимое: "two-column ProductCardView grid".
- Зона "no-results-state" расположена в области "center body". Правило видимости: "visible when searchText is non-empty and filteredProducts is empty". Содержимое: "search icon disk"; "No Results headline"; "query-specific helper text".

### Controloverview

- Кнопка "clear search" расположена в "search-bar trailing edge". Стиль: "icon button".
- Кнопка "bento category card" расположена в "category-browse-region". Стиль: "card tap target".
- Кнопка "product card" расположена в "search-results-grid". Стиль: "card tap target".

### Iconoverview

- Иконка "magnifyingglass" используется в зоне "search-bar leading edge".
- Иконка "xmark.circle.fill" используется в зоне "search-bar trailing clear button".
- Иконка "category icon glyph" используется в зоне "bento cards".
- Иконка "magnifyingglass" используется в зоне "no-results-state hero icon".

### Flowoverview

**Flowid:** category-browse

**Summary:** Поток "category-browse" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer lands on category overview".
- Шаг 2: "Retailer taps a bento category card".
- Шаг 3: "Navigation pushes CategorySuppliersView".

---

**Flowid:** product-search

**Summary:** Поток "product-search" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer types into search bar".
- Шаг 2: "Catalog switches from category bento layout to filtered product grid".
- Шаг 3: "Retailer taps a product card".
- Шаг 4: "Navigation pushes ProductDetailView".

---


### Stateoverview

- Состояние: "loading skeleton grid".
- Состояние: "category bento state".
- Состояние: "search results state".
- Состояние: "no-results state".
- Состояние: "failed-load alert".

### Figureoverview

- Фигура: "retailer iOS catalog bento grid".
- Фигура: "search bar close-up".
- Фигура: "product search results grid".
- Фигура: "no-results state".

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

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-checkout" для роли ритейлер на платформе iOS.

## Localized

**Purpose:** Поверхность "ios-retailer-checkout" представляет страница для роли ритейлер на платформе iOS; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Retailer order-finalization screen with cart recap, payment-method selection, supplier-closed confirmation, offline retry fallback, and success state.

### Layoutoverview

- Зона "toolbar" расположена в области "top navigation bar". Содержимое: "Checkout title"; "circular xmark dismiss button".
- Зона "checkout-scroll-stack" расположена в области "main scroll body while showSuccess is false". Содержимое: "Cart card with line items and quantity steppers"; "Payment card with change button"; "Summary card with subtotal, delivery, and total".
- Зона "submit-bar" расположена в области "bottom sticky region". Правило видимости: "visible when showSuccess is false". Содержимое: "Place Order button".
- Зона "payment-picker-sheet" расположена в области "modal sheet". Правило видимости: "visible when showPaymentPicker is true". Содержимое: "payment method list rows"; "selected checkmark".
- Зона "success-state" расположена в области "full-screen success replacement". Правило видимости: "visible when showSuccess is true". Содержимое: "success icon cluster"; "Order Placed headline"; "supporting copy"; "Done button".

### Controloverview

- Кнопка "dismiss" расположена в "toolbar trailing edge". Стиль: "circular icon button".
- Кнопка "Change" расположена в "payment card trailing edge". Стиль: "text button".
- Кнопка "Place Order" расположена в "submit-bar". Стиль: "full-width primary".
- Кнопка "payment method row" расположена в "payment-picker-sheet". Стиль: "list row".
- Кнопка "I Understand, Place Order" расположена в "supplier-closed confirmation dialog". Стиль: "confirm action".
- Кнопка "Done" расположена в "success-state footer". Стиль: "full-width primary".

### Iconoverview

- Иконка "xmark" используется в зоне "toolbar dismiss control".
- Иконка "cart.fill" используется в зоне "cart section header".
- Иконка "creditcard.fill" используется в зоне "payment section header".
- Иконка "creditcard or wallet.pass or banknote" используется в зоне "payment picker rows".
- Иконка "checkmark.circle.fill" используется в зоне "success-state hero".

### Flowoverview

**Flowid:** payment-method-selection

**Summary:** Поток "payment-method-selection" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer taps Change in payment card".
- Шаг 2: "Payment picker sheet opens".
- Шаг 3: "Retailer selects Click, Payme, Global Pay, or Cash on Delivery".
- Шаг 4: "selectedPayment updates and sheet dismisses".

---

**Flowid:** order-submission

**Summary:** Поток "order-submission" содержит 5 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer taps Place Order".
- Шаг 2: "If supplier is inactive, confirmation dialog interposes".
- Шаг 3: "Checkout posts to /v1/checkout/unified with gateway-mapped code".
- Шаг 4: "Success clears cart and shows success state".
- Шаг 5: "Failure stores PendingOrder for retry and shows alert".

---


### Stateoverview

- Состояние: "review state".
- Состояние: "payment picker sheet".
- Состояние: "supplier closed confirmation dialog".
- Состояние: "error alert".
- Состояние: "submitting state".
- Состояние: "success replacement state".

### Figureoverview

- Фигура: "retailer iOS checkout review state".
- Фигура: "payment picker sheet".
- Фигура: "supplier closed confirmation dialog".
- Фигура: "checkout success state".

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

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-payment-sheet" для роли ритейлер на платформе iOS.

## Localized

**Purpose:** Поверхность "ios-retailer-payment-sheet" представляет страница для роли ритейлер на платформе iOS; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Retailer payment-required overlay for choosing cash or card gateways after offload, waiting for settlement, and confirming successful completion.

### Layoutoverview

- Зона "phase-container" расположена в области "sheet body". Содержимое: "choose content"; "processing content"; "cash pending content"; "success content"; "failed content".
- Зона "choose-phase" расположена в области "sheet body when phase is choose". Правило видимости: "visible when phase equals choose". Содержимое: "warning icon disk"; "amount due stack with optional struck original amount"; "payment method choice list"; "cash option button"; "one or more card gateway option buttons".
- Зона "processing-phase" расположена в области "sheet body when phase is processing". Содержимое: "ProgressView"; "Processing headline"; "connecting helper text".
- Зона "cash-pending-phase" расположена в области "sheet body when phase is cashPending". Содержимое: "banknote icon disk"; "Cash Collection Pending headline"; "amount text"; "waiting pill with progress indicator".
- Зона "success-or-failure-phase" расположена в области "sheet body when phase is success or failed". Содержимое: "success checkmark or failure xmark disk"; "result headline"; "amount or error message"; "Done or Retry and Cancel actions".

### Controloverview

- Кнопка "Close" расположена в "navigation bar cancellation action". Стиль: "text button". Правило видимости: "phase is choose or failed".
- Кнопка "Cash on Delivery option" расположена в "choose-phase". Стиль: "full-width option row".
- Кнопка "card gateway option" расположена в "choose-phase". Стиль: "full-width option row".
- Кнопка "Done" расположена в "success phase footer". Стиль: "full-width primary".
- Кнопка "Try Again" расположена в "failed phase footer". Стиль: "full-width primary".
- Кнопка "Cancel" расположена в "failed phase footer". Стиль: "text button".

### Iconoverview

- Иконка "banknote.fill" используется в зоне "choose and cash-pending hero disks".
- Иконка "creditcard.fill" используется в зоне "card gateway option rows".
- Иконка "checkmark.circle.fill" используется в зоне "success phase hero".
- Иконка "xmark.circle.fill" используется в зоне "failed phase hero".
- Иконка "ProgressView" используется в зоне "processing and cash-pending phases".
- Иконка "chevron.right" используется в зоне "payment option rows".

### Flowoverview

**Flowid:** cash-selection

**Summary:** Поток "cash-selection" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer chooses Cash on Delivery".
- Шаг 2: "Sheet enters processing then cashPending phase".
- Шаг 3: "Sheet listens for driver confirmation or websocket completion".

---

**Flowid:** card-selection

**Summary:** Поток "card-selection" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer chooses Click, Payme, or Global Pay".
- Шаг 2: "Sheet posts checkout request".
- Шаг 3: "External payment URL opens when available".
- Шаг 4: "Sheet remains in processing until paymentSettled or orderCompleted websocket event arrives".

---

**Flowid:** failure-retry

**Summary:** Поток "failure-retry" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Payment attempt fails".
- Шаг 2: "Sheet renders failed phase".
- Шаг 3: "Retailer retries or cancels".

---


### Stateoverview

- Состояние: "choose phase".
- Состояние: "processing phase".
- Состояние: "cash pending phase".
- Состояние: "success phase".
- Состояние: "failed phase".

### Figureoverview

- Фигура: "retailer iOS payment choose phase".
- Фигура: "cash pending phase".
- Фигура: "processing phase".
- Фигура: "success and failure states".

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

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-login" для роли ритейлер на платформе iOS.

## Localized

**Purpose:** Поверхность "ios-retailer-login" представляет страница для роли ритейлер на платформе iOS; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Retailer authentication and registration screen combining login, store onboarding, map-based location capture, and logistics intake fields.

### Layoutoverview

- Зона "brand-stack" расположена в области "top centered column". Содержимое: "gradient storefront logo disk"; "Pegasus title"; "Retailer Portal subtitle".
- Зона "credential-core" расположена в области "main form column". Содержимое: "phone field"; "password field".
- Зона "registration-extension" расположена в области "below core credentials when sign-up mode active". Правило видимости: "visible when isLoginMode is false". Содержимое: "store name field"; "owner name field"; "store address field"; "Open Map button"; "Share Location button"; "selected location label"; "tax ID field"; "receiving window open and close fields"; "loading access type chip row"; "ceiling height field".
- Зона "error-row" расположена в области "below form fields when auth error exists". Правило видимости: "visible when auth.errorMessage is non-null". Содержимое: "warning icon"; "error text".
- Зона "primary-action-region" расположена в области "below form stack". Содержимое: "Sign In or Create Account button"; "mode toggle link".

### Controloverview

- Кнопка "Open Map" расположена в "registration-extension location row". Стиль: "outlined pill".
- Кнопка "Share Location" расположена в "registration-extension location row". Стиль: "outlined pill".
- Кнопка "Street" расположена в "registration-extension access type row". Стиль: "chip toggle".
- Кнопка "Alley" расположена в "registration-extension access type row". Стиль: "chip toggle".
- Кнопка "Dock" расположена в "registration-extension access type row". Стиль: "chip toggle".
- Кнопка "Sign In" расположена в "primary-action-region". Стиль: "full-width gradient CTA". Правило видимости: "isLoginMode true".
- Кнопка "Create Account" расположена в "primary-action-region". Стиль: "full-width gradient CTA". Правило видимости: "isLoginMode false".
- Кнопка "mode toggle link" расположена в "primary-action-region footer". Стиль: "text button".

### Iconoverview

- Иконка "storefront.fill" используется в зоне "brand-stack logo disk".
- Иконка "map" используется в зоне "Open Map button".
- Иконка "location.fill" используется в зоне "Share Location button".
- Иконка "arrow.right" используется в зоне "primary CTA trailing edge when not loading".
- Иконка "ProgressView" используется в зоне "primary CTA when auth.isLoading".
- Иконка "exclamationmark.triangle.fill" используется в зоне "error-row".

### Flowoverview

**Flowid:** retailer-login

**Summary:** Поток "retailer-login" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer enters phone and password".
- Шаг 2: "Retailer taps Sign In".
- Шаг 3: "AuthManager login executes and authenticated state transitions into the app shell".

---

**Flowid:** retailer-registration

**Summary:** Поток "retailer-registration" содержит 5 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer toggles to sign-up mode".
- Шаг 2: "Additional onboarding fields animate into view".
- Шаг 3: "Retailer may open map picker or share GPS location".
- Шаг 4: "Retailer submits Create Account".
- Шаг 5: "AuthManager register executes with location and logistics metadata".

---

**Flowid:** location-capture

**Summary:** Поток "location-capture" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer opens map picker or uses current location".
- Шаг 2: "Latitude and longitude populate state".
- Шаг 3: "Selected location label renders below the location row".

---


### Stateoverview

- Состояние: "login mode".
- Состояние: "registration mode".
- Состояние: "GPS locating state".
- Состояние: "error state".
- Состояние: "submitting state".
- Состояние: "map picker route handoff".

### Figureoverview

- Фигура: "retailer iOS login mode".
- Фигура: "retailer iOS registration mode".
- Фигура: "location capture row".
- Фигура: "error and submitting CTA state".

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

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-orders" для роли ритейлер на платформе iOS.

## Localized

**Purpose:** Поверхность "ios-retailer-orders" представляет страница для роли ритейлер на платформе iOS; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Retailer order-tracking hub with active, pending, and AI-planned tabs, detail-sheet drilldown, and QR overlay access for dispatched orders.

### Layoutoverview

- Зона "top-tabs" расположена в области "top full-width". Содержимое: "Active tab with count badge"; "Pending tab with count badge"; "AI Planned tab with count badge".
- Зона "tab-content-pager" расположена в области "main body". Содержимое: "active order card list"; "pending order card list"; "AI planned forecast list".
- Зона "detail-sheet" расположена в области "bottom sheet". Правило видимости: "visible when selectedOrder is non-null". Содержимое: "OrderDetailSheet at 75 percent height".
- Зона "qr-overlay" расположена в области "full-screen overlay". Правило видимости: "visible when qrOverlayOrder is non-null and status has delivery token". Содержимое: "QROverlay over current tab content".

### Controloverview

- Кнопка "tab selector" расположена в "top-tabs". Стиль: "tab button".
- Кнопка "Details" расположена в "active and pending card action row". Стиль: "pill button".
- Кнопка "Show QR" расположена в "active order card action row". Стиль: "accent pill button". Правило видимости: "order has delivery token".
- Кнопка "Pre-Order" расположена в "AI planned card trailing action". Стиль: "accent pill button".
- Кнопка "View" расположена в "pending order card trailing action". Стиль: "neutral pill button".

### Iconoverview

- Иконка "bolt.fill" используется в зоне "Active tab".
- Иконка "clock.fill" используется в зоне "Pending tab".
- Иконка "sparkles" используется в зоне "AI Planned tab".
- Иконка "shippingbox.fill or clock.fill or qrcode" используется в зоне "order cards and action pills".

### Flowoverview

**Flowid:** tabbed-order-navigation

**Summary:** Поток "tabbed-order-navigation" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer switches between Active, Pending, and AI Planned tabs".
- Шаг 2: "TabView page content swaps without index dots".

---

**Flowid:** order-drilldown

**Summary:** Поток "order-drilldown" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer taps Details or View on an order card".
- Шаг 2: "OrderDetailSheet opens with logistics, line items, totals, and QR content when available".

---

**Flowid:** qr-surface

**Summary:** Поток "qr-surface" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer taps Show QR on an active order".
- Шаг 2: "QROverlay appears over the orders interface".

---


### Stateoverview

- Состояние: "active tab populated".
- Состояние: "pending tab populated".
- Состояние: "AI planned tab populated".
- Состояние: "empty tab states".
- Состояние: "loading state".
- Состояние: "detail sheet open".
- Состояние: "QR overlay visible".

### Figureoverview

- Фигура: "retailer iOS orders active tab".
- Фигура: "retailer iOS orders pending tab".
- Фигура: "AI planned forecast card".
- Фигура: "orders QR overlay".

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

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-root-shell" для роли ритейлер на платформе iOS.

## Localized

**Purpose:** Поверхность "ios-retailer-root-shell" представляет страница для роли ритейлер на платформе iOS; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Authenticated retailer shell for tab navigation, toolbar-driven controls, floating active-order summary, and modal or sheet-based operational flows.

### Layoutoverview

- Зона "tab-layer" расположена в области "full-screen base layer". Содержимое: "Home tab"; "Catalog tab"; "Orders tab"; "Profile tab"; "Suppliers tab".
- Зона "toolbar" расположена в области "top navigation bar within each tab". Содержимое: "left: circular avatar/menu button"; "center: leaf icon plus Pegasus wordmark"; "right: cart button with count badge; notification bell with count badge".
- Зона "floating-summary" расположена в области "bottom above tab bar". Правило видимости: "visible on home, orders, and suppliers tabs only". Содержимое: "FloatingActiveOrdersBar".
- Зона "sheet-and-overlay-layer" расположена в области "above base layer". Содержимое: "SidebarMenu"; "BottomSheetOverlay containing ActiveDeliveriesView"; "DeliveryPaymentSheetView"; "FutureDemandView sheet"; "AutoOrderView sheet"; "CartView sheet"; "InsightsView sheet"; "ProfileView sheet".

### Controloverview

- Кнопка "avatar/menu button" расположена в "toolbar-left". Стиль: "circular gradient button".
- Кнопка "cart button" расположена в "toolbar-right". Стиль: "icon button with numeric badge".
- Кнопка "notification button" расположена в "toolbar-right". Стиль: "icon button with numeric badge".
- Кнопка "floating active orders bar" расположена в "bottom floating layer". Стиль: "pill-like full-width CTA".
- Кнопка "sidebar rows" расположена в "sidebar vertical stack". Стиль: "plain button rows with icon tile and chevron".
- Кнопка "Done toolbar actions" расположена в "sheet top-right confirmation slot". Стиль: "text confirmation button".

### Iconoverview

- Иконка "house / square.grid.2x2 / shippingbox / person.circle / building.2" используется в зоне "tab bar".
- Иконка "leaf.fill" используется в зоне "toolbar center brand mark".
- Иконка "cart" используется в зоне "toolbar cart action".
- Иконка "bell" используется в зоне "toolbar notification action".
- Иконка "chevron.up" используется в зоне "floating active orders bar expand affordance".
- Иконка "square.grid.2x2 / chart.bar / chart.line.uptrend.xyaxis / wand.and.stars / sparkles / tray / person / gearshape / rectangle.portrait.and.arrow.right" используется в зоне "sidebar rows".

### Flowoverview

**Flowid:** active-orders-drilldown

**Summary:** Поток "active-orders-drilldown" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "FloatingActiveOrdersBar appears when active orders exist".
- Шаг 2: "Retailer taps floating bar".
- Шаг 3: "BottomSheetOverlay presents ActiveDeliveriesView".
- Шаг 4: "Retailer navigates to delivery detail or payment sheet based on current order state".

---

**Flowid:** sidebar-mode-switching

**Summary:** Поток "sidebar-mode-switching" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Retailer taps avatar button".
- Шаг 2: "SidebarMenu animates in from left".
- Шаг 3: "Retailer selects dashboard, procurement, insights, auto-order, AI predictions, inbox, profile, or settings".
- Шаг 4: "Shell switches tab or opens target sheet".

---

**Flowid:** payment-event-presentation

**Summary:** Поток "payment-event-presentation" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "RetailerWebSocket sets paymentEvent".
- Шаг 2: "DeliveryPaymentSheetView presents as large sheet".
- Шаг 3: "Retailer resolves payment".
- Шаг 4: "Sheet dismiss triggers active-order reload".

---


### Stateoverview

- Состояние: "base tab shell".
- Состояние: "floating active orders visible".
- Состояние: "sidebar open with dimmed background".
- Состояние: "active deliveries bottom sheet open".
- Состояние: "payment sheet open".
- Состояние: "future demand sheet open".
- Состояние: "cart sheet open".
- Состояние: "insights sheet open".

### Figureoverview

- Фигура: "full iOS retailer shell".
- Фигура: "toolbar control cluster".
- Фигура: "floating active orders bar".
- Фигура: "sidebar overlay state".
- Фигура: "payment sheet state".

---

**Dossierfile:** retailer-ios-secondary-surfaces.json

**Bundleid:** retailer-ios-secondary-surfaces

**Appid:** retailer-app-ios

**Platform:** ios

**Role:** RETAILER

**Status:** implemented

**Entrytype:** bundle

**Localizedsummary:** Локализованный пакет "retailer-ios-secondary-surfaces" охватывает 17 поверхностей приложения "retailer-app-ios".

## Surfaces

**Pageid:** ios-retailer-home

**Viewname:** DashboardView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/DashboardView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-home" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-home" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Retailer service dashboard with breadbox-style tiles, quick reorder, and AI forecast highlights.

#### Layoutoverview

- Зона макета: "hero service tile grid".
- Зона макета: "quick reorder section".
- Зона макета: "AI prediction cards".
- Зона макета: "refresh scaffold".

#### Controloverview

- Элемент управления: "service tiles as navigation targets".

#### Iconoverview

- Иконографическая привязка: "service icons in tiles".

#### Flowoverview

**Summary:** Поток фиксируется как: "tap into catalog, orders, procurement, inbox, insights, history, search, profile".

---

**Summary:** Поток фиксируется как: "refresh dashboard data".

---


#### Dependencyoverview

##### Reads

- orders summary
- reorder products
- AI demand forecasts

##### Writes


##### Localizednotes

- Чтение: "orders summary", "reorder products", "AI demand forecasts".
- Запись: нет.

#### Stateoverview

- Состояние: "normal dashboard".
- Состояние: "refreshing".

#### Figureoverview

- Фигура: "retailer iOS dashboard tile grid".

#### Minifeatureoverview

- Минифункция: "service grid".
- Минифункция: "quick reorder strip".
- Минифункция: "AI cards".
- Минифункция: "pull refresh".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-category-suppliers

**Viewname:** CategorySuppliersView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/CategorySuppliersView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-category-suppliers" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-category-suppliers" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier list filtered by category for drill-down browsing.

#### Layoutoverview

- Зона макета: "navigation header".
- Зона макета: "supplier rows".
- Зона макета: "empty-state region".

#### Controloverview

- Элемент управления: "supplier row tap target".

#### Iconoverview

- Иконографическая привязка: "supplier initials tiles".
- Иконографическая привязка: "chevron affordance".

#### Flowoverview

**Summary:** Поток фиксируется как: "select supplier to enter supplier products".

---


#### Dependencyoverview

##### Reads

- suppliers by category endpoint

##### Writes


##### Localizednotes

- Чтение: "suppliers by category endpoint".
- Запись: нет.

#### Stateoverview

- Состояние: "rows present".
- Состояние: "empty".

#### Figureoverview

- Фигура: "category supplier list".

#### Minifeatureoverview

- Минифункция: "supplier rows".
- Минифункция: "status badge".
- Минифункция: "empty state".

**Minifeaturecount:** 3

---

**Pageid:** ios-retailer-my-suppliers

**Viewname:** MySuppliersView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/MySuppliersView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-my-suppliers" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-my-suppliers" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Favorite supplier gallery with search, refresh, order counts, and auto-order badges.

#### Layoutoverview

- Зона макета: "search field".
- Зона макета: "supplier card grid".
- Зона макета: "empty-state region".

#### Controloverview

- Элемент управления: "supplier card tap target".

#### Iconoverview

- Иконографическая привязка: "avatar initials".
- Иконографическая привязка: "auto-order badge".

#### Flowoverview

**Summary:** Поток фиксируется как: "search favorite suppliers".

---

**Summary:** Поток фиксируется как: "refresh supplier grid".

---

**Summary:** Поток фиксируется как: "open supplier products".

---


#### Dependencyoverview

##### Reads

- favorite suppliers

##### Writes


##### Localizednotes

- Чтение: "favorite suppliers".
- Запись: нет.

#### Stateoverview

- Состояние: "grid populated".
- Состояние: "empty".

#### Figureoverview

- Фигура: "favorite supplier grid with badges".

#### Minifeatureoverview

- Минифункция: "search".
- Минифункция: "grid".
- Минифункция: "order count badge".
- Минифункция: "auto-order badge".
- Минифункция: "refresh".

**Minifeaturecount:** 5

---

**Pageid:** ios-retailer-supplier-products

**Viewname:** SupplierProductsView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/SupplierProductsView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-supplier-products" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-supplier-products" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier catalog grouped by category with supplier follow-state and supplier-level auto-order control.

#### Layoutoverview

- Зона макета: "supplier header card".
- Зона макета: "Add or Remove Supplier control".
- Зона макета: "supplier auto-order toggle".
- Зона макета: "category-grouped product sections".

#### Controloverview

- Элемент управления: "Add or Remove Supplier button".
- Элемент управления: "supplier auto-order toggle".
- Элемент управления: "product row tap target".

#### Iconoverview

- Иконографическая привязка: "supplier avatar initials".
- Иконографическая привязка: "OPEN or CLOSED badge".

#### Flowoverview

**Summary:** Поток фиксируется как: "favorite or unfavorite supplier".

---

**Summary:** Поток фиксируется как: "toggle supplier auto-order".

---

**Summary:** Поток фиксируется как: "open product detail".

---


#### Dependencyoverview

##### Reads

- supplier products endpoint

##### Writes

- favorite supplier endpoint
- supplier auto-order endpoint

##### Localizednotes

- Чтение: "supplier products endpoint".
- Запись: "favorite supplier endpoint", "supplier auto-order endpoint".

#### Stateoverview

- Состояние: "supplier open".
- Состояние: "supplier closed".

#### Figureoverview

- Фигура: "supplier products screen with header card".

#### Minifeatureoverview

- Минифункция: "supplier header".
- Минифункция: "favorite button".
- Минифункция: "auto-order toggle".
- Минифункция: "status badge".
- Минифункция: "grouped products".

**Minifeaturecount:** 5

---

**Pageid:** ios-retailer-product-detail

**Viewname:** ProductDetailView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ProductDetailView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-product-detail" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-product-detail" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Product detail inspector with imagery, quantity selection, variant logic, and add-to-cart flow.

#### Layoutoverview

- Зона макета: "hero image area".
- Зона макета: "product info stack".
- Зона макета: "variant selector".
- Зона макета: "quantity controls".
- Зона макета: "bottom add-to-cart action area".

#### Controloverview

- Элемент управления: "variant chip buttons".
- Элемент управления: "quantity plus and minus".
- Элемент управления: "Add to Cart button".

#### Iconoverview

- Иконографическая привязка: "placeholder image glyph".
- Иконографическая привязка: "nutrition or metadata icons".

#### Flowoverview

**Summary:** Поток фиксируется как: "select variant".

---

**Summary:** Поток фиксируется как: "adjust quantity".

---

**Summary:** Поток фиксируется как: "add product to cart".

---


#### Dependencyoverview

##### Reads

- product detail payload

##### Writes

- cart state mutation

##### Localizednotes

- Чтение: "product detail payload".
- Запись: "cart state mutation".

#### Stateoverview

- Состояние: "image present".
- Состояние: "placeholder image".

#### Figureoverview

- Фигура: "product detail screen with bottom CTA".

#### Minifeatureoverview

- Минифункция: "hero image".
- Минифункция: "variant chips".
- Минифункция: "quantity stepper".
- Минифункция: "Add to Cart CTA".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-category-products

**Viewname:** CategoryProductsView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/CategoryProductsView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-category-products" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-category-products" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Category-scoped products grouped by supplier with inline per-product auto-order controls.

#### Layoutoverview

- Зона макета: "category header".
- Зона макета: "collapsible supplier sections".
- Зона макета: "product rows with quantity and toggle controls".

#### Controloverview

- Элемент управления: "supplier group expand or collapse".
- Элемент управления: "product auto-order toggle".
- Элемент управления: "product row tap target".

#### Iconoverview

- Иконографическая привязка: "section chevrons".
- Иконографическая привязка: "auto-order badge".

#### Flowoverview

**Summary:** Поток фиксируется как: "expand supplier group".

---

**Summary:** Поток фиксируется как: "toggle product auto-order".

---

**Summary:** Поток фиксируется как: "open product detail".

---


#### Dependencyoverview

##### Reads

- products by category

##### Writes

- product auto-order endpoint

##### Localizednotes

- Чтение: "products by category".
- Запись: "product auto-order endpoint".

#### Stateoverview

- Состояние: "collapsed groups".
- Состояние: "expanded groups".

#### Figureoverview

- Фигура: "category products grouped by supplier".

#### Minifeatureoverview

- Минифункция: "group headers".
- Минифункция: "expand collapse".
- Минифункция: "product toggle".
- Минифункция: "quantity adjuster".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-active-order

**Viewname:** ActiveOrderView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ActiveOrderView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-active-order" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-active-order" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Active-order monitor for in-transit orders with live-state emphasis.

#### Layoutoverview

- Зона макета: "active order card list".
- Зона макета: "status emphasis band".
- Зона макета: "refresh scaffold".

#### Controloverview

- Элемент управления: "order card tap target".

#### Iconoverview

- Иконографическая привязка: "live indicator dot".
- Иконографическая привязка: "status badge".

#### Flowoverview

**Summary:** Поток фиксируется как: "refresh active orders".

---

**Summary:** Поток фиксируется как: "open selected order detail".

---


#### Dependencyoverview

##### Reads

- orders filtered by IN_TRANSIT

##### Writes


##### Localizednotes

- Чтение: "orders filtered by IN_TRANSIT".
- Запись: нет.

#### Stateoverview

- Состояние: "active orders present".
- Состояние: "no active orders".

#### Figureoverview

- Фигура: "active order list".

#### Minifeatureoverview

- Минифункция: "live dot".
- Минифункция: "order cards".
- Минифункция: "status pill".
- Минифункция: "refresh".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-arrival

**Viewname:** ArrivalView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ArrivalView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-arrival" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-arrival" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Arrival-state order list emphasizing imminent handoff and manual arrival confirmation.

#### Layoutoverview

- Зона макета: "arrival order cards".
- Зона макета: "ETA countdown label".
- Зона макета: "acknowledge action row".

#### Controloverview

- Элемент управления: "manual arrival confirm button".
- Элемент управления: "order card tap target".

#### Iconoverview

- Иконографическая привязка: "arrival arrow icon".
- Иконографическая привязка: "status indicator".

#### Flowoverview

**Summary:** Поток фиксируется как: "acknowledge arrival".

---

**Summary:** Поток фиксируется как: "open selected order".

---


#### Dependencyoverview

##### Reads

- orders filtered by ARRIVED

##### Writes

- confirm arrival endpoint

##### Localizednotes

- Чтение: "orders filtered by ARRIVED".
- Запись: "confirm arrival endpoint".

#### Stateoverview

- Состояние: "arrival cards present".
- Состояние: "no arrivals".

#### Figureoverview

- Фигура: "arrival card list with confirm action".

#### Minifeatureoverview

- Минифункция: "ETA countdown".
- Минифункция: "confirm arrival action".
- Минифункция: "arrival icon".
- Минифункция: "order cards".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-future-demand

**Viewname:** FutureDemandView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/FutureDemandView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-future-demand" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-future-demand" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** AI demand forecast gallery with confidence indicators and procurement handoff.

#### Layoutoverview

- Зона макета: "forecast header card".
- Зона макета: "stats strip".
- Зона макета: "forecast card list".
- Зона макета: "close control".

#### Controloverview

- Элемент управления: "close button".
- Элемент управления: "drill-down to procurement control".

#### Iconoverview

- Иконографическая привязка: "sparkles icon".
- Иконографическая привязка: "confidence ring graphics".

#### Flowoverview

**Summary:** Поток фиксируется как: "review predicted quantities and confidence".

---

**Summary:** Поток фиксируется как: "move into procurement workflow".

---


#### Dependencyoverview

##### Reads

- AI demand forecasts

##### Writes


##### Localizednotes

- Чтение: "AI demand forecasts".
- Запись: нет.

#### Stateoverview

- Состояние: "forecasts present".
- Состояние: "empty forecasts".

#### Figureoverview

- Фигура: "future demand modal with confidence rings".

#### Minifeatureoverview

- Минифункция: "sparkles header".
- Минифункция: "confidence rings".
- Минифункция: "stats strip".
- Минифункция: "forecast cards".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-auto-order

**Viewname:** AutoOrderView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/AutoOrderView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-auto-order" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-auto-order" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Hierarchical auto-order governance for supplier, category, and product scopes.

#### Layoutoverview

- Зона макета: "header".
- Зона макета: "suggestion cards".
- Зона макета: "checkbox or toggle rows".
- Зона макета: "bulk action bar".

#### Controloverview

- Элемент управления: "row toggles".
- Элемент управления: "Select All".
- Элемент управления: "Deselect All".
- Элемент управления: "submit action".

#### Iconoverview

- Иконографическая привязка: "scope icons and checkmarks".

#### Flowoverview

**Summary:** Поток фиксируется как: "select targets for auto-order".

---

**Summary:** Поток фиксируется как: "choose history or fresh behavior".

---

**Summary:** Поток фиксируется как: "submit updated settings".

---


#### Dependencyoverview

##### Reads

- auto-order settings

##### Writes

- auto-order action endpoints

##### Localizednotes

- Чтение: "auto-order settings".
- Запись: "auto-order action endpoints".

#### Stateoverview

- Состояние: "mixed toggles".
- Состояние: "all off".
- Состояние: "selection pending".

#### Figureoverview

- Фигура: "auto-order hierarchy screen".

#### Minifeatureoverview

- Минифункция: "global or scoped toggles".
- Минифункция: "bulk select".
- Минифункция: "submit".
- Минифункция: "suggestion cards".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-insights

**Viewname:** InsightsView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/InsightsView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-insights" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-insights" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Expense analytics dashboard with selectable windows, KPI cards, charts, and top supplier or product breakdowns.

#### Layoutoverview

- Зона макета: "date-range filter row".
- Зона макета: "KPI cards".
- Зона макета: "chart region".
- Зона макета: "supplier and product expense tables".

#### Controloverview

- Элемент управления: "range buttons".

#### Iconoverview

- Иконографическая привязка: "chart accents and analytics icons".

#### Flowoverview

**Summary:** Поток фиксируется как: "change time horizon".

---

**Summary:** Поток фиксируется как: "refresh insights dataset".

---


#### Dependencyoverview

##### Reads

- retailer analytics endpoint

##### Writes


##### Localizednotes

- Чтение: "retailer analytics endpoint".
- Запись: нет.

#### Stateoverview

- Состояние: "analytics loaded".
- Состояние: "empty analytics".

#### Figureoverview

- Фигура: "insights dashboard with chart".

#### Minifeatureoverview

- Минифункция: "range filters".
- Минифункция: "KPI cards".
- Минифункция: "expense chart".
- Минифункция: "supplier table".
- Минифункция: "product table".

**Minifeaturecount:** 5

---

**Pageid:** ios-retailer-profile

**Viewname:** ProfileView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ProfileView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-profile" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-profile" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Retailer account hub combining profile identity, settings links, support, and global empathy-engine auto-order toggles.

#### Layoutoverview

- Зона макета: "gradient header card".
- Зона макета: "stats row".
- Зона макета: "order history link".
- Зона макета: "empathy engine toggle card".
- Зона макета: "company and support sections".

#### Controloverview

- Элемент управления: "global auto-order toggle".
- Элемент управления: "settings row links".
- Элемент управления: "logout action".

#### Iconoverview

- Иконографическая привязка: "avatar initials".
- Иконографическая привязка: "settings row icons".

#### Flowoverview

**Summary:** Поток фиксируется как: "toggle global auto-order with history or fresh branching".

---

**Summary:** Поток фиксируется как: "open settings links".

---

**Summary:** Поток фиксируется как: "logout".

---


#### Dependencyoverview

##### Reads

- retailer profile

##### Writes

- global auto-order endpoint

##### Localizednotes

- Чтение: "retailer profile".
- Запись: "global auto-order endpoint".

#### Stateoverview

- Состояние: "profile loaded".
- Состояние: "toggle dialog open".

#### Figureoverview

- Фигура: "profile page with gradient header".

#### Minifeatureoverview

- Минифункция: "gradient profile header".
- Минифункция: "stats row".
- Минифункция: "history link".
- Минифункция: "global auto-order".
- Минифункция: "settings sections".
- Минифункция: "logout".

**Minifeaturecount:** 6

---

**Pageid:** ios-retailer-procurement

**Viewname:** ProcurementView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/ProcurementView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-procurement" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-procurement" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** AI-assisted procurement composer where predicted line items can be accepted, edited, and submitted as an order basket.

#### Layoutoverview

- Зона макета: "header card".
- Зона макета: "suggested items list".
- Зона макета: "quantity spinners".
- Зона макета: "bulk action bar".

#### Controloverview

- Элемент управления: "item checkbox".
- Элемент управления: "quantity controls".
- Элемент управления: "Select All".
- Элемент управления: "Deselect All".
- Элемент управления: "Submit Order".

#### Iconoverview

- Иконографическая привязка: "confidence or suggestion icons".

#### Flowoverview

**Summary:** Поток фиксируется как: "toggle predicted items".

---

**Summary:** Поток фиксируется как: "manually adjust quantities".

---

**Summary:** Поток фиксируется как: "submit procurement order".

---


#### Dependencyoverview

##### Reads

- AI demand forecasts

##### Writes

- procurement order endpoint

##### Localizednotes

- Чтение: "AI demand forecasts".
- Запись: "procurement order endpoint".

#### Stateoverview

- Состояние: "suggestions present".
- Состояние: "none selected".
- Состояние: "submit pending".

#### Figureoverview

- Фигура: "procurement suggestion screen".

#### Minifeatureoverview

- Минифункция: "suggestion list".
- Минифункция: "checkboxes".
- Минифункция: "quantity spinners".
- Минифункция: "bulk actions".
- Минифункция: "Submit Order".

**Minifeaturecount:** 5

---

**Pageid:** ios-retailer-inbox

**Viewname:** InboxView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/InboxView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-inbox" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-inbox" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Operational inbox of arriving, loaded, and in-transit orders for attention routing.

#### Layoutoverview

- Зона макета: "order feed list".
- Зона макета: "status pills".
- Зона макета: "ETA countdown or timing labels".

#### Controloverview

- Элемент управления: "order card tap target".

#### Iconoverview

- Иконографическая привязка: "supplier badges".
- Иконографическая привязка: "status pills".

#### Flowoverview

**Summary:** Поток фиксируется как: "refresh inbox feed".

---

**Summary:** Поток фиксируется как: "open order from feed".

---


#### Dependencyoverview

##### Reads

- orders filtered for transit statuses

##### Writes


##### Localizednotes

- Чтение: "orders filtered for transit statuses".
- Запись: нет.

#### Stateoverview

- Состояние: "feed populated".
- Состояние: "empty feed".

#### Figureoverview

- Фигура: "inbox feed with status pills".

#### Minifeatureoverview

- Минифункция: "status filter logic".
- Минифункция: "feed cards".
- Минифункция: "ETA labels".
- Минифункция: "supplier badge".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-history

**Viewname:** HistoryView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/HistoryView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-history" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-history" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Historical order browser with status chip filters and drill-down entry.

#### Layoutoverview

- Зона макета: "status chip scroller".
- Зона макета: "order history card list".

#### Controloverview

- Элемент управления: "status chips".
- Элемент управления: "order card tap target".

#### Iconoverview

- Иконографическая привязка: "status chip accents".

#### Flowoverview

**Summary:** Поток фиксируется как: "filter by status".

---

**Summary:** Поток фиксируется как: "refresh history".

---

**Summary:** Поток фиксируется как: "open specific order".

---


#### Dependencyoverview

##### Reads

- orders by status

##### Writes


##### Localizednotes

- Чтение: "orders by status".
- Запись: нет.

#### Stateoverview

- Состояние: "all orders".
- Состояние: "filtered subset".
- Состояние: "empty filter result".

#### Figureoverview

- Фигура: "history screen with status chips".

#### Minifeatureoverview

- Минифункция: "status chips".
- Минифункция: "history list".
- Минифункция: "refresh".

**Minifeaturecount:** 3

---

**Pageid:** ios-retailer-search

**Viewname:** SearchView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/SearchView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-search" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-search" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Global product search across suppliers and categories.

#### Layoutoverview

- Зона макета: "search bar".
- Зона макета: "clear affordance".
- Зона макета: "empty search prompt".
- Зона макета: "result card grid".

#### Controloverview

- Элемент управления: "clear search button".
- Элемент управления: "product card tap target".

#### Iconoverview

- Иконографическая привязка: "magnifying glass".
- Иконографическая привязка: "clear xmark".

#### Flowoverview

**Summary:** Поток фиксируется как: "type search query".

---

**Summary:** Поток фиксируется как: "clear query".

---

**Summary:** Поток фиксируется как: "open selected product detail".

---


#### Dependencyoverview

##### Reads

- products search endpoint

##### Writes


##### Localizednotes

- Чтение: "products search endpoint".
- Запись: нет.

#### Stateoverview

- Состояние: "empty query".
- Состояние: "results shown".
- Состояние: "no results".

#### Figureoverview

- Фигура: "global product search screen".

#### Minifeatureoverview

- Минифункция: "search bar".
- Минифункция: "clear button".
- Минифункция: "result grid".
- Минифункция: "empty prompt".

**Minifeaturecount:** 4

---

**Pageid:** ios-retailer-location-picker

**Viewname:** LocationPickerView

**Surfacetype:** screen

**Sourcefile:** apps/retailer-app-ios/reatilerapp/reatilerapp/Screens/LocationPickerView.swift

**Localizedsummary:** Локализованный обзор поверхности "ios-retailer-location-picker" для роли unknown-role на платформе unknown-platform.

### Localized

**Purpose:** Поверхность "ios-retailer-location-picker" представляет экран для роли unknown-role на платформе unknown-platform; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** MapKit location picker for retailer signup and location updates.

#### Layoutoverview

- Зона макета: "map view".
- Зона макета: "center pin".
- Зона макета: "address label".
- Зона макета: "close control".
- Зона макета: "Confirm Location button".

#### Controloverview

- Элемент управления: "close xmark button".
- Элемент управления: "Confirm Location button".

#### Iconoverview

- Иконографическая привязка: "mappin circle and arrowtriangle center glyph".

#### Flowoverview

**Summary:** Поток фиксируется как: "move map under fixed pin".

---

**Summary:** Поток фиксируется как: "resolve address".

---

**Summary:** Поток фиксируется как: "confirm chosen location".

---


#### Dependencyoverview

##### Reads

- MapKit reverse geocoder state

##### Writes

- selected location callback

##### Localizednotes

- Чтение: "MapKit reverse geocoder state".
- Запись: "selected location callback".

#### Stateoverview

- Состояние: "default location".
- Состояние: "adjusted location".

#### Figureoverview

- Фигура: "iOS location picker with centered pin".

#### Minifeatureoverview

- Минифункция: "map pin".
- Минифункция: "address label".
- Минифункция: "close button".
- Минифункция: "confirm CTA".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-orders" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-orders" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier order-lifecycle command page for approval, reassignment, search, filtering, history viewing, and order inspection.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "left: headline: Orders; subtitle: Manage the full order lifecycle — approval, dispatch, tracking, and history"; "right: Refresh button".
- Зона "tab-row" расположена в области "below divider". Содержимое: "Active tab button with orders icon and count badge"; "Scheduled tab button with schedule icon and count badge when active".
- Зона "filter-row" расположена в области "below tabs". Содержимое: "search input for order ID or retailer"; "state filter select"; "History toggle chip with ledger icon".
- Зона "bulk-action-bar" расположена в области "conditional row above table". Правило видимости: "visible when one or more rows are selected". Содержимое: "left: selected count"; "right: Reassign Truck button when selection is eligible; Clear button".
- Зона "table-region" расположена в области "primary content card". Содержимое: "select-all checkbox header"; "order ID column"; "retailer column"; "state badge column"; "truck or route column"; "delivery date column"; "items column"; "amount column"; "payment column"; "created timestamp column"; "actions column".

### Controloverview

- Кнопка "Refresh" расположена в "header-right". Стиль: "secondary". Иконка: "returns".
- Кнопка "Active tab" расположена в "tab-row-left". Стиль: "tab". Иконка: "orders".
- Кнопка "Scheduled tab" расположена в "tab-row-left". Стиль: "tab". Иконка: "schedule".
- Кнопка "History" расположена в "filter-row-right". Стиль: "chip-toggle". Иконка: "ledger".
- Кнопка "Approve" расположена в "table-row-actions". Стиль: "primary". Правило видимости: "active tab and order state is PENDING".
- Кнопка "Reject" расположена в "table-row-actions". Стиль: "outline-danger". Правило видимости: "active tab and order state is PENDING".
- Кнопка "Reassign" расположена в "table-row-actions". Стиль: "secondary". Правило видимости: "active tab and order state is PENDING or LOADED and route_id exists".
- Кнопка "Reassign Truck" расположена в "bulk-action-bar-right". Стиль: "primary". Правило видимости: "selected rows all eligible for reassignment".
- Кнопка "Clear" расположена в "bulk-action-bar-right". Стиль: "ghost".
- Кнопка "Cancel" расположена в "reassign-dialog-footer-left". Стиль: "ghost".
- Кнопка "Reassign N Order(s)" расположена в "reassign-dialog-footer-right". Стиль: "primary".
- Кнопка "Reassign to Different Truck" расположена в "detail-drawer-footer". Стиль: "secondary". Правило видимости: "drawer open and order has route_id and state is PENDING or LOADED".

### Iconoverview

- Иконка "returns" используется в зоне "header-right refresh button".
- Иконка "orders" используется в зоне "Active tab button".
- Иконка "schedule" используется в зоне "Scheduled tab button".
- Иконка "ledger" используется в зоне "History filter chip".
- Иконка "StatusBadge" используется в зоне "state column and detail drawer".

### Flowoverview

**Flowid:** approve-pending-order

**Summary:** Поток "approve-pending-order" содержит 5 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier stays on Active tab".
- Шаг 2: "Clicks Approve in row actions".
- Шаг 3: "Page posts to /v1/supplier/orders/vet".
- Шаг 4: "Toast shows result".
- Шаг 5: "Orders reload".

---

**Flowid:** reject-pending-order

**Summary:** Поток "reject-pending-order" содержит 5 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier clicks Reject in row actions".
- Шаг 2: "Inline reason input appears in actions column".
- Шаг 3: "Supplier types reason and confirms Reject".
- Шаг 4: "Page posts to /v1/supplier/orders/vet with decision REJECTED".
- Шаг 5: "Toast shows result and rows reload".

---

**Flowid:** single-or-bulk-reassign

**Summary:** Поток "single-or-bulk-reassign" содержит 7 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier clicks Reassign in a row or selects multiple eligible rows".
- Шаг 2: "Dialog opens".
- Шаг 3: "Supplier selects target truck".
- Шаг 4: "Capacity metrics and capacity bar render".
- Шаг 5: "Supplier confirms reassignment".
- Шаг 6: "Page posts to /v1/fleet/reassign".
- Шаг 7: "Toast summarizes reassign or conflict results".

---

**Flowid:** row-inspection

**Summary:** Поток "row-inspection" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier clicks any table row".
- Шаг 2: "Order detail drawer opens from right".
- Шаг 3: "Drawer shows ID, status, retailer, payment, assignment, timestamps, and optional reassign CTA".

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

- Чтение: "/v1/supplier/orders", "/v1/fleet/active", "/v1/fleet/capacity".
- Запись: "/v1/supplier/orders/vet", "/v1/fleet/reassign".
- Модель обновления: "manual refresh plus 30-second polling when not in history mode".

### Stateoverview

- Состояние: "loading skeleton rows".
- Состояние: "empty state by active or scheduled or history context".
- Состояние: "normal data table".
- Состояние: "selected rows accent-soft highlighting".
- Состояние: "reject inline-input mode".
- Состояние: "detail drawer open".
- Состояние: "reassignment dialog open".

### Figureoverview

- Фигура: "full-page command view with tab row and filter row".
- Фигура: "table row with approve and reject controls".
- Фигура: "bulk-action bar with selected rows and reassign CTA".
- Фигура: "reassignment dialog with capacity bar".
- Фигура: "order detail drawer".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-analytics-demand" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-analytics-demand" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier advanced demand-forecast page comparing predicted versus actual order volume over time and listing upcoming AI-planned order line items.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "circular back button linking to analytics hub"; "headline: AI Demand Analytics"; "subtitle describing predicted versus actual volume over a 30-day window".
- Зона "kpi-row" расположена в области "below header". Содержимое: "Prediction Accuracy card"; "Upcoming AI Orders card"; "Data Points card".
- Зона "chart-card" расположена в области "below KPI row". Содержимое: "dual-axis line chart"; "legend"; "tooltip-driven predicted and actual value inspection"; "empty chart message when time series is absent".
- Зона "upcoming-orders-card" расположена в области "bottom full-width". Содержимое: "table header"; "upcoming AI-planned order rows with date, retailer, SKU, product, predicted quantity"; "pagination controls or empty-state message".

### Controloverview

- Кнопка "Back to analytics" расположена в "header far-left circular control". Стиль: "round surface button".

### Iconoverview

- Иконка "left-arrow glyph" используется в зоне "header back button".

### Flowoverview

**Flowid:** demand-history-bootstrap

**Summary:** Поток "demand-history-bootstrap" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Page reads supplier token from cookie".
- Шаг 2: "If token is absent, page renders supplier-credentials-required error state".
- Шаг 3: "Otherwise page fetches /v1/supplier/analytics/demand/history and binds time series plus upcoming rows".

---

**Flowid:** chart-review

**Summary:** Поток "chart-review" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier inspects dual-axis lines for predicted and actual value and quantity".
- Шаг 2: "Tooltip exposes exact UZS and quantity values for a selected date".

---

**Flowid:** upcoming-order-review

**Summary:** Поток "upcoming-order-review" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier reviews paginated upcoming AI-planned order rows".
- Шаг 2: "Shared pagination controls advance through the upcoming dataset".

---


### Dependencyoverview

#### Reads

- /v1/supplier/analytics/demand/history

#### Writes


#### Localizednotes

- Чтение: "/v1/supplier/analytics/demand/history".
- Запись: нет.
- Модель обновления: "single fetch on mount with local pagination over the upcoming rows".

### Stateoverview

- Состояние: "page loading spinner".
- Состояние: "unauthorized error card".
- Состояние: "history error card".
- Состояние: "chart with time-series data".
- Состояние: "chart empty state with no time-series data available".
- Состояние: "upcoming rows table with pagination".
- Состояние: "upcoming rows empty message".

### Figureoverview

- Фигура: "full advanced demand analytics page with header, KPI row, chart, and table".
- Фигура: "chart card close-up showing four line series and legend".
- Фигура: "upcoming orders table close-up with pagination footer".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-analytics" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-analytics" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier analytics hub presenting financial velocity, AI demand highlights, and deep links into advanced forecast review and dispatch operations.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "headline: Analytics"; "subtitle describing financial overview and operational intelligence"; "Demand Forecast CTA"; "Dispatch Room CTA".
- Зона "ai-demand-card" расположена в области "below header when demand predictions exist". Правило видимости: "visible when demand prediction_count is greater than zero". Содержимое: "analytics avatar circle"; "prediction count chip"; "three-column forecast metrics"; "forecasted SKU pills"; "View Advanced Analytics CTA".
- Зона "kpi-grid" расположена в области "below demand card or directly below header". Содержимое: "Gross Volume card"; "Total Pallets card"; "Avg Velocity per SKU card"; "Top SKU card".
- Зона "velocity-chart-region" расположена в области "below KPI grid". Содержимое: "VelocityChart component spanning page width".
- Зона "sku-breakdown-table" расположена в области "bottom full-width card when velocity data exists". Содержимое: "table with SKU ID, pallets, volume, and share bar".

### Controloverview

- Кнопка "Demand Forecast" расположена в "header top-right". Стиль: "soft accent link button".
- Кнопка "Dispatch Room" расположена в "header top-right". Стиль: "filled accent link button".
- Кнопка "View Advanced Analytics" расположена в "ai-demand-card footer". Стиль: "pill button". Правило видимости: "visible when demand card is rendered".

### Iconoverview

- Иконка "error" используется в зоне "error card".
- Иконка "analytics" используется в зоне "Demand Forecast header CTA and AI demand card avatar".
- Иконка "orders" используется в зоне "Dispatch Room header CTA".
- Иконка "arrow_forward" используется в зоне "View Advanced Analytics CTA trailing icon".

### Flowoverview

**Flowid:** analytics-bootstrap

**Summary:** Поток "analytics-bootstrap" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Page fetches /v1/supplier/analytics/velocity and /v1/supplier/analytics/demand/today in parallel using apiFetch".
- Шаг 2: "Loading skeletons occupy the KPI and chart regions until both requests settle".
- Шаг 3: "Velocity metrics derive from the returned data array and top SKU is computed client-side".

---

**Flowid:** forecast-escalation

**Summary:** Поток "forecast-escalation" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier selects Demand Forecast in the header or View Advanced Analytics in the AI demand card".
- Шаг 2: "Navigation moves to /supplier/analytics/demand for deeper analysis".

---

**Flowid:** dispatch-escalation

**Summary:** Поток "dispatch-escalation" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier selects Dispatch Room".
- Шаг 2: "Navigation leaves analytics and returns to the operational command surface linked by the app root".

---


### Dependencyoverview

#### Reads

- /v1/supplier/analytics/velocity
- /v1/supplier/analytics/demand/today

#### Writes


#### Localizednotes

- Чтение: "/v1/supplier/analytics/velocity", "/v1/supplier/analytics/demand/today".
- Запись: нет.
- Модель обновления: "single load on mount with computed metrics derived in memory".

### Stateoverview

- Состояние: "skeleton loading state".
- Состояние: "error card state".
- Состояние: "analytics hub with AI demand card visible".
- Состояние: "analytics hub without AI demand card".
- Состояние: "velocity chart with SKU breakdown table".
- Состояние: "analytics hub with no velocity rows and no bottom table".

### Figureoverview

- Фигура: "full analytics hub with AI demand card, KPI grid, chart, and breakdown table".
- Фигура: "AI demand card close-up with metric triplet and forecast pills".
- Фигура: "SKU breakdown row with share bar".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-catalog" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-catalog" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier catalog control page combining product creation, promotion creation, category filtering, product-ledger review, status toggling, and modal-based product editing.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "headline: Inventory Control"; "subtitle: Catalog Injection, Product Ledger and Promotional Routing".
- Зона "kpi-strip" расположена в области "below header". Содержимое: "Total SKUs card"; "Active card"; "Inactive card"; "Catalog Value card".
- Зона "category-filter-row" расположена в области "below KPI strip when operating categories exist". Содержимое: "All chip with total count"; "category chips with per-category counts".
- Зона "dual-form-region" расположена в области "two-column band". Содержимое: "SupplierProductForm on left"; "SupplierPromotionForm on right".
- Зона "product-ledger-table" расположена в области "bottom full-width ledger card". Содержимое: "ledger header with product count chip"; "product table with image cell, category chip, price, VU, block, MOQ-step, status, truncated SKU, actions"; "empty state when no products match filter".
- Зона "edit-product-modal" расположена в области "centered modal overlay". Правило видимости: "visible when editProduct exists". Содержимое: "modal header with title and close button"; "optional error banner"; "name input"; "description textarea"; "base price input"; "MOQ, Step, Units-per-Block triplet"; "image preview and file picker"; "Cancel and Save Changes footer buttons".

### Controloverview

- Кнопка "All" расположена в "category-filter-row". Стиль: "chip".
- Кнопка "Category chip" расположена в "category-filter-row". Стиль: "chip".
- Кнопка "Edit" расположена в "product-ledger actions column". Стиль: "soft accent small".
- Кнопка "Deactivate" расположена в "product-ledger actions column". Стиль: "soft danger small". Правило видимости: "product is active".
- Кнопка "Activate" расположена в "product-ledger actions column". Стиль: "soft success small". Правило видимости: "product is inactive".
- Кнопка "Close" расположена в "edit-product-modal header-right". Стиль: "small muted".
- Кнопка "Cancel" расположена в "edit-product-modal footer-left". Стиль: "pill muted".
- Кнопка "Save Changes" расположена в "edit-product-modal footer-right". Стиль: "pill primary".

### Iconoverview

- Иконка "image placeholder glyph" используется в зоне "product table image cell when image_url missing".
- Иконка "catalog" используется в зоне "ledger empty state".
- Иконка "status badge chip" используется в зоне "status column".

### Flowoverview

**Flowid:** catalog-bootstrap

**Summary:** Поток "catalog-bootstrap" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Page fetches /v1/supplier/products and /v1/supplier/profile in parallel".
- Шаг 2: "If operating categories are present, page also maps them against /v1/catalog/platform-categories".
- Шаг 3: "KPI strip and filtered ledger compute from catalog payload".

---

**Flowid:** category-filtering

**Summary:** Поток "category-filtering" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier clicks All or a category chip".
- Шаг 2: "Filtered ledger, active count, inactive count, and catalog value recompute client-side".

---

**Flowid:** product-editing

**Summary:** Поток "product-editing" содержит 6 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier clicks Edit in ledger row".
- Шаг 2: "Edit modal opens prefilled with product data".
- Шаг 3: "Supplier may replace image, update quantities, price, and copy".
- Шаг 4: "Page optionally obtains upload ticket from /v1/supplier/products/upload-ticket and uploads image".
- Шаг 5: "Page puts updated payload to /v1/supplier/products/{sku_id}".
- Шаг 6: "Catalog reloads".

---

**Flowid:** status-toggle

**Summary:** Поток "status-toggle" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier clicks Activate or Deactivate in row action cluster".
- Шаг 2: "Page updates is_active through /v1/supplier/products/{sku_id}".
- Шаг 3: "Ledger refreshes".

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

- Чтение: "/v1/supplier/products", "/v1/supplier/profile", "/v1/catalog/platform-categories".
- Запись: "/v1/supplier/products/{sku_id}", "/v1/supplier/products/upload-ticket".
- Модель обновления: "initial fetch on mount plus targeted reload after edit and status-toggle actions".

### Stateoverview

- Состояние: "page spinner loading state".
- Состояние: "error card state".
- Состояние: "ledger empty state with forms still visible".
- Состояние: "ledger with image thumbnails and category chips".
- Состояние: "row-level toggle pending state".
- Состояние: "edit-product modal open".
- Состояние: "edit-product modal with upload preview".
- Состояние: "edit-product modal saving state".

### Figureoverview

- Фигура: "full catalog page with KPI strip, filters, two-column forms, and ledger".
- Фигура: "product-ledger table close-up".
- Фигура: "edit-product modal".
- Фигура: "ledger row with image placeholder and activation toggle".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-crm" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-crm" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier retailer-relationship page for tracking retailer lifetime value, order history, and contact detail in a table-plus-drawer CRM workspace.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "headline: Retailer CRM"; "subtitle describing retailer relationship and lifetime-value tracking".
- Зона "kpi-grid" расположена в области "below header". Содержимое: "Total Retailers card"; "Active card"; "Total Lifetime Value card".
- Зона "retailer-ledger" расположена в области "main card region". Содержимое: "loading spinner or CRM empty state"; "table of retailers with avatar initials, lifetime value, order count, last order date, and status chip"; "pagination controls".
- Зона "retailer-detail-drawer" расположена в области "slide-out side drawer". Правило видимости: "visible when slideOpen is true". Содержимое: "retailer initials tile"; "status chip"; "contact links for phone and email"; "lifetime value and total orders KPI cards"; "order ledger list with state dot, item count, amount, and date".

### Controloverview

- Кнопка "Retailer row" расположена в "retailer-ledger body rows". Стиль: "full-row tap target opening detail drawer".

### Iconoverview

- Иконка "crm" используется в зоне "CRM empty state".
- Иконка "phone" используется в зоне "detail drawer contact row".
- Иконка "email" используется в зоне "detail drawer contact row".

### Flowoverview

**Flowid:** crm-bootstrap

**Summary:** Поток "crm-bootstrap" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Page reads supplier token and fetches /v1/supplier/crm/retailers".
- Шаг 2: "Returned retailer rows populate KPI cards and paginated ledger".

---

**Flowid:** detail-drawer-open

**Summary:** Поток "detail-drawer-open" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier clicks a retailer row".
- Шаг 2: "Drawer opens immediately in loading state".
- Шаг 3: "Page requests /v1/supplier/crm/retailers/{retailer_id}".
- Шаг 4: "If detail fetch fails, page falls back to a synthesized order-history sample based on the selected base record".

---

**Flowid:** contact-escalation

**Summary:** Поток "contact-escalation" содержит 1 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier can use tel: and mailto: links in the drawer contact section to escalate directly to retailer communications".

---


### Dependencyoverview

#### Reads

- /v1/supplier/crm/retailers
- /v1/supplier/crm/retailers/{retailer_id}

#### Writes


#### Localizednotes

- Чтение: "/v1/supplier/crm/retailers", "/v1/supplier/crm/retailers/{retailer_id}".
- Запись: нет.
- Модель обновления: "single fetch for retailer list plus on-demand detail fetch when a row is opened".

### Stateoverview

- Состояние: "table loading spinner state".
- Состояние: "CRM empty state".
- Состояние: "retailer table with pagination".
- Состояние: "detail drawer loading spinner".
- Состояние: "detail drawer with contact and order ledger".
- Состояние: "detail drawer empty order ledger".

### Figureoverview

- Фигура: "full CRM page with KPI grid and retailer ledger".
- Фигура: "retailer detail drawer over table background".
- Фигура: "drawer order-ledger close-up with state dots and amount column".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-dashboard" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-dashboard" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier analytics landing page combining forward-demand intelligence, SKU velocity summaries, financial KPIs, and volume-share tables.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "left: headline: Analytics; subtitle: Financial overview and operational intelligence"; "right: Dispatch Control Room link button with clipboard-style glyph".
- Зона "future-demand-card" расположена в области "below header when predictions exist". Правило видимости: "visible when demand payload exists and prediction_count > 0". Содержимое: "lightning icon badge"; "AI Future Demand headline and helper copy"; "prediction-count status chip"; "three-metric strip for retailers, pallets, forecast value"; "forecast item pills"; "View Advanced Analytics CTA".
- Зона "kpi-grid" расположена в области "below future-demand card". Содержимое: "Gross Volume card"; "Total Pallets Moved card"; "Avg Velocity per SKU card"; "Top Performing SKU card".
- Зона "velocity-chart" расположена в области "mid-page primary visualization". Содержимое: "VelocityChart component showing SKU volume performance".
- Зона "sku-breakdown-table" расположена в области "bottom card". Правило видимости: "visible when velocityData length > 0". Содержимое: "SKU ID column"; "pallet count column"; "gross volume column"; "share column with right-aligned progress bar and percentage text".

### Controloverview

- Кнопка "Dispatch Control Room" расположена в "header-right". Стиль: "filled pill link". Иконка: "manifest clipboard glyph".
- Кнопка "View Advanced Analytics" расположена в "future-demand-card footer". Стиль: "filled rounded CTA". Иконка: "right arrow".

### Iconoverview

- Иконка "manifest clipboard glyph" используется в зоне "header-right dispatch link".
- Иконка "lightning bolt" используется в зоне "future-demand-card leading badge".
- Иконка "right arrow" используется в зоне "future-demand-card CTA trailing edge".
- Иконка "progress fill bar" используется в зоне "share column in sku-breakdown-table".

### Flowoverview

**Flowid:** analytics-bootstrap

**Summary:** Поток "analytics-bootstrap" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Page reads supplier token from cookie".
- Шаг 2: "Page requests /v1/supplier/analytics/velocity and /v1/supplier/analytics/demand/today in parallel".
- Шаг 3: "KPI cards, chart, demand card, and table compute derived totals from returned payloads".
- Шаг 4: "If demand payload is absent, only analytics baseline regions remain".

---

**Flowid:** advanced-demand-drilldown

**Summary:** Поток "advanced-demand-drilldown" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier reviews AI demand summary card".
- Шаг 2: "Supplier clicks View Advanced Analytics".
- Шаг 3: "Page routes to /supplier/analytics/demand".

---

**Flowid:** dispatch-linkout

**Summary:** Поток "dispatch-linkout" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier uses header CTA".
- Шаг 2: "Page routes to /supplier/manifests legacy dispatch surface".

---


### Dependencyoverview

#### Reads

- /v1/supplier/analytics/velocity
- /v1/supplier/analytics/demand/today

#### Writes


#### Localizednotes

- Чтение: "/v1/supplier/analytics/velocity", "/v1/supplier/analytics/demand/today".
- Запись: нет.
- Модель обновления: "single fetch on mount; data remains static until navigation or reload".

### Stateoverview

- Состояние: "page-level loading skeleton with placeholder header, KPI cards, and chart block".
- Состояние: "unauthorized error card".
- Состояние: "analytics-only state with no demand card".
- Состояние: "full intelligence state with demand card and forecast pills".
- Состояние: "table hidden when velocityData is empty".

### Figureoverview

- Фигура: "full analytics dashboard with future-demand card and KPI grid".
- Фигура: "AI Future Demand card close-up with forecast pills".
- Фигура: "velocity chart region".
- Фигура: "SKU breakdown table with share bar column".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-depot-reconciliation" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-depot-reconciliation" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier depot-reconciliation page for processing quarantined returned loads by vehicle, order, and individual line item, with restock and write-off actions.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "headline: Depot Reconciliation"; "subtitle describing returned loads awaiting restock or write-off"; "Refresh button".
- Зона "vehicle-card-stack" расположена в области "main vertical stack". Содержимое: "one vehicle card per quarantined vehicle with header summary and nested order sections".
- Зона "vehicle-card-header" расположена в области "top of each vehicle card". Содержимое: "fleet icon avatar"; "vehicle class, driver name, route identifier, and order count"; "Restock All button"; "Write Off All button".
- Зона "order-section" расположена в области "within each vehicle card". Содержимое: "order short ID and retailer name"; "quarantine pill"; "item table with product, quantity, unit price, and per-item actions".

### Controloverview

- Кнопка "Retry" расположена в "error fallback state". Стиль: "secondary button".
- Кнопка "Refresh" расположена в "header top-right". Стиль: "outline button with leading icon".
- Кнопка "Restock All" расположена в "vehicle-card-header action cluster". Стиль: "secondary small button".
- Кнопка "Write Off All" расположена в "vehicle-card-header action cluster". Стиль: "outline danger small button".
- Кнопка "Restock" расположена в "item row actions". Стиль: "secondary small button".
- Кнопка "Write Off" расположена в "item row actions". Стиль: "outline danger small button".

### Iconoverview

- Иконка "error" используется в зоне "error fallback".
- Иконка "warehouse" используется в зоне "empty state".
- Иконка "refresh" используется в зоне "Refresh button".
- Иконка "fleet" используется в зоне "vehicle-card header avatar".

### Flowoverview

**Flowid:** quarantine-bootstrap

**Summary:** Поток "quarantine-bootstrap" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Page reads supplier token and fetches /v1/supplier/quarantine-stock".
- Шаг 2: "Loading placeholders occupy the card stack until quarantine vehicles resolve".

---

**Flowid:** bulk-vehicle-reconciliation

**Summary:** Поток "bulk-vehicle-reconciliation" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier presses Restock All or Write Off All in a vehicle card header".
- Шаг 2: "All line_item_ids for that vehicle are posted to /v1/inventory/reconcile-returns with the selected action".
- Шаг 3: "Toast confirms success and vehicle card stack reloads".

---

**Flowid:** item-level-reconciliation

**Summary:** Поток "item-level-reconciliation" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier presses Restock or Write Off in a line-item row".
- Шаг 2: "Single line_item_id is posted to /v1/inventory/reconcile-returns".
- Шаг 3: "Row and enclosing vehicle dataset refresh after reconciliation".

---


### Dependencyoverview

#### Reads

- /v1/supplier/quarantine-stock

#### Writes

- /v1/inventory/reconcile-returns

#### Localizednotes

- Чтение: "/v1/supplier/quarantine-stock".
- Запись: "/v1/inventory/reconcile-returns".
- Модель обновления: "load on mount, manual refresh button, and automatic reload after reconciliation actions".

### Stateoverview

- Состояние: "loading skeleton stack".
- Состояние: "error fallback with retry button".
- Состояние: "empty state with no quarantine stock".
- Состояние: "vehicle stack with nested order sections".
- Состояние: "action-in-progress state disabling reconciliation buttons".

### Figureoverview

- Фигура: "full depot reconciliation page with stacked vehicle cards".
- Фигура: "single vehicle card header with bulk action buttons".
- Фигура: "order section close-up with quarantine pill and per-item restock and write-off controls".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-dispatch" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-dispatch" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Legacy supplier dispatch route that immediately redirects to the canonical supplier orders surface rather than rendering a dedicated dispatch page.

### Layoutoverview

- Зона "redirect-guard" расположена в области "server-side route handler". Содержимое: "no persisted UI; redirect('/supplier/orders') executes during route resolution".

### Controloverview


### Iconoverview


### Flowoverview

**Flowid:** route-alias-redirect

**Summary:** Поток "route-alias-redirect" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier navigates to /supplier/dispatch".
- Шаг 2: "Next.js redirect executes immediately".
- Шаг 3: "Browser lands on /supplier/orders where dispatch actions are actually performed".

---


### Dependencyoverview

#### Reads


#### Writes


#### Localizednotes

- Чтение: нет.
- Запись: нет.
- Модель обновления: "no local data fetch; route delegates to /supplier/orders".

### Stateoverview

- Состояние: "immediate server redirect with no rendered intermediate page".

### Figureoverview

- Фигура: "route-flow figure showing /supplier/dispatch aliasing into /supplier/orders".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-fleet" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-fleet" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier fleet command center for driver provisioning, vehicle registration, assignment control, capacity visibility, and credential reveal for newly provisioned operators.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "left: back link to supplier dashboard; headline: Fleet Management; subtitle: Provision drivers, register vehicles, manage fleet capacity"; "right: conditional + Add Driver CTA when drivers tab active; conditional + Add Vehicle CTA when vehicles tab active".
- Зона "tab-selector" расположена в области "below header". Содержимое: "Drivers tab pill with count"; "Vehicles tab pill with count".
- Зона "kpi-row" расположена в области "below tab-selector". Содержимое: "driver metrics row when drivers tab active"; "vehicle metrics row when vehicles tab active".
- Зона "primary-table-region" расположена в области "main content card". Содержимое: "vehicle table with class, label, plate, capacity, assigned driver, status, actions when vehicles tab active"; "driver table with clickable rows, phone, type badge, assignment select, status when drivers tab active".
- Зона "driver-add-drawer" расположена в области "right slide-out overlay". Правило видимости: "open when showAdd is true". Содержимое: "name input"; "phone input"; "driver-type chip toggle"; "assign-vehicle select"; "license-plate input"; "inline error copy"; "Provision Driver button".
- Зона "pin-reveal-modal" расположена в области "centered modal overlay". Правило видимости: "visible when createdPin exists". Содержимое: "success icon disk"; "driver identity text"; "dashed login-pin panel"; "warning banner instructing copy-once behavior"; "Done button".
- Зона "driver-detail-drawer" расположена в области "right slide-out overlay". Правило видимости: "open when selectedDriver exists". Содержимое: "initial avatar circle"; "driver name and phone"; "type badge"; "detail cell grid"; "optional current-location row".
- Зона "vehicle-add-drawer" расположена в области "right slide-out overlay". Правило видимости: "open when showAddVehicle is true". Содержимое: "class-versus-dimensions mode toggle"; "vehicle-class select"; "computed capacity readout"; "dimension inputs when LxWxH mode selected"; "label input"; "license plate input"; "Register Vehicle button".

### Controloverview

- Кнопка "+ Add Driver" расположена в "header-right". Стиль: "primary". Правило видимости: "drivers tab active".
- Кнопка "+ Add Vehicle" расположена в "header-right". Стиль: "primary". Правило видимости: "vehicles tab active".
- Кнопка "Drivers tab" расположена в "tab-selector". Стиль: "segmented tab".
- Кнопка "Vehicles tab" расположена в "tab-selector". Стиль: "segmented tab".
- Кнопка "Deactivate" расположена в "vehicle row action cluster". Стиль: "ghost-danger". Правило видимости: "vehicle is active".
- Кнопка "Clear Returns" расположена в "vehicle row action cluster". Стиль: "outline-warning". Правило видимости: "vehicle active, assigned, and pending returns exist".
- Кнопка "Provision Driver" расположена в "driver-add-drawer footer". Стиль: "full-width primary".
- Кнопка "Done" расположена в "pin-reveal-modal footer". Стиль: "full-width primary".
- Кнопка "Class" расположена в "vehicle-add-drawer top-right mode toggle". Стиль: "segmented button".
- Кнопка "LxWxH" расположена в "vehicle-add-drawer top-right mode toggle". Стиль: "segmented button".
- Кнопка "Register Vehicle" расположена в "vehicle-add-drawer footer". Стиль: "full-width primary".

### Iconoverview

- Иконка "warning" используется в зоне "pin-reveal-modal warning banner".
- Иконка "success checkmark disk" используется в зоне "pin-reveal-modal top".
- Иконка "driver initial avatar" используется в зоне "driver-detail-drawer header".
- Иконка "driver-type badge" используется в зоне "driver rows and driver-detail-drawer header".

### Flowoverview

**Flowid:** driver-provisioning

**Summary:** Поток "driver-provisioning" содержит 5 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier stays on Drivers tab".
- Шаг 2: "Clicks + Add Driver".
- Шаг 3: "Completes drawer form and optionally preassigns vehicle".
- Шаг 4: "Page posts to /v1/supplier/fleet/drivers".
- Шаг 5: "Drawer closes and one-time PIN modal appears".

---

**Flowid:** vehicle-registration

**Summary:** Поток "vehicle-registration" содержит 5 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier switches to Vehicles tab".
- Шаг 2: "Clicks + Add Vehicle".
- Шаг 3: "Supplier either keeps class mode or enters dimensions for computed VU".
- Шаг 4: "Page posts to /v1/supplier/fleet/vehicles".
- Шаг 5: "Vehicle list reloads".

---

**Flowid:** assignment-control

**Summary:** Поток "assignment-control" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier changes assignment select inside a driver row".
- Шаг 2: "Page patches /v1/supplier/fleet/drivers/{driverId}/assign-vehicle".
- Шаг 3: "Drivers and vehicles refresh to reflect occupancy state".

---

**Flowid:** driver-inspection

**Summary:** Поток "driver-inspection" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier clicks a driver row".
- Шаг 2: "Page requests /v1/supplier/fleet/drivers/{id}".
- Шаг 3: "Right-side drawer opens with identity, stats, and current location when present".

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

- Чтение: "/v1/supplier/fleet/drivers", "/v1/supplier/fleet/vehicles", "/v1/supplier/fleet/capacity", "/v1/supplier/fleet/drivers/{id}".
- Запись: "/v1/supplier/fleet/drivers", "/v1/supplier/fleet/vehicles", "/v1/supplier/fleet/drivers/{driverId}/assign-vehicle", "/v1/supplier/fleet/vehicles/{vehicleId}", "/v1/vehicle/{vehicleId}/clear-returns".
- Модель обновления: "initial fetch on mount followed by targeted reloads after create, assign, deactivate, and clear-return actions".

### Stateoverview

- Состояние: "drivers-tab loading spinner".
- Состояние: "drivers empty state".
- Состояние: "vehicles empty state".
- Состояние: "drivers table with assignment select".
- Состояние: "vehicles table with action chips".
- Состояние: "add-driver drawer open".
- Состояние: "add-vehicle drawer open in class mode".
- Состояние: "add-vehicle drawer open in dimension mode with computed VU".
- Состояние: "PIN reveal modal".
- Состояние: "driver detail drawer open".

### Figureoverview

- Фигура: "fleet page on drivers tab with KPI row and assignment table".
- Фигура: "fleet page on vehicles tab with capacity metrics and vehicle table".
- Фигура: "add-driver drawer".
- Фигура: "PIN reveal modal".
- Фигура: "driver detail drawer".
- Фигура: "add-vehicle drawer in dimensions mode".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-inventory" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-inventory" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier stock-control page for quantity adjustments, low-stock visibility, and immutable audit-log inspection.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "headline: Inventory Management"; "subtitle: Stock levels, replenishment controls, and audit trail".
- Зона "tab-switcher" расположена в области "below header". Содержимое: "Stock Levels segmented chip"; "Audit Log segmented chip".
- Зона "stock-card" расположена в области "primary card when stock tab active". Содержимое: "SKU count label"; "Refresh button"; "inventory table with product, sku, stock, action columns"; "inline adjustment editor replacing action cell when row is in adjust mode"; "pagination controls".
- Зона "audit-card" расположена в области "primary card when audit tab active". Содержимое: "Last 100 adjustments label"; "audit table with product, prev, delta, new, reason, date columns"; "pagination controls".

### Controloverview

- Кнопка "Stock Levels" расположена в "tab-switcher". Стиль: "segmented chip".
- Кнопка "Audit Log" расположена в "tab-switcher". Стиль: "segmented chip".
- Кнопка "Refresh" расположена в "stock-card header-right". Стиль: "outline".
- Кнопка "Adjust" расположена в "stock row action cell". Стиль: "outline small".
- Кнопка "Apply" расположена в "inline adjustment editor". Стиль: "primary small".
- Кнопка "Cancel" расположена в "inline adjustment editor". Стиль: "text small".

### Iconoverview

- Иконка "inventory" используется в зоне "stock empty state".
- Иконка "ledger" используется в зоне "audit empty state".
- Иконка "reason chip" используется в зоне "audit reason column".

### Flowoverview

**Flowid:** inventory-bootstrap

**Summary:** Поток "inventory-bootstrap" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Page reads token via useToken".
- Шаг 2: "Page requests /v1/supplier/inventory and /v1/supplier/inventory/audit".
- Шаг 3: "Stock tab renders by default".

---

**Flowid:** quantity-adjustment

**Summary:** Поток "quantity-adjustment" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier clicks Adjust on a row".
- Шаг 2: "Action cell expands into delta input, reason select, Apply, and Cancel controls".
- Шаг 3: "Page patches /v1/supplier/inventory with adjustment payload".
- Шаг 4: "Page refreshes both stock and audit datasets".

---

**Flowid:** audit-review

**Summary:** Поток "audit-review" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier switches to Audit Log tab".
- Шаг 2: "Page displays signed delta values, reason chip, and timestamped adjustments".

---


### Dependencyoverview

#### Reads

- /v1/supplier/inventory
- /v1/supplier/inventory/audit

#### Writes

- /v1/supplier/inventory

#### Localizednotes

- Чтение: "/v1/supplier/inventory", "/v1/supplier/inventory/audit".
- Запись: "/v1/supplier/inventory".
- Модель обновления: "load on mount plus manual refresh button and automatic re-fetch after successful adjustments".

### Stateoverview

- Состояние: "unauthorized supplier-required card".
- Состояние: "stock-tab loading state".
- Состояние: "stock empty state".
- Состояние: "normal stock table".
- Состояние: "row-level inline adjustment mode".
- Состояние: "audit empty state".
- Состояние: "audit table with positive and negative delta coloring".

### Figureoverview

- Фигура: "inventory page on stock tab".
- Фигура: "stock row in inline adjust mode".
- Фигура: "inventory page on audit tab".
- Фигура: "audit table close-up with reason chip and signed delta".

---

**Dossierfile:** web-supplier-login.json

**Pageid:** web-auth-login

**Route:** /auth/login

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Sourcefile:** apps/admin-portal/app/auth/login/page.tsx

**Entrytype:** page

**Localizedsummary:** Локализованный обзор поверхности "web-auth-login" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-auth-login" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier portal sign-in page with credential form, stale-cookie clearing, inline error handling, and password visibility toggle.

### Layoutoverview

- Зона "mobile-brand-strip" расположена в области "top on mobile only". Содержимое: "brand icon tile"; "Pegasus Hub title"; "Supplier Operations Portal subtitle".
- Зона "login-card" расположена в области "central card". Содержимое: "Sign in headline"; "supporting subtitle"; "optional inline error alert"; "email field"; "password field with show-hide button"; "primary Sign In button"; "link to create account".
- Зона "mobile-footer-copy" расположена в области "bottom on mobile only". Содержимое: "The Lab Industries copyright text".

### Controloverview

- Кнопка "password visibility toggle" расположена в "inside password field trailing edge". Стиль: "icon button".
- Кнопка "Sign In" расположена в "login-card footer". Стиль: "full-width primary CTA".
- Кнопка "Create account link" расположена в "below primary CTA". Стиль: "inline text link".

### Iconoverview

- Иконка "brand warehouse glyph" используется в зоне "mobile-brand-strip".
- Иконка "error alert icon" используется в зоне "inline alert row".
- Иконка "eye or eye-off glyph" используется в зоне "password visibility toggle".
- Иконка "spinner" используется в зоне "Sign In button loading state".

### Flowoverview

**Flowid:** credential-login

**Summary:** Поток "credential-login" содержит 5 шаг(а/ов).

#### Steps

- Шаг 1: "Page clears stale auth cookies on mount".
- Шаг 2: "User enters email and password".
- Шаг 3: "User submits Sign In".
- Шаг 4: "Page posts to /v1/auth/admin/login".
- Шаг 5: "Successful response writes cookies and routes user to /".

---

**Flowid:** password-peek

**Summary:** Поток "password-peek" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "User taps trailing password eye control".
- Шаг 2: "Password field switches between masked and plain text state".

---


### Stateoverview

- Состояние: "idle form".
- Состояние: "inline error state".
- Состояние: "submitting state with spinner".

### Figureoverview

- Фигура: "full login page".
- Фигура: "login card close-up".
- Фигура: "password field with visibility toggle".
- Фигура: "error alert state".
- Фигура: "submitting CTA state".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-manifests" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-manifests" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Legacy supplier manifests route that immediately redirects to the canonical supplier orders surface instead of rendering standalone UI.

### Layoutoverview

- Зона "redirect-guard" расположена в области "server-side route handler". Содержимое: "no persisted UI; redirect('/supplier/orders') executes during route resolution".

### Controloverview


### Iconoverview


### Flowoverview

**Flowid:** route-alias-redirect

**Summary:** Поток "route-alias-redirect" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier navigates to /supplier/manifests".
- Шаг 2: "Next.js redirect executes immediately".
- Шаг 3: "Browser lands on /supplier/orders where the actual manifest and order workflow resides".

---


### Dependencyoverview

#### Reads


#### Writes


#### Localizednotes

- Чтение: нет.
- Запись: нет.
- Модель обновления: "no local data fetch; route delegates to /supplier/orders".

### Stateoverview

- Состояние: "immediate server redirect with no rendered intermediate page".

### Figureoverview

- Фигура: "route-flow figure showing /supplier/manifests aliasing into /supplier/orders".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-onboarding" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-onboarding" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Deprecated supplier onboarding route retained as a transitional redirect because onboarding is now fully embedded into the registration wizard at /auth/register.

### Layoutoverview

- Зона "redirect-indicator" расположена в области "centered full-screen state". Содержимое: "spinner glyph"; "status text: Redirecting to dashboard…".

### Controloverview


### Iconoverview

- Иконка "spinner glyph" используется в зоне "centered redirect-indicator".

### Flowoverview

**Flowid:** conditional-redirect

**Summary:** Поток "conditional-redirect" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Client effect reads supplier token from cookie".
- Шаг 2: "If token is absent, router.replace('/auth/register') executes".
- Шаг 3: "If token is present, router.replace('/supplier/dashboard') executes".

---


### Dependencyoverview

#### Reads


#### Writes


#### Localizednotes

- Чтение: нет.
- Запись: нет.
- Модель обновления: "no network fetch; client-side token check determines redirect target".

### Stateoverview

- Состояние: "transient redirect indicator before navigation resolves".
- Состояние: "redirect to registration wizard when unauthenticated".
- Состояние: "redirect to supplier dashboard when authenticated".

### Figureoverview

- Фигура: "full-screen redirect indicator with spinner and redirect text".
- Фигура: "route-flow figure showing /supplier/onboarding branching to /auth/register or /supplier/dashboard".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-payment-config" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-payment-config" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier gateway-credential administration page for Click, Payme, and Global Pay, supporting manual onboarding, activation-state review, update, and deactivation.

### Layoutoverview

- Зона "header" расположена в области "top constrained column". Содержимое: "headline: Payment Gateways"; "subtitle explaining Click, Payme, and Global Pay configuration".
- Зона "toast-region" расположена в области "below header when toast exists". Правило видимости: "visible when toast state is non-null". Содержимое: "success or error colored banner"; "leading status icon"; "toast message".
- Зона "provider-stack" расположена в области "vertical list of provider cards". Содержимое: "one card per gateway capability"; "icon tile"; "display name"; "active or not-configured chip"; "merchant or service preview text when configured"; "connect placeholder badge or button"; "manual-setup or update button"; "optional deactivate button".
- Зона "expanded-manual-form" расположена в области "inside provider card below divider". Правило видимости: "visible when expandedGateway matches card gateway". Содержимое: "shield-led security hint"; "merchant-id input"; "optional service-id input"; "secret-key password input"; "per-field helper copy"; "Cancel and Save or Update Configuration buttons".
- Зона "empty-state" расположена в области "center card". Правило видимости: "visible when no capabilities and no configs are returned". Содержимое: "payment icon"; "No payment gateways available headline"; "administrator-support body copy".

### Controloverview

- Кнопка "Connect" расположена в "provider-card action cluster". Стиль: "primary small". Правило видимости: "provider supports redirect onboarding".
- Кнопка "Connect coming soon badge" расположена в "provider-card action cluster". Стиль: "disabled status badge". Правило видимости: "manual-only provider".
- Кнопка "Manual setup" расположена в "provider-card action cluster". Стиль: "outline or primary small". Правило видимости: "provider not configured".
- Кнопка "Update" расположена в "provider-card action cluster". Стиль: "outline or primary small". Правило видимости: "provider configured".
- Кнопка "Deactivate" расположена в "provider-card action cluster". Стиль: "danger-soft small". Правило видимости: "config active".
- Кнопка "Cancel" расположена в "expanded-manual-form footer-left". Стиль: "outline".
- Кнопка "Save Configuration" расположена в "expanded-manual-form footer-right". Стиль: "primary". Правило видимости: "new config".
- Кнопка "Update Configuration" расположена в "expanded-manual-form footer-right". Стиль: "primary". Правило видимости: "editing existing config".

### Iconoverview

- Иконка "gateway svg badge" используется в зоне "provider-card leading tile".
- Иконка "check-circle" используется в зоне "active status chip and success toast".
- Иконка "x-circle" используется в зоне "error toast".
- Иконка "clock" используется в зоне "not-configured chip".
- Иконка "shield" используется в зоне "manual-form helper lines".
- Иконка "key-round" используется в зоне "manual-setup toggle button".
- Иконка "chevron-down or chevron-up" используется в зоне "manual-setup toggle button trailing edge".
- Иконка "link2" используется в зоне "connect action".

### Flowoverview

**Flowid:** gateway-bootstrap

**Summary:** Поток "gateway-bootstrap" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Page requests /v1/supplier/payment-config".
- Шаг 2: "Configured gateways and provider capabilities render as stacked cards".
- Шаг 3: "Merchant and service previews appear without secret prefill".

---

**Flowid:** manual-credential-save

**Summary:** Поток "manual-credential-save" содержит 5 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier expands Manual setup or Update on a gateway card".
- Шаг 2: "Page seeds merchant and service values from existing config when present".
- Шаг 3: "Supplier enters required fields".
- Шаг 4: "Page posts to /v1/supplier/payment-config".
- Шаг 5: "Success toast appears and cards reload".

---

**Flowid:** gateway-deactivation

**Summary:** Поток "gateway-deactivation" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier clicks Deactivate on an active gateway card".
- Шаг 2: "Page deletes through /v1/supplier/payment-config with config_id payload".
- Шаг 3: "Success toast appears and list refreshes".

---


### Dependencyoverview

#### Reads

- /v1/supplier/payment-config

#### Writes

- /v1/supplier/payment-config

#### Localizednotes

- Чтение: "/v1/supplier/payment-config".
- Запись: "/v1/supplier/payment-config".
- Модель обновления: "initial fetch on mount plus reload after save and deactivate operations".

### Stateoverview

- Состояние: "loading spinner row".
- Состояние: "provider stack with all cards collapsed".
- Состояние: "expanded manual form for Click".
- Состояние: "expanded manual form for Payme".
- Состояние: "expanded manual form for Global Pay with service-id helper text".
- Состояние: "success toast state".
- Состояние: "error toast state".
- Состояние: "no-capabilities empty state".

### Figureoverview

- Фигура: "full payment gateway stack".
- Фигура: "configured gateway card close-up".
- Фигура: "expanded Global Pay manual form".
- Фигура: "success toast over gateway stack".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-pricing" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-pricing" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier pricing-engine page for composing volume-discount rules and auditing the currently active pricing rule ledger.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "headline: Pricing Engine"; "subtitle: B2B Volume Discount Rules — Upsert and Manage".
- Зона "primary-grid" расположена в области "two-column workspace with wider table column". Содержимое: "left form panel for new pricing rule composition"; "right rules ledger card with table or empty state".
- Зона "rule-form-panel" расположена в области "left column". Содержимое: "SKU selector or fallback text input"; "Min Pallets numeric input"; "Discount percent numeric input with helper copy"; "Target Retailer Tier chip row"; "Valid Until datetime input"; "Tier ID input"; "Lock Pricing Rule CTA".
- Зона "rules-ledger-card" расположена в области "right column". Содержимое: "rules count header"; "loading message or pricing empty state"; "rules table with SKU, pallet threshold, discount chip, retailer tier, expiry, status, and actions".

### Controloverview

- Кнопка "Target Retailer Tier chip" расположена в "rule-form-panel target tier row". Стиль: "chip toggle".
- Кнопка "Lock Pricing Rule" расположена в "rule-form-panel footer". Стиль: "full-width primary button".
- Кнопка "Deactivate" расположена в "rules-ledger actions column". Стиль: "small danger-tinted button". Правило видимости: "rule is active".

### Iconoverview

- Иконка "pricing" используется в зоне "rules empty state".

### Flowoverview

**Flowid:** pricing-bootstrap

**Summary:** Поток "pricing-bootstrap" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Page obtains supplier token".
- Шаг 2: "Page fetches /v1/supplier/pricing/rules and /v1/supplier/products in parallel".
- Шаг 3: "SKU selector options and rules ledger render from those responses".

---

**Flowid:** rule-composition

**Summary:** Поток "rule-composition" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier chooses SKU, pallet threshold, discount percent, retailer tier, expiry, and optional tier ID".
- Шаг 2: "Submit generates UUID when tier_id is blank".
- Шаг 3: "Page posts the assembled rule to /v1/supplier/pricing/rules".
- Шаг 4: "Success resets the form and refreshes the rules ledger".

---

**Flowid:** rule-deactivation

**Summary:** Поток "rule-deactivation" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier presses Deactivate on an active rule row".
- Шаг 2: "Page deletes /v1/supplier/pricing/rules/{tier_id}".
- Шаг 3: "Row status transitions from active to inactive after refresh".

---


### Dependencyoverview

#### Reads

- /v1/supplier/pricing/rules
- /v1/supplier/products

#### Writes

- /v1/supplier/pricing/rules
- /v1/supplier/pricing/rules/{tier_id}

#### Localizednotes

- Чтение: "/v1/supplier/pricing/rules", "/v1/supplier/products".
- Запись: "/v1/supplier/pricing/rules", "/v1/supplier/pricing/rules/{tier_id}".
- Модель обновления: "load on mount and reload after successful create or deactivate actions".

### Stateoverview

- Состояние: "rules loading message state".
- Состояние: "empty rules ledger state".
- Состояние: "form with product select populated".
- Состояние: "form submit in locking state".
- Состояние: "rules table with active and inactive badges".
- Состояние: "row-level deactivation pending state".

### Figureoverview

- Фигура: "full pricing-engine page with form panel and rules table".
- Фигура: "rule composition form close-up showing tier chips and lock CTA".
- Фигура: "rules ledger row showing discount chip, expiry, and deactivate action".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-product-detail" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-product-detail" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier per-SKU detail workspace for inspecting metadata, editing commercial fields, and adjusting logistics constraints and activation status.

### Layoutoverview

- Зона "back-nav" расположена в области "top-left above header". Содержимое: "Back to Products text button with arrow icon".
- Зона "header-row" расположена в области "top full-width below back-nav". Содержимое: "thumbnail block"; "title, status pill, category label, and SKU metadata"; "action cluster with activate or deactivate button and edit-mode controls".
- Зона "save-message" расположена в области "below header when saveMsg exists". Правило видимости: "visible after save or no-op response". Содержимое: "success or error message banner".
- Зона "detail-grid" расположена в области "two-column main region". Содержимое: "Product Details card with name, description, image URL, and base price"; "Logistics and Ordering card with MOQ, step size, block settings, volumetric unit, dimensions, and created date".

### Controloverview

- Кнопка "Back to Products" расположена в "back-nav". Стиль: "text button with leading icon".
- Кнопка "Deactivate" расположена в "header-row action cluster". Стиль: "outline button". Правило видимости: "product is active".
- Кнопка "Activate" расположена в "header-row action cluster". Стиль: "outline button". Правило видимости: "product is inactive".
- Кнопка "Edit Product" расположена в "header-row action cluster". Стиль: "primary button". Правило видимости: "editing is false".
- Кнопка "Cancel" расположена в "header-row action cluster". Стиль: "outline button". Правило видимости: "editing is true".
- Кнопка "Save Changes" расположена в "header-row action cluster". Стиль: "primary button". Правило видимости: "editing is true".

### Iconoverview

- Иконка "arrow_back" используется в зоне "Back to Products control".
- Иконка "image" используется в зоне "thumbnail placeholder when product image is absent".

### Flowoverview

**Flowid:** detail-bootstrap

**Summary:** Поток "detail-bootstrap" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Page reads sku_id from route params and supplier token from auth context".
- Шаг 2: "Page requests /v1/supplier/products/{sku_id}".
- Шаг 3: "Fetched data populates the read-only detail view and edit draft state".

---

**Flowid:** edit-session

**Summary:** Поток "edit-session" содержит 5 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier presses Edit Product".
- Шаг 2: "All editable fields switch to inputs or textarea controls".
- Шаг 3: "Supplier modifies commercial or logistics values".
- Шаг 4: "Page computes a diff against original product state".
- Шаг 5: "PUT request to /v1/supplier/products/{sku_id} persists changed fields only".

---

**Flowid:** activation-toggle

**Summary:** Поток "activation-toggle" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier presses Activate or Deactivate in the header action cluster".
- Шаг 2: "Page submits an is_active update to /v1/supplier/products/{sku_id}".
- Шаг 3: "Detail view reloads and the status pill flips".

---

**Flowid:** save-feedback

**Summary:** Поток "save-feedback" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "After save, page emits a success or error banner beneath the header".
- Шаг 2: "No-change saves collapse edit mode with informational confirmation".

---


### Dependencyoverview

#### Reads

- /v1/supplier/products/{sku_id}

#### Writes

- /v1/supplier/products/{sku_id}

#### Localizednotes

- Чтение: "/v1/supplier/products/{sku_id}".
- Запись: "/v1/supplier/products/{sku_id}".
- Модель обновления: "load on mount and reload after successful save or activation changes".

### Stateoverview

- Состояние: "page loading spinner".
- Состояние: "error or not-found card with back button".
- Состояние: "default read-only detail mode".
- Состояние: "edit mode with form controls in both cards".
- Состояние: "saving state with disabled buttons".
- Состояние: "success message banner".
- Состояние: "error message banner".

### Figureoverview

- Фигура: "full product-detail page in read-only mode".
- Фигура: "product-detail page in edit mode with both cards active".
- Фигура: "header close-up showing thumbnail, status pill, SKU, and action cluster".
- Фигура: "logistics card close-up showing MOQ, step size, units-per-block, and dimensions".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-products" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-products" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier product-portfolio page for SKU search, category filtering, activation toggling, and entry into per-product detail editing.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "headline: My Products"; "subtitle with registered SKU count"; "Add Product CTA linking to supplier catalog".
- Зона "kpi-strip" расположена в области "below header". Содержимое: "Total SKUs card"; "Active card"; "Inactive card"; "Catalog Value card".
- Зона "control-row" расположена в области "below KPI strip". Содержимое: "left search input with embedded search icon"; "right Refresh button".
- Зона "category-chip-row" расположена в области "below control row when categoryOptions exist". Содержимое: "All chip with total count"; "per-category chips with counts".
- Зона "product-grid" расположена в области "main content grid". Содержимое: "responsive product cards with image region, status badge, category pill, title, description, price block, activation icon button, and SKU footer"; "empty state when no products match filters".

### Controloverview

- Кнопка "Add Product" расположена в "header top-right". Стиль: "accent filled link button".
- Кнопка "Refresh" расположена в "control-row right". Стиль: "outline button with refresh icon".
- Кнопка "All" расположена в "category-chip-row". Стиль: "filter chip".
- Кнопка "Category chip" расположена в "category-chip-row". Стиль: "filter chip".
- Кнопка "Deactivate" расположена в "product card bottom-right icon button". Стиль: "round danger-tinted icon button". Правило видимости: "product is active".
- Кнопка "Activate" расположена в "product card bottom-right icon button". Стиль: "round success-tinted icon button". Правило видимости: "product is inactive".

### Iconoverview

- Иконка "add" используется в зоне "Add Product CTA leading icon".
- Иконка "search" используется в зоне "search field left inset".
- Иконка "refresh" используется в зоне "Refresh button leading icon".
- Иконка "image placeholder glyph" используется в зоне "product card media region when image_url missing".
- Иконка "visibility_off" используется в зоне "active product toggle button".
- Иконка "visibility" используется в зоне "inactive product toggle button".
- Иконка "catalog" используется в зоне "empty state".

### Flowoverview

**Flowid:** products-bootstrap

**Summary:** Поток "products-bootstrap" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Page reads supplier token".
- Шаг 2: "Page fetches /v1/supplier/products and /v1/supplier/profile in parallel".
- Шаг 3: "When operating categories exist, page maps them against /v1/catalog/platform-categories".
- Шаг 4: "KPI strip and filtered grid derive from the resulting product dataset".

---

**Flowid:** search-and-filter

**Summary:** Поток "search-and-filter" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier types in the search field or taps a category chip".
- Шаг 2: "Grid filters client-side by category, SKU, product name, and description".
- Шаг 3: "KPI totals recompute from the filtered list".

---

**Flowid:** product-detail-navigation

**Summary:** Поток "product-detail-navigation" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier clicks a product card".
- Шаг 2: "Router pushes to /supplier/products/{sku_id}".
- Шаг 3: "Per-product detail workspace opens".

---

**Flowid:** activation-toggle

**Summary:** Поток "activation-toggle" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier presses the card-level activation icon button".
- Шаг 2: "Page puts new is_active value to /v1/supplier/products/{sku_id}".
- Шаг 3: "Products dataset reloads and card opacity/status badge update".

---


### Dependencyoverview

#### Reads

- /v1/supplier/products
- /v1/supplier/profile
- /v1/catalog/platform-categories

#### Writes

- /v1/supplier/products/{sku_id}

#### Localizednotes

- Чтение: "/v1/supplier/products", "/v1/supplier/profile", "/v1/catalog/platform-categories".
- Запись: "/v1/supplier/products/{sku_id}".
- Модель обновления: "load on mount, manual refresh button, and automatic reload after activation changes".

### Stateoverview

- Состояние: "page loading spinner".
- Состояние: "error card state".
- Состояние: "grid empty state with no products yet".
- Состояние: "grid empty state with no search matches".
- Состояние: "filtered product grid with mixed active and inactive cards".
- Состояние: "row-level activation toggle pending spinner".

### Figureoverview

- Фигура: "full products page with header, KPI strip, filters, and grid".
- Фигура: "single product card showing status badge and activation icon button".
- Фигура: "empty-state view with search field and refresh button retained".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-profile" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-profile" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier profile and operations-account page for managing identity, warehouse, banking, category, and shift-status data in a sectioned desktop form layout.

### Layoutoverview

- Зона "error-banner" расположена в области "top of page when error exists". Правило видимости: "visible when an error is present". Содержимое: "error icon"; "error text".
- Зона "hero-header" расположена в области "top full-width card". Содержимое: "circular warehouse avatar"; "supplier name, configuration status, category-email-phone summary"; "edit action cluster".
- Зона "company-details-section" расположена в области "below hero header". Содержимое: "section title with accent border"; "two-column card containing company and billing fields".
- Зона "warehouse-section" расположена в области "below company details". Содержимое: "warehouse address field"; "latitude and longitude read-only fields".
- Зона "banking-section" расположена в области "below warehouse section". Содержимое: "bank name field"; "account number field"; "card number field"; "payment gateway field".
- Зона "operating-categories-section" расположена в области "below banking when categories exist". Правило видимости: "visible when operating_categories is non-empty". Содержимое: "section title"; "category chips".
- Зона "shift-status-section" расположена в области "bottom card". Содержимое: "status dot"; "shift-state text"; "manual override label when manual_off_shift is true".

### Controloverview

- Кнопка "Retry" расположена в "error fallback screen". Стиль: "secondary button". Правило видимости: "profile failed to load and profile is absent".
- Кнопка "Edit Profile" расположена в "hero-header top-right". Стиль: "primary button with leading edit icon". Правило видимости: "editing is false".
- Кнопка "Cancel" расположена в "hero-header top-right". Стиль: "outline button". Правило видимости: "editing is true".
- Кнопка "Save" расположена в "hero-header top-right". Стиль: "primary button". Правило видимости: "editing is true".

### Iconoverview

- Иконка "error" используется в зоне "error banner and error fallback".
- Иконка "warehouse" используется в зоне "hero avatar".
- Иконка "verified" используется в зоне "configuration status line in hero header".
- Иконка "edit" используется в зоне "Edit Profile button".

### Flowoverview

**Flowid:** profile-bootstrap

**Summary:** Поток "profile-bootstrap" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Page requests /v1/supplier/profile through apiFetch".
- Шаг 2: "Skeleton placeholders render until the profile payload resolves".
- Шаг 3: "Resolved data populates hero, section cards, and status chips".

---

**Flowid:** edit-initialization

**Summary:** Поток "edit-initialization" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier presses Edit Profile".
- Шаг 2: "Page copies editable fields into a draft object".
- Шаг 3: "Read-only display cells switch to outline inputs".

---

**Flowid:** profile-save

**Summary:** Поток "profile-save" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Page diffs the draft against the fetched profile".
- Шаг 2: "Changed fields only are submitted in a PUT request to /v1/supplier/profile".
- Шаг 3: "Profile reloads after save and the page returns to read-only mode".

---

**Flowid:** edit-cancel

**Summary:** Поток "edit-cancel" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier presses Cancel during editing".
- Шаг 2: "Draft state clears and read-only cards restore without network writes".

---


### Dependencyoverview

#### Reads

- /v1/supplier/profile

#### Writes

- /v1/supplier/profile

#### Localizednotes

- Чтение: "/v1/supplier/profile".
- Запись: "/v1/supplier/profile".
- Модель обновления: "load on mount, retry on failure, and reload after successful profile updates".

### Stateoverview

- Состояние: "skeleton loading state".
- Состояние: "hard error fallback with retry button".
- Состояние: "inline error banner above populated profile".
- Состояние: "default read-only profile sections".
- Состояние: "edit mode with outlined field inputs".
- Состояние: "saving state with disabled save button".
- Состояние: "operating categories section visible".
- Состояние: "manual off-shift marker visible".

### Figureoverview

- Фигура: "full supplier profile page with hero header and stacked sections".
- Фигура: "hero header close-up showing avatar, configuration badge, and edit controls".
- Фигура: "company-details section in edit mode with input fields".
- Фигура: "shift-status card with active or off-shift indicator".

---

**Dossierfile:** web-supplier-register.json

**Pageid:** web-auth-register

**Route:** /auth/register

**Platform:** web

**Role:** SUPPLIER

**Status:** implemented

**Sourcefile:** apps/admin-portal/app/auth/register/page.tsx

**Entrytype:** page

**Localizedsummary:** Локализованный обзор поверхности "web-auth-register" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-auth-register" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Four-step supplier registration wizard combining account identity, warehouse location, business and fleet profile, category selection, and payment gateway preference.

### Layoutoverview

- Зона "mobile-brand-strip" расположена в области "top on mobile only". Содержимое: "brand icon tile"; "Pegasus Hub title"; "Supplier Registration subtitle".
- Зона "step-indicator" расположена в области "above main card". Содержимое: "Account step node"; "Location step node"; "Business step node"; "Payments step node".
- Зона "wizard-card" расположена в области "central card". Содержимое: "step headline and subtitle"; "optional inline error alert"; "step-specific form body"; "Back and Continue/Create Account buttons"; "Sign in link on step 1".

### Controloverview

- Кнопка "Locate" расположена в "step 2 location field trailing action". Стиль: "secondary inline CTA".
- Кнопка "category chips" расположена в "step 3 category grid". Стиль: "multi-select chip".
- Кнопка "cold-chain toggle" расположена в "step 3 fleet profile card". Стиль: "switch-like toggle".
- Кнопка "payment gateway rows" расположена в "step 4 payment list". Стиль: "full-width selectable card rows".
- Кнопка "Back" расположена в "wizard footer left". Стиль: "outline CTA when step > 0".
- Кнопка "Continue to next step" расположена в "wizard footer primary slot". Стиль: "full-width primary CTA on non-final steps".
- Кнопка "Create Supplier Account" расположена в "wizard footer primary slot". Стиль: "full-width primary CTA on final step".
- Кнопка "Sign in link" расположена в "step 1 footer text". Стиль: "inline text link".

### Iconoverview

- Иконка "brand warehouse glyph" используется в зоне "mobile-brand-strip".
- Иконка "step icons and checkmark states" используется в зоне "step indicator nodes".
- Иконка "Locate spinner or location glyph" используется в зоне "step 2 Locate button".
- Иконка "category checkmark" используется в зоне "selected category chips".
- Иконка "gateway icons" используется в зоне "step 4 payment gateway rows".
- Иконка "spinner" используется в зоне "Create Supplier Account submitting state".

### Flowoverview

**Flowid:** wizard-progression

**Summary:** Поток "wizard-progression" содержит 6 шаг(а/ов).

#### Steps

- Шаг 1: "User completes account step".
- Шаг 2: "User advances to location step".
- Шаг 3: "User advances to business step with category selection".
- Шаг 4: "User advances to payment step".
- Шаг 5: "User submits registration".
- Шаг 6: "On success cookies are written and user is routed to /supplier/dashboard".

---

**Flowid:** geolocation-capture

**Summary:** Поток "geolocation-capture" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "User presses Locate on step 2".
- Шаг 2: "Browser geolocation retrieves coordinates".
- Шаг 3: "Page reverse-geocodes via Nominatim when possible".
- Шаг 4: "Address and lat/lng fields are populated".

---

**Flowid:** category-and-gateway-selection

**Summary:** Поток "category-and-gateway-selection" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "User filters or browses category chips".
- Шаг 2: "User selects one or more categories".
- Шаг 3: "User selects one payment gateway card row as active".

---


### Stateoverview

- Состояние: "step 1 account fields".
- Состояние: "step 2 location fields".
- Состояние: "step 3 business and category selection".
- Состояние: "step 4 payment gateway selection".
- Состояние: "inline validation error state".
- Состояние: "Create Account submitting state".

### Figureoverview

- Фигура: "full registration wizard step 1".
- Фигура: "step indicator close-up".
- Фигура: "location step with Locate action".
- Фигура: "business step category grid".
- Фигура: "payment step gateway rows".
- Фигура: "final submitting state".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-returns" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-returns" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier dispute-resolution page for reviewing rejected or damaged line items and resolving them as write-offs or returns to stock.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "headline: Dispute and Returns"; "subtitle describing write-off versus return-to-stock resolution intent".
- Зона "summary-strip" расположена в области "below header". Содержимое: "Open Returns metric card"; "Total Damage Value metric card".
- Зона "returns-card-header" расположена в области "top of main card". Содержимое: "Damaged and Rejected Items label"; "Refresh button".
- Зона "returns-ledger" расположена в области "main card body". Содержимое: "loading message or empty state"; "paginated return-item rows with retailer, quantity, value, order reference, and action cluster".
- Зона "inline-resolution-cluster" расположена в области "row action area when resolvingId matches row". Правило видимости: "visible for the selected line item". Содержимое: "resolution select"; "notes input"; "Resolve button"; "dismiss x control".

### Controloverview

- Кнопка "Refresh" расположена в "returns-card-header right". Стиль: "outline button".
- Кнопка "Resolve" расположена в "row action column". Стиль: "outline small button".
- Кнопка "Resolve" расположена в "inline-resolution-cluster". Стиль: "small primary button". Правило видимости: "resolution editor is open".
- Кнопка "x" расположена в "inline-resolution-cluster trailing edge". Стиль: "small text dismiss control". Правило видимости: "resolution editor is open".

### Iconoverview

- Иконка "returns" используется в зоне "empty state".

### Flowoverview

**Flowid:** returns-bootstrap

**Summary:** Поток "returns-bootstrap" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Page reads supplier token".
- Шаг 2: "Page requests /v1/supplier/returns".
- Шаг 3: "Summary cards derive from returned line items".

---

**Flowid:** resolution-open

**Summary:** Поток "resolution-open" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier presses Resolve on a row".
- Шаг 2: "Action cell expands into resolution select, notes field, confirm control, and dismiss control".

---

**Flowid:** resolution-submit

**Summary:** Поток "resolution-submit" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier selects WRITE_OFF or RETURN_TO_STOCK and optionally enters notes".
- Шаг 2: "Page posts to /v1/supplier/returns/resolve with line_item_id, resolution, and notes".
- Шаг 3: "Successful resolution clears the inline editor and reloads the ledger".

---

**Flowid:** pagination-review

**Summary:** Поток "pagination-review" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier moves through paginated rows using shared pagination controls".
- Шаг 2: "Visible rows update while metric cards remain global".

---


### Dependencyoverview

#### Reads

- /v1/supplier/returns

#### Writes

- /v1/supplier/returns/resolve

#### Localizednotes

- Чтение: "/v1/supplier/returns".
- Запись: "/v1/supplier/returns/resolve".
- Модель обновления: "load on mount, manual refresh button, and automatic reload after resolution".

### Stateoverview

- Состояние: "unauthorized supplier-required card".
- Состояние: "returns loading state".
- Состояние: "returns empty state".
- Состояние: "returns ledger list state".
- Состояние: "inline resolution editor open".
- Состояние: "row-level resolution submit pending state".

### Figureoverview

- Фигура: "full returns page with summary cards and paginated ledger".
- Фигура: "return row in inline resolution mode with dropdown and notes field".
- Фигура: "empty-state view with refresh control".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-settings" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-settings" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier operational-settings page for controlling manual off-shift status and day-by-day business-hour windows backed by the shared supplier-shift context.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "headline: Settings"; "subtitle describing business hours and shift availability".
- Зона "shift-status-card" расположена в области "first card below header". Содержимое: "section title and explanatory copy"; "manual off-shift toggle pill"; "effective OPEN or CLOSED status label".
- Зона "business-hours-card" расположена в области "second card below shift-status". Содержимое: "section title and scheduling guidance"; "seven day rows each with enable checkbox, day label, and open-close time controls or Closed text".
- Зона "save-row" расположена в области "bottom action band". Содержимое: "Save Changes button"; "saved-success or failed-save status text".

### Controloverview

- Кнопка "ON SHIFT / OFF SHIFT toggle" расположена в "shift-status-card". Стиль: "pill toggle button".
- Кнопка "Day enabled checkbox" расположена в "business-hours-card day row". Стиль: "checkbox control".
- Кнопка "Save Changes" расположена в "save-row left". Стиль: "primary button".

### Iconoverview

- Иконка "spinner ring" используется в зоне "loading state".
- Иконка "status dot" используется в зоне "manual off-shift toggle pill".

### Flowoverview

**Flowid:** settings-bootstrap

**Summary:** Поток "settings-bootstrap" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Page reads shared supplier shift context from useSupplierShift".
- Шаг 2: "Context bootstraps from /v1/supplier/profile and exposes manual_off_shift, is_active, and operating_schedule".
- Шаг 3: "When the hook finishes loading, local form state mirrors the shared shift state".

---

**Flowid:** schedule-editing

**Summary:** Поток "schedule-editing" содержит 3 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier toggles day checkboxes to enable or disable days".
- Шаг 2: "Enabled days expose open and close time inputs".
- Шаг 3: "Changing a time updates local schedule state only until save".

---

**Flowid:** shift-save

**Summary:** Поток "shift-save" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier presses Save Changes".
- Шаг 2: "Page assembles final enabled-day schedule and manual off-shift value".
- Шаг 3: "Shared hook patches /v1/supplier/shift with manual_off_shift and operating_schedule".
- Шаг 4: "Save status indicator reports success or failure".

---


### Dependencyoverview

#### Reads

- /v1/supplier/profile

#### Writes

- /v1/supplier/shift

#### Localizednotes

- Чтение: "/v1/supplier/profile".
- Запись: "/v1/supplier/shift".
- Модель обновления: "shared-context bootstrap on mount plus explicit save action for schedule changes".

### Stateoverview

- Состояние: "settings loading spinner state".
- Состояние: "default shift and schedule form state".
- Состояние: "days enabled with time pickers".
- Состояние: "days disabled showing Closed text".
- Состояние: "save in progress state".
- Состояние: "saved successfully message".
- Состояние: "failed save message".

### Figureoverview

- Фигура: "full settings page with shift card, hours card, and save row".
- Фигура: "shift-status card close-up with manual off-shift toggle and effective status label".
- Фигура: "business-hours rows showing enabled and disabled day variants".

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

**Localizedsummary:** Локализованный обзор поверхности "web-supplier-staff" для роли поставщик на платформе web.

## Localized

**Purpose:** Поверхность "web-supplier-staff" представляет страница для роли поставщик на платформе web; исходное английское описание назначения сохранено в поле purposeSourceAnchor.

**Purposesourceanchor:** Supplier warehouse-staff management page for provisioning payloader accounts, listing worker credentials, and revealing one-time login PINs.

### Layoutoverview

- Зона "header" расположена в области "top full-width". Содержимое: "back link to supplier dashboard"; "headline: Warehouse Staff"; "subtitle describing payloader provisioning"; "Provision Worker CTA".
- Зона "kpi-row" расположена в области "below header". Содержимое: "Total Workers card"; "Active card"; "Inactive card".
- Зона "worker-ledger" расположена в области "main card region". Содержимое: "loading spinner or empty message"; "worker table with name, phone, worker ID, provision date, and status chip"; "pagination controls".
- Зона "provision-drawer" расположена в области "slide-out drawer". Правило видимости: "visible when showAdd is true". Содержимое: "name input"; "phone input"; "error text when form invalid or request fails"; "Provision Worker and Generate PIN CTA".
- Зона "pin-reveal-modal" расположена в области "center overlay modal". Правило видимости: "visible when createdPin exists". Содержимое: "success glyph"; "worker name and phone text"; "dashed PIN reveal panel"; "warning helper banner"; "Done button".

### Controloverview

- Кнопка "Supplier Dashboard" расположена в "header top-left". Стиль: "text link".
- Кнопка "+ Provision Worker" расположена в "header top-right". Стиль: "primary button".
- Кнопка "Provision Worker and Generate PIN" расположена в "provision-drawer footer". Стиль: "full-width primary button".
- Кнопка "Done" расположена в "pin-reveal-modal footer". Стиль: "full-width primary button".

### Iconoverview

- Иконка "warning" используется в зоне "PIN reveal warning banner".
- Иконка "success checkmark glyph" используется в зоне "PIN reveal modal hero circle".

### Flowoverview

**Flowid:** staff-bootstrap

**Summary:** Поток "staff-bootstrap" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Page fetches /v1/supplier/staff/payloader on mount".
- Шаг 2: "Worker rows populate KPI counts and the paginated table".

---

**Flowid:** worker-provisioning

**Summary:** Поток "worker-provisioning" содержит 4 шаг(а/ов).

#### Steps

- Шаг 1: "Supplier opens the provision drawer".
- Шаг 2: "Supplier enters worker name and phone".
- Шаг 3: "Page posts to /v1/supplier/staff/payloader".
- Шаг 4: "On success, drawer closes, worker table refreshes, and the one-time PIN reveal modal opens".

---

**Flowid:** pin-disclosure

**Summary:** Поток "pin-disclosure" содержит 2 шаг(а/ов).

#### Steps

- Шаг 1: "Modal displays generated login PIN exactly once".
- Шаг 2: "Supplier acknowledges via Done and the PIN overlay is dismissed".

---


### Dependencyoverview

#### Reads

- /v1/supplier/staff/payloader

#### Writes

- /v1/supplier/staff/payloader

#### Localizednotes

- Чтение: "/v1/supplier/staff/payloader".
- Запись: "/v1/supplier/staff/payloader".
- Модель обновления: "load on mount and reload after successful worker provisioning".

### Stateoverview

- Состояние: "table loading spinner state".
- Состояние: "empty worker roster state".
- Состояние: "worker table with pagination".
- Состояние: "provision drawer open".
- Состояние: "provision drawer validation error".
- Состояние: "PIN reveal modal visible".

### Figureoverview

- Фигура: "full staff page with KPI row and worker table".
- Фигура: "provision drawer with name and phone fields".
- Фигура: "PIN reveal modal with dashed PIN panel and warning banner".

---


