# Survey Explorer 2 Context: Frontend Shared Monorepo Package Consolidation

Mission:
Survey the frontend architecture and packages in `pegasus.x`:
- Inspect `contracts/` vs `packages/types/` (or other type packages) to identify type drift, missing DTOs, and duplicate typings.
- Inspect `packages/pulse-ui/` and `packages/ui-kit/` (and any other shared packages in `packages/`).
- Inspect `apps/warehouse-desktop` (and other desktop/mobile apps like `retailer-desktop`, `driver-mobile`, etc.) to locate repeated UI layouts (Navigation Rail, Density Metric Cards, Context Inspector Drawer).
- Check `apps/warehouse-desktop/package.json` for unused or stale dependencies (specifically `firebase`).
- Verify current TypeScript compilation status across packages (`pnpm --filter @pegasusx/types build`, etc.).
- Propose a concrete consolidation plan for extracting control tower primitives into `@pegasusx/pulse-ui` and `@pegasusx/ui-kit`, syncing contracts, and purging `firebase`.
- Write your comprehensive findings to `handoff.md` in your working directory.
