import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi, beforeEach } from "vitest";
import type { CashReconciliationRow } from "@pegasusx/types";

const {
  mockListCashReconciliations,
  mockAcceptCashReconciliation,
  mockWriteOffCashReconciliation,
} = vi.hoisted(() => ({
  mockListCashReconciliations: vi.fn(),
  mockAcceptCashReconciliation: vi.fn(),
  mockWriteOffCashReconciliation: vi.fn(),
}));

vi.mock("@/lib/api", () => ({
  createSupplierApi: () => ({
    listCashReconciliations: mockListCashReconciliations,
    acceptCashReconciliation: mockAcceptCashReconciliation,
    writeOffCashReconciliation: mockWriteOffCashReconciliation,
  }),
}));

vi.mock("@/lib/i18n", () => ({
  usePortalT: () => (key: string) => key,
}));

import CashReconciliationsPage from "@/app/(portal)/treasury/cash-reconciliations/page";

describe("Supplier Portal Cash Reconciliations UI", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("displays expected, declared, and difference amounts as whole so'm without 100x tiyin inflation", async () => {
    const rows: CashReconciliationRow[] = [
      {
        reconciliation_id: "rec-001",
        driver_id: "drv-11",
        route_id: "route-42",
        shift_date: "2026-11-12",
        expected_cash_minor: 142500000, // 1,425,000 so'm
        declared_cash_minor: 140000000, // 1,400,000 so'm
        difference_minor: -2500000, // -25,000 so'm
        status: "PENDING",
        created_at: "2026-11-12T10:00:00Z",
      },
      {
        reconciliation_id: "rec-002",
        driver_id: "drv-12",
        shift_date: "2026-11-12",
        expected_cash_minor: 50000000, // 500,000 so'm
        declared_cash_minor: 50000000,
        difference_minor: -50, // truncates toward 0 whole so'm (no "-0")
        status: "PENDING",
        created_at: "2026-11-12T11:00:00Z",
      },
    ];

    mockListCashReconciliations.mockResolvedValue({ reconciliations: rows });

    render(<CashReconciliationsPage />);

    await waitFor(() => {
      expect(screen.getByText(/Expected 1,425,000 so'm/)).toBeInTheDocument();
    });

    expect(screen.getByText(/Declared 1,400,000 so'm/)).toBeInTheDocument();
    expect(screen.getByText("-25,000 so'm")).toBeInTheDocument();
    expect(screen.getByText(/Expected 500,000 so'm/)).toBeInTheDocument();
    expect(screen.getByText("0 so'm")).toBeInTheDocument();
    expect(screen.queryByText(/142,500,000/)).not.toBeInTheDocument();
    expect(screen.queryByText(/140,000,000/)).not.toBeInTheDocument();
    expect(screen.queryByText(/-0 so'm/)).not.toBeInTheDocument();
    expect(screen.queryByText(/tiyin/i)).not.toBeInTheDocument();
  });
});
