import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const here = dirname(fileURLToPath(import.meta.url));
const pageSource = readFileSync(
  join(here, "../../app/(dashboard)/stock/local-skus/page.tsx"),
  "utf8",
);

describe("retailer local SKUs honesty", () => {
  it("does not treat local-skus HTTP failure as an empty catalog", () => {
    expect(pageSource).toMatch(/\/v1\/retailer\/local-skus/);
    expect(pageSource).toMatch(/local_skus_failed/);
    expect(pageSource).toMatch(/error === "local_skus_failed" \? null/);
    expect(pageSource).not.toMatch(/setItems\(\[\]\)/);
  });
});
