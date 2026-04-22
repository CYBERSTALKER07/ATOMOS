'use client';

import { useState, useEffect, useCallback } from 'react';
import { Button } from '@heroui/react';
import { useToken } from '@/lib/auth';
import Icon from '@/components/Icon';
import { useToast } from '@/components/Toast';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

/* ─── Types ───────────────────────────────────────────────── */

interface QuarantineLineItem {
  line_item_id: string;
  sku_id: string;
  product_name: string;
  quantity: number;
  unit_price: number;
}

interface QuarantineOrder {
  order_id: string;
  retailer_name: string;
  items: QuarantineLineItem[];
}

interface QuarantineVehicle {
  vehicle_id: string;
  vehicle_class: string;
  driver_name: string;
  route_id: string;
  orders: QuarantineOrder[];
}

/* ─── Helpers ─────────────────────────────────────────────── */

function formatAmount(amount: number): string {
  return new Intl.NumberFormat('en-US').format(amount);
}

function shortId(id: string): string {
  return '#' + id.slice(-6).toUpperCase();
}

/* ─── Main Page ───────────────────────────────────────────── */

export default function DepotReconciliationPage() {
  const token = useToken();
  const { toast } = useToast();

  const [vehicles, setVehicles] = useState<QuarantineVehicle[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  const fetchStock = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(`${API}/v1/supplier/quarantine-stock`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const json = await res.json();
      setVehicles(json.data ?? []);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load quarantine stock');
    } finally {
      setLoading(false);
    }
  }, [token]);

  useEffect(() => { fetchStock(); }, [fetchStock]);

  async function reconcile(lineItemIds: string[], action: 'RESTOCK' | 'WRITE_OFF_DAMAGED') {
    if (!token) return;
    const key = lineItemIds.join(',') + ':' + action;
    setActionLoading(key);
    try {
      const res = await fetch(`${API}/v1/inventory/reconcile-returns`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ line_item_ids: lineItemIds, action }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        throw new Error(err.error ?? `HTTP ${res.status}`);
      }
      const label = action === 'RESTOCK' ? 'Restocked' : 'Written off';
      toast(`${label} ${lineItemIds.length} item(s)`, 'success');
      await fetchStock();
    } catch (e: unknown) {
      toast(e instanceof Error ? e.message : 'Reconciliation failed', 'error');
    } finally {
      setActionLoading(null);
    }
  }

  /* ─── Loading ─────────────────────────────────────────── */
  if (loading) {
    return (
      <div className="flex flex-col gap-4 p-6">
        <div className="h-8 w-64 bg-[var(--surface)] rounded animate-pulse" />
        {[1, 2].map(i => (
          <div key={i} className="h-40 rounded-xl bg-[var(--surface)] animate-pulse" />
        ))}
      </div>
    );
  }

  /* ─── Error ───────────────────────────────────────────── */
  if (error) {
    return (
      <div className="flex flex-col items-center justify-center gap-4 p-12 text-center">
        <Icon name="error" className="w-10 h-10 text-[var(--danger)]" />
        <p className="md-typescale-body-large text-[var(--foreground)]">{error}</p>
        <Button variant="secondary" onPress={fetchStock}>Retry</Button>
      </div>
    );
  }

  /* ─── Empty ───────────────────────────────────────────── */
  if (vehicles.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-4 p-12 text-center">
        <Icon name="warehouse" className="w-12 h-12 text-[var(--border)]" />
        <p className="md-typescale-title-medium text-[var(--foreground)]">No quarantine stock</p>
        <p className="md-typescale-body-medium text-[var(--muted)]">
          All returned loads have been reconciled.
        </p>
      </div>
    );
  }

  /* ─── Main ────────────────────────────────────────────── */
  return (
    <div className="p-6 flex flex-col gap-6 max-w-5xl">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="md-typescale-headline-small text-[var(--foreground)]">
            Depot Reconciliation
          </h1>
          <p className="md-typescale-body-medium text-[var(--muted)] mt-1">
            Returned loads awaiting restock or write-off decision.
          </p>
        </div>
        <Button variant="outline" onPress={fetchStock}>
          <Icon name="refresh" className="w-4 h-4" />
          Refresh
        </Button>
      </div>

      {vehicles.map(vehicle => (
        <VehicleCard
          key={vehicle.vehicle_id || 'no-vehicle'}
          vehicle={vehicle}
          actionLoading={actionLoading}
          onReconcile={reconcile}
        />
      ))}
    </div>
  );
}

/* ─── Vehicle Card ────────────────────────────────────────── */

