import { describe, expect, it, vi, beforeEach } from "vitest";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { ApiError } from "@pegasusx/api-core";
import {
  clearTaxSetupDraft,
  readTaxSetupDraft,
  SETUP_TAX_KEY,
  writeTaxSetupDraft,
} from "@/components/setup/constants";
import { NasiyaOpeningBalancesPanel } from "@/components/credit/NasiyaOpeningBalancesPanel";
import * as api from "@/lib/api";
import { apiFetch } from "@/lib/auth";

describe("Retailer Desktop M10.3 parity", () => {
  beforeEach(() => {
    sessionStorage.clear();
    vi.restoreAllMocks();
  });

  it("handles legal, informal, and legacy tax setup drafts", () => {
    writeTaxSetupDraft({ mode: "INFORMAL_PHONE", taxId: "", persisted: true });
    expect(readTaxSetupDraft()).toEqual({
      mode: "INFORMAL_PHONE",
      taxId: "",
      legalName: undefined,
      persisted: true,
    });

    sessionStorage.setItem(SETUP_TAX_KEY, "305556667");
    expect(readTaxSetupDraft()).toEqual({
      mode: "LEGAL_VERIFIED",
      taxId: "305556667",
      persisted: false,
    });

    clearTaxSetupDraft();
    expect(readTaxSetupDraft()).toBeNull();
  });

  it("extracts 409 join_request_pending conflict payloads safely", () => {
    const conflictErr = new ApiError("Conflict", 409, {
      error: "join_request_pending",
      request_id: "jr-101",
      target_tenant_id: "ret-1",
      legal_name: "Baraka Market MChJ",
      normalized_inn: "305556667",
    });
    expect(api.extractJoinRequestConflict(conflictErr)).toMatchObject({
      request_id: "jr-101",
      legal_name: "Baraka Market MChJ",
      normalized_inn: "305556667",
    });
    expect(api.extractJoinRequestConflict(new Error("boom"))).toBeNull();
  });

  it("renders opening balances with MIGRATION badge and supports confirm and dispute actions", async () => {
    const ackSpy = vi
      .spyOn(api, "acknowledgeRetailerNasiyaOpeningBalance")
      .mockResolvedValue({
        retailer_id: "ret-1",
        invoice: {
          invoice_id: "inv-mig-1",
          supplier_id: "sup-1",
          retailer_id: "ret-1",
          principal_minor: 50000000,
          balance_minor: 50000000,
          currency: "UZS",
          status: "ACKNOWLEDGED",
          source: "MIGRATION",
          dunning_frozen: false,
        },
      });
    const disputeSpy = vi
      .spyOn(api, "disputeRetailerNasiyaOpeningBalance")
      .mockResolvedValue({
        retailer_id: "ret-1",
        invoice: {
          invoice_id: "inv-mig-2",
          supplier_id: "sup-1",
          retailer_id: "ret-1",
          principal_minor: 20000000,
          balance_minor: 20000000,
          currency: "UZS",
          status: "DISPUTED",
          source: "MIGRATION",
          dunning_frozen: true,
        },
      });
    const onChanged = vi.fn();

    render(
      <NasiyaOpeningBalancesPanel
        invoices={[
          {
            invoice_id: "inv-mig-1",
            supplier_id: "sup-1",
            supplier_name: "Samarkand Dairy",
            retailer_id: "ret-1",
            external_id: "NSY-001",
            principal_minor: 50000000,
            balance_minor: 50000000,
            currency: "UZS",
            status: "OPEN",
            source: "MIGRATION",
            dunning_frozen: true,
          },
          {
            invoice_id: "inv-mig-2",
            supplier_id: "sup-1",
            supplier_name: "Samarkand Dairy",
            retailer_id: "ret-1",
            external_id: "NSY-002",
            principal_minor: 20000000,
            balance_minor: 20000000,
            currency: "UZS",
            status: "OPEN",
            source: "MIGRATION",
            dunning_frozen: true,
          },
        ]}
        onChanged={onChanged}
      />,
    );

    const confirmButtons = screen.getAllByRole("button", {
      name: /Confirm Balance|Tasdiqlash|Подтвердить/i,
    });
    fireEvent.click(confirmButtons[0]);
    await waitFor(() => expect(ackSpy).toHaveBeenCalledWith("inv-mig-1"));
    await waitFor(() => expect(onChanged).toHaveBeenCalledTimes(1));

    const disputeButtons = screen.getAllByRole("button", {
      name: /Dispute|E'tiroz|Оспорить/i,
    });
    fireEvent.click(disputeButtons[1]);
    const input = screen.getByRole("textbox");
    fireEvent.change(input, { target: { value: "Returned 10 crates earlier" } });
    fireEvent.click(
      screen.getByRole("button", { name: /Submit Dispute|E'tiroz yuborish|Отправить спор/i }),
    );
    await waitFor(() =>
      expect(disputeSpy).toHaveBeenCalledWith(
        "inv-mig-2",
        "Returned 10 crates earlier",
      ),
    );
  });

  it("deduplicates lowercase authorization headers across initial fetch and 401 refresh retries", async () => {
    document.cookie = "pegasus_retailer_jwt=jwt-initial; path=/";
    document.cookie = "pegasus_retailer_refresh=ref-initial; path=/";
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(new Response("{}", { status: 401 }))
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ token: "jwt-refreshed", refresh_token: "ref-next" }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      )
      .mockResolvedValueOnce(new Response(JSON.stringify({ requests: [] }), { status: 200 }));

    const res = await apiFetch("/v1/retailer/join-requests", {
      method: "GET",
      headers: {
        authorization: "Bearer jwt-initial",
        "Idempotency-Key": "idem-header-test",
      },
    });
    expect(res.status).toBe(200);
    expect(fetchMock).toHaveBeenCalledTimes(3);

    const firstHeaders = fetchMock.mock.calls[0]?.[1]?.headers as Record<string, string>;
    const authKeysFirst = Object.keys(firstHeaders).filter((k) => k.toLowerCase() === "authorization");
    expect(authKeysFirst).toEqual(["authorization"]);
    expect(firstHeaders.authorization).toBe("Bearer jwt-initial");
    expect(firstHeaders["idempotency-key"]).toBe("idem-header-test");

    const retryHeaders = fetchMock.mock.calls[2]?.[1]?.headers as Record<string, string>;
    const authKeysRetry = Object.keys(retryHeaders).filter((k) => k.toLowerCase() === "authorization");
    expect(authKeysRetry).toEqual(["authorization"]);
    expect(retryHeaders.authorization).toBe("Bearer jwt-refreshed");
  });
});
