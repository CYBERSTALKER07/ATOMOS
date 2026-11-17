import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { ApiClient, ApiError } from "@pegasusx/api-core";
import { createTranslator, messageCatalogs } from "@pegasusx/i18n";

const { mockRequest, mockSearchParamsRef, mockSupplierTokenRef } = vi.hoisted(() => ({
  mockRequest: vi.fn(),
  mockSearchParamsRef: { current: new URLSearchParams() },
  mockSupplierTokenRef: { current: "" },
}));

vi.mock("@/lib/api", () => ({
  createSupplierApi: () => ({
    request: mockRequest,
  }),
}));

vi.mock("@/lib/auth", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/auth")>();
  return {
    ...actual,
    getSupplierToken: () => mockSupplierTokenRef.current,
    readTokenFromCookie: () => mockSupplierTokenRef.current,
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
  default: ({ state, label }: { state: string; label?: string }) => (
    <span>{label || state}</span>
  ),
}));

vi.mock("@/components/Icon", () => ({
  default: ({ name }: { name: string }) => <span data-icon={name} />,
}));

import {
  createP2bApi,
  extractP2bError,
  formatMilliCelsius,
  parseAnomalyEvidence,
  parseCommittedProposalIds,
  parsePercentToBps,
  parseWhatIfBreakdown,
  parseWhatIfShocks,
} from "@/lib/p2b-api";
import { NAV, allNavHrefs } from "@/lib/nav";
import AnomalyCataloguePage from "@/app/(portal)/control-tower/anomalies/page";
import WhatIfSimulatorPage from "@/app/(portal)/planning/what-if/page";
import HaccpCompliancePage from "@/app/(portal)/compliance/haccp/page";
import ProposalsInboxPage from "@/app/(portal)/proposals/page";
import SupplierGovernancePage from "@/app/(portal)/governance/page";

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

