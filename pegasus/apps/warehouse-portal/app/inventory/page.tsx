'use client';

import { useEffect, useEffectEvent, useState, useCallback, useRef } from 'react';
import { apiFetch, subscribeWarehouseWS, type WarehouseSocketStatus } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';
import { motion } from 'framer-motion';
import type { WarehouseLiveEvent } from '@pegasus/types';

interface InventoryItem {
  product_id: string;
  product_name: string;
  quantity: number;
  reorder_threshold: number;
  sku: string;
}

export default function InventoryPage() {
  const { toast } = useToast();
  const [items, setItems] = useState<InventoryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [lowOnly, setLowOnly] = useState(false);
  const [adjusting, setAdjusting] = useState<string | null>(null);
  const [adjustVal, setAdjustVal] = useState('');
  const [socketStatus, setSocketStatus] = useState<WarehouseSocketStatus>('connecting');
  const [loadError, setLoadError] = useState<string | null>(null);
  const [pulsedProductIds, setPulsedProductIds] = useState<string[]>([]);

  const inventorySnapshotRef = useRef<InventoryItem[]>([]);
  const pendingSyncEventRef = useRef<Extract<WarehouseLiveEvent, { type: 'INVENTORY_SYNC_COMPLETE' }> | null>(null);
  const pulseTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const pulseRows = useCallback((ids: string[]) => {
    const normalized = Array.from(new Set(ids.map((id) => String(id).trim()).filter(Boolean)));
    if (pulseTimerRef.current) {
      clearTimeout(pulseTimerRef.current);
      pulseTimerRef.current = null;
    }
    setPulsedProductIds(normalized);
    if (normalized.length > 0) {
      pulseTimerRef.current = setTimeout(() => {
        setPulsedProductIds([]);
        pulseTimerRef.current = null;
      }, 3000);
    }
  }, []);

  const load = useCallback(async (options?: { silent?: boolean }) => {
    if (!options?.silent) {
      setLoading(true);
    }
    setLoadError(null);
    try {
      const params = new URLSearchParams();
      if (search) params.set('search', search);
      if (lowOnly) params.set('low_stock', 'true');
      const q = params.toString() ? `?${params}` : '';
      const res = await apiFetch(`/v1/warehouse/ops/inventory${q}`);
      if (res.ok) {
        const data = await res.json();
        const nextItems = data.items || [];
        const previousItems = inventorySnapshotRef.current;
        setItems(nextItems);
        inventorySnapshotRef.current = nextItems;

        const pendingSync = pendingSyncEventRef.current;
        if (pendingSync && pendingSync.type === 'INVENTORY_SYNC_COMPLETE') {
          pendingSyncEventRef.current = null;

          let highlighted = Array.isArray(pendingSync.product_ids)
            ? pendingSync.product_ids.map((id) => String(id).trim()).filter(Boolean)
            : [];

          if (highlighted.length === 0) {
            const previousById = new Map(previousItems.map((item) => [item.product_id, item.quantity]));
            highlighted = nextItems
              .filter((item: InventoryItem) => previousById.get(item.product_id) !== item.quantity)
              .map((item: InventoryItem) => item.product_id);
          }

          pulseRows(highlighted);

          const rows = Number(pendingSync.rows_affected || highlighted.length || 0);
          const session = typeof pendingSync.session_id === 'string' ? pendingSync.session_id.slice(0, 8) : '';
          if (rows > 0) {
            toast(`Inventory sync complete: ${rows} rows applied${session ? ` (session ${session})` : ''}.`, 'success');
          } else {
            toast(`Inventory sync complete${session ? ` (session ${session})` : ''}.`, 'success');
          }
        }
      } else {
        const data = await res.json().catch(() => ({} as { error?: string }));
        setLoadError(data.error || 'Failed to load inventory');
      }
    } catch {
      setLoadError('Failed to load inventory');
    } finally {
      if (!options?.silent) {
        setLoading(false);
      }
    }
  }, [search, lowOnly, pulseRows, toast]);

  const handleWarehouseLiveEvent = useEffectEvent((event: WarehouseLiveEvent) => {
    if (event.type !== 'INVENTORY_SYNC_COMPLETE') {
      return;
    }
    pendingSyncEventRef.current = event;
    void load({ silent: true });
  });

  useEffect(() => { void load(); }, [load]);

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

  useEffect(() => {
    return () => {
      if (pulseTimerRef.current) {
        clearTimeout(pulseTimerRef.current);
      }
    };
  }, []);

  async function handleAdjust(productId: string) {
    const qty = parseInt(adjustVal, 10);
    if (isNaN(qty)) return;
    try {
      const res = await apiFetch('/v1/warehouse/ops/inventory', {
        method: 'PATCH',
        body: JSON.stringify({ product_id: productId, quantity: qty }),
      });
      if (res.ok) {
        setAdjusting(null);
        setAdjustVal('');
        void load();
      }
    } catch { /* handled */ }
  }

  return (
    <PageTransition>
      <div className="p-6 space-y-4">
        <div className="flex items-center justify-between flex-wrap gap-3">
          <h1 className="text-xl font-bold tracking-tight text-[var(--foreground)]">Inventory</h1>
          <div className="flex gap-2 items-center">
            <input
              placeholder="Search products..."
              value={search}
              onChange={e => { setSearch(e.target.value); setLoading(true); }}
              className="px-3 py-1.5 rounded-lg border text-sm w-48 focus:ring-2 focus:ring-[var(--primary)] outline-none"
              style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
            />
            <label className="flex items-center gap-1.5 text-sm text-[var(--muted)] cursor-pointer hover:text-[var(--foreground)] transition-colors">
              <input type="checkbox" checked={lowOnly} onChange={e => { setLowOnly(e.target.checked); setLoading(true); }} className="rounded accent-[var(--primary)]" />
              Low stock only
            </label>
            <motion.button 
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              onClick={() => { void load(); }} 
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary active-press"
            >
              <Icon name="refresh" size={16} />
            </motion.button>
          </div>
        </div>

        {socketStatus !== 'idle' && socketStatus !== 'live' && (
          <motion.div
            initial={{ opacity: 0, y: -10 }}
            animate={{ opacity: 1, y: 0 }}
            className={`rounded-xl border px-4 py-3 text-sm shadow-sm ${socketStatus === 'offline'
              ? 'border-[var(--danger)]/30 bg-[var(--danger)]/8 text-[var(--danger)]'
              : 'border-[var(--warning)]/30 bg-[var(--warning)]/8 text-[var(--warning)]'}`}
          >
            <div className="flex items-center gap-2">
              <div className={`w-2 h-2 rounded-full animate-pulse ${socketStatus === 'offline' ? 'bg-[var(--danger)]' : 'bg-[var(--warning)]'}`} />
              {socketStatus === 'offline'
                ? 'Offline. Live inventory sync is paused.'
                : socketStatus === 'reconnecting'
                  ? 'Reconnecting live inventory sync...'
                  : 'Connecting live inventory sync...'}
            </div>
          </motion.div>
        )}

        {loadError && (
          <div className="rounded-xl border border-[var(--warning)]/30 bg-[var(--warning)]/8 p-4 text-sm text-[var(--warning)]">
            {loadError}
          </div>
        )}

        {loading ? (
          <div className="space-y-1">
            {Array.from({ length: 6 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
          </div>
        ) : items.length === 0 ? (
          <EmptyState
            variant={search || lowOnly ? 'no-results' : 'no-data'}
            headline="No inventory items found"
            body={search || lowOnly ? "Try adjusting your search filters to find what you're looking for." : "There are no products in your inventory yet."}
          />
        ) : (
          <motion.div 
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            className="overflow-x-auto rounded-xl border border-[var(--border)] bg-[var(--surface)]"
          >
            <table className="w-full text-sm">
              <thead>
                <tr className="table__header border-b border-[var(--border)] bg-[var(--default)]">
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Product</th>
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">SKU</th>
                  <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Quantity</th>
                  <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Reorder At</th>
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Status</th>
                  <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Action</th>
                </tr>
              </thead>
              <tbody>
                {items.map((item, index) => {
                  const isLow = item.quantity <= item.reorder_threshold;
                  return (
                    <motion.tr 
                      key={item.product_id} 
                      initial={{ opacity: 0, x: -10 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ delay: index * 0.03 }}
                      className={`table__row border-b border-[var(--border)] last:border-0 hover:bg-[var(--default)]/50 transition-colors ${pulsedProductIds.includes(item.product_id) ? 'warehouse-inventory-sync-pulse' : ''}`}
                    >
                      <td className="py-3 px-4 font-medium">{item.product_name}</td>
                      <td className="py-3 px-4 font-mono text-xs text-[var(--muted)]">{item.sku || '—'}</td>
                      <td className="py-3 px-4 text-right font-mono tabular-nums">{item.quantity}</td>
                      <td className="py-3 px-4 text-right font-mono text-[var(--muted)] tabular-nums">{item.reorder_threshold}</td>
                      <td className="py-3 px-4">
                        {isLow ? (
                          <span className="status-chip status-chip--critical">LOW</span>
                        ) : (
                          <span className="status-chip status-chip--stable">OK</span>
                        )}
                      </td>
                      <td className="py-3 px-4 text-right">
                        {adjusting === item.product_id ? (
                          <div className="flex items-center gap-1 justify-end">
                            <input
                              type="number"
                              value={adjustVal}
                              onChange={e => setAdjustVal(e.target.value)}
                              placeholder="New qty"
                              className="w-20 px-2 py-1 rounded border text-xs outline-none focus:ring-1 focus:ring-[var(--primary)]"
                              style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)' }}
                            />
                            <motion.button 
                              whileHover={{ scale: 1.1 }}
                              whileTap={{ scale: 0.9 }}
                              onClick={() => handleAdjust(item.product_id)} 
                              className="px-2 py-1 text-xs button--primary rounded active-press"
                            >
                              Set
                            </motion.button>
                            <motion.button 
                              whileHover={{ scale: 1.1 }}
                              whileTap={{ scale: 0.9 }}
                              onClick={() => setAdjusting(null)} 
                              className="px-2 py-1 text-xs button--secondary rounded active-press"
                            >
                              X
                            </motion.button>
                          </div>
                        ) : (
                          <motion.button 
                            whileHover={{ scale: 1.05, x: -2 }}
                            onClick={() => { setAdjusting(item.product_id); setAdjustVal(String(item.quantity)); }} 
                            className="text-xs text-[var(--link)] font-medium hover:underline flex items-center gap-1 ml-auto"
                          >
                            <Icon name="refresh" size={12} /> Adjust
                          </motion.button>
                        )}
                      </td>
                    </motion.tr>
                  );
                })}
              </tbody>
            </table>
          </motion.div>
        )}
      </div>
    </PageTransition>
  );
}
