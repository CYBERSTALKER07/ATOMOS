# D3 schema — apply from IDE (gcloud)

## Status
- Stuck `go run ./cmd/setup` **stopped**
- Spanner instance **READY** (100 PU)
- Partial schema already present (~16 tables last check)
- Prefer **gcloud batch DDL** (faster than one-by-one Go setup)

## IDE steps (Google Cloud extension + Terminal)

1. Sign in as `blackfoxenterprise3697@gmail.com`
2. Project: `pegasus-503013`
3. Open **Terminal** in IDE:

```bash
cd /Users/shakhzod/Desktop/V.O.I.D/pegasusX
gcloud config set account blackfoxenterprise3697@gmail.com
gcloud config set project pegasus-503013
bash scripts/d3_apply_schema_gcloud.sh
```

4. Wait for `d3-gcloud-schema-ok`
5. In Spanner Studio (extension or Console), verify:

```sql
SELECT COUNT(*) AS tables
FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = '';

SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES
WHERE TABLE_SCHEMA = '' AND TABLE_NAME = 'OrderFiscalReceipts';
```

## Why gcloud is better here
- Batches ~15 statements per API call (Go setup was 1 statement + long wait each)
- Safe to re-run (skips already-exists)
- Same auth as Cloud extension if you ran `gcloud auth login`

## Do NOT
- Re-run old `make phase0-migrate` / `cmd/setup` until this finishes (or only after stop script)
- Manually paste entire 58KB DDL into Studio in one go (timeouts / hard to resume)
