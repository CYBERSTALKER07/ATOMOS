# Android EAN barcode scanner (PegasusX)

Compose preview + keyboard wedge + optional DataWedge broadcast helper for warehouse / payload / supplier scan surfaces.

## Throughput harden (P2 #9)

- ML Kit restricted to **EAN-8 / EAN-13** via `BarcodeScannerOptions`
- Same-code debounce **300 ms** (`EAN_SCAN_DEBOUNCE_MS`)
- Torch toggle on preview
- `KeyboardWedgeBarcodeField` for Zebra keyboard-wedge
- `DataWedgeBarcodeEffect` listens for `com.symbol.datawedge.api.RESULT_ACTION`

## DataWedge profile (Zebra TC-series)

1. New profile → associate with the warehouse / payload app package.
2. Enable **Intent** output:
   - Action: `com.symbol.datawedge.api.RESULT_ACTION`
   - Delivery: Broadcast
   - Key: `com.symbol.datawedge.data_string`
3. Disable Intent start activity so scans do not steal focus from Compose.
