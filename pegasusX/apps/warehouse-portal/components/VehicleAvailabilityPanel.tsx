'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from 'react';
import type { WarehouseFleetVehicle, WarehouseVehicleUnavailableReason } from '@pegasusx/types';
import { warehouseUpdateVehicleKey } from '@pegasusx/api-core';
import { apiFetch } from '@/lib/auth';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import {
  formatUnavailableReason,
  VEHICLE_UNAVAILABLE_REASONS,
  vehicleStatusLabel,
} from '@/lib/vehicle-fleet';

type VehicleAvailabilityPanelProps = {
  vehicle: WarehouseFleetVehicle;
  onUpdated?: (vehicle: WarehouseFleetVehicle) => void;
  compact?: boolean;
};

export default function VehicleAvailabilityPanel({
  vehicle,
  onUpdated,
  compact = false,
}: VehicleAvailabilityPanelProps) {
  const t = usePortalT();
  const [reason, setReason] = useState<WarehouseVehicleUnavailableReason>(
    vehicle.unavailable_reason || 'MANUAL_HOLD',
  );
  const [note, setNote] = useState(vehicle.unavailable_note || '');
  const [mutating, setMutating] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    setReason(vehicle.unavailable_reason || 'MANUAL_HOLD');
    setNote(vehicle.unavailable_note || '');
  }, [vehicle.unavailable_reason, vehicle.unavailable_note, vehicle.vehicle_id, vehicle.is_active]);

  async function toggleAvailability(nextActive: boolean) {
    setMutating(true);
    setError('');
    const warehouseId = warehouseHomeNodeId() || 'warehouse';
    try {
      const unavailableReason = reason || vehicle.unavailable_reason || 'MANUAL_HOLD';
      const unavailableNote = note.trim();
      const body: Record<string, unknown> = nextActive
        ? { is_active: true }
        : {
            is_active: false,
            unavailable_reason: unavailableReason,
            ...(unavailableReason === 'OTHER' && unavailableNote ? { unavailable_note: unavailableNote } : {}),
          };
      const res = await apiFetch(`/v1/warehouse/ops/vehicles/${vehicle.vehicle_id}`, {
        method: 'PATCH',
        body: JSON.stringify(body),
        headers: {
          'Idempotency-Key': warehouseUpdateVehicleKey(
            warehouseId,
            vehicle.vehicle_id,
            nextActive,
            nextActive ? undefined : unavailableReason,
          ),
        },
      });
      if (!res.ok) {
        throw new Error('Unable to update truck availability');
      }
      onUpdated?.({
        ...vehicle,
        is_active: nextActive,
        status: nextActive ? 'ACTIVE' : 'UNAVAILABLE',
        unavailable_reason: nextActive ? undefined : unavailableReason,
        unavailable_note: nextActive ? undefined : unavailableNote,
      });
    } catch (updateError) {
      setError(updateError instanceof Error ? updateError.message : 'Update failed');
    } finally {
      setMutating(false);
    }
  }

  return (
    <div className={`rounded-xl border border-(--border) ${compact ? 'p-3' : 'p-4'}`} style={{ background: 'var(--surface)' }}>
      <div className="flex items-center justify-between gap-3 mb-3">
        <div>
          <h3 className={`font-semibold ${compact ? 'text-sm' : 'text-base'}`}>{t("warehouse_portal.vehicle_availability_panel.text.availability")}</h3>
          <p className="text-xs text-(--muted)">{t("warehouse_portal.vehicle_availability_panel.text.dispatch_and_smart_suggest_exclude_unavailable_trucks_immediatel")}</p>
        </div>
        <span className={`status-chip ${vehicle.is_active ? 'status-chip--stable' : 'status-chip--draft'}`}>
          {vehicleStatusLabel(vehicle.is_active)}
        </span>
      </div>

      {!vehicle.is_active && (vehicle.unavailable_reason || vehicle.unavailable_note) && (
        <p className="text-sm mb-3" style={{ color: 'var(--warning)' }}>
          {formatUnavailableReason(vehicle.unavailable_reason, vehicle.unavailable_note)}
        </p>
      )}

      {vehicle.is_active && (
        <div className={`space-y-2 mb-3 ${compact ? '' : 'max-w-md'}`}>
          <label className="text-xs font-medium text-(--muted) uppercase tracking-wide">{t("warehouse_portal.vehicle_availability_panel.text.unavailable_reason")}</label>
          <select
            value={reason}
            onChange={event => setReason(event.target.value as WarehouseVehicleUnavailableReason)}
            disabled={mutating}
            className="w-full rounded-lg border px-3 py-2 text-sm"
            style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
          >
            {VEHICLE_UNAVAILABLE_REASONS.map(option => (
              <option key={option} value={option}>{formatUnavailableReason(option)}</option>
            ))}
          </select>
          {reason === 'OTHER' && (
            <input
              type="text"
              placeholder={t("warehouse_portal.vehicle_availability_panel.text.custom_reason_required")}
              value={note}
              onChange={event => setNote(event.target.value)}
              disabled={mutating}
              className="w-full rounded-lg border px-3 py-2 text-sm"
              style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
            />
          )}
        </div>
      )}

      {error && (
        <p className="text-sm mb-3" style={{ color: 'var(--danger)' }}>{error}</p>
      )}

      <button
        type="button"
        disabled={mutating || (vehicle.is_active && reason === 'OTHER' && !note.trim())}
        onClick={() => void toggleAvailability(!vehicle.is_active)}
        className="rounded-lg px-4 py-2 text-sm font-medium disabled:opacity-50"
        style={{
          background: vehicle.is_active ? 'color-mix(in srgb, var(--warning) 15%, transparent)' : 'color-mix(in srgb, var(--success) 15%, transparent)',
          color: vehicle.is_active ? 'var(--warning)' : 'var(--success)',
        }}
      >
        {mutating ? 'Updating…' : vehicle.is_active ? 'Mark unavailable' : 'Restore for dispatch'}
      </button>
    </div>
  );
}
