import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";

const here = dirname(fileURLToPath(import.meta.url));
const panel = readFileSync(join(here, "../../components/NetworkPulsePanel.tsx"), "utf8");
const handoff = readFileSync(join(here, "../../components/HandoffTimelinePanel.tsx"), "utf8");

describe("warehouse pulse honesty", () => {
  it("does not treat a failed GET as an empty timeline", () => {
    for (const source of [panel, handoff]) {
      expect(source).toMatch(/error=\{error\}/);
      expect(source).toMatch(/setError\("pulse_failed"\)|setError\('pulse_failed'\)/);
      expect(source).not.toMatch(/setEvents\(\[\]\)/);
    }
  });
});
