'use client';

import { useState } from 'react';
import {
  adminOrderAssignKey,
  adminOrderStatusPatchKey,
  ApiError,
} from '@pegasusx/api-client';
import type { SupplierOrder } from '@pegasusx/types';
import { createSupplierApi } from '@/lib/api';

const STATUS_OPTIONS = [
  'PENDING',
  'LOADED',
  'IN_TRANSIT',
  'ARRIVED',
  'DELAYED',
  'CANCELLED',
  'COMPLETED',
] as const;

const supplierApi = createSupplierApi();

interface AdminOrderOpsPanelProps {
  order: SupplierOrder;
  busy: boolean;
  canAssign: boolean;
  canPatchStatus: boolean;
  onBusyChange: (orderId: string | null) => void;
  onSuccess: () => void;
  onError: (message: string) => void;
}

export function AdminOrderOpsPanel({
  order,
  busy,
  canAssign,
  canPatchStatus,
  onBusyChange,
  onSuccess,
  onError,
}: AdminOrderOpsPanelProps) {
  const [open, setOpen] = useState(false);
  const [driverId, setDriverId] = useState(order.driver_id ?? '');
  const [routeId, setRouteId] = useState(order.route_id ?? '');
  const [vehicleId, setVehicleId] = useState(order.vehicle_id ?? '');
  const [nextStatus, setNextStatus] = useState(order.status);
  const [statusReason, setStatusReason] = useState('');

  const assignDriver = async () => {
    const trimmedDriver = driverId.trim();
    const trimmedRoute = routeId.trim();
    if (!trimmedDriver || !trimmedRoute) {
      onError('Driver ID and route ID are required for assignment.');
      return;
    }
    onBusyChange(order.order_id);
    try {
      await supplierApi.assignOrder(
        order.order_id,
        {
          driver_id: trimmedDriver,
          route_id: trimmedRoute,
          vehicle_id: vehicleId.trim() || undefined,
        },
        adminOrderAssignKey(order.order_id, trimmedDriver),
      );
      onSuccess();
    } catch (err) {
      onError(err instanceof ApiError ? err.message : 'assign_failed');
    } finally {
      onBusyChange(null);
    }
  };

  const patchStatus = async () => {
    const status = nextStatus.trim();
    if (!status) {
      onError('Select a target status.');
      return;
    }
    onBusyChange(order.order_id);
    try {
      await supplierApi.patchOrderStatus(
        order.order_id,
        { status, reason: statusReason.trim() || undefined },
        adminOrderStatusPatchKey(order.order_id, status),
      );
      onSuccess();
    } catch (err) {
      onError(err instanceof ApiError ? err.message : 'status_patch_failed');
    } finally {
      onBusyChange(null);
    }
  };

  return (
    <div className="mt-2">
      <button
        type="button"
        className="md-btn md-btn-text md-typescale-label-medium px-0"
        onClick={() => setOpen((value) => !value)}
      >
        {open ? 'Hide admin ops' : 'Admin ops'}
      </button>
      {open ? (
        <div className="mt-3 space-y-4 rounded-2xl border border-[var(--color-md-outline-variant)] bg-[var(--color-md-surface-container-low)] p-4">
          {canAssign ? (
          <div>
            <p className="md-typescale-label-medium mb-2">Assign driver</p>
            <div className="grid gap-2 sm:grid-cols-3">
              <input
                className="md-input-outlined w-full px-3 py-2"
                placeholder="Driver ID"
                value={driverId}
                onChange={(e) => setDriverId(e.target.value)}
                disabled={busy}
              />
              <input
                className="md-input-outlined w-full px-3 py-2"
                placeholder="Route ID"
                value={routeId}
                onChange={(e) => setRouteId(e.target.value)}
                disabled={busy}
              />
              <input
                className="md-input-outlined w-full px-3 py-2"
                placeholder="Vehicle ID (optional)"
                value={vehicleId}
                onChange={(e) => setVehicleId(e.target.value)}
                disabled={busy}
              />
            </div>
            <button
              type="button"
              className="md-btn md-btn-tonal md-typescale-label-medium mt-2 px-4 py-2"
              disabled={busy}
              onClick={() => void assignDriver()}
            >
              {busy ? 'Working…' : 'Assign'}
            </button>
          </div>
          ) : null}

          {canPatchStatus ? (
          <div>
            <p className="md-typescale-label-medium mb-2">Patch status</p>
            <div className="grid gap-2 sm:grid-cols-2">
              <select
                className="md-input-outlined w-full px-3 py-2"
                value={nextStatus}
                onChange={(e) => setNextStatus(e.target.value)}
                disabled={busy}
              >
                {STATUS_OPTIONS.map((status) => (
                  <option key={status} value={status}>
                    {status}
                  </option>
                ))}
              </select>
              <input
                className="md-input-outlined w-full px-3 py-2"
                placeholder="Reason (optional)"
                value={statusReason}
                onChange={(e) => setStatusReason(e.target.value)}
                disabled={busy}
              />
            </div>
            <button
              type="button"
              className="md-btn md-btn-outlined md-typescale-label-medium mt-2 px-4 py-2"
              disabled={busy}
              onClick={() => void patchStatus()}
            >
              {busy ? 'Working…' : 'Apply status'}
            </button>
          </div>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
