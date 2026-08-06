'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from 'react';
import { useStableCallback } from '@/lib/useStableCallback';
import { useParams, useRouter } from 'next/navigation';
import { apiFetch, subscribeWarehouseWS, type WarehouseSocketStatus } from '@/lib/auth';
import {
  warehouseReceiveTransferKey,
  warehouseSupplyRequestTransitionKey,
} from '@pegasusx/api-client';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useToast } from '@/components/Toast';
import type { WarehouseLiveEvent, WarehouseSupplyRequestDetail } from '@pegasusx/types';

const ACTIONS: Record<string, { label: string; action: string; variant: string }[]> = {
  DRAFT: [
    { label: 'Cancel', action: 'cancel', variant: 'button--danger' },
  ],
  SUBMITTED: [
    { label: 'Cancel', action: 'cancel', variant: 'button--danger' },
  ],
  ACKNOWLEDGED: [
    { label: 'Cancel', action: 'cancel', variant: 'button--danger' },
  ],
  IN_PRODUCTION: [],
  READY: [],
  FULFILLED: [],
  CANCELLED: [],
};

function chipClass(state: string): string {
  const map: Record<string, string> = {
    DRAFT: 'status-chip--draft',
    SUBMITTED: 'status-chip--submitted',
    ACKNOWLEDGED: 'status-chip--acknowledged',
    IN_PRODUCTION: 'status-chip--in-production',
    READY: 'status-chip--ready',
    FULFILLED: 'status-chip--fulfilled',
    CANCELLED: 'status-chip--cancelled',
  };
  return map[state] || 'status-chip--draft';
}

