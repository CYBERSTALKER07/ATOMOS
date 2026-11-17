import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";

const { mockSupplierFetch, mockListPlaybookRuns, mockListScoredExceptions } = vi.hoisted(() => ({
  mockSupplierFetch: vi.fn(),
  mockListPlaybookRuns: vi.fn(),
  mockListScoredExceptions: vi.fn(),
}));

vi.mock("@/lib/auth", () => ({
  supplierFetch: mockSupplierFetch,
  getSupplierToken: () => "",
  readTokenFromCookie: () => "",
  decodeJwtPayload: () => null,
  clearSession: vi.fn(),
}));

vi.mock("@/lib/api", () => ({
  createSupplierApi: () => ({
    listPlaybookRuns: mockListPlaybookRuns,
    listScoredExceptions: mockListScoredExceptions,
    approvePlaybookRun: vi.fn(),
    skipPlaybookRun: vi.fn(),
  }),
}));

vi.mock("next/link", () => ({
  default: ({ children, href }: { children: ReactNode; href: string }) => <a href={href}>{children}</a>,
}));

vi.mock("@/components/PageChrome", () => ({
  PageChrome: ({
    title,
    error,
    empty,
    emptyMessage,
    children,
  }: {
    title: string;
    error?: string | null;
    empty?: boolean;
    emptyMessage?: string;
    children: ReactNode;
  }) => (
    <div>
      <h1>{title}</h1>
      {error ? <p>{error}</p> : null}
      {empty ? <p>{emptyMessage}</p> : children}
    </div>
  ),
}));

vi.mock("@/components/StatusBadge", () => ({
  default: ({ state }: { state: string }) => <span>{state}</span>,
}));

vi.mock("@/components/CreditEnableModal", () => ({
  CreditEnableModal: () => null,
}));

import { getServerSnapshot } from "@/lib/i18n";
import { NAV, allNavHrefs } from "@/lib/nav";
import { buildNavForWorkspaceMode, resolveDefaultWorkspaceMode } from "@/lib/workspace-mode";
import {
  formatWholeSom,
  resolveHandlerGroup,
  sortExceptionsByRisk,
  wholeSomToMinor,
} from "@/lib/exception-desk";
import { PlaybookRunsPanel } from "@/components/exceptions/PlaybookRunsPanel";
import { ScoredExceptionsPanel } from "@/components/control-tower/ScoredExceptionsPanel";
import ExceptionsPage from "@/app/(portal)/exceptions/page";
import CollectionsPage from "@/app/(portal)/credit/collections/page";
import CreditPolicyPage from "@/app/(portal)/credit/policy/page";

function jsonResponse(payload: unknown, ok = true, status = 200) {
  return {
    ok,
    status,
    json: async () => payload,
  };
}

