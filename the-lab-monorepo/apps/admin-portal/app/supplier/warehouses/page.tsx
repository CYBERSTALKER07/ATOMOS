'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Button } from '@heroui/react';
import { apiFetch } from '@/lib/auth';
import StatsCard from '@/components/StatsCard';
import Drawer from '@/components/Drawer';
import Icon from '@/components/Icon';
import EmptyState from '@/components/EmptyState';
import WarehouseForm from './WarehouseForm';
import CoverageEditor from './CoverageEditor';
import WarehouseStaffPanel from './WarehouseStaffPanel';
import OperatingScheduleEditor from './OperatingScheduleEditor';

interface WarehouseItem {
  warehouse_id: string;
  name: string;
  address?: string;
  lat: number;
  lng: number;
  coverage_radius_km: number;
  hex_count: number;
  is_active: boolean;
  is_default: boolean;
  is_on_shift: boolean;
  driver_count: number;
  order_count: number;
}

interface WarehouseDetail extends WarehouseItem {
  h3_indexes?: string[];
  primary_factory_id?: string;
  secondary_factory_id?: string;
  created_at: string;
  updated_at?: string;
}

export default function WarehousesPage() {
  const [warehouses, setWarehouses] = useState<WarehouseItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Drawer state
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'detail' | 'create' | 'coverage' | 'edit' | 'staff'>('detail');
  const [selectedWarehouse, setSelectedWarehouse] = useState<WarehouseDetail | null>(null);
  const [detailLoading, setDetailLoading] = useState(false);

  const fetchWarehouses = useCallback(async () => {
    try {
      setLoading(true);
      const res = await apiFetch('/v1/supplier/warehouses');
      if (!res.ok) throw new Error('Failed to fetch warehouses');
      const data = await res.json();
      setWarehouses(data.warehouses || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchWarehouses();
  }, [fetchWarehouses]);

  const openDetail = async (wh: WarehouseItem) => {
    setDrawerMode('detail');
    setDrawerOpen(true);
    setDetailLoading(true);
    try {
      const res = await apiFetch(`/v1/supplier/warehouses/${wh.warehouse_id}`);
      if (!res.ok) throw new Error('Failed to load detail');
      const data: WarehouseDetail = await res.json();
      setSelectedWarehouse(data);
    } catch {
      setSelectedWarehouse({ ...wh, h3_indexes: [], created_at: '' });
    } finally {
      setDetailLoading(false);
    }
  };

  const openCreate = () => {
    setDrawerMode('create');
    setSelectedWarehouse(null);
    setDrawerOpen(true);
  };

  const openCoverage = (wh: WarehouseDetail) => {
    setDrawerMode('coverage');
    setSelectedWarehouse(wh);
    setDrawerOpen(true);
  };

  const closeDrawer = () => {
    setDrawerOpen(false);
    setSelectedWarehouse(null);
  };

  const handleCreated = () => {
    closeDrawer();
    fetchWarehouses();
  };

  const openEdit = (wh: WarehouseDetail) => {
    setDrawerMode('edit');
    setSelectedWarehouse(wh);
    setDrawerOpen(true);
  };

  const openStaff = (wh: WarehouseDetail) => {
    setDrawerMode('staff');
    setSelectedWarehouse(wh);
    setDrawerOpen(true);
  };

  const handleUpdate = async (updates: Record<string, unknown>) => {
    if (!selectedWarehouse) return;
    const res = await apiFetch(`/v1/supplier/warehouses/${selectedWarehouse.warehouse_id}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
    if (res.ok) {
      closeDrawer();
      fetchWarehouses();
    } else {
      const data = await res.json().catch(() => ({}));
      alert(data.error || 'Failed to update warehouse');
    }
  };

  const handleToggleShift = async (wh: WarehouseItem) => {
    await apiFetch(`/v1/supplier/warehouses/${wh.warehouse_id}`, {
      method: 'PATCH',
      body: JSON.stringify({ is_on_shift: !wh.is_on_shift }),
    });
    fetchWarehouses();
  };

  const handleDeactivate = async (id: string) => {
    const res = await apiFetch(`/v1/supplier/warehouses/${id}`, { method: 'DELETE' });
    if (res.ok) {
      closeDrawer();
      fetchWarehouses();
    } else {
      const data = await res.json();
      alert(data.error || 'Failed to deactivate');
    }
  };

  // KPIs
  const totalDrivers = warehouses.reduce((s, w) => s + w.driver_count, 0);
  const totalOrders = warehouses.reduce((s, w) => s + w.order_count, 0);
  const totalHexes = warehouses.reduce((s, w) => s + w.hex_count, 0);
  const onShift = warehouses.filter(w => w.is_on_shift).length;

  return (
    <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="md-typescale-headline-medium">Warehouses</h1>
          <p className="md-typescale-body-medium mt-1" style={{ color: 'var(--muted)' }}>
            Manage depot locations, coverage zones, and shift status
          </p>
        </div>
        <Button className="button--primary" onPress={openCreate}>
          <Icon name="warehouse" size={18} className="mr-2" />
          Add Warehouse
        </Button>
      </div>

      {/* KPI row */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        <StatsCard label="Warehouses" value={String(warehouses.length)} sub={`${onShift} on shift`} delay={0} />
        <StatsCard label="Drivers" value={String(totalDrivers)} delay={50} />
        <StatsCard label="Active Orders" value={String(totalOrders)} delay={100} />
        <StatsCard label="H3 Cells" value={String(totalHexes)} sub="coverage hexes" delay={150} accent="var(--accent)" />
      </div>

      {/* Table */}
      {loading ? (
        <div className="flex items-center justify-center py-20">
          <div className="w-8 h-8 border-2 border-t-transparent rounded-full animate-spin" style={{ borderColor: 'var(--accent)', borderTopColor: 'transparent' }} />
        </div>
      ) : error ? (
        <EmptyState icon="error" headline="Failed to load" body={error} action="Retry" onAction={fetchWarehouses} />
      ) : warehouses.length === 0 ? (
        <EmptyState icon="warehouse" headline="No warehouses yet" body="Create your first warehouse to define coverage zones." action="Add Warehouse" onAction={openCreate} />
      ) : (
        <div className="md-card md-card-elevated overflow-hidden">
          <table className="md-table w-full">
            <thead>
              <tr>
                <th className="table__column px-4 py-3 text-left md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Name</th>
                <th className="table__column px-4 py-3 text-left md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Address</th>
                <th className="table__column px-4 py-3 text-center md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Hexes</th>
                <th className="table__column px-4 py-3 text-center md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Drivers</th>
                <th className="table__column px-4 py-3 text-center md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Orders</th>
                <th className="table__column px-4 py-3 text-center md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Shift</th>
                <th className="table__column px-4 py-3 text-center md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Default</th>
              </tr>
            </thead>
            <tbody>
              {warehouses.map((wh, i) => (
                <tr
                  key={wh.warehouse_id}
                  onClick={() => openDetail(wh)}
                  className="cursor-pointer transition-colors md-animate-in"
                  style={{
                    animationDelay: `${i * 30}ms`,
                    borderBottom: '1px solid var(--border)',
                  }}
                  onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface)')}
                  onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                >
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-3">
                      <div
                        className="w-9 h-9 rounded-lg flex items-center justify-center"
                        style={{ background: 'var(--accent-soft)' }}
                      >
                        <Icon name="warehouse" size={18} className="text-accent" />
                      </div>
                      <div>
                        <p className="md-typescale-body-medium font-medium">{wh.name}</p>
                        {wh.coverage_radius_km > 0 && (
                          <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                            {wh.coverage_radius_km} km radius
                          </p>
                        )}
                      </div>
                    </div>
                  </td>
                  <td className="px-4 py-3 md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                    {wh.address || '—'}
                  </td>
                  <td className="px-4 py-3 text-center md-typescale-body-small" style={{ fontVariantNumeric: 'tabular-nums' }}>
                    {wh.hex_count}
                  </td>
                  <td className="px-4 py-3 text-center md-typescale-body-small" style={{ fontVariantNumeric: 'tabular-nums' }}>
                    {wh.driver_count}
                  </td>
                  <td className="px-4 py-3 text-center md-typescale-body-small" style={{ fontVariantNumeric: 'tabular-nums' }}>
                    {wh.order_count}
                  </td>
                  <td className="px-4 py-3 text-center" onClick={e => e.stopPropagation()}>
                    <button
                      onClick={() => handleToggleShift(wh)}
                      className="w-8 h-5 rounded-full transition-colors relative"
                      style={{ background: wh.is_on_shift ? 'var(--success)' : 'var(--border)' }}
                    >
                      <div
                        className="absolute top-0.5 w-4 h-4 rounded-full transition-transform"
                        style={{
                          background: 'white',
                          transform: wh.is_on_shift ? 'translateX(14px)' : 'translateX(2px)',
                        }}
                      />
                    </button>
                  </td>
                  <td className="px-4 py-3 text-center">
                    {wh.is_default && (
                      <span className="md-typescale-label-small px-2 py-0.5 rounded-full" style={{ background: 'var(--accent-soft)', color: 'var(--accent)' }}>
                        Default
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Drawer */}
      <Drawer
        open={drawerOpen}
        onClose={closeDrawer}
        title={
          drawerMode === 'create'
            ? 'New Warehouse'
            : drawerMode === 'coverage'
            ? 'Coverage Zone'
            : drawerMode === 'edit'
            ? `Edit: ${selectedWarehouse?.name || 'Warehouse'}`
            : drawerMode === 'staff'
            ? `Staff: ${selectedWarehouse?.name || 'Warehouse'}`
            : selectedWarehouse?.name || 'Warehouse'
        }
      >
        {drawerMode === 'create' && (
          <WarehouseForm onSuccess={handleCreated} onCancel={closeDrawer} />
        )}

        {drawerMode === 'detail' && (
          detailLoading ? (
            <div className="flex items-center justify-center py-20">
              <div className="w-8 h-8 border-2 border-t-transparent rounded-full animate-spin" style={{ borderColor: 'var(--accent)', borderTopColor: 'transparent' }} />
            </div>
          ) : selectedWarehouse ? (
            <div className="p-6 space-y-6">
              {/* Status bar */}
              <div className="flex items-center gap-3 flex-wrap">
                <span
                  className="md-typescale-label-small px-3 py-1 rounded-full"
                  style={{
                    background: selectedWarehouse.is_on_shift ? 'color-mix(in srgb, var(--success) 15%, transparent)' : 'color-mix(in srgb, var(--danger) 15%, transparent)',
                    color: selectedWarehouse.is_on_shift ? 'var(--success)' : 'var(--danger)',
                  }}
                >
                  {selectedWarehouse.is_on_shift ? 'On Shift' : 'Off Shift'}
                </span>
                <span
                  className="md-typescale-label-small px-3 py-1 rounded-full"
                  style={{
                    background: selectedWarehouse.is_active ? 'color-mix(in srgb, var(--success) 15%, transparent)' : 'color-mix(in srgb, var(--danger) 15%, transparent)',
                    color: selectedWarehouse.is_active ? 'var(--success)' : 'var(--danger)',
                  }}
                >
                  {selectedWarehouse.is_active ? 'Active' : 'Disabled'}
                </span>
                {selectedWarehouse.is_default && (
                  <span className="md-typescale-label-small px-3 py-1 rounded-full" style={{ background: 'var(--accent-soft)', color: 'var(--accent)' }}>
                    Default
                  </span>
                )}
              </div>

              {/* Coordinates */}
              <div className="md-card md-card-elevated p-4 space-y-2">
                <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Location</p>
                <p className="md-typescale-body-medium">{selectedWarehouse.address || 'No address'}</p>
                <p className="md-typescale-label-small" style={{ color: 'var(--muted)', fontVariantNumeric: 'tabular-nums' }}>
                  {selectedWarehouse.lat.toFixed(6)}, {selectedWarehouse.lng.toFixed(6)}
                </p>
              </div>

              {/* Stats grid */}
              <div className="grid grid-cols-3 gap-3">
                <StatsCard label="Drivers" value={String(selectedWarehouse.driver_count)} delay={0} />
                <StatsCard label="Orders" value={String(selectedWarehouse.order_count)} delay={50} />
                <StatsCard label="H3 Cells" value={String(selectedWarehouse.hex_count)} delay={100} accent="var(--accent)" />
              </div>

              {/* Coverage radius */}
              <div className="md-card md-card-elevated p-4 flex items-center justify-between">
                <div>
                  <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Coverage Radius</p>
                  <p className="md-typescale-headline-small" style={{ fontVariantNumeric: 'tabular-nums' }}>
                    {selectedWarehouse.coverage_radius_km} km
                  </p>
                </div>
                <Button
                  className="button--primary"
                  size="sm"
                  onPress={() => openCoverage(selectedWarehouse)}
                >
                  <Icon name="hexagon" size={16} className="mr-1" />
                  Edit Zone
                </Button>
              </div>

              {/* Timestamps */}
              {selectedWarehouse.created_at && (
                <div className="space-y-1">
                  <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                    Created: {new Date(selectedWarehouse.created_at).toLocaleDateString()}
                  </p>
                  {selectedWarehouse.updated_at && (
                    <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                      Updated: {new Date(selectedWarehouse.updated_at).toLocaleDateString()}
                    </p>
                  )}
                </div>
              )}

              {/* Actions */}
              <div className="flex gap-3 pt-2 flex-wrap" style={{ borderTop: '1px solid var(--border)' }}>
                <Button
                  className="button--primary flex-1"
                  onPress={() => openEdit(selectedWarehouse)}
                >
                  <Icon name="config" size={16} className="mr-1" />
                  Edit
                </Button>
                <Button
                  variant="bordered"
                  className="flex-1"
                  onPress={() => openStaff(selectedWarehouse)}
                >
                  <Icon name="person" size={16} className="mr-1" />
                  Staff
                </Button>
                <Button
                  variant="bordered"
                  className="flex-1"
                  onPress={() => openCoverage(selectedWarehouse)}
                >
                  <Icon name="hexagon" size={16} className="mr-1" />
                  Coverage
                </Button>
                <Button
                  variant="bordered"
                  className="border-[var(--color-md-error)] text-[var(--color-md-error)]"
                  onPress={() => {
                    if (confirm('Deactivate this warehouse?')) {
                      handleDeactivate(selectedWarehouse.warehouse_id);
                    }
                  }}
                >
                  Deactivate
                </Button>
              </div>
            </div>
          ) : null
        )}

        {drawerMode === 'coverage' && selectedWarehouse && (
          <CoverageEditor
            warehouseId={selectedWarehouse.warehouse_id}
            warehouseName={selectedWarehouse.name}
            lat={selectedWarehouse.lat}
            lng={selectedWarehouse.lng}
            existingHexes={selectedWarehouse.h3_indexes || []}
            onSaved={() => {
              closeDrawer();
              fetchWarehouses();
            }}
          />
        )}

        {drawerMode === 'edit' && selectedWarehouse && (
          <WarehouseEditForm warehouse={selectedWarehouse} onSave={handleUpdate} onCancel={closeDrawer} />
        )}

        {drawerMode === 'staff' && selectedWarehouse && (
          <WarehouseStaffPanel
            warehouseId={selectedWarehouse.warehouse_id}
            warehouseName={selectedWarehouse.name}
          />
        )}
      </Drawer>
    </div>
  );
}

/* ── Warehouse Edit Form (inline) ─────────────────────────────────────────── */

const editFieldStyle = {
  background: 'var(--field-background)',
  color: 'var(--field-foreground)',
  border: '1px solid var(--field-border)',
  borderRadius: '8px',
};

function WarehouseEditForm({
  warehouse,
  onSave,
  onCancel,
}: {
  warehouse: WarehouseDetail;
  onSave: (updates: Record<string, unknown>) => Promise<void>;
  onCancel: () => void;
}) {
  const [name, setName] = useState(warehouse.name);
  const [address, setAddress] = useState(warehouse.address || '');
  const [lat, setLat] = useState(String(warehouse.lat));
  const [lng, setLng] = useState(String(warehouse.lng));
  const [radius, setRadius] = useState(String(warehouse.coverage_radius_km));
  const [isActive, setIsActive] = useState(warehouse.is_active);
  const [isOnShift, setIsOnShift] = useState(warehouse.is_on_shift);
  const [maxCapacity, setMaxCapacity] = useState('');
  const [disabledReason, setDisabledReason] = useState('');
  const [scheduleJson, setScheduleJson] = useState((warehouse as Record<string, unknown>).operating_schedule as string || '{}');
  const [saving, setSaving] = useState(false);

  // VU guardrail state
  const [inflightVU, setInflightVU] = useState<number | null>(null);
  const [vuLoading, setVuLoading] = useState(true);
  const [vuError, setVuError] = useState('');

  // Fetch inflight VU on mount
  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await apiFetch(`/v1/supplier/warehouse-inflight-vu?warehouse_id=${warehouse.warehouse_id}`);
        if (!res.ok) throw new Error('Failed to fetch VU');
        const data = await res.json();
        if (!cancelled) {
          setInflightVU(data.inflight_vu ?? 0);
          if (!maxCapacity) setMaxCapacity(String(data.max_capacity ?? 100));
        }
      } catch (err) {
        if (!cancelled) setVuError(err instanceof Error ? err.message : 'VU fetch failed');
      } finally {
        if (!cancelled) setVuLoading(false);
      }
    })();
    return () => { cancelled = true; };
  }, [warehouse.warehouse_id]); // eslint-disable-line react-hooks/exhaustive-deps

  const capacityNum = parseInt(maxCapacity, 10);
  const vuViolation = !isNaN(capacityNum) && inflightVU !== null && capacityNum < inflightVU;

  const handleSubmit = async () => {
    if (vuViolation) return;
    setSaving(true);
    const updates: Record<string, unknown> = {};
    if (name !== warehouse.name) updates.name = name;
    if (address !== (warehouse.address || '')) updates.address = address;
    if (parseFloat(lat) !== warehouse.lat) updates.lat = parseFloat(lat);
    if (parseFloat(lng) !== warehouse.lng) updates.lng = parseFloat(lng);
    if (parseFloat(radius) !== warehouse.coverage_radius_km) updates.coverage_radius_km = parseFloat(radius);
    if (isActive !== warehouse.is_active) updates.is_active = isActive;
    if (isOnShift !== warehouse.is_on_shift) updates.is_on_shift = isOnShift;
    if (maxCapacity) updates.max_capacity = parseInt(maxCapacity, 10);
    if (!isActive && disabledReason) updates.disabled_reason = disabledReason;

    // Schedule — always send if changed from initial
    const initialSchedule = (warehouse as Record<string, unknown>).operating_schedule as string || '{}';
    if (scheduleJson !== initialSchedule) {
      updates.operating_schedule = scheduleJson;
    }

    if (Object.keys(updates).length === 0) {
      onCancel();
      return;
    }

    await onSave(updates);
    setSaving(false);
  };

  return (
    <div className="p-6 space-y-5">
      <div>
        <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--muted)' }}>Name</label>
        <input className="w-full px-3 py-2.5 md-typescale-body-medium outline-none focus:ring-2 focus:ring-[var(--accent)]" style={editFieldStyle} value={name} onChange={e => setName(e.target.value)} />
      </div>
      <div>
        <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--muted)' }}>Address</label>
        <input className="w-full px-3 py-2.5 md-typescale-body-medium outline-none focus:ring-2 focus:ring-[var(--accent)]" style={editFieldStyle} value={address} onChange={e => setAddress(e.target.value)} />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--muted)' }}>Latitude</label>
          <input className="w-full px-3 py-2.5 md-typescale-body-medium outline-none focus:ring-2 focus:ring-[var(--accent)]" style={editFieldStyle} type="number" step="any" value={lat} onChange={e => setLat(e.target.value)} />
        </div>
        <div>
          <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--muted)' }}>Longitude</label>
          <input className="w-full px-3 py-2.5 md-typescale-body-medium outline-none focus:ring-2 focus:ring-[var(--accent)]" style={editFieldStyle} type="number" step="any" value={lng} onChange={e => setLng(e.target.value)} />
        </div>
      </div>
      <div>
        <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--muted)' }}>Coverage Radius (km)</label>
        <input className="w-full px-3 py-2.5 md-typescale-body-medium outline-none focus:ring-2 focus:ring-[var(--accent)]" style={editFieldStyle} type="number" value={radius} onChange={e => setRadius(e.target.value)} />
      </div>

      {/* Status toggles */}
      <div className="md-card md-card-elevated p-4 space-y-4">
        <p className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Warehouse Status</p>
        <div className="flex items-center justify-between">
          <span className="md-typescale-body-medium">Active</span>
          <button
            onClick={() => setIsActive(!isActive)}
            className="w-10 h-6 rounded-full transition-colors relative"
            style={{ background: isActive ? 'var(--success)' : 'var(--border)' }}
          >
            <div className="absolute top-1 w-4 h-4 rounded-full transition-transform" style={{ background: 'white', transform: isActive ? 'translateX(22px)' : 'translateX(4px)' }} />
          </button>
        </div>
        <div className="flex items-center justify-between">
          <span className="md-typescale-body-medium">On Shift</span>
          <button
            onClick={() => setIsOnShift(!isOnShift)}
            className="w-10 h-6 rounded-full transition-colors relative"
            style={{ background: isOnShift ? 'var(--success)' : 'var(--border)' }}
          >
            <div className="absolute top-1 w-4 h-4 rounded-full transition-transform" style={{ background: 'white', transform: isOnShift ? 'translateX(22px)' : 'translateX(4px)' }} />
          </button>
        </div>
      </div>

      {/* Disabled reason (shown when Active is off) */}
      {!isActive && (
        <div>
          <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--danger)' }}>Disabled Reason</label>
          <input className="w-full px-3 py-2.5 md-typescale-body-medium outline-none focus:ring-2 focus:ring-[var(--accent)]" style={editFieldStyle} placeholder="e.g. Maintenance, Out of stock" value={disabledReason} onChange={e => setDisabledReason(e.target.value)} />
        </div>
      )}

      {/* Max capacity + VU guardrail */}
      <div className="md-card md-card-elevated p-4 space-y-3">
        <p className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Capacity Guardrail</p>
        {vuLoading ? (
          <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Loading utilization...</p>
        ) : vuError ? (
          <p className="md-typescale-label-small" style={{ color: 'var(--danger)' }}>{vuError}</p>
        ) : (
          <div className="flex items-center gap-3">
            <span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Inflight VU:</span>
            <span className="md-typescale-body-medium font-medium" style={{ fontVariantNumeric: 'tabular-nums' }}>
              {inflightVU?.toFixed(1)}
            </span>
            {inflightVU !== null && capacityNum > 0 && (
              <span
                className="md-typescale-label-small px-2 py-0.5 rounded-full"
                style={{
                  background: (inflightVU / capacityNum) > 0.8
                    ? 'color-mix(in srgb, var(--danger) 15%, transparent)'
                    : 'color-mix(in srgb, var(--success) 15%, transparent)',
                  color: (inflightVU / capacityNum) > 0.8 ? 'var(--danger)' : 'var(--success)',
                }}
              >
                {((inflightVU / capacityNum) * 100).toFixed(0)}% utilized
              </span>
            )}
          </div>
        )}
        <div>
          <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--muted)' }}>
            Max Capacity (VU)
          </label>
          <input
            className="w-full px-3 py-2.5 md-typescale-body-medium outline-none focus:ring-2 focus:ring-[var(--accent)]"
            style={{
              ...editFieldStyle,
              ...(vuViolation ? { borderColor: 'var(--danger)' } : {}),
            }}
            type="number"
            min={inflightVU !== null ? Math.ceil(inflightVU) : 1}
            placeholder="100"
            value={maxCapacity}
            onChange={e => setMaxCapacity(e.target.value)}
          />
          {vuViolation && (
            <p className="md-typescale-label-small mt-1" style={{ color: 'var(--danger)' }}>
              Cannot set capacity below current inflight VU ({inflightVU?.toFixed(1)})
            </p>
          )}
        </div>
      </div>

      {/* Operating Schedule */}
      <div className="md-card md-card-elevated p-4">
        <OperatingScheduleEditor value={scheduleJson} onChange={setScheduleJson} />
      </div>

      <div className="flex gap-3 pt-4" style={{ borderTop: '1px solid var(--border)' }}>
        <Button className="button--primary flex-1" isLoading={saving} isDisabled={vuViolation} onPress={handleSubmit}>
          Save Changes
        </Button>
        <Button variant="bordered" className="flex-1" onPress={onCancel}>
          Cancel
        </Button>
      </div>
    </div>
  );
}