export default function SupplyRequestDetailPage() {
  const t = usePortalT();
  const params = useParams();
  const router = useRouter();
  const { toast } = useToast();
  const [detail, setDetail] = useState<WarehouseSupplyRequestDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState(false);
  const [receiveQty, setReceiveQty] = useState<Record<string, number>>({});
  const [socketStatus, setSocketStatus] = useState<WarehouseSocketStatus>('connecting');
  const [restricted, setRestricted] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const id = params.id as string;
  const requestId = id === "placeholder" ? "" : id;

  const loadDetail = useStableCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      if (!requestId) return;
      const res = await apiFetch(`/v1/warehouse/supply-requests/${requestId}`);
      if (res.ok) {
        const data = await res.json() as WarehouseSupplyRequestDetail;
        setDetail(data);
        const nextQty: Record<string, number> = {};
        for (const item of data.items ?? []) {
          nextQty[item.item_id] = item.received_quantity ?? item.shipped_quantity ?? item.requested_quantity;
        }
        setReceiveQty(nextQty);
        setRestricted(false);
      } else if (res.status === 403) {
        setRestricted(true);
        setDetail(null);
      } else if (res.status === 404) {
        toast('Request not found', 'error');
        router.replace('/supply-requests');
      } else {
        const data = await res.json().catch(() => ({} as { error?: string }));
        setLoadError(data.error || 'Failed to load request');
      }
    } catch {
      setLoadError('Failed to load request');
    } finally {
      setLoading(false);
    }
  });

  const handleWarehouseLiveEvent = useStableCallback((event: WarehouseLiveEvent) => {
    if (event.type !== 'SUPPLY_REQUEST_UPDATE' || event.request_id !== requestId) {
      return;
    }
    void loadDetail();
  });

  useEffect(() => {
    void loadDetail();
  }, [requestId, loadDetail]);

  useEffect(() => {
    return subscribeWarehouseWS({
      onStatusChange: setSocketStatus,
      onMessage: payload => {
      try {
        handleWarehouseLiveEvent(JSON.parse(payload) as WarehouseLiveEvent);
      } catch {
        // Ignore unrelated frames.
      }
      },
    });
  }, [handleWarehouseLiveEvent]);

  async function handleReceiveTransfer() {
    const transferId = detail?.linked_transfer_id;
    if (!transferId || !detail?.items?.length) return;
    setActing(true);
    try {
      const res = await apiFetch(`/v1/warehouse/transfers/${transferId}/receive`, {
        method: 'POST',
        headers: {
          'Idempotency-Key': warehouseReceiveTransferKey(transferId),
        },
        body: JSON.stringify({
          items: detail.items.map((item) => ({
            item_id: item.item_id,
            received_quantity: receiveQty[item.item_id] ?? item.shipped_quantity ?? item.requested_quantity,
          })),
        }),
      });
      if (res.ok) {
        toast('Transfer received', 'success');
        void loadDetail();
      } else {
        const data = await res.json().catch(() => ({}));
        toast(data.error || 'Receive failed', 'error');
      }
    } catch {
      toast('Network error', 'error');
    } finally {
      setActing(false);
    }
  }

  async function handleAction(action: string) {
    setActing(true);
    try {
      const res = await apiFetch(`/v1/warehouse/supply-requests/${id}`, {
        method: 'PATCH',
        headers: {
          'Idempotency-Key': warehouseSupplyRequestTransitionKey(id, action),
        },
        body: JSON.stringify({ action }),
      });
      if (res.ok) {
        toast(`Request ${action}ed successfully`, 'success');
        void loadDetail();
      } else {
        const data = await res.json().catch(() => ({}));
        toast(data.error || `Failed to ${action}`, 'error');
      }
    } catch {
      toast('Network error', 'error');
    } finally {
      setActing(false);
    }
  }

  if (!loading && !restricted && !loadError && !detail) return null;

  const stateActions = detail ? (ACTIONS[detail.state ?? detail.status ?? 'DRAFT'] || []) : [];

  return (
    <PageTransition>
      <PageChrome
        icon="supplyRequests"
        title={t("warehouse_portal.supply_requests._id_.text.supply_request")}
        description={detail?.request_id}
        loading={loading}
        error={
          restricted
            ? 'You do not have permission to view this supply request.'
            : loadError
        }
        actions={
          <div className="flex items-center gap-2">
            <button type="button" onClick={() => router.back()} className="p-1 rounded-lg hover:bg-[var(--surface)]">
              <Icon name="left" size={20} />
            </button>
            {detail ? (
              <span className={`status-chip ${chipClass(detail.state ?? detail.status ?? 'DRAFT')}`}>
                {detail.state ?? detail.status}
              </span>
            ) : null}
          </div>
        }
      >
      {detail ? (
      <div className="space-y-6">
      {socketStatus !== 'idle' && socketStatus !== 'live' && (
        <div className={`rounded-xl border px-4 py-3 text-sm ${socketStatus === 'offline'
          ? 'border-[var(--danger)]/30 bg-[var(--danger)]/8 text-[var(--danger)]'
          : 'border-[var(--warning)]/30 bg-[var(--warning)]/8 text-[var(--warning)]'}`}>
          {socketStatus === 'offline'
            ? 'Offline. This supply request will not receive live state changes until the network returns.'
            : socketStatus === 'reconnecting'
              ? 'Live request updates are reconnecting. The current state may be stale.'
              : 'Connecting live request updates…'}
        </div>
      )}

      {/* Metadata grid */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--surface)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">{t("warehouse_portal.supply_requests._id_.text.priority")}</div>
          <div className={`text-sm font-semibold ${
            detail.priority === 'CRITICAL' ? 'text-[var(--danger)]' :
            detail.priority === 'URGENT' ? 'text-[var(--warning)]' : ''
          }`}>{detail.priority}</div>
        </div>
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--surface)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">{t("warehouse_portal.supply_requests._id_.text.delivery_date")}</div>
          <div className="text-sm font-semibold">
            {detail.requested_delivery_date
              ? new Date(detail.requested_delivery_date).toLocaleDateString()
              : 'Not set'}
          </div>
        </div>
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--surface)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">{t("warehouse_portal.supply_requests._id_.text.total_volume")}</div>
          <div className="text-sm font-semibold">{detail.total_volume_vu} VU</div>
        </div>
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--surface)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">{t("warehouse_portal.supply_requests._id_.text.items")}</div>
          <div className="text-sm font-semibold">{detail.items?.length || 0}</div>
        </div>
      </div>

      {/* Notes */}
      {detail.notes && (
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--surface)' }}>
          <div className="text-xs text-[var(--muted)] mb-2">{t("warehouse_portal.supply_requests._id_.text.notes")}</div>
          <p className="text-sm whitespace-pre-wrap">{detail.notes}</p>
        </div>
      )}

      {/* Items table */}
      {detail.items && detail.items.length > 0 && (
        <div className="border border-[var(--border)] rounded-xl overflow-hidden">
          <table className="desk-table w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)]" style={{ background: 'var(--surface)' }}>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">{t("supplier_portal.admin.empathy.hierarchy.product.level")}</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.supply_requests._id_.text.requested")}</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.supply_requests._id_.text.shipped")}</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.supply_requests._id_.text.received")}</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.supply_requests._id_.text.recommended")}</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.supply_requests._id_.text.unit_volume")}</th>
              </tr>
            </thead>
            <tbody>
              {detail.items.map(item => (
                <tr key={item.item_id} className="border-b border-[var(--border)] last:border-b-0">
                  <td className="px-4 py-3 font-mono text-xs">{item.product_id}</td>
                  <td className="px-4 py-3 font-mono">{item.requested_quantity}</td>
                  <td className="px-4 py-3 font-mono">{item.shipped_quantity ?? '—'}</td>
                  <td className="px-4 py-3 font-mono">
                    {detail.linked_transfer_id ? (
                      <input
                        type="number"
                        min={0}
                        className="w-20 rounded border border-[var(--border)] px-2 py-1"
                        value={receiveQty[item.item_id] ?? item.shipped_quantity ?? item.requested_quantity}
                        onChange={(e) => setReceiveQty((prev) => ({
                          ...prev,
                          [item.item_id]: Number(e.target.value),
                        }))}
                      />
                    ) : (
                      item.received_quantity ?? '—'
                    )}
                  </td>
                  <td className="px-4 py-3 font-mono">{item.recommended_qty}</td>
                  <td className="px-4 py-3 text-[var(--muted)]">{item.unit_volume_vu} VU</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Actions */}
      {stateActions.length > 0 && (
        <div className="flex gap-2">
          {stateActions.map((a: { label: string; action: string; variant: string }) => (
            <button
              key={a.action}
              onClick={() => handleAction(a.action)}
              disabled={acting}
              className={`px-4 py-2 rounded-lg text-sm font-semibold ${a.variant} disabled:opacity-50`}
            >
              {acting ? '...' : a.label}
            </button>
          ))}
        </div>
      )}

      {detail.linked_transfer_id && ['FULFILLED', 'ARRIVED', 'IN_TRANSIT'].includes(detail.state ?? detail.status ?? '') && (
        <div className="flex gap-2">
          <button
            type="button"
            onClick={() => void handleReceiveTransfer()}
            disabled={acting}
            className="px-4 py-2 rounded-lg text-sm font-semibold button--primary disabled:opacity-50"
          >
            {acting ? 'Receiving…' : 'Confirm receipt'}
          </button>
        </div>
      )}

      {/* Transfer order link */}
      {detail.transfer_order_id && (
        <div className="text-xs text-[var(--muted)]">
          Linked Transfer Order: <span className="font-mono">{detail.transfer_order_id}</span>
        </div>
      )}
      </div>
      ) : null}
      </PageChrome>
    </PageTransition>
  );
}
