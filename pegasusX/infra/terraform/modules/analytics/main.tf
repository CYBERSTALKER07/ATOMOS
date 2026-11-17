/**
 * Module: Analytics & ML Platform
 * Regional BigQuery datasets (`raw`, `curated`, `features`), typed tables,
 * authorized tenant views & row-access policies, Vertex AI staging bucket, and IAM.
 */

locals {
  effective_cell_id       = trimspace(var.cell_id) != "" ? trimspace(var.cell_id) : "uz"
  sanitized_cell_id       = replace(local.effective_cell_id, "-", "_")
  sanitized_prefix        = trimspace(var.bigquery_dataset_prefix) != "" ? "${replace(trimspace(var.bigquery_dataset_prefix), "-", "_")}_" : ""
  effective_vertex_bucket = trimspace(var.vertex_staging_bucket_name) != "" ? trimspace(var.vertex_staging_bucket_name) : "${var.project_id}-${replace(local.effective_cell_id, "_", "-")}-vertex-staging"
  analytics_labels = merge(
    var.labels,
    {
      cell        = local.effective_cell_id
      environment = var.environment
    }
  )

  dataset_definitions = {
    raw = {
      dataset_id    = "${local.sanitized_prefix}raw"
      friendly_name = "PegasusX Raw Ingestion (${local.effective_cell_id})"
      description   = "Append-only Kafka domain events and ontology snapshots for cell ${local.effective_cell_id}."
    }
    curated = {
      dataset_id    = "${local.sanitized_prefix}curated"
      friendly_name = "PegasusX Curated Financial & Operational Ledger (${local.effective_cell_id})"
      description   = "Curated GL entries, GL postings, reconciliation runs, and tenant-authorized views for cell ${local.effective_cell_id}."
    }
    features = {
      dataset_id    = "${local.sanitized_prefix}features"
      friendly_name = "PegasusX ML Feature Store (${local.effective_cell_id})"
      description   = "Daily SKU demand and retailer credit risk feature tables for Vertex AI and planning workloads in cell ${local.effective_cell_id}."
    }
  }

  effective_authorized_reader = trimspace(var.bigquery_authorized_reader_email) == "" ? "" : (
    can(regex("^(user|group|serviceAccount|domain):", trimspace(var.bigquery_authorized_reader_email)))
    ? trimspace(var.bigquery_authorized_reader_email)
    : "user:${trimspace(var.bigquery_authorized_reader_email)}"
  )

  # Static SA keys keep for_each keys known at plan/validate time even when emails come from module outputs.
  workload_principals = var.enable_iam_bindings ? {
    backend = var.backend_sa_email
    ai      = var.ai_worker_sa_email
  } : {}

  dataset_iam_pairs = flatten([
    for ds_key in keys(local.dataset_definitions) : [
      for sa_key, sa_email in local.workload_principals : {
        key      = "${ds_key}-${sa_key}"
        ds_key   = ds_key
        sa_email = sa_email
      }
    ]
  ])

  cell_row_access_tables = var.enable_iam_bindings ? {
    domain_events = {
      dataset_key      = "raw"
      table_id         = google_bigquery_table.domain_events.table_id
      filter_predicate = "cell_id = '${local.effective_cell_id}'"
    }
    ontology_objects = {
      dataset_key      = "raw"
      table_id         = google_bigquery_table.ontology_objects.table_id
      filter_predicate = "cell_id = '${local.effective_cell_id}'"
    }
    gl_entries = {
      dataset_key      = "curated"
      table_id         = google_bigquery_table.gl_entries.table_id
      filter_predicate = "entity_id IS NOT NULL"
    }
    gl_postings = {
      dataset_key      = "curated"
      table_id         = google_bigquery_table.gl_postings.table_id
      filter_predicate = "entity_id IS NOT NULL"
    }
    recon_runs = {
      dataset_key      = "curated"
      table_id         = google_bigquery_table.recon_runs.table_id
      filter_predicate = "entity_id IS NOT NULL"
    }
    sku_demand_daily = {
      dataset_key      = "features"
      table_id         = google_bigquery_table.sku_demand_daily.table_id
      filter_predicate = "cell_id = '${local.effective_cell_id}'"
    }
    retailer_credit_signals = {
      dataset_key      = "features"
      table_id         = google_bigquery_table.retailer_credit_signals.table_id
      filter_predicate = "cell_id = '${local.effective_cell_id}'"
    }
  } : {}
}

