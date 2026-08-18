import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const here = dirname(fileURLToPath(import.meta.url));
const pageSource = readFileSync(join(here, "../../app/(dashboard)/sections/page.tsx"), "utf8");

describe("retailer sections honesty", () => {
  it("does not treat unassigned-skus HTTP failure as none/no stock", () => {
    expect(pageSource).toMatch(/\/v1\/retailer\/sections\/unassigned-skus/);
    expect(pageSource).toMatch(/unassigned_skus_failed/);
    expect(pageSource).toMatch(/unassignedError/);
    expect(pageSource).not.toMatch(/\/\* ignore \*\//);
  });

  it("does not treat section detail HTTP failure as empty SKUs/none", () => {
    expect(pageSource).toMatch(/\/v1\/retailer\/sections\/\$\{id\}/);
    expect(pageSource).toMatch(/section_detail_failed/);
    expect(pageSource).toMatch(/detailError/);
    expect(pageSource).not.toMatch(/if \(!res\.ok\) return;/);
  });

  it("does not treat section SKU/staff PUT failure as saved", () => {
    expect(pageSource).toMatch(/section_skus_failed/);
    expect(pageSource).toMatch(/section_staff_failed/);
    expect(pageSource).toMatch(/saveError/);
    expect(pageSource).not.toMatch(/sku_map_failed/);
    expect(pageSource).not.toMatch(/staff_assign_failed/);
  });
});
