# REAL_WORLD_CASE_MATRIX

<<<<<<< HEAD
> [!NOTE]
> **Current Project State:** GCP Migration (Phase 2)
> *Status:* Re-provisioning GKE Autopilot to GKE Standard (pd-standard) to resolve SSD quota limits. Migrations pending quota unblock.



> **Purpose:** Map operational edge cases to role surfaces, backend guards, and owner SOPs.  
> **Screen routes:** [`ROLE_ROW_PARITY_MATRIX.md`](./ROLE_ROW_PARITY_MATRIX.md)  
> **Ecosystem spec:** [`FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md`](./FULL_SYSTEM_PARITY_AND_ECOSYSTEM_MASTER_PLAN.md)
=======
Operational cases for hypercare and Retail OS. Expand with tickets as pilots run.
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8

## Cross-ecosystem (prior)

| Case | Expected behavior |
|------|-------------------|
| Shop-closed at doorstep | Retailer respond codes; inventory release on cancel |
| Partial offload | Line qty + proximity unlock gates |
| Claim after COMPLETED | Window + media; chargeback spine |
| Gateway degraded | Honest error; no fake success |

## Retail OS (Phases 0–6)

| Case | Expected behavior |
|------|-------------------|
| Solo owner never opens Capabilities | Full CORE works; no staff tables required beyond bootstrap OWNER |
| Enable POS without STORE_STOCK | Dependency block **or** auto-enable stock on first register (product path) |
| Enable CUSTOMER_ASSIST without SECTIONS/TEAM | Hard deps auto-enabled on first ticket or BLOCKED via pack API |
| Partial truck delivery + claim | Receive accepted qty only; no overstatement of OnHand |
| Two cashiers same SKU last unit | Spanner/memory txn; second sale fails cleanly |
| Manager void after shift closed | Policy: void in open session / manager perm; post-period adjust documented |
| Staff fired mid-shift | Deactivate blocks new actions; open session → manager force-close |
| Multi-location order to store B, staff of A | Location scope fail closed |
| Delivery when all receivers off shift | Owner/admin can still dock |
| Negative cash variance | Shift/POS close stores variance; alert if ≥ threshold |
| Network drop at dock QR | Online-required confirm; offline honesty badge |
| Price change mid-POS cart | Price locked at line add / revalidate on pay |
| Forgot clock-out | Auto-close after `max_shift_hours`; reclock required for POS |
| POS open with SHIFTS + require_shift | `clock_in_required` without open time entry |
| Assist ticket no section staff | Notifies owners/managers + org inbox when NotificationWriter wired |
| Reports with no POS/stock | Empty modules, not fake series; pack may auto-enable on first GET |
| Control Tower cold start | `empty: true` — no mock charts or demo supplier ids |

## Ecosystem hardening (see `ECOSYSTEM_HARDENING_GAP_PLAN.md`)

| Case | Expected behavior |
|------|-------------------|
| Second supplier different zone | Perimeter keys per supplier; checkout uses order’s supplier |
| Supplier CT as real tenant | Never `sup-demo-1` / mock charts |
| Shop-closed after DDL | Grace/proximity columns persist; timeout matrix fires |
| Credit HIGH + enforce on | CREDIT_LEAVE blocked; audit |
| Rescue over capacity | Reject or split with residual warning |
| Offline nonce replay | Second sync rejected |
| Doorstep cash short | Bag recon line auto-seeded |
| Temp breach with qty | Claim + WH reverse OPEN |
| Fiscal stuck | Observability alert (when enabled) |

## Intentional product deferrals

| Item | Notes |
|------|-------|
| Offline POS queue | Split-brain risk; v1 online-required |
| Planogram vision | v2+ — `PLANOGRAM_VISION_PLAN.md` |
| Auto-order execution worker | Settings durable; worker separate epic |
| Family → Team forced migrate | Legacy list may coexist |
