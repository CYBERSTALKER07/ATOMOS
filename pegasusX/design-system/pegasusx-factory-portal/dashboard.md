# Dashboard Page Overrides

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



- Wrap content in `PageChrome` with `icon="dashboard"` and `skeletonVariant="dashboard"`.
- KPI grid: bento cards with 44px+ tap targets; link to loading-bay, transfers, manifests.
- Refresh action in `PageChrome` actions slot.
- Use `desk-card` / portal tokens; no Geist-specific font variables.
