"use client";

import React, { useCallback, useEffect, useMemo, useState } from "react";
import { usePortalT } from "@/lib/i18n";
import { createSupplierApi } from "@/lib/api";
import { sessionSupplierId } from "@/lib/supplier-scope";
import type { ForecastAccuracyDailyRow } from "@pegasusx/types";

const api = createSupplierApi();

interface AccuracyAggregate {
  productId: string;
  warehouseId: string;
  wape28: number;
  bias28: number;
  trackingSignal: number;
  alertTs: boolean;
  days: number;
}

function fmtPct(v: number): string {
  if (!Number.isFinite(v)) return "—";
  return `${(v * 100).toFixed(1)}%`;
}

function fmtTs(v: number): string {
  if (!Number.isFinite(v)) return "—";
  return v.toFixed(2);
}

/** Reduce per-day rows to the latest row per (warehouse, product). */
function aggregate(rows: ForecastAccuracyDailyRow[]): AccuracyAggregate[] {
  const latest = new Map<string, ForecastAccuracyDailyRow>();
  for (const r of rows) {
    const key = `${r.WarehouseId}::${r.ProductId}`;
    const prev = latest.get(key);
    if (!prev || r.ForecastDate > prev.ForecastDate) {
      latest.set(key, r);
    }
  }
  return Array.from(latest.values())
    .map((r) => ({
      productId: r.ProductId,
      warehouseId: r.WarehouseId,
      wape28: r.Wape28,
      bias28: r.Bias28,
      trackingSignal: r.TrackingSignal,
      alertTs: r.AlertTs,
      days: r.SampleDays28,
    }))
    .sort((a, b) => b.wape28 - a.wape28);
}

export function ForecastAccuracyPanel() {
  const t = usePortalT();
  const [rows, setRows] = useState<ForecastAccuracyDailyRow[] | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [days, setDays] = useState(28);

  const load = useCallback(async () => {
    const supplierId = sessionSupplierId();
    if (!supplierId) {
      setError("No supplier session");
      setLoading(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const resp = await api.getForecastAccuracy({ supplierId, days });
      setRows(resp.items ?? []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load forecast accuracy");
      setRows(null);
    } finally {
      setLoading(false);
    }
  }, [days]);

  useEffect(() => {
    void load();
  }, [load]);

  const aggregates = useMemo(() => aggregate(rows ?? []), [rows]);
  const alerts = useMemo(() => aggregates.filter((a) => a.alertTs).length, [aggregates]);
  const avgWape = useMemo(() => {
    if (aggregates.length === 0) return NaN;
    return aggregates.reduce((s, a) => s + a.wape28, 0) / aggregates.length;
  }, [aggregates]);

  return (
    <section className="desk-card p-6 mt-6 overflow-x-auto">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="bento-card-title">Forecast accuracy</h2>
          <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
            Rolling WAPE / bias / tracking signal per warehouse &amp; product. Tracking-signal
            alerts (|TS| &gt; 4) flag systematic over/under-forecasting.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <select
            className="portal-btn portal-btn--text"
            value={days}
            onChange={(e) => setDays(Number(e.target.value))}
            aria-label="Window days"
          >
            {[7, 14, 28, 60, 90].map((d) => (
              <option key={d} value={d}>
                {d}d
              </option>
            ))}
          </select>
          <button type="button" className="portal-btn portal-btn--text" onClick={() => void load()}>
            Refresh
          </button>
        </div>
      </div>

      {loading ? (
        <p className="md-typescale-body-small mt-4" style={{ color: "var(--desk-text-secondary)" }}>
          Loading accuracy…
        </p>
      ) : error ? (
        <p className="md-typescale-body-small mt-4" style={{ color: "var(--desk-danger, #b3261e)" }}>
          {error}
        </p>
      ) : aggregates.length === 0 ? (
        <p className="md-typescale-body-small mt-4" style={{ color: "var(--desk-text-secondary)" }}>
          No accuracy data yet. The nightly accuracy pass runs once baseline + actual demand exist
          (FORECAST_ACCURACY_ENABLED).
        </p>
      ) : (
        <>
          <div className="flex gap-6 mt-4 mb-2">
            <Stat label="Series" value={String(aggregates.length)} />
            <Stat label="Avg WAPE (28d)" value={fmtPct(avgWape)} />
            <Stat
              label="TS alerts"
              value={String(alerts)}
              tone={alerts > 0 ? "danger" : undefined}
            />
          </div>
          <table className="desk-table w-full mt-2">
            <thead>
              <tr style={{ color: "var(--desk-text-secondary)" }}>
                <th className="md-typescale-label-medium p-3 text-left font-medium">Product</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">Warehouse</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">WAPE 28d</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">Bias 28d</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">Tracking signal</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">
                  {t("supplier_portal.compliance.text.status")}
                </th>
              </tr>
            </thead>
            <tbody>
              {aggregates.map((a) => (
                <tr
                  key={`${a.warehouseId}::${a.productId}`}
                  style={{ borderTop: "1px solid var(--desk-border)" }}
                >
                  <td className="p-3 md-typescale-body-medium font-mono text-sm">{a.productId}</td>
                  <td className="p-3 md-typescale-body-medium">{a.warehouseId || "—"}</td>
                  <td className="p-3 md-typescale-body-medium">{fmtPct(a.wape28)}</td>
                  <td
                    className="p-3 md-typescale-body-medium"
                    style={{ color: a.bias28 > 0.05 ? "#b3261e" : a.bias28 < -0.05 ? "#7a5900" : undefined }}
                  >
                    {fmtPct(a.bias28)}
                  </td>
                  <td className="p-3 md-typescale-body-medium">{fmtTs(a.trackingSignal)}</td>
                  <td className="p-3 md-typescale-body-medium">
                    {a.alertTs ? (
                      <span style={{ color: "#b3261e", fontWeight: 600 }}>Alert</span>
                    ) : (
                      "OK"
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}
    </section>
  );
}

function Stat({
  label,
  value,
  tone,
}: {
  label: string;
  value: string;
  tone?: "danger";
}) {
  return (
    <div>
      <div
        className="md-typescale-label-small"
        style={{ color: "var(--desk-text-secondary)" }}
      >
        {label}
      </div>
      <div
        className="md-typescale-title-large"
        style={{ color: tone === "danger" ? "#b3261e" : undefined }}
      >
        {value}
      </div>
    </div>
  );
}