# 0. Optional BigQuery Service Agent CMEK Grant (Gated so dry-run plan/validate never requires live API lookup)
data "google_bigquery_default_service_account" "bq_sa" {
  count   = var.enable_cmek && var.enable_cmek_iam_binding ? 1 : 0
  project = var.project_id
}

resource "google_kms_crypto_key_iam_member" "bq_kms_user" {
  count         = var.enable_cmek && var.enable_cmek_iam_binding ? 1 : 0
  crypto_key_id = var.kms_key_id
  role          = "roles/cloudkms.cryptoKeyEncrypterDecrypter"
  member        = "serviceAccount:${data.google_bigquery_default_service_account.bq_sa[0].email}"
}

# 1. Regional BigQuery Datasets (`raw`, `curated`, `features`)
resource "google_bigquery_dataset" "datasets" {
  for_each = local.dataset_definitions

  project                    = var.project_id
  dataset_id                 = each.value.dataset_id
  friendly_name              = each.value.friendly_name
  description                = each.value.description
  location                   = var.region
  delete_contents_on_destroy = var.environment != "production"

  dynamic "default_encryption_configuration" {
    for_each = var.enable_cmek ? [1] : []
    content {
      kms_key_name = var.kms_key_id
    }
  }

  labels = merge(local.analytics_labels, {
    layer = each.key
  })

  depends_on = [
    google_kms_crypto_key_iam_member.bq_kms_user
  ]
}

# 2. Raw Layer Tables (`raw.domain_events`, `raw.ontology_objects`)
resource "google_bigquery_table" "domain_events" {
  project             = var.project_id
  dataset_id          = google_bigquery_dataset.datasets["raw"].dataset_id
  table_id            = "domain_events"
  description         = "Canonical Kafka domain event stream landed idempotently by void-bq-sink."
  deletion_protection = var.deletion_protection

  time_partitioning {
    type  = "DAY"
    field = "occurred_at"
  }

  clustering = ["cell_id", "supplier_id", "event_type"]

  dynamic "encryption_configuration" {
    for_each = var.enable_cmek ? [1] : []
    content {
      kms_key_name = var.kms_key_id
    }
  }

  schema = jsonencode([
    { name = "event_id", type = "STRING", mode = "REQUIRED" },
    { name = "event_type", type = "STRING", mode = "REQUIRED" },
    { name = "aggregate_type", type = "STRING", mode = "NULLABLE" },
    { name = "aggregate_id", type = "STRING", mode = "NULLABLE" },
    { name = "cell_id", type = "STRING", mode = "REQUIRED" },
    { name = "supplier_id", type = "STRING", mode = "REQUIRED" },
    { name = "topic", type = "STRING", mode = "REQUIRED" },
    { name = "partition_num", type = "INT64", mode = "REQUIRED" },
    { name = "kafka_offset", type = "INT64", mode = "REQUIRED" },
    { name = "trace_id", type = "STRING", mode = "NULLABLE" },
    { name = "schema_version", type = "STRING", mode = "NULLABLE" },
    { name = "payload_json", type = "JSON", mode = "REQUIRED" },
    { name = "occurred_at", type = "TIMESTAMP", mode = "REQUIRED" },
    { name = "ingested_at", type = "TIMESTAMP", mode = "REQUIRED" },
  ])

  labels = merge(local.analytics_labels, { layer = "raw", domain = "events" })
}

resource "google_bigquery_table" "ontology_objects" {
  project             = var.project_id
  dataset_id          = google_bigquery_dataset.datasets["raw"].dataset_id
  table_id            = "ontology_objects"
  description         = "Versioned OntologyObjects projection snapshots streamed from ontology.object.changed events."
  deletion_protection = var.deletion_protection

  time_partitioning {
    type  = "DAY"
    field = "updated_at"
  }

  clustering = ["cell_id", "supplier_id", "object_type"]

  dynamic "encryption_configuration" {
    for_each = var.enable_cmek ? [1] : []
    content {
      kms_key_name = var.kms_key_id
    }
  }

  schema = jsonencode([
    { name = "cell_id", type = "STRING", mode = "REQUIRED" },
    { name = "object_type", type = "STRING", mode = "REQUIRED" },
    { name = "object_id", type = "STRING", mode = "REQUIRED" },
    { name = "supplier_id", type = "STRING", mode = "REQUIRED" },
    { name = "version", type = "INT64", mode = "REQUIRED" },
    { name = "change_kind", type = "STRING", mode = "NULLABLE" },
    { name = "source_event_type", type = "STRING", mode = "NULLABLE" },
    { name = "source_event_id", type = "STRING", mode = "NULLABLE" },
    { name = "security_markings", type = "STRING", mode = "REPEATED" },
    { name = "properties_json", type = "JSON", mode = "NULLABLE" },
    { name = "valid_from", type = "TIMESTAMP", mode = "NULLABLE" },
    { name = "updated_at", type = "TIMESTAMP", mode = "REQUIRED" },
    { name = "ingested_at", type = "TIMESTAMP", mode = "REQUIRED" },
  ])

  labels = merge(local.analytics_labels, { layer = "raw", domain = "ontology" })
}

