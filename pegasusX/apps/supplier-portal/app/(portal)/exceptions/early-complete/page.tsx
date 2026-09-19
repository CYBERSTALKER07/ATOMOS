"use client";

import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import type { Route } from "next";
import { useState } from "react";
import { supplierApproveEarlyCompleteKey } from "@pegasusx/api-core";
import { createSupplierApi } from "@/lib/api";
import { ApiError } from "@pegasusx/api-core";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from '@/components/PageChrome';

const api = createSupplierApi();

export default function EarlyCompletePage() {
  const t = usePortalT();
  const [driverId, setDriverId] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  useSupplierSessionReconcile(() => {
    if (busy) {
      setBusy(false);
      setError(t("supplier_portal.residual.text.connection_restored_verify_approval_status_before_retrying"));
    }
  });

  const approve = async () => {
    const trimmed = driverId.trim();
    if (!trimmed) {
      setError(t("supplier_portal.residual.text.driver_id_is_required"));
      return;
    }
    setBusy(true);
    setError(null);
    setSuccess(null);
    try {
      await api.approveSupplierEarlyComplete(
        { driver_id: trimmed },
        supplierApproveEarlyCompleteKey(trimmed),
      );
      setSuccess(`Early route complete approved for driver ${trimmed.slice(0, 12)}…`);
      setDriverId("");
    } catch (err) {
      setError(err instanceof ApiError ? err.message : t("supplier_portal.residual.text.approve_failed"));
    } finally {
      setBusy(false);
    }
  };

  return (
    <PageChrome
      icon="warning"
      title={t("supplier_portal.exceptions.early_complete.text.early_route_complete")}
      description={t("supplier_portal.residual.text.approve_a_driver_request_to_finish_the_route_before_all_stops_ar")}
      error={error}
    >
      <p className="md-typescale-body-medium text-[var(--color-md-outline)]">
        <Link href={"/exceptions" as Route} className="text-[var(--color-md-primary)] underline">
          All exceptions
        </Link>
      </p>
      {success ? (
        <p className="md-typescale-body-medium text-[var(--color-md-success)]">{success}</p>
      ) : null}
      <div className="md-card p-4 space-y-4 max-w-xl">
        <label className="block space-y-2">
          <span className="md-typescale-label-medium">{t("supplier_portal.exceptions.early_complete.text.driver_id")}</span>
          <input
            type="text"
            value={driverId}
            onChange={(event) => setDriverId(event.target.value)}
            placeholder={t("supplier_portal.exceptions.early_complete.text.driver_uuid_from_escalation_or_telemetry")}
            className="md-input-outlined w-full h-10 px-3"
          />
        </label>
        <button
          type="button"
          className="md-btn md-btn-filled"
          disabled={busy || driverId.trim().length === 0}
          onClick={() => void approve()}
        >
          {busy ? "Approving…" : "Approve early complete"}
        </button>
      </div>
    </PageChrome>
  );
}
