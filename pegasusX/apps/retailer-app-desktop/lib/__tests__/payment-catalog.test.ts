import { describe, expect, it } from "vitest";
import {
  catalogOmits,
  checkoutGatewayForMethod,
  displayPackCurrency,
  filterRetailerCardGateways,
  moneyCurrency,
  retailerCatalogGateways,
} from "../payment-catalog";

const uzCatalog = [
  { code: "CASH", status: "live", selectable: true },
  { code: "GLOBAL_PAY", status: "live", selectable: true },
  { code: "PAYME", status: "unkeyed", selectable: true },
];

describe("GS-R retailer payment catalog", () => {
  it("omits Stripe/Adyen on a UZ catalog", () => {
    expect(catalogOmits(uzCatalog, ["STRIPE", "ADYEN", "AIRWALLEX"])).toBe(true);
    expect(retailerCatalogGateways(uzCatalog)).toEqual(["CASH", "GLOBAL_PAY", "PAYME"]);
  });

  it("does not invent Adyen when the payment-required list is empty", () => {
    expect(filterRetailerCardGateways([], ["CASH", "GLOBAL_PAY"])).toEqual(["GLOBAL_PAY"]);
    expect(filterRetailerCardGateways(["ADYEN", "GLOBAL_PAY"], ["CASH", "GLOBAL_PAY"])).toEqual([
      "GLOBAL_PAY",
    ]);
    expect(filterRetailerCardGateways(["GLOBAL_PAY", "ADYEN"], [])).toEqual(["GLOBAL_PAY"]);
  });

  it("maps checkout methods only onto pack rails", () => {
    expect(checkoutGatewayForMethod("global_pay", ["CASH", "GLOBAL_PAY"])).toBe("GLOBAL_PAY");
    expect(checkoutGatewayForMethod("adyen", ["CASH", "GLOBAL_PAY"])).toBe("CASH");
    expect(checkoutGatewayForMethod("cash", ["CASH"])).toBe("CASH");
  });

  it("does not invent UZS when pack and event currency are empty", () => {
    expect(displayPackCurrency("", "")).toBe("");
    expect(displayPackCurrency("uzs", "")).toBe("UZS");
    expect(displayPackCurrency("", "KZT")).toBe("KZT");
    expect(moneyCurrency("")).toBe("");
  });
});
