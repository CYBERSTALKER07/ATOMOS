import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { ApiClient } from "@pegasusx/api-core";
import { createTranslator, messageCatalogs } from "@pegasusx/i18n";

const { mockRequest, mockSearchParamsRef, mockGetDemandToday, mockGetDemandHistory, mockGetDemandAccuracy } =
  vi.hoisted(() => ({
    mockRequest: vi.fn(),
    mockSearchParamsRef: { current: new URLSearchParams() },
    mockGetDemandToday: vi.fn(),
    mockGetDemandHistory: vi.fn(),
    mockGetDemandAccuracy: vi.fn(),
  }));

vi.mock("@/lib/api", () => ({
  createSupplierApi: () => ({
    request: mockRequest,
    getSupplierDemandToday: mockGetDemandToday,
    getSupplierDemandHistory: mockGetDemandHistory,
    getSupplierDemandAccuracy: mockGetDemandAccuracy,
  }),
}));

vi.mock("@/lib/i18n", () => ({
  usePortalT: () => createTranslator("en"),
}));

vi.mock("@/lib/use-supplier-session-reconcile", () => ({
  useSupplierSessionReconcile: () => {},
}));

vi.mock("next/navigation", () => ({
  useSearchParams: () => mockSearchParamsRef.current,
  useRouter: () => ({ push: vi.fn(), replace: vi.fn() }),
}));

