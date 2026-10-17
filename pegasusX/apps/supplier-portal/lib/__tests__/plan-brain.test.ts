import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import {
  brainForecastLine,
  factoryPlanningDisabledCode,
  isForecastBlocked,
  planBrainTabFromQuery,
} from "@pegasusx/types";

const here = dirname(fileURLToPath(import.meta.url));

describe("GS-U3 Plan & Brain", () => {
  it("deep-links tab=brain and defaults to planning", () => {
    expect(planBrainTabFromQuery("brain")).toBe("brain");
    expect(planBrainTabFromQuery("BRAIN")).toBe("brain");
    expect(planBrainTabFromQuery("planning")).toBe("planning");
    expect(planBrainTabFromQuery(null)).toBe("planning");
  });

  it("shows blocked_reason and does not invent a forecast line", () => {
    const blocked = { label: "insufficient_history", blocked_reason: "sparsity_blocked" };
    expect(isForecastBlocked(blocked)).toBe(true);
    expect(brainForecastLine(blocked, [10, 20, 30])).toBeNull();
    expect(brainForecastLine({ label: "standard" }, [1])).toBeNull();
    expect(brainForecastLine({ label: "standard" }, [1, 2])?.points).toEqual([1, 2]);
  });

  it("surfaces factory_planning_disabled on 409", () => {
    expect(factoryPlanningDisabledCode(409, { error: "factory_planning_disabled" })).toBe(
      "factory_planning_disabled",
    );
    expect(factoryPlanningDisabledCode(200, { error: "factory_planning_disabled" })).toBeNull();
    expect(factoryPlanningDisabledCode(409, { error: "other" })).toBeNull();
  });

  it("moves PlanningBrainPanel off analytics onto /planning tabs", () => {
    const analytics = readFileSync(join(here, "../../app/(portal)/analytics/page.tsx"), "utf8");
    const planning = readFileSync(join(here, "../../app/(portal)/planning/page.tsx"), "utf8");
    expect(analytics).not.toMatch(/PlanningBrainPanel/);
    expect(analytics).toMatch(/\/planning\?tab=brain/);
    expect(planning).toMatch(/PlanBrainTabs/);
    expect(planning).toMatch(/DigitalBrainPanel/);
    expect(planning).toMatch(/params\.get\("tab"\)/);
    expect(planning).toMatch(/q\.set\("tab"/);
    const factoryOps = readFileSync(
      join(here, "../../components/settings/planning/FactoryPlanningOpsPanel.tsx"),
      "utf8",
    );
    expect(factoryOps).toMatch(/factoryPlanningDisabledCode/);
    expect(factoryOps).toMatch(/gs-u-planning-push-status/);
  });
});
