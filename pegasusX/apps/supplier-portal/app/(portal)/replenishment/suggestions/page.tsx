"use client";

import Link from "next/link";
import type { Route } from "next";
import { useCallback, useEffect, useMemo, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { ReorderSuggestionRow } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

export default function ReorderSuggestionsPage() {
  const [suggestions, setSuggestions] = useState<ReorderSuggestionRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [retailerFilter, setRetailerFilter] = useState("");
  const [skuSearch, setSkuSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("OPEN");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [acting, setActing] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await api.listReorderSuggestions({
        retailerId: retailerFilter.trim() || undefined,
        status: statusFilter,
        sku: skuSearch.trim() || undefined,
      });
      setSuggestions(resp.suggestions ?? []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "load_suggestions_failed");
    } finally {
      setLoading(false);
    }
  }, [retailerFilter, skuSearch, statusFilter]);

  useEffect(() => {
    void load();
  }, [load]);

  const rowKey = (row: ReorderSuggestionRow) => `${row.retailer_id}:${row.sku}`;

  const allKeys = useMemo(() => suggestions.map(rowKey), [suggestions]);

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
      setError(err instanceof Error ? err.message : "dismiss_failed");
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
      setError(err instanceof Error ? err.message : "create_draft_failed");
    } finally {
      setActing(null);
    }
  };

  const bulkCreateDrafts = async () => {
    const items = suggestions
      .filter((row) => selected.has(rowKey(row)))
      .map((row) => ({ retailer_id: row.retailer_id, sku: row.sku }));
    if (items.length === 0) return;
    setActing("bulk");
    try {
      await api.bulkCreateDraftsFromReorderSuggestions({ items });
      setSelected(new Set());
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "bulk_create_failed");
    } finally {
      setActing(null);
    }
  };

  return (
    <PageChrome
      icon="inventory"
      title="Reorder suggestions"
      description="OPEN replenishment suggestions from demand signals — create draft orders through normal order capture."
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
          placeholder="Retailer ID"
          value={retailerFilter}
          onChange={(e) => setRetailerFilter(e.target.value)}
        />
        <input
          className="md-input min-w-[180px]"
          placeholder="SKU search"
          value={skuSearch}
          onChange={(e) => setSkuSearch(e.target.value)}
        />
        <select className="md-input" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
          <option value="OPEN">OPEN</option>
          <option value="DISMISSED">DISMISSED</option>
          <option value="CONVERTED">CONVERTED</option>
        </select>
        <button type="button" className="md-btn md-btn-outlined" onClick={() => void load()}>
          Apply filters
        </button>
      </div>

      {suggestions.length === 0 && !loading ? (
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
                    aria-label="Select all"
                  />
                </th>
                <th className="px-3 py-2">Retailer</th>
                <th className="px-3 py-2">SKU</th>
                <th className="px-3 py-2">Suggested qty</th>
                <th className="px-3 py-2">Adj. demand / day</th>
                <th className="px-3 py-2">Stock</th>
                <th className="px-3 py-2">In-flight</th>
                <th className="px-3 py-2">By date</th>
                <th className="px-3 py-2">Status</th>
                <th className="px-3 py-2">Actions</th>
              </tr>
            </thead>
            <tbody>
              {suggestions.map((row) => {
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
                    <td className="px-3 py-2">{row.suggested_qty}</td>
                    <td className="px-3 py-2">{row.adjusted_demand_per_day.toFixed(2)}</td>
                    <td className="px-3 py-2">{row.current_stock}</td>
                    <td className="px-3 py-2">{row.in_flight_qty}</td>
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
