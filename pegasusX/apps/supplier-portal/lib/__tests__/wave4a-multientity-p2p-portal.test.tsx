import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { ApiClient, ApiError } from "@pegasusx/api-core";
import { createTranslator, messageCatalogs } from "@pegasusx/i18n";

const {
  mockRequest,
  mockSearchParamsRef,
  mockSupplierTokenRef,
  mockDownloadFetch,
  mockListKeys,
  mockListHooks,
  mockListDead,
  mockListExports,
  mockGetSftp,
  mockListEdi,
  mockGetCoa,
  mockGetAs2,
} = vi.hoisted(() => ({
  mockRequest: vi.fn(),
  mockSearchParamsRef: { current: new URLSearchParams() },
  mockSupplierTokenRef: { current: "" },
  mockDownloadFetch: vi.fn(),
  mockListKeys: vi.fn(),
  mockListHooks: vi.fn(),
  mockListDead: vi.fn(),
  mockListExports: vi.fn(),
  mockGetSftp: vi.fn(),
  mockListEdi: vi.fn(),
  mockGetCoa: vi.fn(),
  mockGetAs2: vi.fn(),
}));

vi.mock("@/lib/api", () => ({
  createSupplierApi: () => ({
    request: mockRequest,
    listSupplierPartnerKeys: mockListKeys,
    listSupplierPartnerWebhooks: mockListHooks,
    listSupplierPartnerDeadLetter: mockListDead,
    listSupplierPartnerExports: mockListExports,
    getSupplierPartnerSftp: mockGetSftp,
    listSupplierPartnerEdiDocuments: mockListEdi,
    getSupplierPartnerCoa: mockGetCoa,
    getSupplierPartnerAs2: mockGetAs2,
  }),
}));

vi.mock("@/lib/auth", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/auth")>();
  return {
    ...actual,
    getSupplierToken: () => mockSupplierTokenRef.current,
    readTokenFromCookie: () => mockSupplierTokenRef.current,
    supplierFetch: (...args: Parameters<typeof fetch>) => mockDownloadFetch(...args),
  };
});

vi.mock("@/lib/i18n", () => ({
  usePortalT: () => createTranslator("en"),
}));

vi.mock("@/lib/use-supplier-session-reconcile", () => ({
  useSupplierSessionReconcile: () => {},
}));

vi.mock("next/navigation", () => ({
  useSearchParams: () => mockSearchParamsRef.current,
}));

vi.mock("next/link", () => ({
  default: ({
    children,
    href,
    className,
  }: {
    children: ReactNode;
    href: string;
    className?: string;
  }) => (
    <a href={href} className={className}>
      {children}
    </a>
  ),
}));

