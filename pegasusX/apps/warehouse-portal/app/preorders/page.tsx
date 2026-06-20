'use client';

import { useCallback, useEffect, useState } from 'react';
import { ApiError } from '@pegasusx/api-client';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseOps } from '@/lib/warehouse-ops';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';

interface PreorderRow {
  order_id: string;
  status: string;
  order_source?: string;
  confirmation_status?: string;
  requested_delivery_date?: string;
  proposed_delivery_date?: string;
  delivery_proposal_reason?: string;
  preorder_badge?: string;
  total_minor?: number;
}

function isoDeliveryDate(dateInput: string): string {
  const dateOnly = dateInput.slice(0, 10);
  return `${dateOnly}T12:00:00+05:00`;
}

function showsReviewBadge(row: PreorderRow): boolean {
  return row.confirmation_status === 'PENDING_WAREHOUSE' || row.preorder_badge === 'REVIEW_DELIVERY';
}

export default function PreordersPage() {
  const { toast } = useToast();
  const [items, setItems] = useState<PreorderRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [actingId, setActingId] = useState<string | null>(null);
  const [proposeTarget, setProposeTarget] = useState<string | null>(null);
  const [rejectTarget, setRejectTarget] = useState<string | null>(null);
  const [proposedDate, setProposedDate] = useState('');
  const [reason, setReason] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await warehouseApi.getWarehousePreorders();
      const rows = (data.preorders ?? data.items ?? []) as PreorderRow[];
      setItems(rows);
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Failed to load pre-orders', 'error');
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    void load();
  }, [load]);

  const openPropose = (row: PreorderRow) => {
    setProposeTarget(row.order_id);
    setRejectTarget(null);
    setReason('');
    setProposedDate((row.requested_delivery_date ?? '').slice(0, 10) || new Date().toISOString().slice(0, 10));
  };

  const openReject = (row: PreorderRow) => {
    setRejectTarget(row.order_id);
    setProposeTarget(null);
    setReason('');
  };

  const closeActions = () => {
    setProposeTarget(null);
    setRejectTarget(null);
    setReason('');
  };

  async function submitPropose(orderId: string) {
    const trimmedReason = reason.trim();
    if (!proposedDate || !trimmedReason) {
      toast('Date and reason are required', 'error');
      return;
    }
    setActingId(orderId);
    try {
      const resp = await warehouseOps.proposePreorderDelivery(orderId, isoDeliveryDate(proposedDate), trimmedReason);
      toast(`Delivery date proposed · ${resp.status ?? 'ok'}`, 'success');
      closeActions();
      await load();
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Propose failed', 'error');
    } finally {
      setActingId(null);
    }
  }

  async function submitReject(orderId: string) {
    const trimmedReason = reason.trim();
    if (!trimmedReason) {
      toast('Reason is required', 'error');
      return;
    }
    setActingId(orderId);
    try {
      const resp = await warehouseOps.rejectPreorder(orderId, trimmedReason);
      toast(`Pre-order rejected · ${resp.status ?? 'ok'}`, 'success');
      closeActions();
      await load();
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Reject failed', 'error');
    } finally {
      setActingId(null);
    }
  }

  return (
    <PageTransition>
      <PageChrome
        title="Pre-orders"
        description="Scheduled and auto-accepted manual pre-orders"
        loading={loading}
        empty={!loading && items.length === 0}
        emptyMessage="Scheduled pre-orders appear here for T-2 edits and stock planning."
      >
        {items.length > 0 ? (
          <div className="overflow-x-auto rounded-2xl border border-[var(--desk-border)]">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-[var(--desk-border)] text-left text-[var(--desk-text-tertiary)]">
                  <th className="p-3">Order</th>
                  <th className="p-3">Status</th>
                  <th className="p-3">Delivery</th>
                  <th className="p-3">Proposed</th>
                  <th className="p-3">Reason</th>
                  <th className="p-3">Badge</th>
                  <th className="p-3 text-right">Actions</th>
                </tr>
              </thead>
              <tbody>
                {items.map((row) => {
                  const busy = actingId === row.order_id;
                  const proposing = proposeTarget === row.order_id;
                  const rejecting = rejectTarget === row.order_id;
                  return (
                    <tr key={row.order_id} className="border-b border-[var(--desk-border)] last:border-0 align-top">
                      <td className="p-3 font-mono">{row.order_id}</td>
                      <td className="p-3">{row.status}</td>
                      <td className="p-3">{row.requested_delivery_date ?? '—'}</td>
                      <td className="p-3 text-[var(--desk-accent)]">{row.proposed_delivery_date ?? '—'}</td>
                      <td className="p-3 max-w-[12rem] truncate" title={row.delivery_proposal_reason ?? undefined}>
                        {row.delivery_proposal_reason ?? '—'}
                      </td>
                      <td className="p-3">
                        {showsReviewBadge(row) ? (
                          <span className="status-chip status-chip--warning">Awaiting retailer review</span>
                        ) : (
                          row.preorder_badge ?? 'Pre-order'
                        )}
                      </td>
                      <td className="p-3 text-right">
                        <div className="flex flex-col items-end gap-2">
                          <div className="flex flex-wrap justify-end gap-2">
                            <button
                              type="button"
                              className="rounded-lg border border-[var(--desk-border)] px-3 py-1.5 text-xs hover:bg-[var(--desk-surface-hover)] disabled:opacity-50"
                              disabled={busy}
                              onClick={() => openPropose(row)}
                            >
                              Propose
                            </button>
                            <button
                              type="button"
                              className="rounded-lg border border-red-500/40 px-3 py-1.5 text-xs text-red-600 hover:bg-red-500/10 disabled:opacity-50"
                              disabled={busy}
                              onClick={() => openReject(row)}
                            >
                              Reject
                            </button>
                          </div>
                          {proposing ? (
                            <div className="w-full max-w-xs rounded-xl border border-[var(--desk-border)] p-3 text-left space-y-2">
                              <label className="block text-xs text-[var(--desk-text-tertiary)]">
                                Proposed date
                                <input
                                  type="date"
                                  className="mt-1 w-full rounded border border-[var(--desk-border)] p-2 text-sm"
                                  value={proposedDate}
                                  onChange={(e) => setProposedDate(e.target.value)}
                                />
                              </label>
                              <label className="block text-xs text-[var(--desk-text-tertiary)]">
                                Reason
                                <input
                                  className="mt-1 w-full rounded border border-[var(--desk-border)] p-2 text-sm"
                                  value={reason}
                                  onChange={(e) => setReason(e.target.value)}
                                  placeholder="Reason for date change"
                                />
                              </label>
                              <div className="flex justify-end gap-2">
                                <button type="button" className="text-xs px-2 py-1" onClick={closeActions}>
                                  Cancel
                                </button>
                                <button
                                  type="button"
                                  className="rounded bg-[var(--desk-accent)] px-3 py-1 text-xs text-white disabled:opacity-50"
                                  disabled={busy || !reason.trim() || !proposedDate}
                                  onClick={() => void submitPropose(row.order_id)}
                                >
                                  Send proposal
                                </button>
                              </div>
                            </div>
                          ) : null}
                          {rejecting ? (
                            <div className="w-full max-w-xs rounded-xl border border-[var(--desk-border)] p-3 text-left space-y-2">
                              <label className="block text-xs text-[var(--desk-text-tertiary)]">
                                Rejection reason
                                <input
                                  className="mt-1 w-full rounded border border-[var(--desk-border)] p-2 text-sm"
                                  value={reason}
                                  onChange={(e) => setReason(e.target.value)}
                                  placeholder="Required"
                                />
                              </label>
                              <div className="flex justify-end gap-2">
                                <button type="button" className="text-xs px-2 py-1" onClick={closeActions}>
                                  Cancel
                                </button>
                                <button
                                  type="button"
                                  className="rounded bg-red-600 px-3 py-1 text-xs text-white disabled:opacity-50"
                                  disabled={busy || !reason.trim()}
                                  onClick={() => void submitReject(row.order_id)}
                                >
                                  Confirm reject
                                </button>
                              </div>
                            </div>
                          ) : null}
                        </div>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        ) : !loading ? (
          <EmptyState
            headline="No pre-orders"
            body="Scheduled pre-orders appear here for T-2 edits and stock planning."
          />
        ) : null}
      </PageChrome>
    </PageTransition>
  );
}
