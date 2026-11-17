# GS-C3 — EU cell stack (plan only). Project factory: cells/eu/project/.
# Reject: two cells in pegasus-503013; GSM auto {}; shared pegasusx/ssmr state;
# UZ Spanner backup restore. Empty adapters OK (new JWT, commercial fiscal).

project_id  = "pegasusx-cell-eu"
tenant_slug = "eu"
environment = "cell"
region      = "europe-west1"

cell_id                = "eu"
api_hostname           = "api-eu.pegasusx.app"
k8s_namespace          = "pegasusx"
terraform_state_prefix = "pegasusx/cell-eu"
gsm_regional_only      = true
vpc_custom_mode        = true
cell_scoped_iam        = true

vpc_subnet_cidr   = "10.30.0.0/20"
vpc_pods_cidr     = "10.31.0.0/16"
vpc_services_cidr = "10.32.0.0/20"

enable_gke           = true
enable_managed_kafka = true

# Empty topic vars derive cell-eu.events.* (see main.tf locals).
kafka_topic_main             = ""
kafka_topic_main_dlq         = ""
kafka_topic_spatial          = ""
kafka_topic_realtime         = ""
kafka_topic_webhooks         = ""
kafka_topic_freeze_locks     = ""
kafka_topic_inventory_import = ""

# New JWT — mint with scripts/mint_cell_jwt.sh; do not copy the UZ secret.
allow_uz_backup_restore        = false
jwt_secret                     = ""
firebase_auth_enabled          = false
enable_observability_resources = false
billing_account_id             = ""

# M0.5 Analytics, Vertex AI & WORM Audit Storage (cell-regional europe-west1)
worm_audit_bucket_name        = "pegasusx-cell-eu-worm-audit"
vertex_staging_bucket_name    = "pegasusx-cell-eu-vertex-staging"
bigquery_dataset_prefix       = ""
enable_analytics_cmek         = false
enable_analytics_iam_bindings = true
