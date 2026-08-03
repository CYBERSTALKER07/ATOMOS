# Migration Runbook — Product VU + Catalog Images

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



**Change ID:** `20250611-products-unit-volume-vu`  
**Owner:** Platform / Supplier ops  
**Blast radius:** Spanner `Products`, warehouse manual dispatch capacity, supplier catalog surfaces, checkout line-item snapshots  
**Downtime:** None (additive DDL + backward-compatible API)

---

## 1. What this migration enables

| Capability | Depends on |
|---|---|
| Product volumetric unit (`UnitVolumeVU`) on catalog CRUD | Spanner column on `Products` |
| Dispatch capacity = Σ(qty × VU) vs truck `MaxVolumeVU` | Column + supplier-set VU values |
| `unit_volume_vu` snapshot on order line items at checkout | Backend code (no DDL); reads `Products.UnitVolumeVU` |
| Catalog product images (upload ticket + `image_url`) | `Products.ImageURL` (already exists) + `GCS_BUCKET_NAME` on backend pods |

**Canonical DDL (live instances that predate the column):**

`apps/backend-go/schema/migrations/20250611_products_unit_volume_vu.ddl`

```sql
ALTER TABLE Products ADD COLUMN UnitVolumeVU FLOAT64 NOT NULL DEFAULT (1.0);
```

Fresh installs that apply full `schema/spanner.ddl` already include the column — skip §3 if `INFORMATION_SCHEMA` shows it.

---

## 2. Preconditions

- [ ] Backend build containing `catalog/`, `dispatch/volume.go`, `order/volume.go`, and `storage/gcs.go` is staged (same release train as supplier portal + warehouse portal/mobile dispatch).
- [ ] Spanner admin IAM: `spanner.databases.updateDdl` on target database.
- [ ] Maintenance window **not required** — single additive `ALTER TABLE` with default.
- [ ] For **production images**: GCS bucket exists, CORS allows browser `PUT` from supplier portal origin, backend service account can sign V4 URLs (`iam.serviceAccounts.signBlob` or equivalent).

---

## 3. Spanner DDL — production / staging

### 3.1 Idempotency check

```sql
SELECT COLUMN_NAME, SPANNER_TYPE, IS_NULLABLE, COLUMN_DEFAULT
FROM INFORMATION_SCHEMA.COLUMNS
WHERE TABLE_NAME = 'Products' AND COLUMN_NAME = 'UnitVolumeVU';
```

- **0 rows** → apply migration (§3.2).
- **1 row** → already migrated; proceed to §4 verification only.

### 3.2 Apply migration

Replace project/instance/database:

```bash
export SPANNER_PROJECT="<gcp-project>"
export SPANNER_INSTANCE="<instance-id>"
export SPANNER_DATABASE="<database-id>"

gcloud spanner databases ddl update "$SPANNER_DATABASE" \
  --instance="$SPANNER_INSTANCE" \
  --project="$SPANNER_PROJECT" \
  --ddl="ALTER TABLE Products ADD COLUMN UnitVolumeVU FLOAT64 NOT NULL DEFAULT (1.0);"
```

Or from the repo file:

```bash
cd pegasusX
gcloud spanner databases ddl update "$SPANNER_DATABASE" \
  --instance="$SPANNER_INSTANCE" \
  --project="$SPANNER_PROJECT" \
  --ddl-file=apps/backend-go/schema/migrations/20250611_products_unit_volume_vu.ddl
```

**Expected:** DDL operation completes in seconds to low minutes. Existing rows receive `UnitVolumeVU = 1.0` via default (legacy qty-only capacity behavior preserved until suppliers edit VU).

### 3.3 Rollback

Spanner does not support `DROP COLUMN` in place on live tables without a table rebuild. **Do not roll back DDL** in production unless executing a planned table migration.

If a bad deploy must be reverted:

1. Roll back **application** pods only (previous binary).
2. Leave `UnitVolumeVU` in place — old code ignores the column; default `1.0` is safe.

---

## 4. Local / emulator

```bash
cd pegasusX
docker compose -f infra/docker-compose.yml up -d spanner-emulator redis kafka

export SPANNER_EMULATOR_HOST=localhost:9010
export SPANNER_PROJECT=pegasusx-local
export SPANNER_INSTANCE=pegasusx-instance
export SPANNER_DATABASE=pegasusx-db

cd apps/backend-go
go run ./cmd/setup
```

`cmd/setup` applies full `schema/spanner.ddl` idempotently (existing DBs get additive statements). Emulator fresh DBs include `UnitVolumeVU` from CREATE TABLE.

---

## 5. Backend environment (catalog images)

Add to backend deployment / `.env` (not required for VU-only dispatch):

| Variable | Required | Notes |
|---|---|---|
| `GCS_BUCKET_NAME` | Prod images | Omit in local dev → placeholder `placehold.co` URLs |
| `GCS_BUCKET_NAME` unset | Local OK | Upload ticket returns placeholder; portal still creates products |

Backend logs on boot:

- `gcs init failed; catalog image uploads use placeholders` → non-fatal; VU dispatch still works.
- No log + bucket set → signed uploads active.

**GCS CORS (supplier portal direct PUT):** allow `PUT` from portal origin(s), headers `Content-Type`, methods `PUT`.

---

## 6. Deploy order

Execute in this sequence to avoid contract drift:

