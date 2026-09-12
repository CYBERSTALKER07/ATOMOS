"use client";

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierReplenishmentTraceRow } from "@pegasusx/types";

const api = createSupplierApi();

export default function ReplenishmentTraceabilityPanel() {
  const t = usePortalT();
  const [rows, setRows] = useState<SupplierReplenishmentTraceRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    api
      .getSupplierReplenishmentTraceability()
      .then((resp) => {
        if (!cancelled) setRows(resp.rows ?? []);
      })
      .catch(() => {
        if (!cancelled) setError(t("supplier_portal.residual.text.traceability_load_failed"));
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <section className="desk-card p-6 mt-6 overflow-x-auto">
      <h2 className="bento-card-title">{t("supplier_portal.replenishment_traceability_panel.text.replenishment_traceability")}</h2>
      <p className="md-typescale-body-small mt-2" style={{ color: "var(--desk-text-secondary)" }}>
        Insight ID → factory transfer ID for touchless and warehouse-approved replenishment loops.
      </p>

      {error ? (
        <p className="md-typescale-body-small mt-4" style={{ color: "var(--desk-danger)" }}>
          {error}
        </p>
      ) : loading ? (
        <p className="md-typescale-body-small mt-4" style={{ color: "var(--desk-text-secondary)" }}>
          Loading traceability rows…
        </p>
      ) : rows.length === 0 ? (
        <p className="md-typescale-body-small mt-4" style={{ color: "var(--desk-text-secondary)" }}>
          No replenishment insights yet — trigger a cycle from operations or wait for predictive push.
        </p>
      ) : (
        <table className="desk-table w-full mt-4">
          <thead>
            <tr style={{ color: "var(--desk-text-secondary)" }}>
              <th className="md-typescale-label-medium p-3 text-left font-medium">{t("supplier_portal.replenishment_traceability_panel.text.insight")}</th>
              <th className="md-typescale-label-medium p-3 text-left font-medium">{t("supplier_portal.admin.empathy.hierarchy.product.level")}</th>
              <th className="md-typescale-label-medium p-3 text-left font-medium">{t("supplier_portal.replenishment_traceability_panel.text.warehouse")}</th>
              <th className="md-typescale-label-medium p-3 text-left font-medium">{t("supplier_portal.compliance.text.status")}</th>
              <th className="md-typescale-label-medium p-3 text-left font-medium">{t("supplier_portal.replenishment_traceability_panel.text.transfer")}</th>
              <th className="md-typescale-label-medium p-3 text-left font-medium">{t("supplier_portal.admin.control_center.field.reason")}</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={row.insight_id} style={{ borderTop: "1px solid var(--desk-border)" }}>
                <td className="p-3 md-typescale-body-medium font-mono text-xs">{row.insight_id.slice(0, 8)}…</td>
                <td className="p-3 md-typescale-body-medium">{row.product_name || row.product_id}</td>
                <td className="p-3 md-typescale-body-medium">{row.warehouse_name || row.warehouse_id}</td>
                <td className="p-3 md-typescale-body-medium">{row.status}</td>
                <td className="p-3 md-typescale-body-medium font-mono text-xs">
                  {row.transfer_id ? (
                    <span title={row.transfer_id}>
                      {row.transfer_id.slice(0, 8)}…
                      {row.transfer_state ? ` (${row.transfer_state})` : ""}
                    </span>
                  ) : (
                    "—"
                  )}
                </td>
                <td className="p-3 md-typescale-body-medium text-xs" style={{ color: "var(--desk-text-secondary)" }}>
                  {row.reason_code?.replaceAll("_", " ") ?? "—"}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
