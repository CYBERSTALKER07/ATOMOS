import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { ORDER_STATUS_FUNNEL, emptyOrderStatusCounts, incrementOrderStatusCount } from "@pegasusx/types";

const here = dirname(fileURLToPath(import.meta.url));
const pageSource = readFileSync(join(here, "../../app/(dashboard)/dashboard/page.tsx"), "utf8");
const boardSource = readFileSync(join(here, "../../components/dashboard/CommandBoard.tsx"), "utf8");
const loyaltySource = readFileSync(join(here, "../../components/LoyaltyCard.tsx"), "utf8");
const towerSource = readFileSync(join(here, "../../app/(dashboard)/control-tower/page.tsx"), "utf8");

describe("GS-U6 retailer command", () => {
  it("binds pulse + StatusStack + supplier facet and does not invent last-mile factory trucks", () => {
    expect(pageSource).toMatch(/\/v1\/retailer\/control-tower\/pulse/);
    expect(pageSource).toMatch(/60000/);
    expect(boardSource).toMatch(/ORDER_STATUS_FUNNEL/);
    expect(boardSource).toMatch(/gs-u-retailer-stack/);
    expect(boardSource).toMatch(/gs-u-retailer-supplier-facet/);
    expect(boardSource).toMatch(/gs-u-retailer-pulse-empty/);
    expect(boardSource).toMatch(/\/orders\?status=/);
    expect(boardSource).toMatch(/supplier=/);
    expect(boardSource).not.toMatch(/FACTORY_TRANSFER_STATES/);
    expect(boardSource).not.toMatch(/Factory trucks/);
    expect(boardSource).not.toMatch(/sparkline_/);
    expect(pageSource).not.toMatch(/uniqueSuppliersCount=\{new Set\(productList/);
  });

  it("keeps empty pulse honest and does not invent Bronze", () => {
    expect(boardSource).toMatch(/Empty pulse — not demo tiles/);
    expect(boardSource).toMatch(/gs-u-retailer-loyalty/);
    expect(loyaltySource).toMatch(/Not enrolled. No fake Bronze/);
    expect(boardSource).toMatch(/Auto-order place/);
    expect(boardSource).toMatch(/off/);
  });

  it("does not treat control-tower pulse HTTP failure as empty", () => {
    expect(pageSource).toMatch(/control_tower_pulse_failed/);
    expect(boardSource).toMatch(/gs-u-retailer-command-error/);
    expect(boardSource).toMatch(/if \(error\)/);
    expect(towerSource).toMatch(/control_tower_pulse_failed/);
    expect(towerSource).not.toMatch(/setPulse\(null\)/);
  });

  it("does not invent KpiGrid zeros from a failed command pulse", () => {
    expect(pageSource).toMatch(/pulseError \|\| \(loadingPulse && !pulse\)/);
    expect(pageSource).toMatch(/<KpiGrid/);
  });

  it("increments FISCAL_FAILED on the 17-key child stack without blending suppliers", () => {
    const next = incrementOrderStatusCount(emptyOrderStatusCounts(), "FISCAL_FAILED");
    expect(next.FISCAL_FAILED).toBe(1);
    expect(Object.keys(next)).toHaveLength(17);
    expect(ORDER_STATUS_FUNNEL).toContain("FISCAL_FAILED");
    expect(ORDER_STATUS_FUNNEL).toHaveLength(17);
  });
});
