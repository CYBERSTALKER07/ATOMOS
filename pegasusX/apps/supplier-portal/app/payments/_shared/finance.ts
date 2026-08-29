"use client";

import { ApiClient, ApiError } from '@pegasusx/api-core';
import { supplierApiBaseUrl, supplierFetch } from "@/lib/auth";
import type {
  PaymentLedgerResponse,
  ReconciliationMismatchRow,
  SettlementAuthorityResponse,
  SettlementAuthorityRow,
  SettlementCurrencyTotal,
} from "@pegasusx/types";
import { useEffect, useRef, useState } from "react";

export type DataSource = "settlement_authority" | "ledger_fallback";

export interface FinanceAuthoritySnapshot {
  source: DataSource;
  authority: SettlementAuthorityResponse;
  ledger: PaymentLedgerResponse;
  mismatches: ReconciliationMismatchRow[];
  refreshedAt: string;
}

export interface SupplierFinanceLiveState {
  status: "connecting" | "live" | "degraded";
  message: string;
  attempts: number;
  lastEventType?: string;
  lastEventAt?: string;
}

interface SupplierWebSocketSessionResponse {
  token: string;
  expires_at: string;
  websocket_url: string;
}

const supplierFinanceRefreshEventTypes = new Set([
  "PAYMENT_REQUIRED",
  "PAYMENT_CLEARED",
  "FISCAL_RECEIPT_REQUESTED",
  "FISCAL_RECEIPT_SUCCEEDED",
  "FISCAL_RECEIPT_FAILED",
  "ORDER_FORCE_COMPLETED",
  "SETTLEMENT_REQUIRED",
  "DELIVERY_DISPUTED",
]);

export async function loadFinanceAuthoritySnapshot(api: ApiClient): Promise<FinanceAuthoritySnapshot> {
  const ledger = await api.getPaymentLedger({ limit: 200 });
  const mismatchesPromise = loadReconciliationMismatches(api);

  try {
    const authority = await api.getPaymentSettlementAuthority({ group_limit: 200 });
    const mismatches = await mismatchesPromise;
    return {
      source: "settlement_authority",
      authority,
      ledger,
      mismatches,
      refreshedAt: new Date().toISOString(),
    };
  } catch {
    // G3.D: honest fallback — settlement authority endpoint failed; ledger is still real.
    const mismatches = await mismatchesPromise;
    return {
      source: "ledger_fallback",
      authority: buildSummaryFromLedger(ledger),
      ledger,
      mismatches,
      refreshedAt: new Date().toISOString(),
    };
  }
}

async function loadReconciliationMismatches(api: ApiClient): Promise<ReconciliationMismatchRow[]> {
  try {
    const response = await api.getPaymentReconciliationMismatches({
      group_limit: 200,
      mismatch_threshold_minor: 1,
    });
    return response.items;
  } catch {
    return [];
  }
}

export function useSupplierFinanceLiveRefresh(onSignal: (eventType: string) => void): SupplierFinanceLiveState {
  const onSignalRef = useRef(onSignal);
  const [state, setState] = useState<SupplierFinanceLiveState>({
    status: "connecting",
    message: "Connecting live finance updates...",
    attempts: 0,
  });

  useEffect(() => {
    onSignalRef.current = onSignal;
  }, [onSignal]);

  useEffect(() => {
    let cancelled = false;
    let eventSource: EventSource | null = null;
    let refreshTimer: number | undefined;

    const clearTimers = () => {
      if (refreshTimer !== undefined) {
        window.clearTimeout(refreshTimer);
        refreshTimer = undefined;
      }
    };

    const closeStream = () => {
      if (!eventSource) {
        return;
      }
      eventSource.onopen = null;
      eventSource.onmessage = null;
      eventSource.onerror = null;
      eventSource.close();
      eventSource = null;
    };

    const handleEvent = (eventType: string | null) => {
      if (!eventType || !supplierFinanceRefreshEventTypes.has(eventType)) {
        return;
      }
      if (refreshTimer !== undefined) {
        window.clearTimeout(refreshTimer);
      }
      refreshTimer = window.setTimeout(() => {
        if (cancelled) {
          return;
        }
        onSignalRef.current(eventType);
        setState((current) => ({
          status: current.status,
          message: `Live finance updates connected. Last event: ${eventType}.`,
          attempts: current.attempts,
          lastEventType: eventType,
          lastEventAt: new Date().toISOString(),
        }));
      }, 250);
    };

    try {
      setState((current) => ({
        status: current.attempts > 0 ? "degraded" : "connecting",
        message: current.attempts > 0 ? "Reconnecting live finance updates..." : "Connecting live finance updates...",
        attempts: current.attempts,
        lastEventAt: current.lastEventAt,
        lastEventType: current.lastEventType,
      }));

      const sseUrl = `${supplierApiBaseUrl()}/v1/supplier/events`;
      eventSource = new EventSource(sseUrl, { withCredentials: true });

      eventSource.onopen = () => {
        setState((current) => ({
          status: "live",
          message: "Live finance updates connected.",
          attempts: 0,
          lastEventAt: current.lastEventAt,
          lastEventType: current.lastEventType,
        }));
      };

      eventSource.onmessage = (event) => {
        const eventType = parseEventType(event.data);
        handleEvent(eventType);
      };

      for (const type of supplierFinanceRefreshEventTypes) {
        eventSource.addEventListener(type, (event: MessageEvent) => {
          const eventType = parseEventType(event.data) || type;
          handleEvent(eventType);
        });
      }

      eventSource.onerror = () => {
        setState((current) => ({
          status: "degraded",
          message: "Live finance updates interrupted.",
          attempts: current.attempts + 1,
          lastEventAt: current.lastEventAt,
          lastEventType: current.lastEventType,
        }));
      };
    } catch {
      setState((current) => ({
        status: "degraded",
        message: "Live finance updates unavailable.",
        attempts: current.attempts + 1,
        lastEventAt: current.lastEventAt,
        lastEventType: current.lastEventType,
      }));
    }

    return () => {
      cancelled = true;
      clearTimers();
      closeStream();
    };
  }, []);

  return state;
}

