"use client";

import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { supplierShopClosedResolveKey } from "@pegasusx/api-core";
import type { ShopClosedAttemptRow } from "@pegasusx/types";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from '@/components/PageChrome';

const api = createSupplierApi();

export default function ShopClosedExceptionsPage() {
  const t = usePortalT();
  const [rows, setRows] = useState<ShopClosedAttemptRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    api
      .getSupplierShopClosedActive({ limit: 500, offset: 0 })
      .then((resp) => setRows(resp.data ?? []))
      .catch((err) => setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_failed")))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  useSupplierSessionReconcile(() => {
    if (busyId) {
      setBusyId(null);
      setError(t("supplier_portal.residual.text.connection_restored_escalation_queue_refreshed_from_server"));
    }
    load();
  });

  const resolve = async (attemptId: string, action: "WAIT" | "BYPASS" | "RETURN_TO_DEPOT") => {
    setBusyId(attemptId);
    try {
      const body = await api.resolveSupplierShopClosed(
        { attempt_id: attemptId, action },
        supplierShopClosedResolveKey(attemptId, action),
      );
      if (body.queued) {
        setError(t("supplier_portal.residual.text.resolution_queued_for_retry_when_back_online"));
      } else {
        load();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.resolve_failed"));
    } finally {
      setBusyId(null);
    }
  };

  return (
    <PageChrome
      icon="warning"
      title={t("supplier_portal.exceptions.shop_closed.text.shop_closed_escalations")}
      description={t("supplier_portal.residual.text.active_driver_reports_where_the_retailer_did_not_confirm_within_")}
      loading={loading}
      error={error}
      empty={!loading && rows.length === 0}
      emptyMessage={t("supplier_portal.residual.text.no_escalations_in_queue")}
    >
      <p className="md-typescale-body-medium text-[var(--color-md-outline)]">
        <Link href="/exceptions" className="text-[var(--color-md-primary)] underline">
          All exceptions
        </Link>
      </p>
      <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
        {rows.map((row) => (
          <li key={row.attempt_id} className="p-4 space-y-3">
            <div className="flex flex-wrap gap-2 items-center">
              <span className="md-chip h-6 text-xs">{row.resolution || "ESCALATED"}</span>
              {row.shop_closed_resolution ? (
                <span className="md-chip h-6 text-xs">order: {row.shop_closed_resolution}</span>
              ) : null}
              {row.shop_closed_reason ? (
                <span className="md-chip h-6 text-xs">{row.shop_closed_reason}</span>
              ) : null}
              <span className="font-mono text-[var(--color-md-primary)]">{row.order_id}</span>
            </div>
            <p className="md-typescale-body-small text-[var(--color-md-outline)]">
              Driver {row.driver_id}
              {row.original_route_id ? ` · Route ${row.original_route_id}` : ""}
              {row.grace_ends_at ? ` · Grace ends ${row.grace_ends_at}` : ""}
            </p>
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                className="md-btn md-btn-tonal md-typescale-label-large"
                disabled={busyId === row.attempt_id}
                onClick={() => resolve(row.attempt_id, "WAIT")}
              >
                Wait
              </button>
              <button
                type="button"
                className="md-btn md-btn-filled md-typescale-label-large"
                disabled={busyId === row.attempt_id}
                onClick={() => resolve(row.attempt_id, "BYPASS")}
              >
                Bypass
              </button>
              <button
                type="button"
                className="md-btn md-btn-outlined md-typescale-label-large"
                disabled={busyId === row.attempt_id}
                onClick={() => resolve(row.attempt_id, "RETURN_TO_DEPOT")}
              >
                Return to depot
              </button>
            </div>
          </li>
        ))}
      </ul>
    </PageChrome>
  );
}
