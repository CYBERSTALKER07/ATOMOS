'use client';

import type { Dispatch, SetStateAction } from 'react';
import type {
  WarehouseDispatchCapacityWarning,
  WarehouseFleetVehicle,
  WarehouseVehicleUnavailableReason,
} from '@pegasusx/types';
import { usePortalT } from '@/lib/i18n';
import { PageSection } from '@/components/PageSection';

export const VEHICLE_UNAVAILABLE_REASONS: WarehouseVehicleUnavailableReason[] = [
  'MAINTENANCE',
  'TRUCK_DAMAGED',
  'REGULATORY_HOLD',
  'MANUAL_HOLD',
  'OTHER',
];

export function formatUnavailableReason(reason?: string, note?: string) {
  if (!reason) {
    return note || '';
  }
  if (reason.toUpperCase() === 'OTHER' && note?.trim()) {
    return note.trim();
  }
  return reason
    .toLowerCase()
    .split('_')
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

export function formatVU(value: number) {
  return value.toFixed(1);
}

export function DispatchFleetTrucksSection({
  vehicles,
  vehicleReasons,
  setVehicleReasons,
  vehicleNotes,
  setVehicleNotes,
  mutatingVehicleId,
  onToggleAvailability,
}: {
  vehicles: WarehouseFleetVehicle[];
  vehicleReasons: Record<string, WarehouseVehicleUnavailableReason>;
  setVehicleReasons: Dispatch<SetStateAction<Record<string, WarehouseVehicleUnavailableReason>>>;
  vehicleNotes: Record<string, string>;
  setVehicleNotes: Dispatch<SetStateAction<Record<string, string>>>;
  mutatingVehicleId: string;
  onToggleAvailability: (vehicle: WarehouseFleetVehicle, nextActive: boolean) => Promise<void>;
}) {
  const t = usePortalT();
  return (
    <PageSection
      title={`Fleet trucks (${vehicles.length})`}
      description={t("warehouse_portal.residual.text.mark_trucks_unavailable_in_real_time_dispatch_and_smart_suggest_")}
      className="mb-6"
    >
      {vehicles.length === 0 ? (
        <p className="text-sm text-(--muted)">{t("warehouse_portal.dispatch.text.no_vehicles_registered_for_this_warehouse")}</p>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          {vehicles.map(vehicle => (
            <div key={vehicle.vehicle_id} className="rounded-lg border border-(--border) p-3">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <div className="text-sm font-medium">{vehicle.label || vehicle.license_plate}</div>
                  <div className="text-xs text-(--muted)">{vehicle.license_plate} · {vehicle.vehicle_class}</div>
                  {!vehicle.is_active && (
                    <div className="text-xs mt-1" style={{ color: 'var(--warning)' }}>
                      {formatUnavailableReason(vehicle.unavailable_reason, vehicle.unavailable_note)}
                    </div>
                  )}
                </div>
                <span className={`status-chip ${vehicle.is_active ? 'status-chip--stable' : 'status-chip--draft'}`}>
                  {vehicle.is_active ? 'Active' : 'Unavailable'}
                </span>
              </div>
              <div className="mt-3 flex flex-wrap items-center gap-2">
                {vehicle.is_active && (
                  <>
                    <select
                      value={vehicleReasons[vehicle.vehicle_id] || vehicle.unavailable_reason || 'MANUAL_HOLD'}
                      onChange={event => setVehicleReasons(current => ({
                        ...current,
                        [vehicle.vehicle_id]: event.target.value as WarehouseVehicleUnavailableReason,
                      }))}
                      disabled={mutatingVehicleId === vehicle.vehicle_id}
                      className="rounded-lg border px-2 py-1 text-xs"
                      style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
                    >
                      {VEHICLE_UNAVAILABLE_REASONS.map(reason => (
                        <option key={reason} value={reason}>{formatUnavailableReason(reason)}</option>
                      ))}
                    </select>
                    {(vehicleReasons[vehicle.vehicle_id] || 'MANUAL_HOLD') === 'OTHER' && (
                      <>
                        <label htmlFor={`vehicle-custom-reason-${vehicle.vehicle_id}`} className="sr-only">{t("warehouse_portal.dispatch.text.custom_reason")}</label>
                        <input
                          id={`vehicle-custom-reason-${vehicle.vehicle_id}`}
                          aria-label={t("warehouse_portal.dispatch.text.custom_reason")}
                          type="text"
                          placeholder={t("warehouse_portal.dispatch.text.custom_reason")}
                          value={vehicleNotes[vehicle.vehicle_id] || ''}
                          onChange={event => setVehicleNotes(current => ({
                            ...current,
                            [vehicle.vehicle_id]: event.target.value,
                          }))}
                          disabled={mutatingVehicleId === vehicle.vehicle_id}
                          className="rounded-lg border px-2 py-1 text-xs min-w-32"
                          style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
                        />
                      </>
                    )}
                  </>
                )}
                <button
                  type="button"
                  disabled={mutatingVehicleId === vehicle.vehicle_id}
                  onClick={() => void onToggleAvailability(vehicle, !vehicle.is_active)}
                  className="rounded-lg px-2 py-1 text-xs font-medium disabled:opacity-50"
                  style={{
                    background: vehicle.is_active ? 'color-mix(in srgb, var(--warning) 15%, transparent)' : 'color-mix(in srgb, var(--success) 15%, transparent)',
                    color: vehicle.is_active ? 'var(--warning)' : 'var(--success)',
                  }}
                >
                  {mutatingVehicleId === vehicle.vehicle_id ? 'Updating…' : vehicle.is_active ? 'Mark unavailable' : 'Restore'}
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </PageSection>
  );
}

export function DispatchDriverRescueSection({
  rescueDriverId,
  setRescueDriverId,
  rescueRequest,
  setRescueRequest,
  rescueLoading,
  rescueAction,
  setRescueAction,
  rescueNewDate,
  setRescueNewDate,
  onLookupRescue,
  onApproveRescue,
}: {
  rescueDriverId: string;
  setRescueDriverId: (val: string) => void;
  rescueRequest: { order_count: number; status: string; reason?: string } | null;
  setRescueRequest: (val: { order_count: number; status: string; reason?: string } | null) => void;
  rescueLoading: boolean;
  rescueAction: 'CANCEL' | 'RESCHEDULE';
  setRescueAction: (val: 'CANCEL' | 'RESCHEDULE') => void;
  rescueNewDate: string;
  setRescueNewDate: (val: string) => void;
  onLookupRescue: () => void;
  onApproveRescue: () => void;
}) {
  return (
    <PageSection
      title="Active Driver Rescue Requests"
      description="Drivers requesting early complete due to breakdowns. Approve to cancel or reschedule their remaining manifest."
      className="mt-6"
    >
      <div className="rounded-xl border p-5" style={{ borderColor: 'var(--border)', background: 'var(--background)' }}>
        <div className="flex gap-3 max-w-lg mb-4">
          <label htmlFor="rescue-driver-id-input" className="sr-only">Driver ID (e.g. driver-123)</label>
          <input
            id="rescue-driver-id-input"
            aria-label="Driver ID (e.g. driver-123)"
            type="text"
            placeholder="Driver ID (e.g. driver-123)"
            className="portal-input flex-1"
            value={rescueDriverId}
            onChange={e => setRescueDriverId(e.target.value)}
          />
          <button
            type="button"
            className="portal-btn portal-btn--secondary"
            disabled={rescueLoading || !rescueDriverId}
            onClick={onLookupRescue}
          >
            Lookup Request
          </button>
        </div>
        {rescueRequest ? (
          <div className="rounded-xl border border-[var(--warning)]/30 bg-[var(--warning)]/10 p-4 max-w-lg">
            <p className="font-semibold mb-2 text-[var(--warning)]">
              Rescue Requested! ({rescueRequest.order_count} orders pending)
            </p>
            <div className="space-y-3 mt-4">
              <label className="portal-field">
                <span className="portal-label">Action</span>
                <select
                  className="portal-input"
                  value={rescueAction}
                  onChange={e => setRescueAction(e.target.value as 'CANCEL' | 'RESCHEDULE')}
                >
                  <option value="CANCEL">Cancel Remaining Orders</option>
                  <option value="RESCHEDULE">Reschedule Remaining Orders</option>
                </select>
              </label>
              {rescueAction === 'RESCHEDULE' && (
                <label htmlFor="rescue-proposed-date" className="portal-field">
                  <span className="portal-label">Proposed Date</span>
                  <input
                    id="rescue-proposed-date"
                    aria-label="Proposed Date"
                    type="date"
                    className="portal-input"
                    value={rescueNewDate}
                    onChange={e => setRescueNewDate(e.target.value)}
                  />
                </label>
              )}
              <div className="flex gap-2 pt-2">
                <button
                  type="button"
                  className="portal-btn portal-btn--primary w-full"
                  disabled={rescueLoading}
                  onClick={onApproveRescue}
                >
                  Approve Rescue
                </button>
                <button
                  type="button"
                  className="portal-btn portal-btn--outline w-full"
                  onClick={() => setRescueRequest(null)}
                >
                  Dismiss
                </button>
              </div>
            </div>
          </div>
        ) : null}
      </div>
    </PageSection>
  );
}

export function DispatchConfirmAndCapacityModals({
  showSmartConfirm,
  setShowSmartConfirm,
  selectedOrderCount,
  totalOrderCount,
  proposedRouteCount,
  smartStopCount,
  executing,
  onRunSmartDispatch,
  capacityPrompt,
  setCapacityPrompt,
  capacityPromptMode,
  suggestedUnselectCount,
  onApplyCapacitySuggestion,
  onRunManualDispatch,
}: {
  showSmartConfirm: boolean;
  setShowSmartConfirm: (val: boolean) => void;
  selectedOrderCount: number;
  totalOrderCount: number;
  proposedRouteCount: number;
  smartStopCount: number;
  executing: boolean;
  onRunSmartDispatch: (opts?: { forceCapacity?: boolean; acceptPartial?: boolean }) => Promise<void>;
  capacityPrompt: WarehouseDispatchCapacityWarning[] | null;
  setCapacityPrompt: (val: WarehouseDispatchCapacityWarning[] | null) => void;
  capacityPromptMode: 'manual' | 'auto';
  suggestedUnselectCount: number;
  onApplyCapacitySuggestion: () => void;
  onRunManualDispatch: (forceCapacity?: boolean) => Promise<void>;
}) {
  const t = usePortalT();
  return (
    <>
      {showSmartConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4" style={{ background: 'rgba(0,0,0,0.45)' }}>
          <div className="w-full max-w-md rounded-xl border border-(--border) p-5" style={{ background: 'var(--background)' }}>
            <h3 className="text-base font-semibold mb-2">{t("warehouse_portal.dispatch.text.run_smart_dispatch")}</h3>
            <p className="text-sm text-(--muted) mb-4">
              {selectedOrderCount > 0
                ? `Assign ${selectedOrderCount} selected order(s) using the optimizer.`
                : `Assign all ${totalOrderCount} undispatched order(s) using the optimizer.`}
              {proposedRouteCount > 0 && ` ${proposedRouteCount} route(s), ${smartStopCount} stop(s) in preview.`}
            </p>
            <div className="flex justify-end gap-2">
              <button type="button" className="px-3 py-1.5 rounded-lg text-sm button--secondary" onClick={() => setShowSmartConfirm(false)}>
                Cancel
              </button>
              <button
                type="button"
                className="px-3 py-1.5 rounded-lg text-sm button--primary"
                disabled={executing}
                onClick={() => void onRunSmartDispatch()}
              >
                Smart Dispatch
              </button>
            </div>
          </div>
        </div>
      )}

      {capacityPrompt && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4" style={{ background: 'rgba(0,0,0,0.45)' }}>
          <div className="w-full max-w-md rounded-xl border border-(--border) p-5" style={{ background: 'var(--background)' }}>
            <h3 className="text-base font-semibold mb-2">{t("warehouse_portal.dispatch.text.capacity_exceeded")}</h3>
            <p className="text-sm text-(--muted) mb-4">
              {capacityPromptMode === 'auto'
                ? 'Smart dispatch cannot fit all orders on available trucks. Accept to dispatch feasible routes and leave the rest undispatched, or force to override capacity.'
                : 'Selected orders exceed the truck effective capacity (95% Tetris buffer). Apply the suggestion, force dispatch, or cancel.'}
            </p>
            <ul className="text-sm space-y-2 mb-4">
              {capacityPrompt.map(warning => (
                <li key={warning.driver_id} className="font-mono text-xs">
                  {formatVU(warning.loaded_vu)} VU loaded / {formatVU(warning.effective_max_vu)} VU effective max
                  {warning.suggested_unselect_order_ids?.length ? (
                    <div className="mt-1 text-(--muted)">
                      Suggested unselect: {warning.suggested_unselect_order_ids.join(', ')}
                    </div>
                  ) : null}
                  {warning.suggested_defer_order_ids?.length ? (
                    <div className="mt-1 text-(--muted)">
                      Suggested defer: {warning.suggested_defer_order_ids.join(', ')}
                    </div>
                  ) : null}
                </li>
              ))}
            </ul>
            <div className="flex justify-end gap-2 flex-wrap">
              <button
                type="button"
                className="px-3 py-1.5 rounded-lg text-sm button--secondary"
                onClick={() => setCapacityPrompt(null)}
              >
                Cancel
              </button>
              {capacityPromptMode === 'manual' && suggestedUnselectCount > 0 && (
                <button
                  type="button"
                  className="px-3 py-1.5 rounded-lg text-sm button--secondary"
                  onClick={onApplyCapacitySuggestion}
                >
                  Apply suggestion
                </button>
              )}
              {capacityPromptMode === 'auto' && (
                <button
                  type="button"
                  className="px-3 py-1.5 rounded-lg text-sm button--secondary"
                  disabled={executing}
                  onClick={() => {
                    setCapacityPrompt(null);
                    void onRunSmartDispatch({ acceptPartial: true });
                  }}
                >
                  Accept partial
                </button>
              )}
              <button
                type="button"
                className="px-3 py-1.5 rounded-lg text-sm button--primary"
                disabled={executing}
                onClick={() => {
                  setCapacityPrompt(null);
                  if (capacityPromptMode === 'auto') {
                    void onRunSmartDispatch({ forceCapacity: true });
                  } else {
                    void onRunManualDispatch(true);
                  }
                }}
              >
                Force dispatch
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