# 3. Curated Layer Tables (`curated.gl_entries`, `curated.gl_postings`, `curated.recon_runs`)
resource "google_bigquery_table" "gl_entries" {
  project             = var.project_id
  dataset_id          = google_bigquery_dataset.datasets["curated"].dataset_id
  table_id            = "gl_entries"
  description         = "Append-only General Ledger journal entry headers replicated from gl.entry.posted and gl.entry.reversed."
  deletion_protection = var.deletion_protection

  time_partitioning {
    type  = "DAY"
    field = "posted_at"
  }

  clustering = ["entity_id", "period_id", "source_type"]

  dynamic "encryption_configuration" {
    for_each = var.enable_cmek ? [1] : []
    content {
      kms_key_name = var.kms_key_id
    }
  }

  schema = jsonencode([
    { name = "entity_id", type = "STRING", mode = "REQUIRED" },
    { name = "entry_id", type = "STRING", mode = "REQUIRED" },
    { name = "period_id", type = "STRING", mode = "REQUIRED" },
    { name = "source_event_id", type = "STRING", mode = "NULLABLE" },
    { name = "source_type", type = "STRING", mode = "REQUIRED" },
    { name = "source_id", type = "STRING", mode = "REQUIRED" },
    { name = "order_id", type = "STRING", mode = "NULLABLE" },
    { name = "currency", type = "STRING", mode = "REQUIRED" },
    { name = "country_pack", type = "STRING", mode = "REQUIRED" },
    { name = "total_debit_minor", type = "INT64", mode = "REQUIRED" },
    { name = "total_credit_minor", type = "INT64", mode = "REQUIRED" },
    { name = "reverses_entry_id", type = "STRING", mode = "NULLABLE" },
    { name = "prev_hash", type = "STRING", mode = "NULLABLE" },
    { name = "entry_hash", type = "STRING", mode = "REQUIRED" },
    { name = "posted_by", type = "STRING", mode = "NULLABLE" },
    { name = "occurred_at", type = "TIMESTAMP", mode = "NULLABLE" },
    { name = "posted_at", type = "TIMESTAMP", mode = "REQUIRED" },
    { name = "ingested_at", type = "TIMESTAMP", mode = "REQUIRED" },
  ])

  labels = merge(local.analytics_labels, { layer = "curated", domain = "ledger" })
}

resource "google_bigquery_table" "gl_postings" {
  project             = var.project_id
  dataset_id          = google_bigquery_dataset.datasets["curated"].dataset_id
  table_id            = "gl_postings"
  description         = "Balanced General Ledger debit/credit legs (minor units INT64 only)."
  deletion_protection = var.deletion_protection

  time_partitioning {
    type  = "DAY"
    field = "posted_at"
  }

  clustering = ["entity_id", "account_code", "entry_id"]

  dynamic "encryption_configuration" {
    for_each = var.enable_cmek ? [1] : []
    content {
      kms_key_name = var.kms_key_id
    }
  }

  schema = jsonencode([
    { name = "entity_id", type = "STRING", mode = "REQUIRED" },
    { name = "entry_id", type = "STRING", mode = "REQUIRED" },
    { name = "line", type = "INT64", mode = "REQUIRED" },
    { name = "account_code", type = "STRING", mode = "REQUIRED" },
    { name = "direction", type = "STRING", mode = "REQUIRED" },
    { name = "amount_minor", type = "INT64", mode = "REQUIRED" },
    { name = "currency", type = "STRING", mode = "REQUIRED" },
    { name = "functional_amount_minor", type = "INT64", mode = "REQUIRED" },
    { name = "fx_rate_id", type = "STRING", mode = "NULLABLE" },
    { name = "party_type", type = "STRING", mode = "NULLABLE" },
    { name = "party_id", type = "STRING", mode = "NULLABLE" },
    { name = "warehouse_id", type = "STRING", mode = "NULLABLE" },
    { name = "product_id", type = "STRING", mode = "NULLABLE" },
    { name = "lot_id", type = "STRING", mode = "NULLABLE" },
    { name = "posted_at", type = "TIMESTAMP", mode = "REQUIRED" },
    { name = "ingested_at", type = "TIMESTAMP", mode = "REQUIRED" },
  ])

  labels = merge(local.analytics_labels, { layer = "curated", domain = "ledger" })
}

