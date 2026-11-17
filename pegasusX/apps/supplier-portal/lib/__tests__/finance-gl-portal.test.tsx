import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { ReactNode } from "react";
import { ApiClient } from "@pegasusx/api-core";
import { messageCatalogs } from "@pegasusx/i18n";

const {
  mockRequest,
  mockGetPaymentLedger,
  mockGetPaymentSettlementAuthority,
  mockGetPaymentReconciliationMismatches,
  mockSearchParams,
} = vi.hoisted(() => ({
  mockRequest: vi.fn(),
  mockGetPaymentLedger: vi.fn(),
  mockGetPaymentSettlementAuthority: vi.fn(),
  mockGetPaymentReconciliationMismatches: vi.fn(),
  mockSearchParams: new URLSearchParams(),
}));

vi.mock("@/lib/api", () => ({
  createSupplierApi: () => ({
    request: mockRequest,
    getPaymentLedger: mockGetPaymentLedger,
    getPaymentSettlementAuthority: mockGetPaymentSettlementAuthority,
    getPaymentReconciliationMismatches: mockGetPaymentReconciliationMismatches,
  }),
}));

vi.mock("@/lib/i18n", () => ({
  usePortalT: () => (key: string, values?: Record<string, string | number>) => {
    if (!values) return key;
    return `${key}:${JSON.stringify(values)}`;
  },
}));

vi.mock("@/lib/use-supplier-session-reconcile", () => ({
  useSupplierSessionReconcile: () => {},
}));

vi.mock("next/navigation", () => ({
  useSearchParams: () => mockSearchParams,
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
  default: ({ state }: { state: string }) => <span>{state}</span>,
}));

vi.mock("@/components/Icon", () => ({
  default: ({ name }: { name: string }) => <span data-icon={name} />,
}));

import {
  createSupplierFinanceApi,
  evaluateManualJournalLines,
  formatFinanceMinor,
} from "@/lib/finance-api";
import { NAV, allNavHrefs } from "@/lib/nav";
import TrialBalancePage from "@/app/(portal)/finance/trial-balance/page";
import GlLedgerPage from "@/app/(portal)/finance/ledger/page";
import FourWayReconciliationPage from "@/app/(portal)/finance/reconciliation/page";
import ReconciliationPage from "@/app/(portal)/reconciliation/page";

