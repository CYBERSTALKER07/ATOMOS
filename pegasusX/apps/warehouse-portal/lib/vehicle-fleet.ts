import type { WarehouseVehicleUnavailableReason } from '@pegasusx/types';

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

export function vehicleStatusLabel(isActive: boolean) {
  return isActive ? 'Active' : 'Unavailable';
}