| Step | Component | Action |
|---|---|---|
| 1 | **Spanner** | Apply DDL (§3) if column missing |
| 2 | **backend-go** | Deploy pods with `storage`, catalog upload ticket, dispatch volume |
| 3 | **supplier-portal** | Deploy catalog VU + image UI |
| 4 | **supplier-app-android / ios** | Ship catalog create + VU + image edit |
| 5 | **warehouse-portal + warehouse mobile** | Manual dispatch + VU capacity (same release or already live) |
| 6 | **Retailer clients** | No deploy required for VU (read-only `image_url` already wired) |

**Safe partial order:** DDL before backend is mandatory. Backend before supplier portal/mobile is mandatory for image ticket route. Warehouse dispatch can lag only if still using qty stub — capacity will be wrong until backend + VU values are live.

---

## 7. Post-migration verification

### 7.1 Spanner data

```sql
SELECT
  COUNT(*) AS total_products,
  COUNTIF(UnitVolumeVU = 1.0) AS default_vu,
  COUNTIF(UnitVolumeVU <> 1.0) AS customized_vu
FROM Products;
```

After supplier onboarding, `customized_vu` should increase over time.

### 7.2 API smoke (supplier JWT)

```bash
# List products — expect unit_volume_vu on each row
curl -s -H "Authorization: Bearer $SUPPLIER_JWT" \
  "$API_BASE/v1/catalog/products" | jq '.[0].unit_volume_vu'

# Upload ticket (authenticated supplier)
curl -s -H "Authorization: Bearer $SUPPLIER_JWT" \
  "$API_BASE/v1/catalog/products/upload-ticket?ext=jpg" | jq .
```

### 7.3 Dispatch capacity (warehouse scope)

```bash
curl -s -H "Authorization: Bearer $WAREHOUSE_JWT" \
  "$API_BASE/v1/warehouse/dispatch/preview?warehouse_id=$WH_ID" \
  | jq '.undispatched_orders[] | {order_id, volume_vu}'
```

`volume_vu` must reflect Σ(qty × product VU), not raw qty sum.

### 7.4 Checkout snapshot

Place a test retailer order after setting a product VU ≠ 1.0. Inspect `Orders.LineItems` JSON (or order detail API): each line should include `unit_volume_vu`.

### 7.5 Supplier surfaces

| Surface | Check |
|---|---|
| supplier-portal `/catalog` | Create product with image; edit VU; change image on row |
| supplier Android/iOS Catalog | Create + change image + save VU |
| warehouse portal `/dispatch` | Capacity bar moves when VU changes |

---

## 8. Data backfill policy

| Dataset | Backfill needed? | Behavior |
|---|---|---|
| `Products.UnitVolumeVU` | **No** | Default `1.0` = legacy 1:1 qty capacity |
| Historical `Orders` line items | **No** | Missing `unit_volume_vu` in JSON: dispatch/checkout paths fall back to live `Products` lookup or `1.0` |
| New orders after deploy | Automatic | `order/volume.go` enriches at checkout |

**Operational follow-up (product, not platform):** suppliers should set accurate VU per SKU via portal or mobile catalog. Until then, dispatch capacity is conservative (1 VU per unit).

Optional audit query — products still at default:

```sql
SELECT SupplierId, COUNT(*) AS skus_at_default_vu
FROM Products
WHERE UnitVolumeVU = 1.0
GROUP BY SupplierId
ORDER BY skus_at_default_vu DESC
LIMIT 20;
```

---

## 9. Failure modes

| Symptom | Likely cause | Fix |
|---|---|---|
| `unit_volume_vu` missing in API 500s | DDL not applied | Run §3 |
| Capacity = order qty count | Old backend or all VU = 1.0 | Deploy backend; educate suppliers on VU |
| Image upload 403 on ticket | Supplier not authenticated / wrong role | JWT `ADMIN` supplier session |
| Image upload fails PUT to GCS | CORS or signed URL mismatch | Fix bucket CORS + content-type on PUT |
| Placeholder images in prod | `GCS_BUCKET_NAME` unset | Set env + restart backend |
| DDL "already exists" | Column present | Skip; verify §7.1 |

---

## 10. Sign-off checklist

- [ ] `INFORMATION_SCHEMA` shows `Products.UnitVolumeVU`
- [ ] `go build ./...` clean on release tag (`apps/backend-go`)
- [ ] Supplier catalog list returns `unit_volume_vu`
- [ ] Warehouse dispatch preview returns `volume_vu` per order
- [ ] At least one product updated to VU ≠ 1.0 in staging end-to-end test
- [ ] (Optional) `GCS_BUCKET_NAME` set and real image round-trip in staging
- [ ] Release notes sent to supplier ops: **set Unit VU on every SKU before trusting dispatch capacity**

---

## Reference paths

| Artifact | Path |
|---|---|
| Incremental DDL | `pegasusX/apps/backend-go/schema/migrations/20250611_products_unit_volume_vu.ddl` |
| Full schema | `pegasusX/apps/backend-go/schema/spanner.ddl` |
| Local bootstrap | `pegasusX/apps/backend-go/cmd/setup` |
| Upload ticket handler | `GET /v1/catalog/products/upload-ticket` |
| Volume at checkout | `pegasusX/apps/backend-go/order/volume.go` |
| Volume at dispatch | `pegasusX/apps/backend-go/dispatch/volume.go` |