describe("M0.3 Supplier Portal Finance API, Navigation & BigInt Money Helpers", () => {
  it("formats minor units cleanly via BigInt without tiyin inflation or float rounding", () => {
    expect(formatFinanceMinor(142500000, "UZS")).toBe("1,425,000 UZS");
    expect(formatFinanceMinor(142500050, "UZS")).toBe("1,425,000.50 UZS");
    expect(formatFinanceMinor(-2500000, "UZS", { signDisplay: "always" })).toBe("-25,000 UZS");
    expect(formatFinanceMinor(2500000, "UZS", { signDisplay: "always" })).toBe("+25,000 UZS");
    expect(formatFinanceMinor(0, "UZS")).toBe("0 UZS");
    expect(formatFinanceMinor(BigInt("900719925474099300"), "UZS")).toBe("9,007,199,254,740,993 UZS");
  });

  it("evaluates balanced manual journal lines strictly in BigInt minor units", () => {
    const unbalanced = evaluateManualJournalLines(
      [
        { account_code: "CASH:OFFICE", direction: "DEBIT", amount_major: "1500.50" },
        { account_code: "EQUITY:OPENING_BALANCE", direction: "CREDIT", amount_major: "1500.40" },
      ],
      "UZS",
    );
    expect(unbalanced.balanced).toBe(false);
    expect(unbalanced.debitMinor).toBe(BigInt(150050));
    expect(unbalanced.creditMinor).toBe(BigInt(150040));

    const negative = evaluateManualJournalLines(
      [
        { account_code: "CASH:OFFICE", direction: "DEBIT", amount_major: "-10" },
        { account_code: "EQUITY:OPENING_BALANCE", direction: "CREDIT", amount_major: "-10" },
      ],
      "UZS",
    );
    expect(negative.balanced).toBe(false);
    expect(negative.errorLineIndex).toBe(0);

    const balanced = evaluateManualJournalLines(
      [
        { account_code: "CASH:OFFICE", direction: "DEBIT", amount_major: "19.99" },
        { account_code: "EQUITY:OPENING_BALANCE", direction: "CREDIT", amount_major: "19.99" },
      ],
      "UZS",
    );
    expect(balanced.balanced).toBe(true);
    expect(balanced.debitMinor).toBe(BigInt(1999));
    expect(balanced.validLines[0].amount_minor).toBe(1999);
    expect(balanced.validLines[1].amount_minor).toBe(1999);
  });

  it("wires real M0.2 GL and M0.3 Recon URLs, query params, and idempotency keys via ApiClient", async () => {
    const fetchSpy = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      return new Response(JSON.stringify({ ok: true, url: String(input), method: init?.method }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });

    const realClient = new ApiClient({
      baseUrl: "https://api.pegasusx.test",
      getAuthToken: () => "tok-123",
      fetchImpl: fetchSpy as unknown as typeof fetch,
    });
    const financeApi = createSupplierFinanceApi(realClient);

    await financeApi.getTrialBalance({ period: "2026-11" });
    await financeApi.getAccountLedger("AR:RETAILER", { period: "2026-11", limit: 25 });
    await financeApi.getJournalEntry("gle_abc");
    await financeApi.postManualJournal(
      {
        source_id: "man-1",
        memo: "Test",
        currency: "UZS",
        postings: [
          { account_code: "CASH:OFFICE", direction: "DEBIT", amount_minor: 10000 },
          { account_code: "EQUITY:OPENING_BALANCE", direction: "CREDIT", amount_minor: 10000 },
        ],
      },
      "idem-manual-1",
    );
    await financeApi.reverseJournalEntry("gle_abc", { reason: "oops" }, "idem-rev-1");
    await financeApi.listReconRuns({ period_id: "2026-11", limit: 10 });
    await financeApi.triggerReconRun({ period_id: "2026-11", mode: "MANUAL" }, "idem-run-1");
    await financeApi.listReconBreaks({ dimension: "STOCK_GL", status: "OPEN", limit: 50 });
    await financeApi.updateReconBreak("brk-1", { action: "RESOLVE", note: "fixed" }, {}, "idem-brk-1");

    const calls = fetchSpy.mock.calls.map(([u, init]) => ({
      url: String(u),
      method: init?.method,
      headers: new Headers(init?.headers),
    }));

    expect(calls[0].url).toBe("https://api.pegasusx.test/v1/finance/gl/trial-balance?period=2026-11");
    expect(calls[0].method).toBe("GET");
    expect(calls[0].headers.get("Authorization")).toBe("Bearer tok-123");

    expect(calls[1].url).toBe("https://api.pegasusx.test/v1/finance/gl/accounts/AR%3ARETAILER/ledger?period=2026-11&limit=25");
    expect(calls[2].url).toBe("https://api.pegasusx.test/v1/finance/journal/gle_abc");

    expect(calls[3].url).toBe("https://api.pegasusx.test/v1/finance/journal/manual");
    expect(calls[3].method).toBe("POST");
    expect(calls[3].headers.get("Idempotency-Key")).toBe("idem-manual-1");

    expect(calls[4].url).toBe("https://api.pegasusx.test/v1/finance/journal/gle_abc/reverse");
    expect(calls[4].method).toBe("POST");
    expect(calls[4].headers.get("Idempotency-Key")).toBe("idem-rev-1");

    expect(calls[5].url).toBe("https://api.pegasusx.test/v1/finance/recon/runs?period_id=2026-11&limit=10");
    expect(calls[6].url).toBe("https://api.pegasusx.test/v1/finance/recon/run");
    expect(calls[6].method).toBe("POST");
    expect(calls[7].url).toBe("https://api.pegasusx.test/v1/finance/recon/breaks?dimension=STOCK_GL&status=OPEN&limit=50");
    expect(calls[8].url).toBe("https://api.pegasusx.test/v1/finance/recon/breaks/brk-1");
    expect(calls[8].method).toBe("PATCH");
    expect(calls[8].headers.get("Idempotency-Key")).toBe("idem-brk-1");
  });

  it("registers new finance navigation routes with complete en/ru/uz i18n keys", () => {
    const hrefs = allNavHrefs(NAV);
    expect(hrefs).toContain("/finance/trial-balance");
    expect(hrefs).toContain("/finance/ledger");
    expect(hrefs).toContain("/finance/reconciliation");
    expect(hrefs).toContain("/reconciliation");

    for (const key of [
      "portal.nav.trial_balance",
      "portal.nav.gl_ledger",
      "portal.nav.four_way_reconciliation",
      "supplier_portal.finance.trial_balance.title",
      "supplier_portal.finance.gl_ledger.title",
      "supplier_portal.finance.four_way_recon.title",
      "supplier_portal.finance.four_way_recon.waive_note_required",
    ]) {
      expect(messageCatalogs.en[key]).toBeTruthy();
      expect(messageCatalogs.ru[key]).toBeTruthy();
      expect(messageCatalogs.uz[key]).toBeTruthy();
    }
  });
});

