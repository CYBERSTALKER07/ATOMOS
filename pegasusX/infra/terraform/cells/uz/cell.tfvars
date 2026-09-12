# GS-C2 — named UZ cell (live project). Plan only; do not apply from the catalog.
# Matches the live cell attributes. GSM/VPC stay auto so an accidental apply
# does not ForceNew JWT/PSP secrets or the auto-mode VPC.

project_id  = "pegasus-503013"
tenant_slug = "staging"
environment = "staging"
region      = "asia-south1"

cell_id                = "uz"
api_hostname           = "api.pegasusx.app"
k8s_namespace          = "pegasusx"
terraform_state_prefix = "pegasusx/ssmr"
gsm_regional_only      = false
vpc_custom_mode        = false
cell_scoped_iam        = true

enable_gke           = true
enable_managed_kafka = true

# Explicit topics keep today's staging names (empty would derive cell-uz.events.*).
kafka_topic_main             = "staging.events.orders"
kafka_topic_main_dlq         = "staging.events.orders-dlq"
kafka_topic_spatial          = "staging.events.spatial"
kafka_topic_realtime         = "staging.events.realtime"
kafka_topic_webhooks         = "staging.events.webhooks"
kafka_topic_freeze_locks     = "staging.events.freeze-locks"
kafka_topic_inventory_import = "staging.events.inventory-import"

firebase_auth_enabled          = true
enable_observability_resources = false
jwt_secret                     = ""
