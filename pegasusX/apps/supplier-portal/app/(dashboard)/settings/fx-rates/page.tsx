"use client";

import { useMemo, useState } from "react";
import { PageChrome } from "@/components/PageChrome";
import { PageSection } from "@/components/PageSection";
import { useLiveData } from "@/lib/hooks";
import { supplierFetch } from "@/lib/auth";
import { Plus, RefreshCw } from "lucide-react";
import type { FxRateRow, FxRatesListResponse } from "@pegasusx/types";

const DEFAULT_SCALE = 100_000_000;

export default function FxRatesPage() {
  const { data, loading, isRefreshing, mutate } = useLiveData<FxRatesListResponse>("/v1/admin/fx-rates?limit=100");
  const [isCreating, setIsCreating] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [formError, setFormError] = useState<string | null>(null);
  const [formData, setFormData] = useState({
    base_currency: "USD",
    quote_currency: "UZS",
    human_rate: "12750",
  });

  const rates = useMemo(() => data?.rates ?? [], [data]);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setIsSubmitting(true);
    setFormError(null);
    try {
      const human = Number(formData.human_rate);
      if (!Number.isFinite(human) || human <= 0) {
        throw new Error("Rate must be a positive number (quote per 1 base)");
      }
      const rateScaled = Math.round(human * DEFAULT_SCALE);
      const res = await supplierFetch("/v1/admin/fx-rates", {
        method: "PUT",
        body: JSON.stringify({
          base_currency: formData.base_currency.trim().toUpperCase(),
          quote_currency: formData.quote_currency.trim().toUpperCase(),
          rate_scaled: rateScaled,
          scale: DEFAULT_SCALE,
          source: "ADMIN",
        }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || body.code || `Save failed (${res.status})`);
      }
      setIsCreating(false);
      mutate();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : "Save failed");
    } finally {
      setIsSubmitting(false);
    }
  }

  function formatRate(row: FxRateRow): string {
    const scale = row.scale > 0 ? row.scale : DEFAULT_SCALE;
    return (row.rate_scaled / scale).toLocaleString(undefined, { maximumFractionDigits: 8 });
  }

  return (
    <PageChrome
      title="FX rates"
      description="Admin FX quotes for ConvertMinor (operating currency UZS). Missing rates fail closed — never silent 1:1."
      actions={
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => setIsCreating((v) => !v)}
            className="portal-btn portal-btn--primary h-11 px-5 rounded-xl font-light"
          >
            <Plus size={16} className="mr-2" />
            {isCreating ? "Cancel" : "Upsert rate"}
          </button>
          <button
            type="button"
            disabled={loading || isRefreshing}
            onClick={mutate}
            className="portal-btn portal-btn--ghost h-11 px-5 rounded-xl font-light"
          >
            <RefreshCw size={16} className={`mr-2 ${isRefreshing ? "animate-spin" : ""}`} />
            {isRefreshing ? "Syncing" : "Sync"}
          </button>
        </div>
      }
    >
      {isCreating ? (
        <div className="mb-6">
          <PageSection title="Upsert FX rate">
            <form onSubmit={handleCreate} className="space-y-4 p-4">
              <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                <label className="flex flex-col gap-1 text-sm">
                  Base currency
                  <input
                    className="portal-input"
                    value={formData.base_currency}
                    onChange={(e) => setFormData({ ...formData, base_currency: e.target.value })}
                    maxLength={3}
                    required
                  />
                </label>
                <label className="flex flex-col gap-1 text-sm">
                  Quote currency
                  <input
                    className="portal-input"
                    value={formData.quote_currency}
                    onChange={(e) => setFormData({ ...formData, quote_currency: e.target.value })}
                    maxLength={3}
                    required
                  />
                </label>
                <label className="flex flex-col gap-1 text-sm">
                  Rate (quote per 1 base)
                  <input
                    className="portal-input"
                    type="number"
                    step="any"
                    min={0}
                    value={formData.human_rate}
                    onChange={(e) => setFormData({ ...formData, human_rate: e.target.value })}
                    required
                  />
                </label>
              </div>
              {formError ? (
                <p className="text-sm" style={{ color: "var(--desk-danger)" }}>
                  {formError}
                </p>
              ) : null}
              <button type="submit" className="portal-btn portal-btn--primary" disabled={isSubmitting}>
                {isSubmitting ? "Saving…" : "Save rate"}
              </button>
            </form>
          </PageSection>
        </div>
      ) : null}

      <PageSection title="Latest rates">
        {loading && !data ? (
          <p className="p-4 md-typescale-body-small">Loading…</p>
        ) : rates.length === 0 ? (
          <p className="p-4 md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
            No FX rates yet. Seed identity UZS/UZS runs on bootstrap; upsert USD/UZS for cross-currency metering.
          </p>
        ) : (
          <table className="desk-table w-full">
            <thead>
              <tr style={{ color: "var(--desk-text-secondary)" }}>
                <th className="p-3 text-left">Pair</th>
                <th className="p-3 text-left">Rate</th>
                <th className="p-3 text-left">Scaled</th>
                <th className="p-3 text-left">Source</th>
                <th className="p-3 text-left">Effective</th>
              </tr>
            </thead>
            <tbody>
              {rates.map((row) => (
                <tr key={row.rate_id || `${row.base_currency}-${row.quote_currency}-${row.effective_at}`}>
                  <td className="p-3 font-mono text-sm">
                    {row.base_currency}/{row.quote_currency}
                  </td>
                  <td className="p-3">{formatRate(row)}</td>
                  <td className="p-3 font-mono text-sm">{row.rate_scaled}</td>
                  <td className="p-3">{row.source}</td>
                  <td className="p-3 md-typescale-body-small">
                    {row.effective_at ? new Date(row.effective_at).toLocaleString() : "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </PageSection>
    </PageChrome>
  );
}