vi.mock("@/components/settings/OIDCAttachCard", () => ({
  OIDCAttachCard: () => <div data-testid="oidc-attach-mock" />,
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

vi.mock("@/components/StatusChip", () => ({
  default: ({ status, label }: { status: string; label?: string }) => (
    <span>{label || status}</span>
  ),
}));

vi.mock("@/components/StatusBadge", () => ({
  default: ({ state, label }: { state: string; label?: string }) => (
    <span>{label || state}</span>
  ),
}));

vi.mock("@/components/Icon", () => ({
  default: ({ name }: { name: string }) => <span data-icon={name} />,
}));

import {
  computePurchaseOrderPreviewTotals,
  createP2aApi,
  extractBankCreditSummary,
  formatDurationSec,
  formatFxScaledRate,
} from "@/lib/p2a-api";
import { NAV, allNavHrefs } from "@/lib/nav";
import ConsolidationPage from "@/app/(portal)/finance/consolidation/page";
import P2PPage from "@/app/(portal)/p2p/page";
import IntegrationsSettingsPage from "@/app/(portal)/settings/integrations/page";
import AuditorPortalPage from "@/app/(portal)/auditor/page";

const here = dirname(fileURLToPath(import.meta.url));
const portalRoot = join(here, "../..");

function makeJwt(claims: Record<string, unknown>): string {
  const b64 = (obj: unknown) =>
    Buffer.from(JSON.stringify(obj), "utf8")
      .toString("base64")
      .replace(/=+$/g, "")
      .replace(/\+/g, "-")
      .replace(/\//g, "_");
  return `${b64({ alg: "HS256", typ: "JWT" })}.${b64(claims)}.sig`;
}

describe("Wave 4A Multi-Entity Consolidation, P2P 3-Way Match, Process Mining, 2-Way Connectors & Bank Credit Pack", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockSearchParamsRef.current = new URLSearchParams();
    mockSupplierTokenRef.current = makeJwt({
      sub: "owner-1",
      role: "ADMIN",
      supplier_id: "sup-1",
      supplier_role: "OWNER",
    });

    mockListKeys.mockResolvedValue({ keys: [] });
    mockListHooks.mockResolvedValue({ subscriptions: [] });
    mockListDead.mockResolvedValue({ attempts: [] });
    mockListExports.mockResolvedValue({ jobs: [] });
    mockGetSftp.mockResolvedValue({ configured: false });
    mockListEdi.mockResolvedValue({ documents: [] });
    mockGetCoa.mockResolvedValue({
      account_ar: "62.01",
      account_revenue: "90.01",
      account_bank_cash: "51.01",
      using_defaults: true,
    });
    mockGetAs2.mockResolvedValue({ configured: false });
  });

  afterEach(() => {
    cleanup();
  });

  it("calls all Wave 4A endpoints via ApiClient and computes deterministic BigInt math helpers", async () => {
    const fetchSpy = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes("/v1/org/entities") && init?.method === "POST") {
        return new Response(
          JSON.stringify({
            entity: {
              tenant_id: "sup-1",
              entity_id: "ent-sub-1",
              name: "Samarkand Sub",
              country_code: "UZ",
              base_currency: "USD",
              is_parent: false,
              status: "ACTIVE",
              created_at: "2026-11-17T00:00:00Z",
              updated_at: "2026-11-17T00:00:00Z",
            },
          }),
          { status: 201, headers: { "Content-Type": "application/json" } },
        );
      }
      return new Response(
        JSON.stringify({
          ok: true,
          purchase_order: { po_id: "po-1" },
          vendor_invoice: { vendor_invoice_id: "inv-1" },
          template: { code: "P2P-CUSTOM" },
          instance: { instance_id: "inst-1" },
          entity: { entity_id: "ent-1" },
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    });

    const realClient = new ApiClient({
      baseUrl: "https://api.pegasusx.test",
      getAuthToken: () => "tok-4a",
      fetchImpl: fetchSpy as unknown as typeof fetch,
    });
    const api = createP2aApi(realClient);

    await api.listEntities();
    const createdEnt = await api.createEntity({ name: "Samarkand Sub", base_currency: "USD" }, "idem-ent-1");
    expect(createdEnt.entity_id).toBe("ent-sub-1");
    await api.updateEntity("ent-sub-1", { status: "INACTIVE" }, "idem-ent-patch");
    await api.listIntercompanyTransfers({ period: "2026-11" });
    await api.createIntercompanyTransfer(
      {
        from_entity_id: "ent-1",
        to_entity_id: "ent-2",
        transfer_type: "CASH_POOL",
        amount_minor: 500000,
        currency: "UZS",
        period_id: "2026-11",
      },
      "idem-ic-1",
    );
    await api.listFxRevaluations({ period: "2026-11" });
    await api.runFxRevaluation({ period_id: "2026-11", base_currency: "UZS" }, "idem-fx-1");
    await api.getConsolidation({ period: "2026-11", currency: "UZS", entities: ["ent-1", "ent-2"] });

    await api.listProcessTemplates({ domain: "P2P" });
    await api.createProcessTemplate(
      {
        code: "P2P-CUSTOM",
        name: "Custom",
        domain: "P2P",
        steps: [{ code: "CREATED", name: "Created", sequence: 1, sla_seconds: 3600 }],
      },
      "idem-tpl-1",
    );
    await api.listProcessInstances({ object_type: "PurchaseOrder" });
    await api.startOrAdvanceProcessInstance(
      { object_type: "PurchaseOrder", object_id: "po-1", next_step: "APPROVED" },
      "idem-inst-1",
    );
    await api.getProcessMining({ domain: "P2P", entity: "ent-1" });

    await api.listPurchaseOrders({ status: "APPROVED" });
    await api.getPurchaseOrder("po-1");
    await api.createPurchaseOrder(
      {
        vendor_id: "vnd-1",
        currency: "UZS",
        lines: [{ sku: "SKU-1", qty_ordered: 2, unit_cost_minor: 10000, vat_bps: 1200 }],
      },
      "idem-po-1",
    );
    await api.approvePurchaseOrder("po-1", { approval_id: "apr-1" }, "idem-po-app");
    await api.listGoodsReceipts({ po_id: "po-1" });
    await api.postGoodsReceipt(
      { po_id: "po-1", warehouse_id: "wh-1", lines: [{ sku: "SKU-1", qty_received: 2 }] },
      "idem-rec-1",
    );
    await api.listVendorInvoices({ po_id: "po-1" });
    await api.createVendorInvoice(
      { po_id: "po-1", subtotal_minor: 20000, vat_minor: 2400, total_minor: 22400 },
      "idem-inv-1",
    );
    await api.runThreeWayMatch({ vendor_invoice_id: "inv-1", tolerance_bps: 100 }, "idem-3way-1");
    await api.submitConnectorPushApproval(
      {
        object_type: "ConnectorPush",
        object_id: "moysklad",
        action: "push",
        payload: { stream: "goods_receipts" },
      },
      "idem-push-apr-1",
    );
    await api.pushConnector(
      "moysklad",
      { account_label: "ms-main", stream: "goods_receipts", approval_id: "apr-99" },
      "idem-push-1",
    );

    const calledUrls = fetchSpy.mock.calls.map(([u, i]) => ({
      url: String(u),
      method: i?.method,
      headers: i?.headers instanceof Headers ? i.headers : new Headers(i?.headers),
    }));

    expect(calledUrls.find((c) => c.url.includes("/v1/finance/consolidation"))?.url).toBe(
      "https://api.pegasusx.test/v1/finance/consolidation?period=2026-11&currency=UZS&entities=ent-1%2Cent-2",
    );
    expect(calledUrls.find((c) => c.url.includes("/v1/finance/intercompany-transfers") && c.method === "POST")?.headers.get("Idempotency-Key")).toBe("idem-ic-1");
    expect(calledUrls.find((c) => c.url.includes("/v1/finance/fx-revaluation") && c.method === "POST")?.headers.get("Idempotency-Key")).toBe("idem-fx-1");
    expect(calledUrls.find((c) => c.url.includes("/v1/p2p/three-way-match"))?.headers.get("Idempotency-Key")).toBe("idem-3way-1");
    expect(calledUrls.find((c) => c.url.includes("/v1/approvals/requests"))?.method).toBe("POST");
    expect(calledUrls.find((c) => c.url.includes("/v1/connectors/moysklad/push"))?.headers.get("Idempotency-Key")).toBe("idem-push-1");

    // BigInt math verification (zero floats)
    expect(formatFxScaledRate(1285000000000, 100000000)).toBe("12,850");
    expect(formatFxScaledRate(1285025000000, 100000000)).toBe("12,850.25");
    expect(formatFxScaledRate(100000000, 100000000, true)).toBe("1 (inv)");
    expect(formatDurationSec(45)).toBe("45s");
    expect(formatDurationSec(3720)).toBe("1h 2m");
    expect(formatDurationSec(90000)).toBe("1d 1h");
    const totals = computePurchaseOrderPreviewTotals([
      { qtyOrdered: 3, unitCostMinor: 10000, vatBps: 1200 },
    ]);
    expect(totals.subtotalMinor).toBe(BigInt(30000));
    expect(totals.vatMinor).toBe(BigInt(3600));
    expect(totals.totalMinor).toBe(BigInt(33600));

    expect(
      extractBankCreditSummary({
        pack_id: "p1",
        tenant_id: "t1",
        pack_type: "BANK",
        from_date: "",
        to_date: "",
        format: "zip",
        status: "READY",
        watermark_text: "wm",
        row_count: 1,
        sha256_hex: "abc",
        created_by: "u1",
        expires_at: "",
        created_at: "",
        manifest_json: JSON.stringify({
          bank_credit_summary: { total_invoices: 7, dpd_buckets: [], debtor_concentration: [], loss_recovery_curves: [] },
        }),
      })?.total_invoices,
    ).toBe(7);
  });

  it("renders /finance/consolidation, supports entity/IC/FX workflows, handles 422 fx_rate_missing, and enforces AUDITOR read-only", async () => {
    mockRequest.mockImplementation(async (path: string, method: string, opts?: { body?: Record<string, unknown> }) => {
      if (path === "/v1/org/entities" && method === "GET") {
        return {
          tenant_id: "sup-1",
          count: 2,
          entities: [
            {
              tenant_id: "sup-1",
              entity_id: "ent-hq",
              name: "Tashkent HQ",
              tin: "300111222",
              country_code: "UZ",
              base_currency: "UZS",
              is_parent: true,
              status: "ACTIVE",
              created_at: "2026-11-01T00:00:00Z",
              updated_at: "2026-11-01T00:00:00Z",
            },
            {
              tenant_id: "sup-1",
              entity_id: "ent-kz",
              name: "Almaty Sub",
              tin: "990111222",
              country_code: "KZ",
              base_currency: "USD",
              is_parent: false,
              parent_entity_id: "ent-hq",
              status: "ACTIVE",
              created_at: "2026-11-01T00:00:00Z",
              updated_at: "2026-11-01T00:00:00Z",
            },
          ],
        };
      }
      if (path.startsWith("/v1/finance/consolidation") && method === "GET") {
        return {
          tenant_id: "sup-1",
          period_id: "2026-11",
          reporting_currency: "UZS",
          entity_ids: ["ent-hq", "ent-kz"],
          entities: [
            {
              entity_id: "ent-hq",
              name: "Tashkent HQ",
              country_code: "UZ",
              base_currency: "UZS",
              is_parent: true,
              debit_minor: 50000000,
              credit_minor: 50000000,
              net_minor: 0,
              converted_debit_minor: 50000000,
              converted_credit_minor: 50000000,
              converted_net_minor: 0,
              balanced: true,
            },
            {
              entity_id: "ent-kz",
              name: "Almaty Sub",
              country_code: "KZ",
              base_currency: "USD",
              is_parent: false,
              fx_rate_scaled: 1280000000000,
              fx_scale: 100000000,
              debit_minor: 10000,
              credit_minor: 10000,
              net_minor: 0,
              converted_debit_minor: 128000000,
              converted_credit_minor: 128000000,
              converted_net_minor: 0,
              balanced: true,
            },
          ],
          eliminations: [
            {
              transfer_id: "ict-1",
              account_code: "IC:DUE_FROM",
              from_entity_id: "ent-hq",
              to_entity_id: "ent-kz",
              currency: "UZS",
              amount_minor: 10000000,
              converted_amount_minor: 10000000,
              transfer_type: "CASH_POOL",
              description: "Group liquidity sweep",
            },
          ],
          accounts: [
            {
              account_code: "BANK:OPERATING",
              account_name: "Operating Bank",
              class: "ASSET",
              currency: "UZS",
              pre_elimination_debit_minor: 178000000,
              pre_elimination_credit_minor: 0,
              debit_minor: 168000000,
              credit_minor: 0,
              eliminated_minor: 10000000,
              net_minor: 168000000,
            },
          ],
          total_debits_minor: 168000000,
          total_credits_minor: 168000000,
          total_eliminations_minor: 10000000,
          ic_variance_minor: 0,
          balanced: true,
          fx_rates_applied: [
            {
              base_currency: "USD",
              quote_currency: "UZS",
              rate_scaled: 1280000000000,
              scale: 100000000,
              source: "CBU",
            },
          ],
        };
      }
      if (path.startsWith("/v1/finance/intercompany-transfers") && method === "GET") {
        return { tenant_id: "sup-1", count: 0, transfers: [] };
      }
      if (path.startsWith("/v1/finance/fx-revaluation") && method === "GET") {
        return { tenant_id: "sup-1", count: 0, runs: [] };
      }
      if (path === "/v1/org/entities" && method === "POST") {
        return {
          entity: {
            tenant_id: "sup-1",
            entity_id: "ent-bukhara",
            name: opts?.body?.name || "Bukhara LLC",
            country_code: "UZ",
            base_currency: "UZS",
            is_parent: false,
            status: "ACTIVE",
            created_at: "2026-11-17T00:00:00Z",
            updated_at: "2026-11-17T00:00:00Z",
          },
        };
      }
      if (path === "/v1/finance/intercompany-transfers" && method === "POST") {
        return {
          transfer: {
            tenant_id: "sup-1",
            transfer_id: "ict-99",
            from_entity_id: opts?.body?.from_entity_id,
            to_entity_id: opts?.body?.to_entity_id,
            transfer_type: "CASH_POOL",
            amount_minor: 100000,
            currency: "UZS",
            period_id: "2026-11",
            status: "POSTED",
            created_by: "owner-1",
            created_at: "2026-11-17T00:00:00Z",
          },
        };
      }
      if (path === "/v1/finance/fx-revaluation" && method === "POST") {
        throw new ApiError("fx rate missing", 422, {
          code: "fx_rate_missing",
          error: "missing FX rate for EUR/UZS",
        });
      }
      return {};
    });

    render(<ConsolidationPage />);

    await waitFor(() => {
      expect(screen.getByText("Tashkent HQ")).toBeInTheDocument();
    });
    expect(screen.getByText("Group liquidity sweep")).toBeInTheDocument();
    expect(screen.getByText("BANK:OPERATING")).toBeInTheDocument();
    expect(screen.getByText(/USD\/UZS/)).toBeInTheDocument();

    // Legal Entities tab -> create entity
    fireEvent.click(screen.getByRole("tab", { name: "Legal Entities" }));
    fireEvent.change(screen.getByLabelText("Legal Entity Name"), {
      target: { value: "Bukhara LLC" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Create Legal Entity" }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/org/entities",
        "POST",
        expect.objectContaining({
          body: expect.objectContaining({ name: "Bukhara LLC" }),
        }),
      );
    });

    // Intercompany tab -> post transfer
    fireEvent.click(screen.getByRole("tab", { name: "Intercompany Transfers" }));
    fireEvent.change(screen.getByLabelText("Amount"), { target: { value: "1000.00" } });
    fireEvent.click(screen.getByRole("button", { name: "Post Intercompany Transfer" }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/finance/intercompany-transfers",
        "POST",
        expect.objectContaining({
          body: expect.objectContaining({
            from_entity_id: "ent-hq",
            to_entity_id: "ent-kz",
            amount_minor: 100000,
          }),
        }),
      );
    });

    // FX Revaluation tab -> verifies fail-closed 422 alert
    fireEvent.click(screen.getByRole("tab", { name: "FX Revaluation" }));
    fireEvent.click(screen.getByRole("button", { name: "Run FX Revaluation" }));
    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent(/fx_rate_missing/);
    });

    cleanup();

    // AUDITOR read-only safe mode check
    mockSupplierTokenRef.current = makeJwt({
      sub: "aud-1",
      role: "ADMIN",
      supplier_id: "sup-1",
      supplier_role: "AUDITOR",
      auditor_expires_at: new Date(Date.now() + 3600 * 1000).toISOString(),
    });
    render(<ConsolidationPage />);
    await waitFor(() => {
      expect(screen.getByTestId("consolidation-read-only-banner")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole("tab", { name: "Legal Entities" }));
    expect(screen.queryByTestId("create-legal-entity-form")).toBeNull();
    expect(screen.queryByRole("button", { name: "Toggle Status" })).toBeNull();
    fireEvent.click(screen.getByRole("tab", { name: "Intercompany Transfers" }));
    expect(screen.queryByTestId("intercompany-transfer-form")).toBeNull();
    fireEvent.click(screen.getByRole("tab", { name: "FX Revaluation" }));
    expect(screen.queryByTestId("fx-revaluation-form")).toBeNull();
  });

  it("renders /p2p lifecycle (PO with 202 approval, Goods Receipt, Vendor Invoice, 3-Way Match, Process Mining) and enforces AUDITOR read-only", async () => {
    mockRequest.mockImplementation(async (path: string, method: string, opts?: { body?: Record<string, unknown> }) => {
      if (path.startsWith("/v1/p2p/purchase-orders") && method === "GET") {
        return {
          count: 1,
          purchase_orders: [
            {
              tenant_id: "sup-1",
              po_id: "po-100",
              vendor_id: "vnd-uz-1",
              vendor_name: "Tashkent Sugar Co",
              currency: "UZS",
              subtotal_minor: 100000,
              vat_minor: 12000,
              total_minor: 112000,
              status: "PENDING_APPROVAL",
              approval_request_id: "apr-po-100",
              created_by: "buyer-1",
              created_at: "2026-11-17T01:00:00Z",
              updated_at: "2026-11-17T01:00:00Z",
              lines: [
                {
                  tenant_id: "sup-1",
                  po_id: "po-100",
                  line_num: 1,
                  sku: "SUGAR-50KG",
                  qty_ordered: 10,
                  qty_received: 0,
                  unit_cost_minor: 10000,
                  vat_bps: 1200,
                  line_total_minor: 112000,
                },
              ],
            },
          ],
        };
      }
      if (path.startsWith("/v1/p2p/receipts") && method === "GET") {
        return { count: 0, receipts: [] };
      }
      if (path.startsWith("/v1/p2p/vendor-invoices") && method === "GET") {
        return {
          count: 1,
          vendor_invoices: [
            {
              tenant_id: "sup-1",
              vendor_invoice_id: "vinv-100",
              po_id: "po-100",
              vendor_id: "vnd-uz-1",
              currency: "UZS",
              subtotal_minor: 100000,
              vat_minor: 12000,
              total_minor: 112000,
              match_status: "PENDING",
              variance_minor: 0,
              created_by: "ap-clerk-1",
              created_at: "2026-11-17T01:10:00Z",
              updated_at: "2026-11-17T01:10:00Z",
            },
          ],
        };
      }
      if (path.startsWith("/v1/processes/templates") && method === "GET") {
        return {
          count: 1,
          templates: [
            {
              tenant_id: "sup-1",
              template_id: "tpl-p2p-uz",
              code: "P2P-STD-UZ",
              name: "Standard UZ P2P",
              domain: "P2P",
              is_default: true,
              built_in: true,
              status: "ACTIVE",
              steps: [
                { code: "CREATED", name: "Created", sequence: 1, sla_seconds: 3600 },
                { code: "MATCHED", name: "Matched", sequence: 2, sla_seconds: 86400 },
              ],
              created_at: "2026-11-01T00:00:00Z",
              updated_at: "2026-11-01T00:00:00Z",
            },
          ],
        };
      }
      if (path.startsWith("/v1/processes/instances") && method === "GET") {
        return { count: 0, instances: [] };
      }
      if (path.startsWith("/v1/processes/mining") && method === "GET") {
        if (!path.includes("domain=P2P")) {
          return {
            tenant_id: "sup-1",
            total_cases: 0,
            total_transitions: 0,
            sla_breach_count: 0,
            bottleneck_stage: null,
            empty: true,
            edges: [],
          };
        }
        return {
          tenant_id: "sup-1",
          total_cases: 12,
          total_transitions: 12,
          sla_breach_count: 2,
          empty: false,
          bottleneck_stage: {
            from_step: "APPROVED",
            to_step: "RECEIVED",
            domain: "P2P",
            case_count: 12,
            transition_count: 12,
            median_duration_sec: 7200,
            p95_duration_sec: 90000,
            sla_target_sec: 86400,
            sla_breach_count: 2,
          },
          edges: [
            {
              from_step: "APPROVED",
              to_step: "RECEIVED",
              domain: "P2P",
              case_count: 12,
              transition_count: 12,
              avg_duration_sec: 14400,
              median_duration_sec: 7200,
              p95_duration_sec: 90000,
              sla_target_sec: 86400,
              sla_breach_count: 2,
            },
          ],
        };
      }
      if (path === "/v1/org/entities" && method === "GET") {
        return { tenant_id: "sup-1", count: 0, entities: [] };
      }
      if (path === "/v1/p2p/purchase-orders" && method === "POST") {
        return {
          purchase_order: {
            tenant_id: "sup-1",
            po_id: "po-202",
            vendor_id: opts?.body?.vendor_id,
            currency: "UZS",
            subtotal_minor: 50000,
            vat_minor: 6000,
            total_minor: 56000,
            status: "PENDING_APPROVAL",
            approval_request_id: "apr-202",
            created_by: "owner-1",
            created_at: "2026-11-17T02:00:00Z",
            updated_at: "2026-11-17T02:00:00Z",
            lines: [],
          },
          approval_required: true,
          approval_request_id: "apr-202",
        };
      }
      if (path === "/v1/p2p/purchase-orders/po-100/approve" && method === "POST") {
        return {
          purchase_order: {
            po_id: "po-100",
            status: "APPROVED",
          },
        };
      }
      if (path === "/v1/p2p/three-way-match" && method === "POST") {
        return {
          vendor_invoice: {
            tenant_id: "sup-1",
            vendor_invoice_id: "vinv-100",
            po_id: "po-100",
            vendor_id: "vnd-uz-1",
            currency: "UZS",
            subtotal_minor: 100000,
            vat_minor: 12000,
            total_minor: 112000,
            match_status: "MATCHED_3WAY",
            variance_minor: 0,
            journal_entry_id: "je-3way-1",
            created_by: "ap-clerk-1",
            created_at: "2026-11-17T01:10:00Z",
            updated_at: "2026-11-17T02:10:00Z",
          },
          matched: true,
          match_status: "MATCHED_3WAY",
          variance_minor: 0,
          journal_entry_id: "je-3way-1",
          verified_by: "owner-1",
        };
      }
      return {};
    });

    render(<P2PPage />);

    await waitFor(() => {
      expect(screen.getByText("po-100")).toBeInTheDocument();
    });

    // Create PO triggering approval challenge (202 / PENDING_APPROVAL)
    const poForm = screen.getByTestId("create-po-form");
    fireEvent.change(within(poForm).getByLabelText("Vendor ID"), {
      target: { value: "vnd-samarkand" },
    });
    fireEvent.change(within(poForm).getByLabelText("SKU 1"), {
      target: { value: "FLOUR-25KG" },
    });
    fireEvent.change(within(poForm).getByLabelText("Unit Cost 1"), {
      target: { value: "500.00" },
    });
    fireEvent.click(within(poForm).getByRole("button", { name: "Create Purchase Order" }));
    await waitFor(() => {
      expect(screen.getByText(/requires approval \(apr-202\)/i)).toBeInTheDocument();
    });
    expect(screen.getByRole("link", { name: "Review in Approvals" })).toHaveAttribute(
      "href",
      "/approvals?requestId=apr-202",
    );

    // Approve pending PO
    fireEvent.click(screen.getByRole("button", { name: "Approve PO" }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/p2p/purchase-orders/po-100/approve",
        "POST",
        expect.any(Object),
      );
    });

    // 3-Way Match tab -> run 3-way match
    fireEvent.click(screen.getByRole("tab", { name: "3-Way Match" }));
    const matchForm = screen.getByTestId("three-way-match-form");
    fireEvent.click(within(matchForm).getByRole("button", { name: "Run 3-Way Match" }));
    await waitFor(() => {
      expect(screen.getByTestId("three-way-match-outcome")).toHaveTextContent(
        /MATCHED 3-WAY.*je-3way-1/i,
      );
    });

    // Process Mining tab -> initially empty, then populated after filter reload
    fireEvent.click(screen.getByRole("tab", { name: "Process Mining & Templates" }));
    expect(screen.getByTestId("process-mining-empty")).toBeInTheDocument();
    expect(screen.getByText(/P2P-STD-UZ/)).toBeInTheDocument();

    fireEvent.change(screen.getAllByLabelText("Domain")[0], { target: { value: "P2P" } });
    await waitFor(() => {
      expect(screen.getByTestId("process-mining-results")).toBeInTheDocument();
    });
    expect(screen.getByText(/APPROVED → RECEIVED/)).toBeInTheDocument();
    expect(screen.getByText("1d 1h")).toBeInTheDocument();

    cleanup();

    // AUDITOR read-only enforcement on /p2p
    mockSupplierTokenRef.current = makeJwt({
      sub: "aud-1",
      role: "ADMIN",
      supplier_id: "sup-1",
      supplier_role: "AUDITOR",
      auditor_expires_at: new Date(Date.now() + 3600 * 1000).toISOString(),
    });
    render(<P2PPage />);
    await waitFor(() => {
      expect(screen.getByTestId("p2p-read-only-banner")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("create-po-form")).toBeNull();
    expect(screen.queryByRole("button", { name: "Approve PO" })).toBeNull();
    fireEvent.click(screen.getByRole("tab", { name: "3-Way Match" }));
    expect(screen.queryByTestId("three-way-match-form")).toBeNull();
  });

  it("adds moysklad/smartup/iiko and approval-gated 2-way Push Sync on /settings/integrations while keeping AUDITOR safe mode, and renders Bank Credit Tape preview on /auditor", async () => {
    mockRequest.mockImplementation(async (path: string, method: string, opts?: { body?: Record<string, unknown> }) => {
      if (path === "/v1/connectors" && method === "GET") {
        return {
          tenant_id: "sup-1",
          adapters: [
            { provider: "billz", display_name: "Billz", capabilities: ["health", "pull", "push"] },
            { provider: "onec", display_name: "1C", capabilities: ["health", "pull", "push"] },
            { provider: "moysklad", display_name: "MoySklad", capabilities: ["health", "pull", "push"] },
            { provider: "smartup", display_name: "Smartup", capabilities: ["health", "pull", "push"] },
            { provider: "iiko", display_name: "iiko", capabilities: ["health", "pull", "push"] },
          ],
          credentials: [],
          latest_runs: [],
        };
      }
      if (path.startsWith("/v1/connectors/runs") && method === "GET") {
        return { tenant_id: "sup-1", count: 0, runs: [] };
      }
      if (path.startsWith("/v1/approvals/inbox") && method === "GET") {
        return {
          requests: [
            {
              tenant_id: "sup-1",
              request_id: "apr-push-ok",
              policy_id: "pol-1",
              policy_version: 1,
              object_type: "ConnectorPush",
              object_id: "iiko",
              action: "push",
              amount_minor: 0,
              requested_by: "ops-1",
              actor_type: "human",
              payload_hash: "h1",
              status: "APPROVED",
              current_step: 1,
              required_approvers: 1,
              created_at: "2026-11-17T00:00:00Z",
            },
          ],
          count: 1,
        };
      }
      if (path === "/v1/approvals/requests" && method === "POST") {
        return {
          required: true,
          status: "PENDING",
          request: {
            request_id: "apr-push-new",
            object_type: "ConnectorPush",
            object_id: opts?.body?.object_id || "moysklad",
            action: "push",
            status: "PENDING",
            payload_hash: "hash-new",
          },
        };
      }
      if (path === "/v1/connectors/iiko/push" && method === "POST") {
        return {
          run: {
            run_id: "run-push-iiko-1",
            tenant_id: "sup-1",
            provider: "iiko",
            account_label: "default",
            direction: "PUSH",
            stream_name: "goods_receipts",
            status: "COMPLETED",
            records_processed: 4,
            mismatch_count: 0,
            started_at: "2026-11-17T04:00:00Z",
          },
          push_result: {
            provider: "iiko",
            account_label: "default",
            stream: "goods_receipts",
            records_pushed: 4,
            external_doc_id: "IIKO-DOC-77",
            status: "COMPLETED",
            message: "pushed",
          },
          approval_id: opts?.body?.approval_id || "apr-push-ok",
        };
      }
      if (path.startsWith("/v1/audit/verify") && method === "GET") {
        return { tenant_id: "sup-1", day_bucket: "2026-11-17", valid: true, event_count: 2, root_hash: "abc" };
      }
      if (path.startsWith("/v1/audit/anchors") && method === "GET") {
        return { tenant_id: "sup-1", count: 0, anchors: [] };
      }
      if (path === "/v1/finance/packs" || path.startsWith("/v1/finance/packs?")) {
        return {
          tenant_id: "sup-1",
          count: 1,
          packs: [
            {
              pack_id: "pack-bank-4a",
              tenant_id: "sup-1",
              pack_type: "BANK",
              period_id: "2026-11",
              from_date: "2026-11-01T00:00:00Z",
              to_date: "2026-11-30T23:59:59Z",
              format: "zip",
              status: "READY",
              watermark_text: "Prepared for bank-1",
              row_count: 420,
              sha256_hex: "deadbeef",
              created_by: "fin-1",
              expires_at: new Date(Date.now() + 72 * 3600 * 1000).toISOString(),
              created_at: "2026-11-17T03:00:00Z",
            },
          ],
        };
      }
      if (path === "/v1/finance/packs/pack-bank-4a" && method === "GET") {
        return {
          pack: {
            pack_id: "pack-bank-4a",
            tenant_id: "sup-1",
            pack_type: "BANK",
            period_id: "2026-11",
            from_date: "2026-11-01T00:00:00Z",
            to_date: "2026-11-30T23:59:59Z",
            format: "zip",
            status: "READY",
            watermark_text: "Prepared for bank-1",
            row_count: 420,
            sha256_hex: "deadbeef",
            created_by: "fin-1",
            expires_at: new Date(Date.now() + 72 * 3600 * 1000).toISOString(),
            created_at: "2026-11-17T03:00:00Z",
            manifest: {
              bank_credit_summary: {
                as_of: "2026-11-17T03:00:00Z",
                currency: "UZS",
                total_invoices: 18,
                total_retailers: 6,
                consent_verified_retailers: 5,
                total_principal_minor: 500000000,
                total_exposure_minor: 240000000,
                total_recovered_minor: 250000000,
                total_written_off_minor: 10000000,
                top_1_concentration_bps: 3500,
                top_5_concentration_bps: 9200,
                top_10_concentration_bps: 10000,
                avg_relationship_score: 742,
                avg_network_score: 718,
                dpd_buckets: [
                  {
                    bucket: "0-30",
                    invoice_count: 14,
                    retailer_count: 4,
                    principal_minor: 350000000,
                    exposure_minor: 180000000,
                    share_bps: 7500,
                    currency: "UZS",
                  },
                  {
                    bucket: "31-60",
                    invoice_count: 4,
                    retailer_count: 2,
                    principal_minor: 150000000,
                    exposure_minor: 60000000,
                    share_bps: 2500,
                    currency: "UZS",
                  },
                ],
                debtor_concentration: [
                  {
                    rank: 1,
                    retailer_id: "ret-mega-uz",
                    invoice_count: 5,
                    exposure_minor: 84000000,
                    principal_minor: 150000000,
                    credit_limit_minor: 200000000,
                    concentration_bps: 3500,
                    cumulative_bps: 3500,
                    max_dpd_days: 12,
                    consent_verified: true,
                    relationship_score: 790,
                    network_score: 765,
                    currency: "UZS",
                  },
                ],
                loss_recovery_curves: [
                  {
                    vintage_month: "2026-10",
                    invoice_count: 10,
                    origination_minor: 300000000,
                    recovered_minor: 200000000,
                    written_off_minor: 10000000,
                    outstanding_minor: 90000000,
                    default_exposure_minor: 10000000,
                    recovery_rate_bps: 6667,
                    loss_rate_bps: 333,
                    default_rate_bps: 333,
                    currency: "UZS",
                  },
                ],
              },
            },
          },
        };
      }
      if (path.startsWith("/v1/iam/auditor-grants") && method === "GET") {
        return { tenant_id: "sup-1", count: 0, grants: [] };
      }
      return {};
    });

    render(<IntegrationsSettingsPage />);
    await waitFor(() => {
      expect(screen.getByTestId("connector-push-sync-section")).toBeInTheDocument();
    });

    // Provider selector supports moysklad, smartup, iiko
    const providerSelect = screen.getByLabelText("Provider") as HTMLSelectElement;
    const optionValues = Array.from(providerSelect.options).map((o) => o.value);
    expect(optionValues).toEqual(["billz", "onec", "moysklad", "smartup", "iiko"]);

    // Pull Stream filters per provider (omits unsupported "invoices" and omits "shops" for onec)
    const pullStreamSelect = screen.getByLabelText("Pull Stream") as HTMLSelectElement;
    expect(Array.from(pullStreamSelect.options).map((o) => o.value)).toEqual([
      "all",
      "products",
      "stocks",
      "orders",
      "clients",
      "shops",
    ]);
    fireEvent.change(pullStreamSelect, { target: { value: "shops" } });
    expect(pullStreamSelect.value).toBe("shops");
    fireEvent.change(providerSelect, { target: { value: "onec" } });
    expect(Array.from(pullStreamSelect.options).map((o) => o.value)).toEqual([
      "all",
      "products",
      "stocks",
      "orders",
      "clients",
    ]);
    expect(pullStreamSelect.value).toBe("all");
    fireEvent.change(providerSelect, { target: { value: "moysklad" } });
    expect(Array.from(pullStreamSelect.options).map((o) => o.value)).toEqual([
      "all",
      "products",
      "stocks",
      "orders",
      "clients",
      "shops",
    ]);

    // Push Stream filters per provider & resets to provider-specific default
    const pushProviderSelect = screen.getByLabelText("Push Provider") as HTMLSelectElement;
    const pushStreamSelect = screen.getByLabelText("Push Stream") as HTMLSelectElement;
    expect(Array.from(pushStreamSelect.options).map((o) => o.value)).toEqual([
      "goods_receipts",
      "inventory_adjustments",
      "products",
    ]);
    expect(pushStreamSelect.value).toBe("goods_receipts");

    fireEvent.change(pushProviderSelect, { target: { value: "onec" } });
    expect(Array.from(pushStreamSelect.options).map((o) => o.value)).toEqual([
      "journals",
      "goods_receipts",
      "products",
    ]);
    expect(pushStreamSelect.value).toBe("journals");

    fireEvent.change(pushProviderSelect, { target: { value: "smartup" } });
    expect(Array.from(pushStreamSelect.options).map((o) => o.value)).toEqual([
      "goods_receipts",
      "inventory_adjustments",
      "journals",
      "products",
    ]);
    expect(pushStreamSelect.value).toBe("goods_receipts");

    // Request Push Approval for moysklad (resets default to inventory_adjustments)
    fireEvent.change(pushProviderSelect, { target: { value: "moysklad" } });
    expect(Array.from(pushStreamSelect.options).map((o) => o.value)).toEqual([
      "inventory_adjustments",
      "goods_receipts",
      "orders",
    ]);
    expect(pushStreamSelect.value).toBe("inventory_adjustments");
    fireEvent.click(screen.getByRole("button", { name: "Request Push Approval" }));
    await waitFor(() => {
      expect(
        screen.getByText(/Push approval request apr-push-new submitted/i),
      ).toBeInTheDocument();
    });
    expect(mockRequest).toHaveBeenCalledWith(
      "/v1/approvals/requests",
      "POST",
      expect.objectContaining({
        body: expect.objectContaining({
          object_id: "moysklad",
          action: "push",
          payload: expect.objectContaining({
            provider: "moysklad",
            stream: "inventory_adjustments",
          }),
        }),
      }),
    );

    // Execute 2-Way Push for iiko using approved request (resets default to goods_receipts)
    fireEvent.change(pushProviderSelect, { target: { value: "iiko" } });
    expect(Array.from(pushStreamSelect.options).map((o) => o.value)).toEqual([
      "goods_receipts",
      "inventory_adjustments",
      "orders",
    ]);
    expect(pushStreamSelect.value).toBe("goods_receipts");
    fireEvent.change(screen.getByLabelText("Select Approved Request"), {
      target: { value: "apr-push-ok" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Execute 2-Way Push" }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/connectors/iiko/push",
        "POST",
        expect.objectContaining({
          body: expect.objectContaining({
            approval_id: "apr-push-ok",
            stream: "goods_receipts",
          }),
        }),
      );
    });
    await waitFor(() => {
      expect(screen.getByText(/IIKO-DOC-77/)).toBeInTheDocument();
    });

    cleanup();

    // AUDITOR safe mode on /settings/integrations hides 2-way push section & mutating buttons
    mockSupplierTokenRef.current = makeJwt({
      sub: "aud-1",
      role: "ADMIN",
      supplier_id: "sup-1",
      supplier_role: "AUDITOR",
      auditor_expires_at: new Date(Date.now() + 3600 * 1000).toISOString(),
    });
    render(<IntegrationsSettingsPage />);
    await waitFor(() => {
      expect(screen.getByTestId("integrations-read-only-banner")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("connector-push-sync-section")).toBeNull();
    expect(screen.queryByRole("button", { name: "Execute 2-Way Push" })).toBeNull();

    cleanup();

    // AUDITOR inspects BANK Credit Tape preview on /auditor
    render(<AuditorPortalPage />);
    await waitFor(() => {
      expect(screen.getByTestId("inspect-bank-pack-pack-bank-4a")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByTestId("inspect-bank-pack-pack-bank-4a"));
    await waitFor(() => {
      expect(screen.getByTestId("bank-credit-pack-preview")).toBeInTheDocument();
    });
    expect(screen.getByText("0-30")).toBeInTheDocument();
    expect(screen.getByText("ret-mega-uz")).toBeInTheDocument();
    expect(screen.getByText("2026-10")).toBeInTheDocument();
    expect(screen.getByText("742 / 718")).toBeInTheDocument();
  });

  it("registers Wave 4A routes in nav, validates all Wave 4A i18n keys across en/ru/uz, wraps search params in Suspense, and has ZERO @pegasusx/ui imports", () => {
    const hrefs = allNavHrefs(NAV);
    expect(hrefs).toContain("/finance/consolidation");
    expect(hrefs).toContain("/p2p");

    const requiredKeys = [
      "portal.nav.consolidation",
      "portal.nav.p2p",
      "supplier_portal.settings.integrations.connector_spine.provider_moysklad",
      "supplier_portal.settings.integrations.connector_spine.provider_smartup",
      "supplier_portal.settings.integrations.connector_spine.provider_iiko",
      "supplier_portal.settings.integrations.connector_spine.push_section_title",
      "supplier_portal.settings.integrations.connector_spine.push_execute_btn",
      "supplier_portal.auditor.bank_preview_title",
      "supplier_portal.auditor.bank_dpd_title",
      "supplier_portal.auditor.bank_conc_title",
      "supplier_portal.consolidation.title",
      "supplier_portal.consolidation.entity_rollup_title",
      "supplier_portal.consolidation.eliminations_title",
      "supplier_portal.consolidation.fx_reval_form_title",
      "supplier_portal.p2p.title",
      "supplier_portal.p2p.po_form_title",
      "supplier_portal.p2p.match_form_title",
      "supplier_portal.p2p.mining_title",
    ];
    for (const key of requiredKeys) {
      expect(messageCatalogs.en[key], key).toBeTruthy();
      expect(messageCatalogs.ru[key], key).toBeTruthy();
      expect(messageCatalogs.uz[key], key).toBeTruthy();
    }

    const wave4aFiles = [
      "lib/p2a-api.ts",
      "lib/nav.ts",
      "app/(portal)/finance/consolidation/page.tsx",
      "app/(portal)/p2p/page.tsx",
      "app/(portal)/settings/integrations/page.tsx",
      "app/(portal)/auditor/page.tsx",
    ];
    for (const rel of wave4aFiles) {
      const fullPath = join(portalRoot, rel);
      expect(existsSync(fullPath), rel).toBe(true);
      const src = readFileSync(fullPath, "utf8");
      expect(src).not.toContain("@pegasusx/ui");
    }

    for (const pageRel of [
      "app/(portal)/finance/consolidation/page.tsx",
      "app/(portal)/p2p/page.tsx",
      "app/(portal)/settings/integrations/page.tsx",
      "app/(portal)/auditor/page.tsx",
    ]) {
      const src = readFileSync(join(portalRoot, pageRel), "utf8");
      expect(src).toContain("<Suspense");
    }
  });
});
