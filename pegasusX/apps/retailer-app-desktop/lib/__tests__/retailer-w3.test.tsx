import fs from "node:fs";
import path from "node:path";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";

const { apiFetchMock } = vi.hoisted(() => ({
  apiFetchMock: vi.fn(),
}));

vi.mock("@/lib/auth", () => ({
  apiFetch: apiFetchMock,
}));

vi.mock("@/lib/i18n", () => ({
  usePortalLocale: () => "en",
  usePortalT: () => ({}),
}));

import CreditPage from "@/app/(dashboard)/credit/page";
import {
  createArPromise,
  getArInvoices,
  getArStatement,
  payArFifo,
  payArInvoice,
} from "@/lib/api";
import { formatWholeSom, somInputToMinor } from "@/lib/retailer-w2";
import {
  ArApiError,
  generateIdempotencyUuid,
  parseQuarterlyPromiseLimitError,
  sortArInvoicesOldestFirst,
  W3_CREDIT_COPY,
} from "@/lib/retailer-w3";

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("Retailer W3 Credit & AR Desktop", () => {
  beforeEach(() => {
    apiFetchMock.mockReset();
  });

  it("removes legacy mock disputed banner and raw tiyin display from Desktop credit files", () => {
    const creditPageSrc = fs.readFileSync(
      path.resolve(process.cwd(), "app/(dashboard)/credit/page.tsx"),
      "utf8",
    );
    const creditCardSrc = fs.readFileSync(
      path.resolve(process.cwd(), "components/CreditProfileCard.tsx"),
      "utf8",
    );

    expect(creditPageSrc).not.toContain(
      "There is 1 disputed delivery adjustment awaiting Finance review",
    );
    expect(creditPageSrc).not.toContain("@pegasusx/ui-kit");
    expect(creditPageSrc).toContain("formatWholeSom");
    expect(creditPageSrc).not.toMatch(
      /\b(available_credit_minor|current_balance_minor|credit_limit_minor|inv\.balance_minor)\.toLocaleString\(/,
    );
    expect(creditCardSrc).toContain("formatWholeSom");
    expect(creditCardSrc).not.toMatch(
      /\bprofile\.(credit_limit_minor|current_balance_minor|available_credit_minor)\.toLocaleString\(/,
    );
  });

  it("sorts invoices oldest-first and parses whole so'm integers safely", () => {
    const sorted = sortArInvoicesOldestFirst([
      {
        invoice_id: "inv-new",
        status: "OPEN",
        balance_minor: 500_000_00,
        due_at: "2026-11-20T00:00:00Z",
      },
      {
        invoice_id: "inv-old",
        status: "PARTIAL",
        balance_minor: 200_000_00,
        due_at: "2026-10-05T00:00:00Z",
      },
    ]);
    expect(sorted.map((i) => i.invoice_id)).toEqual(["inv-old", "inv-new"]);
    expect(somInputToMinor("500000")).toBe(50_000_000);
    expect(
      generateIdempotencyUuid(),
    ).toMatch(
      /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i,
    );
    expect(Object.keys(W3_CREDIT_COPY).sort()).toEqual(["en", "ru", "uz"]);
  });

  it("calls locked W3 endpoints with UUID Idempotency-Key and parses 409 quarterly promise conflict", async () => {
    apiFetchMock.mockResolvedValueOnce(jsonResponse({ invoices: [] }));
    await getArInvoices();
    expect(apiFetchMock).toHaveBeenCalledWith(
      "/v1/retailer/ar/invoices?status=OPEN%2CPARTIAL&limit=50",
    );

    apiFetchMock.mockResolvedValueOnce(
      jsonResponse({
        supplier_id: "sup-1",
        currency: "UZS",
        total_open_minor: 0,
        total_disputed_minor: 0,
        total_undisputed_minor: 0,
        advance_balance_minor: 0,
        current_quarter_key: "2026-Q4",
        self_serve_promise_used: false,
        max_extension_days: 14,
        invoices: [],
        promises: [],
        ledger_entries: [],
      }),
    );
    await getArStatement("sup-1", 50);
    expect(apiFetchMock).toHaveBeenCalledWith(
      "/v1/retailer/ar/statement?supplier_id=sup-1&limit=50",
    );

    const uuid = "11111111-2222-4333-8444-555555555555";
    apiFetchMock.mockResolvedValueOnce(
      jsonResponse({
        paid_minor: 50_000_000,
        advance_added_minor: 0,
        advance_balance_minor: 0,
        allocations: [],
      }),
    );
    await payArFifo(
      {
        supplier_id: "sup-1",
        amount_minor: 50_000_000,
        currency: "UZS",
        payment_method: "WALLET",
      },
      uuid,
    );
    expect(apiFetchMock).toHaveBeenLastCalledWith(
      "/v1/retailer/ar/pay-fifo",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({ "Idempotency-Key": uuid }),
      }),
    );

    apiFetchMock.mockResolvedValueOnce(jsonResponse({ ok: true }));
    await payArInvoice(
      "inv-1",
      { amount_minor: 10_000_000, payment_method: "CARD" },
      uuid,
    );
    expect(apiFetchMock).toHaveBeenLastCalledWith(
      "/v1/retailer/ar/invoices/inv-1/pay",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({ "Idempotency-Key": uuid }),
      }),
    );

    apiFetchMock.mockResolvedValueOnce(
      jsonResponse(
        {
          code: "quarterly_promise_limit_reached",
          error: "quarterly_promise_limit_reached",
          quarter_key: "2026-Q4",
          message: "Quarterly promise limit already reached",
        },
        409,
      ),
    );
    let caught: unknown = null;
    try {
      await createArPromise(
        { supplier_id: "sup-1", extension_days: 7 },
        uuid,
      );
    } catch (err) {
      caught = err;
    }
    expect(caught).toBeInstanceOf(ArApiError);
    expect(parseQuarterlyPromiseLimitError(caught)).toEqual({
      code: "quarterly_promise_limit_reached",
      quarterKey: "2026-Q4",
      message: "Quarterly promise limit already reached",
    });
  });

  it("renders CreditPage in whole so'm, submits FIFO pay, and displays 409 quarterly promise limit banner", async () => {
    const setupMockFetch = () => {
      apiFetchMock.mockImplementation(
        async (url: string, init?: RequestInit) => {
          if (url === "/v1/retailer/credit-relationships") {
            return jsonResponse({
              relationships: [
                {
                  supplier_id: "sup-uz",
                  supplier_name: "Baraka Dist",
                  credit_enabled: true,
                  terms_days: 14,
                  grace_period_days: 3,
                  credit_limit_minor: 1_000_000_00,
                  available_credit_minor: 400_000_00,
                  current_balance_minor: 600_000_00,
                  on_hold: false,
                  prepay_only_locked: false,
                },
              ],
            });
          }
          if (url.startsWith("/v1/retailer/ar/statement")) {
            return jsonResponse({
              supplier_id: "sup-uz",
              currency: "UZS",
              total_open_minor: 600_000_00,
              total_disputed_minor: 100_000_00,
              total_undisputed_minor: 500_000_00,
              advance_balance_minor: 50_000_00,
              current_quarter_key: "2026-Q4",
              self_serve_promise_used: false,
              max_extension_days: 14,
              invoices: [
                {
                  invoice_id: "inv-newer",
                  supplier_id: "sup-uz",
                  order_id: "ORD-2",
                  status: "OPEN",
                  balance_minor: 400_000_00,
                  disputed_minor: 0,
                  undisputed_minor: 400_000_00,
                  due_at: "2026-11-25T00:00:00Z",
                  aging_bucket: "CURRENT",
                },
                {
                  invoice_id: "inv-older",
                  supplier_id: "sup-uz",
                  order_id: "ORD-1",
                  status: "PARTIAL",
                  balance_minor: 200_000_00,
                  disputed_minor: 100_000_00,
                  undisputed_minor: 100_000_00,
                  due_at: "2026-10-01T00:00:00Z",
                  aging_bucket: "15_30_DAYS",
                },
              ],
              promises: [],
              ledger_entries: [],
            });
          }
          if (url.startsWith("/v1/retailer/ar/invoices?status=OPEN")) {
            return jsonResponse({ invoices: [] });
          }
          if (
            url.startsWith("/v1/retailer/ar/promises") &&
            (!init || init.method !== "POST")
          ) {
            return jsonResponse({ promises: [] });
          }
          if (url === "/v1/retailer/ar/pay-fifo") {
            return jsonResponse({
              paid_minor: 30_000_000,
              advance_added_minor: 0,
              advance_balance_minor: 50_000_00,
              allocations: [
                {
                  invoice_id: "inv-older",
                  applied_minor: 10_000_000,
                  remaining_balance_minor: 10_000_000,
                  status: "PARTIAL",
                },
              ],
            });
          }
          if (url === "/v1/retailer/ar/promises" && init?.method === "POST") {
            return jsonResponse(
              {
                code: "quarterly_promise_limit_reached",
                error: "quarterly_promise_limit_reached",
                quarter_key: "2026-Q4",
                message: "Only 1 self-serve promise per quarter",
              },
              409,
            );
          }
          return jsonResponse({}, 404);
        },
      );
    };

    setupMockFetch();
    render(<CreditPage />);

    await waitFor(() => {
      expect(screen.getByText("Baraka Dist")).toBeTruthy();
    });

    expect(screen.getAllByText(formatWholeSom(400_000_00)).length).toBeGreaterThan(0);
    expect(screen.getByText(formatWholeSom(50_000_00))).toBeTruthy();

    const invoiceHeaders = screen.getAllByText(/Order #ORD-/);
    expect(invoiceHeaders[0].textContent).toContain("ORD-1");
    expect(invoiceHeaders[1].textContent).toContain("ORD-2");

    const amountInput = screen.getByLabelText("Payment amount (so'm)");
    fireEvent.change(amountInput, { target: { value: "300000" } });
    fireEvent.click(
      screen.getByRole("button", { name: "Pay oldest-first (FIFO)" }),
    );

    await waitFor(() => {
      const fifoCall = apiFetchMock.mock.calls.find(
        ([callUrl]) => callUrl === "/v1/retailer/ar/pay-fifo",
      );
      expect(fifoCall).toBeTruthy();
      const body = JSON.parse(String((fifoCall?.[1] as RequestInit).body));
      expect(body.amount_minor).toBe(30_000_000);
      expect(body.supplier_id).toBe("sup-uz");
    });

    const payInvoiceButtons = screen.getAllByRole("button", {
      name: "Pay invoice",
    });
    fireEvent.click(payInvoiceButtons[0]);
    expect((amountInput as HTMLInputElement).value).toBe("100000");

    fireEvent.click(
      screen.getByRole("button", { name: "Request extension" }),
    );

    await waitFor(() => {
      expect(
        screen.getByText(/Quarterly promise limit reached \(2026-Q4\)/),
      ).toBeTruthy();
    });
  });
});
