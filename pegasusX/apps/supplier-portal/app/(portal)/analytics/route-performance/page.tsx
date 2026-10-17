"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { PageChrome } from "@/components/PageChrome";

type RoutePerfRow = {
  route_id: string;
  driver_id?: string;
  planned_stops?: number;
  actual_stops?: number;
  planned_duration_sec?: number;
  actual_duration_sec?: number;
  replan_count?: number;
  computed_at: string;
};

export default function RoutePerformancePage() {
  const t = usePortalT();
  const [rows, setRows] = useState<RoutePerfRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await supplierFetch("/v1/supplier/route-performance?limit=100");
      if (!res.ok) throw new Error(`route_perf_${res.status}`);
      const body = (await res.json()) as { routes?: RoutePerfRow[] };
      setRows(body.routes ?? []);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_failed"));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <PageChrome title={t("portal.nav.route_performance")} description={t("supplier_portal.residual.text.completed_route_efficiency_and_replan_metrics")} loading={loading} error={error}>
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b text-left text-xs uppercase text-[var(--muted)]">
            <th className="py-2 pr-3">{t("supplier_portal.analytics.route_performance.text.route")}</th>
            <th className="py-2 pr-3">{t("supplier_portal.analytics.route_performance.text.driver")}</th>
            <th className="py-2 pr-3 text-right">{t("supplier_portal.analytics.route_performance.text.planned_stops")}</th>
            <th className="py-2 pr-3 text-right">{t("supplier_portal.analytics.route_performance.text.actual_stops")}</th>
            <th className="py-2 pr-3 text-right">{t("supplier_portal.analytics.route_performance.text.replans")}</th>
            <th className="py-2 pr-3">{t("supplier_portal.analytics.route_performance.text.computed")}</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.route_id} className="border-b border-[var(--border)]">
              <td className="py-2 font-mono text-xs">{r.route_id}</td>
              <td className="py-2 font-mono text-xs">{r.driver_id ?? "—"}</td>
              <td className="py-2 text-right">{r.planned_stops ?? "—"}</td>
              <td className="py-2 text-right">{r.actual_stops ?? "—"}</td>
              <td className="py-2 text-right">{r.replan_count ?? 0}</td>
              <td className="py-2 text-xs text-[var(--muted)]">{r.computed_at}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </PageChrome>
  );
}