describe("W5 Supplier Portal Exception Desk, Workspace & Collections", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockListPlaybookRuns.mockResolvedValue({ runs: [] });
    mockListScoredExceptions.mockResolvedValue({ exceptions: [] });
  });

  it("defaults SSR locale snapshot to uz and role-defaults Fast Desk for Exception Lead / Finance Closer", () => {
    expect(getServerSnapshot()).toBe("uz");
    expect(resolveDefaultWorkspaceMode({ role: "EXCEPTION_LEAD" })).toBe("fast_desk");
    expect(resolveDefaultWorkspaceMode({ workspace_role: "FINANCE_CLOSER" })).toBe("fast_desk");
    expect(resolveDefaultWorkspaceMode({ role: "SUPPLIER_ADMIN" })).toBe("full_erp");

    const fastNav = buildNavForWorkspaceMode(NAV, "fast_desk");
    expect(fastNav[0].items.slice(0, 3).map((i) => i.href)).toEqual([
      "/exceptions",
      "/credit/collections",
      "/orders",
    ]);
    const flattened = new Set(fastNav.flatMap((s) => s.items.map((i) => i.href)));
    for (const href of allNavHrefs()) {
      expect(flattened.has(href)).toBe(true);
    }
  });

  it("removes developer env var hints from PlaybookRunsPanel and ScoredExceptionsPanel", async () => {
    mockListPlaybookRuns.mockRejectedValue(new Error("disabled"));
    mockListScoredExceptions.mockRejectedValue(new Error("disabled"));

    const { container: c1 } = render(<PlaybookRunsPanel />);
    const { container: c2 } = render(<ScoredExceptionsPanel />);

    await waitFor(() => {
      expect(c1.textContent).toContain("Automated playbook suggestions are currently unavailable");
      expect(c2.textContent).toContain("Priority scoring is currently unavailable");
    });
    expect(c1.textContent).not.toContain("CONTROL_TOWER_PLAYBOOKS_ENABLED");
    expect(c2.textContent).not.toContain("CONTROL_TOWER_PLAYBOOKS_ENABLED");
  });

  it("groups & sorts 6-handler exceptions by severity & money-at-risk in whole so'm and executes batch-resolve", async () => {
    expect(formatWholeSom(250_000_00)).toBe("250,000 so'm");
    expect(wholeSomToMinor("5000000")).toBe(500_000_000);
    expect(resolveHandlerGroup({ kind: "EXPORT_REJECTED" })).toBe("FISCAL_AND_EXPORT_BLOCKERS");
    expect(resolveHandlerGroup({ kind: "TIN_DEDUP_CANDIDATE" })).toBe("FULFILLMENT_AND_CATALOG");

    const sorted = sortExceptionsByRisk([
      {
        order_id: "ord-low",
        kind: "ORDER_EXCEPTION",
        status: "OPEN",
        severity_tier: "LOW",
        money_at_risk_minor: 900_000_00,
        updated_at: "2026-11-12T10:00:00Z",
      },
      {
        order_id: "ord-crit-small",
        kind: "EXPORT_REJECTED",
        status: "OPEN",
        severity_tier: "CRITICAL",
        money_at_risk_minor: 100_000_00,
        updated_at: "2026-11-12T10:00:00Z",
      },
      {
        order_id: "ord-crit-big",
        kind: "FISCAL_FAILED",
        status: "OPEN",
        severity_tier: "CRITICAL",
        money_at_risk_minor: 500_000_00,
        updated_at: "2026-11-12T10:00:00Z",
      },
    ]);
    expect(sorted.map((r) => r.order_id)).toEqual(["ord-crit-big", "ord-crit-small", "ord-low"]);

    const rows = [
      {
        order_id: "post-101",
        posting_id: "post-101",
        kind: "EXPORT_REJECTED",
        status: "OPEN",
        handler_group: "FISCAL_AND_EXPORT_BLOCKERS",
        severity_tier: "CRITICAL",
        money_at_risk_minor: 450_000_00,
        display_name: "Olmazor Mini Market",
        tin: "305998877",
        phone: "+998901112233",
        retailer_id: "ret-1",
        updated_at: "2026-11-12T10:00:00Z",
      },
      {
        order_id: "dedup-7",
        dedup_cluster_id: "cl-7",
        kind: "TIN_DEDUP_CANDIDATE",
        status: "OPEN",
        handler_group: "FULFILLMENT_AND_CATALOG",
        severity_tier: "HIGH",
        money_at_risk_minor: 120_000_00,
        display_name: "Baraka Savdo",
        tin: "309112233",
        phone: "+998909998877",
        retailer_id: "ret-7",
        updated_at: "2026-11-12T09:00:00Z",
      },
    ];

    mockSupplierFetch.mockImplementation(async (url: string, init?: RequestInit) => {
      if (url.startsWith("/v1/supplier/exceptions") && (!init || !init.method || init.method === "GET")) {
        return jsonResponse({ exceptions: rows });
      }
      if (url === "/v1/supplier/exceptions/batch-resolve") {
        return jsonResponse({ resolved: 2, failed: [], results: [] });
      }
      return jsonResponse({ status: "resolved" });
    });

    render(<ExceptionsPage />);

    await waitFor(() => {
      expect(screen.getByText("Olmazor Mini Market")).toBeInTheDocument();
    });

    expect(screen.getByText(/TIN 305998877/)).toBeInTheDocument();
    expect(screen.getByText("+998901112233")).toBeInTheDocument();
    expect(screen.getByText("450,000 so'm")).toBeInTheDocument();
    expect(screen.getByText("120,000 so'm")).toBeInTheDocument();
    expect(screen.getByText("Retry 1C Export")).toBeInTheDocument();
    expect(screen.getByText("Merge Duplicate")).toBeInTheDocument();

    fireEvent.click(screen.getByLabelText("Select all visible exceptions"));
    fireEvent.change(screen.getByLabelText("Batch resolution note"), {
      target: { value: "morning triage" },
    });
    fireEvent.click(screen.getByText("Batch Resolve (2)"));

    await waitFor(() => {
      const batchCall = mockSupplierFetch.mock.calls.find(
        (call) => call[0] === "/v1/supplier/exceptions/batch-resolve",
      );
      expect(batchCall).toBeDefined();
      const body = JSON.parse(String((batchCall?.[1] as RequestInit).body));
      expect(body.items).toHaveLength(2);
      expect(body.items[0]).toMatchObject({
        kind: "EXPORT_REJECTED",
        id: "post-101",
        action: "retry",
        note: "morning triage",
      });
    });
  });

  it("hides Pending Finance Approval when draft count is 0, shows shop metadata, and converts whole so'm limits on save", async () => {
    mockSupplierFetch.mockImplementation(async (url: string, init?: RequestInit) => {
      if (url.startsWith("/v1/supplier/credit-profiles")) {
        return jsonResponse({
          pending_credit_note_count: 0,
          profiles: [
            {
              retailer_id: "ret-55",
              supplier_id: "sup-1",
              display_name: "Chilonzor Super",
              tin: "301234567",
              phone: "+998901234567",
              credit_limit_minor: 500_000_000, // 5,000,000 so'm
              current_balance_minor: 150_000_000, // 1,500,000 so'm
              available_credit_minor: 350_000_000,
              status: "ACTIVE",
              pending_credit_note_count: 0,
            },
          ],
        });
      }
      if (url === "/v1/supplier/retailer-credit-profile" && init?.method === "PATCH") {
        return jsonResponse({ ok: true });
      }
      return jsonResponse({});
    });

    render(<CollectionsPage />);

    await waitFor(() => {
      expect(screen.getByText("Chilonzor Super")).toBeInTheDocument();
    });

    expect(screen.queryByText(/Pending Finance Approval/)).not.toBeInTheDocument();
    expect(screen.getByText("TIN 301234567")).toBeInTheDocument();
    expect(screen.getByText("+998901234567")).toBeInTheDocument();
    expect(screen.getByText("5,000,000 so'm")).toBeInTheDocument();
    expect(screen.getByText("1,500,000 so'm")).toBeInTheDocument();

    fireEvent.click(screen.getByText("Set limit"));
    const input = screen.getByLabelText("Credit limit (so'm)") as HTMLInputElement;
    expect(input.value).toBe("5000000");

    fireEvent.change(input, { target: { value: "7500000" } });
    fireEvent.click(screen.getByText("Save"));

    await waitFor(() => {
      const patchCall = mockSupplierFetch.mock.calls.find(
        (call) => call[0] === "/v1/supplier/retailer-credit-profile",
      );
      expect(patchCall).toBeDefined();
      const body = JSON.parse(String((patchCall?.[1] as RequestInit).body));
      expect(body.credit_limit_minor).toBe(750_000_000);
    });
  });

  it("shows Pending Finance Approval only when pending_credit_note_count > 0", async () => {
    mockSupplierFetch.mockResolvedValue(
      jsonResponse({
        pending_credit_note_count: 3,
        profiles: [],
      }),
    );

    render(<CollectionsPage />);

    await waitFor(() => {
      expect(screen.getByText("Pending Finance Approval (3)")).toBeInTheDocument();
    });
  });

  it("converts Credit Policy defaults input and table cells to whole so'm while preserving wire minor fields", async () => {
    mockSupplierFetch.mockImplementation(async (url: string, init?: RequestInit) => {
      if (url === "/v1/supplier/credit-program") {
        return jsonResponse({
          supplier_id: "sup-1",
          program_enabled: true,
          global_terms_days: 14,
          global_grace_days: 3,
          global_default_limit_minor: 200_000_000, // 2,000,000 so'm
        });
      }
      if (url === "/v1/supplier/credit-relationships") {
        return jsonResponse({
          relationships: [
            {
              retailer_id: "ret-9",
              display_name: "Yunusobod Do'kon",
              tin: "308765432",
              credit_enabled: true,
              terms_days: 14,
              grace_period_days: 3,
              credit_limit_minor: 300_000_000, // 3,000,000 so'm
              current_balance_minor: 75_000_000, // 750,000 so'm
              use_global_defaults: false,
              profile_status: "ACTIVE",
            },
          ],
        });
      }
      if (url === "/v1/supplier/credit-program/defaults" && init?.method === "PATCH") {
        return jsonResponse({ ok: true });
      }
      return jsonResponse({});
    });

    render(<CreditPolicyPage />);

    await waitFor(() => {
      expect(screen.getByText("Yunusobod Do'kon")).toBeInTheDocument();
    });

    expect(screen.getByText(/default limit 2,000,000 so'm/)).toBeInTheDocument();
    expect(screen.getByText("3,000,000 so'm")).toBeInTheDocument();
    expect(screen.getByText("750,000 so'm")).toBeInTheDocument();

    const defaultInput = screen.getByLabelText("Default limit (so'm)") as HTMLInputElement;
    expect(defaultInput.value).toBe("2000000");

    fireEvent.change(defaultInput, { target: { value: "4500000" } });
    fireEvent.click(screen.getByText("Save defaults"));

    await waitFor(() => {
      const patchCall = mockSupplierFetch.mock.calls.find(
        (call) => call[0] === "/v1/supplier/credit-program/defaults",
      );
      expect(patchCall).toBeDefined();
      const body = JSON.parse(String((patchCall?.[1] as RequestInit).body));
      expect(body.global_default_limit_minor).toBe(450_000_000);
    });
  });
});
