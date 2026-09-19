# External WMS ASN (G5.D)

## Outbound

EDI **DESADV** (with SSCC packing when sealed) is the ASN to external WMS.
Profile flag `asn_as_desadv` (default true on new packs).

## Inbound

`POST /partner/v1/wms/asn` (`inventory:write`)

```json
{
  "external_asn_id": "ASN-1",
  "warehouse_id": "wh_…",
  "plant_id": "optional-external-plant",
  "lines": [{ "sku": "…", "gtin": "…", "qty": 10, "lot_code": "L1" }]
}
```

- Idempotent on `external_asn_id`
- `plant_id` resolved via master-data plant map
- Flag: `PARTNER_WMS_ASN_ENABLED` (default on for code path)

Stock apply uses partner stock upsert path when inventory service is wired.
