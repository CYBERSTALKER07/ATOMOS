# Barcode Go-Live Checklist

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



Before enabling returns inbound gate (payload terminal, warehouse portal, native scan flows), supplier catalog must have valid EAN/GTIN barcodes.

**Policy:** [`pegasus/docs/BARCODE_SCANNING.md`](../../pegasus/docs/BARCODE_SCANNING.md)  
**SSMR proof:** `PX_E2E_RETURN_GATE_RECEIVE_OK`  
**Matrix:** [`ROLE_ROW_PARITY_MATRIX.md`](./ROLE_ROW_PARITY_MATRIX.md) § Barcode catalog

---

## Pre-go-live (supplier)

| Step | Owner | Pass criteria |
|------|-------|---------------|
| 1 | Catalog ops | Every SKU in returns pilot has `Products.Barcode` in Spanner |
| 2 | Supplier portal | `/catalog` — manual EAN + checksum validation passes |
| 3 | Supplier native | Camera + manual capture on Android/iOS catalog detail |
| 4 | Staging seed | SSMR SKUs (`SSMR-SKU-1`, etc.) have barcodes for war-story QA |

---

## Gate behavior

| Scan result | System response | Operator action |
|-------------|-----------------|-----------------|
| Valid EAN matches catalog | Inbound scan accepted | Confirm quantity |
| Unknown EAN | Reject with mismatch | Fix catalog or quarantine goods |
| Duplicate scan (retry) | Idempotency replay | No double credit |

---

## Surfaces to verify

- `payload-terminal` inbound + offline queue
- `payload-app-android` / `payload-app-ios` tablet
- `warehouse-portal` `/returns`
- `warehouse-app-android` / `warehouse-app-ios`

---

## Sign-off

| Field | Value |
|-------|-------|
| Environment | staging / prod |
| SKU count with barcode | |
| Pilot warehouse/payload node | |
| Signed | |
