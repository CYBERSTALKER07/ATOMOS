## 2026-08-21T00:26:28Z

Explorer for Requirement 2 (R2: Geography, Maps, and Security).
Repository Root: /Users/shakhzod/Desktop/V.O.I.D
Authoritative Request: /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md

Tasks:
1. Locate all H3 geospatial resolution usages across matching writers, settlement, and perimeter logic.
   - Check where H3 resolution 7 must be enforced in matching writers.
   - Check where resolution 9 is used in settlement/perimeter logic and identify if it needs a distinct named field (e.g., `H3Res9Index` or similar distinct field name) rather than ambiguous/shared fields.
2. Locate all geocode API endpoints and handlers in the backend.
   - Check authentication middleware coverage. Verify where `RequireRole` or `RequireAnyAuthenticated` is missing and must be added.
   - Check how country-bias is passed, parsed, or missing in geocode requests/handlers, and identify how it should be implemented.
3. Locate the factory fleet list endpoint/handler and its current data source.
   - Check the Spanner schema and models for `FactoryTruckManifests`.
   - Identify the exact changes needed to switch the factory fleet list to pull from Spanner `FactoryTruckManifests`.