resource "google_bigquery_table" "recon_runs" {
  project             = var.project_id
  dataset_id          = google_bigquery_dataset.datasets["curated"].dataset_id
  table_id            = "recon_runs"
  description         = "Four-way financial reconciliation run outcomes and minor-unit exposure totals."
  deletion_protection = var.deletion_protection

  time_partitioning {
    type  = "DAY"
    field = "started_at"
  }

  clustering = ["entity_id", "period_id", "status"]

  dynamic "encryption_configuration" {
    for_each = var.enable_cmek ? [1] : []
    content {
      kms_key_name = var.kms_key_id
    }
  }

  schema = jsonencode([
    { name = "entity_id", type = "STRING", mode = "REQUIRED" },
    { name = "run_id", type = "STRING", mode = "REQUIRED" },
    { name = "period_id", type = "STRING", mode = "REQUIRED" },
    { name = "mode", type = "STRING", mode = "REQUIRED" },
    { name = "status", type = "STRING", mode = "REQUIRED" },
    { name = "break_count", type = "INT64", mode = "REQUIRED" },
    { name = "exposure_minor", type = "INT64", mode = "REQUIRED" },
    { name = "currency", type = "STRING", mode = "NULLABLE" },
    { name = "triggered_by", type = "STRING", mode = "REQUIRED" },
    { name = "checks_json", type = "JSON", mode = "NULLABLE" },
    { name = "summary_json", type = "JSON", mode = "NULLABLE" },
    { name = "started_at", type = "TIMESTAMP", mode = "REQUIRED" },
    { name = "completed_at", type = "TIMESTAMP", mode = "NULLABLE" },
    { name = "ingested_at", type = "TIMESTAMP", mode = "REQUIRED" },
  ])

  labels = merge(local.analytics_labels, { layer = "curated", domain = "reconciliation" })
}

# 4. Feature Store Tables (`features.sku_demand_daily`, `features.retailer_credit_signals`)
resource "google_bigquery_table" "sku_demand_daily" {
  project             = var.project_id
  dataset_id          = google_bigquery_dataset.datasets["features"].dataset_id
  table_id            = "sku_demand_daily"
  description         = "Daily SKU demand, Croston/SBA dispersion, and minor-unit revenue features for Vertex AI forecasting."
  deletion_protection = var.deletion_protection

  time_partitioning {
    type  = "DAY"
    field = "feature_date"
  }

  clustering = ["supplier_id", "warehouse_id", "product_id"]

  dynamic "encryption_configuration" {
    for_each = var.enable_cmek ? [1] : []
    content {
      kms_key_name = var.kms_key_id
    }
  }

  schema = jsonencode([
    { name = "feature_date", type = "DATE", mode = "REQUIRED" },
    { name = "cell_id", type = "STRING", mode = "REQUIRED" },
    { name = "supplier_id", type = "STRING", mode = "REQUIRED" },
    { name = "warehouse_id", type = "STRING", mode = "REQUIRED" },
    { name = "product_id", type = "STRING", mode = "REQUIRED" },
    { name = "sku", type = "STRING", mode = "NULLABLE" },
    { name = "currency", type = "STRING", mode = "NULLABLE" },
    { name = "units_ordered", type = "INT64", mode = "REQUIRED" },
    { name = "units_delivered", type = "INT64", mode = "REQUIRED" },
    { name = "revenue_minor", type = "INT64", mode = "REQUIRED" },
    { name = "baseline_qty", type = "INT64", mode = "NULLABLE" },
    { name = "confidence_pct", type = "INT64", mode = "NULLABLE" },
    { name = "adi", type = "FLOAT64", mode = "NULLABLE" },
    { name = "cv2", type = "FLOAT64", mode = "NULLABLE" },
    { name = "demand_category", type = "STRING", mode = "NULLABLE" },
    { name = "stockout_events", type = "INT64", mode = "NULLABLE" },
    { name = "updated_at", type = "TIMESTAMP", mode = "REQUIRED" },
  ])

  labels = merge(local.analytics_labels, { layer = "features", domain = "forecasting" })
}

