"use client";

import React, { useState } from "react";
import { getAdminToken } from "@/lib/auth";
import { usePolling } from "@/lib/usePolling";
import { useLocale } from "@/hooks/useLocale";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

interface EmpathyAdoption {
  total_retailers: number;
  global_enabled: number;
  supplier_overrides: number;
  product_overrides: number;
  variant_overrides: number;
  predictions_dormant: number;
  predictions_waiting: number;
  predictions_fired: number;
  predictions_rejected: number;
}

const EMPTY: EmpathyAdoption = {
  total_retailers: 0,
  global_enabled: 0,
  supplier_overrides: 0,
  product_overrides: 0,
  variant_overrides: 0,
  predictions_dormant: 0,
  predictions_waiting: 0,
  predictions_fired: 0,
  predictions_rejected: 0,
};

export default function EmpathyDashboard() {
  const { locale, t } = useLocale();
  const [data, setData] = useState<EmpathyAdoption>(EMPTY);
  const [isLive, setIsLive] = useState(false);
  const [error, setError] = useState<string | null>(null);

  usePolling(async (signal) => {
    try {
      const token = await getAdminToken();
      const res = await fetch(`${API}/v1/admin/empathy/adoption`, {
        headers: { Authorization: `Bearer ${token}` }, signal,
      });
      if (!res.ok) throw new Error(t("supplier_portal.admin.empathy.error.telemetry_disconnected"));
      const json = await res.json();
      setData(json);
      setIsLive(true);
      setError(null);
    } catch (err) {
      if ((err as Error).name === 'AbortError') return;
      console.error("[EMPATHY ERROR]", err);
      setIsLive(false);
      setError(err instanceof Error ? err.message : t("common.error.unknown"));
    }
  }, 10_000, [t]);

  const adoptionRate =
    data.total_retailers > 0
      ? Math.round((data.global_enabled / data.total_retailers) * 100)
      : 0;

  const totalPredictions =
    data.predictions_dormant +
    data.predictions_waiting +
    data.predictions_fired +
    data.predictions_rejected;

  const pipelineItems = [
    { label: t("supplier_portal.admin.empathy.pipeline.dormant"), value: data.predictions_dormant, color: "var(--border)" },
    { label: t("supplier_portal.admin.empathy.pipeline.waiting"), value: data.predictions_waiting, color: "var(--muted)" },
    { label: t("supplier_portal.admin.empathy.pipeline.fired"), value: data.predictions_fired, color: "var(--success)" },
    { label: t("supplier_portal.admin.empathy.pipeline.rejected"), value: data.predictions_rejected, color: "var(--danger)" },
  ];

  return (
    <div
      className="min-h-full p-6 md:p-10"
      style={{
        background: "var(--background)",
        color: "var(--foreground)",
      }}
    >
      {/* ── Header ───────────────────────────────────────────────── */}
      <header
        className="mb-10 pb-6 flex justify-between items-end"
        style={{ borderBottom: "1px solid var(--border)" }}
      >
        <div>
          <h1 className="md-typescale-headline-medium">{t("supplier_portal.admin.empathy.title")}</h1>
          <p
            className="md-typescale-body-medium mt-2"
            style={{ color: "var(--muted)" }}
          >
            {t("supplier_portal.admin.empathy.subtitle")}
          </p>
        </div>
        <div className="text-right">
          {isLive ? (
            <div className="md-chip" style={{ cursor: "default" }}>
              <div
                className="w-2 h-2 rounded-full animate-pulse"
                style={{ background: "var(--success)" }}
              />
              <span className="md-typescale-label-small">{t("supplier_portal.admin.empathy.state.online")}</span>
            </div>
          ) : (
            <div
              className="md-chip"
              style={{ cursor: "default", borderColor: "var(--danger)" }}
            >
              <div
                className="w-2 h-2 rounded-full"
                style={{ background: "var(--danger)" }}
              />
              <span
                className="md-typescale-label-small"
                style={{ color: "var(--danger)" }}
              >
                {error || t("supplier_portal.admin.empathy.state.offline")}
              </span>
            </div>
          )}
        </div>
      </header>

      {/* ── Adoption KPIs ────────────────────────────────────────── */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-10">
        <KpiCard
          label={t("supplier_portal.admin.empathy.kpi.adoption_rate")}
          value={`${adoptionRate}%`}
          sub={t("supplier_portal.admin.empathy.kpi.adoption_sub", {
            enabled: data.global_enabled,
            total: data.total_retailers,
          })}
          accent="var(--accent)"
          delay={0}
        />
        <KpiCard
          label={t("supplier_portal.admin.empathy.kpi.supplier_overrides")}
          value={data.supplier_overrides.toLocaleString(locale)}
          sub={t("supplier_portal.admin.empathy.kpi.supplier_overrides_sub")}
          accent="var(--muted)"
          delay={50}
        />
        <KpiCard
          label={t("supplier_portal.admin.empathy.kpi.product_overrides")}
          value={data.product_overrides.toLocaleString(locale)}
          sub={t("supplier_portal.admin.empathy.kpi.product_overrides_sub")}
          accent="var(--muted)"
          delay={100}
        />
        <KpiCard
          label={t("supplier_portal.admin.empathy.kpi.variant_overrides")}
          value={data.variant_overrides.toLocaleString(locale)}
          sub={t("supplier_portal.admin.empathy.kpi.variant_overrides_sub")}
          accent="var(--border)"
          delay={150}
        />
      </div>

      {/* ── Prediction Pipeline ──────────────────────────────────── */}
      <div className="mb-10">
        <h2 className="md-typescale-title-medium mb-4">{t("supplier_portal.admin.empathy.section.pipeline")}</h2>
        <div className="md-card md-card-elevated p-6 md-animate-in">
          {/* Status bar */}
          <div
            className="flex w-full h-6 rounded-full overflow-hidden mb-6"
            style={{ background: "var(--surface)" }}
          >
            {totalPredictions > 0 &&
              pipelineItems.map((item) => {
                const pct = (item.value / totalPredictions) * 100;
                if (pct === 0) return null;
                return (
                  <div
                    key={item.label}
                    style={{ width: `${pct}%`, background: item.color }}
                    title={`${item.label}: ${item.value}`}
                  />
                );
              })}
          </div>

          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            {pipelineItems.map((item) => (
              <div key={item.label} className="flex items-center gap-3">
                <div
                  className="w-3 h-3 rounded-full shrink-0"
                  style={{ background: item.color }}
                />
                <div>
                  <p
                    className="md-typescale-label-small"
                    style={{ color: "var(--muted)" }}
                  >
                    {item.label}
                  </p>
                   <p className="md-typescale-title-medium" style={{ fontVariantNumeric: 'tabular-nums' }}>{item.value.toLocaleString(locale)}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* ── Resolution Hierarchy ─────────────────────────────────── */}
      <div>
        <h2 className="md-typescale-title-medium mb-4">{t("supplier_portal.admin.empathy.section.hierarchy")}</h2>
        <div className="md-card md-card-filled p-6 md-animate-in" style={{ animationDelay: "100ms" }}>
          <p
            className="md-typescale-body-medium mb-4"
            style={{ color: "var(--muted)" }}
          >
            {t("supplier_portal.admin.empathy.hierarchy.description")}
          </p>
          <div className="flex flex-col gap-3">
            {[
              {
                level: t("supplier_portal.admin.empathy.hierarchy.variant.level"),
                desc: t("supplier_portal.admin.empathy.hierarchy.variant.description"),
                count: data.variant_overrides,
              },
              {
                level: t("supplier_portal.admin.empathy.hierarchy.product.level"),
                desc: t("supplier_portal.admin.empathy.hierarchy.product.description"),
                count: data.product_overrides,
              },
              {
                level: t("supplier_portal.admin.empathy.hierarchy.supplier.level"),
                desc: t("supplier_portal.admin.empathy.hierarchy.supplier.description"),
                count: data.supplier_overrides,
              },
              {
                level: t("supplier_portal.admin.empathy.hierarchy.global.level"),
                desc: t("supplier_portal.admin.empathy.hierarchy.global.description"),
                count: data.global_enabled,
              },
            ].map((row, i) => (
              <div
                key={row.level}
                className="flex items-center justify-between px-4 py-3 rounded-xl"
                style={{
                  background:
                    i === 0
                      ? "var(--accent-soft)"
                      : "var(--surface)",
                  color:
                    i === 0
                      ? "var(--accent-soft-foreground)"
                      : "var(--foreground)",
                }}
              >
                <div>
                  <p className="md-typescale-label-large">{row.level}</p>
                  <p
                    className="md-typescale-body-small"
                    style={{
                      color:
                        i === 0
                          ? "var(--accent-soft-foreground)"
                          : "var(--muted)",
                      opacity: 0.8,
                    }}
                  >
                    {row.desc}
                  </p>
                </div>
                <p className="md-typescale-title-medium">
                  {t("supplier_portal.admin.empathy.hierarchy.active_count", { count: row.count.toLocaleString(locale) })}
                </p>
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

/* ── Reusable KPI Card ─────────────────────────────────────────────── */
function KpiCard({
  label,
  value,
  sub,
  accent,
  delay,
}: {
  label: string;
  value: string;
  sub: string;
  accent: string;
  delay: number;
}) {
  return (
    <div
      className="md-card md-card-elevated p-6 relative overflow-hidden md-animate-in"
      style={{ animationDelay: `${delay}ms` }}
    >
      <div
        className="absolute top-0 right-0 w-1.5 h-full"
        style={{ background: accent }}
      />
      <p
        className="md-typescale-label-small mb-4"
        style={{ color: "var(--muted)" }}
      >
        {label}
      </p>
      <p className="md-typescale-display-small tracking-tight" style={{ fontVariantNumeric: 'tabular-nums' }}>{value}</p>
      <p
        className="md-typescale-label-small mt-3"
        style={{ color: accent }}
      >
        {sub}
      </p>
    </div>
  );
}
