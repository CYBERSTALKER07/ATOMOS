"use client";

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { createSupplierApi } from "@/lib/api";
import { ForecastConfidenceCard } from "@/components/ForecastConfidenceCard";
import {
  forecastConfidenceFromDemand,
  formatForecastUpdatedAt,
  isForecastStale,
} from "@/lib/forecast-confidence";

const api = createSupplierApi();

type MEIOSummary = {
  warehouses_scanned: number;
  skus_analyzed: number;
  insights_generated: number;
  transfer_recommendations: number;
  warehouse_balances: Array<{
    warehouse_id: string;
    critical_skus: number;
    warning_skus: number;
    avg_days_cover: number;
    target_stock?: number;
    on_hand_stock?: number;
  }>;
};

export default function MeioNetworkPanel() {
  const t = usePortalT();
  const [summary, setSummary] = useState<MEIOSummary | null>(null);
  const [demandGeneratedAt, setDemandGeneratedAt] = useState<string | undefined>();
  const [demandConfidence, setDemandConfidence] = useState<ReturnType<typeof forecastConfidenceFromDemand> | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [meioRes, demandResp] = await Promise.all([
          supplierFetch("/v1/supplier/meio/network-summary"),
          api.getSupplierDemandToday(),
        ]);
        if (!meioRes.ok) {
          throw new Error(`MEIO summary ${meioRes.status}`);
        }
        const data = (await meioRes.json()) as MEIOSummary;
        if (!cancelled) {
          setSummary(data);
          setDemandGeneratedAt(demandResp.generated_at);
          setDemandConfidence(forecastConfidenceFromDemand(demandResp));
        }
      } catch (err) {
        if (!cancelled) setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.meio_unavailable"));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <div className="p-5 flex flex-col gap-3 h-full">
      <div>
        <h2 className="md-typescale-title-medium" style={{ color: "var(--desk-text-secondary)" }}>
          MEIO network
        </h2>
        <p className="md-typescale-body-small mt-1" style={{ color: "var(--desk-text-secondary)" }}>
          Multi-echelon inventory optimization across warehouses.
        </p>
      </div>
      {error ? (
        <p className="md-typescale-body-small" style={{ color: "var(--desk-danger)" }}>
          {error}
        </p>
      ) : summary ? (
        <>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
            <Stat label={t("portal.nav.warehouses")} value={summary.warehouses_scanned} />
            <Stat label={t("supplier_portal.residual.text.skus")} value={summary.skus_analyzed} />
            <Stat label={t("portal.nav.insights")} value={summary.insights_generated} />
            <Stat label={t("portal.nav.transfers")} value={summary.transfer_recommendations} />
          </div>
          {demandConfidence ? (
            <ForecastConfidenceCard
              confidence={demandConfidence}
              updatedAt={formatForecastUpdatedAt(demandGeneratedAt)}
              stale={isForecastStale(demandGeneratedAt)}
            />
          ) : null}
          {summary.warehouse_balances?.length ? (
            <ul className="divide-y rounded-lg border" style={{ borderColor: "var(--desk-border)" }}>
              {summary.warehouse_balances.map((wh) => (
                <li key={wh.warehouse_id} className="flex flex-wrap gap-3 p-3 md-typescale-body-small">
                  <span className="font-mono">{wh.warehouse_id}</span>
                  <span>critical {wh.critical_skus}</span>
                  <span>warning {wh.warning_skus}</span>
                  {wh.on_hand_stock != null ? <span>on-hand {wh.on_hand_stock}</span> : null}
                  {wh.target_stock != null ? <span>target {wh.target_stock}</span> : null}
                </li>
              ))}
            </ul>
          ) : null}
        </>
      ) : (
        <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
          Loading network scan…
        </p>
      )}
    </div>
  );
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <div className="rounded-lg p-3" style={{ background: "var(--desk-surface-raised)" }}>
      <div className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
        {label}
      </div>
      <div className="md-kpi-value text-xl">{value}</div>
    </div>
  );
}
