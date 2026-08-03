# Shop-Closed End-to-End SOP

> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



Operational playbook for the shop-closed escalation chain across driver, retailer, and supplier.

**SSMR proof:** `PX_E2E_SHOP_CLOSED_OK`  
**Related:** [`DRIVER_SUPPORT_PLAYBOOK.md`](./DRIVER_SUPPORT_PLAYBOOK.md), [`DELIVERY_ESCALATION_POLICY.md`](./DELIVERY_ESCALATION_POLICY.md), [`REAL_WORLD_CASE_MATRIX.md`](./REAL_WORLD_CASE_MATRIX.md)

---

## Roles and surfaces

| Role | Primary UI | Backend |
|------|------------|---------|
| DRIVER | ShopClosed wait screen, mission detail | `order/shop_closed.go`, driver edges |
| RETAILER | tracking, OPEN_NOW action | `retailerroutes` |
| SUPPLIER | deep link `/exceptions/shop-closed` (nav removed; API wired) | `POST /v1/supplier/shop-closed/resolve` |

**Config:** `SHOP_CLOSED_GRACE_MINUTES` (default `5`) before `SHOP_CLOSED_ESCALATED`.

---

## Happy path

1. Driver arrives → shop closed → reports via driver app (`ARRIVED_SHOP_CLOSED`).
2. Retailer receives WS/notification → opens shop or marks OPEN_NOW.
3. Driver receives `SHOP_CLOSED_RESPONSE` → continues offload/payment path.
4. Order completes via normal cash/card edges.

---

## Escalation path

1. Grace timer elapses → `SHOP_CLOSED_ESCALATED` fans to supplier + retailer + driver.
2. Supplier ops opens exceptions deep link → reviews retailer coordinates and history.
3. Options:
   - Ask retailer to open (retailer action).
   - Supplier resolve / bypass per policy (`bypass-offload` driver path when authorized).
4. Never ask driver to use admin portal mid-route.

---

## Support triage

| Symptom | First action | Escalate when |
|---------|--------------|---------------|
| Driver stuck in wait | Confirm retailer notified; check WS pill | Grace elapsed + no OPEN_NOW |
| Retailer cannot open | Verify profile coordinates | Geofence mismatch |
| Payment stuck after bypass | Check `collect-cash` / webhook | [`PAYMENT_EXCEPTION_SOP.md`](./PAYMENT_EXCEPTION_SOP.md) |

---

## Logging

Record: `order_id`, `driver_id`, `retailer_id`, grace start time, resolution action, final order status.
