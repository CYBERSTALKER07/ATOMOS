"use client";

import Link from "next/link";
import type { Route } from "next";
import { useCallback, useEffect, useState } from "react";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { createSupplierApi } from "@/lib/api";
import type { FlywheelDemandItem } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

/**
 * B4.4 STORE_POS flywheel feed — retailer POS sell-through that emits DEMAND_SIGNAL.
 * Distinct from planning Demand Signals (holiday/weather/promo multipliers).
 */
export default function DemandFlywheelPage() {
  const [items, setItems] = useState<FlywheelDemandItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [days, setDays] = useState(7);
  const [refreshTick, setRefreshTick] = useState(0);
  useSupplierSessionReconcile(() => setRefreshTick((t) => t + 1));

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    api
      .getSupplierDemandFlywheel({ days, limit: 200 })
      .then((res) => {
        setItems(Array.isArray(res.items) ? res.items : []);
        if (res.feed_error) {
          setError("feed_unavailable_apply_ddl_or_wait_for_pos");
        }
      })
      .catch((err) => setError(err instanceof Error ? err.message : "load_flywheel_failed"))
      .finally(() => setLoading(false));
  }, [days]);

  useEffect(() => {
    load();
  }, [load, refreshTick]);

  const saleCount = items.filter((i) => i.kind === "sale").length;
  const voidCount = items.filter((i) => i.kind === "void").length;
  const netDelta = items.reduce((sum, i) => sum + (i.qty_delta || 0), 0);

  return (
    <PageChrome
      icon="campaign"
      title="POS flywheel demand"
      description="Live retailer floor sell-through (STORE_POS). Kafka DEMAND_SIGNAL feed — not planning multipliers."
      loading={loading}
      error={error}
      empty={!loading && items.length === 0}
      emptyMessage="No STORE_POS flywheel events yet. When retailers complete POS sales for your SKUs, rows appear here."
      actions={
        <div className="flex flex-wrap gap-2">
          <select
            className="desk-input h-10"
            value={days}
            onChange={(e) => setDays(Number(e.target.value) || 7)}
            aria-label="Days window"
          >
            <option value={1}>Last 1 day</option>
            <option value={7}>Last 7 days</option>
            <option value={14}>Last 14 days</option>
            <option value={30}>Last 30 days</option>
          </select>
          <button type="button" className="md-btn md-btn-outlined h-10 px-4" onClick={() => load()}>
            Refresh
          </button>
          <Link
            href={"/analytics/demand" as Route}
            className="md-btn md-btn-tonal h-10 px-4 inline-flex items-center"
          >
            Forecast
          </Link>
          <Link
            href={"/analytics/demand/signals" as Route}
            className="md-btn md-btn-tonal h-10 px-4 inline-flex items-center"
          >
            Planning signals
          </Link>
        </div>
      }
    >
      <div className="mb-4 rounded-xl border border-[var(--color-md-outline-variant)] bg-[var(--color-md-surface-container-low)] p-4 text-sm text-[var(--color-md-outline)]">
        <p>
          <strong className="text-[var(--color-md-on-surface)]">Flywheel vs planning:</strong> this
          page shows POS qty sold/voided at retailer stores. Planning demand signals
          (holiday/weather/promo multipliers) live under{" "}
          <Link href={"/analytics/demand/signals" as Route} className="underline text-[var(--color-md-primary)]">
            Demand Signals
          </Link>
          .
        </p>
        {items.length > 0 && (
          <p className="mt-2">
            Window: {days}d · events {items.length} · sales {saleCount} · voids {voidCount} · net
            qty Δ {netDelta}
          </p>
        )}
      </div>

      {items.length > 0 && (
        <div className="overflow-x-auto rounded-xl border border-[var(--color-md-outline-variant)]">
          <table className="w-full text-sm text-left">
            <thead className="bg-[var(--color-md-surface-container)] text-xs uppercase tracking-wide text-[var(--color-md-outline)]">
              <tr>
                <th className="px-3 py-2">When</th>
                <th className="px-3 py-2">Day</th>
                <th className="px-3 py-2">Retailer</th>
                <th className="px-3 py-2">SKU</th>
                <th className="px-3 py-2">Kind</th>
                <th className="px-3 py-2">Δ qty</th>
                <th className="px-3 py-2">Net sold</th>
                <th className="px-3 py-2">Source</th>
              </tr>
            </thead>
            <tbody>
              {items.map((row) => (
                <tr
                  key={row.signal_id}
                  className="border-t border-[var(--color-md-outline-variant)]"
                >
                  <td className="px-3 py-2 font-mono text-xs text-[var(--color-md-outline)]">
                    {row.created_at
                      ? new Date(row.created_at).toLocaleString()
                      : "—"}
                  </td>
                  <td className="px-3 py-2 font-mono text-xs">{row.day}</td>
                  <td className="px-3 py-2 font-mono text-xs">{row.retailer_id}</td>
                  <td className="px-3 py-2 font-mono text-xs">{row.sku}</td>
                  <td className="px-3 py-2">
                    <span
                      className={
                        row.kind === "void"
                          ? "text-[var(--color-md-error)]"
                          : "text-[var(--color-md-primary)]"
                      }
                    >
                      {row.kind}
                    </span>
                  </td>
                  <td className="px-3 py-2 font-mono">
                    {row.qty_delta > 0 ? `+${row.qty_delta}` : row.qty_delta}
                  </td>
                  <td className="px-3 py-2 font-mono">{row.net_sold}</td>
                  <td className="px-3 py-2 text-xs">{row.source}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </PageChrome>
  );
}
