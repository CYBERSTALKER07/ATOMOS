'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback, useRef } from 'react';
import { useStableCallback } from '@/lib/useStableCallback';
import { apiFetch, subscribeWarehouseWS, type WarehouseSocketStatus } from '@/lib/auth';
import { warehouseAdjustInventoryKey, warehouseInventoryPolicyKey } from '@pegasusx/api-client';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import InventoryStockList from '@/components/inventory/InventoryStockList';
import { useToast } from '@/components/Toast';
import { motion } from 'framer-motion';
import type { WarehouseLiveEvent } from '@pegasusx/types';

interface InventoryItem {
  product_id: string;
  product_name: string;
  quantity: number;
  reorder_threshold: number;
  sku: string;
  sku_id?: string;
  out_of_stock_policy?: string;
  effective_policy?: string;
  accepts_backorder?: boolean;
}

export default function InventoryPage() {
  const t = usePortalT();
  const { toast } = useToast();
  const [items, setItems] = useState<InventoryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [lowOnly, setLowOnly] = useState(false);
  const [adjusting, setAdjusting] = useState<string | null>(null);
  const [adjustVal, setAdjustVal] = useState('');
  const [confirmItem, setConfirmItem] = useState<InventoryItem | null>(null);
  const [confirmQty, setConfirmQty] = useState<number | null>(null);
  const [adjustReason, setAdjustReason] = useState('');
  const [adjustSubmitting, setAdjustSubmitting] = useState(false);
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

  const handleWarehouseLiveEvent = useStableCallback((event: WarehouseLiveEvent) => {
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

  async function handlePolicyChange(productId: string, policy: string) {
    const warehouseId = warehouseHomeNodeId() || 'warehouse';
    try {
      const res = await apiFetch(`/v1/warehouse/ops/inventory/${productId}/policy`, {
        method: 'PATCH',
        body: JSON.stringify({ out_of_stock_policy: policy }),
        headers: {
          'Idempotency-Key': warehouseInventoryPolicyKey(warehouseId, productId, policy),
        },
      });
      if (res.ok) {
        toast('Stock policy updated', 'success');
        void load({ silent: true });
      }
    } catch { /* handled */ }
  }

  function openAdjustConfirm(item: InventoryItem) {
    const qty = parseInt(adjustVal, 10);
    if (Number.isNaN(qty)) return;
    setConfirmItem(item);
    setConfirmQty(qty);
    setAdjustReason('');
  }

  function closeAdjustFlow() {
    setAdjusting(null);
    setAdjustVal('');
    setConfirmItem(null);
    setConfirmQty(null);
    setAdjustReason('');
    setAdjustSubmitting(false);
  }

  async function handleAdjustConfirm() {
    if (!confirmItem || confirmQty == null) return;
    const warehouseId = warehouseHomeNodeId() || 'warehouse';
    setAdjustSubmitting(true);
    try {
      const body: { product_id: string; quantity: number; reason?: string } = {
        product_id: confirmItem.product_id,
        quantity: confirmQty,
      };
      const trimmedReason = adjustReason.trim();
      if (trimmedReason) body.reason = trimmedReason;

      const res = await apiFetch('/v1/warehouse/ops/inventory', {
        method: 'PATCH',
        body: JSON.stringify(body),
        headers: {
          'Idempotency-Key': warehouseAdjustInventoryKey(warehouseId, confirmItem.product_id, confirmQty),
        },
      });
      if (res.ok) {
        toast('Inventory updated', 'success');
        closeAdjustFlow();
        void load();
      } else {
        const data = await res.json().catch(() => ({} as { error?: string }));
        toast(data.error || 'Update failed', 'error');
      }
    } catch {
      toast('Update failed', 'error');
    } finally {
      setAdjustSubmitting(false);
    }
  }

  return (
    <PageTransition>
      <PageChrome
        icon="inventory"
        title={t("portal.nav.inventory")}
        description={t("warehouse_portal.residual.text.on_hand_stock_levels_with_live_sync_and_quantity_adjustments")}
        error={loadError}
        actions={
          <div className="flex gap-2 items-center">
            <input
              placeholder={t("warehouse_portal.inventory.text.search_products")}
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
        }
      >
        <div className="space-y-4">
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

        {loading ? (
          <div className="space-y-1">
            {Array.from({ length: 6 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
          </div>
        ) : (
          <InventoryStockList
            items={items}
            loading={false}
            search={search}
            lowOnly={lowOnly}
            adjusting={adjusting}
            adjustVal={adjustVal}
            pulsedProductIds={pulsedProductIds}
            onAdjustValChange={setAdjustVal}
            onStartAdjust={(item) => { setAdjusting(item.product_id); setAdjustVal(String(item.quantity)); }}
            onReviewAdjust={(item) => openAdjustConfirm(item)}
            onCancelAdjust={() => closeAdjustFlow()}
            onPolicyChange={(productId, policy) => { void handlePolicyChange(productId, policy); }}
          />
        )}
        </div>

        {confirmItem && confirmQty != null && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
            <div
              className="w-full max-w-md rounded-xl border p-5 space-y-4 shadow-lg"
              style={{ background: 'var(--surface)', borderColor: 'var(--border)' }}
            >
              <h3 className="text-sm font-semibold">{t("warehouse_portal.inventory.text.confirm_inventory_change")}</h3>
              <p className="text-sm text-[var(--muted)]">
                Change {confirmItem.sku || confirmItem.sku_id || confirmItem.product_id} from {confirmItem.quantity} to {confirmQty}?
                This affects retailer availability immediately.
              </p>
              <label className="block text-sm space-y-1">
                <span className="text-xs text-[var(--muted)]">{t("warehouse_portal.inventory.text.reason_optional")}</span>
                <input
                  type="text"
                  value={adjustReason}
                  onChange={(e) => setAdjustReason(e.target.value)}
                  placeholder={t("warehouse_portal.inventory.text.e_g_cycle_count_damaged_goods")}
                  className="w-full px-3 py-2 rounded-lg border text-sm"
                  style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)' }}
                />
              </label>
              <div className="flex justify-end gap-2">
                <button
                  type="button"
                  disabled={adjustSubmitting}
                  onClick={() => closeAdjustFlow()}
                  className="px-3 py-1.5 rounded-lg text-sm button--secondary"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  disabled={adjustSubmitting}
                  onClick={() => void handleAdjustConfirm()}
                  className="px-3 py-1.5 rounded-lg text-sm button--primary disabled:opacity-50"
                >
                  {adjustSubmitting ? 'Saving…' : 'Confirm change'}
                </button>
              </div>
            </div>
          </div>
        )}
      </PageChrome>
    </PageTransition>
  );
}
