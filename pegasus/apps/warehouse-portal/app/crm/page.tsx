'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';

interface Retailer {
  retailer_id: string;
  store_name: string;
  total_orders: number;
  total_revenue: number;
  last_order_date: string;
  receiving_window_open?: string;
  receiving_window_close?: string;
}

export default function CRMPage() {
  const [retailers, setRetailers] = useState<Retailer[]>([]);
  const [loading, setLoading] = useState(true);
  const [editing, setEditing] = useState<Retailer | null>(null);
  const [windowOpen, setWindowOpen] = useState('');
  const [windowClose, setWindowClose] = useState('');
  const [saving, setSaving] = useState(false);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/warehouse/ops/crm');
      if (res.ok) {
        const data = await res.json();
        setRetailers(data.retailers || []);
      }
    } catch { /* handled */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

  const openEdit = (r: Retailer) => {
    setEditing(r);
    setWindowOpen(r.receiving_window_open || '');
    setWindowClose(r.receiving_window_close || '');
  };

  const saveWindow = async () => {
    if (!editing) return;
    setSaving(true);
    try {
      const res = await apiFetch(`/v1/warehouse/ops/crm/${editing.retailer_id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          receiving_window_open: windowOpen,
          receiving_window_close: windowClose,
        }),
      });
      if (res.ok) {
        setEditing(null);
        setLoading(true);
        await load();
      }
    } catch { /* handled */ }
    finally { setSaving(false); }
  };

  const windowLabel = (r: Retailer) => {
    if (r.receiving_window_open && r.receiving_window_close) {
      return `${r.receiving_window_open} – ${r.receiving_window_close}`;
    }
    return '—';
  };

  return (
    <div className="p-6 space-y-4 md-animate-in">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <h1 className="text-xl font-bold tracking-tight">Retailer CRM</h1>
        <button onClick={() => { setLoading(true); load(); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
          <Icon name="refresh" size={16} /> Refresh
        </button>
      </div>

      {loading ? (
        <div className="space-y-1">
          {Array.from({ length: 5 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
        </div>
      ) : retailers.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-[var(--muted)]">
          <Icon name="crm" size={48} className="mb-3 opacity-40" />
          <p className="text-sm">No retailer relationships yet</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)]">
                <th className="text-left py-2 px-3 font-medium">Store</th>
                <th className="text-right py-2 px-3 font-medium">Orders</th>
                <th className="text-right py-2 px-3 font-medium">Revenue (UZS)</th>
                <th className="text-right py-2 px-3 font-medium">Last Order</th>
                <th className="text-left py-2 px-3 font-medium">Receiving Window</th>
                <th className="text-right py-2 px-3 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {retailers.map(r => (
                <tr key={r.retailer_id} className="border-b border-[var(--border)] hover:bg-[var(--surface)] transition-colors">
                  <td className="py-2.5 px-3 font-medium">{r.store_name || '—'}</td>
                  <td className="py-2.5 px-3 text-right font-mono">{fmt(r.total_orders)}</td>
                  <td className="py-2.5 px-3 text-right font-mono">{fmt(r.total_revenue)}</td>
                  <td className="py-2.5 px-3 text-right text-[var(--muted)]">
                    {r.last_order_date ? new Date(r.last_order_date).toLocaleDateString() : '—'}
                  </td>
                  <td className="py-2.5 px-3 text-[var(--muted)]">{windowLabel(r)}</td>
                  <td className="py-2.5 px-3 text-right">
                    <button onClick={() => openEdit(r)} className="text-xs button--secondary px-2 py-1 rounded">
                      Edit window
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {editing && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
          <div className="w-full max-w-md rounded-xl border border-[var(--border)] p-4 space-y-3" style={{ background: 'var(--background)' }}>
            <h2 className="text-sm font-semibold">Receiving window — {editing.store_name}</h2>
            <label className="block text-xs text-[var(--muted)]">
              Open (HH:MM)
              <input value={windowOpen} onChange={e => setWindowOpen(e.target.value)} className="mt-1 w-full md-input-outlined px-3 py-2 text-sm" placeholder="09:00" />
            </label>
            <label className="block text-xs text-[var(--muted)]">
              Close (HH:MM)
              <input value={windowClose} onChange={e => setWindowClose(e.target.value)} className="mt-1 w-full md-input-outlined px-3 py-2 text-sm" placeholder="18:00" />
            </label>
            <div className="flex justify-end gap-2 pt-2">
              <button onClick={() => setEditing(null)} className="button--secondary px-3 py-1.5 rounded-lg text-sm">Cancel</button>
              <button onClick={saveWindow} disabled={saving} className="button--primary px-3 py-1.5 rounded-lg text-sm">
                {saving ? 'Saving…' : 'Save'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
