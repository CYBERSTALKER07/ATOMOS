/**
 * Module: Analytics & ML Platform
 * Output declarations
 */

output "dataset_ids" {
  description = "Map of BigQuery dataset IDs keyed by layer (raw, curated, features)."
  value       = { for k, ds in google_bigquery_dataset.datasets : k => ds.dataset_id }
}

output "table_ids" {
  description = "Map of fully-qualified BigQuery table identifiers created by this module."
  value = {
    domain_events             = "${google_bigquery_dataset.datasets["raw"].dataset_id}.${google_bigquery_table.domain_events.table_id}"
    ontology_objects          = "${google_bigquery_dataset.datasets["raw"].dataset_id}.${google_bigquery_table.ontology_objects.table_id}"
    gl_entries                = "${google_bigquery_dataset.datasets["curated"].dataset_id}.${google_bigquery_table.gl_entries.table_id}"
    gl_postings               = "${google_bigquery_dataset.datasets["curated"].dataset_id}.${google_bigquery_table.gl_postings.table_id}"
    recon_runs                = "${google_bigquery_dataset.datasets["curated"].dataset_id}.${google_bigquery_table.recon_runs.table_id}"
    sku_demand_daily          = "${google_bigquery_dataset.datasets["features"].dataset_id}.${google_bigquery_table.sku_demand_daily.table_id}"
    retailer_credit_signals   = "${google_bigquery_dataset.datasets["features"].dataset_id}.${google_bigquery_table.retailer_credit_signals.table_id}"
    tenant_access_principals  = "${google_bigquery_dataset.datasets["curated"].dataset_id}.${google_bigquery_table.tenant_access_principals.table_id}"
    v_tenant_domain_events    = "${google_bigquery_dataset.datasets["curated"].dataset_id}.${google_bigquery_table.v_tenant_domain_events.table_id}"
    v_tenant_gl_entries       = "${google_bigquery_dataset.datasets["curated"].dataset_id}.${google_bigquery_table.v_tenant_gl_entries.table_id}"
    v_tenant_sku_demand_daily = "${google_bigquery_dataset.datasets["curated"].dataset_id}.${google_bigquery_table.v_tenant_sku_demand_daily.table_id}"
  }
}

output "vertex_staging_bucket_name" {
  description = "GCS bucket name for Vertex AI pipeline staging."
  value       = google_storage_bucket.vertex_staging.name
}

output "vertex_staging_bucket_url" {
  description = "GCS URL (gs://...) for Vertex AI pipeline staging."
  value       = google_storage_bucket.vertex_staging.url
}