describe("M0.3 Supplier Portal Finance Pages", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockSearchParams.forEach((_, key) => mockSearchParams.delete(key));
  });

  it("renders TrialBalancePage with BigInt-formatted balances, filters, and ledger drill-down", async () => {
    mockRequest.mockResolvedValue({
      entity_id: "sup-1",
      period_id: "2026-11",
      currency: "UZS",
      total_debits_minor: 150000000,
      total_credits_minor: 150000000,
      balanced: true,
      accounts: [
        {
          account_code: "AR:RETAILER",
          account_name: "Accounts Receivable - Retailers",
          class: "ASSET",
          currency: "UZS",
          debit_minor: 150000000,
          credit_minor: 0,
          net_minor: 150000000,
        },
        {
          account_code: "REV:SALES",
          account_name: "Merchandise Sales Revenue",
          class: "REVENUE",
          currency: "UZS",
          debit_minor: 0,
          credit_minor: 150000000,
          net_minor: 150000000,
        },
      ],
    });

    render(<TrialBalancePage />);

    await waitFor(() => {
      expect(screen.getByText("AR:RETAILER")).toBeInTheDocument();
    });
    expect(screen.getByText("REV:SALES")).toBeInTheDocument();
    expect(screen.getAllByText("1,500,000 UZS").length).toBeGreaterThanOrEqual(2);
    expect(screen.getByText("supplier_portal.finance.trial_balance.balanced_badge")).toBeInTheDocument();

    const ledgerLink = screen.getAllByRole("link", {
      name: /supplier_portal\.finance\.trial_balance\.open_ledger/,
    })[0];
    expect(ledgerLink).toHaveAttribute("href", "/finance/ledger?account=AR%3ARETAILER&period=2026-11");

    fireEvent.change(screen.getByLabelText("supplier_portal.finance.trial_balance.class_filter"), {
      target: { value: "REVENUE" },
    });
    expect(screen.queryByText("AR:RETAILER")).not.toBeInTheDocument();
    expect(screen.getByText("REV:SALES")).toBeInTheDocument();
  });

  it("shows honest API error state on TrialBalancePage without mock fallback", async () => {
    mockRequest.mockRejectedValue(new Error("spanner unavailable"));

    render(<TrialBalancePage />);

    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("spanner unavailable");
    });
  });

  it("renders GlLedgerPage, inspects entry hashes, reverses entry, and posts balanced manual journal", async () => {
    mockRequest.mockImplementation(async (path: string, method: string, options?: { body?: unknown }) => {
      if (path.startsWith("/v1/finance/gl/accounts/") && method === "GET") {
        return {
          entity_id: "sup-1",
          account_code: "AR:RETAILER",
          period_id: "2026-11",
          limit: 50,
          count: 1,
          items: [
            {
              entry_id: "gle_100",
              period_id: "2026-11",
              line: 1,
              account_code: "AR:RETAILER",
              direction: "DEBIT",
              amount_minor: 50000000,
              currency: "UZS",
              functional_amount_minor: 50000000,
              source_type: "ORDER",
              source_id: "ord-1",
              memo: "Nasiya delivery",
              entry_hash: "abcdef1234567890",
              occurred_at: "2026-11-16T10:00:00Z",
              posted_at: "2026-11-16T10:00:01Z",
            },
          ],
        };
      }
      if (path.startsWith("/v1/finance/journal/gle_") && method === "GET") {
        return {
          entity_id: "sup-1",
          entry_id: "gle_100",
          period_id: "2026-11",
          source_type: "ORDER",
          source_id: "ord-1",
          memo: "Nasiya delivery",
          currency: "UZS",
          country_pack: "UZ",
          idempotency_key: "gl:order:ord-1",
          prev_hash: "0000000000000000",
          entry_hash: "abcdef1234567890",
          posted_by: "system",
          occurred_at: "2026-11-16T10:00:00Z",
          posted_at: "2026-11-16T10:00:01Z",
          postings: [
            {
              entity_id: "sup-1",
              entry_id: "gle_100",
              line: 1,
              account_code: "AR:RETAILER",
              direction: "DEBIT",
              amount_minor: 50000000,
              currency: "UZS",
              functional_amount_minor: 50000000,
            },
            {
              entity_id: "sup-1",
              entry_id: "gle_100",
              line: 2,
              account_code: "REV:SALES",
              direction: "CREDIT",
              amount_minor: 50000000,
              currency: "UZS",
              functional_amount_minor: 50000000,
            },
          ],
        };
      }
      if (path === "/v1/finance/journal/gle_100/reverse" && method === "POST") {
        return {
          entry_id: "gle_rev_100",
          entity_id: "sup-1",
          period_id: "2026-11",
          prev_hash: "abcdef1234567890",
          hash: "fedcba0987654321",
          already_posted: false,
          total_debit_minor: 50000000,
          total_credit_minor: 50000000,
          line_count: 2,
        };
      }
      if (path === "/v1/finance/journal/manual" && method === "POST") {
        expect((options?.body as { postings: Array<{ amount_minor: number }> }).postings[0].amount_minor).toBe(125000);
        return {
          entry_id: "gle_manual_1",
          entity_id: "sup-1",
          period_id: "2026-11",
          prev_hash: "fedcba0987654321",
          hash: "1122334455667788",
          already_posted: false,
          total_debit_minor: 125000,
          total_credit_minor: 125000,
          line_count: 2,
        };
      }
      return {};
    });

    render(<GlLedgerPage />);

    await waitFor(() => {
      expect(screen.getByText("gle_100")).toBeInTheDocument();
    });
    expect(screen.getByText("500,000 UZS")).toBeInTheDocument();

    // Inspect entry
    fireEvent.click(screen.getByRole("button", { name: "supplier_portal.finance.gl_ledger.inspect_entry" }));
    await waitFor(() => {
      expect(screen.getByText("0000000000000000")).toBeInTheDocument();
    });

    // Reverse entry
    fireEvent.change(
      screen.getByLabelText("supplier_portal.finance.gl_ledger.reverse_reason_placeholder"),
      { target: { value: "duplicate entry" } },
    );
    fireEvent.click(screen.getByRole("button", { name: "supplier_portal.finance.gl_ledger.reverse_button" }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/finance/journal/gle_100/reverse",
        "POST",
        expect.objectContaining({
          body: expect.objectContaining({ reason: "duplicate entry" }),
        }),
      );
    });

    // Open manual form and post balanced entry
    fireEvent.click(screen.getByRole("button", { name: "supplier_portal.finance.gl_ledger.post_manual_button" }));
    const submitBtn = screen.getByRole("button", { name: "supplier_portal.finance.gl_ledger.submit_manual" });
    expect(submitBtn).toBeDisabled();

    fireEvent.change(screen.getByLabelText("supplier_portal.finance.gl_ledger.memo_label"), {
      target: { value: "Petty cash opening" },
    });
    const amtInputs = screen.getAllByLabelText("supplier_portal.finance.gl_ledger.amount_major_label");
    fireEvent.change(amtInputs[0], { target: { value: "1250" } });
    fireEvent.change(amtInputs[1], { target: { value: "1250" } });

    expect(submitBtn).not.toBeDisabled();
    fireEvent.click(submitBtn);

    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/finance/journal/manual",
        "POST",
        expect.objectContaining({
          body: expect.objectContaining({
            memo: "Petty cash opening",
            currency: "UZS",
          }),
        }),
      );
    });
  });

  it("renders FourWayReconciliationPage and upgraded ReconciliationPage with real 4-way runs & break actions", async () => {
    const sampleRun = {
      run_id: "run_01",
      entity_id: "sup-1",
      period_id: "2026-11",
      mode: "MANUAL",
      status: "BREAKS_FOUND",
      break_count: 1,
      exposure_minor: 2500000,
      currency: "UZS",
      checks: {
        ORDERS_AR: {
          dimension: "ORDERS_AR",
          status: "GREEN",
          expected_minor: 10000000,
          actual_minor: 10000000,
          delta_minor: 0,
          break_count: 0,
          exposure_minor: 0,
          currency: "UZS",
        },
        STOCK_GL: {
          dimension: "STOCK_GL",
          status: "BREAK",
          expected_minor: 50000000,
          actual_minor: 47500000,
          delta_minor: -2500000,
          break_count: 1,
          exposure_minor: 2500000,
          currency: "UZS",
        },
        CASH_PSP: {
          dimension: "CASH_PSP",
          status: "GREEN",
          expected_minor: 20000000,
          actual_minor: 20000000,
          delta_minor: 0,
          break_count: 0,
          exposure_minor: 0,
          currency: "UZS",
        },
        TAX_FISCAL: {
          dimension: "TAX_FISCAL",
          status: "GREEN",
          expected_minor: 3000000,
          actual_minor: 3000000,
          delta_minor: 0,
          break_count: 0,
          exposure_minor: 0,
          currency: "UZS",
        },
      },
      triggered_by: "admin-1",
      started_at: "2026-11-16T12:00:00Z",
    };

    const sampleBreak = {
      break_id: "brk_01",
      run_id: "run_01",
      entity_id: "sup-1",
      dimension: "STOCK_GL",
      object_type: "STOCK_LOT",
      object_id: "wh-1:sku-1",
      expected_minor: 50000000,
      actual_minor: 47500000,
      delta_minor: -2500000,
      currency: "UZS",
      severity: "HIGH",
      status: "OPEN",
      control_tower_exception_id: "exc_01",
      created_at: "2026-11-16T12:00:00Z",
      updated_at: "2026-11-16T12:00:00Z",
    };

    mockRequest.mockImplementation(async (path: string, method: string) => {
      if (path.startsWith("/v1/finance/recon/runs") && method === "GET") {
        return { entity_id: "sup-1", runs: [sampleRun], count: 1 };
      }
      if (path.startsWith("/v1/finance/recon/breaks?") && method === "GET") {
        return { entity_id: "sup-1", breaks: [sampleBreak], count: 1 };
      }
      if (path === "/v1/finance/recon/run" && method === "POST") {
        return { run: sampleRun, breaks: [sampleBreak] };
      }
      if (path === "/v1/finance/recon/breaks/brk_01" && method === "PATCH") {
        return { break: { ...sampleBreak, status: "RESOLVED", resolution_note: "repaired" } };
      }
      return {};
    });

    const { unmount } = render(<FourWayReconciliationPage />);

    await waitFor(() => {
      expect(screen.getByText("brk_01")).toBeInTheDocument();
    });
    expect(screen.getAllByText("-25,000 UZS").length).toBeGreaterThanOrEqual(2);

    // Waive without note must reject locally before calling PATCH
    fireEvent.click(screen.getByRole("button", { name: "supplier_portal.finance.four_way_recon.action_waive" }));
    await waitFor(() => {
      expect(screen.getByRole("alert")).toHaveTextContent("supplier_portal.finance.four_way_recon.waive_note_required");
    });

    // Resolve with note calls PATCH
    fireEvent.change(screen.getByLabelText("Note for brk_01"), { target: { value: "repaired" } });
    fireEvent.click(screen.getByRole("button", { name: "supplier_portal.finance.four_way_recon.action_resolve" }));
    await waitFor(() => {
      expect(mockRequest).toHaveBeenCalledWith(
        "/v1/finance/recon/breaks/brk_01",
        "PATCH",
        expect.objectContaining({
          body: expect.objectContaining({ action: "RESOLVE", note: "repaired" }),
        }),
      );
    });

    unmount();

    mockGetPaymentLedger.mockResolvedValue({ items: [], limit: 200 });
    mockGetPaymentSettlementAuthority.mockResolvedValue({
      items: [],
      count: 0,
      totals_by_currency: [{ currency: "UZS", entry_count: 1, amount_minor_total: 10000000 }],
      operating_currency: "UZS",
      operating_currency_total_minor: 10000000,
    });
    mockGetPaymentReconciliationMismatches.mockResolvedValue({ items: [] });

    render(<ReconciliationPage />);
    await waitFor(() => {
      expect(screen.getByText("STOCK_LOT:wh-1:sku-1")).toBeInTheDocument();
    });
    expect(screen.getByRole("link", { name: /portal\.nav\.four_way_reconciliation/ })).toHaveAttribute(
      "href",
      "/finance/reconciliation",
    );
    expect(screen.getByRole("link", { name: /portal\.nav\.trial_balance/ })).toHaveAttribute(
      "href",
      "/finance/trial-balance",
    );
  });
});
