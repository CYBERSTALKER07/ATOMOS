'use client';

import type { WarehouseDispatchDriver, WarehouseUnavailableDispatchDriver } from '@pegasusx/types';
import { PageSection } from '@/components/PageSection';
import EmptyState from '@/components/EmptyState';

function formatVU(vu: number) {
  return vu.toFixed(1);
}

function formatUnavailableReason(reason?: string, note?: string) {
  const labels: Record<string, string> = {
    MAINTENANCE: 'Scheduled maintenance',
    TRUCK_DAMAGED: 'Vehicle damaged',
    REGULATORY_HOLD: 'Regulatory hold',
    MANUAL_HOLD: 'Manually held',
    OTHER: note || 'Other reason',
  };
  return reason ? labels[reason] ?? reason : '';
}

interface DispatchDriverListProps {
  drivers: WarehouseDispatchDriver[];
  unavailableDrivers: WarehouseUnavailableDispatchDriver[];
}

/**
 * Available and unavailable drivers section of the Dispatch screen.
 *
 * Renders driver cards with status badges, VU capacity, and
 * unavailability reasons in a split layout.
 */
export default function DispatchDriverList({
  drivers,
  unavailableDrivers,
}: DispatchDriverListProps) {
  return (
    <PageSection title={`Available drivers (${drivers.length})`}>
      <div className="space-y-4 max-h-80 overflow-y-auto">
        {drivers.length === 0 ? (
          <EmptyState variant="no-data" headline="No drivers available" body="All drivers are on route or blocked." />
        ) : (
          <div className="space-y-2">
            {drivers.map(driver => (
              <div key={driver.driver_id} className="flex items-center justify-between p-3 rounded-lg border border-(--border)">
                <div>
                  <div className="text-sm font-medium">{driver.name}</div>
                  <div className="text-xs text-(--muted)">{driver.vehicle_label || driver.phone || 'Assigned vehicle'}</div>
                </div>
                <div className="text-right">
                  <span className="status-chip status-chip--stable">{driver.truck_status || 'IDLE'}</span>
                  <div className="text-xs text-(--muted) mt-1 font-mono">
                    {driver.free_volume_vu != null && driver.free_volume_vu > 0
                      ? `${formatVU(driver.free_volume_vu)} VU free`
                      : `${formatVU(driver.max_volume_vu ?? 0)} VU max`}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        <div className="border-t border-(--border) pt-4">
          <h3 className="text-xs font-semibold uppercase tracking-[0.16em] text-(--muted) mb-2">
            Unavailable ({unavailableDrivers.length})
          </h3>
          {unavailableDrivers.length === 0 ? (
            <p className="text-sm text-(--muted) py-2 text-center">No drivers blocked — all assigned trucks eligible or off-shift reasons shown here in real time.</p>
          ) : (
            <div className="space-y-2">
              {unavailableDrivers.map(driver => (
                <div key={driver.driver_id} className="rounded-lg border border-(--border) p-3">
                  <div className="flex items-center justify-between gap-3">
                    <div>
                      <div className="text-sm font-medium">{driver.name}</div>
                      <div className="text-xs text-(--muted)">{driver.vehicle_label || driver.phone || 'Assigned vehicle unavailable'}</div>
                    </div>
                    <span className="status-chip status-chip--draft">{driver.truck_status || 'IDLE'}</span>
                  </div>
                  {driver.unavailable_reason && (
                    <div className="mt-2 text-xs" style={{ color: 'var(--warning)' }}>
                      {formatUnavailableReason(driver.unavailable_reason)}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    </PageSection>
  );
}
