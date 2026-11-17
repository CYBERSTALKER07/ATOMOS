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
  mockCreatePartnerExport,
  mockGetPartnerExport,
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
  mockCreatePartnerExport: vi.fn(),
  mockGetPartnerExport: vi.fn(),
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
    createSupplierPartnerExport: mockCreatePartnerExport,
    getSupplierPartnerExport: mockGetPartnerExport,
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
  default: ({ children, href, className }: { children: ReactNode; href: string; className?: string }) => (
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
  default: ({ status, label }: { status: string; label?: string }) => <span>{label || status}</span>,
}));

vi.mock("@/components/StatusBadge", () => ({
  default: ({ state, label }: { state: string; label?: string }) => <span>{label || state}</span>,
}));

vi.mock("@/components/Icon", () => ({
  default: ({ name }: { name: string }) => <span data-icon={name} />,
}));

import {
  createConnectorsAuditorApi,
  formatRemainingCountdown,
  parseShadowRunDetail,
  resolvePortalSessionRole,
  unwrapExportPack,
} from "@/lib/connectors-auditor-api";
import { NAV, allNavHrefs } from "@/lib/nav";
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

describe("Wave 3C Connectors, Unlimited Exports, ExportPacks & Auditor Portal (M1.9 + M1.10)", () => {
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
    mockCreatePartnerExport.mockResolvedValue({
      job_id: "job-legacy",
      resource: "orders",
      format: "csv",
      status: "PENDING",
    });
    mockGetPartnerExport.mockResolvedValue({
      job_id: "job-legacy",
      resource: "orders",
      format: "csv",
      status: "SUCCEEDED",
    });
  });

  afterEach(() => {
    cleanup();
  });

  it("calls all M1.9/M1.10 endpoints via ApiClient and handles binary/error pack downloads", async () => {
    const fetchSpy = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes("/v1/finance/packs/pack-1/download")) {
        return new Response(new Uint8Array([80, 75, 3, 4]), {
          status: 200,
          headers: {
            "Content-Type": "application/zip",
            "Content-Disposition": 'attachment; filename="audit-pack-2026-11.zip"',
            "X-Atomos-Watermark": "Prepared for aud-1 | Tenant sup-1 | Pack pack-1",
            "X-Atomos-Pack-Sha256": "deadbeef1234",
          },
        });
      }
      if (url.includes("/v1/finance/packs/pack-expired/download")) {
        return new Response(JSON.stringify({ error: "pack_expired" }), {
          status: 410,
          headers: { "Content-Type": "application/json" },
        });
      }
      return new Response(JSON.stringify({ ok: true, url, method: init?.method }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });

    const realClient = new ApiClient({
      baseUrl: "https://api.pegasusx.test",
      getAuthToken: () => "tok-3c",
      fetchImpl: fetchSpy as unknown as typeof fetch,
    });

    const api = createConnectorsAuditorApi({
      transport: realClient,
      downloadFetch: fetchSpy as unknown as typeof fetch,
      downloadBaseUrl: "https://api.pegasusx.test",
    });

    await api.listConnectors();
    await api.upsertConnectorCredential(
      {
        provider: "billz",
        account_label: "billz-main",
        auth_type: "bearer",
        secret_ref: "env://BILLZ_API_TOKEN",
        base_url: "https://api-admin.billz.ai",
      },
      "idem-cred-1",
    );
    await api.pingConnectorHealth("billz", { account_label: "billz-main" }, "idem-health-1");
    await api.pullConnector("billz", { account_label: "billz-main", stream: "stocks" }, "idem-pull-1");
    await api.runShadowCompare("billz", { account_label: "billz-main" }, "idem-shadow-1");
    await api.listConnectorRuns({ provider: "billz", limit: 20 });
    await api.getConnectorRun("run-1");
    await api.createPartnerExport(
      { resource: "ledger", format: "xlsx", unlimited: true },
      "idem-exp-1",
    );
    await api.createExportPack(
      { pack_type: "TAX", period_id: "2026-11", format: "zip", ttl_hours: 72 },
      "idem-pack-1",
    );
    await api.listExportPacks({ pack_type: "TAX", limit: 10 });
    await api.getExportPack("pack-1");
    const dl = await api.downloadExportPack("pack-1", { format: "zip", token: "tok-dl" });
    expect(dl.filename).toBe("audit-pack-2026-11.zip");
    expect(dl.watermark).toBe("Prepared for aud-1 | Tenant sup-1 | Pack pack-1");
    expect(dl.sha256).toBe("deadbeef1234");
    await expect(
      api.downloadExportPack("pack-expired", { format: "xlsx" }),
    ).rejects.toBeInstanceOf(ApiError);

    await api.listAuditorGrants();
    await api.createAuditorGrant({ user_id: "aud-1", ttl_hours: 48, reason: "Q4 audit" }, "idem-grant-1");
    await api.revokeAuditorGrant("aud-1", "idem-revoke-1");

    const calls = fetchSpy.mock.calls.map(([url, init]) => ({
      url: String(url),
      method: init?.method,
      headers: init?.headers instanceof Headers ? init.headers : new Headers(init?.headers),
      body: init?.body ? JSON.parse(String(init.body)) : undefined,
    }));

    expect(calls[0].url).toBe("https://api.pegasusx.test/v1/connectors");
    expect(calls[1].url).toBe("https://api.pegasusx.test/v1/connectors/credentials");
    expect(calls[1].method).toBe("PUT");
    expect(calls[1].headers.get("Idempotency-Key")).toBe("idem-cred-1");
    expect(calls[2].url).toBe("https://api.pegasusx.test/v1/connectors/billz/health");
    expect(calls[3].url).toBe("https://api.pegasusx.test/v1/connectors/billz/pull");
    expect(calls[3].body).toEqual({
      account_label: "billz-main",
      stream: "stocks",
      idempotency_key: "idem-pull-1",
    });
    expect(calls[4].url).toBe("https://api.pegasusx.test/v1/connectors/billz/shadow-compare");
    expect(calls[5].url).toBe("https://api.pegasusx.test/v1/connectors/runs?provider=billz&limit=20");
    expect(calls[6].url).toBe("https://api.pegasusx.test/v1/connectors/runs/run-1");
    expect(calls[7].url).toBe("https://api.pegasusx.test/v1/supplier/partner-exports");
    expect(calls[7].body).toEqual({ resource: "ledger", format: "xlsx", unlimited: true });
    expect(calls[8].url).toBe("https://api.pegasusx.test/v1/finance/packs");
    expect(calls[9].url).toBe("https://api.pegasusx.test/v1/finance/packs?pack_type=TAX&limit=10");
    expect(calls[10].url).toBe("https://api.pegasusx.test/v1/finance/packs/pack-1");
    expect(calls[11].url).toBe("https://api.pegasusx.test/v1/finance/packs/pack-1/download?format=zip&token=tok-dl");
    expect(calls[13].url).toBe("https://api.pegasusx.test/v1/iam/auditor-grants");
    expect(calls[14].url).toBe("https://api.pegasusx.test/v1/iam/auditor-grants");
    expect(calls[14].method).toBe("POST");
    expect(calls[15].url).toBe("https://api.pegasusx.test/v1/iam/auditor-grants/aud-1/revoke");
    expect(calls[15].method).toBe("POST");
  });

  it("enforces role helpers, expiry countdown math, and shadow-run JSON summary fallback", () => {
    const auditor = resolvePortalSessionRole({
      sub: "aud-1",
      role: "ADMIN",
      supplier_role: "AUDITOR",
      auditor_expires_at: "2026-11-18T00:00:00Z",
    });
    expect(auditor.isAuditor).toBe(true);
    expect(auditor.canManagePacks).toBe(false);
    expect(auditor.canManageAuditorGrants).toBe(false);
    expect(auditor.canManageConnectors).toBe(false);

    const agent = resolvePortalSessionRole({ sub: "bot-1", role: "ADMIN", actor_type: "agent" });
    expect(agent.isAgent).toBe(true);
    expect(agent.canManagePacks).toBe(false);
    expect(agent.canManageConnectors).toBe(false);

    const financeMgr = resolvePortalSessionRole({
      sub: "fin-1",
      role: "ADMIN",
      supplier_role: "FINANCE_MANAGER",
    });
    expect(financeMgr.isAuditor).toBe(false);
    expect(financeMgr.canManagePacks).toBe(true);
    expect(financeMgr.canManageAuditorGrants).toBe(true);

    const countdown = formatRemainingCountdown(
      Date.UTC(2026, 10, 18, 2, 30, 5),
      Date.UTC(2026, 10, 17, 0, 0, 0),
    );
    expect(countdown.expired).toBe(false);
    expect(countdown.formatted).toBe("1d 02:30:05");

    const expired = formatRemainingCountdown(1000, 2000);
    expect(expired.expired).toBe(true);
    expect(expired.formatted).toBe("00:00:00");

    const parsed = parseShadowRunDetail({
      run: {
        run_id: "run-hist",
        tenant_id: "sup-1",
        provider: "billz",
        account_label: "default",
        direction: "SHADOW_COMPARE",
        stream_name: "all",
        status: "COMPLETED",
        records_processed: 5,
        mismatch_count: 1,
        mismatch_summary_json: JSON.stringify({
          summary: { total_mismatches: 1, stock_delta_count: 1 },
          mismatches: [
            {
              mismatch_type: "stock_delta",
              object_type: "stock",
              external_id: "bz-sku-1",
              internal_id: "prd-1",
              external_stock: 20,
              internal_stock: 15,
              stock_delta: 5,
            },
          ],
        }),
        started_at: "2026-11-17T01:00:00Z",
      },
    });
    expect(parsed.summary?.stock_delta_count).toBe(1);
    expect(parsed.mismatches).toHaveLength(1);

    const directPack = {
      pack_id: "pack-direct",
      tenant_id: "sup-1",
      pack_type: "TAX",
      from_date: "2026-11-01T00:00:00Z",
      to_date: "2026-11-30T23:59:59Z",
      format: "zip",
      status: "READY",
      watermark_text: "wm",
      row_count: 10,
      sha256_hex: "abc",
      created_by: "owner-1",
      expires_at: "2026-11-20T00:00:00Z",
      created_at: "2026-11-17T00:00:00Z",
    };
    expect(unwrapExportPack(directPack).pack_id).toBe("pack-direct");
    expect(unwrapExportPack({ pack: directPack }).pack_id).toBe("pack-direct");
  });

  it("renders /settings/integrations with Connector Spine (Billz & 1C), Shadow Compare diff, sync runs, and unlimited XLSX export", async () => {
    mockRequest.mockImplementation(async (path: string, method: string, opts?: { body?: Record<string, unknown> }) => {
      if (path === "/v1/connectors" && method === "GET") {
        return {
          tenant_id: "sup-1",
          adapters: [
            { provider: "billz", display_name: "Billz POS", capabilities: ["health", "pull", "shadow_compare"] },
            { provider: "onec", display_name: "1C CommerceML", capabilities: ["health", "pull", "push"] },
          ],
          credentials: [
            {
              credential_id: "cred-billz-1",
              tenant_id: "sup-1",
              provider: "billz",
              account_label: "billz-main",
              auth_type: "bearer",
              secret_ref_masked: "env://BILLZ_****OKEN",
              base_url: "https://api-admin.billz.ai",
              status: "ACTIVE",
              created_by: "owner-1",
              created_at: "2026-11-17T01:00:00Z",
              updated_at: "2026-11-17T01:00:00Z",
            },
          ],
          latest_runs: [],
        };
      }
      if (path.startsWith("/v1/connectors/runs?") && method === "GET") {
        return {
          tenant_id: "sup-1",
          count: 1,
          runs: [
            {
              run_id: "run-hist-1",
              tenant_id: "sup-1",
              provider: "onec",
              account_label: "1c-tashkent",
              direction: "SHADOW_COMPARE",
              stream_name: "all",
              status: "COMPLETED",
              records_processed: 42,
              mismatch_count: 1,
              started_at: "2026-11-17T02:00:00Z",
            },
          ],
        };
      }
      if (path === "/v1/connectors/credentials" && method === "PUT") {
        return {
          credential: {
            credential_id: "cred-onec-1",
            tenant_id: "sup-1",
            provider: opts?.body?.provider || "onec",
            account_label: opts?.body?.account_label || "1c-tashkent",
            auth_type: "bearer",
            secret_ref_masked: "env://ONEC_****CRET",
            base_url: "https://1c.example.uz/odata",
            status: "ACTIVE",
            created_by: "owner-1",
            created_at: "2026-11-17T03:00:00Z",
            updated_at: "2026-11-17T03:00:00Z",
          },
        };
      }
      if (path === "/v1/connectors/billz/health" && method === "POST") {
        return {
          healthy: true,
          provider: "billz",
          account_label: "billz-main",
          latency_ms: 42,
          status_code: 200,
          message: "OK",
          checked_at: "2026-11-17T03:01:00Z",
        };
      }
      if (path === "/v1/connectors/billz/pull" && method === "POST") {
        return {
          run: {
            run_id: "run-pull-9",
            tenant_id: "sup-1",
            provider: "billz",
            account_label: "billz-main",
            direction: "PULL",
            stream_name: "stocks",
            status: "COMPLETED",
            records_processed: 18,
            mismatch_count: 0,
            started_at: "2026-11-17T03:02:00Z",
          },
          records_count: 18,
        };
      }
      if (path === "/v1/connectors/billz/shadow-compare" && method === "POST") {
        return {
          run: {
            run_id: "run-shadow-7",
            tenant_id: "sup-1",
            provider: "billz",
            account_label: "billz-main",
            direction: "SHADOW_COMPARE",
            stream_name: "all",
            status: "COMPLETED",
            records_processed: 30,
            mismatch_count: 2,
            started_at: "2026-11-17T03:03:00Z",
          },
          summary: {
            total_mismatches: 2,
            missing_external: 0,
            missing_internal: 0,
            stock_delta_count: 1,
            balance_delta_count: 1,
            matched_count: 28,
          },
          mismatches: [
            {
              mismatch_type: "stock_delta",
              object_type: "stock",
              external_id: "BZ-SKU-100",
              internal_id: "PRD-100",
              external_stock: 120,
              internal_stock: 115,
              stock_delta: 5,
              detail: "Warehouse A qty mismatch",
            },
            {
              mismatch_type: "balance_delta_minor",
              object_type: "invoice",
              external_id: "BZ-INV-200",
              internal_id: "INV-200",
              external_amount_minor: 50000000,
              internal_amount_minor: 25000000,
              balance_delta_minor: 25000000,
              currency: "UZS",
              detail: "Partial invoice delta",
            },
          ],
        };
      }
      if (path === "/v1/connectors/runs/run-hist-1" && method === "GET") {
        return {
          run: {
            run_id: "run-hist-1",
            tenant_id: "sup-1",
            provider: "onec",
            account_label: "1c-tashkent",
            direction: "SHADOW_COMPARE",
            stream_name: "all",
            status: "COMPLETED",
            records_processed: 42,
            mismatch_count: 1,
            started_at: "2026-11-17T02:00:00Z",
          },
          summary: { total_mismatches: 1, missing_external_count: 1 },
          mismatches: [
            {
              mismatch_type: "missing_external",
              object_type: "product",
              internal_id: "PRD-1C-ONLY",
              detail: "Not yet exported to 1C",
            },
          ],
        };
      }
      if (path === "/v1/supplier/partner-exports" && method === "POST") {
        return {
          job_id: "exp-unlim-1",
          resource: "ledger",
          format: "xlsx",
          status: "SUCCEEDED",
          download_url: "/downloads/exp-unlim-1.xlsx",
        };
      }
      return {};
    });

    render(<IntegrationsSettingsPage />);

    await waitFor(() => {
      expect(screen.getByTestId("enterprise-connector-spine")).toBeInTheDocument();
    });
    // Existing sections remain rendered
    expect(screen.getByText("API keys")).toBeInTheDocument();
    expect(screen.getByText("Webhooks")).toBeInTheDocument();
    expect(screen.getByText("1C chart of accounts")).toBeInTheDocument();
    expect(screen.getByText("AS2 transport")).toBeInTheDocument();
    expect(screen.getByText("env://BILLZ_****OKEN")).toBeInTheDocument();

    // Upsert 1C credential
    fireEvent.change(screen.getByLabelText("Provider"), { target: { value: "onec" } });
    fireEvent.change(screen.getByLabelText("Account Label"), { target: { value: "1c-tashkent" } });
    fireEvent.change(screen.getByLabelText("Secret Reference"), {
      target: { value: "env://ONEC_SECRET" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Save Credential" }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/connectors/credentials",
        "PUT",
        expect.objectContaining({
          body: expect.objectContaining({
            provider: "onec",
            account_label: "1c-tashkent",
            secret_ref: "env://ONEC_SECRET",
          }),
        }),
      );
    });

    // Health Ping
    fireEvent.click(screen.getByRole("button", { name: "Health Ping" }));
    await waitFor(() => {
      expect(screen.getByText(/billz \(billz-main\) healthy · 42 ms/i)).toBeInTheDocument();
    });

    // Pull Sync with stream=stocks
    fireEvent.change(screen.getByLabelText("Pull Stream"), { target: { value: "stocks" } });
    fireEvent.click(screen.getByRole("button", { name: "Pull Sync" }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/connectors/billz/pull",
        "POST",
        expect.objectContaining({
          body: expect.objectContaining({
            account_label: "billz-main",
            stream: "stocks",
          }),
        }),
      );
    });

    // Shadow Compare runner + BigInt UZS diff table
    fireEvent.click(screen.getByRole("button", { name: "Shadow Compare" }));
    await waitFor(() => {
      expect(screen.getByText("BZ-SKU-100")).toBeInTheDocument();
    });
    expect(screen.getByText("120 / 115 / 5")).toBeInTheDocument();
    expect(screen.getByText(/250,000 UZS/)).toBeInTheDocument();

    // Historical run inspection
    fireEvent.click(screen.getByRole("button", { name: "Inspect Diff" }));
    await waitFor(() => {
      expect(screen.getByText("PRD-1C-ONLY")).toBeInTheDocument();
    });

    // Unlimited + XLSX export
    fireEvent.change(screen.getByLabelText("Export format"), { target: { value: "xlsx" } });
    fireEvent.click(
      screen.getByLabelText("Unlimited export (bypass 50k row / 90-day cap)"),
    );
    fireEvent.click(screen.getByRole("button", { name: "Request export" }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/supplier/partner-exports",
        "POST",
        expect.objectContaining({
          body: expect.objectContaining({
            format: "xlsx",
            unlimited: true,
          }),
        }),
      );
    });
  });

  it("renders /settings/integrations in read-only safe mode for AUDITOR with ZERO mutating controls across Connector Spine and legacy sections", async () => {
    mockSupplierTokenRef.current = makeJwt({
      sub: "auditor-ext-1",
      role: "ADMIN",
      supplier_id: "sup-1",
      supplier_role: "AUDITOR",
      auditor_expires_at: new Date(Date.now() + 2 * 3600 * 1000).toISOString(),
    });

    mockListKeys.mockResolvedValue({
      keys: [{ key_id: "k-1", key_prefix: "pxk_live", status: "ACTIVE", scopes: ["orders:read"] }],
    });
    mockListHooks.mockResolvedValue({
      subscriptions: [
        {
          subscription_id: "wh-1",
          url: "https://example.com/hook",
          is_active: true,
          event_types: ["ORDER_CREATED"],
        },
      ],
    });
    mockListDead.mockResolvedValue({
      attempts: [{ attempt_id: "dlq-1", event_type: "ORDER_CREATED", status: "FAILED", last_error: "502" }],
    });
    mockListExports.mockResolvedValue({
      jobs: [{ job_id: "exp-1", resource: "orders", format: "csv", status: "SUCCEEDED", download_url: "https://dl.test/a.csv" }],
    });
    mockGetSftp.mockResolvedValue({
      configured: true,
      host: "sftp.partner.uz",
      port: 22,
      edi_enabled: true,
      inbound_dir: "in",
      outbound_dir: "out",
      archive_dir: "arch",
    });
    mockListEdi.mockResolvedValue({
      documents: [{ document_id: "edi-1", direction: "OUTBOUND", doc_type: "INVOIC", external_doc_id: "INV-1", status: "FAILED" }],
    });
    mockGetAs2.mockResolvedValue({
      configured: true,
      as2_enabled: true,
      our_as2_id: "ATOMOS-AS2",
      partner_as2_id: "RETAILER-AS2",
    });

    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path === "/v1/connectors" && method === "GET") {
        return {
          tenant_id: "sup-1",
          adapters: [{ provider: "billz", display_name: "Billz POS", capabilities: ["health", "pull", "shadow_compare"] }],
          credentials: [
            {
              credential_id: "cred-billz-1",
              tenant_id: "sup-1",
              provider: "billz",
              account_label: "billz-main",
              auth_type: "bearer",
              secret_ref_masked: "env://BILLZ_****OKEN",
              base_url: "https://api-admin.billz.ai",
              status: "ACTIVE",
              created_by: "owner-1",
              created_at: "2026-11-17T01:00:00Z",
              updated_at: "2026-11-17T01:00:00Z",
            },
          ],
          latest_runs: [],
        };
      }
      if (path.startsWith("/v1/connectors/runs") && method === "GET") {
        return {
          tenant_id: "sup-1",
          count: 1,
          runs: [
            {
              run_id: "run-ro-1",
              tenant_id: "sup-1",
              provider: "billz",
              account_label: "billz-main",
              direction: "SHADOW_COMPARE",
              stream_name: "all",
              status: "COMPLETED",
              records_processed: 10,
              mismatch_count: 0,
              started_at: "2026-11-17T02:00:00Z",
            },
          ],
        };
      }
      return {};
    });

    render(<IntegrationsSettingsPage />);

    await waitFor(() => {
      expect(screen.getByTestId("integrations-read-only-banner")).toBeInTheDocument();
    });

    // Read-only data and read-only inspect action remain visible
    expect(screen.getByText("env://BILLZ_****OKEN")).toBeInTheDocument();
    expect(screen.getByText(/pxk_live/)).toBeInTheDocument();
    expect(screen.getByText("https://example.com/hook")).toBeInTheDocument();
    expect(screen.getByText("Configured: sftp.partner.uz:22 · EDI on · in/out/arch")).toBeInTheDocument();
    expect(screen.getByText("Configured: ATOMOS-AS2 ↔ RETAILER-AS2 · AS2 on")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Inspect Diff" })).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Download" })).toBeInTheDocument();

    // Zero mutating controls rendered across OIDC, Connector Spine, and all legacy sections
    expect(screen.queryByTestId("oidc-attach-mock")).toBeNull();
    for (const btnName of [
      "Save Credential",
      "Health Ping",
      "Pull Sync",
      "Shadow Compare",
      "Issue key",
      "Revoke",
      "Create",
      "Ping",
      "Deactivate",
      "Replay",
      "Save CoA",
      "Request export",
      "Save SFTP / EDI config",
      "Save AS2 config",
    ]) {
      expect(screen.queryByRole("button", { name: btnName })).toBeNull();
    }
    expect(screen.getByLabelText("AR (receivable)")).toBeDisabled();
    expect(screen.getByLabelText("Revenue")).toBeDisabled();
    expect(screen.getByLabelText("Bank / cash")).toBeDisabled();
  });

  it("renders /auditor in strict read-only safe mode for AUDITOR with expiry countdown, pack downloads, and ZERO mutating controls", async () => {
    const futureExpiry = new Date(Date.now() + 3 * 3600 * 1000).toISOString();
    mockSupplierTokenRef.current = makeJwt({
      sub: "auditor-ext-1",
      role: "ADMIN",
      supplier_id: "sup-1",
      supplier_role: "AUDITOR",
      auditor_expires_at: futureExpiry,
    });

    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path.startsWith("/v1/audit/verify") && method === "GET") {
        return {
          tenant_id: "sup-1",
          day_bucket: "2026-11-17",
          valid: true,
          event_count: 14,
          root_hash: "abcdef1234567890abcdef1234567890",
          anchored_root_hash: "abcdef1234567890abcdef1234567890",
          anchor_matched: true,
        };
      }
      if (path.startsWith("/v1/audit/anchors") && method === "GET") {
        return { tenant_id: "sup-1", count: 0, anchors: [] };
      }
      if (path.startsWith("/v1/finance/packs") && method === "GET") {
        return {
          tenant_id: "sup-1",
          count: 1,
          packs: [
            {
              pack_id: "pack-tax-1",
              tenant_id: "sup-1",
              pack_type: "TAX",
              period_id: "2026-11",
              from_date: "2026-11-01T00:00:00Z",
              to_date: "2026-11-30T23:59:59Z",
              format: "zip",
              status: "READY",
              watermark_text: "Prepared for auditor-ext-1 | Tenant sup-1 | Pack pack-tax-1",
              row_count: 1200,
              sha256_hex: "99887766554433221100ffeeddccbbaa",
              download_token: "dl-secret-tok",
              created_by: "owner-1",
              expires_at: futureExpiry,
              created_at: "2026-11-17T02:00:00Z",
            },
          ],
        };
      }
      if (path.startsWith("/v1/iam/auditor-grants") && method === "GET") {
        return { tenant_id: "sup-1", count: 0, grants: [] };
      }
      return {};
    });

    mockDownloadFetch.mockResolvedValue(
      new Response(new Uint8Array([1, 2, 3]), {
        status: 200,
        headers: {
          "Content-Type": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
          "Content-Disposition": 'attachment; filename="tax-pack-2026-11.xlsx"',
          "X-Atomos-Watermark": "Prepared for auditor-ext-1 | Tenant sup-1 | Pack pack-tax-1",
          "X-Atomos-Pack-Sha256": "99887766554433221100ffeeddccbbaa",
        },
      }),
    );

    render(<AuditorPortalPage />);

    await waitFor(() => {
      expect(screen.getByTestId("auditor-safe-mode-banner")).toBeInTheDocument();
    });
    expect(screen.getByTestId("auditor-safe-mode-badge")).toHaveTextContent("READ-ONLY SAFE MODE");
    expect(screen.getByTestId("auditor-expiry-countdown")).toHaveTextContent(/\d{2}:\d{2}:\d{2}/);
    expect(screen.getByTestId("audit-chain-status-card")).toHaveTextContent("CHAIN VALID");
    expect(screen.getByText("pack-tax-1")).toBeInTheDocument();
    expect(
      screen.getByText("Prepared for auditor-ext-1 | Tenant sup-1 | Pack pack-tax-1"),
    ).toBeInTheDocument();

    // Zero mutating actions rendered for AUDITOR
    expect(screen.queryByTestId("generate-export-pack-form")).toBeNull();
    expect(screen.queryByTestId("auditor-access-manager")).toBeNull();
    expect(screen.queryByRole("button", { name: /Generate Export Pack/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /Grant Temporary AUDITOR Access/i })).toBeNull();
    expect(screen.queryByRole("button", { name: /Revoke Grant/i })).toBeNull();

    // Multi-format download links (.zip, .xlsx, .csv, 1C .xml, .json) work cleanly
    for (const label of [".zip", ".xlsx", ".csv", "1C .xml", ".json"]) {
      expect(screen.getByRole("button", { name: label })).toBeInTheDocument();
    }
    fireEvent.click(screen.getByRole("button", { name: ".xlsx" }));
    await waitFor(() => {
      expect(mockDownloadFetch).toHaveBeenCalledWith(
        "/v1/finance/packs/pack-tax-1/download?format=xlsx&token=dl-secret-tok",
        { method: "GET" },
      );
    });
    await waitFor(() => {
      expect(screen.getByText(/tax-pack-2026-11\.xlsx/)).toBeInTheDocument();
    });
  });

  it("allows authorized finance roles on /auditor to generate ExportPacks, issue temporary AUDITOR bearer token, and revoke grants", async () => {
    const futureExpiry = new Date(Date.now() + 72 * 3600 * 1000).toISOString();
    mockSupplierTokenRef.current = makeJwt({
      sub: "fin-mgr-1",
      role: "ADMIN",
      supplier_id: "sup-1",
      supplier_role: "FINANCE_MANAGER",
    });

    mockRequest.mockImplementation(async (path: string, method: string, opts?: { body?: Record<string, unknown> }) => {
      if (path.startsWith("/v1/audit/verify") && method === "GET") {
        return {
          tenant_id: "sup-1",
          day_bucket: "2026-11-17",
          valid: true,
          event_count: 5,
          root_hash: "11223344556677889900",
          anchor_matched: false,
        };
      }
      if (path.startsWith("/v1/audit/anchors") && method === "GET") {
        return { tenant_id: "sup-1", count: 0, anchors: [] };
      }
      if (path.startsWith("/v1/finance/packs") && method === "GET") {
        return { tenant_id: "sup-1", count: 0, packs: [] };
      }
      if (path === "/v1/finance/packs" && method === "POST") {
        return {
          pack_id: "pack-audit-99",
          tenant_id: "sup-1",
          pack_type: opts?.body?.pack_type || "AUDIT",
          period_id: "2026-11",
          from_date: "2026-11-01T00:00:00Z",
          to_date: "2026-11-30T23:59:59Z",
          format: "zip",
          status: "READY",
          watermark_text: "Prepared for fin-mgr-1 | Tenant sup-1",
          row_count: 52500,
          sha256_hex: "aabbccddeeff00112233445566778899",
          created_by: "fin-mgr-1",
          expires_at: futureExpiry,
          created_at: "2026-11-17T03:00:00Z",
        };
      }
      if (path.startsWith("/v1/iam/auditor-grants") && method === "GET") {
        return {
          tenant_id: "sup-1",
          count: 1,
          grants: [
            {
              tenant_id: "sup-1",
              user_id: "ext-auditor-existing",
              role_name: "AUDITOR",
              granted_by: "owner-1",
              status: "ACTIVE",
              expires_at: futureExpiry,
              created_at: "2026-11-17T01:00:00Z",
            },
          ],
        };
      }
      if (path === "/v1/iam/auditor-grants" && method === "POST") {
        return {
          grant: {
            tenant_id: "sup-1",
            user_id: opts?.body?.user_id || "ext-kpmg-1",
            role_name: "AUDITOR",
            granted_by: "fin-mgr-1",
            status: "ACTIVE",
            expires_at: futureExpiry,
            created_at: "2026-11-17T03:10:00Z",
          },
          token: "eyJ.temp-auditor-bearer-token.sig",
          expires_at: futureExpiry,
        };
      }
      if (path === "/v1/iam/auditor-grants/ext-auditor-existing/revoke" && method === "POST") {
        return {
          tenant_id: "sup-1",
          user_id: "ext-auditor-existing",
          revoked: true,
        };
      }
      return {};
    });

    render(<AuditorPortalPage />);

    await waitFor(() => {
      expect(screen.getByTestId("generate-export-pack-form")).toBeInTheDocument();
    });
    expect(screen.queryByTestId("auditor-safe-mode-banner")).toBeNull();
    expect(screen.getByTestId("auditor-access-manager")).toBeInTheDocument();

    // Generate TAX pack
    const packForm = screen.getByTestId("generate-export-pack-form");
    fireEvent.change(within(packForm).getByLabelText("Pack Type"), { target: { value: "TAX" } });
    fireEvent.click(within(packForm).getByRole("button", { name: "Generate Export Pack" }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/finance/packs",
        "POST",
        expect.objectContaining({
          body: expect.objectContaining({ pack_type: "TAX", format: "zip" }),
        }),
      );
    });
    await waitFor(() => {
      expect(screen.getByText(/pack-audit-99/)).toBeInTheDocument();
    });

    // Grant Temporary AUDITOR Access & show copyable scoped bearer token
    fireEvent.change(screen.getByLabelText("Auditor User ID / Email"), {
      target: { value: "ext-kpmg-1" },
    });
    fireEvent.change(screen.getByLabelText("Engagement / Reason"), {
      target: { value: "FY26 Tax Review" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Grant Temporary AUDITOR Access" }));

    await waitFor(() => {
      expect(screen.getByTestId("auditor-bearer-token-notice")).toBeInTheDocument();
    });
    expect(screen.getByTestId("auditor-bearer-token-value")).toHaveTextContent(
      "eyJ.temp-auditor-bearer-token.sig",
    );

    // Revoke existing grant
    fireEvent.click(screen.getByRole("button", { name: "Revoke Grant" }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/iam/auditor-grants/ext-auditor-existing/revoke",
        "POST",
        expect.any(Object),
      );
    });
  });

  it("registers /auditor in nav, translates all Wave 3C i18n keys across en/ru/uz, wraps search params in Suspense, and has zero @pegasusx/ui imports", () => {
    expect(allNavHrefs(NAV)).toContain("/auditor");
    expect(allNavHrefs(NAV)).toContain("/settings/integrations");

    const requiredKeys = [
      "portal.nav.auditor",
      "supplier_portal.settings.integrations.text.xlsx",
      "supplier_portal.settings.integrations.connector_spine.section_title",
      "supplier_portal.settings.integrations.connector_spine.export_unlimited_label",
      "supplier_portal.settings.integrations.connector_spine.shadow_title",
      "supplier_portal.auditor.title",
      "supplier_portal.auditor.safe_mode_badge",
      "supplier_portal.auditor.chain_card_title",
      "supplier_portal.auditor.packs_table_title",
      "supplier_portal.auditor.generate_pack_title",
      "supplier_portal.auditor.grants_manager_title",
    ];
    for (const key of requiredKeys) {
      expect(messageCatalogs.en[key], key).toBeTruthy();
      expect(messageCatalogs.ru[key], key).toBeTruthy();
      expect(messageCatalogs.uz[key], key).toBeTruthy();
    }

    const wave3cFiles = [
      "lib/connectors-auditor-api.ts",
      "lib/nav.ts",
      "app/(portal)/settings/integrations/page.tsx",
      "app/(portal)/auditor/page.tsx",
    ];
    for (const rel of wave3cFiles) {
      const fullPath = join(portalRoot, rel);
      expect(existsSync(fullPath), rel).toBe(true);
      const src = readFileSync(fullPath, "utf8");
      expect(src).not.toContain("@pegasusx/ui");
    }

    for (const pageRel of [
      "app/(portal)/settings/integrations/page.tsx",
      "app/(portal)/auditor/page.tsx",
    ]) {
      const src = readFileSync(join(portalRoot, pageRel), "utf8");
      expect(src).toContain("<Suspense");
    }
  });
});
