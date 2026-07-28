"use client";

import Link from "next/link";
import type { Route } from "next";
import { useCallback, useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { PageChrome } from "@/components/PageChrome";

type LedgerItem = {
  ledger_entry_id?: string;
  order_id?: string;
  supplier_id?: string;
  retailer_id?: string;
  gateway?: string;
  entry_type?: string;
  amount_minor?: number;
  currency?: string;
  reference_id?: string;
  source?: string;
  occurred_at?: string;
  created_at?: string;
};

function fmt(n?: number) {
  return (n ?? 0).toLocaleString();
}

function claimIdFromRef(ref?: string, source?: string): string {
  const r = (ref || "").toLowerCase();
  if (r.startsWith("chargeback_clm_")) return r.replace(/^chargeback_/, "");
  if (r.includes("clm_")) {
    const i = r.indexOf("clm_");
    return r.slice(i);
  }
  const s = source || "";
  const m = s.match(/claims\.settle_chargeback:(.+)$/i);
  return m?.[1] || "—";
}

export default function ClaimChargebacksPage() {
  const [items, setItems] = useState<LedgerItem[]>([]);
  const [orderFilter, setOrderFilter] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const q = new URLSearchParams({ limit: "100" });
      if (orderFilter.trim()) q.set("order_id", orderFilter.trim());
      const res = await supplierFetch(`/v1/supplier/claim-chargebacks?${q.toString()}`);
      if (!res.ok) {
        throw new Error(`load_${res.status}`);
      }
      const body = (await res.json()) as { items?: LedgerItem[]; count?: number };
      setItems(body.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "load_failed");
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, [orderFilter]);

  useEffect(() => {
    void load();
  }, [load]);

  const total = items.reduce((s, i) => s + (i.amount_minor ?? 0), 0);

  return (
    <PageChrome
      title="Claim chargebacks"
      description="Supplier-scoped ledger rows from logistics claim approve (chargeback_clm_*). Manual PSP chargebacks stay on Finance → Chargebacks."
      icon="warning"
      loading={loading}
      error={error}
      empty={!loading && items.length === 0}
      emptyMessage="No claim chargebacks yet. Approve a claim in Exceptions → Claims to create one."
    >
      <div className="mb-4 flex flex-wrap items-center gap-3">
        <input
          className="border border-[var(--border)] rounded-lg px-3 py-1.5 text-sm font-mono min-w-[200px]"
          placeholder="Filter order id"
          value={orderFilter}
          onChange={(e) => setOrderFilter(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") void load();
          }}
        />
        <button type="button" className="text-sm underline" onClick={() => void load()}>
          Refresh
        </button>
        <span className="text-xs text-[var(--muted)]">
          {items.length} rows · total {fmt(total)} minor
        </span>
      </div>

      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-[var(--border)] text-left text-xs uppercase tracking-wide text-[var(--muted)]">
              <th className="py-2 pr-3">When</th>
              <th className="py-2 pr-3">Order</th>
              <th className="py-2 pr-3">Claim / ref</th>
              <th className="py-2 pr-3">Retailer</th>
              <th className="py-2 pr-3">Gateway</th>
              <th className="py-2 pr-3 text-right">Amount</th>
              <th className="py-2 pr-3">Source</th>
            </tr>
          </thead>
          <tbody>
            {items.map((it, idx) => {
              const when = it.occurred_at || it.created_at;
              return (
                <tr
                  key={it.ledger_entry_id || it.reference_id || String(idx)}
                  className="border-b border-[var(--border)]"
                >
                  <td className="py-2.5 pr-3 text-xs text-[var(--muted)]">
                    {when ? new Date(when).toLocaleString() : "—"}
                  </td>
                  <td className="py-2.5 pr-3 font-mono text-xs">{it.order_id || "—"}</td>
                  <td className="py-2.5 pr-3 font-mono text-xs">
                    {claimIdFromRef(it.reference_id, it.source)}
                    <div className="text-[10px] text-[var(--muted)]">{it.reference_id}</div>
                  </td>
                  <td className="py-2.5 pr-3 font-mono text-xs">{it.retailer_id || "—"}</td>
                  <td className="py-2.5 pr-3 text-xs">{it.gateway || "—"}</td>
                  <td className="py-2.5 pr-3 text-right font-mono font-medium">
                    {fmt(it.amount_minor)} {it.currency || "UZS"}
                  </td>
                  <td className="py-2.5 pr-3 text-[10px] text-[var(--muted)] max-w-[180px] truncate">
                    {it.source || it.entry_type || "—"}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      <div className="mt-4 flex flex-wrap gap-4 text-sm">
        <Link href={"/exceptions/claims" as Route} className="underline text-[var(--color-md-primary)]">
          Claims queue
        </Link>
        <Link href={"/chargebacks" as Route} className="underline text-[var(--color-md-primary)]">
          Manual chargebacks
        </Link>
        <Link href={"/ledger" as Route} className="underline text-[var(--color-md-primary)]">
          Full ledger
        </Link>
        <Link href={"/credit/collections" as Route} className="underline text-[var(--color-md-primary)]">
          Credit collections
        </Link>
      </div>
    </PageChrome>
  );
}
