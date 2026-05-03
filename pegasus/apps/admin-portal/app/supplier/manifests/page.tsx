'use client';

import { useState, useCallback, useEffect } from 'react';
import { useToken } from '@/lib/auth';
import EmptyState from '@/components/EmptyState';
import { Skeleton } from '@/components/Skeleton';
import Icon from '@/components/Icon';
import { useToast } from '@/components/Toast';
import { Button } from '@heroui/react';
import {
  buildSupplierManifestInjectOrderIdempotencyKey,
  buildSupplierManifestSealIdempotencyKey,
} from '../_shared/idempotency';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

/* ─── Types ───────────────────────────────────────────────── */

interface ManifestLine {
  sku_id: string;
  product_name: string;
  total_qty: number;
  order_count: number;
}

interface ManifestOrder {
  order_id: string;
  retailer_name: string;
  item_count: number;
  state: string;
  created_at: string;
}

interface ManifestEntity {
  manifest_id: string;
  driver_id: string;
  truck_id: string;
  state: string;
  stop_count: number;
  total_volume_vu: number;
  max_volume_vu: number;
  warehouse_id: string;
  region_code: string;
  sealed_at: string;
  dispatched_at: string;
  created_at: string;
}

type Tab = 'pick-list' | 'orders' | 'manifest-ops';

function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}

function formatAmount(amount: number): string {
  return new Intl.NumberFormat('en-US').format(amount);
}

function shortId(id: string): string {
  return id.length > 12 ? id.slice(0, 12) + '…' : id;
}

/* ─── Main Page ───────────────────────────────────────────── */

