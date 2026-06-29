# Retailer Receiving Windows Guide

Non-technical guide for retailers and account managers: **receiving windows are a contract with drivers**, not just a profile preference.

**Backend:** `Retailers.ReceivingWindowOpen/Close` on `PUT /v1/retailer/profile`  
**SSMR:** `PX_E2E_RETAILER_RECEIVING_WINDOW_OK`  
**Surfaces:** desktop Settings, Android `AccountProfileScreen`, iOS `AccountProfileView`

---

## What receiving windows mean

- Drivers and dispatch planning treat your window as **when someone can accept delivery**.
- Orders created outside the window may still dispatch if policy allows, but **shop-closed** and geofence flows become more likely if nobody is present.
- Windows are snapshotted on new orders (`receiving_window_open/close` on order row).

---

## Retailer responsibilities

1. Set accurate open/close times per location (include lunch closures if applicable).
2. Update windows before holidays or Ramadan hour changes.
3. When driver reports shop closed:
   - Open the retailer app → mark shop open / respond to notification promptly.
4. Keep delivery address coordinates accurate (Settings → company/location).

---

## What retailers should not expect

- Drivers cannot change your window from the driver app.
- Warehouse cannot override your window without supplier policy changes.
- Negotiation on quantity is **disabled** in v1 (410).

---

## Support scripts

| Retailer says | Response |
|---------------|----------|
| “Driver came at wrong time” | Check window + order snapshot; verify dispatch SLA |
| “App says shop closed” | Guide to OPEN_NOW; link [`SHOP_CLOSED_E2E_SOP.md`](./SHOP_CLOSED_E2E_SOP.md) |
| “Can't checkout” | Check zone (`zone_miss`) and stock caps |

---

## Training checklist

- [ ] Registration captures receiving window
- [ ] Manager knows how to edit window on mobile + desktop
- [ ] Night-shift staff know OPEN_NOW action for shop-closed
