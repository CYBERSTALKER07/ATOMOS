import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { ApiClient } from "@pegasusx/api-core";
import { createTranslator, messageCatalogs } from "@pegasusx/i18n";

const { mockRequest, mockSearchParamsRef } = vi.hoisted(() => ({
  mockRequest: vi.fn(),
  mockSearchParamsRef: { current: new URLSearchParams() },
}));

vi.mock("@/lib/api", () => ({
  createWarehouseApi: () => ({
    request: mockRequest,
  }),
}));

vi.mock("@/lib/warehouse-scope", () => ({
  warehouseHomeNodeId: () => "wh-main",
}));

vi.mock("@/lib/i18n", () => ({
  usePortalT: () => createTranslator("en"),
}));

vi.mock("@/lib/use-warehouse-session-reconcile", () => ({
  useWarehouseSessionReconcile: () => {},
}));

vi.mock("next/navigation", () => ({
  useSearchParams: () => mockSearchParamsRef.current,
}));

vi.mock("next/link", () => ({
  default: ({ children, href, className }: { children: ReactNode; href: string; className?: string }) => (
    <a href={href} className={className}>
      {children}
    </a>
  ),
}));

vi.mock("@/components/PageTransition", () => ({
  default: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

vi.mock("@/components/PageChrome", () => ({
  PageChrome: ({
    title,
    description,
    actions,
    loading,
    error,
    children,
  }: {
    title: string;
    description?: string;
    actions?: ReactNode;
    loading?: boolean;
    error?: string | null;
    children: ReactNode;
  }) => (
    <div>
      <h1>{title}</h1>
      {description ? <p>{description}</p> : null}
      {actions}
      {loading ? <p>loading</p> : error ? <p role="alert">{error}</p> : children}
    </div>
  ),
}));

vi.mock("@/components/Icon", () => ({
  default: ({ name }: { name: string }) => <span data-icon={name} />,
}));

import {
  createWarehouseValuationApi,
  formatWarehouseMinor,
  parseMajorToSafeMinor,
} from "@/lib/valuation-api";
import { NAV, allNavHrefs } from "@/lib/nav";
import WarehouseInventoryValuationPage from "@/app/inventory/valuation/page";

const here = dirname(fileURLToPath(import.meta.url));
const portalRoot = join(here, "../..");

describe("M1.3 Warehouse Inventory Valuation & Movements", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockSearchParamsRef.current = new URLSearchParams();
  });

  afterEach(() => {
    cleanup();
  });

  it("formats and parses minor units strictly via BigInt without float rounding", () => {
    expect(formatWarehouseMinor(BigInt("900719925474099300"), "UZS")).toBe("9,007,199,254,740,993 UZS");
    expect(formatWarehouseMinor(1250000, "UZS")).toBe("12,500 UZS");
    expect(formatWarehouseMinor(1250050, "UZS")).toBe("12,500.50 UZS");
    expect(parseMajorToSafeMinor("150.25")).toBe(15025);
    expect(parseMajorToSafeMinor("150.255")).toBeNull();
  });

  it("calls valuation, movements, and cost-layer receipt endpoints with exact params and idempotency key", async () => {
    const calls: Array<{ url: string; method: string; body?: string; idem?: string | null }> = [];
    const fetchSpy = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const headers = new Headers(init?.headers);
      calls.push({
        url: String(input),
        method: init?.method || "GET",
        body: typeof init?.body === "string" ? init.body : undefined,
        idem: headers.get("Idempotency-Key"),
      });
      return new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });

    const realClient = new ApiClient({
      baseUrl: "https://wh.pegasusx.test",
      getAuthToken: () => "tok-wh",
      fetchImpl: fetchSpy as unknown as typeof fetch,
    });
    const api = createWarehouseValuationApi(realClient);

    await api.getValuation({ warehouse_id: "wh-1", product_id: "sku-1", expiry_risk_days: 14 });
    await api.listMovements({ warehouse_id: "wh-1", movement_type: "ISSUE_COGS", limit: 50 });
    await api.recordCostLayerReceipt(
      {
        warehouse_id: "wh-1",
        product_id: "sku-1",
        lot_id: "lot-1",
        receipt_ref_id: "rcpt-100",
        quantity: 20,
        unit_cost_minor: 1500000,
        vat_minor: 180000,
        credit_account: "AP:VENDOR",
      },
      "idem-rcpt-1",
    );

    expect(calls.map((c) => `${c.method} ${c.url}`)).toEqual([
      "GET https://wh.pegasusx.test/v1/inventory/valuation?warehouse_id=wh-1&product_id=sku-1&expiry_risk_days=14",
      "GET https://wh.pegasusx.test/v1/inventory/movements?warehouse_id=wh-1&movement_type=ISSUE_COGS&limit=50",
      "POST https://wh.pegasusx.test/v1/inventory/cost-layers/receipt",
    ]);
    expect(calls[2].idem).toBe("idem-rcpt-1");
    expect(JSON.parse(calls[2].body || "{}")).toMatchObject({
      warehouse_id: "wh-1",
      quantity: 20,
      unit_cost_minor: 1500000,
      idempotency_key: "idem-rcpt-1",
    });
  });

  it("renders FEFO cost layers, moving-average SKUs, movements audit trail, and submits new receipt layer", async () => {
    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path.startsWith("/v1/inventory/valuation")) {
        return {
          entity_id: "sup-1",
          warehouse_id: "wh-main",
          currency: "UZS",
          total_qty_remaining: 30,
          total_value_minor: 45000000,
          expiry_risk_days: 14,
          skus: [
            {
              entity_id: "sup-1",
              warehouse_id: "wh-main",
              product_id: "sku-cola-1l",
              currency: "UZS",
              total_qty_remaining: 30,
              total_value_minor: 45000000,
              moving_avg_unit_cost_minor: 1500000,
              layer_count: 2,
            },
          ],
          layers: [
            {
              entity_id: "sup-1",
              warehouse_id: "wh-main",
              product_id: "sku-cola-1l",
              layer_id: "layer-fefo-1",
              lot_id: "lot-early",
              receipt_ref_type: "PURCHASE_RECEIPT",
              receipt_ref_id: "po-1",
              qty_remaining: 10,
              original_qty: 20,
              unit_cost_minor: 1400000,
              total_value_minor: 14000000,
              currency: "UZS",
              received_at: "2026-11-01T00:00:00Z",
              expiry_date: "2026-11-24",
              near_expiry: true,
              days_to_expiry: 7,
            },
          ],
        };
      }
      if (path.startsWith("/v1/inventory/movements")) {
        return {
          entity_id: "sup-1",
          count: 1,
          movements: [
            {
              entity_id: "sup-1",
              movement_id: "mov-cogs-1",
              movement_type: "ISSUE_COGS",
              reference_type: "PICK_TASK",
              reference_id: "pick-88",
              warehouse_id: "wh-main",
              product_id: "sku-cola-1l",
              lot_id: "lot-early",
              layer_id: "layer-fefo-1",
              qty_delta: -10,
              unit_cost_minor: 1400000,
              total_cost_minor: 14000000,
              currency: "UZS",
              gl_entry_id: "gl-cogs-88",
              created_at: "2026-11-16T12:00:00Z",
            },
          ],
        };
      }
      if (path === "/v1/inventory/cost-layers/receipt" && method === "POST") {
        return {
          layer: { layer_id: "layer-new-2" },
          movement: { movement_id: "mov-new-2" },
          gl_entry_id: "gl-rcpt-2",
          total_cost_minor: 30000000,
          vat_minor: 0,
        };
      }
      return {};
    });

    render(<WarehouseInventoryValuationPage />);

    await waitFor(() => {
      expect(screen.getByText("450,000 UZS")).toBeInTheDocument();
      expect(screen.getByText("layer-fefo-1")).toBeInTheDocument();
    });
    expect(screen.getByText("Expires in 7d")).toBeInTheDocument();
    expect(screen.getByText("14,000 UZS")).toBeInTheDocument();

    // Moving-average SKU tab
    fireEvent.click(screen.getByRole("tab", { name: "Moving-average SKUs" }));
    expect(screen.getByText("15,000 UZS")).toBeInTheDocument();

    // Movements audit trail tab
    fireEvent.click(screen.getByRole("tab", { name: "Movements audit trail" }));
    expect(screen.getByText("mov-cogs-1")).toBeInTheDocument();
    expect(screen.getByText("-10")).toBeInTheDocument();
    expect(screen.getByText("gl-cogs-88")).toBeInTheDocument();

    // Record new receipt layer using BigInt major->minor input
    fireEvent.click(screen.getByRole("button", { name: "Record receipt layer" }));
    fireEvent.change(document.getElementById("rcpt-product")!, { target: { value: "sku-juice" } });
    fireEvent.change(document.getElementById("rcpt-ref")!, { target: { value: "po-rcpt-7" } });
    fireEvent.change(document.getElementById("rcpt-qty")!, { target: { value: "25" } });
    fireEvent.change(document.getElementById("rcpt-cost")!, { target: { value: "12000" } });
    fireEvent.click(screen.getByRole("button", { name: "Post receipt" }));

    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/inventory/cost-layers/receipt",
        "POST",
        expect.objectContaining({
          body: expect.objectContaining({
            warehouse_id: "wh-main",
            product_id: "sku-juice",
            receipt_ref_id: "po-rcpt-7",
            quantity: 25,
            unit_cost_minor: 1200000,
          }),
        }),
      );
    });
  });

  it("registers /inventory/valuation in nav, links from /inventory, wraps Suspense, localizes i18n, and imports zero @pegasusx/ui", () => {
    expect(allNavHrefs(NAV)).toContain("/inventory/valuation");

    const invSrc = readFileSync(join(portalRoot, "app/inventory/page.tsx"), "utf8");
    expect(invSrc).toContain("/inventory/valuation");

    const valPageSrc = readFileSync(join(portalRoot, "app/inventory/valuation/page.tsx"), "utf8");
    const valApiSrc = readFileSync(join(portalRoot, "lib/valuation-api.ts"), "utf8");
    expect(valPageSrc).toContain("<Suspense");
    expect(valPageSrc).not.toContain("@pegasusx/ui");
    expect(valApiSrc).not.toContain("@pegasusx/ui");
    expect(existsSync(join(portalRoot, "app/inventory/valuation/page.tsx"))).toBe(true);

    for (const key of [
      "portal.nav.inventory_valuation",
      "warehouse_portal.valuation.title",
      "warehouse_portal.valuation.layers_title",
      "warehouse_portal.valuation.movements_title",
    ]) {
      expect(messageCatalogs.en[key], key).toBeTruthy();
      expect(messageCatalogs.ru[key], key).toBeTruthy();
      expect(messageCatalogs.uz[key], key).toBeTruthy();
    }
  });
});
