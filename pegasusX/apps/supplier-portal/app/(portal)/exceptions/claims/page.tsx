"use client";

import Link from "next/link";
import type { Route } from "next";
import { useCallback, useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import StatusBadge from "@/components/StatusBadge";
import { PageChrome } from "@/components/PageChrome";

type ClaimLine = {
  sku: string;
  quantity: number;
  reason?: string;
  unit_price_minor?: number;
  amount_minor?: number;
};

type Claim = {
  claim_id: string;
  order_id: string;
  retailer_id: string;
  claim_type: string;
  status: string;
  amount_minor?: number;
  currency?: string;
  description?: string;
  line_items?: ClaimLine[];
  evidences?: { uri: string; evidence_type: string }[];
  created_at: string;
};

type Settlement = {
  chargeback_id?: string;
  amount_minor: number;
  mode: string;
  gateway_refunded?: boolean;
};

const SETTLEMENT_MODES = [
  {
    value: "LEDGER_ONLY",
    label: "Ledger only",
    hint: "Debit supplier settlement only (safe default)",
  },
  {
    value: "STORE_CREDIT",
    label: "Store credit",
    hint: "Ledger + reduce retailer credit balance due",
  },
  {
    value: "GATEWAY_REFUND",
    label: "Card refund (GP)",
    hint: "Ledger + Global Pay partial refund when session is card",
  },
] as const;

export default function ClaimsQueuePage() {
  const [claims, setClaims] = useState<Claim[]>([]);
  const [statusFilter, setStatusFilter] = useState("OPEN");
  const [settlementMode, setSettlementMode] = useState<string>("LEDGER_ONLY");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [lastSettlement, setLastSettlement] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const q = statusFilter ? `?status=${encodeURIComponent(statusFilter)}&limit=50` : "?limit=50";
      const res = await supplierFetch(`/v1/supplier/claims${q}`);
      if (!res.ok) {
        throw new Error(`claims_load_${res.status}`);
      }
      const body = (await res.json()) as { claims?: Claim[] };
      setClaims(body.claims ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "load_claims_failed");
    } finally {
      setLoading(false);
    }
  }, [statusFilter]);

  useEffect(() => {
    void load();
  }, [load]);

  async function approve(claimId: string) {
    setBusyId(claimId);
    setLastSettlement(null);
    try {
      const res = await supplierFetch(`/v1/claims/${encodeURIComponent(claimId)}/approve`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          resolution_note: "approved_via_supplier_portal",
          settlement_mode: settlementMode,
          skip_gateway_refund: settlementMode !== "GATEWAY_REFUND",
        }),
      });
      if (!res.ok) {
        throw new Error(`approve_${res.status}`);
      }
      const body = (await res.json()) as { settlement?: Settlement };
      if (body.settlement) {
        setLastSettlement(
          `${body.settlement.mode} · ${body.settlement.amount_minor} · refund=${Boolean(body.settlement.gateway_refunded)} · id=${body.settlement.chargeback_id ?? "—"}`,
        );
      }
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "approve_failed");
    } finally {
      setBusyId(null);
    }
  }

  async function reject(claimId: string) {
    setBusyId(claimId);
    try {
      const res = await supplierFetch(`/v1/claims/${encodeURIComponent(claimId)}/reject`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ resolution_note: "rejected_via_supplier_portal" }),
      });
      if (!res.ok) {
        throw new Error(`reject_${res.status}`);
      }
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "reject_failed");
    } finally {
      setBusyId(null);
    }
  }

  return (
    <PageChrome
      title="Claims queue"
      description="Post-delivery damage, missing, and OS&D claims. Choose settlement mode before approve."
      icon="warning"
      loading={loading}
      error={error}
      empty={!loading && claims.length === 0}
      emptyMessage="No claims in this filter. Retailer post-delivery claims appear here within the 48h window."
    >
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <label className="md-typescale-label-large flex items-center gap-2">
          Status
          <select
            className="md-btn md-btn-outlined px-3 py-1"
            value={statusFilter}
            onChange={(e) => setStatusFilter(e.target.value)}
          >
            <option value="OPEN">OPEN</option>
            <option value="UNDER_REVIEW">UNDER_REVIEW</option>
            <option value="RESOLVED">RESOLVED</option>
            <option value="REJECTED">REJECTED</option>
            <option value="">ALL</option>
          </select>
        </label>
        <label className="md-typescale-label-large flex items-center gap-2">
          Settlement
          <select
            className="md-btn md-btn-outlined px-3 py-1 max-w-[220px]"
            value={settlementMode}
            onChange={(e) => setSettlementMode(e.target.value)}
          >
            {SETTLEMENT_MODES.map((m) => (
              <option key={m.value} value={m.value}>
                {m.label}
              </option>
            ))}
          </select>
        </label>
        <button type="button" className="md-btn md-btn-outlined px-3 py-1" onClick={() => void load()}>
          Refresh
        </button>
        {lastSettlement ? (
          <span className="md-typescale-body-small text-[var(--color-md-outline)]">
            Last settle: {lastSettlement}
          </span>
        ) : null}
      </div>
      <p className="mb-4 text-xs text-[var(--color-md-outline)]">
        {SETTLEMENT_MODES.find((m) => m.value === settlementMode)?.hint}
        {" · "}
        <Link href={"/chargebacks/claims" as Route} className="underline text-[var(--color-md-primary)]">
          Claim chargebacks ledger
        </Link>
      </p>

      <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
        {claims.map((c) => (
          <li key={c.claim_id} className="p-4 md-typescale-body-medium">
            <div className="flex flex-wrap items-center gap-2">
              <span className="md-chip h-6 text-xs">{c.claim_type}</span>
              <StatusBadge state={c.status} />
              <span className="font-mono text-[var(--color-md-primary)]">{c.claim_id}</span>
            </div>
            <p className="mt-1 text-sm">
              Order <span className="font-mono">{c.order_id}</span> · Retailer{" "}
              <span className="font-mono">{c.retailer_id}</span>
            </p>
            <p className="mt-1 text-sm text-[var(--color-md-outline)]">
              Amount {c.amount_minor ?? 0} {c.currency ?? "UZS"} ·{" "}
              {c.created_at ? new Date(c.created_at).toLocaleString() : "—"}
            </p>
            {c.description ? <p className="mt-2">{c.description}</p> : null}
            {c.line_items && c.line_items.length > 0 ? (
              <ul className="mt-2 text-sm text-[var(--color-md-outline)]">
                {c.line_items.map((li) => (
                  <li key={`${c.claim_id}-${li.sku}`}>
                    {li.sku} × {li.quantity}
                    {li.amount_minor != null ? ` · ${li.amount_minor}` : ""}
                    {li.reason ? ` · ${li.reason}` : ""}
                  </li>
                ))}
              </ul>
            ) : null}
            {c.evidences && c.evidences.length > 0 ? (
              <div className="mt-2 flex flex-wrap gap-2">
                {c.evidences.map((e, i) => (
                  <a
                    key={`${c.claim_id}-ev-${i}`}
                    href={e.uri}
                    target="_blank"
                    rel="noreferrer"
                    className="text-[var(--color-md-primary)] underline text-sm"
                  >
                    Evidence {i + 1}
                  </a>
                ))}
              </div>
            ) : null}
            {(c.status === "OPEN" || c.status === "UNDER_REVIEW") && (
              <div className="mt-3 flex flex-wrap gap-2">
                <button
                  type="button"
                  className="md-btn md-btn-filled px-3 py-1"
                  disabled={busyId === c.claim_id}
                  onClick={() => void approve(c.claim_id)}
                >
                  Approve ({SETTLEMENT_MODES.find((m) => m.value === settlementMode)?.label})
                </button>
                <button
                  type="button"
                  className="md-btn md-btn-outlined px-3 py-1"
                  disabled={busyId === c.claim_id}
                  onClick={() => void reject(c.claim_id)}
                >
                  Reject
                </button>
              </div>
            )}
          </li>
        ))}
      </ul>

      <div className="mt-4 flex flex-wrap gap-4">
        <Link href={"/exceptions" as Route} className="text-[var(--color-md-primary)] underline">
          All exceptions
        </Link>
        <Link href={"/credit/collections" as Route} className="text-[var(--color-md-primary)] underline">
          Credit collections
        </Link>
        <Link href={"/chargebacks/claims" as Route} className="text-[var(--color-md-primary)] underline">
          Claim chargebacks
        </Link>
      </div>
    </PageChrome>
  );
}
