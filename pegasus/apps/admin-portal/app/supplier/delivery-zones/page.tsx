'use client';

import { useState, useCallback, useEffect } from 'react';
import { useToken } from '@/lib/auth';
import EmptyState from '@/components/EmptyState';
import { Skeleton } from '@/components/Skeleton';
import Icon from '@/components/Icon';
import { useToast } from '@/components/Toast';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

function buildDeliveryZoneCreateIdempotencyKey(payload: Record<string, unknown>): string {
  return ['supplier-delivery-zone-create', JSON.stringify(payload)].join(':');
}

function buildDeliveryZoneDeactivateIdempotencyKey(zoneId: string): string {
  return ['supplier-delivery-zone-deactivate', zoneId.trim()].join(':');
}

/* ─── Types ───────────────────────────────────────────────── */

interface DeliveryZone {
  zone_id: string;
  supplier_id: string;
  warehouse_id: string;
  zone_name: string;
  min_distance_km: number;
  max_distance_km: number;
  fee_minor: number;
  priority: number;
  is_active: boolean;
}

/* ─── Main Page ───────────────────────────────────────────── */

export default function DeliveryZonesPage() {
  const token = useToken();
  const { toast } = useToast();

  const [zones, setZones] = useState<DeliveryZone[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [creating, setCreating] = useState(false);

  // Create form
  const [zoneName, setZoneName] = useState('');
  const [minDist, setMinDist] = useState('0');
  const [maxDist, setMaxDist] = useState('');
  const [fee, setFee] = useState('');
  const [priority, setPriority] = useState('0');
  const [warehouseId, setWarehouseId] = useState('');

  /* ─── Fetch ─────────────────────────────────────────────── */

  const fetchZones = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    try {
      const res = await fetch(`${API}/v1/supplier/delivery-zones`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (res.ok) {
        const j = await res.json();
        setZones(j.zones || []);
      }
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setLoading(false);
    }
  }, [token, toast]);

  useEffect(() => {
    fetchZones();
  }, [fetchZones]);

  /* ─── Create Zone ───────────────────────────────────────── */

  const handleCreate = useCallback(async () => {
    if (!token || !zoneName || !maxDist || !fee) return;
    setCreating(true);
    try {
      const payload = {
        zone_name: zoneName,
        min_distance_km: parseFloat(minDist) || 0,
        max_distance_km: parseFloat(maxDist),
        fee_minor: parseInt(fee, 10),
        priority: parseInt(priority, 10) || 0,
        warehouse_id: warehouseId || undefined,
      };
      const res = await fetch(`${API}/v1/supplier/delivery-zones`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Idempotency-Key': buildDeliveryZoneCreateIdempotencyKey(payload),
        },
        body: JSON.stringify(payload),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      toast('Delivery zone created', 'success');
      setShowCreate(false);
      setZoneName('');
      setMinDist('0');
      setMaxDist('');
      setFee('');
      setPriority('0');
      setWarehouseId('');
      fetchZones();
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setCreating(false);
    }
  }, [token, zoneName, minDist, maxDist, fee, priority, warehouseId, toast, fetchZones]);

  /* ─── Deactivate Zone ───────────────────────────────────── */

  const handleDeactivate = useCallback(async (zoneId: string) => {
    if (!token) return;
    try {
      const res = await fetch(`${API}/v1/supplier/delivery-zones/${zoneId}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${token}`,
          'Idempotency-Key': buildDeliveryZoneDeactivateIdempotencyKey(zoneId),
        },
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      toast('Zone deactivated', 'success');
      fetchZones();
    } catch (e) {
      toast((e as Error).message, 'error');
    }
  }, [token, toast, fetchZones]);

  /* ─── Render ────────────────────────────────────────────── */

  return (
    <div className="flex flex-col gap-6 w-full max-w-7xl mx-auto px-4 py-6">
      {/* Header */}
      <div className="flex items-center justify-between flex-wrap gap-4">
        <div>
          <h1 className="md-typescale-headline-small" style={{ color: 'var(--color-md-on-surface)' }}>
            Delivery Zones
          </h1>
          <p className="md-typescale-body-small mt-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
            Distance-based delivery fee bands per warehouse. Retailers are charged based on zone.
          </p>
        </div>
        <button
          onClick={() => setShowCreate(true)}
          className="md-btn md-btn-filled md-typescale-label-large px-4 py-2 flex items-center gap-2"
        >
          <Icon name="fleet" size={16} />
          Create Zone
        </button>
      </div>

      {/* Loading */}
      {loading && (
        <div className="flex flex-col gap-3">
          {[1, 2, 3].map((i) => (
            <Skeleton key={i} className="h-12 w-full rounded-lg" />
          ))}
        </div>
      )}

      {/* Zone Table */}
      {!loading && (
        <>
          {zones.length === 0 ? (
            <EmptyState
              icon="fleet"
              headline="No delivery zones"
              body="Create distance-based fee bands to charge retailers for delivery."
            />
          ) : (
            <div className="md-card md-elevation-1 md-shape-md overflow-hidden" style={{ background: 'var(--color-md-surface)' }}>
              <table className="w-full text-sm">
                <thead>
                  <tr
                    className="border-b"
                    style={{ borderColor: 'var(--color-md-outline-variant)', color: 'var(--color-md-on-surface-variant)' }}
                  >
                    <th className="text-left px-4 py-3 md-typescale-label-medium">Zone Name</th>
                    <th className="text-right px-4 py-3 md-typescale-label-medium">Min (km)</th>
                    <th className="text-right px-4 py-3 md-typescale-label-medium">Max (km)</th>
                    <th className="text-right px-4 py-3 md-typescale-label-medium">Fee (minor)</th>
                    <th className="text-right px-4 py-3 md-typescale-label-medium">Priority</th>
                    <th className="text-left px-4 py-3 md-typescale-label-medium">Warehouse</th>
                    <th className="text-left px-4 py-3 md-typescale-label-medium">Status</th>
                    <th className="text-left px-4 py-3 md-typescale-label-medium">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {zones.map((z) => (
                    <tr
                      key={z.zone_id}
                      className="border-b last:border-b-0"
                      style={{
                        borderColor: 'var(--color-md-outline-variant)',
                        color: 'var(--color-md-on-surface)',
                        opacity: z.is_active ? 1 : 0.5,
                      }}
                    >
                      <td className="px-4 py-3 font-medium">{z.zone_name}</td>
                      <td className="px-4 py-3 text-right tabular-nums">{z.min_distance_km}</td>
                      <td className="px-4 py-3 text-right tabular-nums">{z.max_distance_km}</td>
                      <td className="px-4 py-3 text-right tabular-nums font-mono">
                        {new Intl.NumberFormat('en-US').format(z.fee_minor)}
                      </td>
                      <td className="px-4 py-3 text-right tabular-nums">{z.priority}</td>
                      <td className="px-4 py-3 font-mono text-xs">
                        {z.warehouse_id ? z.warehouse_id.slice(0, 12) + '…' : 'All'}
                      </td>
                      <td className="px-4 py-3">
                        <span
                          className="md-shape-full px-2 py-0.5 text-xs font-medium inline-block"
                          style={{
                            background: z.is_active ? 'var(--color-md-success-container, #D1FAE5)' : 'var(--color-md-error-container, #FEE2E2)',
                            color: z.is_active ? 'var(--color-md-on-success-container, #065F46)' : 'var(--color-md-on-error-container, #991B1B)',
                          }}
                        >
                          {z.is_active ? 'Active' : 'Inactive'}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        {z.is_active && (
                          <button
                            onClick={() => handleDeactivate(z.zone_id)}
                            className="md-btn md-btn-outlined md-typescale-label-small px-3 py-1.5 md-shape-sm"
                            style={{ borderColor: 'var(--color-md-error)', color: 'var(--color-md-error)' }}
                          >
                            Deactivate
                          </button>
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

      {/* Create Zone Modal */}
      {showCreate && (
        <div className="fixed inset-0 z-50 flex items-center justify-center" style={{ background: 'rgba(0,0,0,0.5)' }}>
          <div
            className="md-card md-elevation-3 md-shape-lg w-full max-w-lg"
            style={{ background: 'var(--color-md-surface)' }}
          >
            <div className="px-6 py-4 border-b" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
              <h2 className="md-typescale-title-large" style={{ color: 'var(--color-md-on-surface)' }}>
                Create Delivery Zone
              </h2>
            </div>
            <div className="px-6 py-4 flex flex-col gap-4">
              <div>
                <label className="md-typescale-label-medium block mb-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                  Zone Name
                </label>
                <input
                  type="text"
                  value={zoneName}
                  onChange={(e) => setZoneName(e.target.value)}
                  placeholder="e.g. City Center"
                  className="md-input-outlined w-full px-3 py-2 md-typescale-body-medium md-shape-sm"
                  style={{ background: 'var(--color-md-surface)', color: 'var(--color-md-on-surface)', borderColor: 'var(--color-md-outline)' }}
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="md-typescale-label-medium block mb-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                    Min Distance (km)
                  </label>
                  <input
                    type="number"
                    value={minDist}
                    onChange={(e) => setMinDist(e.target.value)}
                    className="md-input-outlined w-full px-3 py-2 md-typescale-body-medium md-shape-sm"
                    style={{ background: 'var(--color-md-surface)', color: 'var(--color-md-on-surface)', borderColor: 'var(--color-md-outline)' }}
                  />
                </div>
                <div>
                  <label className="md-typescale-label-medium block mb-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                    Max Distance (km)
                  </label>
                  <input
                    type="number"
                    value={maxDist}
                    onChange={(e) => setMaxDist(e.target.value)}
                    className="md-input-outlined w-full px-3 py-2 md-typescale-body-medium md-shape-sm"
                    style={{ background: 'var(--color-md-surface)', color: 'var(--color-md-on-surface)', borderColor: 'var(--color-md-outline)' }}
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="md-typescale-label-medium block mb-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                    Fee (minor currency)
                  </label>
                  <input
                    type="number"
                    value={fee}
                    onChange={(e) => setFee(e.target.value)}
                    placeholder="e.g. 50000"
                    className="md-input-outlined w-full px-3 py-2 md-typescale-body-medium md-shape-sm"
                    style={{ background: 'var(--color-md-surface)', color: 'var(--color-md-on-surface)', borderColor: 'var(--color-md-outline)' }}
                  />
                </div>
                <div>
                  <label className="md-typescale-label-medium block mb-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                    Priority
                  </label>
                  <input
                    type="number"
                    value={priority}
                    onChange={(e) => setPriority(e.target.value)}
                    className="md-input-outlined w-full px-3 py-2 md-typescale-body-medium md-shape-sm"
                    style={{ background: 'var(--color-md-surface)', color: 'var(--color-md-on-surface)', borderColor: 'var(--color-md-outline)' }}
                  />
                </div>
              </div>
              <div>
                <label className="md-typescale-label-medium block mb-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                  Warehouse ID (optional — blank = all warehouses)
                </label>
                <input
                  type="text"
                  value={warehouseId}
                  onChange={(e) => setWarehouseId(e.target.value)}
                  placeholder="Leave blank for global zone"
                  className="md-input-outlined w-full px-3 py-2 md-typescale-body-medium md-shape-sm font-mono"
                  style={{ background: 'var(--color-md-surface)', color: 'var(--color-md-on-surface)', borderColor: 'var(--color-md-outline)' }}
                />
              </div>
              <div className="flex gap-3 justify-end pt-2">
                <button
                  onClick={() => setShowCreate(false)}
                  className="md-btn md-btn-outlined md-typescale-label-large px-4 py-2"
                >
                  Cancel
                </button>
                <button
                  onClick={handleCreate}
                  disabled={!zoneName || !maxDist || !fee || creating}
                  className="md-btn md-btn-filled md-typescale-label-large px-4 py-2"
                  style={{ opacity: !zoneName || !maxDist || !fee || creating ? 0.5 : 1 }}
                >
                  {creating ? 'Creating…' : 'Create Zone'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
