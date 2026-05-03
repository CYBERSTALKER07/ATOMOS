'use client';

import { useEffect, useState, useRef } from 'react';
import { Button } from '@heroui/react';
import { readTokenFromCookie as getToken } from '@/lib/auth';
import Link from 'next/link';
import { usePagination } from '@/lib/usePagination';
import PaginationControls from '@/components/PaginationControls';
import Icon from '@/components/Icon';
import Drawer from '@/components/Drawer';
import { useToast } from '@/components/Toast';
import {
  buildSupplierFleetClearReturnsIdempotencyKey,
  buildSupplierFleetDriverAssignIdempotencyKey,
  buildSupplierFleetDriverCreateIdempotencyKey,
  buildSupplierFleetVehicleCreateIdempotencyKey,
  buildSupplierFleetVehicleDeactivateIdempotencyKey,
} from '../_shared/idempotency';

/* ── Types ─────────────────────────────────────────────────────────────── */

interface Driver {
  driver_id: string;
  name: string;
  phone: string;
  driver_type: 'IN_HOUSE' | 'CONTRACTOR';
  vehicle_type: string;
  license_plate: string;
  is_active: boolean;
  created_at: string;
  vehicle_id: string;
  vehicle_class: string;
  max_volume_vu: number;
  truck_status: string;
  offline_reason?: string;
  offline_reason_note?: string;
  offline_at?: string;
}

interface DriverDetail extends Driver {
  total_deliveries: number;
  current_location: string;
}

interface CreatedDriver {
  driver_id: string;
  name: string;
  phone: string;
  driver_type: string;
  vehicle_type: string;
  license_plate: string;
  pin: string;
}

interface Vehicle {
  vehicle_id: string;
  vehicle_class: string;
  class_label: string;
  label: string;
  license_plate: string;
  max_volume_vu: number;
  is_active: boolean;
  assigned_driver_id: string | null;
  assigned_driver_name: string | null;
  created_at: string;
}

interface CapacityInfo {
  route_id: string;
  max_volume_vu: number;
  used_volume_vu: number;
  free_volume_vu: number;
  pending_returns_vu: number;
}

interface ActiveMission {
  route_id: string;
}

type FleetTab = 'drivers' | 'vehicles';

const VEHICLE_CLASSES: { value: string; label: string; vu: number }[] = [
  { value: 'CLASS_A', label: 'Light / Van', vu: 50 },
  { value: 'CLASS_B', label: 'Medium Truck', vu: 150 },
  { value: 'CLASS_C', label: 'Heavy / Semi', vu: 400 },
];

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

/* ── Main Page ─────────────────────────────────────────────────────────── */

