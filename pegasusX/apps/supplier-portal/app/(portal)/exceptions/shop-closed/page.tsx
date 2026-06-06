"use client";

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { ShopClosedAttemptRow } from "@pegasusx/types";
import { PortalSurface } from "../../_components/PortalSurface";

const api = createSupplierApi();

function resolveIdempotencyKey(attemptId: string, action: string): string {
  return `shop-closed-resolve:${attemptId}:${action}`;
}

export default function ShopClosedExceptionsPage() {
  const [rows, setRows] = useState<ShopClosedAttemptRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    api
      .getSupplierShopClosedActive()
      .then((resp) => setRows(resp.data ?? []))
      .catch((err) => setError(err instanceof Error ? err.message : "load_failed"))
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const resolve = async (attemptId: string, action: "WAIT" | "BYPASS" | "RETURN_TO_DEPOT") => {
    setBusyId(attemptId);
    try {
      const body = await api.resolveSupplierShopClosed(
        { attempt_id: attemptId, action },
        resolveIdempotencyKey(attemptId, action),
      );
      if (body.queued) {
        setError("Resolution queued for retry when back online.");
      } else {
        load();
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "resolve_failed");
    } finally {
      setBusyId(null);
    }
  };

  return (
    <PortalSurface
      title="Shop closed escalations"
      description="Active driver reports where the retailer did not confirm within the grace window."
      loading={loading}
      error={error}
      empty={!loading && rows.length === 0}
      emptyMessage="No escalations in queue."
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
              <span className="font-mono text-[var(--color-md-primary)]">{row.order_id}</span>
            </div>
            <p className="md-typescale-body-small text-[var(--color-md-outline)]">
              Driver {row.driver_id}
              {row.original_route_id ? ` · Route ${row.original_route_id}` : ""}
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
    </PortalSurface>
  );
}
