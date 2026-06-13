"use client";

import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { ApiError } from "@pegasusx/api-client";
import type { SupplierDispatchPreview } from "@pegasusx/types";
import { useDispatchData } from "./use-dispatch-data";
import { PortalSurface } from "../_components/PortalSurface";
import DispatchPreviewMap from "@/components/DispatchPreviewMap";

const api = createSupplierApi();

export default function DispatchPage() {
  const { manifests, loading, error, refresh } = useDispatchData();
  const [preview, setPreview] = useState<SupplierDispatchPreview | null>(null);
  const [previewError, setPreviewError] = useState<string | null>(null);
  const [executing, setExecuting] = useState(false);
  const [executeError, setExecuteError] = useState<string | null>(null);
  const [executeSuccess, setExecuteSuccess] = useState<string | null>(null);

  const loadPreview = useCallback(() => {
    api
      .getSupplierDispatchPreview()
      .then(setPreview)
      .catch((err) => setPreviewError(err instanceof Error ? err.message : "preview_failed"));
  }, []);

  useEffect(() => {
    loadPreview();
  }, [loadPreview]);

  const runAutoDispatch = useCallback(async () => {
    setExecuting(true);
    setExecuteError(null);
    setExecuteSuccess(null);
    try {
      const result = await api.executeSupplierDispatch({ mode: "AUTO" });
      if (result.status === "dispatched") {
        const parts = [
          `Dispatch committed: ${result.manifests_created ?? 0} manifest(s), ${result.orders_assigned ?? 0} order(s).`,
        ];
        if (result.optimizer_source) {
          parts.push(`Optimizer: ${result.optimizer_source}.`);
        }
        if (result.orphan_order_ids?.length) {
          parts.push(`${result.orphan_order_ids.length} order(s) could not be assigned.`);
        }
        setExecuteSuccess(parts.join(" "));
      } else if (result.warnings?.length) {
        setExecuteError(result.warnings.join("; "));
      } else {
        setExecuteSuccess("No orders were dispatched.");
      }
      if (result.warnings?.length && result.status === "dispatched") {
        setExecuteError(result.warnings.join("; "));
      }
      await refresh();
      loadPreview();
    } catch (err) {
      setExecuteError(err instanceof ApiError ? err.message : "dispatch_execute_failed");
    } finally {
      setExecuting(false);
    }
  }, [loadPreview, refresh]);

  const draft = manifests.filter((m) => m.status === "DRAFT");
  const loadingColumn = manifests.filter((m) => m.status === "LOADING");
  const dispatched = manifests.filter((m) => m.status === "DISPATCHED");

  return (
    <PortalSurface
      title="Dispatch queue"
      description="Manage manifests and dispatch operations."
      loading={loading}
      error={error}
      actions={
        <div className="flex gap-3">
          <button
            type="button"
            className="md-btn md-btn-filled"
            disabled={executing || (preview?.pending_count ?? 0) === 0}
            onClick={() => void runAutoDispatch()}
          >
            {executing ? "Dispatching…" : "Auto dispatch"}
          </button>
          <div className="relative">
            <input
              type="text"
              placeholder="Search manifest…"
              className="md-input-outlined h-10 w-64 pl-10"
            />
            <svg
              className="w-5 h-5 absolute left-3 top-2.5"
              style={{ color: "var(--desk-text-secondary)" }}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
              />
            </svg>
          </div>
          <button type="button" className="md-btn md-btn-filled">
            Create manifest
          </button>
        </div>
      }
    >
      <div className="space-y-6 flex flex-col min-h-0">
      <div className="md-card p-4 grid grid-cols-1 md:grid-cols-3 gap-4">
        <div>
          <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Pending dispatch</p>
          <p className="md-typescale-headline-small mt-1">{preview?.pending_count ?? "—"}</p>
        </div>
        <div>
          <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Available drivers</p>
          <p className="md-typescale-headline-small mt-1">{preview?.available_driver_count ?? "—"}</p>
        </div>
        <div>
          <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Unavailable drivers</p>
          <p className="md-typescale-headline-small mt-1">{preview?.unavailable_drivers?.length ?? "—"}</p>
        </div>
        {previewError ? (
          <p className="md-typescale-body-small text-[var(--color-md-error)] md:col-span-3">{previewError}</p>
        ) : null}
        {executeError ? (
          <p className="md-typescale-body-small text-[var(--color-md-error)] md:col-span-3">{executeError}</p>
        ) : null}
        {executeSuccess ? (
          <p className="md-typescale-body-small text-[var(--color-md-success)] md:col-span-3">{executeSuccess}</p>
        ) : null}
      </div>

      {preview?.proposed_routes && preview.proposed_routes.length > 0 ? (
        <div className="md-card p-4 space-y-3">
          <div className="flex items-center justify-between gap-3">
            <h2 className="md-typescale-title-medium">Smart suggest route map</h2>
            {preview.optimizer_source ? (
              <span className="md-typescale-label-small text-[var(--color-md-outline)]">
                Source: {preview.optimizer_source}
              </span>
            ) : null}
          </div>
          <DispatchPreviewMap
            routes={preview.proposed_routes}
            className="h-80 w-full rounded-xl border border-[var(--color-md-outline-variant)] overflow-hidden"
          />
        </div>
      ) : null}

      <div className="flex-1 grid grid-cols-1 lg:grid-cols-3 gap-6 overflow-hidden">
        {/* Draft Column */}
        <div className="flex flex-col bg-[var(--color-md-surface-container-low)] rounded-xl border border-[var(--color-md-outline-variant)] overflow-hidden">
          <div className="p-4 border-b border-[var(--color-md-outline-variant)] flex justify-between items-center bg-[var(--color-md-surface-container)]">
            <h2 className="font-semibold">Draft</h2>
            <span className="bg-[var(--color-md-surface-container-high)] text-sm px-2 py-0.5 rounded-full">{draft.length}</span>
          </div>
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {draft.map((m) => (
              <ManifestCard key={m.id} data={m} />
            ))}
          </div>
        </div>

        {/* Loading Column */}
        <div className="flex flex-col bg-[#fff8e1]/20 dark:bg-[#fff8e1]/5 rounded-xl border border-[var(--color-md-outline-variant)] overflow-hidden">
          <div className="p-4 border-b border-[var(--color-md-outline-variant)] flex justify-between items-center bg-[#fff8e1]/50 dark:bg-[#fff8e1]/10">
            <h2 className="font-semibold">Loading</h2>
            <span className="bg-[#fff8e1] dark:bg-[#410e0b] text-[#b15a00] dark:text-[#ffb4a9] text-sm px-2 py-0.5 rounded-full">{loadingColumn.length}</span>
          </div>
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {loadingColumn.map((m) => (
              <ManifestCard key={m.id} data={m} />
            ))}
          </div>
        </div>

        {/* Dispatched Column */}
        <div className="flex flex-col bg-[var(--color-md-primary-container)]/10 rounded-xl border border-[var(--color-md-outline-variant)] overflow-hidden">
          <div className="p-4 border-b border-[var(--color-md-outline-variant)] flex justify-between items-center bg-[var(--color-md-primary-container)]/30">
            <h2 className="font-semibold">Dispatched</h2>
            <span className="bg-[var(--color-md-primary-container)] text-[var(--color-md-on-primary-container)] text-sm px-2 py-0.5 rounded-full">{dispatched.length}</span>
          </div>
          <div className="flex-1 overflow-y-auto p-4 space-y-4">
            {dispatched.map((m) => (
              <ManifestCard key={m.id} data={m} />
            ))}
          </div>
        </div>
      </div>
      </div>
    </PortalSurface>
  );
}

