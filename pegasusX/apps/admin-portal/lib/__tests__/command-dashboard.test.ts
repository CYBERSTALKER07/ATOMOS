import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { deadLetterHealth, deadLetterLabel } from "../deadLetterHealth";

const here = dirname(fileURLToPath(import.meta.url));
const board = readFileSync(join(here, "../../components/CommandBoard.tsx"), "utf8");
const ops = readFileSync(join(here, "../../components/OpsPanel.tsx"), "utf8");
const page = readFileSync(join(here, "../../app/page.tsx"), "utf8");
const accuracy = readFileSync(join(here, "../../components/AccuracyPanel.tsx"), "utf8");
const apiSrc = readFileSync(join(here, "../api.ts"), "utf8");

describe("GS-U8 dead-letter health", () => {
  it("does not treat missing count as zero", () => {
    expect(deadLetterHealth({})).toEqual({ kind: "unavailable" });
    expect(deadLetterHealth({ dead_letter_count: 0 })).toEqual({ kind: "unavailable" });
    expect(deadLetterHealth({ dead_letter_available: false, dead_letter_count: 0 })).toEqual({
      kind: "unavailable",
    });
    expect(deadLetterLabel(deadLetterHealth({}))).toBe("unavailable");
  });

  it("zero is empty when COUNT(*) is available", () => {
    expect(deadLetterHealth({ dead_letter_available: true, dead_letter_count: 0 })).toEqual({ kind: "zero" });
    expect(deadLetterLabel(deadLetterHealth({ dead_letter_available: true, dead_letter_count: 0 }))).toBe("empty");
  });

  it("uses dead_letter_count not page length", () => {
    const h = deadLetterHealth({
      dead_letter_available: true,
      dead_letter_count: 7,
      page_count: 2,
      items: [{}, {}],
    });
    expect(h).toEqual({ kind: "count", count: 7 });
  });
});

describe("GS-U8 admin command bind", () => {
  it("binds COUNT(*) dead letters and does not invent mape28 or UZS", () => {
    expect(page).toMatch(/CommandBoard/);
    expect(page).toMatch(/AccuracyPanel/);
    expect(board).toMatch(/gs-u-admin-health/);
    expect(board).toMatch(/deadLetterHealth/);
    expect(board).toMatch(/listPendingFlags/);
    expect(board).toMatch(/outboxSummary/);
    expect(board).toMatch(/IsRegistered is tenant-register/);
    expect(board).not.toMatch(/sparkline/);
    expect(board).not.toMatch(/UZS/);
    expect(ops).toMatch(/COUNT\(\*\)/);
    expect(ops).toMatch(/deadLetterHealth/);
    expect(ops).not.toMatch(/None or table unavailable/);
    expect(apiSrc).toMatch(/\/v1\/admin\/planning\/accuracy/);
    expect(accuracy).toMatch(/listPlanningAccuracy/);
    expect(accuracy).toMatch(/no invented mape28 line/);
    expect(accuracy).not.toMatch(/recharts/i);
  });
});