resource "google_bigquery_table" "retailer_credit_signals" {
  project             = var.project_id
  dataset_id          = google_bigquery_dataset.datasets["features"].dataset_id
  table_id            = "retailer_credit_signals"
  description         = "Daily retailer credit exposure, overdue, and payment-promise risk features (minor units INT64)."
  deletion_protection = var.deletion_protection

  time_partitioning {
    type  = "DAY"
    field = "signal_date"
  }

  clustering = ["supplier_id", "retailer_id", "risk_tier"]

  dynamic "encryption_configuration" {
    for_each = var.enable_cmek ? [1] : []
    content {
      kms_key_name = var.kms_key_id
    }
  }

  schema = jsonencode([
    { name = "signal_date", type = "DATE", mode = "REQUIRED" },
    { name = "cell_id", type = "STRING", mode = "REQUIRED" },
    { name = "supplier_id", type = "STRING", mode = "REQUIRED" },
    { name = "retailer_id", type = "STRING", mode = "REQUIRED" },
    { name = "currency", type = "STRING", mode = "REQUIRED" },
    { name = "credit_limit_minor", type = "INT64", mode = "REQUIRED" },
    { name = "current_balance_minor", type = "INT64", mode = "REQUIRED" },
    { name = "overdue_balance_minor", type = "INT64", mode = "REQUIRED" },
    { name = "open_invoice_count", type = "INT64", mode = "REQUIRED" },
    { name = "late_payments_60d", type = "INT64", mode = "REQUIRED" },
    { name = "promise_breaches_90d", type = "INT64", mode = "REQUIRED" },
    { name = "risk_tier", type = "STRING", mode = "REQUIRED" },
    { name = "delinquent", type = "BOOL", mode = "REQUIRED" },
    { name = "on_hold", type = "BOOL", mode = "REQUIRED" },
    { name = "updated_at", type = "TIMESTAMP", mode = "REQUIRED" },
  ])

  labels = merge(local.analytics_labels, { layer = "features", domain = "credit" })
}

# 5. Tenant Row-Access & Authorized View Pattern
resource "google_bigquery_table" "tenant_access_principals" {
  project             = var.project_id
  dataset_id          = google_bigquery_dataset.datasets["curated"].dataset_id
  table_id            = "tenant_access_principals"
  description         = "Tenant principal entitlement lookup used by BigQuery authorized tenant views."
  deletion_protection = var.deletion_protection

  clustering = ["cell_id", "tenant_id", "principal_email"]

  dynamic "encryption_configuration" {
    for_each = var.enable_cmek ? [1] : []
    content {
      kms_key_name = var.kms_key_id
    }
  }

  schema = jsonencode([
    { name = "tenant_id", type = "STRING", mode = "REQUIRED" },
    { name = "cell_id", type = "STRING", mode = "REQUIRED" },
    { name = "principal_email", type = "STRING", mode = "REQUIRED" },
    { name = "role_scope", type = "STRING", mode = "REQUIRED" },
    { name = "updated_at", type = "TIMESTAMP", mode = "REQUIRED" },
  ])

  labels = merge(local.analytics_labels, { layer = "curated", domain = "tenancy" })
}

resource "google_bigquery_table" "v_tenant_domain_events" {
  project             = var.project_id
  dataset_id          = google_bigquery_dataset.datasets["curated"].dataset_id
  table_id            = "v_tenant_domain_events"
  description         = "Authorized tenant-scoped view over raw.domain_events filtered by SESSION_USER() entitlement."
  deletion_protection = false

  view {
    query          = <<-SQL
      SELECT
        e.event_id,
        e.event_type,
        e.aggregate_type,
        e.aggregate_id,
        e.cell_id,
        e.supplier_id,
        e.topic,
        e.partition_num,
        e.kafka_offset,
        e.trace_id,
        e.schema_version,
        e.payload_json,
        e.occurred_at,
        e.ingested_at
      FROM `${var.project_id}.${google_bigquery_dataset.datasets["raw"].dataset_id}.${google_bigquery_table.domain_events.table_id}` AS e
      INNER JOIN `${var.project_id}.${google_bigquery_dataset.datasets["curated"].dataset_id}.${google_bigquery_table.tenant_access_principals.table_id}` AS p
        ON e.supplier_id = p.tenant_id
       AND e.cell_id = p.cell_id
      WHERE LOWER(p.principal_email) = LOWER(SESSION_USER())
    SQL
    use_legacy_sql = false
  }

  labels = merge(local.analytics_labels, { layer = "curated", domain = "tenancy", view = "authorized" })
}

