# Substance Gate — Client sign-off (2026-08-05)

```text
Environment: SSMR
API base: https://api-ssmr.pegasusx.app
API SoT: artifacts/SUBSTANCE_GATE_API_SIGNOFF_2026-08-04.md (API PASS all roles)
Date: 2026-08-05  Operator: agent (Client Parity Closure)

Role          API   WEB             AND             IOS
Retailer      PASS  READY_FOR_WALK  READY_FOR_WALK  READY_FOR_WALK
Supplier      PASS  READY_FOR_WALK  READY_FOR_WALK  READY_FOR_WALK
Warehouse     PASS  READY_FOR_WALK  READY_FOR_WALK  READY_FOR_WALK
Factory       PASS  READY_FOR_WALK  READY_FOR_WALK  READY_FOR_WALK
Driver        PASS  N/A             READY_FOR_WALK  READY_FOR_WALK
Payload       PASS  N/A             READY_FOR_WALK  READY_FOR_WALK
```

**Legend:** `READY_FOR_WALK` = client-policy HTTP 200 + engineering unblocks landed; interactive PASS/FAIL still requires a human Phase-2 session per `docs/SUBSTANCE_GATE.md` §5. Do **not** treat READY_FOR_WALK as Done.

## Preflight

| Check | Result |
|-------|--------|
| `GET /healthz` | OK `{"status":"ok"}` |
| `GET /v1/platform/client-policy` × 6 roles × web/android/ios | All HTTP **200** |
| API marker gate (prior) | PASS (`SUBSTANCE_GATE_API_SIGNOFF_2026-08-04`) |

## Engineering unblocks (this program)

| Item | Status |
|------|--------|
| P0-4 iOS offline geofence → silent success | Fixed (`FleetServiceLive` + `isNetworkEnqueueable`) |
| AUTHORIZE_BYPASS photo_url | Wired desktop + Android + iOS |
| Return-policy settings | Portal + mobile supplier/WH |
| Driver PoD photo gate (credit leave) | Wired Android + iOS |
| Empty chart theatre | Unmounted / SpendAnalytics thin-wired |
| Portal i18n bootstrap | Partial — shell + dashboard/orders PageChrome on 4 portals; not full i18n / no mobile |

## Minimal walk pack (operator)

Run against SSMR after login with real Firebase/JWT as available.

1. **Retailer WEB:** cart → checkout (`ORDER`/`CHECKOUT_PREVIEW`) → file claim with photo → shop-closed AUTHORIZE_BYPASS with photo  
2. **Supplier WEB:** return-policy settings save → claim approve  
3. **Warehouse WEB:** return-policy section save → dispatch execute  
4. **Factory WEB:** supply / manifest lifecycle  
5. **Driver AND + IOS:** settle path; geofence 403 must **not** queue offline; credit leave requires PoD photo  
6. **Payload AND + IOS:** seal / inject  
7. Spot WS/inbox on two roles  

Fill PASS/FAIL into the matrix above when complete; rename status from READY_FOR_WALK.

## Still ops / product deferred (not client parity)

- Global Pay card SUCCESS (`PX_E2E_PAYMENT_CARD_SUCCESS_*`)
- Firebase SMS / SHA-1 / APNs
- Quantity negotiations, Soliq OFD UI, offline POS (product flags)

## Theatre notes closed this pass

- Supplier/WH claim-window portal UX — **settings screens shipped** (was open on API sign-off)
