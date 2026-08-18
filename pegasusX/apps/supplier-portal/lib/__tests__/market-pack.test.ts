import { describe, expect, it } from "vitest";
import { displayPackCurrency, fiscalReceiptLabel, formatPackMoney, mapInitialViewState, moneyCurrency, packAllowsPsp, packCurrency, packMapCenter, pinApiBaseUrl, selectablePackPsps } from "@pegasusx/api-client";
import { canonicalizeOrderStatus, emptyOrderStatusCounts, ORDER_STATUS_FUNNEL } from "@pegasusx/types";

describe("GS-R market pack bind", () => {
  it("labels Soliq vs commercial and does not invent UZS", () => {
    expect(fiscalReceiptLabel("MY_SOLIQ")).toBe("Soliq");
    expect(fiscalReceiptLabel("COMMERCIAL")).toBe("commercial");
    expect(packCurrency(null)).toBe("");
    expect(packCurrency({ currency_code: "eur" })).toBe("EUR");
    expect(formatPackMoney(150000, { currency_code: "UZS", currency_decimal_places: 2 })).toBe("1 500 UZS");
    expect(displayPackCurrency("", "")).toBe("");
    expect(displayPackCurrency("kzt", "")).toBe("KZT");
    expect(moneyCurrency("")).toBe("");
  });

  it("does not advertise Stripe on a UZ pack", () => {
    expect(packAllowsPsp({ code: "UZ", name: "Uzbekistan", status: "shipped", home_cell: "cell-uz", timezone: "Asia/Tashkent", currency_code: "UZS", fiscal_adapter: "MY_SOLIQ", psp_adapters: ["GLOBAL_PAY", "CASH"] }, "STRIPE")).toBe(false);
    expect(packAllowsPsp({ code: "UZ", name: "Uzbekistan", status: "shipped", home_cell: "cell-uz", timezone: "Asia/Tashkent", currency_code: "UZS", fiscal_adapter: "MY_SOLIQ", psp_adapters: ["GLOBAL_PAY", "CASH"] }, "GLOBAL_PAY")).toBe(true);
    expect(selectablePackPsps([{ code: "STRIPE", selectable: false }, { code: "CASH", selectable: true }])).toEqual(["CASH"]);
  });

  it("does not invent Tashkent for empty or planned packs", () => {
    expect(packMapCenter(null)).toBeNull();
    expect(packMapCenter({ status: "planned", map_center_lat: 41.2995, map_center_lng: 69.2401 })).toBeNull();
    expect(packMapCenter({ status: "shipped", map_center_lat: 0, map_center_lng: 0 })).toBeNull();
    expect(packMapCenter({ status: "shipped", map_center_lat: 41.2995, map_center_lng: 69.2401 })).toEqual({
      lat: 41.2995,
      lng: 69.2401,
    });
    expect(mapInitialViewState(null)).toEqual({ latitude: 0, longitude: 0, zoom: 1 });
  });

  it("exposes the full order-status funnel and maps aliases", () => {
    const counts = emptyOrderStatusCounts();
    expect(ORDER_STATUS_FUNNEL).toHaveLength(17);
    expect(Object.keys(counts)).toHaveLength(17);
    expect(counts.COMPLETED).toBe(0);
    expect(canonicalizeOrderStatus("en_route")).toBe("IN_TRANSIT");
    expect(canonicalizeOrderStatus("DISPATCHED")).toBe("LOADED");
    expect(canonicalizeOrderStatus("SHOP_CLOSED_PENDING")).toBe("ARRIVED_SHOP_CLOSED");
  });

  it("keeps localhost bootstrap and pins cell-eu off-box", () => {
    expect(pinApiBaseUrl({ bootstrap: "http://localhost:8180", homeCell: "cell-eu" })).toBe("http://localhost:8180");
    expect(pinApiBaseUrl({ bootstrap: "https://api.pegasusx.app", homeCell: "cell-eu" })).toBe("https://api-eu.pegasusx.app");
  });
});