export default function FleetPage() {
  const [activeTab, setActiveTab] = useState<FleetTab>('drivers');
  const [drivers, setDrivers] = useState<Driver[]>([]);
  const [vehicles, setVehicles] = useState<Vehicle[]>([]);
  const [loading, setLoading] = useState(true);
  const driverPagination = usePagination(drivers, 25);
  const vehiclePagination = usePagination(vehicles, 25);
  const [showAdd, setShowAdd] = useState(false);
  const [showAddVehicle, setShowAddVehicle] = useState(false);
  const [createdPin, setCreatedPin] = useState<CreatedDriver | null>(null);
  const [selectedDriver, setSelectedDriver] = useState<DriverDetail | null>(null);
  const [pendingReturns, setPendingReturns] = useState<Record<string, number>>({});

  // Driver form state
  const [formName, setFormName] = useState('');
  const [formPhone, setFormPhone] = useState('+998');
  const [formType, setFormType] = useState<'IN_HOUSE' | 'CONTRACTOR'>('IN_HOUSE');
  const [formVehicleId, setFormVehicleId] = useState('');
  const [formPlate, setFormPlate] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState('');

  // Vehicle form state
  const [vehClass, setVehClass] = useState('CLASS_A');
  const [vehLabel, setVehLabel] = useState('');
  const [vehPlate, setVehPlate] = useState('');
  const [vehSubmitting, setVehSubmitting] = useState(false);
  const [vehError, setVehError] = useState('');
  const [vehVuMode, setVehVuMode] = useState<'class' | 'dimensions'>('class');
  const [vehLengthCM, setVehLengthCM] = useState('');
  const [vehWidthCM, setVehWidthCM] = useState('');
  const [vehHeightCM, setVehHeightCM] = useState('');

  const { toast } = useToast();
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => { fetchDrivers(); fetchVehicles(); fetchCapacity(); }, []);

  async function fetchCapacity() {
    try {
      const activeRes = await fetch(`${API}/v1/fleet/active`, {
        headers: { Authorization: `Bearer ${getToken()}` },
      });

      if (!activeRes.ok) {
        return;
      }

      const activeMissions: ActiveMission[] = await activeRes.json();
      const routeIds = [...new Set((activeMissions ?? []).map((mission) => mission.route_id).filter(Boolean))];

      if (routeIds.length === 0) {
        setPendingReturns({});
        return;
      }

      const capacityResponses = await Promise.all(
        routeIds.map(async (routeId) => {
          const res = await fetch(`${API}/v1/fleet/capacity?route_id=${encodeURIComponent(routeId)}`, {
            headers: { Authorization: `Bearer ${getToken()}` },
          });
          if (!res.ok) return null;
          return (await res.json()) as CapacityInfo;
        }),
      );

      const map: Record<string, number> = {};
      for (const capacity of capacityResponses) {
        if (capacity && capacity.pending_returns_vu > 0) {
          map[capacity.route_id] = capacity.pending_returns_vu;
        }
      }
      setPendingReturns(map);
    } catch (e) {
      console.error('Capacity fetch error:', e);
    }
  }

  async function fetchVehicles() {
    try {
      const res = await fetch(`${API}/v1/supplier/fleet/vehicles`, {
        headers: { Authorization: `Bearer ${getToken()}` },
      });
      if (res.ok) {
        const data = await res.json();
        setVehicles(data || []);
      }
    } catch (e) {
      console.error('Vehicle fetch error:', e);
    }
  }

  async function fetchDrivers() {
    try {
      const res = await fetch(`${API}/v1/supplier/fleet/drivers`, {
        headers: { Authorization: `Bearer ${getToken()}` },
      });
      if (res.ok) {
        const data = await res.json();
        setDrivers(data || []);
      }
    } catch (e) {
      console.error('Fleet fetch error:', e);
    } finally {
      setLoading(false);
    }
  }

  async function handleCreate() {
    setFormError('');
    if (!formName.trim() || formPhone.length < 5) {
      setFormError('Name and phone are required');
      return;
    }
    setSubmitting(true);
    try {
      const selectedVehicle = vehicles.find(v => v.vehicle_id === formVehicleId);
      const payload = {
        name: formName.trim(),
        phone: formPhone.trim(),
        driver_type: formType,
        vehicle_type: selectedVehicle?.class_label || '',
        license_plate: formPlate.trim(),
        vehicle_id: formVehicleId || undefined,
      };
      const res = await fetch(`${API}/v1/supplier/fleet/drivers`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${getToken()}`,
          'Idempotency-Key': buildSupplierFleetDriverCreateIdempotencyKey(payload),
        },
        body: JSON.stringify(payload),
      });
      if (!res.ok) {
        const errBody = await res.text();
        setFormError(errBody || 'Failed to create driver');
        return;
      }
      const created: CreatedDriver = await res.json();
      setCreatedPin(created);
      setShowAdd(false);
      resetForm();
      fetchDrivers();
    } catch {
      setFormError('Network error');
    } finally {
      setSubmitting(false);
    }
  }

  async function fetchDriverDetail(id: string) {
    try {
      const res = await fetch(`${API}/v1/supplier/fleet/drivers/${id}`, {
        headers: { Authorization: `Bearer ${getToken()}` },
      });
      if (res.ok) {
        setSelectedDriver(await res.json());
      }
    } catch (e) {
      console.error('Driver detail fetch error:', e);
    }
  }

  async function handleAssignVehicle(driverId: string, vehicleId: string) {
    try {
      const res = await fetch(`${API}/v1/supplier/fleet/drivers/${driverId}/assign-vehicle`, {
        method: 'PATCH',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${getToken()}`,
          'Idempotency-Key': buildSupplierFleetDriverAssignIdempotencyKey(driverId, vehicleId),
        },
        body: JSON.stringify({ vehicle_id: vehicleId }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({ error: 'Assignment failed' }));
        toast(body.error || 'Failed to assign vehicle' , 'error');
        return;
      }
      fetchDrivers();
      fetchVehicles();
    } catch {
      toast('Network error during vehicle assignment' , 'error');
    }
  }

  function resetForm() {
    setFormName('');
    setFormPhone('+998');
    setFormType('IN_HOUSE');
    setFormVehicleId('');
    setFormPlate('');
    setFormError('');
  }

  async function handleCreateVehicle() {
    setVehError('');
    setVehSubmitting(true);
    const body: Record<string, unknown> = {
      vehicle_class: vehClass,
      label: vehLabel.trim(),
      license_plate: vehPlate.trim(),
    };
    if (vehVuMode === 'dimensions' && vehLengthCM && vehWidthCM && vehHeightCM) {
      body.length_cm = parseFloat(vehLengthCM);
      body.width_cm = parseFloat(vehWidthCM);
      body.height_cm = parseFloat(vehHeightCM);
    }
    try {
      const res = await fetch(`${API}/v1/supplier/fleet/vehicles`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${getToken()}`,
          'Idempotency-Key': buildSupplierFleetVehicleCreateIdempotencyKey(body),
        },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const errBody = await res.text();
        setVehError(errBody || 'Failed to create vehicle');
        return;
      }
      setShowAddVehicle(false);
      setVehClass('CLASS_A');
      setVehLabel('');
      setVehPlate('');
      setVehVuMode('class');
      setVehLengthCM('');
      setVehWidthCM('');
      setVehHeightCM('');
      fetchVehicles();
    } catch {
      setVehError('Network error');
    } finally {
      setVehSubmitting(false);
    }
  }

  async function handleDeactivateVehicle(vehicleId: string) {
    try {
      const res = await fetch(`${API}/v1/supplier/fleet/vehicles/${vehicleId}`, {
        method: 'DELETE',
        headers: {
          Authorization: `Bearer ${getToken()}`,
          'Idempotency-Key': buildSupplierFleetVehicleDeactivateIdempotencyKey(vehicleId),
        },
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({ error: 'Deactivate vehicle failed' }));
        toast(body.error || 'Failed to deactivate vehicle', 'error');
        return;
      }
      fetchVehicles();
    } catch (e) {
      console.error('Deactivate vehicle error:', e);
      toast('Network error deactivating vehicle', 'error');
    }
  }

  async function handleClearReturns(vehicleId: string) {
    try {
      const res = await fetch(`${API}/v1/vehicle/${vehicleId}/clear-returns`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${getToken()}`,
          'Idempotency-Key': buildSupplierFleetClearReturnsIdempotencyKey(vehicleId),
        },
      });
      if (res.ok) {
        toast('Returns cleared — depot receipt confirmed', 'success');
        fetchCapacity();
      } else {
        const body = await res.json().catch(() => ({ error: 'Clear returns failed' }));
        toast(body.error || 'Failed to clear returns', 'error');
      }
    } catch {
      toast('Network error clearing returns', 'error');
    }
  }

  /* ── Render ──────────────────────────────────────────────────────────── */

  return (
    <div className="flex-1 overflow-y-auto p-8 relative" style={{ background: 'var(--background)' }}>
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <Link href="/supplier/dashboard" className="md-typescale-label-medium" style={{ color: 'var(--accent)', textDecoration: 'none' }}>
            ← Supplier Dashboard
          </Link>
          <h1 className="md-typescale-headline-medium mt-1" style={{ color: 'var(--foreground)' }}>
            Fleet Management
          </h1>
          <p className="md-typescale-body-medium" style={{ color: 'var(--muted)' }}>
            Provision drivers, register vehicles, manage fleet capacity.
          </p>
        </div>
        <div className="flex gap-2">
          {activeTab === 'vehicles' && (
            <Button variant="primary" onPress={() => { setShowAddVehicle(true); }}>
              + Add Vehicle
            </Button>
          )}
          {activeTab === 'drivers' && (
            <Button variant="primary" onPress={() => { setShowAdd(true); setCreatedPin(null); }}>
              + Add Driver
            </Button>
          )}
        </div>
      </div>

      {/* Tab Selector */}
      <div className="flex gap-1 mb-6 p-1" style={{ background: 'var(--surface)', borderRadius: '12px', width: 'fit-content' }}>
        {(['drivers', 'vehicles'] as FleetTab[]).map(tab => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className="px-4 py-2 md-typescale-label-large transition-all"
            style={{
              borderRadius: '10px',
              background: activeTab === tab ? 'var(--accent)' : 'transparent',
              color: activeTab === tab ? 'var(--accent-foreground)' : 'var(--muted)',
            }}
          >
            {tab === 'drivers' ? `Drivers (${drivers.length})` : `Vehicles (${vehicles.length})`}
          </button>
        ))}
      </div>

      {/* KPI Row */}
      {activeTab === 'drivers' ? (
        <div className="grid grid-cols-4 gap-4 mb-6">
          <div className="md-card-filled p-4" style={{ background: 'var(--surface)' }}>
            <p className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Total Drivers</p>
            <p className="md-typescale-headline-small" style={{ color: 'var(--foreground)' }}>{drivers.length}</p>
          </div>
          <div className="md-card-filled p-4" style={{ background: 'var(--surface)' }}>
            <p className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>In-House</p>
            <p className="md-typescale-headline-small" style={{ color: 'var(--accent-soft)' }}>
              {drivers.filter(d => d.driver_type === 'IN_HOUSE').length}
            </p>
          </div>
          <div className="md-card-filled p-4" style={{ background: 'var(--surface)' }}>
            <p className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Contractors</p>
            <p className="md-typescale-headline-small" style={{ color: 'var(--warning)' }}>
              {drivers.filter(d => d.driver_type === 'CONTRACTOR').length}
            </p>
          </div>
          <div className="md-card-filled p-4" style={{ background: 'var(--surface)' }}>
            <p className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Offline</p>
            <p className="md-typescale-headline-small" style={{ color: 'var(--danger)' }}>
              {drivers.filter(d => !d.is_active && d.offline_reason).length}
            </p>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-3 gap-4 mb-6">
          <div className="md-card-filled p-4" style={{ background: 'var(--surface)' }}>
            <p className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Total Vehicles</p>
            <p className="md-typescale-headline-small" style={{ color: 'var(--foreground)' }}>{vehicles.length}</p>
          </div>
          <div className="md-card-filled p-4" style={{ background: 'var(--surface)' }}>
            <p className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Total Capacity</p>
            <p className="md-typescale-headline-small" style={{ color: 'var(--accent-soft)' }}>
              {vehicles.reduce((sum, v) => sum + v.max_volume_vu, 0).toFixed(0)} VU
            </p>
          </div>
          <div className="md-card-filled p-4" style={{ background: 'var(--surface)' }}>
            <p className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Assigned</p>
            <p className="md-typescale-headline-small" style={{ color: 'var(--success)' }}>
              {vehicles.filter(v => v.assigned_driver_id).length} / {vehicles.length}
            </p>
          </div>
        </div>
      )}

      {/* ── Vehicles Tab ─────────────────────────────────────────────── */}
      {activeTab === 'vehicles' && (
        <>
          {vehicles.length === 0 ? (
            <div className="text-center py-20">
              <p className="md-typescale-title-medium" style={{ color: 'var(--muted)' }}>No vehicles registered yet</p>
              <p className="md-typescale-body-medium mt-2" style={{ color: 'var(--muted)' }}>
                Click &quot;+ Add Vehicle&quot; to register your first truck.
              </p>
            </div>
          ) : (
            <div className="md-card-outlined overflow-hidden" style={{ borderRadius: '12px' }}>
              <table className="md-table w-full">
                <thead>
                  <tr>
                    <th className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Class</th>
                    <th className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Label</th>
                    <th className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Plate</th>
                    <th className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Capacity</th>
                    <th className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Assigned Driver</th>
                    <th className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Status</th>
                    <th className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}></th>
                  </tr>
                </thead>
                <tbody>
                  {vehiclePagination.pageItems.map(v => (
                    <tr key={v.vehicle_id}>
                      <td>
                        <span
                          className="md-typescale-label-small px-2 py-0.5"
                          style={{
                            borderRadius: '99px',
                            background: v.vehicle_class === 'CLASS_C' ? 'color-mix(in srgb, var(--accent) 15%, transparent)'
                              : v.vehicle_class === 'CLASS_B' ? 'color-mix(in srgb, var(--accent-soft) 15%, transparent)'
                              : 'color-mix(in srgb, var(--muted) 15%, transparent)',
                            color: v.vehicle_class === 'CLASS_C' ? 'var(--accent)'
                              : v.vehicle_class === 'CLASS_B' ? 'var(--accent-soft)'
                              : 'var(--muted)',
                          }}
                        >
                          {v.vehicle_class.replace('CLASS_', '')}
                        </span>
                        <span className="md-typescale-body-small ml-2" style={{ color: 'var(--muted)' }}>{v.class_label}</span>
                      </td>
                      <td className="md-typescale-body-medium" style={{ color: 'var(--foreground)' }}>{v.label || '—'}</td>
                      <td className="md-typescale-body-medium font-mono" style={{ color: 'var(--muted)' }}>{v.license_plate || '—'}</td>
                      <td className="md-typescale-body-medium font-mono" style={{ color: 'var(--accent)' }}>{v.max_volume_vu} VU</td>
                      <td className="md-typescale-body-medium" style={{ color: v.assigned_driver_name ? 'var(--foreground)' : 'var(--muted)' }}>
                        {v.assigned_driver_name || 'Unassigned'}
                      </td>
                      <td>
                        <span
                          className="md-typescale-label-small px-2 py-0.5"
                          style={{
                            borderRadius: '99px',
                            background: v.is_active ? 'color-mix(in srgb, var(--success) 15%, transparent)' : 'color-mix(in srgb, var(--danger) 15%, transparent)',
                            color: v.is_active ? 'var(--success)' : 'var(--danger)',
                          }}
                        >
                          {v.is_active ? 'Active' : 'Inactive'}
                        </span>
                      </td>
                      <td>
                        <div className="flex items-center gap-2 flex-wrap">
                        {v.is_active && (
                          <Button
                            variant="ghost"
                            className="text-danger md-typescale-label-small"
                            onPress={() => handleDeactivateVehicle(v.vehicle_id)}
                          >
                            Deactivate
                          </Button>
                        )}
                        {v.is_active && v.assigned_driver_id && (() => {
                          // find pending returns for this vehicle's active route
                          const vuPending = Object.values(pendingReturns).reduce((a, b) => a + b, 0);
                          // per-vehicle pending check: re-fetch with vehicle granularity if needed
                          // For now surface if any pending returns exist on fleet level
                          return vuPending > 0 ? (
                            <Button
                              variant="outline"
                              className="text-warning border-warning md-typescale-label-small px-3 py-1"
                              onPress={() => handleClearReturns(v.vehicle_id)}
                              aria-label={`${vuPending.toFixed(1)} VU pending depot return`}
                            >
                              Clear Returns · {vuPending.toFixed(1)} VU
                            </Button>
                          ) : null;
                        })()}
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
              <PaginationControls pagination={vehiclePagination} />
            </div>
          )}
        </>
      )}

      {/* ── Drivers Tab ──────────────────────────────────────────────── */}
      {activeTab === 'drivers' && (
        <>
      {/* Driver Table */}
      {loading ? (
        <div className="flex items-center justify-center py-20">
          <div className="w-8 h-8 border-3 rounded-full animate-spin" style={{ borderColor: 'var(--border)', borderTopColor: 'var(--accent)' }} />
        </div>
      ) : drivers.length === 0 ? (
        <div className="text-center py-20">
          <p className="md-typescale-title-medium" style={{ color: 'var(--muted)' }}>No drivers provisioned yet</p>
          <p className="md-typescale-body-medium mt-2" style={{ color: 'var(--muted)' }}>
            Click &quot;+ Add Driver&quot; to provision your first fleet member.
          </p>
        </div>
      ) : (
        <div className="md-card-outlined overflow-hidden" style={{ borderRadius: '12px' }}>
          <table className="md-table w-full">
            <thead>
              <tr>
                <th className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Name</th>
                <th className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Phone</th>
                <th className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Type</th>
                <th className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Assigned Vehicle</th>
                <th className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Status</th>
              </tr>
            </thead>
            <tbody>
              {driverPagination.pageItems.map(d => (
                <tr
                  key={d.driver_id}
                  onClick={() => fetchDriverDetail(d.driver_id)}
                  className="cursor-pointer"
                  style={{ transition: 'background 0.15s' }}
                  onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface)')}
                  onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                >
                  <td className="md-typescale-body-medium" style={{ color: 'var(--foreground)' }}>{d.name}</td>
                  <td className="md-typescale-body-medium font-mono" style={{ color: 'var(--muted)' }}>{d.phone}</td>
                  <td>
                    <DriverTypeBadge type={d.driver_type} />
                  </td>
                  <td onClick={e => e.stopPropagation()}>
                    <select
                      value={d.vehicle_id || ''}
                      onChange={e => handleAssignVehicle(d.driver_id, e.target.value)}
                      className="md-typescale-body-small px-2 py-1 rounded-lg outline-none cursor-pointer"
                      style={{
                        background: 'var(--surface)',
                        color: 'var(--foreground)',
                        border: '1px solid var(--border)',
                        maxWidth: '200px',
                      }}
                    >
                      <option value="">— Unassigned —</option>
                      {vehicles.filter(v => v.is_active).map(v => (
                        <option key={v.vehicle_id} value={v.vehicle_id}>
                          {v.class_label} · {v.license_plate || v.vehicle_id.slice(0, 8)}
                        </option>
                      ))}
                    </select>
                  </td>
                  <td>
                    <span
                      className="md-typescale-label-small px-2 py-0.5"
                      style={{
                        borderRadius: '99px',
                        background: d.is_active ? 'color-mix(in srgb, var(--success) 15%, transparent)' : 'color-mix(in srgb, var(--danger) 15%, transparent)',
                        color: d.is_active ? 'var(--success)' : 'var(--danger)',
                      }}
                    >
                      {d.is_active ? 'Active' : 'Inactive'}
                    </span>
                    {!d.is_active && d.offline_reason && (
                      <span
                        className="md-typescale-label-small ml-2 px-2 py-0.5"
                        style={{
                          borderRadius: '99px',
                          background: d.offline_reason === 'TRUCK_DAMAGED'
                            ? 'color-mix(in srgb, var(--warning) 15%, transparent)'
                            : 'color-mix(in srgb, var(--muted) 15%, transparent)',
                          color: d.offline_reason === 'TRUCK_DAMAGED' ? 'var(--warning)' : 'var(--muted)',
                        }}
                        title={d.offline_reason_note || undefined}
                      >
                        {d.offline_reason.replace(/_/g, ' ')}
                      </span>
                    )}
                    {d.truck_status === 'MAINTENANCE' && (
                      <span
                        className="md-typescale-label-small ml-2 px-2 py-0.5"
                        style={{
                          borderRadius: '99px',
                          background: 'color-mix(in srgb, var(--danger) 15%, transparent)',
                          color: 'var(--danger)',
                        }}
                      >
                        MAINTENANCE
                      </span>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <PaginationControls pagination={driverPagination} />
        </div>
      )}
      </>
      )}

      {/* ── Slide-out: Add Driver ────────────────────────────────────── */}
      <Drawer open={showAdd} onClose={() => { setShowAdd(false); resetForm(); }} title="Add Driver">
          <div className="p-6 flex flex-col gap-4">
              <div>
                <label className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Name *</label>
                <input
                  className="md-input-outlined w-full mt-1"
                  value={formName}
                  onChange={e => setFormName(e.target.value)}
                  placeholder="Driver full name"
                />
              </div>

              <div>
                <label className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Phone (+998) *</label>
                <input
                  className="md-input-outlined w-full mt-1 font-mono"
                  value={formPhone}
                  onChange={e => setFormPhone(e.target.value)}
                  placeholder="+998901234567"
                />
              </div>

              <div>
                <label className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Driver Type</label>
                <div className="flex gap-2 mt-1">
                  {(['IN_HOUSE', 'CONTRACTOR'] as const).map(t => (
                    <button
                      key={t}
                      onClick={() => setFormType(t)}
                      className={formType === t ? 'md-chip md-chip-selected' : 'md-chip'}
                      style={{
                        borderColor: formType === t
                          ? (t === 'IN_HOUSE' ? 'var(--accent-soft)' : 'var(--warning)')
                          : 'var(--border)',
                        background: formType === t
                          ? (t === 'IN_HOUSE' ? 'color-mix(in srgb, var(--accent-soft) 12%, transparent)' : 'color-mix(in srgb, var(--warning) 12%, transparent)')
                          : 'transparent',
                        color: formType === t
                          ? (t === 'IN_HOUSE' ? 'var(--accent-soft)' : 'var(--warning)')
                          : 'var(--muted)',
                      }}
                    >
                      {t === 'IN_HOUSE' ? 'In-House' : 'Contractor'}
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <label className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Assign Vehicle</label>
                <select
                  className="md-input-outlined w-full mt-1"
                  value={formVehicleId}
                  onChange={e => setFormVehicleId(e.target.value)}
                >
                  <option value="">— No vehicle —</option>
                  {vehicles.filter(v => v.is_active).map(v => (
                    <option key={v.vehicle_id} value={v.vehicle_id}>
                      {v.vehicle_class.replace('CLASS_', '')} · {v.label || v.class_label} · {v.license_plate || '—'} ({v.max_volume_vu} VU)
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>License Plate / Tracking Number</label>
                <input
                  className="md-input-outlined w-full mt-1 font-mono"
                  value={formPlate}
                  onChange={e => setFormPlate(e.target.value)}
                  placeholder="01 A 123 BC"
                />
              </div>

              {formError && (
                <p className="md-typescale-body-small" style={{ color: 'var(--danger)' }}>{formError}</p>
              )}

              <Button
                variant="primary"
                fullWidth
                className="mt-2"
                onPress={handleCreate}
                isDisabled={submitting}
              >
                {submitting ? 'Provisioning…' : 'Provision Driver'}
              </Button>
            </div>
      </Drawer>

      {/* ── PIN Reveal Modal ─────────────────────────────────────────── */}
      {createdPin && (
        <div className="fixed inset-0 z-50 flex items-center justify-center" style={{ background: 'rgba(0,0,0,0.5)' }}>
          <div
            className="p-8 flex flex-col items-center gap-4 max-w-sm w-full"
            style={{
              background: 'var(--surface)',
              borderRadius: '28px',
              animation: 'md-scale-in 0.3s ease',
            }}
          >
            <div
              className="w-14 h-14 flex items-center justify-center"
              style={{ background: 'color-mix(in srgb, var(--success) 15%, transparent)', borderRadius: '9999px' }}
            >
              <svg width="28" height="28" viewBox="0 0 24 24" fill="var(--success)">
                <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z" />
              </svg>
            </div>

            <h3 className="md-typescale-title-large text-center" style={{ color: 'var(--foreground)' }}>
              Driver Provisioned
            </h3>

            <p className="md-typescale-body-medium text-center" style={{ color: 'var(--muted)' }}>
              {createdPin.name} — {createdPin.phone}
            </p>

            <div
              className="w-full p-4 text-center"
              style={{
                background: 'var(--surface)',
                borderRadius: '16px',
                border: '2px dashed var(--border)',
              }}
            >
              <p className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>
                LOGIN PIN
              </p>
              <p className="font-mono mt-1" style={{ fontSize: '32px', letterSpacing: '8px', color: 'var(--accent)', fontWeight: 700 }}>
                {createdPin.pin}
              </p>
            </div>

            <div
              className="w-full p-3"
              style={{
                background: 'color-mix(in srgb, var(--warning) 12%, transparent)',
                borderRadius: '12px',
                border: '1px solid color-mix(in srgb, var(--warning) 30%, transparent)',
              }}
            >
              <p className="md-typescale-label-medium text-center" style={{ color: 'var(--warning)' }}>
                <Icon name="warning" size={16} className="inline-block align-text-bottom mr-1" /> Copy this PIN. It will not be shown again.
              </p>
            </div>

            <Button
              variant="primary"
              fullWidth
              className="mt-2"
              onPress={() => setCreatedPin(null)}
            >
              Done
            </Button>
          </div>
        </div>
      )}

      {/* ── Driver Detail Slide-out ──────────────────────────────────── */}
      <Drawer open={!!selectedDriver} onClose={() => setSelectedDriver(null)} title="Driver Detail">
          {selectedDriver && (
            <div className="p-6 flex flex-col gap-5">
            <div className="flex items-center gap-4">
              <div
                className="w-14 h-14 flex items-center justify-center md-typescale-title-large"
                style={{
                  background: 'var(--accent-soft)',
                  color: 'var(--accent-soft-foreground)',
                  borderRadius: '9999px',
                }}
              >
                {selectedDriver.name.charAt(0).toUpperCase()}
              </div>
              <div>
                <p className="md-typescale-title-medium" style={{ color: 'var(--foreground)' }}>{selectedDriver.name}</p>
                <p className="md-typescale-body-medium font-mono" style={{ color: 'var(--muted)' }}>{selectedDriver.phone}</p>
              </div>
              <div className="ml-auto">
                <DriverTypeBadge type={selectedDriver.driver_type} />
              </div>
            </div>

            <div className="md-divider" />

            <div className="grid grid-cols-2 gap-4">
              <DetailCell label="Vehicle Type" value={selectedDriver.vehicle_type} />
              <DetailCell label="License Plate" value={selectedDriver.license_plate} mono />
              <DetailCell label="Total Deliveries" value={String(selectedDriver.total_deliveries)} />
              <DetailCell label="Status" value={selectedDriver.is_active ? 'Active' : 'Inactive'} />
              <DetailCell label="Provisioned" value={new Date(selectedDriver.created_at).toLocaleDateString()} />
              <DetailCell label="Driver ID" value={selectedDriver.driver_id} mono />
            </div>

            {selectedDriver.current_location && (
              <>
                <div className="md-divider" />
                <DetailCell label="Current Location" value={selectedDriver.current_location} />
              </>
            )}
            </div>
          )}
      </Drawer>

      {/* ── Slide-out: Add Vehicle ───────────────────────────────────── */}
      <Drawer open={showAddVehicle} onClose={() => setShowAddVehicle(false)} title="Add Vehicle">
          <div className="p-6 flex flex-col gap-4">
              <div>
                <div className="flex items-center justify-between">
                  <label className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Vehicle Class *</label>
                  <div className="flex rounded overflow-hidden text-[10px] font-medium" style={{ border: '1px solid var(--border)' }}>
                    <button type="button" onClick={() => setVehVuMode('class')} className="px-3 py-1 transition-colors"
                      style={vehVuMode === 'class' ? { background: 'var(--accent)', color: 'var(--accent-foreground)' } : { background: 'var(--background)', color: 'var(--muted)' }}>
                      Class
                    </button>
                    <button type="button" onClick={() => setVehVuMode('dimensions')} className="px-3 py-1 transition-colors"
                      style={vehVuMode === 'dimensions' ? { background: 'var(--accent)', color: 'var(--accent-foreground)' } : { background: 'var(--background)', color: 'var(--muted)' }}>
                      L×W×H
                    </button>
                  </div>
                </div>
                <select
                  className="md-input-outlined w-full mt-1"
                  value={vehClass}
                  onChange={e => setVehClass(e.target.value)}
                >
                  {VEHICLE_CLASSES.map(c => (
                    <option key={c.value} value={c.value}>
                      {c.value.replace('CLASS_', '')} — {c.label} ({c.vu} VU)
                    </option>
                  ))}
                </select>
                {vehVuMode === 'class' ? (
                  <p className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>
                    Capacity: <strong>{VEHICLE_CLASSES.find(c => c.value === vehClass)?.vu ?? 0} VU</strong>
                  </p>
                ) : (
                  <div className="mt-2 space-y-2">
                    <div className="grid grid-cols-3 gap-2">
                      {([['L', vehLengthCM, setVehLengthCM], ['W', vehWidthCM, setVehWidthCM], ['H', vehHeightCM, setVehHeightCM]] as [string, string, React.Dispatch<React.SetStateAction<string>>][]).map(([axis, val, setter]) => (
                        <div key={axis}>
                          <label className="block md-typescale-label-small mb-1" style={{ color: 'var(--muted)' }}>{axis} (cm)</label>
                          <input
                            type="number" min="0.1" step="0.1"
                            value={val}
                            className="md-input-outlined w-full font-mono"
                            onChange={e => setter(e.target.value)}
                            placeholder="0"
                          />
                        </div>
                      ))}
                    </div>
                    {(() => {
                      const l = parseFloat(vehLengthCM), w = parseFloat(vehWidthCM), h = parseFloat(vehHeightCM);
                      const vu = (l > 0 && w > 0 && h > 0) ? ((l * w * h) / 5000).toFixed(2) : null;
                      return (
                        <div className="flex items-center justify-between px-2 py-1 rounded" style={{ background: 'var(--accent-soft)' }}>
                          <span className="md-typescale-label-small" style={{ color: 'var(--accent-soft-foreground)' }}>Computed VU</span>
                          <span className="md-typescale-label-large font-semibold" style={{ color: 'var(--accent-soft-foreground)' }}>{vu ? `${vu} VU` : '—'}</span>
                        </div>
                      );
                    })()}
                  </div>
                )}
              </div>

              <div>
                <label className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Label</label>
                <input
                  className="md-input-outlined w-full mt-1"
                  value={vehLabel}
                  onChange={e => setVehLabel(e.target.value)}
                  placeholder="e.g. Transit Van #3"
                />
              </div>

              <div>
                <label className="md-typescale-label-medium" style={{ color: 'var(--muted)' }}>License Plate</label>
                <input
                  className="md-input-outlined w-full mt-1 font-mono"
                  value={vehPlate}
                  onChange={e => setVehPlate(e.target.value)}
                  placeholder="01 A 123 BC"
                />
              </div>

              {vehError && (
                <p className="md-typescale-body-small" style={{ color: 'var(--danger)' }}>{vehError}</p>
              )}

              <Button
                variant="primary"
                fullWidth
                className="mt-2"
                onPress={handleCreateVehicle}
                isDisabled={vehSubmitting}
              >
                {vehSubmitting ? 'Creating…' : 'Register Vehicle'}
              </Button>
            </div>
      </Drawer>
    </div>
  );
}

/* ── Components ────────────────────────────────────────────────────────── */

function DriverTypeBadge({ type }: { type: string }) {
  const isInHouse = type === 'IN_HOUSE';
  return (
    <span
      className="md-typescale-label-small px-2.5 py-0.5 inline-flex items-center gap-1"
      style={{
        borderRadius: '99px',
        ...(isInHouse
          ? {
              background: 'color-mix(in srgb, var(--accent-soft) 15%, transparent)',
              color: 'var(--accent-soft)',
            }
          : {
              background: 'transparent',
              border: '1px solid var(--warning)',
              color: 'var(--warning)',
            }),
      }}
    >
      {isInHouse ? 'In-House' : 'Contractor'}
    </span>
  );
}

function DetailCell({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div>
      <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>{label}</p>
      <p
        className={`md-typescale-body-medium mt-0.5 ${mono ? 'font-mono' : ''}`}
        style={{ color: 'var(--foreground)' }}
      >
        {value || '—'}
      </p>
    </div>
  );
}
