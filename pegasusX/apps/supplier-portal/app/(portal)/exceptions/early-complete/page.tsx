"use client";

import Link from "next/link";
import type { Route } from "next";
import { useState } from "react";
import { supplierApproveEarlyCompleteKey } from "@pegasusx/api-client";
import { createSupplierApi } from "@/lib/api";
import { ApiError } from "@pegasusx/api-client";
import { PortalSurface } from "../../_components/PortalSurface";

const api = createSupplierApi();

export default function EarlyCompletePage() {
  const [driverId, setDriverId] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const approve = async () => {
    const trimmed = driverId.trim();
    if (!trimmed) {
      setError("Driver ID is required.");
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
      setError(err instanceof ApiError ? err.message : "approve_failed");
    } finally {
      setBusy(false);
    }
  };

  return (
    <PortalSurface
      title="Early route complete"
      description="Approve a driver request to finish the route before all stops are completed."
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
          <span className="md-typescale-label-medium">Driver ID</span>
          <input
            type="text"
            value={driverId}
            onChange={(event) => setDriverId(event.target.value)}
            placeholder="Driver UUID from escalation or telemetry"
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
    </PortalSurface>
  );
}
