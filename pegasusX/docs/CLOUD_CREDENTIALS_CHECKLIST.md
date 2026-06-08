# pegasusX Cloud Credentials Checklist (Boss Handoff)

## Not required for local Docker / SSMR green

- Spanner, Redis, Kafka cloud endpoints
- Google Maps keys (maps show placeholder without keys)
- Live payment gateway keys (GlobalPay.uz production test credentials + dev webhook secrets in `.env.ssmr.example`)
- Terraform apply

## Required for staging / production (GCP)

### Google Cloud

- GCP `project_id` with billing enabled
- Terraform state bucket (default: `gs://void-terraform-state` — confirm or override)
- Workload Identity: GKE → Spanner, Redis, Secret Manager
- Cloud Spanner instance + database
- Memorystore Redis (VPC-attached)
- Managed Kafka or Confluent bootstrap + SASL credentials
- Artifact Registry push for `backend-go` and `ai-worker` images
- Cloud Monitoring notification channels for `infra/terraform/observability.tf`

### Identity & push

- `FIREBASE_PROJECT_ID`
- Firebase Admin SDK service account JSON (server)
- iOS: APNs key in Firebase; Android: FCM + `google-services.json` per app
- Set `FIREBASE_AUTH_ENABLED=true` on backend

### Maps (Android native)

- `DRIVER_ANDROID_MAPS_API_KEY` or `MAPS_API_KEY` per `apps/driver-app-android/README.md`
- iOS MapKit: Apple Developer Program only

### Payments (GlobalPay.uz Setup)

- Secret Manager: `GLOBAL_PAY_WEBHOOK_SECRET`
- Provider API keys: GlobalPay.uz keys (Note: stubbed out in `globalpay.go` until contract signed)
- Webhook URLs: `https://<api-host>/v1/webhooks/...`
- `AIRWALLEX_DIRECT_EXECUTION_ENABLED=true` only if Airwallex is live

### App distribution

- Apple Developer Program (iOS apps)
- Google Play Console (Android apps)
- Tauri code-signing (supplier / warehouse / factory / retailer desktop)
- Expo EAS (payload-terminal) if OTA required

## Terraform inputs (minimum)

| Variable | Purpose |
|----------|---------|
| `project_id` | GCP project |
| `tenant_slug` | Resource namespace |
| `region` | Primary region (default `asia-south1`) |
| `ai_worker_monitoring_host` | Optional uptime checks |

## Post-provision commands

```bash
cd pegasusX/infra/terraform && terraform init && terraform apply
cd pegasusX && go run ./apps/backend-go/cmd/setup
cd pegasusX && make validate-launch-readiness
```
