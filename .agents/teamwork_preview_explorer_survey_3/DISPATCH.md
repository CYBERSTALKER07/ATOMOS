## 2026-09-16T13:16:07Z

You are teamwork_preview_explorer (Survey Specialist 3: Middleware, Global Pay, Events & Fleet Hub).
Your working directory is: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_3
Your parent is: teamwork_preview_orchestrator (conv ID: 755199e9-0b8c-404a-b2f0-93e7b22240ee)

MANDATORY FIRST STEP:
Read /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md completely before doing any other work.

TASK:
Perform a deep, read-only survey of middleware, payment gateway logic, events, and fleet endpoints in `pegasus.x/backend`:
1. Middleware & Routing:
   - Check how HTTP routing (Chi or standard router) and middleware are implemented in `pegasus.x/backend/internal/`.
   - How does JWT extraction work? Where are user/supplier ID and claims stored in `r.Context()`?
   - Design `RequireSupplierOnboardingCompleted` middleware:
     - Block operational endpoints with HTTP 428 Precondition Required (`{"error": "onboarding_incomplete", "onboarding_status": "...", "next_step": "..."}`) if `onboarding_status != 'COMPLETED'`.
     - Whitelist `/v1/auth/*` and `/v1/supplier/onboarding/*`.
2. Phased Onboarding Wizard Endpoints:
   - Step 1: `POST /v1/supplier/onboarding/products` (Add, Edit, Delete). Check validation: 17-digit statutory MXIK code regex/format, EAN-13 checksum/uniqueness, 64-bit integer tiyin price, 12% VAT.
   - Step 2: `POST /v1/supplier/onboarding/payment` (Cash default enabled; Global Pay `GLOBAL_PAY` corporate card gateway with service ID, secret key, and corporate card BIN validation - Uzbekistan B2B corporate card prefixes/BINs).
   - Step 3: `POST /v1/supplier/onboarding/complete` (validate at least 1 product active, payment configured; update status to 'COMPLETED', emit outbox & Redis event).
3. Warehouse & Fleet Management:
   - `POST/GET/PUT/DELETE /v1/supplier/warehouses`
   - Mandatory lat/lon (`DOUBLE PRECISION`).
   - Redis proximity cache invalidation (check Redis client in `pegasus.x/backend/internal/redis/` or similar) and emit `warehouse.relocated`.
   - Warehouse deletion guard: check stock > 0 or active orders -> HTTP 409 Conflict.
   - Trucks: `POST/GET /v1/supplier/warehouses/{id}/trucks` (license plate, capacity kg/m³, fuel type).
   - Payloaders: `POST/GET /v1/supplier/warehouses/{id}/payloaders` (name, phone, warehouse_id).
4. Realtime & Outbox:
   - Inspect `pegasus.x` transactional outbox / Redis streams / WebSocket hub mechanisms.

OUTPUT:
Write your comprehensive findings and API contract designs to `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_survey_3/report.md` and a standard `handoff.md`.
When finished, send a message to parent with the summary and report path.
