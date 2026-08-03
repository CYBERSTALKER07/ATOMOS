# Orders Page Overrides

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



- `PageChrome` with `icon="orders"`, `skeletonVariant="table"`.
- Split list/detail layout preserved; `desk-table-wrap` on queue table.
- Preorder / cancel / delivery-proposal actions use `portal-btn`.
