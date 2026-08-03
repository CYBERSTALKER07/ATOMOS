# Real Codebase Infrastructure & Architecture

<<<<<<< HEAD
> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


*Auto-generated from actual Terraform and infrastructure definitions.*

=======
*Auto-generated from actual Terraform and infrastructure definitions.*

>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
## Provisioned Resources
- `google_artifact_registry_repository (pegasusx)`
- `google_billing_budget (pegasusx_monthly)`
- `google_compute_network (pegasusx_vpc)`
- `google_container_cluster (pegasusx)`
- `google_monitoring_alert_policy (ai_worker_consumer_lag)`
- `google_monitoring_alert_policy (ai_worker_down)`
- `google_monitoring_alert_policy (ai_worker_not_ready)`
- `google_monitoring_alert_policy (backend_5xx_rate)`
- `google_monitoring_alert_policy (backend_kafka_lag)`
- `google_monitoring_alert_policy (optimizer_fallback_rate)`
- `google_monitoring_alert_policy (outbox_backlog_high)`
- `google_monitoring_alert_policy (spanner_cpu_high)`
- `google_monitoring_dashboard (ai_worker_launch)`
- `google_monitoring_dashboard (pilot_launch)`
<<<<<<< HEAD
=======
- `google_monitoring_notification_channel (budget_email)`
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
- `google_monitoring_uptime_check_config (ai_worker_health)`
- `google_monitoring_uptime_check_config (ai_worker_ready)`
- `google_project_iam_member (backend_redis)`
- `google_project_iam_member (backend_secrets)`
- `google_project_iam_member (backend_spanner)`
- `google_project_service (gke_apis)`
- `google_project_service (required_apis)`
- `google_redis_instance (cache)`
- `google_secret_manager_secret (adyen_webhook_secret)`
- `google_secret_manager_secret (apple_notarize_app_password)`
- `google_secret_manager_secret (apple_notarize_apple_id)`
- `google_secret_manager_secret (apple_notarize_team_id)`
- `google_secret_manager_secret (firebase_auth_enabled)`
- `google_secret_manager_secret (firebase_project_id)`
- `google_secret_manager_secret (global_pay_webhook_secret)`
- `google_secret_manager_secret (google_maps_api_key)`
- `google_secret_manager_secret (jwt_secret)`
- `google_secret_manager_secret (kafka_bootstrap_servers)`
- `google_secret_manager_secret (kafka_topic_main)`
- `google_secret_manager_secret (kafka_topic_realtime)`
- `google_secret_manager_secret (kafka_topic_spatial)`
- `google_secret_manager_secret (kafka_topic_webhooks)`
- `google_secret_manager_secret (stripe_webhook_secret)`
- `google_secret_manager_secret (tauri_signing_private_key)`
- `google_secret_manager_secret (tauri_updater_pubkey)`
- `google_secret_manager_secret (windows_codesign_password)`
- `google_secret_manager_secret (windows_codesign_pfx)`
- `google_secret_manager_secret_version (adyen_webhook_secret)`
- `google_secret_manager_secret_version (apple_notarize_app_password)`
- `google_secret_manager_secret_version (apple_notarize_apple_id)`
- `google_secret_manager_secret_version (apple_notarize_team_id)`
- `google_secret_manager_secret_version (firebase_auth_enabled)`
- `google_secret_manager_secret_version (firebase_project_id)`
- `google_secret_manager_secret_version (global_pay_webhook_secret)`
- `google_secret_manager_secret_version (google_maps_api_key)`
- `google_secret_manager_secret_version (jwt_secret)`
- `google_secret_manager_secret_version (kafka_bootstrap_servers)`
- `google_secret_manager_secret_version (kafka_topic_main)`
- `google_secret_manager_secret_version (kafka_topic_realtime)`
- `google_secret_manager_secret_version (kafka_topic_spatial)`
- `google_secret_manager_secret_version (kafka_topic_webhooks)`
- `google_secret_manager_secret_version (stripe_webhook_secret)`
- `google_secret_manager_secret_version (tauri_signing_private_key)`
- `google_secret_manager_secret_version (tauri_updater_pubkey)`
- `google_secret_manager_secret_version (windows_codesign_password)`
- `google_secret_manager_secret_version (windows_codesign_pfx)`
- `google_service_account (backend_runtime)`
- `google_service_account_iam_member (backend_wi)`
- `google_spanner_database (main)`
- `google_spanner_instance (ledger)`
- `google_storage_bucket (app_updates)`
- `google_storage_bucket_iam_member (public_updates)`