describe("Wave 4B Anomaly Catalogue, What-If Simulator, HACCP, L2 Soak & Custom Roles / SCIM v2", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockSearchParamsRef.current = new URLSearchParams();
    mockSupplierTokenRef.current = makeJwt({
      sub: "owner-1",
      role: "ADMIN",
      supplier_id: "sup-1",
      supplier_role: "OWNER",
    });
  });

  afterEach(() => {
    cleanup();
  });

  it("calls all Wave 4B HTTP & SCIM v2 routes via ApiClient and computes pure BigInt helpers", async () => {
    expect(formatMilliCelsius(-18500)).toBe("-18.5°C");
    expect(formatMilliCelsius(2300)).toBe("2.3°C");
    expect(formatMilliCelsius(0)).toBe("0.0°C");
    expect(parsePercentToBps("15.5")).toBe(1550);
    expect(parsePercentToBps("-10")).toBe(-1000);
    expect(parsePercentToBps("bad%")).toBeNull();
    expect(
      parseAnomalyEvidence({
        evidence_json: '{"shared_tin":"301234567","order_count":4}',
      }),
    ).toEqual({ shared_tin: "301234567", order_count: 4 });
    expect(
      parseWhatIfShocks({
        shocks_json:
          '{"demand_delta_bps":1500,"factory_downtime_hours":12,"fx_rate_shock_bps":500,"credit_limit_delta_bps":-1000,"unit_price_delta_bps":0}',
      })?.demand_delta_bps,
    ).toBe(1500);
    const breakdownRows = parseWhatIfBreakdown({
      breakdown_json:
        '{"currency":"UZS","scenario_applied":true,"patch_count":2,"demand":{"forecast_delta_minor":2500000,"shocked_revenue_minor":12500000,"sku_count":4},"replenishment":{"cost_delta_minor":400000,"shocked_cost_minor":3400000,"affected_skus":["SKU-1"]},"dispatch":{"dispatch_cost_delta_minor":120000,"shocked_cost_minor":900000,"route_count":2,"order_count":8},"credit":{"credit_exposure_delta_minor":-300000,"risk_cost_delta_minor":50000,"over_limit_retailers":1},"pnl":{"net_pnl_delta_minor":1980000,"cogs_delta_minor":400000}}',
    });
    expect(breakdownRows).toEqual([
      {
        stage: "DEMAND",
        label: "Demand & Forecast",
        delta_minor: 2500000,
        cost_minor: 12500000,
        detail: "SKUs: 4",
      },
      {
        stage: "REPLENISHMENT",
        label: "Replenishment & FX",
        delta_minor: 400000,
        cost_minor: 3400000,
        detail: "Affected SKUs: 1",
      },
      {
        stage: "DISPATCH",
        label: "Fleet & Dispatch",
        delta_minor: 120000,
        cost_minor: 900000,
        detail: "Routes: 2 · Orders: 8",
      },
      {
        stage: "CREDIT",
        label: "AR & Credit Risk",
        delta_minor: -300000,
        cost_minor: 50000,
        detail: "Over-limit retailers: 1",
      },
      {
        stage: "PNL",
        label: "Net PnL Rollup",
        delta_minor: 1980000,
        cost_minor: 400000,
        detail: "Scenario patches: 2",
      },
    ]);
    expect(parseCommittedProposalIds({ committed_proposal_ids_json: '["prop-a","prop-b"]' })).toEqual([
      "prop-a",
      "prop-b",
    ]);
    expect(
      extractP2bError(
        new ApiError(
          "scim error",
          403,
          '{"schemas":["urn:ietf:params:scim:api:messages:2.0:Error"],"detail":"AUDITOR is read-only","status":"403"}',
        ),
        "fallback",
      ).message,
    ).toContain("AUDITOR is read-only");

    const calls: Array<{ url: string; method: string; idempotencyKey?: string | null }> = [];
    const fetchSpy = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      const headers = new Headers(init?.headers);
      const method = init?.method || "GET";
      calls.push({ url, method, idempotencyKey: headers.get("Idempotency-Key") });

      if (url.includes("/v1/compliance/haccp") && url.includes("format=csv")) {
        return new Response("record_type,reading_id,temp_milli_c\nEXCURSION,rd-1,9500\n", {
          status: 200,
          headers: { "Content-Type": "text/csv" },
        });
      }
      if (url.includes("/scim/v2/Users") && method === "DELETE") {
        return new Response(null, { status: 204 });
      }
      if (url.includes("/scim/v2/Groups") && method === "DELETE") {
        return new Response(null, { status: 204 });
      }
      if (url.includes("/scim/v2/Users")) {
        const body =
          method === "GET" && !url.includes("/scim/v2/Users/")
            ? {
                schemas: ["urn:ietf:params:scim:api:messages:2.0:ListResponse"],
                totalResults: 1,
                startIndex: 1,
                itemsPerPage: 50,
                Resources: [{ id: "usr-1", userName: "alice@acme.uz", active: true }],
              }
            : { id: "usr-1", userName: "alice@acme.uz", active: true };
        return new Response(JSON.stringify(body), {
          status: 200,
          headers: { "Content-Type": "application/scim+json" },
        });
      }
      if (url.includes("/scim/v2/Groups")) {
        const body =
          method === "GET" && !url.includes("/scim/v2/Groups/")
            ? {
                schemas: ["urn:ietf:params:scim:api:messages:2.0:ListResponse"],
                totalResults: 1,
                startIndex: 1,
                itemsPerPage: 50,
                Resources: [{ id: "grp-1", displayName: "FinanceApprovers" }],
              }
            : { id: "grp-1", displayName: "FinanceApprovers" };
        return new Response(JSON.stringify(body), {
          status: 200,
          headers: { "Content-Type": "application/scim+json" },
        });
      }
      return new Response(
        JSON.stringify({
          simulation: { simulation_id: "sim-1" },
          custom_role: { role_name: "REGIONAL_CONTROLLER" },
          ok: true,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      );
    });

    const realClient = new ApiClient({
      baseUrl: "https://api.pegasusx.test",
      getAuthToken: () => "tok-4b",
      fetchImpl: fetchSpy as unknown as typeof fetch,
    });
    const api = createP2bApi(realClient);

    await api.getAnomalyCatalogue();
    await api.scanAnomalies({ entity_id: "ent-1", rule_codes: ["CASH_GEOFENCE_MISMATCH"] }, "idem-scan");
    await api.listAnomalyCases({ status: "OPEN", severity: "HIGH", category: "FRAUD", limit: 25 });
    await api.resolveAnomalyCase(
      "case-1",
      { status: "ESCALATED", resolution_note: "Verify cash", create_proposal: true },
      "idem-res",
    );

    const sim = await api.runWhatIfSimulation(
      {
        name: "Stress Q4",
        horizon_days: 30,
        demand_delta_bps: 1500,
        factory_downtime_hours: 12,
        fx_rate_shock_bps: 500,
        credit_limit_delta_bps: -1000,
        unit_price_delta_bps: 0,
      },
      "idem-sim",
    );
    expect(sim.simulation_id).toBe("sim-1");
    await api.listWhatIfSimulations({ limit: 20 });
    await api.getWhatIfSimulation("sim-1");
    await api.commitWhatIfProposals("sim-1", {}, "idem-commit");

    await api.getHaccpReport({ from: "2026-11-01", to: "2026-11-17" });
    const csv = await api.downloadHaccpCsv({ from: "2026-11-01", to: "2026-11-17" });
    expect(csv).toContain("EXCURSION,rd-1,9500");

    await api.evaluateL2Readiness({ max_l2_amount_minor: 5_000_000_00 });
    await api.runL2AutoExecute({ auto_execute: true, max_l2_amount_minor: 5_000_000_00 }, "idem-l2");

    await api.listCustomRoles();
    const role = await api.createCustomRole(
      { role_name: "REGIONAL_CONTROLLER", permissions: ["audit.read", "finance.read"] },
      "idem-cr",
    );
    expect(role.role_name).toBe("REGIONAL_CONTROLLER");
    await api.deleteCustomRole("REGIONAL_CONTROLLER", "idem-cr-del");

    const scimUsers = await api.listScimUsers({ count: 50 });
    expect(scimUsers.Resources[0].userName).toBe("alice@acme.uz");
    await api.createScimUser({ userName: "alice@acme.uz", active: true }, "idem-su-create");
    await api.getScimUser("usr-1");
    await api.replaceScimUser("usr-1", { userName: "alice@acme.uz", active: false }, "idem-su-put");
    await api.patchScimUser(
      "usr-1",
      { Operations: [{ op: "replace", path: "active", value: false }] },
      "idem-su-patch",
    );
    await api.deleteScimUser("usr-1", "idem-su-del");

    const scimGroups = await api.listScimGroups({ count: 50 });
    expect(scimGroups.Resources[0].displayName).toBe("FinanceApprovers");
    await api.createScimGroup({ displayName: "FinanceApprovers" }, "idem-sg-create");
    await api.getScimGroup("grp-1");
    await api.replaceScimGroup("grp-1", { displayName: "FinanceApprovers" }, "idem-sg-put");
    await api.patchScimGroup("grp-1", { Operations: [{ op: "add", path: "members" }] }, "idem-sg-patch");
    await api.deleteScimGroup("grp-1", "idem-sg-del");

    for (const c of calls.filter((x) => x.method === "POST")) {
      expect(c.idempotencyKey, `Missing Idempotency-Key on ${c.url}`).toBeTruthy();
    }
    expect(calls.some((c) => c.url.includes("/v1/anomalies/cases/case-1/resolve"))).toBe(true);
    expect(calls.some((c) => c.url.includes("/v1/whatif/simulations/sim-1/commit-proposals"))).toBe(true);
    expect(calls.some((c) => c.url.includes("/v1/proposals/l2-evaluate"))).toBe(true);
    expect(calls.some((c) => c.url.includes("/v1/iam/custom-roles/REGIONAL_CONTROLLER"))).toBe(true);
    expect(calls.some((c) => c.url.includes("/scim/v2/Users/usr-1") && c.method === "PATCH")).toBe(true);
  });

  it("renders AnomalyCataloguePage, runs live scan, inspects evidence, escalates case, and enforces AUDITOR read-only", async () => {
    let escalated = false;
    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path === "/v1/anomalies/catalogue" && method === "GET") {
        return {
          count: 1,
          rules: [
            {
              rule_code: "BASKET_SPIKE_SHARED_INN",
              name: "Shared INN & Basket Spike",
              category: "COLLUSION_FRAUD",
              default_severity: "HIGH",
              description: "Detects multiple retailers sharing the same normalized TIN.",
              default_threshold_bps: 3000,
              playbook_action: "FREEZE_CREDIT",
            },
          ],
        };
      }
      if (path.startsWith("/v1/anomalies/cases?") || path === "/v1/anomalies/cases") {
        return {
          count: 1,
          cases: [
            {
              tenant_id: "sup-1",
              case_id: "case-101",
              rule_code: "BASKET_SPIKE_SHARED_INN",
              category: "COLLUSION_FRAUD",
              severity: "HIGH",
              status: escalated ? "ESCALATED" : "OPEN",
              object_type: "Retailer",
              object_id: "ret-77",
              exposure_minor: 8_500_000_00,
              score_bps: 9100,
              evidence_json: '{"shared_tin":"301112223","retailers":["ret-77","ret-78"]}',
              playbook_action: "FREEZE_CREDIT",
              proposal_id: escalated ? "prop-anom-1" : undefined,
              resolution_note: escalated ? "Escalate shared TIN ring" : undefined,
              detected_at: "2026-11-17T10:00:00Z",
              updated_at: "2026-11-17T10:00:00Z",
            },
          ],
        };
      }
      if (path === "/v1/anomalies/scan" && method === "POST") {
        return { created_count: 1, cases: [{ case_id: "case-101" }] };
      }
      if (path === "/v1/anomalies/cases/case-101/resolve" && method === "POST") {
        escalated = true;
        return {
          case: {
            tenant_id: "sup-1",
            case_id: "case-101",
            rule_code: "BASKET_SPIKE_SHARED_INN",
            category: "FRAUD",
            severity: "HIGH",
            status: "ESCALATED",
            object_type: "Retailer",
            object_id: "ret-77",
            exposure_minor: 8_500_000_00,
            score_bps: 9100,
            proposal_id: "prop-anom-1",
            resolution_note: "Escalate shared TIN ring",
            detected_at: "2026-11-17T10:00:00Z",
            updated_at: "2026-11-17T10:05:00Z",
          },
          proposal_id: "prop-anom-1",
        };
      }
      return {};
    });

    render(<AnomalyCataloguePage />);

    await waitFor(() => {
      expect(screen.getAllByText("BASKET_SPIKE_SHARED_INN").length).toBeGreaterThan(0);
      expect(screen.getAllByText("8,500,000 UZS").length).toBeGreaterThan(0);
    });

    fireEvent.click(screen.getByRole("button", { name: "Run Anomaly Scan" }));
    await waitFor(() => {
      expect(screen.getByText(/Scan finished: 1 new case/)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Inspect" }));
    await waitFor(() => {
      expect(screen.getByRole("dialog")).toBeInTheDocument();
      expect(screen.getByText(/301112223/)).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("Resolution Note"), {
      target: { value: "Escalate shared TIN ring" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Escalate Case" }));

    await waitFor(() => {
      expect(screen.getByText(/L1 proposal prop-anom-1 created/)).toBeInTheDocument();
      expect(screen.getByRole("link", { name: /prop-anom-1/ })).toHaveAttribute(
        "href",
        "/proposals?proposal_id=prop-anom-1",
      );
    });

    cleanup();
    mockSupplierTokenRef.current = makeJwt({
      sub: "aud-1",
      role: "ADMIN",
      supplier_id: "sup-1",
      supplier_role: "AUDITOR",
    });

    render(<AnomalyCataloguePage />);
    await waitFor(() => {
      expect(screen.getByTestId("anomalies-read-only-banner")).toBeInTheDocument();
    });
    expect(screen.queryByRole("button", { name: "Run Anomaly Scan" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "Inspect" }));
    expect(screen.queryByRole("button", { name: "Resolve Case" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Escalate Case" })).not.toBeInTheDocument();
  });

  it("renders WhatIfSimulatorPage, converts % shocks to bps, commits L1 proposals, and enforces AUDITOR read-only", async () => {
    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path.startsWith("/v1/whatif/simulations?") || path === "/v1/whatif/simulations") {
        return { count: 0, simulations: [] };
      }
      if (path === "/v1/whatif/simulate" && method === "POST") {
        return {
          simulation: {
            tenant_id: "sup-1",
            simulation_id: "sim-q4-1",
            name: "Navruz Shock",
            horizon_days: 30,
            shocks_json:
              '{"demand_delta_bps":2000,"factory_downtime_hours":12,"fx_rate_shock_bps":500,"credit_limit_delta_bps":-1000,"unit_price_delta_bps":0,"currency":"UZS"}',
            forecast_delta_minor: 12_000_000_00,
            replenishment_cost_minor: 4_500_000_00,
            dispatch_cost_delta_minor: 800_000_00,
            credit_exposure_delta_minor: -1_500_000_00,
            net_pnl_delta_minor: 6_700_000_00,
            breakdown: [
              {
                stage: "FORECAST",
                label: "Demand Forecast",
                delta_minor: 12_000_000_00,
                metric_bps: 2000,
                detail: "+20% demand shock",
              },
            ],
            status: "SIMULATED",
            created_by: "owner-1",
            created_at: "2026-11-17T10:00:00Z",
            updated_at: "2026-11-17T10:00:00Z",
          },
        };
      }
      if (path === "/v1/whatif/simulations/sim-q4-1/commit-proposals" && method === "POST") {
        return {
          simulation: {
            tenant_id: "sup-1",
            simulation_id: "sim-q4-1",
            name: "Navruz Shock",
            horizon_days: 30,
            shocks_json: "{}",
            forecast_delta_minor: 12_000_000_00,
            replenishment_cost_minor: 4_500_000_00,
            dispatch_cost_delta_minor: 800_000_00,
            credit_exposure_delta_minor: -1_500_000_00,
            net_pnl_delta_minor: 6_700_000_00,
            committed_proposal_ids: ["prop-sim-1"],
            status: "COMMITTED",
            created_by: "owner-1",
            created_at: "2026-11-17T10:00:00Z",
            updated_at: "2026-11-17T10:05:00Z",
          },
          committed_proposal_ids: ["prop-sim-1"],
          proposals: [
            {
              proposal_id: "prop-sim-1",
              agent_id: "REORDER_AGENT",
              proposal_type: "REORDER",
              target_object_type: "Sku",
              target_object_id: "SKU-1",
              title: "Reorder SKU-1",
              summary: "What-if reorder",
              payload_hash: "h1",
              confidence_bps: 8500,
              estimated_impact_minor: 4_500_000_00,
              currency: "UZS",
              status: "PENDING",
            },
          ],
        };
      }
      return {};
    });

    render(<WhatIfSimulatorPage />);

    await waitFor(() => {
      expect(screen.getByLabelText("Simulation Name")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("Simulation Name"), { target: { value: "Navruz Shock" } });
    fireEvent.change(screen.getByLabelText("Demand Delta (%)"), { target: { value: "20" } });
    fireEvent.click(screen.getByRole("button", { name: "Run What-If Simulation" }));

    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/whatif/simulate",
        "POST",
        expect.objectContaining({
          body: expect.objectContaining({
            name: "Navruz Shock",
            demand_delta_bps: 2000,
            factory_downtime_hours: 12,
          }),
        }),
      );
      expect(screen.getAllByText("6,700,000 UZS").length).toBeGreaterThan(0);
      expect(screen.getByText("Demand Forecast")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Commit as L1 Proposals" }));
    await waitFor(() => {
      expect(screen.getByText(/Committed 1 L1 proposal\(s\) from simulation sim-q4-1/)).toBeInTheDocument();
      expect(screen.getByRole("link", { name: /prop-sim-1/ })).toHaveAttribute(
        "href",
        "/proposals?proposal_id=prop-sim-1",
      );
    });

    cleanup();
    mockSupplierTokenRef.current = makeJwt({
      sub: "aud-1",
      role: "ADMIN",
      supplier_id: "sup-1",
      supplier_role: "AUDITOR",
    });

    render(<WhatIfSimulatorPage />);
    await waitFor(() => {
      expect(screen.getByTestId("whatif-read-only-banner")).toBeInTheDocument();
    });
    expect(screen.queryByRole("button", { name: "Run What-If Simulation" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Commit as L1 Proposals" })).not.toBeInTheDocument();
  });

  it("renders HaccpCompliancePage with milli-Celsius excursions, affected lots, and CSV export", async () => {
    mockRequest.mockImplementation(async (path: string) => {
      if (path.includes("format=csv")) {
        return "record_type,reading_id,temp_milli_c\nEXCURSION,rd-99,11250\n";
      }
      return {
        tenant_id: "sup-1",
        generated_at: "2026-11-17T12:00:00Z",
        total_readings: 120,
        excursion_count: 3,
        compliance_rate_bps: 9750,
        open_breaches: 1,
        affected_lots_count: 1,
        excursions: [
          {
            reading_id: "rd-99",
            manifest_id: "man-12",
            warehouse_id: "wh-cold-1",
            sensor_id: "sens-4",
            recorded_at: "2026-11-17T11:30:00Z",
            temp_milli_c: 11250,
            min_milli_c: 0,
            max_milli_c: 8000,
            breach_delta_milli_c: 3250,
            severity: "CRITICAL",
            status: "OPEN",
          },
        ],
        affected_lots: [
          {
            lot_id: "lot-chilled-1",
            warehouse_id: "wh-cold-1",
            product_id: "prod-dairy-1",
            product_name: "Fresh Milk 1L",
            location_id: "bin-ambient-2",
            lot_code: "LOT-2026-11",
            status: "QUARANTINE",
            quantity_on_hand: 240,
            required_zone: "CHILLED",
            location_zone: "AMBIENT",
            reason: "Temperature excursion +3.25°C",
          },
        ],
        cold_zone_mismatches: [
          {
            lot_id: "lot-chilled-1",
            product_id: "prod-dairy-1",
            product_name: "Fresh Milk 1L",
            location_id: "bin-ambient-2",
            required_zone: "CHILLED",
            location_zone: "AMBIENT",
            warning: "Move chilled lot into refrigerated bin",
          },
        ],
        condition_reports: [
          {
            report_id: "cr-55",
            order_id: "ord-88",
            sku: "MILK-1L",
            condition_type: "TEMPERATURE_BREACH",
            severity: "HIGH",
            resolution_status: "OPEN",
            description: "Truck cabin above +8°C",
            created_at: "2026-11-17T11:40:00Z",
          },
        ],
      };
    });

    mockSupplierTokenRef.current = makeJwt({
      sub: "aud-1",
      role: "ADMIN",
      supplier_id: "sup-1",
      supplier_role: "AUDITOR",
    });

    render(<HaccpCompliancePage />);

    await waitFor(() => {
      expect(screen.getByTestId("haccp-read-only-banner")).toBeInTheDocument();
      expect(screen.getByText("97.50%")).toBeInTheDocument();
      expect(screen.getByText("11.25°C")).toBeInTheDocument();
      expect(screen.getAllByText("Fresh Milk 1L").length).toBeGreaterThan(0);
      expect(screen.getByText(/cr-55/)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Export HACCP CSV" }));
    await waitFor(() => {
      expect(screen.getByText(/EXCURSION,rd-99,11250/)).toBeInTheDocument();
    });
  });

  it("renders L2 Soak Gate panel on ProposalsInboxPage, executes eligible L2 proposals, and enforces AUDITOR read-only", async () => {
    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path.startsWith("/v1/proposals/l2-evaluate") && method === "GET") {
        return {
          tenant_id: "sup-1",
          soak_days: 16,
          min_soak_days: 14,
          executed_count: 12,
          min_executed_count: 10,
          rollback_rate_bps: 200,
          max_rollback_rate_bps: 500,
          max_l2_amount_minor: 10_000_000_00,
          eligible: true,
          blockers: [],
          eligible_count: 1,
          eligible_proposal_ids: ["prop-reord-1"],
        };
      }
      if (path === "/v1/proposals/l2-evaluate" && method === "POST") {
        return {
          tenant_id: "sup-1",
          soak_days: 16,
          min_soak_days: 14,
          executed_count: 13,
          min_executed_count: 10,
          rollback_rate_bps: 200,
          max_rollback_rate_bps: 500,
          max_l2_amount_minor: 10_000_000_00,
          eligible: true,
          blockers: [],
          auto_executed_count: 1,
          auto_executed_proposal_ids: ["prop-reord-1"],
          blocked_proposals: [
            {
              proposal_id: "prop-cred-9",
              proposal_type: "CREDIT_LIMIT",
              reason_code: "high_risk_action_blocked",
              reason: "CREDIT_LIMIT never auto-executes at L2",
            },
          ],
        };
      }
      return {
        count: 1,
        proposals: [
          {
            proposal_id: "prop-reord-1",
            agent_id: "REORDER_AGENT",
            proposal_type: "REORDER",
            target_object_type: "Sku",
            target_object_id: "SKU-5",
            title: "Auto-reorder SKU-5",
            summary: "Low stock",
            rationale: "Soak passed",
            payload: { qty: 10 },
            payload_hash: "h-reord",
            confidence_bps: 9200,
            estimated_impact_minor: 500_000_00,
            currency: "UZS",
            roi_category: "STOCKOUT_AVOIDED",
            autonomy_level: "L2",
            status: "EXECUTED",
          },
        ],
      };
    });

    render(<ProposalsInboxPage />);

    await waitFor(() => {
      const panel = screen.getByTestId("l2-soak-gate-panel");
      expect(panel).toHaveTextContent("16 / 14d");
      expect(panel).toHaveTextContent("12 / 10");
      expect(panel).toHaveTextContent("L2 Ready");
      expect(screen.getAllByText(/L2/).length).toBeGreaterThan(0);
    });

    fireEvent.click(screen.getByRole("button", { name: "Auto-Execute Eligible L2" }));
    await waitFor(() => {
      expect(screen.getByText(/1 proposal\(s\) promoted to L2/)).toBeInTheDocument();
      expect(screen.getByText(/high_risk_action_blocked/)).toBeInTheDocument();
    });

    cleanup();
    mockSupplierTokenRef.current = makeJwt({
      sub: "agent-1",
      role: "AGENT",
      actor_type: "agent",
      supplier_id: "sup-1",
      supplier_role: "AGENT",
    });

    render(<ProposalsInboxPage />);
    await waitFor(() => {
      expect(screen.getByTestId("proposals-read-only-banner")).toBeInTheDocument();
    });
    expect(screen.queryByRole("button", { name: "Run Reorder Agent" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Auto-Execute Eligible L2" })).not.toBeInTheDocument();
  });

  it("renders Custom Roles & SCIM v2 tab on SupplierGovernancePage and enforces AUDITOR read-only", async () => {
    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path.startsWith("/v1/audit/verify")) {
        return {
          tenant_id: "sup-1",
          day_bucket: "2026-11-17",
          valid: true,
          event_count: 1,
          root_hash: "abc1234567890",
        };
      }
      if (path.startsWith("/v1/audit/anchors")) return { count: 0, anchors: [] };
      if (path.startsWith("/v1/audit/events")) return { count: 0, pii_masked: false, events: [] };
      if (path.startsWith("/v1/audit/legal-holds")) return { count: 0, holds: [] };
      if (path.startsWith("/v1/iam/roles")) return { roles: [], user_roles: [] };
      if (path === "/v1/iam/permissions") {
        return {
          count: 1,
          permissions: [{ key: "audit.read", category: "Audit", description: "Read audit log" }],
        };
      }
      if (path.startsWith("/v1/iam/sod-policies")) return { count: 0, policies: [] };
      if (path.startsWith("/v1/iam/sod-violations")) return { count: 0, violations: [] };
      if (path === "/v1/iam/custom-roles" && method === "GET") {
        return {
          count: 1,
          custom_roles: [
            {
              tenant_id: "sup-1",
              role_name: "REGIONAL_VIEWER",
              description: "Read-only branch audit",
              permissions: ["audit.read"],
              read_only: true,
              created_by: "owner-1",
              status: "ACTIVE",
              created_at: "2026-11-17T08:00:00Z",
              updated_at: "2026-11-17T08:00:00Z",
            },
          ],
        };
      }
      if (path === "/v1/iam/custom-roles" && method === "POST") {
        return {
          custom_role: {
            tenant_id: "sup-1",
            role_name: "FIN_OPS_CUSTOM",
            permissions: ["audit.read"],
            read_only: false,
            status: "ACTIVE",
          },
        };
      }
      if (path.startsWith("/v1/iam/custom-roles/") && method === "DELETE") {
        return { deleted: true, role_name: "REGIONAL_VIEWER" };
      }
      if (path.startsWith("/scim/v2/Users") && method === "GET") {
        return {
          totalResults: 1,
          startIndex: 1,
          itemsPerPage: 50,
          Resources: [
            {
              id: "scim-u-1",
              userName: "bob@acme.uz",
              displayName: "Bob",
              emails: [{ value: "bob@acme.uz", primary: true }],
              active: true,
              roles: [{ value: "FINANCE_MANAGER" }],
            },
          ],
        };
      }
      if (path === "/scim/v2/Users" && method === "POST") {
        return { id: "scim-u-2", userName: "carol@acme.uz", active: true };
      }
      if (path.startsWith("/scim/v2/Users/") && method === "PATCH") {
        return { id: "scim-u-1", userName: "bob@acme.uz", active: false };
      }
      if (path.startsWith("/scim/v2/Groups") && method === "GET") {
        return {
          totalResults: 1,
          startIndex: 1,
          itemsPerPage: 50,
          Resources: [{ id: "scim-g-1", displayName: "FinanceApprovers", members: [{ value: "scim-u-1" }] }],
        };
      }
      if (path === "/scim/v2/Groups" && method === "POST") {
        return { id: "scim-g-2", displayName: "OpsAuditors" };
      }
      return {};
    });

    render(<SupplierGovernancePage />);

    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Custom Roles & SCIM v2" })).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole("button", { name: "Custom Roles & SCIM v2" }));

    await waitFor(() => {
      expect(screen.getByTestId("governance-custom-scim-panel")).toBeInTheDocument();
      expect(screen.getByText("REGIONAL_VIEWER")).toBeInTheDocument();
      expect(screen.getAllByText("bob@acme.uz").length).toBeGreaterThan(0);
      expect(screen.getByText("FinanceApprovers")).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("Custom Role Name"), {
      target: { value: "FIN_OPS_CUSTOM" },
    });
    fireEvent.change(screen.getByLabelText("Additional Permissions (comma-separated)"), {
      target: { value: "audit.read" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Create Custom Role" }));
    await waitFor(() => {
      expect(screen.getByText(/Custom role FIN_OPS_CUSTOM saved/)).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("Username"), { target: { value: "carol@acme.uz" } });
    fireEvent.click(screen.getByRole("button", { name: "Provision SCIM User" }));
    await waitFor(() => {
      expect(screen.getByText(/SCIM user carol@acme.uz \(scim-u-2\) provisioned/)).toBeInTheDocument();
    });

    fireEvent.click(screen.getByRole("button", { name: "Deactivate" }));
    await waitFor(() => {
      expect(screen.getByText(/SCIM user bob@acme.uz updated \(INACTIVE\)/)).toBeInTheDocument();
    });

    fireEvent.change(screen.getByLabelText("Group Name"), { target: { value: "OpsAuditors" } });
    fireEvent.click(screen.getByRole("button", { name: "Create SCIM Group" }));
    await waitFor(() => {
      expect(screen.getByText(/SCIM group OpsAuditors \(scim-g-2\) created/)).toBeInTheDocument();
    });

    cleanup();
    mockSupplierTokenRef.current = makeJwt({
      sub: "aud-1",
      role: "ADMIN",
      supplier_id: "sup-1",
      supplier_role: "AUDITOR",
    });
    render(<SupplierGovernancePage />);
    await waitFor(() => {
      expect(screen.getByTestId("governance-read-only-banner")).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole("button", { name: "Custom Roles & SCIM v2" }));
    expect(screen.queryByRole("button", { name: "Create Custom Role" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Provision SCIM User" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Create SCIM Group" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Delete Role" })).not.toBeInTheDocument();
  });

  it("enforces Wave 4B nav registration, Suspense wrapping, en/ru/uz i18n parity, and zero @pegasusx/ui imports", () => {
    const hrefs = allNavHrefs(NAV);
    expect(hrefs).toContain("/control-tower/anomalies");
    expect(hrefs).toContain("/planning/what-if");
    expect(hrefs).toContain("/compliance/haccp");

    const requiredKeys = [
      "portal.nav.anomalies",
      "portal.nav.what_if",
      "portal.nav.haccp",
      "supplier_portal.anomalies.title",
      "supplier_portal.anomalies.run_scan_button",
      "supplier_portal.what_if.title",
      "supplier_portal.what_if.commit_proposals_button",
      "supplier_portal.haccp.title",
      "supplier_portal.haccp.export_csv_button",
      "supplier_portal.proposals.l2_title",
      "supplier_portal.proposals.l2_auto_execute_button",
      "supplier_portal.governance.tab_custom_scim",
      "supplier_portal.governance.custom_roles_title",
      "supplier_portal.governance.scim_users_title",
      "supplier_portal.governance.scim_groups_title",
    ];
    for (const k of requiredKeys) {
      expect(messageCatalogs.en[k], `en missing ${k}`).toBeTruthy();
      expect(messageCatalogs.ru[k], `ru missing ${k}`).toBeTruthy();
      expect(messageCatalogs.uz[k], `uz missing ${k}`).toBeTruthy();
    }

    const ownedFiles = [
      "lib/p2b-api.ts",
      "lib/nav.ts",
      "app/(portal)/control-tower/anomalies/page.tsx",
      "app/(portal)/planning/what-if/page.tsx",
      "app/(portal)/compliance/haccp/page.tsx",
      "app/(portal)/proposals/page.tsx",
      "app/(portal)/governance/page.tsx",
    ];
    for (const rel of ownedFiles) {
      const full = join(portalRoot, rel);
      expect(existsSync(full), rel).toBe(true);
      const src = readFileSync(full, "utf8");
      expect(src, `${rel} must never import @pegasusx/ui`).not.toContain("@pegasusx/ui");
      if (rel.endsWith("page.tsx")) {
        expect(src, `${rel} must wrap useSearchParams in Suspense`).toContain("<Suspense");
      }
    }
  });
});
