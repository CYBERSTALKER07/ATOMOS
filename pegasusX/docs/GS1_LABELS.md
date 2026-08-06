# GS1 labels (Gate-3 Wave 2C)

Logistics identifiers and Zebra ZPL without EDI/AS2.

## Scope

| Capability | Status |
|------------|--------|
| GLN on supplier / warehouse / retailer location | Wired (Mod-10 validate) |
| `Gs1CompanyPrefix` on supplier profile | Wired (7–10 digits) |
| SSCC-18 on `ManifestShipUnits` at payload seal | Wired (one SSCC per manifest order) |
| ZPL `text/plain` GS1-128 `(00)` SSCC | Wired |
| EDI DESADV SSCC segments (CPS/PAC/GIN+BJ) | **Wired** (from `ManifestShipUnits` at emit) |
| AS2 / certified EDIFACT / 1C / registry sync | AS2 transport Wired (not Drummond); certified EDIFACT / 1C package still open |

Migration: `apps/backend-go/schema/migrations/20260806_gs1_labels.ddl`.

## Flags

| Env | Default | Meaning |
|-----|---------|---------|
| `GS1_LABELS_ENABLED` | on | Gate mint + label APIs |
| `GS1_COMPANY_PREFIX` | empty | Override company prefix (else `SupplierProfiles.Gs1CompanyPrefix`) |

## APIs

| Method | Path | Notes |
|--------|------|--------|
| GET/PUT | `/v1/supplier/profile` | `gln`, `gs1_company_prefix` |
| GET/PATCH | `/v1/warehouse/ops/location` | `gln` |
| PATCH | `/v1/retailer/locations/{id}` | `gln` (ship-to) |
| GET | `/v1/{payloader\|payload\|warehouse\|supplier}/manifests/{id}/ship-units` | List SSCC rows |
| POST | `…/manifests/{id}/labels` | Optional `{order_id}` → ZPL attachment |

Shared helpers: `apps/backend-go/gs1/` (`NormalizeGTIN/GLN/SSCC`, `GenerateSSCC`, `AICode128ZPL`).

## Seal behavior

On successful payload seal (`seal`, `seal-completed`, `seal-all`, `/v1/payload/seal` with manifest):

1. Load company prefix (env override or profile).
2. Soft-skip (log) if prefix empty.
3. For each `ManifestOrders` row without a ship unit: mint SSCC-18 (extension `0`) and insert `ManifestShipUnits`.
4. Idempotent on `(ManifestId, OrderId)` / unique `Sscc`.

## Clients

- Supplier portal profile: GLN + company prefix fields.
- Warehouse portal manifests: **Print labels** downloads `.zpl`.
- Payload native: API only this wave (list + labels).

## SSMR

Markers: `PX_E2E_GS1_SSCC_OK` / `_SKIPPED`, `PX_E2E_GS1_ZPL_OK` / `_SKIPPED` (after payloader seal). Set `GS1_COMPANY_PREFIX` on the API process for hermetic smoke.
