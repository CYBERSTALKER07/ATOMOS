"use client";

import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierManifestExceptionRow } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

const REASON_COLORS: Record<string, string> = {
  OVERFLOW: "var(--color-md-warning)",
  DAMAGED: "var(--color-md-error)",
  MANUAL: "var(--color-md-info)",
  NO_CAPACITY: "var(--color-md-error)",
};

function shortId(id: string): string {
  return id.length > 12 ? `${id.slice(0, 8)}…` : id;
}

export default function ManifestExceptionsPage() {
  const t = usePortalT();
  const [exceptions, setExceptions] = useState<SupplierManifestExceptionRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [escalatedOnly, setEscalatedOnly] = useState(false);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    api
      .getSupplierManifestExceptions({ escalated: escalatedOnly })
      .then((resp) => setExceptions(resp.exceptions))
      .catch((err) => setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_manifest_exceptions_failed")))
      .finally(() => setLoading(false));
  }, [escalatedOnly]);

  useSupplierSessionReconcile(load);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <PageChrome
      icon="warning"
      title={t("supplier_portal.manifest_exceptions.text.manifest_gate_exceptions")}
      description={t("supplier_portal.residual.text.loading_gate_overflows_damage_reports_and_manual_removals_raised")}
      loading={loading}
      error={error}
      empty={!loading && exceptions.length === 0}
      emptyMessage={t("supplier_portal.residual.text.no_manifest_exceptions_in_the_current_window")}
    >
      <div className="flex flex-wrap items-center gap-3 mb-4">
        <label className="flex items-center gap-2 md-typescale-body-medium">
          <input
            type="checkbox"
            checked={escalatedOnly}
            onChange={(event) => setEscalatedOnly(event.target.checked)}
          />
          Escalated only
        </label>
        <button type="button" className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2" onClick={load}>
          Refresh
        </button>
        <Link href="/manifests" className="md-btn md-btn-text md-typescale-label-large px-2">
          Manifest queue
        </Link>
      </div>
      <div className="md-card overflow-hidden">
        <table className="desk-table w-full">
          <thead>
            <tr className="border-b border-[var(--color-md-outline-variant)] text-[var(--color-md-outline)]">
              <th className="md-typescale-label-medium p-4 font-medium">{t("supplier_portal.admin.control_center.field.reason")}</th>
              <th className="md-typescale-label-medium p-4 font-medium">{t("supplier_portal.manifest_exceptions.text.manifest")}</th>
              <th className="md-typescale-label-medium p-4 font-medium">{t("supplier_portal.chargebacks.claims.text.order")}</th>
              <th className="md-typescale-label-medium p-4 font-medium text-right">{t("supplier_portal.manifest_exceptions.text.attempts")}</th>
              <th className="md-typescale-label-medium p-4 font-medium">{t("supplier_portal.analytics.demand.flywheel.text.when")}</th>
            </tr>
          </thead>
          <tbody>
            {exceptions.map((row) => (
              <tr key={row.exception_id} className="border-b border-[var(--color-md-outline-variant)] last:border-0">
                <td className="p-4 md-typescale-body-medium">
                  <span
                    className="md-chip h-6 text-xs"
                    style={{
                      background: REASON_COLORS[row.reason] ?? "var(--color-md-outline)",
                      color: "#fff",
                    }}
                  >
                    {row.reason}
                  </span>
                  {row.escalated ? (
                    <span className="ml-2 md-chip h-6 text-xs" style={{ background: "var(--color-md-error)", color: "#fff" }}>
                      Escalated
                    </span>
                  ) : null}
                </td>
                <td className="p-4 md-typescale-body-medium font-mono">
                  <Link href={`/manifests/${row.manifest_id}`} className="text-[var(--color-md-primary)] underline">
                    {shortId(row.manifest_id)}
                  </Link>
                </td>
                <td className="p-4 md-typescale-body-medium font-mono">{shortId(row.order_id)}</td>
                <td className="p-4 md-typescale-body-medium text-right">{row.attempt_count}</td>
                <td className="p-4 md-typescale-body-medium text-[var(--color-md-outline)]">
                  {new Date(row.created_at).toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </PageChrome>
  );
}
