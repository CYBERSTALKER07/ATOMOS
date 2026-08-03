# PegasusX Platform

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.


PegasusX is the next-generation wire-ready logistics and retail platform.

## Current Project State: GCP Migration (Phase 2)
We are currently executing a comprehensive migration to Google Cloud Platform (GCP). The local development environment is fully wire-ready, and foundational cloud infrastructure has been provisioned via Terraform.

### Active Blocker
**Google Cloud Quota Limits:** We are currently paused on Spanner schema provisioning and fiscal migrations (~16 tables) pending a quota increase from GCP Support.

## Core Infrastructure Stack
- **Database:** Google Cloud Spanner (Global consistency, high availability)
- **Caching & Pub/Sub:** Redis (VPC-peered) & Confluent Kafka
- **Compute:** Google Kubernetes Engine (GKE) for API, Worker, and AI-Worker services
- **Secrets Management:** Google Secret Manager (GSM) + External Secrets Operator
- **Routing & Auth:** Ingress with TLS, Firebase Auth (OTP) + FCM

For the full detailed status, please see [Current Status](context/current_status.md) and [Architecture](context/architecture.md).
