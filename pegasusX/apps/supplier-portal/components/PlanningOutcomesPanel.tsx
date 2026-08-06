"use client";

import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import type { Route } from "next";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { ForecastConfidenceCard } from "@/components/ForecastConfidenceCard";
import {
  forecastConfidenceFromDemand,
  formatForecastUpdatedAt,
  isForecastStale,
} from "@/lib/forecast-confidence";
import type {
  SupplierDemandSummaryResponse,
  SupplierMEIONetworkSummary,
  SupplierReplenishmentPolicy,
} from "@pegasusx/types";

const api = createSupplierApi();

type PipelineStage = {
  key: string;
  title: string;
  metric: string;
  detail: string;
  href: Route;
};

export default function PlanningOutcomesPanel() {
  const t = usePortalT();
  const [demand, setDemand] = useState<SupplierDemandSummaryResponse | null>(null);
  const [meio, setMeio] = useState<SupplierMEIONetworkSummary | null>(null);
  const [policy, setPolicy] = useState<SupplierReplenishmentPolicy | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [demandResp, meioResp, policyResp] = await Promise.all([
          api.getSupplierDemandToday(),
          api.getSupplierMEIONetworkSummary(),
          api.getSupplierReplenishmentPolicies(),
        ]);
        if (!cancelled) {
          setDemand(demandResp);
          setMeio(meioResp);
          setPolicy(policyResp);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.planning_outcomes_unavailable"));
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const demandConfidence = demand ? forecastConfidenceFromDemand(demand) : null;
  const touchlessEnabled =
    policy?.auto_approve_stable || policy?.auto_approve_predictive_push || false;
  const criticalSkus =
    meio?.warehouse_balances.reduce((sum, node) => sum + node.critical_skus, 0) ?? 0;

  const stages: PipelineStage[] = [
    {
      key: "baseline",
      title: t("supplier_portal.residual.text.demand_baseline"),
      metric: demand ? `${demand.prediction_count.toLocaleString()} SKUs` : "—",
      detail: demand
        ? `${demand.total_pallets.toLocaleString()} pallets · ${demand.total_retailers} retailers`
        : "Today's forecast projection",
      href: "/analytics" as Route,
    },
    {
      key: "insights",
      title: t("factory_portal.insights.text.replenishment_insights"),
      metric: meio ? meio.insights_generated.toLocaleString() : "—",
      detail: "Predictive push and warehouse scan outputs",
      href: "/operations" as Route,
    },
    {
      key: "touchless",
      title: t("supplier_portal.residual.text.touchless_transfers"),
      metric: meio ? meio.transfer_recommendations.toLocaleString() : "—",
      detail: touchlessEnabled
        ? `Auto-approve on · max ${policy?.max_daily_transfer_units?.toLocaleString() ?? "—"} units/day`
        : "Manual approval — configure policies",
      href: "/operations/replenishment-policies" as Route,
    },
    {
      key: "meio",
      title: t("supplier_portal.residual.text.meio_network"),
      metric: meio ? `${meio.warehouses_scanned} WH` : "—",
      detail: meio
        ? `${meio.skus_analyzed.toLocaleString()} SKUs · ${criticalSkus} critical`
        : "Multi-echelon inventory balance",
      href: "/analytics" as Route,
    },
  ];

  return (
    <div className="p-5 flex flex-col gap-4 h-full">
      <div>
        <h2 className="md-typescale-title-medium" style={{ color: "var(--desk-text-secondary)" }}>
          Planning outcomes
        </h2>
        <p className="md-typescale-body-small mt-1" style={{ color: "var(--desk-text-secondary)" }}>
          Baseline → insights → touchless transfers → MEIO network health.
        </p>
      </div>

      {error ? (
        <p className="md-typescale-body-small" style={{ color: "var(--desk-danger)" }}>
          {error}
        </p>
      ) : loading ? (
        <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
          Loading planning pipeline…
        </p>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-4 gap-3">
            {stages.map((stage, index) => (
              <div key={stage.key} className="flex items-stretch gap-2">
                <Link
                  href={stage.href}
                  className="flex-1 rounded-lg p-3 transition-colors hover:opacity-90"
                  style={{ background: "var(--desk-surface-raised)" }}
                >
                  <div className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
                    {stage.title}
                  </div>
                  <div className="md-kpi-value text-xl mt-1">{stage.metric}</div>
                  <p className="md-typescale-body-small mt-1" style={{ color: "var(--desk-text-secondary)" }}>
                    {stage.detail}
                  </p>
                </Link>
                {index < stages.length - 1 ? (
                  <span
                    className="hidden md:flex items-center md-typescale-label-medium shrink-0"
                    style={{ color: "var(--desk-text-secondary)" }}
                    aria-hidden
                  >
                    →
                  </span>
                ) : null}
              </div>
            ))}
          </div>

          {demandConfidence ? (
            <ForecastConfidenceCard
              confidence={demandConfidence}
              updatedAt={formatForecastUpdatedAt(demand?.generated_at)}
              stale={isForecastStale(demand?.generated_at)}
            />
          ) : null}
        </>
      )}
    </div>
  );
}
