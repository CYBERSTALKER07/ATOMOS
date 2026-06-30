'use client';

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useParams } from 'next/navigation';
import { apiFetch } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import FactoryPageState from '@/components/FactoryPageState';

interface SupplyRequestDetail {
  request_id: string;
  warehouse_id: string;
  factory_id: string;
  state: string;
  priority: string;
  requested_delivery_date?: string;
  total_volume_vu: number;
  notes?: string;
  items?: Array<{
    item_id: string;
    product_id: string;
    requested_quantity: number;
    unit_volume_vu: number;
  }>;
}

interface QCRecord {
  request_id: string;
  result: string;
  notes?: string;
  inspected_by?: string;
  inspected_at?: string;
}

export default function SupplyRequestDetailPage() {
  const params = useParams();
  const requestId = String(params.id || '');
  const { toast } = useToast();
  const [detail, setDetail] = useState<SupplyRequestDetail | null>(null);
  const [qc, setQc] = useState<QCRecord | null>(null);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [qcNotes, setQcNotes] = useState('');

  const load = useCallback(async () => {
    if (!requestId) return;
    setLoading(true);
    try {
      const [detailRes, qcRes] = await Promise.all([
        apiFetch(`/v1/factory/supply-requests/${requestId}`),
        apiFetch(`/v1/factory/supply-requests/${requestId}/qc`),
      ]);
      if (!detailRes.ok) throw new Error('Supply request not found');
      const detailData = await detailRes.json();
      setDetail(detailData);
      if (qcRes.ok) {
        const qcData = await qcRes.json() as QCRecord;
        setQc(qcData.result ? qcData : null);
        setQcNotes(qcData.notes || '');
      }
    } catch {
      setDetail(null);
    } finally {
      setLoading(false);
    }
  }, [requestId]);

  useEffect(() => { void load(); }, [load]);

  async function submitQC(result: 'PASS' | 'FAIL') {
    setSaving(true);
    try {
      const res = await apiFetch(`/v1/factory/supply-requests/${requestId}/qc`, {
        method: 'POST',
        body: JSON.stringify({ result, notes: qcNotes }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        toast(err.error || 'QC save failed', 'error');
        return;
      }
      const data = await res.json() as QCRecord;
      setQc(data);
      toast(`QC recorded: ${result}`, 'success');
    } catch {
      toast('QC save failed', 'error');
    } finally {
      setSaving(false);
    }
  }

  if (loading) {
    return (
      <PageTransition>
        <div className="p-6">
          <FactoryPageState kind="loading" title="Supply Request" subtitle="Loading request detail." />
        </div>
      </PageTransition>
    );
  }

  if (!detail) {
    return (
      <PageTransition>
        <div className="p-6">
          <FactoryPageState
            kind="error"
            title="Supply Request"
            headline="Request not found"
            body="This supply request may have been fulfilled or removed."
            actionLabel="Back to queue"
            onAction={() => { window.location.href = '/supply-requests'; }}
          />
        </div>
      </PageTransition>
    );
  }

  return (
    <PageTransition>
      <div className="p-6 space-y-6 max-w-3xl">
        <div className="flex items-center gap-3">
          <Link href="/supply-requests" className="text-sm text-(--muted) hover:underline">← Supply Requests</Link>
        </div>

        <header>
          <h1 className="text-xl font-semibold">Supply Request {detail.request_id.slice(0, 8)}</h1>
          <p className="text-sm mt-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
            {detail.state.replace(/_/g, ' ')} · {detail.priority} · {detail.total_volume_vu.toLocaleString()} VU
          </p>
        </header>

        <div className="rounded-xl border border-(--border) p-4 space-y-2" style={{ background: 'var(--surface)' }}>
          <div className="grid grid-cols-2 gap-3 text-sm">
            <div><span className="text-(--muted)">Warehouse</span><div className="font-mono">{detail.warehouse_id.slice(0, 12)}…</div></div>
            <div><span className="text-(--muted)">Delivery</span><div>{detail.requested_delivery_date ? new Date(detail.requested_delivery_date).toLocaleDateString() : '—'}</div></div>
          </div>
          {detail.notes && <p className="text-sm text-(--muted) border-t border-(--border) pt-3">{detail.notes}</p>}
        </div>

        {detail.items && detail.items.length > 0 && (
          <div className="rounded-xl border border-(--border) overflow-hidden">
            <table className="w-full text-sm">
              <thead>
                <tr style={{ background: 'var(--color-md-surface-container)' }}>
                  <th className="text-left px-4 py-2">Product</th>
                  <th className="text-right px-4 py-2">Qty</th>
                  <th className="text-right px-4 py-2">VU</th>
                </tr>
              </thead>
              <tbody>
                {detail.items.map((item) => (
                  <tr key={item.item_id} className="border-t border-(--border)">
                    <td className="px-4 py-2 font-mono text-xs">{item.product_id.slice(0, 10)}…</td>
                    <td className="px-4 py-2 text-right tabular-nums">{item.requested_quantity}</td>
                    <td className="px-4 py-2 text-right tabular-nums">{item.unit_volume_vu}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        <section className="rounded-xl border border-(--border) p-4 space-y-4" style={{ background: 'var(--surface)' }}>
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold flex items-center gap-2">
              <Icon name="verified" size={16} /> Factory QC
            </h2>
            {qc?.result && (
              <span
                className="px-2 py-0.5 rounded text-xs font-light uppercase"
                style={{
                  background: qc.result === 'PASS' ? 'var(--color-md-success)' : 'var(--color-md-error)',
                  color: 'white',
                }}
              >
                {qc.result}
              </span>
            )}
          </div>

          <textarea
            value={qcNotes}
            onChange={(e) => setQcNotes(e.target.value)}
            placeholder="QC notes (optional)"
            rows={3}
            className="w-full rounded-lg border px-3 py-2 text-sm"
            style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)' }}
          />

          <div className="flex gap-2">
            <button
              onClick={() => void submitQC('PASS')}
              disabled={saving}
              className="px-4 py-2 rounded-lg text-sm font-medium text-white disabled:opacity-50"
              style={{ background: 'var(--color-md-success)' }}
            >
              {saving ? 'Saving…' : 'Pass'}
            </button>
            <button
              onClick={() => void submitQC('FAIL')}
              disabled={saving}
              className="px-4 py-2 rounded-lg text-sm font-medium text-white disabled:opacity-50"
              style={{ background: 'var(--color-md-error)' }}
            >
              Fail
            </button>
          </div>

          {qc?.inspected_at && (
            <p className="text-xs text-(--muted)">
              Last inspected {new Date(qc.inspected_at).toLocaleString()}
            </p>
          )}
        </section>

        <Link
          href="/supply-requests/calendar"
          className="inline-flex items-center gap-2 text-sm text-(--muted) hover:underline"
        >
          <Icon name="schedule" size={14} /> View production calendar
        </Link>
      </div>
    </PageTransition>
  );
}
