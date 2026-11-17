"use client";

import Link from "next/link";
import type { Route } from "next";
import { useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import {
  createIntelRoiApi,
  extractControlError,
  formatFinanceMinor,
  formatRateBps,
  formatRoiMultipleBps,
  formatSignedBps,
  parseMajorToSafeMinor,
  type RoiBaseline,
  type RoiEvent,
  type RoiSummaryResponse,
} from "@/lib/intel-roi-api";
import { minorToWholeTrunc } from "@/lib/money";
import { usePortalT } from "@/lib/i18n";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PageChrome } from "@/components/PageChrome";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import StatusBadge from "@/components/StatusBadge";

const WINDOW_OPTIONS = ["30", "90", "180", "365"] as const;
const EVENT_CATEGORIES = [
  "ALL",
  "BAD_DEBT_AVOIDED",
  "STOCKOUT_AVOIDED",
  "EXPIRY_SAVED",
  "LOGISTICS_SAVED",
  "LABOUR_SAVED",
  "WORKING_CAPITAL_SAVED",
] as const;

export default function RoiDashboardPage() {
  return (
    <Suspense fallback={null}>
      <RoiDashboardInner />
    </Suspense>
  );
}

function RoiDashboardInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const intelApi = useMemo(() => createIntelRoiApi(createSupplierApi()), []);

  const initialDays = (searchParams.get("days") || "90").trim();
  const initialCategory = (searchParams.get("category") || "ALL").trim().toUpperCase();

  const [days, setDays] = useState<string>(
    WINDOW_OPTIONS.includes(initialDays as (typeof WINDOW_OPTIONS)[number]) ? initialDays : "90",
  );
  const [categoryFilter, setCategoryFilter] = useState<string>(
    EVENT_CATEGORIES.includes(initialCategory as (typeof EVENT_CATEGORIES)[number])
      ? initialCategory
      : "ALL",
  );

  const [summary, setSummary] = useState<RoiSummaryResponse | null>(null);
  const [baseline, setBaseline] = useState<RoiBaseline | null>(null);
  const [events, setEvents] = useState<RoiEvent[]>([]);
  const [qbrCsvPreview, setQbrCsvPreview] = useState<string | null>(null);

  // Baseline capture form (major units)
  const [windowDaysInput, setWindowDaysInput] = useState<string>("90");
  const [badDebtMajor, setBadDebtMajor] = useState<string>("0");
  const [expiredMajor, setExpiredMajor] = useState<string>("0");
  const [stockoutMajor, setStockoutMajor] = useState<string>("0");
  const [logisticsMajor, setLogisticsMajor] = useState<string>("0");
  const [labourHoursInput, setLabourHoursInput] = useState<string>("0");
  const [hourlyWageMajor, setHourlyWageMajor] = useState<string>("0");
  const [surveyNotes, setSurveyNotes] = useState<string>("");

  const [loading, setLoading] = useState<boolean>(true);
  const [busyAction, setBusyAction] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const hydrateBaselineForm = (b: RoiBaseline | null) => {
    if (!b) return;
    setWindowDaysInput(String(b.window_days || 90));
    setBadDebtMajor(String(minorToWholeTrunc(b.baseline_bad_debt_minor || 0)));
    setExpiredMajor(String(minorToWholeTrunc(b.baseline_expired_minor || 0)));
    setStockoutMajor(String(minorToWholeTrunc(b.baseline_stockout_minor || 0)));
    setLogisticsMajor(String(minorToWholeTrunc(b.baseline_logistics_minor || 0)));
    setLabourHoursInput(String(b.baseline_labour_hours || 0));
    setHourlyWageMajor(String(minorToWholeTrunc(b.survey_hourly_wage_minor || 0)));
    setSurveyNotes(b.survey_notes || "");
  };

  const loadAll = useCallback(async () => {
    setLoading(true);
    setError(null);
    const parsedDays = Number.parseInt(days, 10) || 90;
    const [sumRes, baseRes, evRes] = await Promise.allSettled([
      intelApi.getRoiSummary({ days: parsedDays }),
      intelApi.getRoiBaseline(),
      intelApi.listRoiEvents({
        category: categoryFilter !== "ALL" ? categoryFilter : undefined,
        limit: 100,
      }),
    ]);

    if (sumRes.status === "fulfilled") {
      setSummary(sumRes.value);
      if (sumRes.value.baseline) {
        setBaseline(sumRes.value.baseline);
        hydrateBaselineForm(sumRes.value.baseline);
      }
    } else {
      const parsed = extractControlError(sumRes.reason, tRef.current("supplier_portal.roi.load_failed"));
      setError(parsed.message);
    }

    if (baseRes.status === "fulfilled" && baseRes.value.baseline) {
      setBaseline(baseRes.value.baseline);
      hydrateBaselineForm(baseRes.value.baseline);
    }

    if (evRes.status === "fulfilled") {
      setEvents(evRes.value.events ?? []);
    }

    setLoading(false);
  }, [categoryFilter, days, intelApi]);

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  useSupplierSessionReconcile(() => {
    void loadAll();
  });

  const handleSaveBaseline = async (e: React.FormEvent) => {
    e.preventDefault();
    const badDebtMinor = parseMajorToSafeMinor(badDebtMajor || "0", false);
    const expiredMinor = parseMajorToSafeMinor(expiredMajor || "0", false);
    const stockoutMinor = parseMajorToSafeMinor(stockoutMajor || "0", false);
    const logisticsMinor = parseMajorToSafeMinor(logisticsMajor || "0", false);
    const hourlyWageMinor = parseMajorToSafeMinor(hourlyWageMajor || "0", false);
    const winDays = Number.parseInt(windowDaysInput || "90", 10);
    const labourHours = Number.parseInt(labourHoursInput || "0", 10);

    if (
      badDebtMinor === null ||
      expiredMinor === null ||
      stockoutMinor === null ||
      logisticsMinor === null ||
      hourlyWageMinor === null ||
      !Number.isFinite(winDays) ||
      winDays <= 0 ||
      !Number.isFinite(labourHours) ||
      labourHours < 0
    ) {
      setError("Baseline fields must contain valid non-negative numeric values.");
      return;
    }

    setBusyAction("baseline");
    setError(null);
    setNotice(null);
    try {
      const res = await intelApi.captureRoiBaseline({
        window_days: winDays,
        baseline_bad_debt_minor: badDebtMinor,
        baseline_expired_minor: expiredMinor,
        baseline_stockout_minor: stockoutMinor,
        baseline_logistics_minor: logisticsMinor,
        baseline_labour_hours: labourHours,
        survey_hourly_wage_minor: hourlyWageMinor,
        survey_notes: surveyNotes.trim() || undefined,
        currency: summary?.currency || baseline?.currency || "UZS",
      });
      setBaseline(res.baseline);
      setNotice(
        t("supplier_portal.roi.baseline_saved", {
          baselineId: res.baseline.baseline_id,
        }),
      );
      await loadAll();
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.roi.load_failed"));
      setError(parsed.message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleDownloadQbrCsv = async () => {
    const parsedDays = Number.parseInt(days, 10) || 90;
    setBusyAction("csv");
    setError(null);
    try {
      const csvText = await intelApi.downloadRoiReportCsv({ days: parsedDays });
      setQbrCsvPreview(csvText);
      if (typeof window !== "undefined" && typeof Blob !== "undefined" && typeof URL.createObjectURL === "function") {
        const blob = new Blob([csvText], { type: "text/csv;charset=utf-8;" });
        const url = URL.createObjectURL(blob);
        const link = document.createElement("a");
        link.href = url;
        link.download = `pegasusx-roi-qbr-${parsedDays}d.csv`;
        link.click();
        URL.revokeObjectURL(url);
      }
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.roi.load_failed"));
      setError(parsed.message);
    } finally {
      setBusyAction(null);
    }
  };

  const currency = summary?.currency || baseline?.currency || "UZS";
  const holdout = summary?.holdout_stats ?? null;

  return (
    <PageChrome
      icon="analytics"
      title={t("supplier_portal.roi.title")}
      description={t("supplier_portal.roi.subtitle")}
      loading={loading && !summary && !error}
      skeletonVariant="dashboard"
      error={error}
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <PortalSelect
            aria-label="ROI window days"
            value={days}
            onChange={(e) => setDays(e.target.value)}
          >
            {WINDOW_OPTIONS.map((opt) => (
              <option key={opt} value={opt}>
                {opt}d
              </option>
            ))}
          </PortalSelect>
          <button
            type="button"
            disabled={busyAction === "csv"}
            onClick={() => void handleDownloadQbrCsv()}
            className="portal-btn portal-btn--primary text-xs"
          >
            {t("supplier_portal.roi.download_qbr_csv")}
          </button>
          <Link href={"/proposals" as Route} className="portal-btn portal-btn--outline text-xs">
            {t("supplier_portal.roi.open_proposals")}
          </Link>
        </div>
      }
    >
      <div className="flex flex-col gap-6">
        {notice ? (
          <div
            role="status"
            className="rounded-xl border px-4 py-3 text-sm"
            style={{
              background: "var(--desk-success-soft)",
              borderColor: "color-mix(in srgb, var(--desk-success) 35%, transparent)",
              color: "var(--desk-success)",
            }}
          >
            {notice}
          </div>
        ) : null}

        {summary ? (
          <div
            className="desk-card p-6 border"
            style={{
              background: "color-mix(in srgb, var(--desk-accent) 10%, var(--desk-surface))",
              borderColor: "color-mix(in srgb, var(--desk-accent) 30%, var(--desk-border))",
            }}
          >
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div>
                <p className="text-xs font-semibold uppercase tracking-wider" style={{ color: "var(--desk-accent-strong)" }}>
                  {t("supplier_portal.roi.banner_title", { days: summary.window_days })}
                </p>
                <div className="text-3xl font-light mt-1 tabular-nums">
                  {formatFinanceMinor(summary.total_savings_minor, currency)}
                </div>
                <p className="text-xs mt-1" style={{ color: "var(--desk-text-secondary)" }}>
                  Realized: {formatFinanceMinor(summary.realized_savings_minor, currency)} · Estimated:{" "}
                  {formatFinanceMinor(summary.estimated_savings_minor, currency)}
                </p>
              </div>
              <div className="text-right">
                <div className="text-xs uppercase tracking-wider" style={{ color: "var(--desk-text-secondary)" }}>
                  {t("supplier_portal.roi.roi_multiple")}
                </div>
                <div className="text-3xl font-semibold tabular-nums" style={{ color: "var(--desk-accent-strong)" }}>
                  {formatRoiMultipleBps(summary.roi_multiple_bps)}
                </div>
              </div>
            </div>
          </div>
        ) : null}

        {summary ? (
          <KpiStatGrid columns={4}>
            <KpiStatCard
              label={t("supplier_portal.roi.total_savings")}
              value={formatFinanceMinor(summary.total_savings_minor, currency)}
              sub={`Holdout ${formatFinanceMinor(summary.holdout_savings_minor, currency)}`}
            />
            <KpiStatCard
              label={t("supplier_portal.roi.platform_fees")}
              value={formatFinanceMinor(summary.platform_fees_minor, currency)}
            />
            <KpiStatCard
              label={t("supplier_portal.roi.net_savings")}
              value={formatFinanceMinor(summary.net_savings_minor, currency)}
            />
            <KpiStatCard
              label={t("supplier_portal.roi.roi_multiple")}
              value={formatRoiMultipleBps(summary.roi_multiple_bps)}
              sub={`${summary.window_days}d QBR window`}
            />
          </KpiStatGrid>
        ) : null}

        {holdout ? (
          <PortalSection icon="reconcile" title={t("supplier_portal.roi.holdout_title")}>
            <KpiStatGrid columns={4}>
              <KpiStatCard
                label="Treatment group"
                value={formatFinanceMinor(holdout.treatment_savings_minor, currency)}
                sub={`${holdout.treatment_event_count} events`}
              />
              <KpiStatCard
                label="Holdout control"
                value={formatFinanceMinor(holdout.holdout_savings_minor, currency)}
                sub={`${holdout.holdout_event_count} events`}
              />
              <KpiStatCard
                label="Incremental lift"
                value={formatFinanceMinor(holdout.incremental_lift_minor, currency)}
              />
              <KpiStatCard
                label="Lift vs holdout"
                value={formatSignedBps(holdout.lift_bps)}
              />
            </KpiStatGrid>
          </PortalSection>
        ) : null}

        <PortalSection icon="ledger" title={t("supplier_portal.roi.methodology_title")}>
          <div className="overflow-x-auto">
            <table className="desk-table w-full text-sm">
              <thead>
                <tr style={{ color: "var(--desk-text-secondary)" }}>
                  <th className="p-2.5 text-left">Category</th>
                  <th className="p-2.5 text-right">Events</th>
                  <th className="p-2.5 text-right">Total Savings</th>
                  <th className="p-2.5 text-right">Realized / Est.</th>
                  <th className="p-2.5 text-right">Holdout</th>
                  <th className="p-2.5 text-right">Avg Confidence</th>
                  <th className="p-2.5 text-left">Methodology</th>
                </tr>
              </thead>
              <tbody>
                {(summary?.categories ?? []).length === 0 ? (
                  <tr>
                    <td colSpan={7} className="p-6 text-center" style={{ color: "var(--desk-text-secondary)" }}>
                      No category savings recorded in this window yet.
                    </td>
                  </tr>
                ) : (
                  summary!.categories.map((cat) => (
                    <tr key={cat.category} style={{ borderTop: "1px solid var(--desk-border)" }}>
                      <td className="p-2.5">
                        <span className="font-mono text-xs font-semibold">{cat.category}</span>
                        {cat.estimated_minor > 0 ? (
                          <span className="ml-2">
                            <StatusBadge state="WARN" label="Estimate" />
                          </span>
                        ) : null}
                      </td>
                      <td className="p-2.5 text-right font-mono">{cat.event_count}</td>
                      <td className="p-2.5 text-right font-mono font-semibold">
                        {formatFinanceMinor(cat.total_savings_minor, currency)}
                      </td>
                      <td className="p-2.5 text-right font-mono text-xs">
                        {formatFinanceMinor(cat.realized_minor, currency)} /{" "}
                        {formatFinanceMinor(cat.estimated_minor, currency)}
                      </td>
                      <td className="p-2.5 text-right font-mono text-xs">
                        {formatFinanceMinor(cat.holdout_savings_minor, currency)}
                      </td>
                      <td className="p-2.5 text-right font-mono">{formatRateBps(cat.avg_confidence_bps)}</td>
                      <td className="p-2.5 text-xs" style={{ color: "var(--desk-text-secondary)" }}>
                        {cat.methodology}
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </PortalSection>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <PortalSection
            icon="calendar"
            title={t("supplier_portal.roi.baseline_title")}
            description={
              baseline
                ? `Active baseline ${baseline.baseline_id} · AR ${formatFinanceMinor(baseline.baseline_ar_minor, currency)}`
                : "Capture 90-day baseline metrics & hourly wage survey."
            }
          >
            <form onSubmit={handleSaveBaseline} className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <label className="flex flex-col gap-1 text-xs">
                <span style={{ color: "var(--desk-text-secondary)" }}>Window days</span>
                <PortalInput
                  aria-label="Baseline window days"
                  type="number"
                  min={1}
                  value={windowDaysInput}
                  onChange={(e) => setWindowDaysInput(e.target.value)}
                />
              </label>
              <label className="flex flex-col gap-1 text-xs">
                <span style={{ color: "var(--desk-text-secondary)" }}>Labour hours (90d)</span>
                <PortalInput
                  aria-label="Baseline labour hours"
                  type="number"
                  min={0}
                  value={labourHoursInput}
                  onChange={(e) => setLabourHoursInput(e.target.value)}
                />
              </label>
              <label className="flex flex-col gap-1 text-xs">
                <span style={{ color: "var(--desk-text-secondary)" }}>Bad debt baseline ({currency})</span>
                <PortalInput
                  aria-label="Bad debt baseline"
                  value={badDebtMajor}
                  onChange={(e) => setBadDebtMajor(e.target.value)}
                />
              </label>
              <label className="flex flex-col gap-1 text-xs">
                <span style={{ color: "var(--desk-text-secondary)" }}>Expired stock baseline ({currency})</span>
                <PortalInput
                  aria-label="Expired stock baseline"
                  value={expiredMajor}
                  onChange={(e) => setExpiredMajor(e.target.value)}
                />
              </label>
              <label className="flex flex-col gap-1 text-xs">
                <span style={{ color: "var(--desk-text-secondary)" }}>Stockout baseline ({currency})</span>
                <PortalInput
                  aria-label="Stockout baseline"
                  value={stockoutMajor}
                  onChange={(e) => setStockoutMajor(e.target.value)}
                />
              </label>
              <label className="flex flex-col gap-1 text-xs">
                <span style={{ color: "var(--desk-text-secondary)" }}>Logistics baseline ({currency})</span>
                <PortalInput
                  aria-label="Logistics baseline"
                  value={logisticsMajor}
                  onChange={(e) => setLogisticsMajor(e.target.value)}
                />
              </label>
              <label className="flex flex-col gap-1 text-xs">
                <span style={{ color: "var(--desk-text-secondary)" }}>Hourly wage ({currency})</span>
                <PortalInput
                  aria-label="Survey hourly wage"
                  value={hourlyWageMajor}
                  onChange={(e) => setHourlyWageMajor(e.target.value)}
                />
              </label>
              <label className="flex flex-col gap-1 text-xs">
                <span style={{ color: "var(--desk-text-secondary)" }}>Survey notes</span>
                <PortalInput
                  aria-label="Baseline survey notes"
                  value={surveyNotes}
                  onChange={(e) => setSurveyNotes(e.target.value)}
                  placeholder="Pre-deployment onboarding survey"
                />
              </label>
              <div className="sm:col-span-2 pt-1">
                <button
                  type="submit"
                  disabled={busyAction === "baseline"}
                  className="portal-btn portal-btn--primary text-xs"
                >
                  {t("supplier_portal.roi.capture_baseline")}
                </button>
              </div>
            </form>
          </PortalSection>

          <PortalSection
            icon="manifests"
            title={t("supplier_portal.roi.events_title")}
            actions={
              <PortalSelect
                aria-label="ROI event category filter"
                value={categoryFilter}
                onChange={(e) => setCategoryFilter(e.target.value)}
              >
                {EVENT_CATEGORIES.map((c) => (
                  <option key={c} value={c}>
                    {c}
                  </option>
                ))}
              </PortalSelect>
            }
          >
            {events.length === 0 ? (
              <p className="text-sm" style={{ color: "var(--desk-text-secondary)" }}>
                No ROI events recorded for this category yet.
              </p>
            ) : (
              <div className="space-y-2 max-h-96 overflow-y-auto pr-1">
                {events.map((ev) => (
                  <div
                    key={ev.event_id}
                    className="rounded-lg border p-3 text-xs flex flex-col gap-1"
                    style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-raised)" }}
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div className="flex items-center gap-2">
                        <span className="font-mono font-semibold">{ev.category}</span>
                        {ev.is_estimate ? <StatusBadge state="WARN" label="ESTIMATE" /> : null}
                        {ev.holdout_control ? <StatusBadge state="NEUTRAL" label="HOLDOUT" /> : null}
                      </div>
                      <span className="font-mono font-semibold">
                        {formatFinanceMinor(ev.savings_minor, ev.currency || currency)}
                      </span>
                    </div>
                    <p style={{ color: "var(--desk-text-secondary)" }}>{ev.methodology}</p>
                    <div className="flex flex-wrap gap-3 font-mono text-[11px]" style={{ color: "var(--desk-text-tertiary)" }}>
                      <span>Conf {formatRateBps(ev.confidence_bps)}</span>
                      {ev.proposal_id ? <span>Proposal {ev.proposal_id}</span> : null}
                      {ev.evidence_ref_id ? (
                        <span>
                          {ev.evidence_ref_type || "Ref"}:{ev.evidence_ref_id}
                        </span>
                      ) : null}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </PortalSection>
        </div>

        {qbrCsvPreview ? (
          <PortalSection icon="ledger" title="QBR CSV Export Preview">
            <pre
              className="rounded-lg border p-3 text-xs font-mono overflow-x-auto max-h-60"
              style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-raised)" }}
            >
              {qbrCsvPreview}
            </pre>
          </PortalSection>
        ) : null}
      </div>
    </PageChrome>
  );
}
