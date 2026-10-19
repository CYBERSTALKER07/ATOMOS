import { Zap, ArrowUpRight, ChevronRight } from "lucide-react";
import { moneyCurrency } from "../../lib/payment-catalog";
import type { TopProduct } from "../../lib/types";

type DetailedSeriesRow = {
  order_id: string;
  total_minor?: number;
  currency?: string;
};

interface InsightsSidebarProps {
  totalThisMonth: number;
  topProducts: TopProduct[];
  detailedError: unknown;
  detailedSeries: DetailedSeriesRow[];
}

export function InsightsSidebar({
  totalThisMonth,
  topProducts,
  detailedError,
  detailedSeries,
}: InsightsSidebarProps) {
  return (
    <aside className="w-[360px] shrink-0 hidden lg:flex flex-col gap-6">
      <div className="p-8 bg-[var(--desk-text-primary)] rounded-3xl text-white shadow-2xl relative overflow-hidden">
        <Zap className="absolute top-[-10px] right-[-10px] w-32 h-32 opacity-10 rotate-12" />
        <span className="md-typescale-label-small uppercase tracking-[0.2em] opacity-60 mb-4 block">
          Fleet Efficiency
        </span>
        <h3 className="md-typescale-display-small font-light mb-2 tabular-nums">
          {totalThisMonth.toLocaleString()}
        </h3>
        <p className="text-sm opacity-60 font-medium">
          Monthly Operational Volume
        </p>
        <div className="mt-8 pt-6 border-t border-white/10 flex items-center justify-between">
          <span className="text-xs font-light opacity-60 uppercase tracking-widest">
            Trend
          </span>
          <div className="flex items-center gap-2 text-[var(--desk-success)] font-light">
            <ArrowUpRight size={16} />
            <span>+12.4%</span>
          </div>
        </div>
      </div>

      <div className="p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
        <h3 className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-6">
          Top Resource Demand
        </h3>
        {topProducts.length === 0 ? (
          <div className="rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface-subtle)] p-4 text-center">
            <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] uppercase font-light tracking-widest">
              No product demand rankings available yet
            </p>
          </div>
        ) : (
          <div className="space-y-4">
            {topProducts.slice(0, 5).map((item, i: number) => (
              <div
                key={item.product_id}
                className="flex items-center gap-4 group"
              >
                <div className="w-8 h-8 rounded-lg bg-[var(--desk-surface-subtle)] flex items-center justify-center text-[10px] font-black text-[var(--desk-text-tertiary)] group-hover:bg-[var(--desk-accent-soft)] group-hover:text-[var(--desk-accent)] transition-colors">
                  {i + 1}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="md-typescale-body-medium font-light text-[var(--desk-text-primary)] truncate">
                    {item.product_name}
                  </p>
                  <p className="text-[10px] text-[var(--desk-text-tertiary)] font-light uppercase">
                    {item.quantity} Units
                  </p>
                </div>
                <ChevronRight
                  size={14}
                  className="text-[var(--desk-text-tertiary)] opacity-0 group-hover:opacity-100 transition-opacity"
                />
              </div>
            ))}
          </div>
        )}
      </div>

      <div className="p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
        <h3 className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-4">
          Detailed spend series
        </h3>
        {detailedError ? (
          <p className="text-sm text-[var(--desk-text-tertiary)]">
            Advanced analytics unavailable right now.
          </p>
        ) : detailedSeries.length === 0 ? (
          <p className="text-sm text-[var(--desk-text-tertiary)]">
            No completed-order series yet.
          </p>
        ) : (
          <div className="space-y-2 max-h-48 overflow-y-auto">
            {detailedSeries.slice(0, 8).map((row) => (
              <div
                key={row.order_id}
                className="flex items-center justify-between py-2 border-b border-[var(--desk-border)] last:border-0"
              >
                <span className="text-xs font-mono text-[var(--desk-text-secondary)]">
                  #{row.order_id.slice(-8)}
                </span>
                <span className="text-sm font-light tabular-nums">
                  {(row.total_minor ?? 0).toLocaleString()}{" "}
                  {moneyCurrency(row.currency)}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </aside>
  );
}
