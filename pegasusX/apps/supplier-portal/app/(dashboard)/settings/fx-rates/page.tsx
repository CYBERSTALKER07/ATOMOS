"use client";

import { useMemo } from "react";
import { PageChrome } from "@/components/PageChrome";
import { PageSection } from "@/components/PageSection";
import { useLiveData } from "@/lib/hooks";
import { RefreshCw } from "lucide-react";
import type { FxRateRow, FxRatesListResponse } from "@pegasusx/types";
import { sessionPackCurrency } from "@pegasusx/api-core";

const DEFAULT_SCALE = 100_000_000;

export default function FxRatesPage() {
  const { data, loading, isRefreshing, mutate } = useLiveData<FxRatesListResponse>("/v1/supplier/fx-rates?limit=100");

  const rates = useMemo(() => data?.rates ?? [], [data]);

  function formatRate(row: FxRateRow): string {
    const scale = row.scale > 0 ? row.scale : DEFAULT_SCALE;
    return (row.rate_scaled / scale).toLocaleString(undefined, { maximumFractionDigits: 8 });
  }

  return (
    <PageChrome
      title="FX rates"
      description={`Read-only ConvertMinor quotes (operating currency ${sessionPackCurrency() || "pack"}). Writes are platform-admin only. Missing rates fail closed — never silent 1:1.`}
      actions={
        <button
          type="button"
          disabled={loading || isRefreshing}
          onClick={mutate}
          className="portal-btn portal-btn--ghost h-11 px-5 rounded-xl font-light"
        >
          <RefreshCw size={16} className={`mr-2 ${isRefreshing ? "animate-spin" : ""}`} />
          {isRefreshing ? "Syncing" : "Sync"}
        </button>
      }
    >
      <PageSection title="Latest rates">
        {loading && !data ? (
          <p className="p-4 md-typescale-body-small">Loading…</p>
        ) : rates.length === 0 ? (
          <p className="p-4 md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
            No FX rates yet. Identity pairs seed on bootstrap. Platform admin upserts cross-currency pairs.
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