function ManifestCard({ data }: { data: any }) {
  return (
    <div className="md-card p-4 hover:shadow-md transition-shadow cursor-pointer bg-[var(--color-md-surface)]">
      <div className="flex justify-between items-start mb-3">
        <div className="font-mono text-sm font-medium">{data.id.substring(0, 12)}</div>
        <span className={`md-chip h-6 text-[10px] px-2 ${
          data.status === 'DISPATCHED' ? 'bg-[var(--color-md-primary-container)] text-[var(--color-md-on-primary-container)] border-transparent' :
          data.status === 'LOADING' ? 'bg-[#fff8e1] text-[#b15a00] border-[#b15a00]' : ''
        }`}>
          {data.status}
        </span>
      </div>
      
      <div className="space-y-2 mb-4">
        <div className="flex items-center text-sm gap-2">
          <svg className="w-4 h-4 text-[var(--color-md-outline)]" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" /></svg>
          <span className="truncate font-medium">{data.driverName}</span>
        </div>
        <div className="flex items-center text-sm gap-2">
          <svg className="w-4 h-4 text-[var(--color-md-outline)]" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" /></svg>
          <span className="font-mono text-xs px-1.5 py-0.5 bg-[var(--color-md-surface-container-high)] rounded">{data.vehiclePlate}</span>
        </div>
      </div>

      <div className="flex justify-between items-end border-t border-[var(--color-md-outline-variant)] pt-3">
        <div className="flex gap-4">
          <div className="flex flex-col">
            <span className="text-[10px] text-[var(--color-md-outline)] uppercase">Orders</span>
            <span className="text-sm font-medium">{data.orderCount}</span>
          </div>
          <div className="flex flex-col">
            <span className="text-[10px] text-[var(--color-md-outline)] uppercase">Vol. Units</span>
            <span className="text-sm font-medium">{data.totalVu}</span>
          </div>
        </div>
        <div className="text-sm font-medium text-[var(--color-md-primary)]">
          {data.estimatedTime}
        </div>
      </div>
    </div>
  );
}
