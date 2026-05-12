'use client';

import { useCallback, useState } from 'react';
import { Button } from '@heroui/react';
import { apiFetch, apiFetchNoQueue } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import EmptyState from '@/components/EmptyState';

interface RefundRecord {
  refund_id: string;
  status: string;
  amount?: number;
  currency?: string;
  amount_uzs: number;
  gateway: string;
  provider_refund_id?: string;
}

function formatAmount(v: number): string {
  return new Intl.NumberFormat('en-US').format(v);
}

function buildRefundIdempotencyKey(orderId: string, reason: string, amount: number, currency: string): string {
  return ['refund', orderId.trim(), reason.trim().toUpperCase(), String(amount), currency.trim().toUpperCase()].join(':');
}

function resolveRefundAmount(record: RefundRecord): number {
  if (typeof record.amount === 'number' && Number.isFinite(record.amount)) {
    return record.amount;
  }
  return record.amount_uzs;
}

function resolveRefundCurrency(record: RefundRecord): string {
  const code = (record.currency || 'UZS').trim().toUpperCase();
  return code || 'UZS';
}

export default function RefundsPage() {
  const { toast } = useToast();

  const [orderId, setOrderId] = useState('');
  const [reason, setReason] = useState('Customer request');
  const [amount, setAmount] = useState<string>('0');
  const [currency, setCurrency] = useState<string>('UZS');
  const [rows, setRows] = useState<RefundRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const loadRefunds = useCallback(async () => {
    if (!orderId.trim()) return;
    setLoading(true);
    try {
      const res = await apiFetch(`/v1/order/refunds?order_id=${encodeURIComponent(orderId.trim())}`);
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
  }, [orderId, toast]);

  const initiateRefund = useCallback(async () => {
    if (!orderId.trim()) {
      toast('Order ID is required', 'error');
      return;
    }
    setSubmitting(true);
    try {
      const parsedAmount = Number(amount || '0');
      const normalizedCurrency = (currency || 'UZS').trim().toUpperCase() || 'UZS';
      const res = await apiFetchNoQueue('/v1/order/refund', {
        method: 'POST',
        headers: {
          'Idempotency-Key': buildRefundIdempotencyKey(orderId, reason, Number.isFinite(parsedAmount) ? parsedAmount : 0, normalizedCurrency),
        },
        body: JSON.stringify({
          order_id: orderId.trim(),
          reason,
          amount: Number.isFinite(parsedAmount) ? parsedAmount : 0,
          currency: normalizedCurrency,
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
  }, [amount, currency, loadRefunds, orderId, reason, toast]);

  return (
    <div className="min-h-full w-full max-w-7xl mx-auto px-4 py-6 flex flex-col gap-6" style={{ background: 'var(--desk-bg)' }}>
      <div>
        <h1 className="md-typescale-headline-small" style={{ color: 'var(--desk-text-primary)' }}>Refunds</h1>
        <p className="md-typescale-body-small mt-1" style={{ color: 'var(--desk-text-secondary)' }}>
          Initiate and inspect order-level refunds.
        </p>
      </div>

      <div className="desk-card p-4 grid grid-cols-1 md:grid-cols-5 gap-3" style={{ background: 'var(--desk-surface)' }}>
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
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
          placeholder="Amount UZS (0=full)"
          className="md-input-outlined px-3 py-2"
          type="number"
          min="0"
        />
        <input
          value={currency}
          onChange={(e) => setCurrency(e.target.value)}
          placeholder="Currency (e.g. UZS)"
          className="md-input-outlined px-3 py-2"
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
        <div className="desk-card overflow-hidden" style={{ background: 'var(--desk-surface)' }}>
          <table className="md-table">
            <thead>
              <tr className="border-b" style={{ borderColor: 'var(--desk-border)' }}>
                <th className="text-left px-4 py-3">Refund ID</th>
                <th className="text-left px-4 py-3">Status</th>
                <th className="text-right px-4 py-3">Amount</th>
                <th className="text-left px-4 py-3">Gateway</th>
                <th className="text-left px-4 py-3">Provider Ref</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.refund_id} className="border-b last:border-b-0" style={{ borderColor: 'var(--desk-border)' }}>
                  <td className="px-4 py-3 font-mono text-xs">{r.refund_id}</td>
                  <td className="px-4 py-3">{r.status}</td>
                  <td className="px-4 py-3 text-right tabular-nums">{formatAmount(resolveRefundAmount(r))} {resolveRefundCurrency(r)}</td>
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
