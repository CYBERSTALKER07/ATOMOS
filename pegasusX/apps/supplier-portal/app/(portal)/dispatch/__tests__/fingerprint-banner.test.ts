import { describe, expect, it } from "vitest";

/** Mirrors supplier dispatch preview drift gate (PX-DESK-4D). */
function shouldShowDispatchFingerprintBanner(preview: {
  plan_fingerprint_mismatch?: boolean;
} | null): boolean {
  return Boolean(preview?.plan_fingerprint_mismatch);
}

describe("dispatch fingerprint banner", () => {
  it("shows when supplier and warehouse fingerprints diverge", () => {
    expect(
      shouldShowDispatchFingerprintBanner({ plan_fingerprint_mismatch: true }),
    ).toBe(true);
  });

  it("hides when fingerprints match", () => {
    expect(
      shouldShowDispatchFingerprintBanner({ plan_fingerprint_mismatch: false }),
    ).toBe(false);
    expect(shouldShowDispatchFingerprintBanner(null)).toBe(false);
  });
});
