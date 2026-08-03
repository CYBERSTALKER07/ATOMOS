# Launch Readiness Runbook

<<<<<<< HEAD
> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



This run book is the release-owner checklist for the currently implemented pegasusX engine. It covers the support, evidence, rollback, and launch-support gates for the core supplier, retailer, driver, warehouse, factory, payload, payment, telemetry, and ai-worker seams. It does not mark deferred AI recommendation product surfaces as complete.
=======
Release ownership: platform engineering + on-call SRE.
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8

Preflight:
- make test-ssmr-infra
- make validate-ai-worker-k8s
- make validate-launch-readiness
- make p0-preflight

rollback: revert deployment image tag and re-run cloud-smoke-ssmr.

launch support: follow hypercare checklist in docs/P0_LAUNCH_CHECKLIST.md.

