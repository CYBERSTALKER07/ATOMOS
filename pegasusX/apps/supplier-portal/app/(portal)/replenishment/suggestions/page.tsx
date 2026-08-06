"use client";

import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import type { Route } from "next";
import { useCallback, useEffect, useMemo, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { ReorderSuggestionRow } from "@pegasusx/types";
import { DemandSourceChips } from "@pegasusx/ui-kit/portal";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

export default function ReorderSuggestionsPage() {
  const t = usePortalT();
  const [suggestions, setSuggestions] = useState<ReorderSuggestionRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [retailerFilter, setRetailerFilter] = useState("");
  const [skuSearch, setSkuSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("OPEN");
  const [sourceFilter, setSourceFilter] = useState<"ALL" | "STORE_POS" | "WHOLESALE">("ALL");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [acting, setActing] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await api.listReorderSuggestions({
        retailerId: retailerFilter.trim() || undefined,
        status: statusFilter,
        sku: skuSearch.trim() || undefined,
        // Server-side POS filter; wholesale-only stays client-side (exclusive).
        source: sourceFilter === "STORE_POS" ? "STORE_POS" : undefined,
      });
      setSuggestions(resp.suggestions ?? []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_suggestions_failed"));
    } finally {
      setLoading(false);
    }
  }, [retailerFilter, skuSearch, statusFilter, sourceFilter]);

  useEffect(() => {
    void load();
  }, [load]);

  const rowKey = (row: ReorderSuggestionRow) => `${row.retailer_id}:${row.sku}`;

  const filteredSuggestions = useMemo(() => {
    if (sourceFilter === "ALL") return suggestions;
    return suggestions.filter((row) => {
      const src = row.sources?.length ? row.sources : ["WHOLESALE_HISTORY"];
      if (sourceFilter === "STORE_POS") return src.includes("STORE_POS");
      // WHOLESALE only — has wholesale and no POS
      return src.includes("WHOLESALE_HISTORY") && !src.includes("STORE_POS");
    });
  }, [suggestions, sourceFilter]);

  const allKeys = useMemo(() => filteredSuggestions.map(rowKey), [filteredSuggestions]);

  const toggleAll = () => {
    if (selected.size === allKeys.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(allKeys));
    }
  };

  const toggleRow = (key: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const dismiss = async (row: ReorderSuggestionRow) => {
    setActing(rowKey(row));
    try {
      await api.dismissReorderSuggestion({ retailer_id: row.retailer_id, sku: row.sku });
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.dismiss_failed"));
    } finally {
      setActing(null);
    }
  };

  const createDraft = async (row: ReorderSuggestionRow) => {
    setActing(rowKey(row));
    try {
      await api.createDraftFromReorderSuggestion({
        retailer_id: row.retailer_id,
        sku: row.sku,
      });
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.create_draft_failed"));
    } finally {
      setActing(null);
    }
  };

  const bulkCreateDrafts = async () => {
    const items = filteredSuggestions
      .filter((row) => selected.has(rowKey(row)))
      .map((row) => ({ retailer_id: row.retailer_id, sku: row.sku }));
    if (items.length === 0) return;
    setActing("bulk");
    try {
      await api.bulkCreateDraftsFromReorderSuggestions({ items });
      setSelected(new Set());
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.bulk_create_failed"));
    } finally {
      setActing(null);
    }
  };

  return (
    <PageChrome
      icon="inventory"
      title={t("supplier_portal.replenishment.suggestions.text.reorder_suggestions")}
      description={t("supplier_portal.residual.text.open_replenishment_suggestions_from_demand_signals_create_draft_")}
      loading={loading}
      skeletonVariant="table"
      error={error}
      actions={
        <div className="flex gap-2 items-center">
          <button
            type="button"
            className="md-btn md-btn-outlined"
            disabled={selected.size === 0 || acting === "bulk"}
            onClick={() => void bulkCreateDrafts()}
          >
            Create drafts for selected ({selected.size})
          </button>
          <Link href={"/operations/replenishment-policies" as Route} className="md-btn md-btn-text">
            Policies
          </Link>
        </div>
      }
    >
      <div className="flex flex-wrap gap-3 mb-4">
        <input
          className="md-input min-w-[180px]"
          placeholder={t("supplier_portal.chargebacks.text.retailer_id")}
          value={retailerFilter}
          onChange={(e) => setRetailerFilter(e.target.value)}
        />
        <input
          className="md-input min-w-[180px]"
          placeholder={t("supplier_portal.replenishment.suggestions.text.sku_search")}
          value={skuSearch}
          onChange={(e) => setSkuSearch(e.target.value)}
        />
        <select className="md-input" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
          <option value="OPEN">OPEN</option>
          <option value="DISMISSED">DISMISSED</option>
          <option value="CONVERTED">CONVERTED</option>
        </select>
        <label className="text-xs flex flex-col gap-1">
          Source
          <select
            className="md-input"
            value={sourceFilter}
            onChange={(e) =>
              setSourceFilter(e.target.value as "ALL" | "STORE_POS" | "WHOLESALE")
            }
          >
            <option value="ALL">{t("supplier_portal.replenishment.suggestions.text.all_sources")}</option>
            <option value="STORE_POS">{t("supplier_portal.replenishment.suggestions.text.has_store_pos")}</option>
            <option value="WHOLESALE">{t("supplier_portal.replenishment.suggestions.text.wholesale_only")}</option>
          </select>
        </label>
        <button type="button" className="md-btn md-btn-outlined" onClick={() => void load()}>
          Apply filters
        </button>
      </div>

      {filteredSuggestions.length === 0 && !loading ? (
        <p className="text-sm" style={{ color: "var(--desk-text-secondary)" }}>
          No suggestions for the current filters. The reorder worker will re-suggest when demand remains.
        </p>
      ) : (
        <div className="overflow-x-auto border rounded-lg" style={{ borderColor: "var(--desk-border)" }}>
          <table className="w-full text-left text-sm">
            <thead className="uppercase text-xs" style={{ background: "var(--desk-surface-muted)" }}>
              <tr>
                <th className="px-3 py-2">
                  <input
                    type="checkbox"
                    checked={selected.size > 0 && selected.size === allKeys.length}
                    onChange={toggleAll}
                    aria-label={t("supplier_portal.replenishment.suggestions.text.select_all")}
                  />
                </th>
                <th className="px-3 py-2">{t("supplier_portal.analytics.demand.flywheel.text.retailer")}</th>
                <th className="px-3 py-2">SKU</th>
                <th className="px-3 py-2">{t("supplier_portal.replenishment.suggestions.text.sources")}</th>
                <th className="px-3 py-2">{t("supplier_portal.replenishment.suggestions.text.pos_vel_day")}</th>
                <th className="px-3 py-2">{t("supplier_portal.replenishment.suggestions.text.base_demand_day")}</th>
                <th className="px-3 py-2">{t("supplier_portal.replenishment.suggestions.text.suggested_qty")}</th>
                <th className="px-3 py-2">{t("supplier_portal.replenishment.suggestions.text.adj_demand_day")}</th>
                <th className="px-3 py-2">{t("portal.nav.stock")}</th>
                <th className="px-3 py-2">{t("supplier_portal.replenishment.suggestions.text.in_flight")}</th>
                <th className="px-3 py-2">{t("supplier_portal.replenishment.suggestions.text.safety_stock")}</th>
                <th className="px-3 py-2">{t("supplier_portal.replenishment.suggestions.text.by_date")}</th>
                <th className="px-3 py-2">{t("supplier_portal.compliance.text.status")}</th>
                <th className="px-3 py-2">{t("supplier_portal.catalog.components.catalog_table.text.actions")}</th>
              </tr>
            </thead>
            <tbody>
              {filteredSuggestions.map((row) => {
                const key = rowKey(row);
                return (
                  <tr key={key} className="border-t" style={{ borderColor: "var(--desk-border)" }}>
                    <td className="px-3 py-2">
                      <input
                        type="checkbox"
                        checked={selected.has(key)}
                        onChange={() => toggleRow(key)}
                        aria-label={`Select ${key}`}
                      />
                    </td>
                    <td className="px-3 py-2">
                      <div className="font-medium">{row.retailer_name || row.retailer_id}</div>
                      <div className="text-xs font-mono opacity-70">{row.retailer_id}</div>
                    </td>
                    <td className="px-3 py-2">
                      <div>{row.sku_name || row.sku}</div>
                      <div className="text-xs font-mono opacity-70">{row.sku}</div>
                    </td>
                    <td className="px-3 py-2">
                      <DemandSourceChips sources={row.sources} />
                    </td>
                    <td className="px-3 py-2 font-mono text-xs">
                      {row.sell_through_velocity != null && row.sell_through_velocity > 0
                        ? row.sell_through_velocity.toFixed(2)
                        : "—"}
                    </td>
                    <td className="px-3 py-2 font-mono text-xs">
                      {row.base_demand_per_day != null && row.base_demand_per_day > 0
                        ? row.base_demand_per_day.toFixed(2)
                        : "—"}
                    </td>
                    <td className="px-3 py-2">{row.suggested_qty}</td>
                    <td className="px-3 py-2">{row.adjusted_demand_per_day.toFixed(2)}</td>
                    <td className="px-3 py-2">{row.current_stock}</td>
                    <td className="px-3 py-2">{row.in_flight_qty}</td>
                    <td className="px-3 py-2 font-mono text-xs">
                      {row.safety_stock != null && row.safety_stock > 0
                        ? row.safety_stock.toFixed(1)
                        : "—"}
                    </td>
                    <td className="px-3 py-2">{row.suggested_by_date}</td>
                    <td className="px-3 py-2">{row.status}</td>
                    <td className="px-3 py-2 flex gap-2">
                      <button
                        type="button"
                        className="md-btn md-btn-text"
                        disabled={acting === key || row.status !== "OPEN"}
                        onClick={() => void createDraft(row)}
                      >
                        Create draft
                      </button>
                      <button
                        type="button"
                        className="md-btn md-btn-text"
                        disabled={acting === key || row.status !== "OPEN"}
                        onClick={() => void dismiss(row)}
                      >
                        Dismiss
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </PageChrome>
  );

}
