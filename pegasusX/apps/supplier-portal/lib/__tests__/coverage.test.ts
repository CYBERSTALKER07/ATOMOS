import { describe, expect, it } from "vitest";
import {
  catalogGatewayCodes,
  catalogOmits,
  coverageModeLabel,
  gatewayLabel,
  normalizeCoverageMode,
  pinKey,
} from "../coverage";

describe("GS-R coverage + pack PSP helpers", () => {
  it("labels engine modes", () => {
    expect(coverageModeLabel("PINNED")).toBe("Pinned");
    expect(coverageModeLabel("CITY_CELLS")).toBe("City cells");
    expect(normalizeCoverageMode("COUNTRY_CLOSEST")).toBe("COUNTRY_CLOSEST");
    expect(normalizeCoverageMode("")).toBe("COUNTRY_CLOSEST");
  });

  it("does not advertise Stripe on a UZ catalog", () => {
    const uz = [
      { code: "CASH", status: "live", selectable: true },
      { code: "GLOBAL_PAY", status: "live", selectable: true },
      { code: "PAYME", status: "unkeyed", selectable: true },
    ];
    expect(catalogOmits(uz, ["STRIPE", "ADYEN", "AIRWALLEX"])).toBe(true);
    expect(catalogGatewayCodes(uz)).toEqual(["CASH", "GLOBAL_PAY", "PAYME"]);
    expect(gatewayLabel("GLOBAL_PAY")).toBe("Global Pay");
  });

  it("dedupes pins by type+id", () => {
    expect(pinKey({ target_type: "location", target_id: " loc-s " })).toBe("LOCATION:loc-s");
  });
});
