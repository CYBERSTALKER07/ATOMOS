"use client";

import Link from "next/link";
import type { Route } from "next";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { createSupplierApi } from "@/lib/api";
import { decodeJwtPayload, getSupplierToken, readTokenFromCookie } from "@/lib/auth";
import { resolvePortalSessionRole } from "@/lib/connectors-auditor-api";
import { usePortalT } from "@/lib/i18n";
import {
  createP2bApi,
  extractP2bError,
  formatMilliCelsius,
  formatRateBps,
} from "@/lib/p2b-api";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PortalInput, PortalSection } from "@/components/portal";
import StatusBadge from "@/components/StatusBadge";
import Icon from "@/components/Icon";
import type { HaccpComplianceReport } from "@pegasusx/types";

const DATE_RE = /^\d{4}-\d{2}-\d{2}$/;

function triggerCsvDownload(csvText: string, filename: string) {
  if (typeof window === "undefined" || typeof URL.createObjectURL !== "function") {
    return;
  }
  const blob = new Blob([csvText], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement("a");
  anchor.href = url;
  anchor.download = filename;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(url);
}

export default function HaccpCompliancePage() {
  return (
    <Suspense fallback={null}>
      <HaccpComplianceInner />
    </Suspense>
  );
}

function HaccpComplianceInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const p2bApi = useMemo(() => createP2bApi(createSupplierApi()), []);

  const initialFrom = (searchParams.get("from") || "").trim();
  const initialTo = (searchParams.get("to") || "").trim();

  const [roleResolved, setRoleResolved] = useState(false);
  const [isReadOnlyVisitor, setIsReadOnlyVisitor] = useState(false);

  const [fromDate, setFromDate] = useState<string>(DATE_RE.test(initialFrom) ? initialFrom : "");
  const [toDate, setToDate] = useState<string>(DATE_RE.test(initialTo) ? initialTo : "");
  const [report, setReport] = useState<HaccpComplianceReport | null>(null);
  const [csvPreview, setCsvPreview] = useState<string | null>(null);

  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const syncRoleFromToken = useCallback(() => {
    const token = getSupplierToken() || readTokenFromCookie();
    if (!token) {
      setIsReadOnlyVisitor(false);
      setRoleResolved(true);
      return;
    }
    const roleInfo = resolvePortalSessionRole(decodeJwtPayload(token));
    setIsReadOnlyVisitor(roleInfo.isAuditor || roleInfo.isAgent);
    setRoleResolved(true);
  }, []);

  useEffect(() => {
    syncRoleFromToken();
  }, [syncRoleFromToken]);

  const loadReport = useCallback(
    async (silent = false) => {
      syncRoleFromToken();
      if (!silent) {
        setLoading(true);
      } else {
        setRefreshing(true);
      }
      setError(null);
      try {
        const res = await p2bApi.getHaccpReport({
          from: DATE_RE.test(fromDate.trim()) ? fromDate.trim() : undefined,
          to: DATE_RE.test(toDate.trim()) ? toDate.trim() : undefined,
        });
        setReport(res);
      } catch (err) {
        setError(
          extractP2bError(err, tRef.current("supplier_portal.haccp.load_failed")).message,
        );
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [fromDate, p2bApi, syncRoleFromToken, toDate],
  );

  useEffect(() => {
    void loadReport();
  }, [loadReport]);

  useSupplierSessionReconcile(() => {
    void loadReport(true);
  });

  const handleExportCsv = async () => {
    setExporting(true);
    setError(null);
    setNotice(null);
    try {
      const csv = await p2bApi.downloadHaccpCsv({
        from: DATE_RE.test(fromDate.trim()) ? fromDate.trim() : undefined,
        to: DATE_RE.test(toDate.trim()) ? toDate.trim() : undefined,
      });
      setCsvPreview(csv);
      const stamp = new Date().toISOString().slice(0, 10);
      triggerCsvDownload(csv, `haccp_compliance_${stamp}.csv`);
      setNotice(t("supplier_portal.haccp.export_success"));
    } catch (err) {
      setError(extractP2bError(err, t("supplier_portal.haccp.load_failed")).message);
    } finally {
      setExporting(false);
    }
  };

  const excursions = report?.excursions ?? [];
  const affectedLots = report?.affected_lots ?? [];
  const zoneMismatches = report?.cold_zone_mismatches ?? [];
  const conditionReports = report?.condition_reports ?? [];

  return (
    <PageChrome
      icon="warning"
      title={t("supplier_portal.haccp.title")}
      description={t("supplier_portal.haccp.subtitle")}
      loading={loading && !report && !error}
      skeletonVariant="dashboard"
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Link href={"/compliance" as Route} className="portal-btn portal-btn--outline text-xs">
            <Icon name="verified" size={16} />
            <span>{t("portal.nav.compliance_audit")}</span>
          </Link>
          <button
            type="button"
            disabled={exporting}
            onClick={() => void handleExportCsv()}
            className="portal-btn portal-btn--primary text-xs"
          >
            {exporting ? t("common.status.loading") : t("supplier_portal.haccp.export_csv_button")}
          </button>
          <button
            type="button"
            disabled={loading || refreshing}
            onClick={() => void loadReport(true)}
            className="portal-btn portal-btn--outline text-xs"
          >
            <Icon name="refresh" size={16} className={refreshing ? "animate-spin" : ""} />
            <span>{t("common.action.refresh")}</span>
          </button>
        </div>
      }
    >
      <div className="flex flex-col gap-6">
        {roleResolved && isReadOnlyVisitor ? (
          <div
            data-testid="haccp-read-only-banner"
            className="portal-alert portal-alert--info"
            role="status"
          >
            <p className="portal-alert-text text-xs">
              {t("supplier_portal.auditor.safe_mode_badge")} —{" "}
              {t("supplier_portal.haccp.read_only_notice")}
            </p>
          </div>
        ) : null}

        {error ? (
          <div className="portal-alert portal-alert--error" role="alert">
            <p className="portal-alert-text font-semibold">{error}</p>
          </div>
        ) : null}

        {notice ? (
          <div className="portal-alert portal-alert--info" role="status">
            <p className="portal-alert-text">{notice}</p>
          </div>
        ) : null}

        <div className="desk-card p-4">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 items-end">
            <div>
              <label htmlFor="haccp-from-date" className="portal-label block mb-1">
                {t("supplier_portal.haccp.from_date_label")}
              </label>
              <PortalInput
                id="haccp-from-date"
                type="date"
                value={fromDate}
                onChange={(e) => setFromDate(e.target.value)}
              />
            </div>
            <div>
              <label htmlFor="haccp-to-date" className="portal-label block mb-1">
                {t("supplier_portal.haccp.to_date_label")}
              </label>
              <PortalInput
                id="haccp-to-date"
                type="date"
                value={toDate}
                onChange={(e) => setToDate(e.target.value)}
              />
            </div>
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                onClick={() => void loadReport(true)}
                className="portal-btn portal-btn--outline text-xs"
              >
                {t("supplier_portal.haccp.apply_window_button")}
              </button>
              <button
                type="button"
                onClick={() => {
                  setFromDate("");
                  setToDate("");
                }}
                className="portal-btn portal-btn--ghost text-xs"
              >
                {t("supplier_portal.haccp.clear_window_button")}
              </button>
            </div>
          </div>
        </div>

        <KpiStatGrid columns={4}>
          <KpiStatCard
            label={t("supplier_portal.haccp.kpi_total_readings")}
            value={report ? report.total_readings : "—"}
            sub={report?.generated_at ? new Date(report.generated_at).toLocaleString() : undefined}
          />
          <KpiStatCard
            label={t("supplier_portal.haccp.kpi_excursions")}
            value={report ? report.excursion_count : "—"}
          />
          <KpiStatCard
            label={t("supplier_portal.haccp.kpi_compliance_rate")}
            value={report ? formatRateBps(report.compliance_rate_bps) : "—"}
          />
          <KpiStatCard
            label={t("supplier_portal.haccp.kpi_open_breaches_lots")}
            value={report ? `${report.open_breaches} / ${report.affected_lots_count}` : "—"}
          />
        </KpiStatGrid>

        <PortalSection icon="warning" title={t("supplier_portal.haccp.excursions_title")}>
          {excursions.length === 0 ? (
            <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
              {t("supplier_portal.haccp.no_excursions")}
            </p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse text-xs">
                <thead>
                  <tr
                    className="border-b uppercase"
                    style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                  >
                    <th className="py-2 px-2.5">{t("supplier_portal.haccp.col_reading")}</th>
                    <th className="py-2 px-2.5">{t("supplier_portal.haccp.col_manifest_sensor")}</th>
                    <th className="py-2 px-2.5 text-right">{t("supplier_portal.haccp.col_temp")}</th>
                    <th className="py-2 px-2.5 text-right">{t("supplier_portal.haccp.col_band")}</th>
                    <th className="py-2 px-2.5 text-right">{t("supplier_portal.haccp.col_delta")}</th>
                    <th className="py-2 px-2.5">{t("supplier_portal.anomalies.col_severity")}</th>
                    <th className="py-2 px-2.5">{t("supplier_portal.anomalies.col_status")}</th>
                    <th className="py-2 px-2.5">{t("supplier_portal.haccp.col_recorded_at")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                  {excursions.map((ex) => (
                    <tr key={ex.reading_id}>
                      <td className="py-2 px-2.5 font-mono font-semibold">{ex.reading_id}</td>
                      <td className="py-2 px-2.5 font-mono">
                        {ex.manifest_id}
                        {ex.sensor_id ? ` · ${ex.sensor_id}` : ""}
                        {ex.warehouse_id ? ` · ${ex.warehouse_id}` : ""}
                      </td>
                      <td className="py-2 px-2.5 font-mono text-right font-semibold">
                        {formatMilliCelsius(ex.temp_milli_c)}
                      </td>
                      <td className="py-2 px-2.5 font-mono text-right">
                        {formatMilliCelsius(ex.min_milli_c)} .. {formatMilliCelsius(ex.max_milli_c)}
                      </td>
                      <td className="py-2 px-2.5 font-mono text-right">
                        {formatMilliCelsius(ex.breach_delta_milli_c)}
                      </td>
                      <td className="py-2 px-2.5">
                        <StatusBadge state={ex.severity} />
                      </td>
                      <td className="py-2 px-2.5">
                        <StatusBadge state={ex.status} />
                      </td>
                      <td className="py-2 px-2.5 whitespace-nowrap">
                        {new Date(ex.recorded_at).toLocaleString()}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </PortalSection>

        <PortalSection icon="inventory" title={t("supplier_portal.haccp.affected_lots_title")}>
          {affectedLots.length === 0 ? (
            <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
              {t("supplier_portal.haccp.no_affected_lots")}
            </p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse text-xs">
                <thead>
                  <tr
                    className="border-b uppercase"
                    style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                  >
                    <th className="py-2 px-2.5">{t("supplier_portal.haccp.col_lot")}</th>
                    <th className="py-2 px-2.5">{t("supplier_portal.haccp.col_product")}</th>
                    <th className="py-2 px-2.5">{t("supplier_portal.haccp.col_zones")}</th>
                    <th className="py-2 px-2.5 text-right">{t("supplier_portal.haccp.col_qty")}</th>
                    <th className="py-2 px-2.5">{t("supplier_portal.anomalies.col_status")}</th>
                    <th className="py-2 px-2.5">{t("supplier_portal.haccp.col_reason")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                  {affectedLots.map((lot) => (
                    <tr key={`${lot.lot_id}:${lot.location_id || ""}`}>
                      <td className="py-2 px-2.5 font-mono">
                        <div className="font-semibold">{lot.lot_code || lot.lot_id}</div>
                        <div style={{ color: "var(--desk-text-tertiary)" }}>
                          {lot.warehouse_id || "—"} {lot.location_id ? `· ${lot.location_id}` : ""}
                        </div>
                      </td>
                      <td className="py-2 px-2.5">
                        <div className="font-medium">{lot.product_name || lot.product_id}</div>
                        <div className="font-mono text-[11px]" style={{ color: "var(--desk-text-tertiary)" }}>
                          {lot.product_id}
                        </div>
                      </td>
                      <td className="py-2 px-2.5 font-mono">
                        {lot.required_zone || "—"} → {lot.location_zone || "—"}
                      </td>
                      <td className="py-2 px-2.5 font-mono text-right">{lot.quantity_on_hand}</td>
                      <td className="py-2 px-2.5">
                        <StatusBadge state={lot.status} />
                      </td>
                      <td className="py-2 px-2.5">{lot.reason || "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </PortalSection>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <PortalSection icon="warehouse" title={t("supplier_portal.haccp.zone_mismatches_title")}>
            {zoneMismatches.length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.haccp.no_zone_mismatches")}
              </p>
            ) : (
              <div className="space-y-2 text-xs">
                {zoneMismatches.map((zm, idx) => (
                  <div
                    key={`${zm.lot_id}:${zm.location_id}:${idx}`}
                    className="rounded-lg border p-3"
                    style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}
                  >
                    <div className="font-mono font-semibold">
                      {zm.lot_id} · {zm.location_id} ({zm.required_zone} vs {zm.location_zone})
                    </div>
                    <div style={{ color: "var(--desk-text-secondary)" }}>
                      {zm.product_name || zm.product_id} — {zm.warning}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </PortalSection>

          <PortalSection icon="error" title={t("supplier_portal.haccp.condition_reports_title")}>
            {conditionReports.length === 0 ? (
              <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.haccp.no_condition_reports")}
              </p>
            ) : (
              <div className="space-y-2 text-xs">
                {conditionReports.map((cr) => (
                  <div
                    key={cr.report_id}
                    className="rounded-lg border p-3 flex flex-col gap-1"
                    style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}
                  >
                    <div className="flex items-center justify-between gap-2">
                      <span className="font-mono font-semibold">
                        {cr.report_id} · {cr.condition_type}
                      </span>
                      <div className="flex items-center gap-1.5">
                        <StatusBadge state={cr.severity} />
                        <StatusBadge state={cr.resolution_status} />
                      </div>
                    </div>
                    <div>
                      Order <span className="font-mono">{cr.order_id}</span>
                      {cr.sku ? ` · ${cr.sku}` : ""}
                      {cr.description ? ` — ${cr.description}` : ""}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </PortalSection>
        </div>

        {csvPreview ? (
          <PortalSection icon="manifests" title={t("supplier_portal.haccp.csv_preview_title")}>
            <pre
              className="rounded-lg border p-3 text-xs font-mono overflow-x-auto"
              style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-raised)" }}
            >
              {csvPreview}
            </pre>
          </PortalSection>
        ) : null}
      </div>
    </PageChrome>
  );
}
