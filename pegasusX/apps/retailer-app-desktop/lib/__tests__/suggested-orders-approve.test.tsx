import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { createTranslator } from "@pegasusx/i18n";

const { apiFetchMock } = vi.hoisted(() => ({
  apiFetchMock: vi.fn(),
}));

vi.mock("@/lib/auth", () => ({
  apiFetch: apiFetchMock,
}));

vi.mock("@/lib/retailer-profile", () => ({
  getRetailerId: () => "ret-1",
}));

vi.mock("@/lib/i18n", () => ({
  usePortalLocale: () => "en",
  usePortalT: () => createTranslator("en"),
}));

import {
  approveReplenishmentSuggestion,
  getRetailerPlanningForecast,
  getRetailerReorderSuggestions,
} from "@/lib/api";
import { SuggestedOrdersApprovePanel } from "@/components/dashboard/SuggestedOrdersApprovePanel";

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("Retailer Desktop Forecast & Suggested Order One-Tap Approve", () => {
  beforeEach(() => {
    cleanup();
    apiFetchMock.mockReset();
  });

  it("calls POST /v1/replenishment/suggestions/{id}/approve with Idempotency-Key", async () => {
    const idem = "11111111-2222-4333-8444-555555555555";
    apiFetchMock.mockResolvedValueOnce(
      jsonResponse({
        suggestion_id: "ret-1:SKU-101",
        retailer_id: "ret-1",
        sku: "SKU-101",
        status: "CONVERTED",
        approved_qty: 24,
        order_id: "ord-auto-1",
      }),
    );

    const res = await approveReplenishmentSuggestion(
      "ret-1:SKU-101",
      { retailer_id: "ret-1", sku: "SKU-101", qty: 24, mode: "order" },
      idem,
    );
    expect(res.status).toBe("CONVERTED");
    expect(res.order_id).toBe("ord-auto-1");
    expect(apiFetchMock).toHaveBeenCalledWith(
      "/v1/replenishment/suggestions/ret-1%3ASKU-101/approve",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({ "Idempotency-Key": idem }),
      }),
    );

    apiFetchMock.mockResolvedValueOnce(
      jsonResponse({ items: [{ suggestion_id: "ret-1:SKU-101", sku: "SKU-101", suggested_qty: 12 }] }),
    );
    const items = await getRetailerReorderSuggestions("OPEN");
    expect(items).toHaveLength(1);

    apiFetchMock.mockResolvedValueOnce(
      jsonResponse({ run: { run_id: "run-1", winner_model: "CALENDAR_GLOBAL_V2", champion_wape_bps: 1800, challenger_wape_bps: 1100, horizon_days: 14 }, forecasts: [] }),
    );
    const fc = await getRetailerPlanningForecast({ horizon_days: 14 });
    expect(fc.run?.winner_model).toBe("CALENDAR_GLOBAL_V2");
  });

  it("renders SuggestedOrdersApprovePanel with forecast summary and approves suggestion in one tap", async () => {
    const onApproved = vi.fn();
    apiFetchMock.mockImplementation(async (url: string, init?: RequestInit) => {
      if (url.startsWith("/v1/retailer/reorder-suggestions")) {
        return jsonResponse({
          items: [
            {
              suggestion_id: "ret-1:SKU-MILK",
              retailer_id: "ret-1",
              sku: "SKU-MILK",
              sku_name: "Fresh Milk 1L",
              suggested_qty: 18,
              current_stock: 4,
              safety_stock: 10,
              sources: ["STORE_POS", "WHOLESALE_HISTORY"],
              status: "OPEN",
            },
          ],
        });
      }
      if (url.startsWith("/v1/planning/forecast")) {
        return jsonResponse({
          run: {
            run_id: "run-retail-1",
            winner_model: "CALENDAR_GLOBAL_V2",
            champion_model: "SES",
            challenger_model: "CALENDAR_GLOBAL_V2",
            champion_wape_bps: 1900,
            challenger_wape_bps: 1150,
            horizon_days: 14,
            regressors: [{ name: "Payday", category: "CALENDAR", multiplier_bps: 900, description: "Payday lift" }],
          },
          forecasts: [
            {
              forecast_date: "2026-11-18",
              sku: "SKU-MILK",
              model: "CALENDAR_GLOBAL_V2",
              is_winner: true,
              p10_milli: 12000,
              p50_milli: 18000,
              p90_milli: 25000,
            },
          ],
        });
      }
      if (url.startsWith("/v1/replenishment/suggestions/") && init?.method === "POST") {
        return jsonResponse({
          suggestion_id: "ret-1:SKU-MILK",
          retailer_id: "ret-1",
          sku: "SKU-MILK",
          status: "CONVERTED",
          approved_qty: 20,
          order_id: "ord-retail-99",
        });
      }
      return jsonResponse({});
    });

    render(<SuggestedOrdersApprovePanel onApproved={onApproved} />);

    await waitFor(() => {
      expect(screen.getByText(/Fresh Milk 1L \(SKU-MILK\)/)).toBeInTheDocument();
      expect(screen.getByTestId("retailer-forecast-summary")).toHaveTextContent("CALENDAR_GLOBAL_V2");
      expect(screen.getByText(/P10–P90 12 \/ 18 \/ 25/)).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("Quantity for SKU-MILK"), { target: { value: "20" } });
    fireEvent.click(screen.getByRole("button", { name: "One-tap approve" }));

    await waitFor(() => {
      expect(apiFetchMock).toHaveBeenCalledWith(
        "/v1/replenishment/suggestions/ret-1%3ASKU-MILK/approve",
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({
            retailer_id: "ret-1",
            sku: "SKU-MILK",
            qty: 20,
            mode: "order",
          }),
        }),
      );
      expect(screen.getByText(/Approved SKU-MILK \(20 units\) — status CONVERTED/)).toBeInTheDocument();
      expect(onApproved).toHaveBeenCalledWith(
        expect.objectContaining({ order_id: "ord-retail-99", status: "CONVERTED" }),
      );
    });
  });

  it("contains zero @pegasusx/ui imports across modified retailer desktop files", () => {
    const files = [
      "components/dashboard/SuggestedOrdersApprovePanel.tsx",
      "app/(dashboard)/procurement/page.tsx",
      "app/(dashboard)/dashboard/page.tsx",
      "app/(dashboard)/auto-order/page.tsx",
      "lib/api.ts",
    ];
    for (const rel of files) {
      const src = readFileSync(resolve(process.cwd(), rel), "utf8");
      expect(src, `${rel} must not import @pegasusx/ui`).not.toContain("@pegasusx/ui");
    }
  });
});
