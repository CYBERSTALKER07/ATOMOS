"use client";

import { useCallback, useEffect, useState } from "react";
import { Download, Loader2 } from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { apiFetch } from "@/lib/auth";

type Summary = {
  sales_minor?: number;
  sale_count?: number;
  on_hand_sku_count?: number;
  low_stock_count?: number;
  open_variances?: number;
  top_skus?: { sku: string; sales_minor: number; units: number }[];
};

type SalesItem = { key: string; sales_minor: number; sale_count: number; units: number };

function formatMoney(minor: number) {
  return (minor / 100).toLocaleString(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  });
}

export default function ReportsProPage() {
  const [summary, setSummary] = useState<Summary | null>(null);
  const [sales, setSales] = useState<SalesItem[]>([]);
  const [banner, setBanner] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setBusy(true);
    try {
      const [sRes, salesRes] = await Promise.all([
        apiFetch("/v1/retailer/reports/summary"),
        apiFetch("/v1/retailer/reports/sales?group_by=sku"),
      ]);
      if (sRes.ok) setSummary((await sRes.json()) as Summary);
      if (salesRes.ok) {
        const json = (await salesRes.json()) as { items?: SalesItem[] };
        setSales(json.items ?? []);
      }
      setBanner("REPORTS_PRO pack auto-enabled on first view if needed");
    } catch {
      setBanner("Failed to load reports");
    } finally {
      setBusy(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const downloadCSV = async () => {
    const res = await apiFetch("/v1/retailer/reports/export?report=sales");
    if (!res.ok) {
      setBanner("Export failed");
      return;
    }
    const blob = await res.blob();
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "sales.csv";
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <PageChrome
      title="Reports Pro"
      description="Sales, inventory, and shift variance digest. CSV export for desktop ops."
    >
      <div className="mx-auto flex max-w-3xl flex-col gap-6 p-4">
        {banner && (
          <div className="rounded-lg border border-border bg-muted/40 px-3 py-2 text-sm">{banner}</div>
        )}
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            disabled={busy}
            onClick={() => void load()}
            className="inline-flex items-center gap-2 rounded-lg border border-border px-3 py-1.5 text-sm"
          >
            {busy && <Loader2 className="h-4 w-4 animate-spin" />}
            Refresh
          </button>
          <button
            type="button"
            onClick={() => void downloadCSV()}
            className="inline-flex items-center gap-2 rounded-lg bg-primary px-3 py-1.5 text-sm font-medium text-primary-foreground"
          >
            <Download className="h-4 w-4" />
            Export sales CSV
          </button>
        </div>

        {summary && (
          <section className="grid gap-3 sm:grid-cols-2">
            {[
              ["Sales", formatMoney(summary.sales_minor ?? 0)],
              ["Sale count", String(summary.sale_count ?? 0)],
              ["On-hand SKUs", String(summary.on_hand_sku_count ?? 0)],
              ["Low stock bins", String(summary.low_stock_count ?? 0)],
              ["Shift variances", String(summary.open_variances ?? 0)],
            ].map(([label, val]) => (
              <div key={label} className="rounded-xl border border-border bg-card p-4">
                <p className="text-xs text-muted-foreground">{label}</p>
                <p className="text-xl font-semibold">{val}</p>
              </div>
            ))}
          </section>
        )}

        <section className="rounded-xl border border-border bg-card p-4">
          <h2 className="mb-3 font-semibold">Sales by SKU</h2>
          {sales.length === 0 ? (
            <p className="text-sm text-muted-foreground">No POS sales in window (last 7 days).</p>
          ) : (
            <ul className="space-y-2 text-sm">
              {sales.slice(0, 30).map((row) => (
                <li key={row.key} className="flex justify-between border-b border-border/50 py-1">
                  <span>{row.key}</span>
                  <span className="text-muted-foreground">
                    {formatMoney(row.sales_minor)} · {row.units} u
                  </span>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </PageChrome>
  );
}
