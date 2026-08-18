import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const here = dirname(fileURLToPath(import.meta.url));
const root = join(here, "../../../..");
const apps = join(root, "apps");

function src(rel: string): string {
  const path = join(root, rel);
  expect(existsSync(path), path).toBe(true);
  return readFileSync(path, "utf8");
}

/** Dated skips are the U9 exit when a role-row cell is not a live board. */
const DATED_SKIPS_2026_08_16 = {
  "U8-admin-ios": "2026-08-16 platform admin is web-only",
  "U8-admin-android": "2026-08-16 platform admin is web-only",
  "U7-driver-web": "2026-08-16 driver has no portal",
  "U3-plan-brain-non-supplier": "2026-08-16 Plan & Brain is supplier row only",
  "U4-truck-duty-filter": "2026-08-16 warehouse truck-duty chip has no vehicles duty filter",
  "U5-vehicle-driver-filter": "2026-08-16 factory vehicle/driver chips have no fleet key filter",
  "topology-polygon-edit": "2026-08-16 desktop handoff",
  "pin-draw": "2026-08-16 desktop handoff",
  "edi-cert-upload": "2026-08-16 desktop handoff",
  "playbook-authoring": "2026-08-16 desktop handoff",
} as const;

describe("GS-U9 role-row + responsive lock", () => {
  it("records dated skips instead of inventing Wired", () => {
    for (const [key, note] of Object.entries(DATED_SKIPS_2026_08_16)) {
      expect(note.startsWith("2026-08-16"), key).toBe(true);
    }
    expect(Object.keys(DATED_SKIPS_2026_08_16)).toHaveLength(10);
  });

  it("U2 supplier chip uses the same status key on web, iOS, Android", () => {
    const web = src("apps/supplier-portal/app/(portal)/dashboard/page.tsx");
    const iosDash = src("apps/supplier-app-ios/SupplierApp/Views/Dashboard/DashboardView.swift");
    const iosVm = src("apps/supplier-app-ios/SupplierApp/ViewModels/OrdersViewModel.swift");
    const androidDash = src("apps/supplier-app-android/app/src/main/java/com/pegasusx/supplier/ui/screens/dashboard/DashboardScreen.kt");
    const androidVm = src("apps/supplier-app-android/app/src/main/java/com/pegasusx/supplier/ui/viewmodel/OrdersViewModel.kt");
    expect(web).toMatch(/\/orders\?status=/);
    expect(iosDash).toMatch(/onSelect:\s*\{\s*commandJump/);
    expect(iosDash).toMatch(/horizontalSizeClass/);
    expect(iosVm).toMatch(/func resolveSupplierOrdersQuery/);
    expect(iosVm).toMatch(/func applyCommandStatus/);
    expect(androidDash).toMatch(/onSelect = onOpenOrderStatus/);
    expect(androidVm).toMatch(/fun resolveSupplierOrdersQuery/);
    expect(androidVm).toMatch(/fun setCommandStatus/);
  });

  it("U3 Plan & Brain tabs stay on the supplier row", () => {
    const web = src("apps/supplier-portal/app/(portal)/planning/page.tsx");
    const ios = src("apps/supplier-app-ios/SupplierApp/Views/Planning/PlanningBrainView.swift");
    const android = src("apps/supplier-app-android/app/src/main/java/com/pegasusx/supplier/ui/screens/planning/PlanningBrainScreen.kt");
    expect(web).toMatch(/params\.get\("tab"\)/);
    expect(ios).toMatch(/gs-u-plan-brain-tabs/);
    expect(android).toMatch(/PLANNING_BRAIN|gs-u-plan-brain|Plan & Brain|Digital Brain/);
  });

  it("U4 warehouse chip uses the same state key on web, iOS, Android", () => {
    const web = src("apps/warehouse-portal/app/page.tsx");
    const ios = src("apps/warehouse-app-ios/WarehouseApp/Views/Dashboard/DashboardView.swift");
    const android = src("apps/warehouse-app-android/app/src/main/java/com/pegasusx/warehouse/ui/screens/dashboard/DashboardScreen.kt");
    expect(web).toMatch(/\/orders\?state=/);
    expect(ios).toMatch(/onSelect:\s*\{\s*commandJump/);
    expect(ios).toMatch(/horizontalSizeClass/);
    expect(android).toMatch(/WarehouseRoutes\.orders\(/);
    expect(android).not.toMatch(/FISCAL_FAILED.*FACTORY_TRANSFER/);
  });

  it("U5 factory chip keeps the factory plane and jumps transfers?state=", () => {
    const web = src("apps/factory-portal/app/page.tsx");
    const ios = src("apps/factory-app-ios/FactoryApp/Views/Dashboard/DashboardView.swift");
    const android = src("apps/factory-app-android/app/src/main/java/com/pegasusx/factory/ui/screens/dashboard/DashboardScreen.kt");
    expect(web).toMatch(/\/transfers\?state=/);
    expect(web).toMatch(/gs-u-factory-trucks/);
    expect(web).toMatch(/Last-mile retailer IN_TRANSIT is not a factory truck/);
    expect(ios).toMatch(/onOpenTransfers/);
    expect(ios).toMatch(/gs-u-factory-trucks/);
    expect(android).toMatch(/FactoryRoutes\.transfers\(/);
    expect(android).toMatch(/FactoryRoutes\.LOADING_BAY/);
    expect(android).toMatch(/FACTORY_TRANSFER_STATES/);
  });

  it("U6 retailer chip uses status + optional supplier on web, iOS, Android", () => {
    const web = src("apps/retailer-app-desktop/components/dashboard/CommandBoard.tsx");
    const ios = src("apps/retailer-app-ios/retailerapp/retailerapp/Screens/DashboardView.swift");
    const android = src("apps/retailer-app-android/app/src/main/java/com/pegasusx/retailer/ui/screens/dashboard/DashboardScreen.kt");
    expect(web).toMatch(/\/orders\?status=/);
    expect(web).toMatch(/supplier=/);
    expect(ios).toMatch(/onSelect:\s*\{\s*commandJump/);
    expect(ios).toMatch(/gs-u-retailer-stack/);
    expect(android).toMatch(/onOpenOrderStatus/);
    expect(android).toMatch(/onSelect = \{ onOpenOrderStatus/);
  });

  it("U7 field boards stay ritual (no 18-chip wall) and U8 admin stays web-only", () => {
    const driver = src("apps/driver-app-ios/driverappios/driverappios/Utilities/RemainingStops.swift");
    const payload = src("apps/payload-app-ios/payload-app-ios/Utilities/ManifestBoard.swift");
    const admin = src("apps/admin-portal/components/CommandBoard.tsx");
    expect(driver).toMatch(/ARRIVED_SHOP_CLOSED/);
    expect(driver).toMatch(/FISCAL_FAILED/);
    expect(payload).toMatch(/DRAFT/);
    expect(payload).toMatch(/DISPATCHED/);
    expect(admin).toMatch(/deadLetterHealth/);
    expect(admin).toMatch(/gs-u-admin-command/);
    expect(existsSync(join(apps, "admin-app-ios"))).toBe(false);
    expect(existsSync(join(apps, "admin-app-android"))).toBe(false);
    expect(existsSync(join(apps, "driver-portal"))).toBe(false);
  });

  it("StatusStack chips encode status as a label, not color-only, and stay 44/48 hit targets", () => {
    const ios = src("packages/mobile-ios-design/StatusStack.swift");
    const android = src("packages/mobile-android-design/src/main/java/com/pegasus/design/StatusStack.kt");
    const web = src("packages/ui-kit/src/portal/StatusStack.tsx");
    expect(ios).toMatch(/var onSelect/);
    expect(ios).toMatch(/minHeight: 44/);
    expect(ios).toMatch(/gs-u-chip-/);
    expect(android).toMatch(/onSelect: \(\(String\) -> Unit\)\? = null/);
    expect(android).toMatch(/heightIn\(min = 48\.dp\)/);
    expect(android).toMatch(/row\.key\.replace/);
    expect(web).toMatch(/onSelect\?: \(key: string\) => void/);
    expect(web).toMatch(/labelFor\(row\.key\)/);
  });

  it("does not add a 1s poll or a new motion dependency on command boards", () => {
    const supplier = src("apps/supplier-portal/app/(portal)/dashboard/page.tsx");
    const warehouse = src("apps/warehouse-portal/app/page.tsx");
    const factory = src("apps/factory-portal/app/page.tsx");
    expect(supplier).not.toMatch(/setInterval\(\s*\w+,\s*1000\s*\)/);
    expect(warehouse).not.toMatch(/setInterval\(\s*\w+,\s*1000\s*\)/);
    expect(factory).not.toMatch(/setInterval\(\s*\w+,\s*1000\s*\)/);
    expect(src("apps/supplier-app-ios/SupplierApp/Views/Dashboard/DashboardView.swift")).toMatch(/60_000_000_000/);
  });
});