function VehicleCard({
  vehicle,
  actionLoading,
  onReconcile,
}: {
  vehicle: QuarantineVehicle;
  actionLoading: string | null;
  onReconcile: (ids: string[], action: 'RESTOCK' | 'WRITE_OFF_DAMAGED') => Promise<void>;
}) {
  const allItems = vehicle.orders.flatMap(o => o.items.map(i => i.line_item_id));

  return (
    <div
      className="md-card md-elevation-1 md-shape-md overflow-hidden"
      style={{ background: 'var(--surface)' }}
    >
      {/* Vehicle Header */}
      <div
        className="flex items-center justify-between px-5 py-4 border-b"
        style={{ borderColor: 'var(--border)' }}
      >
        <div className="flex items-center gap-3">
          <div
            className="flex items-center justify-center w-9 h-9 rounded-full"
            style={{ background: 'var(--default)' }}
          >
            <Icon name="fleet" className="w-4 h-4 text-[var(--default-foreground)]" />
          </div>
          <div>
            <p className="md-typescale-title-small text-[var(--foreground)] font-semibold">
              {vehicle.vehicle_class || 'Unknown Vehicle'} — {vehicle.driver_name}
            </p>
            <p className="md-typescale-body-small text-[var(--muted)]">
              Route {vehicle.route_id ? shortId(vehicle.route_id) : 'N/A'}
              {' · '}{vehicle.orders.length} order{vehicle.orders.length !== 1 ? 's' : ''}
            </p>
          </div>
        </div>
        <div className="flex gap-2">
          <Button
            variant="secondary"
            size="sm"
            isDisabled={actionLoading !== null || allItems.length === 0}
            onPress={() => onReconcile(allItems, 'RESTOCK')}
          >
            Restock All
          </Button>
          <Button
            variant="outline"
            size="sm"
            isDisabled={actionLoading !== null || allItems.length === 0}
            onPress={() => onReconcile(allItems, 'WRITE_OFF_DAMAGED')}
            className="text-danger border-danger"
          >
            Write Off All
          </Button>
        </div>
      </div>

      {/* Orders */}
      <div className="flex flex-col divide-y" style={{ borderColor: 'var(--border)' }}>
        {vehicle.orders.map(order => (
          <OrderSection
            key={order.order_id}
            order={order}
            actionLoading={actionLoading}
            onReconcile={onReconcile}
          />
        ))}
      </div>
    </div>
  );
}

/* ─── Order Section ───────────────────────────────────────── */

function OrderSection({
  order,
  actionLoading,
  onReconcile,
}: {
  order: QuarantineOrder;
  actionLoading: string | null;
  onReconcile: (ids: string[], action: 'RESTOCK' | 'WRITE_OFF_DAMAGED') => Promise<void>;
}) {
  return (
    <div className="px-5 py-3">
      <div className="flex items-center justify-between mb-2">
        <div>
          <span className="md-typescale-label-large text-[var(--foreground)] font-semibold">
            {shortId(order.order_id)}
          </span>
          <span className="ml-2 md-typescale-body-small text-[var(--muted)]">
            {order.retailer_name}
          </span>
        </div>
        <div
          className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full md-typescale-label-small"
          style={{ background: 'var(--danger)', color: 'var(--danger-foreground)' }}
        >
          <span className="w-1.5 h-1.5 rounded-full bg-current" />
          QUARANTINE
        </div>
      </div>

      <table className="w-full text-sm">
        <thead>
          <tr className="text-left" style={{ color: 'var(--muted)' }}>
            <th className="pb-1 md-typescale-label-small font-medium">Product</th>
            <th className="pb-1 md-typescale-label-small font-medium text-right">Qty</th>
            <th className="pb-1 md-typescale-label-small font-medium text-right">Unit Price</th>
            <th className="pb-1 md-typescale-label-small font-medium text-right w-44">Actions</th>
          </tr>
        </thead>
        <tbody>
          {order.items.map(item => {
            const keyRestock = `${item.line_item_id}:RESTOCK`;
            const keyWriteOff = `${item.line_item_id}:WRITE_OFF_DAMAGED`;
            const busy = actionLoading === keyRestock || actionLoading === keyWriteOff;
            return (
              <tr key={item.line_item_id} className="border-t" style={{ borderColor: 'var(--border)' }}>
                <td className="py-2 md-typescale-body-small text-[var(--foreground)]">
                  {item.product_name}
                </td>
                <td className="py-2 md-typescale-body-small text-[var(--foreground)] text-right" style={{ fontVariantNumeric: 'tabular-nums' }}>
                  {item.quantity}
                </td>
                <td className="py-2 md-typescale-body-small text-[var(--muted)] text-right" style={{ fontVariantNumeric: 'tabular-nums' }}>
                  {formatAmount(item.unit_price)}
                </td>
                <td className="py-2 text-right">
                  <div className="flex justify-end gap-1">
                    <Button
                      variant="secondary"
                      size="sm"
                      isDisabled={actionLoading !== null}
                      onPress={() => onReconcile([item.line_item_id], 'RESTOCK')}
                    >
                      {busy && actionLoading === keyRestock ? '…' : 'Restock'}
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      isDisabled={actionLoading !== null}
                      onPress={() => onReconcile([item.line_item_id], 'WRITE_OFF_DAMAGED')}
                      className="text-danger border-danger"
                    >
                      {busy && actionLoading === keyWriteOff ? '…' : 'Write Off'}
                    </Button>
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