vi.mock("next/link", () => ({
  default: ({ children, href, className }: { children: ReactNode; href: string; className?: string }) => (
    <a href={href} className={className}>
      {children}
    </a>
  ),
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

vi.mock("@/components/StatusBadge", () => ({
  default: ({ state, label }: { state: string; label?: string }) => <span>{label || state}</span>,
}));

vi.mock("@/components/Icon", () => ({
  default: ({ name }: { name: string }) => <span data-icon={name} />,
}));

vi.mock("recharts", () => ({
  ResponsiveContainer: ({ children }: { children: ReactNode }) => (
    <div data-testid="recharts-container">{children}</div>
  ),
  LineChart: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  CartesianGrid: () => null,
  XAxis: () => null,
  YAxis: () => null,
  Tooltip: () => null,
  Legend: () => null,
  Line: () => null,
}));

vi.mock("@/components/FleetLiveMapPanel", () => ({
  default: () => <div data-testid="fleet-map-mock" />,
}));

vi.mock("@/components/PlanningOutcomesPanel", () => ({
  default: () => <div data-testid="planning-outcomes-mock" />,
}));

vi.mock("@/components/NetworkPulsePanel", () => ({
  default: () => <div data-testid="network-pulse-mock" />,
}));

vi.mock("@/app/(portal)/dashboard/use-dashboard-data", () => ({
  useDashboardData: () => ({
    metrics: {
      ordersByStatus: { COMPLETED: 10 },
      manifestsByState: { DRAFT: 1, LOADING: 0, DISPATCHED: 2 },
      revenueToday: 500000,
      completedToday: 8,
      attemptedToday: 10,
      activeDrivers: 3,
      totalDrivers: 5,
      deliveryCompletionRate: 80,
      retailersOrderedToday: 4,
      totalRetailers: 10,
      fleetVuUsed: 100,
      fleetVuTotal: 200,
      fleetVuAvailable: true,
      asOf: "2026-11-17T04:00:00Z",
    },
    recentManifests: [],
    recentEvents: [],
    isPaymentConfigured: true,
    loading: false,
    error: null,
  }),
}));

vi.mock("@/app/(portal)/dashboard/use-dashboard-history", () => ({
  useDashboardHistory: () => ({
    range: "7d",
    setRange: vi.fn(),
    revenue: { source: "unavailable", points: [] },
    velocity: { source: "unavailable", points: [] },
    revenueDeltaPct: null,
    loading: false,
  }),
}));

import {
  createIntelRoiApi,
  formatFinanceMinor,
  formatMilliUnits,
  formatRateBps,
  formatRoiMultipleBps,
  formatSignedBps,
  isApprovalPendingResponse,
  parseMajorToSafeMinor,
} from "@/lib/intel-roi-api";
import { allNavHrefs } from "@/lib/nav";
import StatusChip from "@/components/StatusChip";
import DemandAnalyticsPage from "@/app/(portal)/analytics/demand/page";
import CreditScoresPage from "@/app/(portal)/credit/scores/page";
import ProposalsInboxPage from "@/app/(portal)/proposals/page";
import RoiDashboardPage from "@/app/(portal)/roi/page";
import DashboardPage from "@/app/(portal)/dashboard/page";

const here = dirname(fileURLToPath(import.meta.url));
const portalRoot = join(here, "../..");

describe("Wave 3B Intel, Proposals & ROI Supplier Portal (M1.5-M1.8)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockSearchParamsRef.current = new URLSearchParams();
    mockGetDemandToday.mockRejectedValue(new Error("skip_legacy"));
    mockGetDemandHistory.mockRejectedValue(new Error("skip_legacy"));
    mockGetDemandAccuracy.mockRejectedValue(new Error("skip_legacy"));
  });

  afterEach(() => {
    cleanup();
  });

  it("formats milli quantities, ROI multiple bps, and minor money in pure BigInt without floats", () => {
    expect(formatMilliUnits(12500, "units")).toBe("12.5 units");
    expect(formatMilliUnits(12000)).toBe("12");
    expect(formatMilliUnits(-2250)).toBe("-2.25");
    expect(formatRoiMultipleBps(34500)).toBe("3.45x");
    expect(formatRoiMultipleBps(0)).toBe("0.00x");
    expect(formatSignedBps(1250)).toBe("+12.50%");
    expect(formatSignedBps(-400)).toBe("-4%");
    expect(formatFinanceMinor(BigInt("900719925474099300"), "UZS")).toBe("9,007,199,254,740,993 UZS");
    expect(formatRateBps(850)).toBe("8.50%");
    expect(parseMajorToSafeMinor("1200000.25")).toBe(120000025);
  });

  it("wires all Wave 3B HTTP endpoints, CSV report, 202 approval gate, and Idempotency-Key via ApiClient", async () => {
    const calls: Array<{ url: string; method: string; body?: string; idempotencyKey?: string | null }> = [];
    const fetchSpy = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const headers = new Headers(init?.headers);
      calls.push({
        url,
        method: init?.method || "GET",
        body: typeof init?.body === "string" ? init.body : undefined,
        idempotencyKey: headers.get("Idempotency-Key"),
      });
      if (url.includes("format=csv")) {
        return new Response("category,savings_minor\nSTOCKOUT_AVOIDED,5000000\n", {
          status: 200,
          headers: { "Content-Type": "text/csv" },
        });
      }
      if (url.includes("/v1/credit/limits/ret-1")) {
        return new Response(
          JSON.stringify({
            status: "APPROVAL_REQUIRED",
            approval_required: true,
            approval_request_id: "apr-limit-1",
            policy_id: "pol-1",
            policy_version: 2,
            payload_hash: "sha256-abc",
            old_limit_minor: 300_000_00,
            new_limit_minor: 500_000_00,
          }),
          { status: 202, headers: { "Content-Type": "application/json" } },
        );
      }
      return new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });

    const client = new ApiClient({
      baseUrl: "https://api.pegasusx.test",
      getAuthToken: () => "tok-wave3b",
      fetchImpl: fetchSpy as typeof fetch,
    });
    const api = createIntelRoiApi(client);

    await api.getPlanningForecast({ sku: "SKU-1", horizon_days: 14 });
    await api.runPlanningForecast({ location_id: "wh-1", horizon_days: 14, active_promo: true });
    await api.approveForecastExplanation("exp-1", { decision: "APPROVE", adjustment_bps: 500 });
    await api.approveReplenishmentSuggestion("ret-1:SKU-1", { qty: 10, mode: "order" });
    await api.getCreditPortfolio({ status: "ACTIVE", risk_tier: "MEDIUM" });
    await api.getRetailerCreditScore("ret-1", { scope: "BOTH" });
    const limitRes = await api.updateCreditLimit("ret-1", { new_limit_minor: 500_000_00, reason: "expand" });
    expect(limitRes.status).toBe("APPROVAL_REQUIRED");
    expect(limitRes.approval_required).toBe(true);
    expect(isApprovalPendingResponse(limitRes)).toBe(true);
    expect(limitRes.approval_request_id).toBe("apr-limit-1");
    const limitCall = calls.find((c) => c.url.includes("/v1/credit/limits/ret-1"));
    expect(JSON.parse(limitCall?.body || "{}")).toMatchObject({
      new_limit_minor: 500_000_00,
      reason: "expand",
    });

    await api.listProposals({ agent_id: "REORDER_AGENT", status: "PENDING" });
    await api.createProposal({
      proposal_type: "REPLENISH_ORDER",
      target_object_type: "Sku",
      target_object_id: "SKU-1",
      title: "Replenish SKU-1",
      payload: { qty: 10 },
    });
    await api.runProposalAgent("REORDER_AGENT");
    await api.getProposal("prop-1");
    await api.approveProposal("prop-1", { payload_hash: "h1", edited_payload: { qty: 25 } });
    await api.rejectProposal("prop-2", { comment: "no" });
    await api.executeProposal("prop-1", { payload_hash: "h1" });

    await api.getRoiBaseline();
    await api.captureRoiBaseline({ window_days: 90, baseline_bad_debt_minor: 100000 });
    await api.getRoiSummary({ days: 90 });
    await api.listRoiEvents({ category: "STOCKOUT_AVOIDED" });
    await api.recordRoiEvent({
      category: "STOCKOUT_AVOIDED",
      savings_minor: 500_000,
      methodology: "Matched holdout",
    });
    await api.getRoiReportJson({ days: 90 });
    const csv = await api.downloadRoiReportCsv({ days: 90 });
    expect(csv).toContain("STOCKOUT_AVOIDED,5000000");

    const postCalls = calls.filter((c) => c.method === "POST");
    expect(postCalls.length).toBeGreaterThanOrEqual(11);
    for (const c of postCalls) {
      expect(c.idempotencyKey, `Missing Idempotency-Key on ${c.url}`).toBeTruthy();
    }
    expect(calls.some((c) => c.url.includes("/v1/planning/forecast?sku=SKU-1&horizon_days=14"))).toBe(true);
    expect(calls.some((c) => c.url.includes("/v1/proposals/agents/REORDER_AGENT/run"))).toBe(true);
    expect(calls.some((c) => c.url.includes("/v1/roi/report?days=90&format=csv"))).toBe(true);
  });

  it("enforces Wave 3B nav entries, Suspense wrapping, i18n parity, and zero @pegasusx/ui imports", () => {
    const hrefs = allNavHrefs();
    expect(hrefs).toContain("/analytics/demand");
    expect(hrefs).toContain("/credit/scores");
    expect(hrefs).toContain("/proposals");
    expect(hrefs).toContain("/roi");

    const pagePaths = [
      "lib/intel-roi-api.ts",
      "app/(portal)/analytics/demand/page.tsx",
      "app/(portal)/credit/scores/page.tsx",
      "app/(portal)/proposals/page.tsx",
      "app/(portal)/roi/page.tsx",
      "app/(portal)/dashboard/page.tsx",
    ];
    for (const rel of pagePaths) {
      const full = join(portalRoot, rel);
      expect(existsSync(full), rel).toBe(true);
      const src = readFileSync(full, "utf8");
      expect(src, `${rel} must never import @pegasusx/ui`).not.toContain("@pegasusx/ui");
      if (rel !== "lib/intel-roi-api.ts" && rel !== "app/(portal)/dashboard/page.tsx") {
        expect(src, `${rel} must wrap useSearchParams in Suspense`).toContain("Suspense");
      }
    }

    const requiredKeys = [
      "portal.nav.credit_scores",
      "portal.nav.proposals",
      "portal.nav.roi",
      "supplier_portal.forecast_v2.title",
      "supplier_portal.credit_scores.title",
      "supplier_portal.proposals.title",
      "supplier_portal.roi.title",
      "retailer_desktop.suggested_orders.title",
    ];
    for (const k of requiredKeys) {
      expect(messageCatalogs.en[k], `en missing ${k}`).toBeTruthy();
      expect(messageCatalogs.ru[k], `ru missing ${k}`).toBeTruthy();
      expect(messageCatalogs.uz[k], `uz missing ${k}`).toBeTruthy();
    }
  });

  it("renders DemandAnalyticsPage Forecast v2, triggers Run Forecast v2, and approves adjustment", async () => {
    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path.startsWith("/v1/planning/forecast/run") && method === "POST") {
        return {
          run: {
            run_id: "run-new-2",
            entity_id: "sup-1",
            horizon_days: 14,
            status: "COMPLETED",
            champion_model: "HOLT_WINTERS",
            challenger_model: "CALENDAR_GLOBAL_V2",
            winner_model: "CALENDAR_GLOBAL_V2",
            champion_wape_bps: 1850,
            challenger_wape_bps: 1120,
            wape_improvement_bps: 730,
            feature_source: "calendar+promo",
            regressors: [
              { name: "Navruz", category: "HOLIDAY", multiplier_bps: 1450, description: "Spring holiday lift" },
            ],
          },
          forecasts: [
            {
              run_id: "run-new-2",
              forecast_date: "2026-11-18",
              location_id: "wh-1",
              sku: "SKU-100",
              model: "CALENDAR_GLOBAL_V2",
              is_winner: true,
              demand_class: "SMOOTH",
              p10_milli: 8000,
              p50_milli: 12500,
              p90_milli: 18000,
              baseline_p50_milli: 10000,
              wape_bps: 1120,
              bias_bps: 150,
              confidence_bps: 8800,
            },
          ],
          explanations: [],
        };
      }
      if (path.includes("/explanations/exp-1/approve") && method === "POST") {
        return {
          explanation: {
            explanation_id: "exp-1",
            run_id: "run-1",
            location_id: "wh-1",
            sku: "SKU-100",
            source: "GEMINI",
            model_name: "gemini-2.5-flash",
            summary_text: "Navruz uplift +15% backed by 2025 POS surge.",
            adjustment_bps: 1500,
            status: "APPROVED",
            baseline_p50_milli: 10000,
            adjusted_p50_milli: 11500,
            forecast_value_add_bps: 420,
            approved_by: "owner-1",
          },
        };
      }
      return {
        run: {
          run_id: "run-1",
          entity_id: "sup-1",
          horizon_days: 14,
          status: "COMPLETED",
          champion_model: "SES",
          challenger_model: "CALENDAR_GLOBAL_V2",
          winner_model: "CALENDAR_GLOBAL_V2",
          champion_wape_bps: 1900,
          challenger_wape_bps: 1200,
          wape_improvement_bps: 700,
          regressors: [
            { name: "Payday", category: "CALENDAR", multiplier_bps: 900, description: "1st/15th payday surge" },
          ],
        },
        forecasts: [
          {
            run_id: "run-1",
            forecast_date: "2026-11-18",
            location_id: "wh-1",
            sku: "SKU-100",
            model: "CALENDAR_GLOBAL_V2",
            is_winner: true,
            demand_class: "SMOOTH",
            p10_milli: 8000,
            p50_milli: 12500,
            p90_milli: 18000,
            baseline_p50_milli: 10000,
            wape_bps: 1200,
            bias_bps: 100,
            confidence_bps: 8600,
          },
        ],
        explanations: [
          {
            explanation_id: "exp-1",
            run_id: "run-1",
            location_id: "wh-1",
            sku: "SKU-100",
            source: "GEMINI",
            model_name: "gemini-2.5-flash",
            summary_text: "Navruz uplift +15% backed by 2025 POS surge.",
            evidence_refs: [
              { ref_type: "Signal", ref_id: "sig-9", label: "POS Spike", value_text: "+18%" },
            ],
            adjustment_bps: 1500,
            status: "PENDING",
            baseline_p50_milli: 10000,
            adjusted_p50_milli: 11500,
            forecast_value_add_bps: 420,
          },
        ],
      };
    });

    render(<DemandAnalyticsPage />);

    await waitFor(() => {
      expect(screen.getByText("19%")).toBeInTheDocument();
      expect(screen.getAllByText("12%").length).toBeGreaterThan(0);
      expect(screen.getByText("Payday")).toBeInTheDocument();
      expect(screen.getByText(/Navruz uplift \+15%/)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Approve adjustment" }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/planning/forecast/explanations/exp-1/approve",
        "POST",
        expect.objectContaining({
          body: expect.objectContaining({ decision: "APPROVE", adjustment_bps: 1500 }),
        }),
      );
      expect(screen.getByText(/Explanation exp-1 updated \(APPROVED\)/)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Run forecast v2" }));
    await waitFor(() => {
      expect(screen.getByText(/Forecast run run-new-2 completed/)).toBeInTheDocument();
    });
  });

  it("renders CreditScoresPage with Relationship/Network scores, Reason Codes, and 202 CEL Limit Gate", async () => {
    mockRequest.mockImplementation(async (path: string, method: string, options?: { body?: unknown }) => {
      if (path.startsWith("/v1/credit/portfolio")) {
        return {
          supplier_id: "sup-1",
          summary: {
            total_retailers: 2,
            total_exposure_minor: 200_000_00,
            total_limit_minor: 500_000_00,
            utilization_bps: 4000,
            frozen_count: 0,
            high_risk_count: 0,
            avg_score: 82,
            consent_verified_count: 1,
          },
          items: [
            {
              retailer_id: "ret-1",
              retailer_name: "Baraka Market",
              credit_limit_minor: 300_000_00,
              current_balance_minor: 100_000_00,
              reserved_minor: 0,
              available_credit_minor: 200_000_00,
              utilization_bps: 3333,
              risk_score: 86,
              pd_bps: 240,
              risk_tier: "LOW",
              status: "ACTIVE",
              prepay_only_locked: false,
              consent_verified: true,
              suggested_limit_minor: 400_000_00,
            },
          ],
        };
      }
      if (path.startsWith("/v1/credit/retailers/ret-1/score")) {
        return {
          retailer_id: "ret-1",
          relationship_score: {
            score_id: "sc-rel-1",
            scope: "RELATIONSHIP",
            score: 86,
            pd_bps: 240,
            risk_tier: "LOW",
            suggested_limit_minor: 400_000_00,
            current_limit_minor: 300_000_00,
            consent_verified: true,
            disputes_isolated_count: 0,
            model_version: "credit_v2",
            scorer_source: "ML_CALIBRATED",
            reason_codes: [
              {
                code: "LOW_DPD",
                label: "Zero overdue invoices (90d)",
                impact_bps: 1200,
                detail: "All 12 invoices paid before due date",
                source: "AR_LEDGER",
              },
            ],
          },
          network_score: {
            score_id: "sc-net-1",
            scope: "NETWORK",
            score: 81,
            pd_bps: 310,
            risk_tier: "LOW",
            suggested_limit_minor: 380_000_00,
            current_limit_minor: 300_000_00,
            consent_verified: true,
            disputes_isolated_count: 1,
            model_version: "credit_v2",
            scorer_source: "NETWORK_CONSENTED",
          },
          history: [],
        };
      }
      if (path.startsWith("/v1/credit/limits/ret-1") && method === "POST") {
        expect(options?.body).toMatchObject({
          new_limit_minor: 500_000_00,
          reason: "Strong repayment history",
        });
        return {
          status: "APPROVAL_REQUIRED",
          approval_required: true,
          approval_request_id: "apr-cel-77",
          policy_id: "pol-credit",
          policy_version: 1,
          payload_hash: "hash-xyz",
          old_limit_minor: 300_000_00,
          new_limit_minor: 500_000_00,
        };
      }
      return {};
    });

    render(<CreditScoresPage />);

    await waitFor(() => {
      expect(screen.getByText("Baraka Market")).toBeInTheDocument();
      expect(screen.getByText("LOW_DPD")).toBeInTheDocument();
      expect(screen.getByText(/Isolated disputes: 1/)).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("New credit limit"), { target: { value: "500000" } });
    fireEvent.change(screen.getByLabelText("Adjustment reason"), {
      target: { value: "Strong repayment history" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Submit limit adjustment" }));

    await waitFor(() => {
      expect(screen.getByText(/CEL approval required \(apr-cel-77\)/)).toBeInTheDocument();
      const links = screen.getAllByRole("link", { name: "Open in Approvals Inbox" });
      expect(links.some((link) => link.getAttribute("href") === "/approvals?requestId=apr-cel-77")).toBe(true);
    });

    const tierSelect = screen.getByLabelText("Risk tier filter") as HTMLSelectElement;
    const tierOptions = Array.from(tierSelect.options).map((o) => o.value);
    expect(tierOptions).toEqual(["ALL", "LOW", "MEDIUM", "HIGH", "BLOCK"]);
    fireEvent.change(tierSelect, { target: { value: "BLOCK" } });
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        expect.stringContaining("/v1/credit/portfolio?risk_tier=BLOCK"),
        "GET",
      );
    });
  });

  it("styles BLOCK risk tier with danger-strong tone in StatusChip", () => {
    render(<StatusChip status="BLOCK" />);
    const chip = screen.getByText("BLOCK");
    expect(chip).toHaveStyle({ background: "var(--desk-danger)" });
  });

  it("renders ProposalsInboxPage, runs L1 agent, edits/approves, and executes proposal", async () => {
    let statusState = "PENDING";
    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path === "/v1/proposals/agents/REORDER_AGENT/run" && method === "POST") {
        return { agent_id: "REORDER_AGENT", proposals: [], created_count: 1 };
      }
      if (path === "/v1/proposals/prop-1/approve" && method === "POST") {
        statusState = "APPROVED";
        return {
          proposal: {
            proposal_id: "prop-1",
            agent_id: "REORDER_AGENT",
            proposal_type: "REPLENISH_ORDER",
            target_object_type: "Sku",
            target_object_id: "SKU-1",
            title: "Replenish SKU-1",
            summary: "Low stock cover (1.5d)",
            rationale: "7d velocity 20/day",
            payload: { qty: 120 },
            payload_hash: "hash-2",
            evidence_refs: [{ type: "StockLot", id: "lot-1", label: "Remaining 30 units" }],
            confidence_bps: 8800,
            estimated_impact_minor: 450_000_00,
            currency: "UZS",
            roi_category: "STOCKOUT_AVOIDED",
            status: "APPROVED",
          },
        };
      }
      if (path === "/v1/proposals/prop-1/execute" && method === "POST") {
        statusState = "EXECUTED";
        return {
          proposal: {
            proposal_id: "prop-1",
            agent_id: "REORDER_AGENT",
            proposal_type: "REPLENISH_ORDER",
            target_object_type: "Sku",
            target_object_id: "SKU-1",
            title: "Replenish SKU-1",
            summary: "Low stock cover (1.5d)",
            rationale: "7d velocity 20/day",
            payload: { qty: 120 },
            payload_hash: "hash-2",
            confidence_bps: 8800,
            estimated_impact_minor: 450_000_00,
            currency: "UZS",
            roi_category: "STOCKOUT_AVOIDED",
            status: "EXECUTED",
            roi_event_id: "roi-ev-1",
          },
          execution_result: { order_id: "ord-draft-9" },
          roi_event: {
            event_id: "roi-ev-1",
            category: "STOCKOUT_AVOIDED",
            savings_minor: 450_000_00,
            currency: "UZS",
            methodology: "Cover uplift",
            confidence_bps: 8800,
            is_estimate: false,
            holdout_control: false,
          },
        };
      }
      if (path === "/v1/proposals/prop-1") {
        return {
          proposal: {
            proposal_id: "prop-1",
            agent_id: "REORDER_AGENT",
            proposal_type: "REPLENISH_ORDER",
            target_object_type: "Sku",
            target_object_id: "SKU-1",
            title: "Replenish SKU-1",
            summary: "Low stock cover (1.5d)",
            rationale: "7d velocity 20/day",
            payload: { qty: 100 },
            payload_hash: "hash-1",
            evidence_refs: [{ type: "StockLot", id: "lot-1", label: "Remaining 30 units" }],
            confidence_bps: 8800,
            estimated_impact_minor: 450_000_00,
            currency: "UZS",
            roi_category: "STOCKOUT_AVOIDED",
            status: statusState,
          },
        };
      }
      return {
        proposals: [
          {
            proposal_id: "prop-1",
            agent_id: "REORDER_AGENT",
            proposal_type: "REPLENISH_ORDER",
            target_object_type: "Sku",
            target_object_id: "SKU-1",
            title: "Replenish SKU-1",
            summary: "Low stock cover (1.5d)",
            rationale: "7d velocity 20/day",
            payload: { qty: 100 },
            payload_hash: "hash-1",
            evidence_refs: [{ type: "StockLot", id: "lot-1", label: "Remaining 30 units" }],
            confidence_bps: 8800,
            estimated_impact_minor: 450_000_00,
            currency: "UZS",
            roi_category: "STOCKOUT_AVOIDED",
            status: statusState,
          },
        ],
        count: 1,
      };
    });

    render(<ProposalsInboxPage />);

    await waitFor(() => {
      expect(screen.getByText("Replenish SKU-1")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Run Reorder Agent" }));
    await waitFor(() => {
      expect(screen.getByText(/REORDER_AGENT run completed \(1 proposal\(s\) created\)/)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Inspect" }));
    await waitFor(() => {
      expect(screen.getByRole("dialog")).toBeInTheDocument();
      expect(screen.getByText(/StockLot:lot-1/)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Edit payload JSON" }));
    fireEvent.change(screen.getByLabelText("Edit proposal payload JSON"), {
      target: { value: '{"qty": 120}' },
    });
    fireEvent.click(screen.getByRole("button", { name: "Edit & approve" }));

    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/proposals/prop-1/approve",
        "POST",
        expect.objectContaining({
          body: expect.objectContaining({ edited_payload: { qty: 120 }, payload_hash: "hash-1" }),
        }),
      );
      expect(screen.getByText(/Proposal prop-1 approved/)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Execute proposal" }));
    await waitFor(() => {
      expect(screen.getByText(/Proposal prop-1 executed/)).toBeInTheDocument();
      expect(screen.getByText(/ROI event recorded: roi-ev-1/)).toBeInTheDocument();
    });
  });

  it("renders RoiDashboardPage and Owner Home ROI Banner on DashboardPage", async () => {
    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path.startsWith("/v1/roi/summary")) {
        return {
          currency: "UZS",
          window_days: 90,
          total_savings_minor: 15_000_000_00,
          realized_savings_minor: 13_000_000_00,
          estimated_savings_minor: 2_000_000_00,
          holdout_savings_minor: 1_200_000_00,
          platform_fees_minor: 3_000_000_00,
          net_savings_minor: 12_000_000_00,
          roi_multiple_bps: 50000,
          categories: [
            {
              category: "STOCKOUT_AVOIDED",
              event_count: 8,
              total_savings_minor: 13_000_000_00,
              realized_minor: 13_000_000_00,
              estimated_minor: 0,
              holdout_savings_minor: 1_200_000_00,
              avg_confidence_bps: 8600,
              methodology: "Treatment vs holdout margin capture",
            },
            {
              category: "LABOUR_SAVED",
              event_count: 2,
              total_savings_minor: 2_000_000_00,
              realized_minor: 0,
              estimated_minor: 2_000_000_00,
              holdout_savings_minor: 0,
              avg_confidence_bps: 7000,
              methodology: "Survey hourly wage benchmark",
            },
          ],
          holdout_stats: {
            treatment_event_count: 8,
            treatment_savings_minor: 13_000_000_00,
            holdout_event_count: 3,
            holdout_savings_minor: 1_200_000_00,
            incremental_lift_minor: 11_800_000_00,
            lift_bps: 4200,
          },
        };
      }
      if (path.startsWith("/v1/roi/baseline") && method === "GET") {
        return { baseline: null };
      }
      if (path.startsWith("/v1/roi/baseline") && method === "POST") {
        return {
          baseline: {
            baseline_id: "base-90d",
            window_days: 90,
            baseline_ar_minor: 0,
            baseline_bad_debt_minor: 500_000_00,
            baseline_expired_minor: 0,
            baseline_stockout_minor: 0,
            baseline_logistics_minor: 0,
            baseline_labour_hours: 40,
            survey_hourly_wage_minor: 25_000_00,
            currency: "UZS",
          },
        };
      }
      if (path.startsWith("/v1/roi/events")) {
        return { events: [], count: 0 };
      }
      if (path.includes("/v1/roi/report") && path.includes("format=csv")) {
        return "category,savings_minor\nSTOCKOUT_AVOIDED,1300000000\n";
      }
      return {};
    });

    render(<RoiDashboardPage />);

    await waitFor(() => {
      expect(screen.getAllByText("15,000,000 UZS").length).toBeGreaterThan(0);
      expect(screen.getAllByText("5.00x").length).toBeGreaterThan(0);
      expect(screen.getByText("+42%")).toBeInTheDocument();
      expect(screen.getAllByText("LABOUR_SAVED").length).toBeGreaterThan(0);
    });

    fireEvent.change(screen.getByLabelText("Bad debt baseline"), { target: { value: "500000" } });
    fireEvent.change(screen.getByLabelText("Baseline labour hours"), { target: { value: "40" } });
    fireEvent.change(screen.getByLabelText("Survey hourly wage"), { target: { value: "25000" } });
    fireEvent.click(screen.getByRole("button", { name: "Save baseline" }));

    await waitFor(() => {
      expect(screen.getByText(/ROI baseline base-90d captured/)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Download QBR CSV" }));
    await waitFor(() => {
      expect(screen.getByText(/STOCKOUT_AVOIDED,1300000000/)).toBeInTheDocument();
    });

    cleanup();

    render(<DashboardPage />);
    await waitFor(() => {
      const banner = screen.getByTestId("owner-roi-banner");
      expect(banner).toHaveTextContent("15,000,000 UZS");
      expect(banner).toHaveTextContent("5.00x");
    });
  });
});
