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
  PENDING_REVIEW: 'status-chip--draft',
  SCHEDULED: 'status-chip--draft',
  AUTO_ACCEPTED: 'status-chip--ready',
  BACKORDERED: 'status-chip--draft',
  DELAYED: 'status-chip--draft',
  LOADED: 'status-chip--ready',
  DISPATCHED: 'status-chip--active',
  IN_TRANSIT: 'status-chip--active',
  ARRIVING: 'status-chip--ready',
  ARRIVED: 'status-chip--ready',
  SHOP_CLOSED_PENDING: 'status-chip--warning',
  ARRIVED_SHOP_CLOSED: 'status-chip--warning',
  AWAITING_PAYMENT: 'status-chip--warning',
  PENDING_CASH_COLLECTION: 'status-chip--warning',
  DELIVERED_ON_CREDIT: 'status-chip--stable',
  FISCALIZING: 'status-chip--warning',
  FISCAL_FAILED: 'status-chip--danger',
  COMPLETED: 'status-chip--stable',
  CANCEL_REQUESTED: 'status-chip--danger',
  NO_CAPACITY: 'status-chip--critical',
  CANCELLED: 'status-chip--critical',
  QUARANTINE: 'status-chip--danger',
  RECONCILIATION_REQUIRED: 'status-chip--warning',
  SYNC_CONFLICT: 'status-chip--danger',
};
