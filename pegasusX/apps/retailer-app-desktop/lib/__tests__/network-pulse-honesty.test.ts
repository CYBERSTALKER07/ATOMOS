import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const here = dirname(fileURLToPath(import.meta.url));
const panel = readFileSync(join(here, "../../components/NetworkPulsePanel.tsx"), "utf8");

describe("retailer network pulse honesty", () => {
  it("does not treat a failed GET as an empty timeline", () => {
    expect(panel).toMatch(/error=\{error\}/);
    expect(panel).toMatch(/if \(!res\.ok\) throw new Error\("pulse_failed"\)/);
    expect(panel).toMatch(/setError\("pulse_failed"\)/);
    expect(panel).not.toMatch(/setEvents\(\[\]\)/);
  });
});
