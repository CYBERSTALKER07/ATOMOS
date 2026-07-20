export type OrderActionFlags = {
  /** Warehouse proposes a new delivery date; retailer accepts or rejects. */
  canDelay: boolean;
  canReject: boolean;
  canOverflow: boolean;
  canReassign: boolean;
};

/** States eligible for POST .../propose-delivery (not LOADED/IN_TRANSIT/COMPLETED/CANCELLED). */
export function orderActionFlags(state: string): OrderActionFlags {
  const s = state.toUpperCase();
  const terminal = s === 'COMPLETED' || s === 'CANCELLED';
  const inFlight = s === 'LOADED' || s === 'IN_TRANSIT';
  return {
    canDelay: !terminal && !inFlight,
    canReject:
      s === 'PENDING' ||
      s === 'LOADED' ||
      s === 'IN_TRANSIT' ||
      s === 'SCHEDULED' ||
      s === 'AUTO_ACCEPTED' ||
      s === 'DELAYED' ||
      s === 'ARRIVED',
    canOverflow: s === 'LOADED' || s === 'IN_TRANSIT',
    canReassign: !terminal && s !== 'SCHEDULED' && s !== 'DELAYED',
  };
}

export const ORDER_STATE_CHIP: Record<string, string> = {
  PENDING: 'status-chip--draft',
  LOADED: 'status-chip--ready',
  IN_TRANSIT: 'status-chip--active',
  DELAYED: 'status-chip--draft',
  ARRIVED: 'status-chip--ready',
  COMPLETED: 'status-chip--stable',
  FISCALIZING: 'status-chip--warning',
  FISCAL_FAILED: 'status-chip--danger',
  CANCELLED: 'status-chip--critical',
  SCHEDULED: 'status-chip--draft',
  AUTO_ACCEPTED: 'status-chip--ready',
};
