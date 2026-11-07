import { describe, expect, it } from "vitest";
import { deadLetterHealth, deadLetterLabel } from "../deadLetterHealth";

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
