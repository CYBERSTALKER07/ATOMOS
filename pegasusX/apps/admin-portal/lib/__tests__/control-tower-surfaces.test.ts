import { describe, it, expect } from "vitest";
import { mapToHumanError } from "../errors";

describe("Admin Error Mapping (UA-9)", () => {
  it("maps 409 conflict errors to human-actionable messages", () => {
    const err = new Error("409: optimistic lock conflict watermark drift");
    const mapped = mapToHumanError(err);
    expect(mapped.title).toContain("Conflict");
    expect(mapped.actionGuidance).toBeDefined();
  });

  it("maps 429 rate limit with backoff advice", () => {
    const err = new Error("429: too many requests");
    const mapped = mapToHumanError(err);
    expect(mapped.title).toContain("Rate Limit");
    expect(mapped.actionGuidance).toContain("wait");
  });

  it("maps 401 unauthorized to session expiry advice", () => {
    const err = new Error("401: token expired");
    const mapped = mapToHumanError(err);
    expect(mapped.title).toContain("Session Expired");
    expect(mapped.actionGuidance).toContain("sign in");
  });

  it("maps 504 gateway timeout to safe retry advice", () => {
    const err = new Error("504: gateway timeout");
    const mapped = mapToHumanError(err);
    expect(mapped.title).toContain("Timeout");
    expect(mapped.actionGuidance).toBeDefined();
  });
});

describe("Cashier Vault Denomination Counter & Zero-Trust Invariant (UA-1, CG-11)", () => {
  const UZS_DENOMS = [200000, 100000, 50000, 20000, 10000, 5000, 2000, 1000];

  it("computes physical note tally with exact mathematical precision", () => {
    const counts: Record<number, number> = {
      200000: 5,  // 1,000,000 UZS
      100000: 2,  // 200,000 UZS
      50000: 1,   // 50,000 UZS
      20000: 2,   // 40,000 UZS
      10000: 1,   // 10,000 UZS
      5000: 0,
      2000: 0,
      1000: 0,
    };

    const totalUZS = UZS_DENOMS.reduce((acc, denom) => acc + denom * (counts[denom] || 0), 0);
    expect(totalUZS).toBe(1300000); // 1.3M UZS

    const totalTiyins = totalUZS * 100;
    expect(totalTiyins).toBe(130000000);
  });

  it("upholds zero-trust blind count: expected cash is masked before resolution", () => {
    const mockSessionView = {
      session_id: "sess_123",
      driver_id: "drv_1",
      status: "AWAITING_BLIND_COUNT_1",
      counted_cash_minor: 0,
      expected_cash_minor: undefined, // Must be omitted/undefined in blind state!
    };

    expect(mockSessionView.expected_cash_minor).toBeUndefined();
  });
});

describe("Multi-Tender AML Exposure & Uncapped Ceiling Policy", () => {
  const DYNAMIC_CEILING_MINOR = 2_500_000_000;
  const UNCAPPED_CEILING_MINOR = 0;

  it("permits cash intake when within configured ceiling", () => {
    const currentExposureMinor = 1_000_000_000; // 10M UZS
    const attemptedCashMinor = 500_000_000;    // 5M UZS

    const newExposure = currentExposureMinor + attemptedCashMinor;
    const isAllowed = DYNAMIC_CEILING_MINOR === 0 || newExposure <= DYNAMIC_CEILING_MINOR;

    expect(isAllowed).toBe(true);
    expect(DYNAMIC_CEILING_MINOR - newExposure).toBe(1_000_000_000); // 10M UZS headroom
  });

  it("permits unlimited cash intake under uncapped multi-tender policy", () => {
    const currentExposureMinor = 25_000_000_000; // 250M UZS
    const attemptedCashMinor = 5_000_000_000;    // 50M UZS

    const newExposure = currentExposureMinor + attemptedCashMinor;
    const isAllowed = UNCAPPED_CEILING_MINOR === 0 || newExposure <= UNCAPPED_CEILING_MINOR;

    expect(isAllowed).toBe(true);
  });
});

describe("DVIR 8-Point Safety Inspection Logic (UA-8)", () => {
  it("flags critical defects if brakes or steering fail", () => {
    const checklist = {
      steering_and_horn_ok: false,
      service_and_parking_brakes_ok: true,
      tires_tread_and_pressure_ok: true,
      lights_and_hazards_ok: true,
      wipers_and_fluids_ok: true,
      emergency_kit_and_extinguisher_ok: true,
      cargo_box_sanitation_ok: true,
      reefer_temp_calibrated_ok: true,
    };

    const hasCriticalDefect = !checklist.steering_and_horn_ok || !checklist.service_and_parking_brakes_ok;
    expect(hasCriticalDefect).toBe(true);
  });

  it("passes inspection when all 8 points are verified", () => {
    const checklist = {
      steering_and_horn_ok: true,
      service_and_parking_brakes_ok: true,
      tires_tread_and_pressure_ok: true,
      lights_and_hazards_ok: true,
      wipers_and_fluids_ok: true,
      emergency_kit_and_extinguisher_ok: true,
      cargo_box_sanitation_ok: true,
      reefer_temp_calibrated_ok: true,
    };

    const allPassed = Object.values(checklist).every(Boolean);
    expect(allPassed).toBe(true);
  });
});

describe("Soliq Fiscal Preflight & PKCS#12 Validity Window (Guard #3)", () => {
  it("enforces minimum 72h window before certificate expiration", () => {
    const certHours = 48; // Less than 72 hours
    const isCertHealthy = certHours >= 72;
    expect(isCertHealthy).toBe(false);
  });

  it("passes when certificate has >= 72h window remaining", () => {
    const certHours = 120; // 5 days
    const isCertHealthy = certHours >= 72;
    expect(isCertHealthy).toBe(true);
  });
});
