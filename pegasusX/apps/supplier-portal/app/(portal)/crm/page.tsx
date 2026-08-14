"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { ApiError } from "@pegasusx/api-client";
import type { SupplierCRMRetailer, SupplierCRMRetailerDetail } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import EmptyState from "@/components/EmptyState";
import Icon from "@/components/Icon";

const api = createSupplierApi();

function fmtMinor(n: number): string {
  return new Intl.NumberFormat("uz-UZ").format(n);
}

export default function SupplierCRMPage() {
  const t = usePortalT();
  const [retailers, setRetailers] = useState<SupplierCRMRetailer[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<SupplierCRMRetailerDetail | null>(null);
  const [detailError, setDetailError] = useState<string | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await api.getSupplierCRMRetailers();
      setRetailers(resp.retailers ?? []);
    } catch (err) {
      const message =
        err instanceof ApiError
          ? err.status === 503
            ? "CRM unavailable"
            : err.message
          : err instanceof Error
            ? err.message
            : "Failed to load retailers";
      setError(message);
      setRetailers([]);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    if (!selectedId) {
      setDetail(null);
      setDetailError(null);
      return;
    }
    let cancelled = false;
    setDetailLoading(true);
    setDetailError(null);
    api
      .getSupplierCRMRetailer(selectedId)
      .then((row) => {
        if (!cancelled) setDetail(row);
      })
      .catch((err) => {
        if (cancelled) return;
        setDetail(null);
        setDetailError(err instanceof Error ? err.message : "Failed to load retailer");
      })
      .finally(() => {
        if (!cancelled) setDetailLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [selectedId]);

  return (
    <PageChrome
      icon="crm"
      title={t("portal.nav.crm")}
      description="Retailer lifetime rollup for this supplier (order TotalMinor). Not warehouse CRM."
      loading={loading}
      error={error}
      actions={
        <button
          type="button"
          onClick={() => {
            void load();
          }}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary"
        >
          <Icon name="refresh" size={16} /> Refresh
        </button>
      }
    >
      {!loading && !error && retailers.length === 0 ? (
        <EmptyState
          variant="no-data"
          headline="No retailer orders yet"
          body="Retailers appear here after they place orders with this supplier."
        />
      ) : null}

      {retailers.length > 0 ? (
        <div className="overflow-x-auto">
          <table className="desk-table w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)]">
                <th className="text-left py-2 px-3 font-medium">Retailer</th>
                <th className="text-left py-2 px-3 font-medium">Phone</th>
                <th className="text-right py-2 px-3 font-medium">Orders</th>
                <th className="text-right py-2 px-3 font-medium">Lifetime (minor)</th>
                <th className="text-left py-2 px-3 font-medium">Status</th>
                <th className="text-right py-2 px-3 font-medium">Last order</th>
              </tr>
            </thead>
            <tbody>
              {retailers.map((row) => (
                <tr
                  key={row.retailer_id}
                  className="border-b border-[var(--border)] hover:bg-[var(--surface)] transition-colors cursor-pointer"
                  onClick={() => setSelectedId(row.retailer_id)}
                >
                  <td className="py-2.5 px-3 font-medium">{row.retailer_name || "—"}</td>
                  <td className="py-2.5 px-3 text-[var(--muted)]">{row.phone || "—"}</td>
                  <td className="py-2.5 px-3 text-right font-mono">{fmtMinor(row.order_count)}</td>
                  <td className="py-2.5 px-3 text-right font-mono">{fmtMinor(row.lifetime)}</td>
                  <td className="py-2.5 px-3">
                    <span className="text-xs uppercase tracking-wide">{row.status}</span>
                  </td>
                  <td className="py-2.5 px-3 text-right text-[var(--muted)]">
                    {row.last_order_date ? new Date(row.last_order_date).toLocaleDateString() : "—"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      {selectedId ? (
        <section className="desk-card p-6 mt-6">
          <h2 className="bento-card-title">Retailer detail</h2>
          {detailLoading ? <div className="md-skeleton md-skeleton-row mt-3" /> : null}
          {detailError ? <p className="text-sm text-[var(--danger)] mt-2">{detailError}</p> : null}
          {detail ? (
            <div className="mt-3 space-y-3">
              <p className="text-sm text-[var(--muted)]">
                {detail.retailer_name} · {detail.status} · {fmtMinor(detail.lifetime)} minor
              </p>
              <div className="overflow-x-auto">
                <table className="desk-table w-full text-sm">
                  <thead>
                    <tr className="border-b border-[var(--border)]">
                      <th className="text-left py-2 px-3 font-medium">Order</th>
                      <th className="text-left py-2 px-3 font-medium">State</th>
                      <th className="text-right py-2 px-3 font-medium">Amount (minor)</th>
                      <th className="text-right py-2 px-3 font-medium">Items</th>
                      <th className="text-right py-2 px-3 font-medium">Created</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(detail.orders ?? []).map((order) => (
                      <tr key={order.order_id} className="border-b border-[var(--border)]">
                        <td className="py-2 px-3 font-mono text-xs">{order.order_id}</td>
                        <td className="py-2 px-3">{order.state}</td>
                        <td className="py-2 px-3 text-right font-mono">{fmtMinor(order.amount)}</td>
                        <td className="py-2 px-3 text-right font-mono">{order.item_count}</td>
                        <td className="py-2 px-3 text-right text-[var(--muted)]">
                          {order.created_at ? new Date(order.created_at).toLocaleDateString() : "—"}
                        </td>
                      </tr>
                      {(order.lines ?? []).map((line, idx) => (
                        <tr key={`${order.order_id}-line-${idx}`} className="border-b border-[var(--border)] bg-[var(--surface-2)]">
                          <td className="py-1 px-3 pl-8 text-xs" colSpan={2}>
                            {line.product_name || line.sku || "SKU"}
                          </td>
                          <td className="py-1 px-3 text-right font-mono text-xs">{fmtMinor(line.amount_minor ?? 0)}</td>
                          <td className="py-1 px-3 text-right font-mono text-xs">{line.qty}</td>
                          <td className="py-1 px-3 text-right text-xs text-[var(--muted)]">line</td>
                        </tr>
                      ))}
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ) : null}
        </section>
      ) : null}
    </PageChrome>
  );
}
