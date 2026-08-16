import { describe, expect, it } from "vitest";
import { fiscalReceiptLabel, formatPackMoney, packAllowsPsp, packCurrency, pinApiBaseUrl } from "@pegasusx/api-client";

describe("GS-R market pack bind", () => {
  it("labels Soliq vs commercial and does not invent UZS", () => {
    expect(fiscalReceiptLabel("MY_SOLIQ")).toBe("Soliq");
    expect(fiscalReceiptLabel("COMMERCIAL")).toBe("commercial");
    expect(packCurrency(null)).toBe("");
    expect(packCurrency({ currency_code: "eur" })).toBe("EUR");
    expect(formatPackMoney(150000, { currency_code: "UZS", currency_decimal_places: 2 })).toBe("1 500 UZS");
  });

  it("does not advertise Stripe on a UZ pack", () => {
    expect(packAllowsPsp({ code: "UZ", name: "Uzbekistan", status: "shipped", home_cell: "cell-uz", timezone: "Asia/Tashkent", currency_code: "UZS", fiscal_adapter: "MY_SOLIQ", psp_adapters: ["GLOBAL_PAY", "CASH"] }, "STRIPE")).toBe(false);
    expect(packAllowsPsp({ code: "UZ", name: "Uzbekistan", status: "shipped", home_cell: "cell-uz", timezone: "Asia/Tashkent", currency_code: "UZS", fiscal_adapter: "MY_SOLIQ", psp_adapters: ["GLOBAL_PAY", "CASH"] }, "GLOBAL_PAY")).toBe(true);
  });

  it("keeps localhost bootstrap and pins cell-eu off-box", () => {
    expect(pinApiBaseUrl({ bootstrap: "http://localhost:8180", homeCell: "cell-eu" })).toBe("http://localhost:8180");
    expect(pinApiBaseUrl({ bootstrap: "https://api.pegasusx.app", homeCell: "cell-eu" })).toBe("https://api-eu.pegasusx.app");
  });
});
