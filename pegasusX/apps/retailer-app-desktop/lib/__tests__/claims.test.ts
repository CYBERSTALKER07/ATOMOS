import { describe, expect, it } from "vitest";
import { claimTypeNeedsPhoto } from "../api";

describe("claimTypeNeedsPhoto", () => {
  it("requires photo for damage-like types", () => {
    expect(claimTypeNeedsPhoto("DAMAGED")).toBe(true);
    expect(claimTypeNeedsPhoto("concealed_damage")).toBe(true);
    expect(claimTypeNeedsPhoto("TAMPER")).toBe(true);
    expect(claimTypeNeedsPhoto("TEMPERATURE")).toBe(true);
  });

  it("does not require photo for missing/other", () => {
    expect(claimTypeNeedsPhoto("MISSING")).toBe(false);
    expect(claimTypeNeedsPhoto("OTHER")).toBe(false);
  });
});
