"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { DemandSourceChips } from "@pegasusx/ui-kit/portal";
import { Loader2 } from "lucide-react";
import { apiFetch } from "@/lib/auth";

type SellThroughItem = {
  retailer_id?: string;
  location_id?: string;
  sku_id: string;
  day: string;
  qty_sold: number;
  qty_voided?: number;
  net_sold: number;
  qty_on_hand_eod?: number;
  source?: string;
};

type SellThroughResponse = {
  source?: string;
  items?: SellThroughItem[];
};

/**
 * L3 sell-through insights — POS floor velocity with STORE_POS chips.
 * GET /v1/retailer/insights/sell-through
 */
export function SellThroughPanel() {
  const t = usePortalT();
  const [items, setItems] = useState<SellThroughItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiFetch("/v1/retailer/insights/sell-through");
      if (!res.ok) {
        setError(`sell_through_${res.status}`);
        setItems([]);
        return;
      }
      const data = (await res.json()) as SellThroughResponse;
      setItems(Array.isArray(data.items) ? data.items : []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("retailer_desktop.residual.text.load_failed"));
      setItems([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <div className="mb-8 p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
      <div className="flex flex-wrap items-center justify-between gap-3 mb-4">
        <div>
          <h3 className="md-typescale-title-medium font-light text-[var(--desk-text-primary)]">
            Store sell-through
          </h3>
          <p className="text-xs text-[var(--desk-text-tertiary)] mt-1">
            Floor POS net sold — feeds reorder sources (Store POS)
          </p>
        </div>
        <DemandSourceChips sources={["STORE_POS"]} />
      </div>

      {loading ? (
        <div className="flex justify-center py-6">
          <Loader2 size={18} className="animate-spin text-[var(--desk-text-tertiary)]" />
        </div>
      ) : error ? (
        <p className="text-sm text-[var(--desk-warning)]">{error}</p>
      ) : items.length === 0 ? (
        <p className="text-sm text-[var(--desk-text-secondary)]">
          No sell-through rows yet. Complete POS sales to populate daily rollups.
        </p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="text-xs uppercase tracking-wide text-[var(--desk-text-tertiary)]">
                <th className="py-2 pr-3">{t("retailer_desktop.insights.sell_through_panel.text.day")}</th>
                <th className="py-2 pr-3">SKU</th>
                <th className="py-2 pr-3">{t("retailer_desktop.insights.sell_through_panel.text.net_sold")}</th>
                <th className="py-2 pr-3">{t("retailer_desktop.hq.text.sold")}</th>
                <th className="py-2 pr-3">{t("retailer_desktop.hq.text.voided")}</th>
                <th className="py-2">{t("retailer_desktop.insights.sell_through_panel.text.source")}</th>
              </tr>
            </thead>
            <tbody>
              {items.slice(0, 20).map((row) => (
                <tr
                  key={`${row.day}:${row.sku_id}:${row.location_id || ""}`}
                  className="border-t border-[var(--desk-border)]"
                >
                  <td className="py-2 pr-3 font-mono text-xs">{row.day}</td>
                  <td className="py-2 pr-3">{row.sku_id}</td>
                  <td className="py-2 pr-3 font-medium">{row.net_sold}</td>
                  <td className="py-2 pr-3">{row.qty_sold}</td>
                  <td className="py-2 pr-3">{row.qty_voided ?? 0}</td>
                  <td className="py-2">
                    <DemandSourceChips
                      sources={row.source ? [row.source] : ["STORE_POS"]}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
