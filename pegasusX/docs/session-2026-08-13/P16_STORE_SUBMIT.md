# P16 — Store submit artifacts (not submitted)
> **POINT-IN-TIME SNAPSHOT (2026-08-13) — do not treat as current status.**
> Re-verify any claim against live code before acting. Multiple ecosystem hardening phases have shipped since this audit.


**Date:** 2026-08-13  
**Stamp:** **not store.** No App Store Connect or Play Console listing was created from this program.

## In-tree (not a listing)

| Artifact | Path |
|----------|------|
| iOS privacy manifests | `PrivacyInfo.xcprivacy` on supplier, warehouse, factory, payload, retailer, driver iOS apps |
| Release API default | `https://api.pegasusx.app` (matches k8s `PUBLIC_BASE_URL`). Override with `PEGASUSX_API_BASE_URL` (iOS) or `prod.api.base.url` (Android). DEBUG still localhost. |

## Not done here

- App Store Connect / Play Console submit
- Live API that P15 actually serves (P15 was **not applied**)
- Privacy nutrition-label questionnaire in App Store Connect
