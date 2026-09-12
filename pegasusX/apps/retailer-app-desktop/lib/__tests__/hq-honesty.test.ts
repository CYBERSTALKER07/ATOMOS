import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const here = dirname(fileURLToPath(import.meta.url));
const pageSource = readFileSync(join(here, "../../app/(dashboard)/hq/page.tsx"), "utf8");

describe("retailer HQ honesty", () => {
  it("does not treat HQ HTTP failure as zero tiles", () => {
    expect(pageSource).toMatch(/\/v1\/retailer\/hq\/summary/);
    expect(pageSource).toMatch(/hq_failed/);
    expect(pageSource).toMatch(/hqError/);
    expect(pageSource).toMatch(/summary \? \(/);
    expect(pageSource).not.toMatch(/if \(sRes\.ok\) \{\s*setSummary/);
  });
});
