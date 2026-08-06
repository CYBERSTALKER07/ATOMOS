"use client";

import { usePortalT } from "@/lib/i18n";
import type { ReactNode } from "react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import {
  supplierManifestInjectKey,
  supplierManifestSealKey,
  supplierManifestStartLoadingKey,
} from "@pegasusx/api-client";
import type { SupplierManifestDetail } from "@pegasusx/types";
import StatusBadge from "@/components/StatusBadge";
import { PageChrome } from '@/components/PageChrome';
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";

const api = createSupplierApi();

export default function ManifestDetailPage() {
  const t = usePortalT();
  const params = useParams<{ id: string }>();
  const manifestId = decodeURIComponent(params.id);
  const [manifest, setManifest] = useState<SupplierManifestDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [injectOrderId, setInjectOrderId] = useState("");

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    api
      .getSupplierManifestDetail(manifestId)
      .then(setManifest)
      .catch((err) => setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_manifest_failed")))
      .finally(() => setLoading(false));
  }, [manifestId]);

  useEffect(() => {
    load();
  }, [load]);

  useSupplierSessionReconcile(() => {
    load();
    if (busy) {
      setBusy(null);
      setActionError(null);
    }
  });

  const runAction = async (label: string, action: () => Promise<void>) => {
    setBusy(label);
    setActionError(null);
    try {
      await action();
      load();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : `${label}_failed`);
    } finally {
      setBusy(null);
    }
  };

  const state = (manifest?.state || manifest?.status || "").toUpperCase();

  return (
    <PageChrome
      icon="manifests"
      title={t("supplier_portal.manifests._id_.text.manifest_detail")}
      description={manifestId}
      loading={loading}
      error={error}
      empty={!manifest}
    >
      {manifest ? (
        <div className="space-y-6">
          <div className="md-card p-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <Metric label={t("supplier_portal.compliance.text.status")} value={<StatusBadge state={manifest.status} />} />
            <Metric label={t("portal.nav.orders")} value={String(manifest.orders_count)} />
            <Metric label={t("supplier_portal.analytics.route_performance.text.driver")} value={manifest.driver_name || manifest.driver_id || "—"} />
            <Metric label={t("supplier_portal.org_fleet.components.driver_table.text.vehicle")} value={manifest.vehicle_plate || manifest.vehicle_id || "—"} />
            <Metric label={t("supplier_portal.residual.text.volume_vu")} value={String(manifest.total_volume_vu ?? manifest.total_vu)} />
            <Metric label={t("supplier_portal.residual.text.max_vu")} value={String(manifest.max_volume_vu ?? "—")} />
            <Metric label={t("warehouse_portal.payment_config.text.updated")} value={new Date(manifest.updated_at).toLocaleString()} />
          </div>

          {actionError ? (
            <div className="md-card p-3 md-typescale-body-medium" style={{ color: "var(--color-md-error)" }}>
              {actionError}
            </div>
          ) : null}

          <div className="flex flex-wrap gap-3">
            {state === "DRAFT" ? (
              <button
                type="button"
                className="md-btn md-btn-filled md-typescale-label-large px-4 py-2"
                disabled={busy !== null}
                onClick={() =>
                  void runAction("start_loading", async () => {
                    await api.startSupplierManifestLoading(
                      manifestId,
                      supplierManifestStartLoadingKey(manifestId),
                    );
                  })
                }
              >
                {busy === "start_loading" ? "Starting…" : "Start loading"}
              </button>
            ) : null}
            {state === "LOADING" ? (
              <>
                <div className="flex flex-wrap items-end gap-2">
                  <label className="block">
                    <div className="md-typescale-label-medium text-[var(--color-md-outline)]">{t("supplier_portal.manifests._id_.text.inject_order_id")}</div>
                    <input
                      className="md-input-outlined mt-1 px-3 py-2"
                      value={injectOrderId}
                      onChange={(event) => setInjectOrderId(event.target.value)}
                      placeholder={t("supplier_portal.manifests._id_.text.order_uuid")}
                    />
                  </label>
                  <button
                    type="button"
                    className="md-btn md-btn-tonal md-typescale-label-large px-4 py-2"
                    disabled={busy !== null || !injectOrderId.trim()}
                    onClick={() =>
                      void runAction("inject_order", async () => {
                        const orderId = injectOrderId.trim();
                        await api.injectSupplierManifestOrder(
                          manifestId,
                          { order_id: orderId },
                          supplierManifestInjectKey(manifestId, orderId),
                        );
                        setInjectOrderId("");
                      })
                    }
                  >
                    {busy === "inject_order" ? "Injecting…" : "Inject order"}
                  </button>
                </div>
                <button
                  type="button"
                  className="md-btn md-btn-filled md-typescale-label-large px-4 py-2"
                  disabled={busy !== null}
                  onClick={() =>
                    void runAction("seal", async () => {
                    await api.sealSupplierManifest(
                      manifestId,
                      supplierManifestSealKey(manifestId, "supplier"),
                    );
                  })
                  }
                >
                  {busy === "seal" ? "Sealing…" : "Seal manifest"}
                </button>
              </>
            ) : null}
            <Link href="/manifest-exceptions" className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2 inline-flex">
              Gate exceptions
            </Link>
            <Link href="/manifests" className="md-btn md-btn-text md-typescale-label-large px-4 py-2 inline-flex">
              Back to queue
            </Link>
          </div>

          {manifest.orders && manifest.orders.length > 0 ? (
            <div className="md-card overflow-hidden">
              <table className="desk-table w-full">
                <thead>
                  <tr className="border-b border-[var(--color-md-outline-variant)] text-[var(--color-md-outline)]">
                    <th className="md-typescale-label-medium p-4 font-medium">{t("supplier_portal.chargebacks.claims.text.order")}</th>
                    <th className="md-typescale-label-medium p-4 font-medium">{t("supplier_portal.compliance.text.status")}</th>
                    <th className="md-typescale-label-medium p-4 font-medium text-right">{t("supplier_portal.ledger.text.amount")}</th>
                  </tr>
                </thead>
                <tbody>
                  {manifest.orders.map((order) => (
                    <tr key={order.order_id} className="border-b border-[var(--color-md-outline-variant)] last:border-0">
                      <td className="p-4 md-typescale-body-medium font-mono">{order.order_id}</td>
                      <td className="p-4 md-typescale-body-medium">
                        <StatusBadge state={order.status || order.state} />
                      </td>
                      <td className="p-4 md-typescale-body-medium text-right">{order.amount}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : null}
        </div>
      ) : null}
    </PageChrome>
  );
}

function Metric({ label, value }: { label: string; value: ReactNode }) {
  return (
    <div>
      <div className="md-typescale-label-medium text-[var(--color-md-outline)]">{label}</div>
      <div className="md-typescale-title-medium mt-1">{value}</div>
    </div>
  );
}
