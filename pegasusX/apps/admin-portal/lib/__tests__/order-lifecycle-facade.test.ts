import { describe, expect, it } from "vitest";
import * as fs from "fs";
import * as path from "path";
import {
  toOrderCustomerStage,
  getCustomerStageProgressPercent,
  type OrderCustomerStage,
} from "@pegasusx/types";

describe("Customer Stage Facade - Golden Parity", () => {
  const fixturePath = path.resolve(__dirname, "../../../../contracts/order-customer-stage.json");
  const fixture = JSON.parse(fs.readFileSync(fixturePath, "utf-8"));

  it("matches all status mappings in contracts/order-customer-stage.json", () => {
    for (const [status, expectedStage] of Object.entries(fixture.mapping)) {
      const actualStage = toOrderCustomerStage(status);
      expect(actualStage).toBe(expectedStage);
    }
  });

  it("matches progress percent for all stages in fixture", () => {
    for (const [stage, expectedPercent] of Object.entries(fixture.progress_percent)) {
      const actualPercent = getCustomerStageProgressPercent(stage as OrderCustomerStage);
      expect(actualPercent).toBe(expectedPercent);
    }
  });

  it("handles whitespace and case insensitivity", () => {
    expect(toOrderCustomerStage("  arriving ")).toBe("OUT_FOR_DELIVERY");
    expect(toOrderCustomerStage("scheduled")).toBe("PLACED");
  });

  it("falls back to PROCESSING for unknown status", () => {
    expect(toOrderCustomerStage("UNKNOWN_CUSTOM_STATUS")).toBe(fixture.default);
  });
});