function extractSessionError(payload: SupplierWebSocketSessionResponse | { error?: string } | null): string {
  if (payload && "error" in payload && typeof payload.error === "string" && payload.error.trim() !== "") {
    return payload.error;
  }
  return "Unable to open live finance session.";
}

function parseEventType(raw: unknown): string | null {
  if (typeof raw !== "string" || raw.trim() === "") {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as { type?: string };
    return typeof parsed.type === "string" ? parsed.type : null;
  } catch {
    return null;
  }
}

export function buildSummaryFromLedger(response: PaymentLedgerResponse): SettlementAuthorityResponse {
  const groups = new Map<string, SettlementAuthorityRow>();

  for (const item of response.items) {
    const key = `${item.gateway}|${item.entry_type}|${item.currency}`;
    const existing = groups.get(key);
    if (!existing) {
      groups.set(key, {
        gateway: item.gateway,
        entry_type: item.entry_type,
        currency: item.currency,
        entry_count: 1,
        amount_minor_total: item.amount_minor,
        first_occurred_at: item.occurred_at,
        last_occurred_at: item.occurred_at,
      });
      continue;
    }

    existing.entry_count += 1;
    existing.amount_minor_total += item.amount_minor;
    if (parseDate(item.occurred_at) < parseDate(existing.first_occurred_at)) {
      existing.first_occurred_at = item.occurred_at;
    }
    if (parseDate(item.occurred_at) > parseDate(existing.last_occurred_at)) {
      existing.last_occurred_at = item.occurred_at;
    }
    groups.set(key, existing);
  }

  const items = Array.from(groups.values()).sort((a, b) => {
    if (a.gateway === b.gateway) {
      if (a.entry_type === b.entry_type) {
        return a.currency.localeCompare(b.currency);
      }
      return a.entry_type.localeCompare(b.entry_type);
    }
    return a.gateway.localeCompare(b.gateway);
  });

  const totalsByCurrency = new Map<string, SettlementCurrencyTotal>();
  let entryCountTotal = 0;
  for (const row of items) {
    entryCountTotal += row.entry_count;
    const existing = totalsByCurrency.get(row.currency);
    if (!existing) {
      totalsByCurrency.set(row.currency, {
        currency: row.currency,
        entry_count: row.entry_count,
        amount_minor_total: row.amount_minor_total,
      });
      continue;
    }
    existing.entry_count += row.entry_count;
    existing.amount_minor_total += row.amount_minor_total;
    totalsByCurrency.set(row.currency, existing);
  }

  return {
    items,
    count: items.length,
    group_limit: response.limit,
    supplier_id: response.supplier_id,
    entry_count_total: entryCountTotal,
    totals_by_currency: Array.from(totalsByCurrency.values()).sort((a, b) => a.currency.localeCompare(b.currency)),
    filters: {
      gateway: response.filters?.gateway,
      entry_type: response.filters?.entry_type,
      occurred_from: response.filters?.occurred_from ?? null,
      occurred_to: response.filters?.occurred_to ?? null,
    },
  };
}

export function parseDate(value: string): number {
  const timestamp = Date.parse(value);
  if (Number.isNaN(timestamp)) {
    return 0;
  }
  return timestamp;
}

export function formatDateTime(value: string): string {
  const timestamp = parseDate(value);
  if (timestamp <= 0) {
    return value;
  }
  return new Date(timestamp).toLocaleString();
}

export function formatMinor(amountMinor: number, currency: string): string {
  return `${amountMinor.toLocaleString()} ${currency}`;
}

export function createIdempotencyKey(prefix: string): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return `${prefix}-${crypto.randomUUID()}`;
  }
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

export function errorToMessage(error: unknown): string {
  if (error instanceof ApiError && typeof error.payload === "object" && error.payload !== null) {
    const payload = error.payload as { error?: string; message?: string };
    if (typeof payload.message === "string" && payload.message.trim() !== "") {
      return payload.message;
    }
    if (typeof payload.error === "string" && payload.error.trim() !== "") {
      return payload.error;
    }
  }
  if (error instanceof Error && error.message.trim() !== "") {
    return error.message;
  }
  return "Failed to load supplier finance authority.";
}