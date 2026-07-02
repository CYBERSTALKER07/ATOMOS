'use client';

import { useCallback, useEffect, useState } from 'react';
import { warehouseInboundConfirmKey, warehouseInboundScanKey } from '@pegasusx/api-client';
import { apiFetch } from '@/lib/auth';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';

type InboundRow = {
  return_id: string;
  order_id: string;
  product_name: string;
  expected_qty: number;
  received_qty: number;
  reason: string;
  physical_status: string;
  driver_name?: string;
  suggested_disposition: string;
  barcode?: string;
};

export default function ReturnsPage() {
  const [rows, setRows] = useState<InboundRow[]>([]);
  const [history, setHistory] = useState<InboundRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState<'inbound' | 'history'>('inbound');
  const [barcode, setBarcode] = useState('');
  const [sessionId, setSessionId] = useState<string | null>(null);
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [error, setError] = useState<string | null>(null);

  const loadInbound = useCallback(async () => {
    const res = await apiFetch('/v1/returns/inbound?physical_status=ARRIVED&limit=100');
    if (!res.ok) throw new Error('load_inbound_failed');
    const data = await res.json();
    setRows(data.data ?? []);
  }, []);

  const loadHistory = useCallback(async () => {
    const res = await apiFetch('/v1/returns/history?limit=50');
    if (!res.ok) throw new Error('load_history_failed');
    const data = await res.json();
    setHistory(data.data ?? []);
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      if (tab === 'inbound') await loadInbound();
      else await loadHistory();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'load_failed');
    } finally {
      setLoading(false);
    }
  }, [tab, loadInbound, loadHistory]);

  useEffect(() => { void load(); }, [load]);

  useWarehouseSessionReconcile(() => {
    void load();
  });

  async function ensureSession() {
    if (sessionId) return sessionId;
    const res = await apiFetch('/v1/returns/inbound/sessions', { method: 'POST', body: '{}' });
    if (!res.ok) throw new Error('session_failed');
    const data = await res.json();
    setSessionId(data.session_id);
    return data.session_id as string;
  }

  async function handleScan() {
    if (!barcode.trim()) return;
    try {
      const sid = await ensureSession();
      const trimmed = barcode.trim();
      const warehouseId = warehouseHomeNodeId() || 'warehouse';
      const res = await apiFetch('/v1/returns/inbound/scan', {
        method: 'POST',
        headers: {
          'Idempotency-Key': warehouseInboundScanKey(warehouseId, trimmed, sid),
        },
        body: JSON.stringify({ barcode: trimmed, qty: 1, session_id: sid }),
      });
      const body = await res.json();
      if (!res.ok) throw new Error(body.error || body.message || 'scan_failed');
      setBarcode('');
      await loadInbound();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'scan_failed');
    }
  }

  async function handleConfirm(disposition: 'RESTOCK' | 'WRITE_OFF') {
    const targets = rows.filter(r => selected.has(r.return_id));
    if (targets.length === 0) {
      setError('Select at least one line');
      return;
    }
    try {
      const sid = await ensureSession();
      const warehouseId = warehouseHomeNodeId() || 'warehouse';
      const returnIds = targets.map(r => r.return_id);
      const res = await apiFetch('/v1/returns/inbound/confirm', {
        method: 'POST',
        headers: {
          'Idempotency-Key': warehouseInboundConfirmKey(warehouseId, returnIds, disposition),
        },
        body: JSON.stringify({
          session_id: sid,
          lines: targets.map(r => ({
            return_id: r.return_id,
            disposition,
            qty: r.received_qty > 0 ? r.received_qty : r.expected_qty,
          })),
        }),
      });
      if (!res.ok) {
        const body = await res.json();
        throw new Error(body.error || 'confirm_failed');
      }
      setSelected(new Set());
      await loadInbound();
    } catch (e) {
      setError(e instanceof Error ? e.message : 'confirm_failed');
    }
  }

  const list = tab === 'inbound' ? rows : history;

  return (
    <PageTransition>
      <PageChrome
        icon="returns"
        title="Inbound Returns"
        description="Scan driver-returned goods at the warehouse gate — restock or write off."
        actions={
          <button type="button" onClick={() => void load()} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
            <Icon name="refresh" size={16} /> Refresh
          </button>
        }
      >
        <div className="flex gap-2 mb-4">
          <button type="button" className={`px-3 py-1.5 rounded-lg text-sm ${tab === 'inbound' ? 'button--primary' : 'button--secondary'}`} onClick={() => setTab('inbound')}>Gate queue</button>
          <button type="button" className={`px-3 py-1.5 rounded-lg text-sm ${tab === 'history' ? 'button--primary' : 'button--secondary'}`} onClick={() => setTab('history')}>History</button>
        </div>

        {tab === 'inbound' && (
          <div className="mb-4 flex flex-col gap-2 md:flex-row md:items-end">
            {/* USB/BT wedge scanners type into the field and send Enter — no camera lib needed. */}
            <label className="flex-1 text-sm">
              <span className="text-[var(--muted)]">Barcode (EAN)</span>
              <input
                autoFocus
                className="mt-1 w-full rounded-lg border border-[var(--border)] bg-[var(--surface)] px-3 py-2"
                value={barcode}
                onChange={e => setBarcode(e.target.value)}
                onKeyDown={e => {
                  if (e.key === 'Enter') {
                    e.preventDefault();
                    void handleScan();
                  }
                }}
                placeholder="Scan or type (wedge scanner + Enter)"
              />
            </label>
            <button type="button" className="button--primary px-4 py-2 rounded-lg text-sm" onClick={() => void handleScan()}>Scan</button>
            <button type="button" className="button--primary px-4 py-2 rounded-lg text-sm bg-emerald-600" onClick={() => void handleConfirm('RESTOCK')}>Restock selected</button>
            <button type="button" className="button--secondary px-4 py-2 rounded-lg text-sm" onClick={() => void handleConfirm('WRITE_OFF')}>Write off</button>
          </div>
        )}

        {error && <p className="text-sm text-red-500 mb-3">{error}</p>}

        {loading ? (
          <div className="space-y-1">{Array.from({ length: 5 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}</div>
        ) : list.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-[var(--muted)]">
            <Icon name="returns" size={48} className="mb-3 opacity-40" />
            <p className="text-sm">{tab === 'inbound' ? 'No trucks awaiting receive' : 'No completed receives yet'}</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="desk-table w-full text-sm">
              <thead>
                <tr className="border-b border-[var(--border)]">
                  {tab === 'inbound' && <th className="w-8" />}
                  <th className="text-left py-2 px-3">Product</th>
                  <th className="text-left py-2 px-3">EAN</th>
                  <th className="text-left py-2 px-3">Driver</th>
                  <th className="text-right py-2 px-3">Qty</th>
                  <th className="text-left py-2 px-3">Reason</th>
                  <th className="text-left py-2 px-3">Status</th>
                </tr>
              </thead>
              <tbody>
                {list.map(item => (
                  <tr key={item.return_id} className="border-b border-[var(--border)]">
                    {tab === 'inbound' && (
                      <td className="py-2 px-2">
                        <input type="checkbox" checked={selected.has(item.return_id)} onChange={() => setSelected(prev => { const n = new Set(prev); if (n.has(item.return_id)) n.delete(item.return_id); else n.add(item.return_id); return n; })} />
                      </td>
                    )}
                    <td className="py-2.5 px-3 font-medium">{item.product_name}</td>
                    <td className="py-2.5 px-3 font-mono text-xs text-[var(--muted)]">{item.barcode || '—'}</td>
                    <td className="py-2.5 px-3 text-[var(--muted)]">{item.driver_name || '—'}</td>
                    <td className="py-2.5 px-3 text-right font-mono">{item.received_qty}/{item.expected_qty}</td>
                    <td className="py-2.5 px-3">{item.reason}</td>
                    <td className="py-2.5 px-3"><span className="status-chip">{item.physical_status}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </PageChrome>
    </PageTransition>
  );
}
