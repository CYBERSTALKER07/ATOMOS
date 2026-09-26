import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const here = dirname(fileURLToPath(import.meta.url));
const pageSource = readFileSync(join(here, "../../app/(dashboard)/reports/page.tsx"), "utf8");

describe("retailer reports honesty", () => {
  it("does not treat summary HTTP failure as zero tiles", () => {
    expect(pageSource).toMatch(/\/v1\/retailer\/reports\/summary/);
    expect(pageSource).toMatch(/reports_failed/);
    expect(pageSource).toMatch(/summaryError/);
    expect(pageSource).toMatch(/salesError/);
    expect(pageSource).not.toMatch(/if \(sRes\.ok\) setSummary/);
  });
});
