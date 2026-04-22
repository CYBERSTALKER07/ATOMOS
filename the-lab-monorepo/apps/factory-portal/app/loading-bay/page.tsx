'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';

type TransferState = 'APPROVED' | 'LOADING' | 'DISPATCHED';

interface Transfer {
  id: string;
  warehouse_name: string;
  total_items: number;
  total_volume_m3: number;
  state: string;
  created_at: string;
  updated_at: string;
}

const COLUMNS: { key: TransferState; label: string; css: string }[] = [
  { key: 'APPROVED', label: 'Ready for Loading', css: 'status-chip--approved' },
  { key: 'LOADING', label: 'Now Loading', css: 'status-chip--loading' },
  { key: 'DISPATCHED', label: 'Dispatched', css: 'status-chip--dispatched' },
];

export default function LoadingBayPage() {
  const { toast } = useToast();
  const [transfers, setTransfers] = useState<Transfer[]>([]);
  const [loading, setLoading] = useState(true);
  const [dispatching, setDispatching] = useState(false);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/factory/transfers?states=APPROVED,LOADING,DISPATCHED');
      if (res.ok) {
        const data = await res.json();
        setTransfers(data.transfers || []);
      }
    } catch {
      // handled
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const grouped = COLUMNS.map(col => ({
    ...col,
    items: transfers.filter(t => t.state === col.key),
  }));

  async function handleDispatch() {
    setDispatching(true);
    try {
      const res = await apiFetch('/v1/factory/dispatch', { method: 'POST' });
      if (res.ok) {
        const data = await res.json();
        toast(`Dispatched ${data.manifests_created || 0} manifest(s)`, 'success');
        load();
      } else {
        const err = await res.json().catch(() => ({}));
        toast(err.error || 'Dispatch failed', 'error');
      }
    } catch {
      toast('Dispatch request failed', 'error');
    } finally {
      setDispatching(false);
    }
  }

  if (loading) {
    return (
      <div className="p-6">
        <div className="md-skeleton md-skeleton-title" />
        <div className="flex gap-4 mt-4">
          {[1, 2, 3].map(i => (
            <div key={i} className="kanban-col">
              {[1, 2].map(j => <div key={j} className="md-skeleton md-skeleton-card" />)}
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="p-6 space-y-4 md-animate-in">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold tracking-tight">Loading Bay</h1>
        <div className="flex items-center gap-3">
          <button
            onClick={() => load()}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary"
          >
            <Icon name="refresh" size={16} /> Refresh
          </button>
          <button
            onClick={handleDispatch}
            disabled={dispatching}
            className="flex items-center gap-1.5 px-4 py-1.5 rounded-lg text-sm font-semibold button--primary disabled:opacity-50"
          >
            {dispatching ? 'Dispatching...' : 'Batch Dispatch'}
          </button>
        </div>
      </div>

      {transfers.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-[var(--muted)]">
          <Icon name="loadingBay" size={48} className="mb-3 opacity-40" />
          <p className="text-sm">No active transfers in the loading bay</p>
        </div>
      ) : (
        <div className="flex gap-4 overflow-x-auto pb-4">
          {grouped.map(col => (
            <div key={col.key} className="kanban-col">
              <div className="flex items-center justify-between mb-1">
                <span className={`status-chip ${col.css} text-[10px]`}>{col.label}</span>
                <span className="text-xs text-[var(--muted)]">{col.items.length}</span>
              </div>
              {col.items.map(t => (
                <a key={t.id} href={`/transfers/${t.id}`} className="kanban-card block">
                  <div className="text-sm font-semibold truncate">{t.warehouse_name}</div>
                  <div className="flex items-center gap-3 mt-1 text-xs text-[var(--muted)]">
                    <span>{t.total_items} items</span>
                    <span>{t.total_volume_m3.toFixed(1)} m³</span>
                  </div>
                  <div className="text-[10px] text-[var(--muted)] mt-1">
                    {new Date(t.created_at).toLocaleDateString()}
                  </div>
                </a>
              ))}
              {col.items.length === 0 && (
                <div className="text-xs text-[var(--muted)] text-center py-8">Empty</div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
