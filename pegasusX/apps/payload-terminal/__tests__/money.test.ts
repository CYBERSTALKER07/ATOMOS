import { describe, it, expect } from "vitest";
import {
  parseMajorToMinor,
  parseMajorToMinorBigInt,
  parseMinorUnitsInt,
  majorIntToMinor,
  majorIntToMinorBigInt,
  minorToWholeTrunc,
  minorToWholeRound,
  applyDiscountBpsMinor,
  applyDiscountBpsBigInt,
  applyBpsFee,
  applyBpsFeeBigInt,
  formatMinorUnits,
  formatMinorFixed,
  formatBpsPercent,
  formatRatioPercentTenths,
  ratioPercentInt,
  formatGroupedInteger,
} from "../utils/money";

describe("Deterministic Money & Basis-Point Math (Invariant 4)", () => {
  describe("parseMajorToMinor / parseMajorToMinorBigInt", () => {
    it("parses IEEE-754 hazard decimals ('19.99', '0.29', '19,99') without float drift", () => {
      expect(parseMajorToMinor("19.99")).toBe(1999);
      expect(parseMajorToMinor("0.29")).toBe(29);
      expect(parseMajorToMinor("19,99")).toBe(1999);
      expect(parseMajorToMinor("0,29")).toBe(29);
      expect(parseMajorToMinor("0.05")).toBe(5);
      expect(parseMajorToMinor("0.5")).toBe(50);
      expect(parseMajorToMinor(".50")).toBe(50);
      expect(parseMajorToMinor("100")).toBe(10000);
      expect(parseMajorToMinor("0")).toBe(0);
      expect(parseMajorToMinor("0.00")).toBe(0);
      expect(parseMajorToMinor("19.990")).toBe(1999);
    });

    it("supports grouped major-unit amounts (comma, dot, and space grouping)", () => {
      expect(parseMajorToMinor("1,250,000.50")).toBe(125000050);
      expect(parseMajorToMinor("1.250.000,50")).toBe(125000050);
      expect(parseMajorToMinor("12 500 000.75")).toBe(1250000075);
      expect(parseMajorToMinor("1,250,000")).toBe(125000000);
    });

    it("parses large UZS amounts up to Number.MAX_SAFE_INTEGER and arbitrary BigInt beyond", () => {
      // 90,071,992,547,409.91 UZS == 9007199254740991 tiyin (Number.MAX_SAFE_INTEGER)
      expect(parseMajorToMinor("90071992547409.91")).toBe(Number.MAX_SAFE_INTEGER);
      expect(parseMajorToMinor("90071992547409.92")).toBeNull();
      expect(parseMajorToMinorBigInt("90071992547409.92")).toBe(BigInt("9007199254740992"));
      expect(parseMajorToMinorBigInt("999999999999999999.99")).toBe(
        BigInt("99999999999999999999"),
      );
    });

    it("rejects sub-minor precision, malformed strings, and disallowed negatives", () => {
      expect(parseMajorToMinor("19.995")).toBeNull();
      expect(parseMajorToMinor("19,995")).toBeNull();
      expect(parseMajorToMinor("0.001")).toBeNull();
      expect(parseMajorToMinor("")).toBeNull();
      expect(parseMajorToMinor("   ")).toBeNull();
      expect(parseMajorToMinor(".")).toBeNull();
      expect(parseMajorToMinor(",")).toBeNull();
      expect(parseMajorToMinor("abc")).toBeNull();
      expect(parseMajorToMinor("12.3.4")).toBeNull();
      expect(parseMajorToMinor("-19.99")).toBeNull();
      expect(parseMajorToMinor("-19.99", { allowNegative: true })).toBe(-1999);
      expect(Object.is(parseMajorToMinor("-0.00", { allowNegative: true }), 0)).toBe(true);
    });
  });

  describe("parseMinorUnitsInt & major/minor conversions", () => {
    it("parses integer minor units and rejects fractional strings", () => {
      expect(parseMinorUnitsInt("4500000")).toBe(4500000);
      expect(parseMinorUnitsInt("4,500,000")).toBe(4500000);
      expect(parseMinorUnitsInt("4 500 000")).toBe(4500000);
      expect(parseMinorUnitsInt(4500000)).toBe(4500000);
      expect(parseMinorUnitsInt("19.99")).toBeNull();
      expect(parseMinorUnitsInt("-500")).toBeNull();
      expect(parseMinorUnitsInt("-500", { allowNegative: true })).toBe(-500);
    });

    it("converts between major and minor units without negative-floor bugs", () => {
      expect(majorIntToMinor(25000000)).toBe(2500000000);
      expect(majorIntToMinorBigInt("500000000000")).toBe(BigInt("50000000000000"));
      // Truncation toward zero on negative variance (-50 tiyin -> 0 whole UZS, not -1)
      expect(minorToWholeTrunc(-50)).toBe(0);
      expect(minorToWholeTrunc(-150)).toBe(-1);
      expect(minorToWholeRound(2550)).toBe(26);
      expect(minorToWholeRound(2549)).toBe(25);
    });
  });

  describe("applyDiscountBpsMinor / applyDiscountBpsBigInt & applyBpsFee", () => {
    it("computes exact basis-point discounts with half-up rounding and boundary clamping", () => {
      expect(applyDiscountBpsMinor(10000, 1500)).toBe(8500);
      expect(applyDiscountBpsMinor(1999, 1500)).toBe(1699);
      expect(applyDiscountBpsMinor(1999, 333)).toBe(1932);
      expect(applyDiscountBpsMinor(29, 500)).toBe(28);
      // Tie-break at 0.5 minor unit rounds half-up
      expect(applyDiscountBpsMinor(100, 50)).toBe(100);
      expect(applyDiscountBpsMinor(100, 150)).toBe(99);
      // Clamping [0, 10000]
      expect(applyDiscountBpsMinor(1999, 0)).toBe(1999);
      expect(applyDiscountBpsMinor(1999, -250)).toBe(1999);
      expect(applyDiscountBpsMinor(1999, 10000)).toBe(0);
      expect(applyDiscountBpsMinor(1999, 12500)).toBe(0);
    });

    it("handles large UZS amounts without 53-bit intermediate overflow", () => {
      // 50,000,000,000,000 tiyin (500B UZS) * 8500 would overflow 53-bit Number without BigInt
      const largeMinor = 50_000_000_000_000;
      expect(applyDiscountBpsMinor(largeMinor, 1500)).toBe(42_500_000_000_000);
      expect(applyDiscountBpsBigInt(BigInt("900000000000000000"), 2500)).toBe(
        BigInt("675000000000000000"),
      );
    });

    it("computes basis-point fees, VAT, and reserves deterministically", () => {
      expect(applyBpsFee(482000000, 285)).toBe(13737000);
      expect(applyBpsFee(13737000, 1200)).toBe(1648440);
      expect(applyBpsFee(482000000, 250)).toBe(12050000);
      expect(applyBpsFeeBigInt(BigInt("1000000000000"), 500)).toBe(BigInt("50000000000"));
    });
  });

  describe("formatMinorUnits / formatMinorFixed / formatBpsPercent / ratios", () => {
    it("formats minor units, bps, and credit utilization ratios without float division", () => {
      expect(formatMinorFixed(1999)).toBe("19.99");
      expect(formatMinorFixed(29)).toBe("0.29");
      expect(formatMinorFixed(-50)).toBe("-0.50");
      expect(formatMinorFixed(12500, { signDisplay: "always", currency: "UZS" })).toBe(
        "+125.00 UZS",
      );
      expect(
        formatMinorUnits(1250050, {
          groupSeparator: " ",
          minimumFractionDigits: 0,
          maximumFractionDigits: 2,
          currency: "UZS",
        }),
      ).toBe("12 500.5 UZS");
      expect(
        formatMinorUnits(1250000, {
          groupSeparator: ",",
          minimumFractionDigits: 0,
          maximumFractionDigits: 0,
          currency: "UZS",
        }),
      ).toBe("12,500 UZS");
      expect(formatBpsPercent(1500, 2)).toBe("15.00");
      expect(formatBpsPercent(285, 2)).toBe("2.85");
      expect(formatBpsPercent(7550, 1)).toBe("75.5");
      expect(formatRatioPercentTenths(250000000, 1000000000)).toBe("25.0");
      expect(formatRatioPercentTenths(1, 3)).toBe("33.3");
      expect(formatRatioPercentTenths(2, 3)).toBe("66.7");
      expect(ratioPercentInt(2000000000, 2500000000)).toBe(80);
      expect(ratioPercentInt(3000000000, 2500000000)).toBe(100);
      expect(formatGroupedInteger(25000000)).toBe("25,000,000");
    });
  });
});
