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

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), replace: vi.fn(), back: vi.fn() }),
  usePathname: () => "/dashboard",
}));

vi.mock("@/lib/i18n", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/i18n")>();
  return {
    ...actual,
    usePortalLocale: () => "en",
  };
});

import {
  buildFastHomeFallback,
  computeCutoffCountdown,
  computeServerClockOffsetMs,
  formatFastWholeSom,
  resolveDefaultWorkspaceMode,
  resolveEffectiveWorkspaceMode,
  WORKSPACE_MODE_STORAGE_KEY,
  writeStoredWorkspaceMode,
} from "@/lib/fast-home";
import { FastHomeView } from "@/components/dashboard/FastHomeView";
import { TelegramLinkCard } from "@/components/settings/TelegramLinkCard";
import { CommandBoard } from "@/components/dashboard/CommandBoard";
import { getTelegramLinkStatus, requestTelegramLink } from "@/lib/api";

function jsonResponse(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("Retailer W5 Fast Home, SSR Locale & Telegram Link", () => {
  beforeEach(() => {
    apiFetchMock.mockReset();
    window.localStorage.clear();
  });

  it("defaults SSR locale snapshot to uz and fixes CommandBoard raw minor formatting", () => {
    const i18nSrc = fs.readFileSync(
      path.resolve(process.cwd(), "lib/i18n.ts"),
      "utf8",
    );
    expect(i18nSrc).toMatch(/function getServerSnapshot\(\):\s*Locale\s*\{\s*return "uz";/);

    render(
      <CommandBoard
        pulse={{
          retailer_id: "ret-1",
          generated_at: "2026-11-13T08:00:00Z",
          open_orders: 2,
          active_fulfillments: 1,
          dock_pending: 0,
          pos_open_sessions: 1,
          open_shifts: 1,
          open_assist_tickets: 0,
          low_stock_sku_bins: 0,
          shift_variances_7d: 0,
          sales_minor_7d: 125_000,
          capabilities: [],
          empty: false,
          source: "spanner",
          orders_by_status: {},
          orders_by_supplier: [],
        }}
        loading={false}
      />,
    );
    expect(screen.getByText(/1\s?250\s+(so'm|UZS)/i)).toBeInTheDocument();
    expect(screen.queryByText(/125000/)).not.toBeInTheDocument();
  });

  it("computes server-synced cutoff countdown without device clock drift", () => {
    // Device clock is 3 hours AHEAD of server clock
    const serverNow = "2026-11-13T10:00:00Z";
    const cutoffAt = "2026-11-13T12:30:15Z"; // exactly 02:30:15 after serverNow
    const skewedDeviceNowMs = Date.parse("2026-11-13T13:00:00Z");

    const offsetMs = computeServerClockOffsetMs(serverNow, skewedDeviceNowMs);
    expect(offsetMs).toBe(-3 * 3600 * 1000);

    const countdown = computeCutoffCountdown(cutoffAt, offsetMs, skewedDeviceNowMs);
    expect(countdown.valid).toBe(true);
    expect(countdown.expired).toBe(false);
    expect(countdown.formatted).toBe("02:30:15");

    // 2h31m later on device clock -> cutoff passed
    const passed = computeCutoffCountdown(
      cutoffAt,
      offsetMs,
      skewedDeviceNowMs + 9_060_000,
    );
    expect(passed.expired).toBe(true);
    expect(passed.formatted).toBe("00:00:00");
  });

  it("applies role-default workspace modes and persists owner override", () => {
    expect(resolveDefaultWorkspaceMode("BUYER", false)).toBe("fast");
    expect(resolveDefaultWorkspaceMode("CASHIER", false)).toBe("fast");
    expect(resolveDefaultWorkspaceMode("MANAGER", false)).toBe("fast");
    expect(resolveDefaultWorkspaceMode("OWNER", true)).toBe("full");

    expect(resolveEffectiveWorkspaceMode(null, "OWNER", true)).toBe("full");
    expect(resolveEffectiveWorkspaceMode("fast", "OWNER", true)).toBe("fast");

    writeStoredWorkspaceMode("fast");
    expect(window.localStorage.getItem(WORKSPACE_MODE_STORAGE_KEY)).toBe("fast");
  });

  it("builds safe fallback snapshot from legacy orders, predictions, and credit profile", () => {
    const fallback = buildFastHomeFallback({
      now: new Date("2026-11-13T09:00:00Z"),
      orders: [
        {
          order_id: "ord-deliv-1",
          retailer_id: "ret-1",
          supplier_id: "sup-1",
          supplier_name: "Atlas Milk",
          state: "SCHEDULED",
          items: [
            {
              line_item_id: "li-1",
              order_id: "ord-deliv-1",
              sku_id: "p1",
              sku_name: "Milk 1L",
              quantity: 10,
              unit_price: 100_000,
              status: "SCHEDULED",
            },
          ],
          amount: 1_000_000,
          currency: "UZS",
          payment_gateway: "CREDIT",
          payment_status: "PENDING",
          route_id: null,
          order_source: "MANUAL",
          auto_confirm_at: null,
          deliver_before: null,
          delivery_token: null,
          version: 1,
          created_at: "2026-11-12T10:00:00Z",
          proposed_delivery_date: "2026-11-14T08:00:00Z",
        },
      ],
      predictions: [
        {
          order_id: "ord-ai-1",
          order_source: "AI_PREDICTED",
          confirmation_status: "PENDING_REVIEW",
          total_minor: 500_000,
          currency: "UZS",
          auto_confirm_at: "2026-11-13T13:00:00Z",
          requested_delivery_date: "2026-11-14T10:00:00Z",
          updated_at: "2026-11-13T08:00:00Z",
          line_items: [{ sku: "p2", name: "Bread", quantity: 5, unit_price_minor: 100_000 }],
        },
      ],
      creditProfile: {
        retailer_id: "ret-1",
        supplier_id: "sup-1",
        credit_limit_minor: 5_000_000,
        current_balance_minor: 1_250_000,
        available_credit_minor: 3_750_000,
        risk_tier: "LOW",
        delinquency_count: 0,
        status: "ACTIVE",
      },
      pendingApprovalsCount: 2,
    });

    expect(fallback.degraded_fallback).toBe(true);
    expect(fallback.tomorrow_delivery?.order_id).toBe("ord-deliv-1");
    expect(fallback.balance_owed_minor).toBe(1_250_000);
    expect(fallback.credit_headroom_minor).toBe(3_750_000);
    expect(fallback.pending_proposal?.order_id).toBe("ord-ai-1");
    expect(fallback.pending_approvals_count).toBe(2);
    expect(formatFastWholeSom(fallback.balance_owed_minor)).toContain("12 500 so'm");
  });

  it("renders FastHomeView 3 core cards in whole so'm and triggers Confirm / Hold", async () => {
    const onRefresh = vi.fn();
    apiFetchMock.mockResolvedValue(jsonResponse({ ok: true }));

    render(
      <FastHomeView
        data={{
          server_now: new Date().toISOString(),
          cutoff_at: new Date(Date.now() + 3600_000).toISOString(),
          next_delivery_date: "2026-11-14",
          tomorrow_delivery: {
            order_id: "ord-t1",
            supplier_name: "Fresh Dairy",
            state: "SCHEDULED",
            item_count: 12,
            amount_minor: 2_400_000,
            currency: "UZS",
            delivery_date: "2026-11-14",
          },
          balance_owed_minor: 5_000_000,
          overdue_minor: 1_000_000,
          days_overdue: 4,
          credit_headroom_minor: 10_000_000,
          pending_proposal: {
            order_id: "ord-ai-9",
            source: "AI_PREDICTED",
            supplier_name: "Atlas Beverage",
            state: "PENDING_REVIEW",
            item_count: 4,
            amount_minor: 800_000,
            currency: "UZS",
            items: [
              {
                product_id: "sku-water",
                product_name: "Mineral Water 1.5L",
                quantity: 4,
                unit_price_minor: 200_000,
                line_total_minor: 800_000,
              },
            ],
          },
          pending_approvals_count: 1,
          currency: "UZS",
        }}
        loading={false}
        onRefresh={onRefresh}
        onSwitchToFullErp={vi.fn()}
      />,
    );

    expect(screen.getByTestId("fast-home-tomorrow-card")).toHaveTextContent("Fresh Dairy");
    expect(screen.getByTestId("fast-home-owe-card")).toHaveTextContent("50 000 so'm");
    expect(screen.getByTestId("fast-home-proposal-card")).toHaveTextContent("8 000 so'm");

    fireEvent.click(screen.getByTestId("fast-home-confirm-btn"));
    await waitFor(() => {
      expect(apiFetchMock).toHaveBeenCalledWith(
        "/v1/retailer/orders/confirm-ai",
        expect.objectContaining({ method: "POST" }),
      );
    });
    expect(onRefresh).toHaveBeenCalled();
  });

  it("handles Telegram status and link-request OTP generation in TelegramLinkCard", async () => {
    apiFetchMock
      .mockResolvedValueOnce(jsonResponse({ linked: false }))
      .mockResolvedValueOnce(
        jsonResponse({
          ok: true,
          otp_code: "654321",
          expires_at: "2026-11-13T10:15:00Z",
          bot_deep_link: "https://t.me/pegasus_retailer_bot?start=654321",
        }),
      );

    render(<TelegramLinkCard />);

    await waitFor(() => {
      expect(apiFetchMock).toHaveBeenCalledWith("/v1/retailer/telegram/status");
    });

    fireEvent.click(screen.getByTestId("telegram-generate-otp"));
    await waitFor(() => {
      expect(screen.getByTestId("telegram-otp-code")).toHaveTextContent("654321");
    });

    // Also test typed API helpers directly
    apiFetchMock.mockResolvedValueOnce(jsonResponse({ linked: true, telegram_chat_id: "9988" }));
    const st = await getTelegramLinkStatus();
    expect(st.linked).toBe(true);

    apiFetchMock.mockResolvedValueOnce(
      jsonResponse({ ok: true, otp_code: "112233", expires_at: "2026-11-13T10:30:00Z" }),
    );
    const req = await requestTelegramLink("idem-1");
    expect(req.otp_code).toBe("112233");
  });
});