resource "google_bigquery_table" "v_tenant_gl_entries" {
  project             = var.project_id
  dataset_id          = google_bigquery_dataset.datasets["curated"].dataset_id
  table_id            = "v_tenant_gl_entries"
  description         = "Tenant-scoped GL entries view enforcing finance/owner/auditor entitlement via SESSION_USER()."
  deletion_protection = false

  view {
    query          = <<-SQL
      SELECT
        g.entity_id,
        g.entry_id,
        g.period_id,
        g.source_event_id,
        g.source_type,
        g.source_id,
        g.order_id,
        g.currency,
        g.country_pack,
        g.total_debit_minor,
        g.total_credit_minor,
        g.reverses_entry_id,
        g.prev_hash,
        g.entry_hash,
        g.posted_by,
        g.occurred_at,
        g.posted_at,
        g.ingested_at
      FROM `${var.project_id}.${google_bigquery_dataset.datasets["curated"].dataset_id}.${google_bigquery_table.gl_entries.table_id}` AS g
      INNER JOIN `${var.project_id}.${google_bigquery_dataset.datasets["curated"].dataset_id}.${google_bigquery_table.tenant_access_principals.table_id}` AS p
        ON g.entity_id = p.tenant_id
      WHERE LOWER(p.principal_email) = LOWER(SESSION_USER())
        AND p.role_scope IN ('FINANCE', 'FINANCE_MANAGER', 'ACCOUNTANT', 'OWNER', 'PLATFORM_ADMIN', 'AUDITOR')
    SQL
    use_legacy_sql = false
  }

  labels = merge(local.analytics_labels, { layer = "curated", domain = "tenancy", view = "authorized" })
}

resource "google_bigquery_table" "v_tenant_sku_demand_daily" {
  project             = var.project_id
  dataset_id          = google_bigquery_dataset.datasets["curated"].dataset_id
  table_id            = "v_tenant_sku_demand_daily"
  description         = "Authorized tenant-scoped view over features.sku_demand_daily filtered by SESSION_USER() entitlement."
  deletion_protection = false

  view {
    query          = <<-SQL
      SELECT
        f.feature_date,
        f.cell_id,
        f.supplier_id,
        f.warehouse_id,
        f.product_id,
        f.sku,
        f.currency,
        f.units_ordered,
        f.units_delivered,
        f.revenue_minor,
        f.baseline_qty,
        f.confidence_pct,
        f.adi,
        f.cv2,
        f.demand_category,
        f.stockout_events,
        f.updated_at
      FROM `${var.project_id}.${google_bigquery_dataset.datasets["features"].dataset_id}.${google_bigquery_table.sku_demand_daily.table_id}` AS f
      INNER JOIN `${var.project_id}.${google_bigquery_dataset.datasets["curated"].dataset_id}.${google_bigquery_table.tenant_access_principals.table_id}` AS p
        ON f.supplier_id = p.tenant_id
       AND f.cell_id = p.cell_id
      WHERE LOWER(p.principal_email) = LOWER(SESSION_USER())
    SQL
    use_legacy_sql = false
  }

  labels = merge(local.analytics_labels, { layer = "curated", domain = "tenancy", view = "authorized" })
}

# Authorize cross-dataset views to read from `raw` and `features` without granting direct table read to BI users
resource "google_bigquery_dataset_access" "authorize_raw_tenant_view" {
  project    = var.project_id
  dataset_id = google_bigquery_dataset.datasets["raw"].dataset_id

  view {
    project_id = var.project_id
    dataset_id = google_bigquery_dataset.datasets["curated"].dataset_id
    table_id   = google_bigquery_table.v_tenant_domain_events.table_id
  }
}

resource "google_bigquery_dataset_access" "authorize_features_tenant_view" {
  project    = var.project_id
  dataset_id = google_bigquery_dataset.datasets["features"].dataset_id

  view {
    project_id = var.project_id
    dataset_id = google_bigquery_dataset.datasets["curated"].dataset_id
    table_id   = google_bigquery_table.v_tenant_sku_demand_daily.table_id
  }
}