export default function ManifestsPage() {
  const token = useToken();
  const { toast } = useToast();

  const [date, setDate] = useState(todayISO);
  const [tab, setTab] = useState<Tab>('pick-list');
  const [lines, setLines] = useState<ManifestLine[]>([]);
  const [orders, setOrders] = useState<ManifestOrder[]>([]);
  const [loading, setLoading] = useState(true);
  const [exporting, setExporting] = useState(false);
  const [manifests, setManifests] = useState<ManifestEntity[]>([]);
  const [injectModalManifest, setInjectModalManifest] = useState<string | null>(null);
  const [injectOrderId, setInjectOrderId] = useState('');
  const [isInjecting, setIsInjecting] = useState(false);
  const [forceSealingId, setForceSealingId] = useState<string | null>(null);

  /* ─── Fetch ─────────────────────────────────────────────── */

  const fetchData = useCallback(async () => {
    if (!token) return;
      setLoading(true);
      try {
        const [linesRes, ordersRes, manifestsRes] = await Promise.all([
        fetch(`${API}/v1/supplier/picking-manifests?date=${date}`, {
          headers: { Authorization: `Bearer ${token}` },
        }),
        fetch(`${API}/v1/supplier/picking-manifests/orders?date=${date}`, {
          headers: { Authorization: `Bearer ${token}` },
        }),
        fetch(`${API}/v1/supplier/manifests`, {
          headers: { Authorization: `Bearer ${token}` },
        }),
      ]);
      if (linesRes.ok) {
        const j = await linesRes.json();
        setLines(j.lines || []);
      }
      if (ordersRes.ok) {
        const j = await ordersRes.json();
        setOrders(j.data || j.orders || []);
      }
      if (manifestsRes.ok) {
        const j = await manifestsRes.json();
        setManifests(j.manifests || []);
      }
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setLoading(false);
    }
  }, [token, date, toast]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  /* ─── CSV Export ────────────────────────────────────────── */

  const exportCSV = useCallback(async () => {
    if (!token) return;
    setExporting(true);
    try {
      const res = await fetch(`${API}/v1/supplier/picking-manifests?date=${date}&format=csv`, {
        headers: { Authorization: `Bearer ${token}`, Accept: 'text/csv' },
      });
      if (!res.ok) throw new Error('CSV export failed');
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `manifest-${date}.csv`;
      a.click();
      URL.revokeObjectURL(url);
      toast('CSV downloaded', 'success');
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setExporting(false);
    }
  }, [token, date, toast]);

  /* ─── Computed ──────────────────────────────────────────── */

  const totalQty = lines.reduce((a, l) => a + l.total_qty, 0);
  const totalOrders = orders.length;
  const totalSKUs = lines.length;
  const loadingManifests = manifests.filter(m => m.state === 'LOADING');

  /* ─── Inject Order into LOADING Manifest ────────────────── */

  const handleInjectOrder = useCallback(async () => {
    if (!token || !injectModalManifest || !injectOrderId.trim()) return;
    setIsInjecting(true);
    try {
      const res = await fetch(`${API}/v1/supplier/manifests/${injectModalManifest}/inject-order`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Idempotency-Key': buildSupplierManifestInjectOrderIdempotencyKey(injectModalManifest, injectOrderId.trim()),
        },
        body: JSON.stringify({ order_id: injectOrderId.trim() }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
        throw new Error(err.error || `HTTP ${res.status}`);
      }
      toast('Order injected into manifest', 'success');
      setInjectModalManifest(null);
      setInjectOrderId('');
      fetchData();
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setIsInjecting(false);
    }
  }, [token, injectModalManifest, injectOrderId, toast, fetchData]);

  /* ─── Force-Seal LOADING Manifest ───────────────────────── */

  const handleForceSeal = useCallback(async (manifestId: string) => {
    if (!token) return;
    const reason = prompt('Force-seal reason (audit trail):');
    if (!reason) return;
    setForceSealingId(manifestId);
    try {
      const res = await fetch(
        `${API}/v1/supplier/manifests/${manifestId}/seal?override=admin&reason=${encodeURIComponent(reason)}`,
        {
          method: 'POST',
          headers: {
            Authorization: `Bearer ${token}`,
            'Content-Type': 'application/json',
            'Idempotency-Key': buildSupplierManifestSealIdempotencyKey(manifestId, reason),
          },
        }
      );
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: `HTTP ${res.status}` }));
        throw new Error(err.error || `HTTP ${res.status}`);
      }
      toast('Manifest force-sealed', 'success');
      fetchData();
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setForceSealingId(null);
    }
  }, [token, toast, fetchData]);

  /* ─── Render ────────────────────────────────────────────── */

  return (
    <div className="flex flex-col gap-6 w-full max-w-7xl mx-auto px-4 py-6">
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-4">
        <div>
          <h1 className="md-typescale-headline-small" style={{ color: 'var(--color-md-on-surface)' }}>
            Warehouse Manifests
          </h1>
          <p className="md-typescale-body-small mt-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
            Daily picking lists and order manifests for warehouse operations.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <input
            type="date"
            value={date}
            onChange={(e) => setDate(e.target.value)}
            className="md-input-outlined px-3 py-2 md-typescale-body-medium md-shape-sm"
            style={{
              background: 'var(--color-md-surface)',
              color: 'var(--color-md-on-surface)',
              borderColor: 'var(--color-md-outline)',
            }}
          />
          <Button
            variant="outline"
            isDisabled={exporting}
            onPress={exportCSV}
            className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2"
          >
            <Icon name="orders" size={16} className="mr-1.5" />
            {exporting ? 'Exporting…' : 'Export CSV'}
          </Button>
        </div>
      </div>

      {/* Summary Cards */}
      {!loading && (
        <div className="grid grid-cols-3 gap-4">
          {[
            { label: 'SKUs', value: totalSKUs },
            { label: 'Total Units', value: totalQty },
            { label: 'Orders', value: totalOrders },
          ].map((s) => (
            <div
              key={s.label}
              className="md-card md-elevation-0 md-shape-md p-4"
              style={{ background: 'var(--color-md-surface-container)' }}
            >
              <span className="md-typescale-label-small block" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                {s.label}
              </span>
              <span className="md-typescale-headline-small tabular-nums" style={{ color: 'var(--color-md-on-surface)' }}>
                {s.value.toLocaleString()}
              </span>
            </div>
          ))}
        </div>
      )}

      {/* Tabs */}
      <div className="flex gap-1 p-1 rounded-xl" style={{ background: 'var(--color-md-surface-container)' }}>
        {([
          { key: 'pick-list' as Tab, label: 'Pick List', icon: 'manifests' },
          { key: 'orders' as Tab, label: 'Orders', icon: 'orders' },
          { key: 'manifest-ops' as Tab, label: 'Manifest Ops', icon: 'fleet' },
        ] as const).map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`flex-1 flex items-center justify-center gap-2 px-4 py-2.5 rounded-lg md-typescale-label-large transition-all ${
              tab === t.key ? 'md-elevation-1' : ''
            }`}
            style={{
              background: tab === t.key ? 'var(--color-md-surface)' : 'transparent',
              color: tab === t.key ? 'var(--color-md-on-surface)' : 'var(--color-md-on-surface-variant)',
            }}
          >
            <Icon name={t.icon} size={16} />
            {t.label}
          </button>
        ))}
      </div>

      {/* Loading */}
      {loading && (
        <div className="flex flex-col gap-3">
          {[1, 2, 3, 4, 5].map((i) => (
            <Skeleton key={i} className="h-12 w-full rounded-lg" />
          ))}
        </div>
      )}

      {/* Pick List Tab */}
      {!loading && tab === 'pick-list' && (
        <>
          {lines.length === 0 ? (
            <EmptyState
              icon="manifests"
              headline="No manifest lines"
              body={`No picking data for ${date}. Orders must be in LOADED state or later.`}
            />
          ) : (
            <div className="md-card md-elevation-1 md-shape-md overflow-hidden" style={{ background: 'var(--color-md-surface)' }}>
              <table className="w-full text-sm">
                <thead>
                  <tr
                    className="border-b"
                    style={{ borderColor: 'var(--color-md-outline-variant)', color: 'var(--color-md-on-surface-variant)' }}
                  >
                    <th className="text-left px-4 py-3 md-typescale-label-medium">SKU</th>
                    <th className="text-left px-4 py-3 md-typescale-label-medium">Product</th>
                    <th className="text-right px-4 py-3 md-typescale-label-medium">Total Qty</th>
                    <th className="text-right px-4 py-3 md-typescale-label-medium">Orders</th>
                  </tr>
                </thead>
                <tbody>
                  {lines.map((l) => (
                    <tr
                      key={l.sku_id}
                      className="border-b last:border-b-0"
                      style={{ borderColor: 'var(--color-md-outline-variant)', color: 'var(--color-md-on-surface)' }}
                    >
                      <td className="px-4 py-3 font-mono text-xs">{l.sku_id}</td>
                      <td className="px-4 py-3">{l.product_name}</td>
                      <td className="px-4 py-3 text-right tabular-nums font-medium">{l.total_qty.toLocaleString()}</td>
                      <td className="px-4 py-3 text-right tabular-nums">{l.order_count}</td>
                    </tr>
                  ))}
                </tbody>
                <tfoot>
                  <tr
                    className="border-t-2 font-medium"
                    style={{ borderColor: 'var(--color-md-outline)', color: 'var(--color-md-on-surface)' }}
                  >
                    <td className="px-4 py-3" colSpan={2}>Total</td>
                    <td className="px-4 py-3 text-right tabular-nums">{totalQty.toLocaleString()}</td>
                    <td className="px-4 py-3 text-right tabular-nums">{lines.reduce((a, l) => a + l.order_count, 0)}</td>
                  </tr>
                </tfoot>
              </table>
            </div>
          )}
        </>
      )}

      {/* Orders Tab */}
      {!loading && tab === 'orders' && (
        <>
          {orders.length === 0 ? (
            <EmptyState
              icon="orders"
              headline="No orders"
              body={`No orders found for ${date}.`}
            />
          ) : (
            <div className="md-card md-elevation-1 md-shape-md overflow-hidden" style={{ background: 'var(--color-md-surface)' }}>
              <table className="w-full text-sm">
                <thead>
                  <tr
                    className="border-b"
                    style={{ borderColor: 'var(--color-md-outline-variant)', color: 'var(--color-md-on-surface-variant)' }}
                  >
                    <th className="text-left px-4 py-3 md-typescale-label-medium">Order ID</th>
                    <th className="text-left px-4 py-3 md-typescale-label-medium">Retailer</th>
                    <th className="text-right px-4 py-3 md-typescale-label-medium">Items</th>
                    <th className="text-left px-4 py-3 md-typescale-label-medium">State</th>
                    <th className="text-left px-4 py-3 md-typescale-label-medium">Created</th>
                  </tr>
                </thead>
                <tbody>
                  {orders.map((o) => (
                    <tr
                      key={o.order_id}
                      className="border-b last:border-b-0"
                      style={{ borderColor: 'var(--color-md-outline-variant)', color: 'var(--color-md-on-surface)' }}
                    >
                      <td className="px-4 py-3 font-mono text-xs">{shortId(o.order_id)}</td>
                      <td className="px-4 py-3">{o.retailer_name}</td>
                      <td className="px-4 py-3 text-right tabular-nums">{o.item_count}</td>
                      <td className="px-4 py-3">
                        <span
                          className="md-shape-full px-2 py-0.5 text-xs font-medium inline-block"
                          style={{
                            background: 'var(--color-md-secondary-container)',
                            color: 'var(--color-md-on-secondary-container)',
                          }}
                        >
                          {o.state}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-xs tabular-nums" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                        {new Date(o.created_at).toLocaleString()}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {/* Manifest Ops Tab */}
      {!loading && tab === 'manifest-ops' && (
        <>
          {manifests.length === 0 ? (
            <EmptyState
              icon="fleet"
              headline="No manifests"
              body="No manifests found. Manifests are auto-created by the dispatch optimizer."
            />
          ) : (
            <div className="md-card md-elevation-1 md-shape-md overflow-hidden" style={{ background: 'var(--color-md-surface)' }}>
              <table className="w-full text-sm">
                <thead>
                  <tr
                    className="border-b"
                    style={{ borderColor: 'var(--color-md-outline-variant)', color: 'var(--color-md-on-surface-variant)' }}
                  >
                    <th className="text-left px-4 py-3 md-typescale-label-medium">Manifest</th>
                    <th className="text-left px-4 py-3 md-typescale-label-medium">State</th>
                    <th className="text-left px-4 py-3 md-typescale-label-medium">Driver</th>
                    <th className="text-left px-4 py-3 md-typescale-label-medium">Truck</th>
                    <th className="text-right px-4 py-3 md-typescale-label-medium">Stops</th>
                    <th className="text-right px-4 py-3 md-typescale-label-medium">Volume</th>
                    <th className="text-left px-4 py-3 md-typescale-label-medium">Created</th>
                    <th className="text-left px-4 py-3 md-typescale-label-medium">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {manifests.map((m) => (
                    <tr
                      key={m.manifest_id}
                      className="border-b last:border-b-0"
                      style={{ borderColor: 'var(--color-md-outline-variant)', color: 'var(--color-md-on-surface)' }}
                    >
                      <td className="px-4 py-3 font-mono text-xs">{shortId(m.manifest_id)}</td>
                      <td className="px-4 py-3">
                        <span
                          className="md-shape-full px-2 py-0.5 text-xs font-medium inline-block"
                          style={{
                            background: m.state === 'LOADING' ? 'var(--color-md-warning-container, #FEF3C7)'
                              : m.state === 'SEALED' ? 'var(--color-md-success-container, #D1FAE5)'
                              : 'var(--color-md-secondary-container)',
                            color: m.state === 'LOADING' ? 'var(--color-md-on-warning-container, #92400E)'
                              : m.state === 'SEALED' ? 'var(--color-md-on-success-container, #065F46)'
                              : 'var(--color-md-on-secondary-container)',
                          }}
                        >
                          {m.state}
                        </span>
                      </td>
                      <td className="px-4 py-3 font-mono text-xs">{shortId(m.driver_id)}</td>
                      <td className="px-4 py-3 font-mono text-xs">{shortId(m.truck_id)}</td>
                      <td className="px-4 py-3 text-right tabular-nums">{m.stop_count}</td>
                      <td className="px-4 py-3 text-right tabular-nums">
                        {m.total_volume_vu?.toFixed(1)}/{m.max_volume_vu?.toFixed(1)} VU
                      </td>
                      <td className="px-4 py-3 text-xs tabular-nums" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                        {new Date(m.created_at).toLocaleString()}
                      </td>
                      <td className="px-4 py-3">
                        {m.state === 'LOADING' && (
                          <div className="flex gap-2">
                            <button
                              onClick={() => setInjectModalManifest(m.manifest_id)}
                              className="md-btn md-btn-tonal md-typescale-label-small px-3 py-1.5 md-shape-sm"
                            >
                              Add Order
                            </button>
                            <button
                              onClick={() => handleForceSeal(m.manifest_id)}
                              disabled={forceSealingId === m.manifest_id}
                              className="md-btn md-btn-outlined md-typescale-label-small px-3 py-1.5 md-shape-sm"
                              style={{ borderColor: 'var(--color-md-error)', color: 'var(--color-md-error)' }}
                            >
                              {forceSealingId === m.manifest_id ? 'Sealing…' : 'Force Seal'}
                            </button>
                          </div>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {/* Inject Order Modal */}
      {injectModalManifest && (
        <div className="fixed inset-0 z-50 flex items-center justify-center" style={{ background: 'rgba(0,0,0,0.5)' }}>
          <div
            className="md-card md-elevation-3 md-shape-lg w-full max-w-md"
            style={{ background: 'var(--color-md-surface)' }}
          >
            <div className="px-6 py-4 border-b" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
              <h2 className="md-typescale-title-large" style={{ color: 'var(--color-md-on-surface)' }}>
                Inject Order
              </h2>
              <p className="md-typescale-body-small mt-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                Add a PENDING order to manifest {shortId(injectModalManifest)} during LOADING.
              </p>
            </div>
            <div className="px-6 py-4 flex flex-col gap-4">
              <div>
                <label className="md-typescale-label-medium block mb-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                  Order ID
                </label>
                <input
                  type="text"
                  value={injectOrderId}
                  onChange={(e) => setInjectOrderId(e.target.value)}
                  placeholder="Enter Order UUID"
                  className="md-input-outlined w-full px-3 py-2 md-typescale-body-medium md-shape-sm font-mono"
                  style={{
                    background: 'var(--color-md-surface)',
                    color: 'var(--color-md-on-surface)',
                    borderColor: 'var(--color-md-outline)',
                  }}
                />
              </div>
              <div className="flex gap-3 justify-end">
                <button
                  onClick={() => { setInjectModalManifest(null); setInjectOrderId(''); }}
                  className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2"
                >
                  Cancel
                </button>
                <button
                  onClick={handleInjectOrder}
                  disabled={!injectOrderId.trim() || isInjecting}
                  className="md-btn md-btn-filled md-typescale-label-large px-4 py-2"
                  style={{ opacity: !injectOrderId.trim() || isInjecting ? 0.5 : 1 }}
                >
                  {isInjecting ? 'Injecting…' : 'Inject Order'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
