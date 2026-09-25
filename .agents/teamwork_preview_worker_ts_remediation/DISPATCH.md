## 2026-09-25T17:32:02Z
You are teamwork_preview_worker_ts_remediation.
Working directory: /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_worker_ts_remediation
Workspace root: /Users/shakhzod/Desktop/V.O.I.D
Authoritative Request: Read /Users/shakhzod/Desktop/V.O.I.D/.agents/ORIGINAL_REQUEST.md completely.
Full Explorer Investigation & Roadmap: Read /Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_ts_remediation/handoff.md completely.
Victory Auditor Report: Read /Users/shakhzod/Desktop/V.O.I.D/.agents/victory_auditor_orch_4/handoff.md completely.

Write Ownership:
- Configuration and source files in `pegasusX/` and `pegasus/` as detailed in the Explorer's handoff.
- Scripts in `scripts/verify_all_16_apps_typecheck.sh`.
- Do NOT touch `pegasus.x/` (it is already 100% green).

Objective:
Execute the complete, actionable remediation plan from section 5 of `/Users/shakhzod/Desktop/V.O.I.D/.agents/teamwork_preview_explorer_ts_remediation/handoff.md`:

1. Code & Configuration Changes:
   - Fix JSX Syntax in `pegasus/apps/warehouse-portal/app/vehicles/page.tsx:137-183` (add missing closing `</div>` before line 179).
   - In `pegasusX/packages/ui-maps/package.json`: remove `@types/mapbox-gl`, add `"@types/geojson": "^7946.0.14"`.
   - In `pegasus/apps/admin-portal/package.json`: remove `@types/mapbox-gl`.
   - In `pegasus/apps/retailer-app-desktop/package.json`: remove `@types/mapbox-gl`, add `"framer-motion": "^12.38.0"`.
   - In `pegasusX/apps/payload-terminal/package.json`: add `"@types/node": "^20.0.0"` to devDependencies.
   - In `pegasus/apps/payload-terminal/package.json`: add `"@types/node": "^20.0.0"` to devDependencies.
   - In `pegasusX/apps/factory-portal/package.json`: change `"@pegasusx/ui-maps": "workspace:^"` to `"workspace:*"`.
   - Strict Mode Type Annotations:
     - `pegasusX/packages/ui-kit/src/desktop/VirtualScrollList.tsx`: add `(index: number, item: T)` annotations.
     - `pegasusX/packages/ui-maps/src/HexagonalControlTowerMap.tsx`: add `(d: { hex: string; count: number })` annotations.
     - `pegasusX/packages/ui-maps/src/GenericFleetLiveMap.tsx`: add `(ref: any)` annotation.
     - `pegasusX/apps/supplier-portal/app/api/api/[...path]/route.ts` & `ws-session/route.ts`: add `(value: string, key: string)` annotations.
     - `pegasusX/apps/supplier-portal/components/LiveOpsMap.tsx`: add `(ref: any)` and `(evt: any)` annotations.
     - `pegasusX/apps/supplier-portal/components/DispatchPreviewMap.tsx`: add `(ref: any)` annotation.
     - `pegasusX/apps/payload-terminal/components/ManifestWorkspaceScreen.tsx`: add `({ data }: { data: string })` annotations.

2. Dependency & Dist Restoration:
   - Run `pnpm install --no-frozen-lockfile --force` in `pegasusX`.
   - Clean stale `@types/mapbox-gl` symlinks in `pegasusX/node_modules/@types/mapbox-gl`.
   - Reinstall dependencies in `pegasus/apps/admin-portal`, `warehouse-portal`, `factory-portal`, `retailer-app-desktop`, `payload-terminal` (using `pnpm install --force 2>/dev/null || npm install --force`) and remove any `@types/mapbox-gl` stubs.

3. Complete 16-Application Verification:
   - Create and run `/Users/shakhzod/Desktop/V.O.I.D/scripts/verify_all_16_apps_typecheck.sh` (as defined in section 6 of explorer handoff).
   - Ensure EVERY SINGLE ONE of the 16 applications exits with code 0 on `tsc --noEmit`:
     - Group 1: pegasus.x (5 apps: supplier-desktop, warehouse-desktop, retailer-desktop, payloader-tablet, telegram-miniapp)
     - Group 2: pegasusX (6 apps: admin-portal, retailer-app-desktop, supplier-portal, warehouse-portal, factory-portal, payload-terminal)
     - Group 3: pegasus (5 apps: admin-portal, warehouse-portal, factory-portal, retailer-app-desktop, payload-terminal)
   - Re-run UX audit scanner to confirm 0 findings and score 95/100:
     `python3 /Users/shakhzod/.gemini/antigravity-cli/brain/ddb82dce-4f0b-4193-b133-2e4dcf9308ea/scratch/audit_scanner.py`
