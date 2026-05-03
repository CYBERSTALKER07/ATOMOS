'use client';

import { useCallback, useState } from 'react';
import { Button } from '@heroui/react';
import { useToken } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import EmptyState from '@/components/EmptyState';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface RefundRecord {
  refund_id: string;
  status: string;
  amount_uzs: number;
  gateway: string;
  provider_refund_id?: string;
}

function formatAmount(v: number): string {
  return new Intl.NumberFormat('en-US').format(v);
}

function buildRefundIdempotencyKey(orderId: string, reason: string, amountUZS: number): string {
  return ['refund', orderId.trim(), reason.trim().toUpperCase(), String(amountUZS)].join(':');
}

export default function RefundsPage() {
  const token = useToken();
  const { toast } = useToast();

  const [orderId, setOrderId] = useState('');
  const [reason, setReason] = useState('Customer request');
  const [amountUZS, setAmountUZS] = useState<string>('0');
  const [rows, setRows] = useState<RefundRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const loadRefunds = useCallback(async () => {
    if (!token || !orderId.trim()) return;
    setLoading(true);
    try {
      const res = await fetch(`${API}/v1/order/refunds?order_id=${encodeURIComponent(orderId.trim())}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      const data = (await res.json()) as RefundRecord[];
      setRows(Array.isArray(data) ? data : []);
    } catch (e) {
      toast(e instanceof Error ? e.message : 'Failed to load refunds', 'error');
    } finally {
      setLoading(false);
    }
  }, [orderId, token, toast]);

  const initiateRefund = useCallback(async () => {
    if (!token) return;
    if (!orderId.trim()) {
      toast('Order ID is required', 'error');
      return;
    }
    setSubmitting(true);
    try {
      const parsedAmount = Number(amountUZS || '0');
      const res = await fetch(`${API}/v1/order/refund`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Idempotency-Key': buildRefundIdempotencyKey(orderId, reason, Number.isFinite(parsedAmount) ? parsedAmount : 0),
        },
        body: JSON.stringify({
          order_id: orderId.trim(),
          reason,
          amount_uzs: Number.isFinite(parsedAmount) ? parsedAmount : 0,
        }),
      });
      if (!res.ok) {
        throw new Error(await res.text());
      }
      toast('Refund initiated', 'success');
      await loadRefunds();
    } catch (e) {
      toast(e instanceof Error ? e.message : 'Refund failed', 'error');
    } finally {
      setSubmitting(false);
    }
  }, [amountUZS, loadRefunds, orderId, reason, token, toast]);

  return (
    <div className="flex flex-col gap-6 w-full max-w-7xl mx-auto px-4 py-6">
      <div>
        <h1 className="md-typescale-headline-small" style={{ color: 'var(--color-md-on-surface)' }}>Refunds</h1>
        <p className="md-typescale-body-small mt-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
          Initiate and inspect order-level refunds.
        </p>
      </div>

      <div className="md-card md-elevation-1 md-shape-md p-4 grid grid-cols-1 md:grid-cols-4 gap-3" style={{ background: 'var(--color-md-surface)' }}>
        <input
          value={orderId}
          onChange={(e) => setOrderId(e.target.value)}
          placeholder="Order ID"
          className="md-input-outlined px-3 py-2"
        />
        <input
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Refund reason"
          className="md-input-outlined px-3 py-2"
        />
        <input
          value={amountUZS}
          onChange={(e) => setAmountUZS(e.target.value)}
          placeholder="Amount UZS (0=full)"
          className="md-input-outlined px-3 py-2"
          type="number"
          min="0"
        />
        <div className="flex gap-2">
          <Button variant="outline" onPress={loadRefunds} isDisabled={loading || !orderId.trim()} className="w-full">
            {loading ? 'Loading...' : 'Load'}
          </Button>
          <Button variant="primary" onPress={initiateRefund} isDisabled={submitting || !orderId.trim()} className="w-full">
            {submitting ? 'Submitting...' : 'Refund'}
          </Button>
        </div>
      </div>

      {rows.length === 0 ? (
        <EmptyState
          icon="treasury"
          headline="No refund records"
          body="Load an order to inspect existing refunds or initiate a new one."
        />
      ) : (
        <div className="md-card md-elevation-1 md-shape-md overflow-hidden" style={{ background: 'var(--color-md-surface)' }}>
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                <th className="text-left px-4 py-3">Refund ID</th>
                <th className="text-left px-4 py-3">Status</th>
                <th className="text-right px-4 py-3">Amount</th>
                <th className="text-left px-4 py-3">Gateway</th>
                <th className="text-left px-4 py-3">Provider Ref</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.refund_id} className="border-b last:border-b-0" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                  <td className="px-4 py-3 font-mono text-xs">{r.refund_id}</td>
                  <td className="px-4 py-3">{r.status}</td>
                  <td className="px-4 py-3 text-right tabular-nums">{formatAmount(r.amount_uzs)}</td>
                  <td className="px-4 py-3">{r.gateway}</td>
                  <td className="px-4 py-3 text-xs">{r.provider_refund_id || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
