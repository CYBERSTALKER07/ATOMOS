import { existsSync, readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { ApiClient, ApiError } from "@pegasusx/api-core";
import { createTranslator, messageCatalogs } from "@pegasusx/i18n";

const { mockRequest, mockSearchParamsRef } = vi.hoisted(() => ({
  mockRequest: vi.fn(),
  mockSearchParamsRef: { current: new URLSearchParams() },
}));

vi.mock("@/lib/api", () => ({
  createSupplierApi: () => ({
    request: mockRequest,
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

vi.mock("@/components/audit/ObjectHistoryPanel", () => ({
  ObjectHistoryPanel: ({ objectType, objectId }: { objectType: string; objectId: string }) => (
    <div data-testid="object-history-panel">{`${objectType}:${objectId}`}</div>
  ),
}));

import {
  createControlAccountingApi,
  evaluateVatTieOut,
  formatFinanceMinor,
  formatRateBps,
  parseMajorToSafeMinor,
} from "@/lib/control-accounting-api";
import { NAV, allNavHrefs } from "@/lib/nav";
import PeriodClosePage from "@/app/(portal)/finance/close/page";
import ApprovalsPage from "@/app/(portal)/approvals/page";
import FiscalEsfConsolePage from "@/app/(portal)/fiscal/page";

const here = dirname(fileURLToPath(import.meta.url));
const portalRoot = join(here, "../..");

describe("Wave 3A Control & Accounting Portal (M1.1, M1.2, M1.4)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockSearchParamsRef.current = new URLSearchParams();
  });

  afterEach(() => {
    cleanup();
  });

  it("handles BigInt minor units, bps rates, and VAT DiffMinor == 0 without floating-point rounding", () => {
    expect(formatFinanceMinor(BigInt("900719925474099300"), "UZS")).toBe("9,007,199,254,740,993 UZS");
    expect(parseMajorToSafeMinor("1500000.50")).toBe(150000050);
    expect(parseMajorToSafeMinor("-10")).toBeNull();
    expect(parseMajorToSafeMinor("12.345")).toBeNull();
    expect(formatRateBps(1200)).toBe("12%");
    expect(formatRateBps(1250)).toBe("12.50%");

    const tied = evaluateVatTieOut({
      entity_id: "sup-1",
      period_id: "2026-11",
      currency: "UZS",
      esf_output_vat_minor: 12000000,
      esf_input_vat_minor: 3600000,
      esf_net_vat_minor: 8400000,
      gl_output_vat_minor: 12000000,
      gl_input_vat_minor: 3600000,
      gl_net_vat_minor: 8400000,
      output_diff_minor: 0,
      input_diff_minor: 0,
      diff_minor: 0,
      reconciled: true,
      outbound_count: 5,
      inbound_count: 2,
      rejected_count: 0,
    });
    expect(tied.isZeroDiff).toBe(true);
    expect(tied.diffMinor).toBe(BigInt(0));
  });

  it("wires all Wave 3A HTTP routes, query params, CSV snapshot, and idempotency keys via ApiClient", async () => {
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
        return new Response("section,account_code,net_minor\ntrial_balance,INVENTORY,500000\n", {
          status: 200,
          headers: { "Content-Type": "text/csv" },
        });
      }
      return new Response(JSON.stringify({ ok: true }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });

    const realClient = new ApiClient({
      baseUrl: "https://api.pegasusx.test",
      getAuthToken: () => "tok-wave3a",
      fetchImpl: fetchSpy as unknown as typeof fetch,
    });
    const api = createControlAccountingApi(realClient);

    // M1.1 Periods
    await api.listPeriods({ status: "OPEN" });
    await api.createPeriod({ period_id: "2026-11", timezone: "Asia/Tashkent" }, "idem-p1");
    await api.softClosePeriod("2026-11", {}, "idem-p2");
    await api.closePeriod("2026-11", {}, "idem-p3");
    await api.reopenPeriod("2026-11", { reason: "Audit adj" }, "idem-p4");
    await api.evaluatePeriodChecklist("2026-11");
    await api.getPeriodSnapshot("2026-11");
    const csv = await api.downloadPeriodSnapshotCsv("2026-11");
    expect(csv).toContain("trial_balance,INVENTORY,500000");

    // M1.2 Approvals
    await api.listApprovalPolicies({ object_type: "JournalEntry" });
    await api.createApprovalPolicy({
      name: "Manual GL Guard",
      object_type: "JournalEntry",
      action: "manual_post",
      condition_expr: "amount_minor > 50000000",
      steps: [{ step_index: 0, allowed_roles: ["FINANCE_MANAGER"], required_approvers: 1, require_mfa: true }],
      enabled: true,
    });
    await api.toggleApprovalPolicy("pol-1", { enabled: false });
    await api.listApprovalInbox({ status: "PENDING", limit: 25 });
    await api.getApprovalRequest("apr-1");
    await api.approveRequest("apr-1", { mfa_verified: true, totp_code: "123456", payload_hash: "sha256-abc" });
    await api.rejectRequest("apr-2", { comment: "Missing doc" });
    await api.listDelegations({ actor_id: "usr-1" });
    await api.createDelegation({ delegate_id: "usr-2", ends_at: "2026-12-01T00:00:00Z", reason: "Leave" });
    await api.revokeDelegation("del-1");

    // M1.4 ESF & Fiscal
    await api.listEsfDocuments({ direction: "OUTBOUND", period_id: "2026-11" });
    await api.createOutboundEsf({ order_id: "ord-1", submit: true });
    await api.getEsfDocument("esf-1");
    await api.resendEsfDocument("esf-1");
    await api.recordInboundEsf({
      counterparty_tin: "301234567",
      counterparty_name: "Vendor LLC",
      total_minor: 1120000,
      vat_minor: 120000,
      vat_rate_bps: 1200,
      operator_ref: "SOLIQ-IN-1",
    });
    await api.transitionEsfStatus("esf-1", { action: "REJECT", reason: "Buyer dispute" });
    await api.getVatSummary({ period_id: "2026-11" });
    await api.listMissingIkpu({ supplier_id: "sup-1" });

    expect(calls.map((c) => `${c.method} ${c.url}`)).toEqual([
      "GET https://api.pegasusx.test/v1/finance/periods?status=OPEN",
      "POST https://api.pegasusx.test/v1/finance/periods",
      "POST https://api.pegasusx.test/v1/finance/periods/2026-11/soft-close",
      "POST https://api.pegasusx.test/v1/finance/periods/2026-11/close",
      "POST https://api.pegasusx.test/v1/finance/periods/2026-11/reopen",
      "GET https://api.pegasusx.test/v1/finance/periods/2026-11/checklist",
      "GET https://api.pegasusx.test/v1/finance/periods/2026-11/snapshot?format=json",
      "GET https://api.pegasusx.test/v1/finance/periods/2026-11/snapshot?format=csv",
      "GET https://api.pegasusx.test/v1/approvals/policies?object_type=JournalEntry",
      "POST https://api.pegasusx.test/v1/approvals/policies",
      "PATCH https://api.pegasusx.test/v1/approvals/policies/pol-1",
      "GET https://api.pegasusx.test/v1/approvals/inbox?status=PENDING&limit=25",
      "GET https://api.pegasusx.test/v1/approvals/apr-1",
      "POST https://api.pegasusx.test/v1/approvals/apr-1/approve",
      "POST https://api.pegasusx.test/v1/approvals/apr-2/reject",
      "GET https://api.pegasusx.test/v1/approvals/delegations?actor_id=usr-1",
      "POST https://api.pegasusx.test/v1/approvals/delegations",
      "POST https://api.pegasusx.test/v1/approvals/delegations/del-1/revoke",
      "GET https://api.pegasusx.test/v1/fiscal/esf?direction=OUTBOUND&period_id=2026-11",
      "POST https://api.pegasusx.test/v1/fiscal/esf",
      "GET https://api.pegasusx.test/v1/fiscal/esf/esf-1",
      "POST https://api.pegasusx.test/v1/fiscal/esf/esf-1/resend",
      "POST https://api.pegasusx.test/v1/fiscal/esf/inbound",
      "POST https://api.pegasusx.test/v1/fiscal/esf/esf-1/status",
      "GET https://api.pegasusx.test/v1/fiscal/vat-summary?period_id=2026-11",
      "GET https://api.pegasusx.test/v1/fiscal/missing-ikpu?supplier_id=sup-1",
    ]);
    expect(calls[1].idempotencyKey).toBe("idem-p1");
    expect(calls[10].idempotencyKey).toBeTruthy();
  });

  it("renders /finance/close with cutoff display, checklist evaluation, 409 failure UI, 202 2-approver reopen, SoD error, and snapshot preview", async () => {
    mockSearchParamsRef.current = new URLSearchParams("period=2026-10");
    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path.startsWith("/v1/finance/periods?") || path === "/v1/finance/periods") {
        if (method === "GET") {
          return {
            entity_id: "sup-1",
            timezone: "Asia/Tashkent",
            periods: [
              {
                entity_id: "sup-1",
                period_id: "2026-10",
                status: "CLOSED",
                timezone: "Asia/Tashkent",
                cutoff_local: "2026-10-31 23:59:59 +05",
                closed_by: "usr-cfo",
                closed_at: "2026-11-01T01:00:00Z",
                snapshot_uri: "file:///worm/sup-1/2026-10.json",
                snapshot_hash: "sha256-deadbeef1234567890",
              },
              {
                entity_id: "sup-1",
                period_id: "2026-11",
                status: "OPEN",
                timezone: "Asia/Tashkent",
                cutoff_local: "2026-11-30 23:59:59 +05",
              },
            ],
          };
        }
      }
      if (path === "/v1/finance/periods/2026-11/close" && method === "POST") {
        throw new ApiError("checklist failed", 409, {
          code: "period_checklist_failed",
          error: "Open recon breaks block close",
          checklist: {
            entity_id: "sup-1",
            period_id: "2026-11",
            timezone: "Asia/Tashkent",
            ready_to_close: false,
            blocking_count: 1,
            items: [
              {
                key: "nightly_recon",
                label: "Nightly 4-way reconciliation",
                passed: false,
                blocking: true,
                open_count: 2,
                detail: "2 open breaks in STOCK_GL",
              },
            ],
          },
        });
      }
      if (path === "/v1/finance/periods/2026-10/reopen" && method === "POST") {
        return {
          status: "PENDING_APPROVAL",
          code: "approval_required",
          approval_request_id: "apr-reopen-10",
          required_approvers: 2,
          payload_hash: "hash-reopen-10",
        };
      }
      if (path === "/v1/finance/periods/2026-10/snapshot?format=json") {
        return {
          entity_id: "sup-1",
          period_id: "2026-10",
          snapshot_hash: "sha256-deadbeef1234567890",
          snapshot_uri: "file:///worm/sup-1/2026-10.json",
          trial_balance: { currency: "UZS", total_debits_minor: 50000000, total_credits_minor: 50000000 },
        };
      }
      if (path === "/v1/finance/periods/2026-10/snapshot?format=csv") {
        return "section,account_code,net_minor\ntrial_balance,INVENTORY,50000000\n";
      }
      return {};
    });

    render(<PeriodClosePage />);

    await waitFor(() => {
      expect(screen.getAllByText("2026-10-31 23:59:59 +05").length).toBeGreaterThan(0);
    });
    expect(screen.getAllByText("sha256-dea…34567890").length).toBeGreaterThan(0);

    // Load snapshot JSON preview & verify BigInt minor formatting
    fireEvent.click(screen.getByRole("button", { name: "Preview JSON" }));
    await waitFor(() => {
      expect(screen.getAllByText("500,000 UZS").length).toBeGreaterThan(0);
    });

    // Load CSV preview
    fireEvent.click(screen.getByRole("button", { name: "Download CSV" }));
    await waitFor(() => {
      expect(screen.getByText(/trial_balance,INVENTORY,50000000/)).toBeInTheDocument();
    });

    // Trigger 202 PENDING_APPROVAL on Reopen
    fireEvent.change(screen.getByLabelText("Reopen reason"), {
      target: { value: "Post late VAT credit note" },
    });
    fireEvent.click(screen.getByRole("button", { name: /Request \/ Reopen 2026-10/ }));
    await waitFor(() => {
      expect(screen.getByText("Open in Approvals Inbox")).toBeInTheDocument();
    });
    expect(screen.getByRole("link", { name: "Open in Approvals Inbox" })).toHaveAttribute(
      "href",
      "/approvals?requestId=apr-reopen-10",
    );

    // Trigger Hard-Close 409 checklist failure on 2026-11
    const hardCloseButtons = screen.getAllByRole("button", { name: "Hard-close" });
    fireEvent.click(hardCloseButtons[1]);
    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("period_checklist_failed");
      expect(screen.getByText("2 open breaks in STOCK_GL")).toBeInTheDocument();
    });
  });

  it("renders /approvals inbox with MFA, payload hash, SoD/mismatch errors, delegations, and CEL rules", async () => {
    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path.startsWith("/v1/approvals/inbox")) {
        return {
          count: 1,
          requests: [
            {
              tenant_id: "sup-1",
              request_id: "apr-101",
              policy_id: "pol-high-gl",
              policy_version: 3,
              object_type: "JournalEntry",
              object_id: "je-88",
              action: "manual_post",
              amount_minor: 75000000,
              currency: "UZS",
              requested_by: "usr-maker",
              actor_type: "human",
              payload_json: JSON.stringify({ memo: "Quarter-end accrual", amount_minor: 75000000 }),
              payload_hash: "sha256-payload-pin-999",
              status: "PENDING",
              current_step: 0,
              required_approvers: 2,
              created_at: "2026-11-17T01:00:00Z",
            },
          ],
        };
      }
      if (path === "/v1/approvals/apr-101") {
        return {
          request: {
            tenant_id: "sup-1",
            request_id: "apr-101",
            policy_id: "pol-high-gl",
            policy_version: 3,
            object_type: "JournalEntry",
            object_id: "je-88",
            action: "manual_post",
            amount_minor: 75000000,
            currency: "UZS",
            requested_by: "usr-maker",
            actor_type: "human",
            payload_json: JSON.stringify({ memo: "Quarter-end accrual", amount_minor: 75000000 }),
            payload_hash: "sha256-payload-pin-999",
            status: "PENDING",
            current_step: 0,
            required_approvers: 2,
            created_at: "2026-11-17T01:00:00Z",
          },
          decisions: [
            {
              request_id: "apr-101",
              decision_id: "dec-1",
              step_index: 0,
              approver_id: "usr-cfo",
              delegated_from: "usr-owner",
              decision: "APPROVE",
              comment: "Step 1 verified",
              mfa_verified: true,
              created_at: "2026-11-17T01:05:00Z",
            },
          ],
          policy: {
            tenant_id: "sup-1",
            policy_id: "pol-high-gl",
            name: "High Value GL",
            object_type: "JournalEntry",
            action: "manual_post",
            condition_expr: "amount_minor > 50000000",
            steps: [
              {
                step_index: 0,
                name: "Controller + CFO",
                allowed_roles: ["FINANCE_MANAGER", "OWNER"],
                required_approvers: 2,
                require_mfa: true,
                mfa_threshold_minor: 50000000,
              },
            ],
            version: 3,
            enabled: true,
          },
        };
      }
      if (path === "/v1/approvals/delegations") {
        return {
          count: 1,
          delegations: [
            {
              tenant_id: "sup-1",
              delegation_id: "del-9",
              delegator_id: "usr-owner",
              delegate_id: "usr-cfo",
              role_scope: "FINANCE_MANAGER",
              starts_at: "2026-11-16T00:00:00Z",
              ends_at: "2026-11-25T00:00:00Z",
              reason: "Travel coverage",
              enabled: true,
            },
          ],
        };
      }
      if (path === "/v1/approvals/policies" && method === "GET") {
        return {
          count: 1,
          policies: [
            {
              tenant_id: "sup-1",
              policy_id: "pol-high-gl",
              name: "High Value GL",
              object_type: "JournalEntry",
              action: "manual_post",
              condition_expr: "amount_minor > 50000000",
              steps: [
                {
                  step_index: 0,
                  allowed_roles: ["FINANCE_MANAGER"],
                  required_approvers: 2,
                  require_mfa: true,
                  mfa_threshold_minor: 50000000,
                },
              ],
              version: 3,
              enabled: true,
            },
          ],
        };
      }
      if (path === "/v1/approvals/apr-101/reject" && method === "POST") {
        throw new ApiError("SoD blocked", 403, {
          code: "sod_violation",
          error: "requester cannot decide own request",
        });
      }
      if (path === "/v1/approvals/apr-101/approve" && method === "POST") {
        return {
          request: {
            tenant_id: "sup-1",
            request_id: "apr-101",
            policy_id: "pol-high-gl",
            policy_version: 3,
            object_type: "JournalEntry",
            object_id: "je-88",
            action: "manual_post",
            amount_minor: 75000000,
            currency: "UZS",
            requested_by: "usr-maker",
            actor_type: "human",
            payload_hash: "sha256-payload-pin-999",
            status: "APPROVED",
            current_step: 1,
            required_approvers: 2,
            approvals_received: 2,
            created_at: "2026-11-17T01:00:00Z",
          },
          decision: {
            request_id: "apr-101",
            decision_id: "dec-2",
            step_index: 0,
            approver_id: "usr-controller",
            decision: "APPROVE",
            comment: "Second signoff",
            mfa_verified: true,
            created_at: "2026-11-17T01:10:00Z",
          },
        };
      }
      if (path === "/v1/approvals/policies" && method === "POST") {
        throw new ApiError("malformed CEL", 400, {
          code: "invalid_cel_expression",
          error: "floats are forbidden in CEL predicates",
        });
      }
      if (path === "/v1/approvals/policies/pol-high-gl" && method === "PATCH") {
        return { policy: { policy_id: "pol-high-gl", version: 4, enabled: false } };
      }
      return {};
    });

    render(<ApprovalsPage />);

    await waitFor(() => {
      expect(screen.getAllByText("apr-101").length).toBeGreaterThan(0);
      expect(screen.getAllByText("sha256-payload-pin-999").length).toBeGreaterThan(0);
      expect(screen.getByText("DELEGATED:usr-owner")).toBeInTheDocument();
    });
    expect(screen.getAllByText("750,000 UZS").length).toBeGreaterThan(0);

    // Verify 403 sod_violation error alert renders cleanly, then Approve with MFA + TOTP + Comment
    fireEvent.click(screen.getByRole("button", { name: "Reject" }));
    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("sod_violation");
    });

    fireEvent.change(screen.getByLabelText("MFA / TOTP code"), { target: { value: "654321" } });
    fireEvent.change(screen.getByLabelText("Reviewer comment"), { target: { value: "Second signoff" } });
    fireEvent.click(screen.getByRole("button", { name: "Approve" }));

    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/approvals/apr-101/approve",
        "POST",
        expect.objectContaining({
          body: expect.objectContaining({
            mfa_verified: true,
            totp_code: "654321",
            comment: "Second signoff",
            payload_hash: "sha256-payload-pin-999",
          }),
        }),
      );
    });

    expect(screen.getByRole("option", { name: "Period" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "FinancialDraft" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "FeatureFlag" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "PurchaseOrder" })).toBeInTheDocument();

    // Delegations tab
    fireEvent.click(screen.getByRole("tab", { name: "Delegations" }));
    expect(screen.getByText("Travel coverage")).toBeInTheDocument();

    // CEL policies tab -> toggle & malformed CEL 400 error UI
    fireEvent.click(screen.getByRole("tab", { name: "CEL policy rules" }));
    expect(screen.getByText("amount_minor > 50000000")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Disable" }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/approvals/policies/pol-high-gl",
        "PATCH",
        expect.objectContaining({
          body: { tenant_id: undefined, enabled: false },
        }),
      );
    });

    fireEvent.change(screen.getByLabelText("Policy name"), { target: { value: "Bad Float Rule" } });
    fireEvent.change(screen.getByLabelText("CEL expression"), { target: { value: "amount_minor > 1.5" } });
    fireEvent.click(screen.getByRole("button", { name: "Save CEL policy" }));
    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("invalid_cel_expression");
    });
    expect(mockRequest).toHaveBeenCalledWith(
      "/v1/approvals/policies",
      "POST",
      expect.objectContaining({
        body: expect.objectContaining({
          object_type: "JournalEntry",
          action: "manual_post",
        }),
      }),
    );
  });

  it("renders /fiscal ESF registry, status timeline, Resend ESF, VAT DiffMinor == 0 badge, and Missing IKPU table", async () => {
    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path.startsWith("/v1/fiscal/esf?")) {
        return {
          entity_id: "sup-1",
          count: 1,
          documents: [
            {
              entity_id: "sup-1",
              esf_id: "esf-out-1",
              direction: "OUTBOUND",
              order_id: "ord-77",
              counterparty_tin: "305551112",
              counterparty_name: "Korzinka Mart",
              total_minor: 11200000,
              vat_minor: 1200000,
              vat_rate_bps: 1200,
              currency: "UZS",
              status: "SUBMITTED",
              operator: "direct",
              operator_ref: "SOLIQ-REF-77",
              payload_hash: "sha256-esf-hash-77",
              period_id: "2026-11",
              created_at: "2026-11-17T01:00:00Z",
            },
          ],
        };
      }
      if (path === "/v1/fiscal/esf/esf-out-1" && method === "GET") {
        return {
          document: {
            entity_id: "sup-1",
            esf_id: "esf-out-1",
            direction: "OUTBOUND",
            order_id: "ord-77",
            counterparty_tin: "305551112",
            counterparty_name: "Korzinka Mart",
            total_minor: 11200000,
            vat_minor: 1200000,
            vat_rate_bps: 1200,
            currency: "UZS",
            status: "SUBMITTED",
            operator: "direct",
            operator_ref: "SOLIQ-REF-77",
            payload_hash: "sha256-esf-hash-77",
            created_at: "2026-11-17T01:00:00Z",
          },
          history: [
            {
              history_id: "hist-1",
              from_status: "DRAFT",
              to_status: "SUBMITTED",
              actor_id: "usr-fin",
              reason: "Signed via EDS",
              created_at: "2026-11-17T01:01:00Z",
            },
          ],
        };
      }
      if (path.startsWith("/v1/fiscal/vat-summary")) {
        return {
          entity_id: "sup-1",
          period_id: "2026-11",
          currency: "UZS",
          esf_output_vat_minor: 1200000,
          esf_input_vat_minor: 200000,
          esf_net_vat_minor: 1000000,
          gl_output_vat_minor: 1200000,
          gl_input_vat_minor: 200000,
          gl_net_vat_minor: 1000000,
          output_diff_minor: 0,
          input_diff_minor: 0,
          diff_minor: 0,
          reconciled: true,
          outbound_count: 1,
          inbound_count: 1,
          rejected_count: 0,
          by_rate: [
            {
              vat_rate_bps: 1200,
              outbound_base_minor: 10000000,
              outbound_vat_minor: 1200000,
              inbound_base_minor: 1666667,
              inbound_vat_minor: 200000,
              doc_count: 2,
            },
          ],
        };
      }
      if (path.startsWith("/v1/fiscal/missing-ikpu")) {
        return {
          supplier_id: "sup-1",
          count: 1,
          items: [
            {
              product_id: "sku-no-ikpu",
              name: "Sparkling Water 1L",
              barcode: "4780000001234",
              ikpu_code: "",
              package_code: "",
              marking_code: "",
              missing_fields: ["ikpu_code", "package_code"],
            },
          ],
        };
      }
      if (path === "/v1/fiscal/esf/esf-out-1/resend" && method === "POST") {
        return {
          entity_id: "sup-1",
          esf_id: "esf-out-1",
          direction: "OUTBOUND",
          order_id: "ord-77",
          counterparty_tin: "305551112",
          counterparty_name: "Korzinka Mart",
          total_minor: 11200000,
          vat_minor: 1200000,
          vat_rate_bps: 1200,
          currency: "UZS",
          status: "SUBMITTED",
          operator: "direct",
          created_at: "2026-11-17T01:00:00Z",
        };
      }
      if (path === "/v1/fiscal/esf/esf-out-1/status" && method === "POST") {
        return {
          document: {
            entity_id: "sup-1",
            esf_id: "esf-out-1",
            direction: "OUTBOUND",
            order_id: "ord-77",
            counterparty_tin: "305551112",
            counterparty_name: "Korzinka Mart",
            total_minor: 11200000,
            vat_minor: 1200000,
            vat_rate_bps: 1200,
            currency: "UZS",
            status: "REJECTED",
            operator: "direct",
            created_at: "2026-11-17T01:00:00Z",
          },
          history: [
            {
              history_id: "hist-2",
              from_status: "SUBMITTED",
              to_status: "REJECTED",
              actor_id: "usr-fin",
              reason: "Counterparty rejected",
              created_at: "2026-11-17T01:06:00Z",
            },
          ],
          vat_reversed_minor: 1200000,
          gl_reversal_entry_id: "gl-rev-77",
        };
      }
      return {};
    });

    render(<FiscalEsfConsolePage />);

    await waitFor(() => {
      expect(screen.getAllByText("esf-out-1").length).toBeGreaterThan(0);
      expect(screen.getByText("DiffMinor == 0")).toBeInTheDocument();
      expect(screen.getByText("Signed via EDS")).toBeInTheDocument();
    });

    // Resend ESF action (returns EsfDocument directly and refreshes timeline)
    const resendBtns = screen.getAllByRole("button", { name: "Resend ESF" });
    fireEvent.click(resendBtns[0]);
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/fiscal/esf/esf-out-1/resend",
        "POST",
        expect.any(Object),
      );
    });

    // Reject status transition + GL/VAT reversal display
    fireEvent.click(screen.getByRole("button", { name: "Mark rejected" }));
    await waitFor(() => {
      const reversalEl = screen.getByText(/gl-rev-77/);
      expect(reversalEl).toBeInTheDocument();
      expect(reversalEl).toHaveTextContent("12,000 UZS");
    });

    // Monthly VAT Summary tab
    fireEvent.click(screen.getByRole("tab", { name: "Monthly VAT summary" }));
    expect(screen.getByText("Output VAT (TAX:VAT:OUTPUT)")).toBeInTheDocument();
    expect(screen.getByText("12%")).toBeInTheDocument();

    // Missing IKPU tab
    fireEvent.click(screen.getByRole("tab", { name: "Missing IKPU / MXIK" }));
    expect(screen.getByText("Sparkling Water 1L")).toBeInTheDocument();
    expect(screen.getByText("ikpu_code")).toBeInTheDocument();
    expect(screen.getByText("package_code")).toBeInTheDocument();
  });

  it("registers Wave 3A nav routes, preserves i18n parity across en/ru/uz, wraps useSearchParams in Suspense, and imports zero @pegasusx/ui", () => {
    const hrefs = allNavHrefs(NAV);
    expect(hrefs).toContain("/finance/close");
    expect(hrefs).toContain("/approvals");
    expect(hrefs).toContain("/fiscal");

    const requiredKeys = [
      "portal.nav.period_close",
      "portal.nav.approvals",
      "portal.nav.fiscal_esf",
      "supplier_portal.finance.close.title",
      "supplier_portal.approvals.title",
      "supplier_portal.fiscal.title",
      "supplier_portal.fiscal.diff_minor_zero_badge",
    ];
    for (const key of requiredKeys) {
      expect(messageCatalogs.en[key], key).toBeTruthy();
      expect(messageCatalogs.ru[key], key).toBeTruthy();
      expect(messageCatalogs.uz[key], key).toBeTruthy();
    }

    const filesToCheck = [
      "lib/control-accounting-api.ts",
      "app/(portal)/finance/close/page.tsx",
      "app/(portal)/approvals/page.tsx",
      "app/(portal)/fiscal/page.tsx",
    ];
    for (const rel of filesToCheck) {
      const fullPath = join(portalRoot, rel);
      expect(existsSync(fullPath), rel).toBe(true);
      const src = readFileSync(fullPath, "utf8");
      expect(src).not.toContain("@pegasusx/ui");
    }

    for (const pageRel of [
      "app/(portal)/finance/close/page.tsx",
      "app/(portal)/approvals/page.tsx",
      "app/(portal)/fiscal/page.tsx",
    ]) {
      const src = readFileSync(join(portalRoot, pageRel), "utf8");
      expect(src, pageRel).toContain("<Suspense");
    }
  });
});
