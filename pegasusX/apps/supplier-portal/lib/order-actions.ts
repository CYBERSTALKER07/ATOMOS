export type OrderActionFlags = {
  canDelay: boolean;
  canReject: boolean;
  canOverflow: boolean;
};

export function orderActionFlags(state: string): OrderActionFlags {
  const s = state.toUpperCase();
  return {
    canDelay: s === 'PENDING' || s === 'LOADED',
    canReject: s === 'PENDING' || s === 'LOADED' || s === 'IN_TRANSIT' || s === 'SCHEDULED' || s === 'AUTO_ACCEPTED',
    canOverflow: s === 'LOADED' || s === 'IN_TRANSIT',
  };
}

export const ORDER_STATE_CHIP: Record<string, string> = {
  PENDING: 'status-chip--draft',
  LOADED: 'status-chip--ready',
  IN_TRANSIT: 'status-chip--active',
  DELAYED: 'status-chip--draft',
  ARRIVED: 'status-chip--ready',
  COMPLETED: 'status-chip--stable',
  CANCELLED: 'status-chip--critical',
  SCHEDULED: 'status-chip--draft',
  AUTO_ACCEPTED: 'status-chip--ready',
};