# Cell-level Row Access Policies for backend-go and ai-worker service accounts
resource "google_bigquery_row_access_policy" "cell_workload_policy" {
  for_each = local.cell_row_access_tables

  project          = var.project_id
  dataset_id       = google_bigquery_dataset.datasets[each.value.dataset_key].dataset_id
  table_id         = each.value.table_id
  policy_id        = "cell-${local.sanitized_cell_id}-${each.key}"
  filter_predicate = each.value.filter_predicate
  grantees = [
    "serviceAccount:${var.backend_sa_email}",
    "serviceAccount:${var.ai_worker_sa_email}",
  ]
}

# Optional per-tenant row-level security policies on domain_events for direct analyst groups
resource "google_bigquery_row_access_policy" "tenant_domain_events_policy" {
  for_each = var.tenant_row_access_policies

  project          = var.project_id
  dataset_id       = google_bigquery_dataset.datasets["raw"].dataset_id
  table_id         = google_bigquery_table.domain_events.table_id
  policy_id        = "tenant-${replace(each.key, "-", "_")}"
  filter_predicate = "cell_id = '${local.effective_cell_id}' AND supplier_id = '${each.value.tenant_id}'"
  grantees         = each.value.grantees
}

# 6. Vertex AI Pipeline Staging GCS Bucket (Cell-Regional, Uniform Access, No Public Access)
resource "google_storage_bucket" "vertex_staging" {
  name                        = local.effective_vertex_bucket
  location                    = var.region
  project                     = var.project_id
  storage_class               = "STANDARD"
  uniform_bucket_level_access = true
  public_access_prevention    = "enforced"

  versioning {
    enabled = true
  }

  lifecycle_rule {
    condition {
      age = 90
    }
    action {
      type = "Delete"
    }
  }

  lifecycle_rule {
    condition {
      age = 7
    }
    action {
      type = "AbortIncompleteMultipartUpload"
    }
  }

  dynamic "encryption" {
    for_each = var.enable_cmek ? [1] : []
    content {
      default_kms_key_name = var.kms_key_id
    }
  }

  labels = merge(local.analytics_labels, {
    purpose = "vertex-ai-staging"
  })
}

# 7. Service Account IAM Bindings (`roles/bigquery.dataEditor`, `roles/aiplatform.user`, GCS staging)
resource "google_bigquery_dataset_iam_member" "workload_bq_data_editor" {
  for_each = var.cell_scoped_iam ? { for p in local.dataset_iam_pairs : p.key => p } : {}

  project    = var.project_id
  dataset_id = google_bigquery_dataset.datasets[each.value.ds_key].dataset_id
  role       = "roles/bigquery.dataEditor"
  member     = "serviceAccount:${each.value.sa_email}"
}

resource "google_project_iam_member" "workload_bq_data_editor_project" {
  for_each = !var.cell_scoped_iam ? local.workload_principals : {}

  project = var.project_id
  role    = "roles/bigquery.dataEditor"
  member  = "serviceAccount:${each.value}"
}

resource "google_project_iam_member" "workload_bq_job_user" {
  for_each = local.workload_principals

  project = var.project_id
  role    = "roles/bigquery.jobUser"
  member  = "serviceAccount:${each.value}"
}

resource "google_project_iam_member" "workload_aiplatform_user" {
  for_each = local.workload_principals

  project = var.project_id
  role    = "roles/aiplatform.user"
  member  = "serviceAccount:${each.value}"
}

resource "google_storage_bucket_iam_member" "vertex_staging_object_admin" {
  for_each = local.workload_principals

  bucket = google_storage_bucket.vertex_staging.name
  role   = "roles/storage.objectAdmin"
  member = "serviceAccount:${each.value}"
}

resource "google_project_iam_member" "authorized_reader_job_user" {
  count = local.effective_authorized_reader != "" ? 1 : 0

  project = var.project_id
  role    = "roles/bigquery.jobUser"
  member  = local.effective_authorized_reader
}

resource "google_bigquery_dataset_iam_member" "authorized_reader_curated_viewer" {
  count = local.effective_authorized_reader != "" ? 1 : 0

  project    = var.project_id
  dataset_id = google_bigquery_dataset.datasets["curated"].dataset_id
  role       = "roles/bigquery.dataViewer"
  member     = local.effective_authorized_reader
}
