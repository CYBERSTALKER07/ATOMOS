# Detailed Migration Status & Execution Plan

*Last Updated: 2026-07-21*

## 1. Completed Milestones (✅)
- **Local wire-ready validation:** System successfully tested in local orchestration.
- **GCP Identity & Project Foundation:** Core GCP project established with proper IAM roles.
- **Terraform Provisioning:** The following infrastructure has been provisioned via IaC:
  - Cloud Spanner Instance
  - Redis (Instance exists)
  - GKE Cluster
  - VPC Network
  - Artifact Registry (AR)
  - Google Secret Manager (GSM)
  - Budget & Billing Alerts

## 2. Current Active Task (🔄 BLOCKED)
- **Spanner Full Schema + Fiscal Migrations:** We need to execute the schema migrations for the ~16 core tables.
  - *Blocker:* Waiting on GCP Support to grant quota limit increases before this can be safely executed. 

## 3. Immediate Next Steps (Upon Quota Approval)
Once Support unblocks the Spanner quota, the execution order is strictly as follows:
1. **Redis AUTH:** Wire Redis authentication securely to GSM and validate in-VPC connectivity (PING).
2. **Kafka Initialization:** Provision Confluent Kafka topics and wire credentials to GSM.
3. **Secret Injection:** Configure External Secrets to pull from GSM into the GKE cluster.
4. **Kube Credentials:** Bind kubectl context and credentials for deployments.
5. **CI/CD Pipeline:** Build and push all service images to Artifact Registry.

## 4. Deployment & Validation Phase
1. Deploy `API`, `worker`, and `ai-worker` to GKE.
2. **Cloud Smoke Test:** Execute an end-to-end simulated order flow (`Order` -> `FISCALIZING` -> `COMPLETED`).
3. Configure Ingress, DNS, and TLS certificates.
4. Wire Firebase (OTP/FCM), Maps API (Geocode), and Global Pay staging webhooks.
5. Verify OFD Sandbox integration post-smoke test.
6. Point client applications and portals at the new staging environment.
7. Finalize HPA configuration and observability dashboards.
8. Prep for Production Promotion.

## 5. Cleanup
- Destroy legacy `void-494000` infrastructure to prevent duplicate billing.
