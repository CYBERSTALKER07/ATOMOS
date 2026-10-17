import { OrderStatus } from "./primitives";

export type LifecyclePhase =
  | "intake"
  | "fulfillment"
  | "exception"
  | "fiscal"
  | "terminal";

export interface LifecycleNodeMeta {
  readonly status: OrderStatus;
  readonly label: string;
  readonly phase: LifecyclePhase;
  readonly isTerminal?: boolean;
  readonly description?: string;
}

/**
 * Authoritative Canonical Adjacency Map for Order Lifecycle.
 * Mirrors pegasusX/apps/backend-go/order/state_machine.go & graph.go.
 */
export const CANONICAL_ORDER_GRAPH: Record<string, readonly OrderStatus[]> = {
  SCHEDULED: [
    "AUTO_ACCEPTED",
    "PENDING",
    "CANCELLED",
    "CANCEL_REQUESTED",
  ],
  AUTO_ACCEPTED: [
    "PENDING",
    "CANCELLED",
    "CANCEL_REQUESTED",
  ],
  BACKORDERED: [
    "PENDING",
    "SCHEDULED",
    "CANCELLED",
  ],
  PENDING: [
    "LOADED",
    "DELAYED",
    "CANCELLED",
  ],
  DELAYED: [
    "PENDING",
  ],
  LOADED: [
    "IN_TRANSIT",
    "PENDING",
    "DELAYED",
    "CANCELLED",
    "CANCEL_REQUESTED",
  ],
  IN_TRANSIT: [
    "ARRIVED",
    "PENDING",
    "CANCELLED",
    "CANCEL_REQUESTED",
  ],
  ARRIVED: [
    "AWAITING_PAYMENT",
    "PENDING_CASH_COLLECTION",
    "DELIVERED_ON_CREDIT",
    "SHOP_CLOSED_PENDING",
    "CANCEL_REQUESTED",
  ],
  SHOP_CLOSED_PENDING: [
    "AWAITING_PAYMENT",
    "PENDING_CASH_COLLECTION",
    "DELIVERED_ON_CREDIT",
    "ARRIVED",
    "CANCEL_REQUESTED",
    "CANCELLED",
  ],
  AWAITING_PAYMENT: [
    "FISCALIZING",
    "PENDING_CASH_COLLECTION",
    "DELIVERED_ON_CREDIT",
  ],
  PENDING_CASH_COLLECTION: [
    "FISCALIZING",
  ],
  DELIVERED_ON_CREDIT: [
    "FISCALIZING",
  ],
  FISCALIZING: [
    "COMPLETED",
    "FISCAL_FAILED",
  ],
  FISCAL_FAILED: [
    "FISCALIZING",
    "COMPLETED",
  ],
  CANCEL_REQUESTED: [
    "CANCELLED",
    "LOADED",
    "IN_TRANSIT",
    "ARRIVED",
  ],
  CANCELLED: [
    "RECONCILIATION_REQUIRED",
  ],
  RECONCILIATION_REQUIRED: [
    "COMPLETED",
    "CANCELLED",
  ],
  COMPLETED: [],
};

export const ORDER_LIFECYCLE_NODES: Record<string, LifecycleNodeMeta> = {
  SCHEDULED: {
    status: "SCHEDULED",
    label: "Scheduled (Preorder)",
    phase: "intake",
    description: "Preorder reservation pending midnight confirmation window.",
  },
  AUTO_ACCEPTED: {
    status: "AUTO_ACCEPTED",
    label: "Auto-Accepted",
    phase: "intake",
    description: "AI midnight guard approved preorder before operational execution.",
  },
  BACKORDERED: {
    status: "BACKORDERED",
    label: "Backordered",
    phase: "intake",
    description: "Stock shortage awaiting warehouse replenishment.",
  },
  PENDING: {
    status: "PENDING",
    label: "Pending Queue",
    phase: "intake",
    description: "Active operational root queue for picking & staging.",
  },
  DELAYED: {
    status: "DELAYED",
    label: "Delayed Slip",
    phase: "intake",
    description: "Slipped picking/loading SLA deadline.",
  },
  LOADED: {
    status: "LOADED",
    label: "Loaded & Staged",
    phase: "fulfillment",
    description: "Order manifest packed into transport cage or delivery vehicle.",
  },
  IN_TRANSIT: {
    status: "IN_TRANSIT",
    label: "In Transit",
    phase: "fulfillment",
    description: "Driver en route to destination geofence.",
  },
  ARRIVED: {
    status: "ARRIVED",
    label: "Arrived at Destination",
    phase: "fulfillment",
    description: "Driver reached delivery coordinates; doorstep handover active.",
  },
  SHOP_CLOSED_PENDING: {
    status: "SHOP_CLOSED_PENDING",
    label: "Shop Closed Bypass",
    phase: "exception",
    description: "Shop closed protocol active; photo evidence verified.",
  },
  AWAITING_PAYMENT: {
    status: "AWAITING_PAYMENT",
    label: "Awaiting Digital Payment",
    phase: "fiscal",
    description: "Card or QR payment gateway in-flight.",
  },
  PENDING_CASH_COLLECTION: {
    status: "PENDING_CASH_COLLECTION",
    label: "Cash on Delivery",
    phase: "fiscal",
    description: "Driver physical cash collection pending receipt.",
  },
  DELIVERED_ON_CREDIT: {
    status: "DELIVERED_ON_CREDIT",
    label: "Delivered on Credit",
    phase: "fiscal",
    description: "Credit leave-behind (§9.1 non-terminal; awaits settlement).",
  },
  FISCALIZING: {
    status: "FISCALIZING",
    label: "ADR-009 Fiscalizing",
    phase: "fiscal",
    description: "Soliq OFD fiscal machine receipt generation in-flight.",
  },
  FISCAL_FAILED: {
    status: "FISCAL_FAILED",
    label: "Fiscal Failed (Retry)",
    phase: "fiscal",
    description: "OFD gateway failure; automatic retry or supervisor override allowed.",
  },
  CANCEL_REQUESTED: {
    status: "CANCEL_REQUESTED",
    label: "Cancel Requested",
    phase: "exception",
    description: "Two-party cancellation handshake in-flight (anti-brick protected).",
  },
  CANCELLED: {
    status: "CANCELLED",
    label: "Cancelled",
    phase: "terminal",
    isTerminal: false,
    description: "Order cancelled; eligible for inventory reconciliation audit.",
  },
  RECONCILIATION_REQUIRED: {
    status: "RECONCILIATION_REQUIRED",
    label: "Reconciliation Audit",
    phase: "exception",
    description: "Post-cancellation stock and financial ledger reconciliation.",
  },
  COMPLETED: {
    status: "COMPLETED",
    label: "Completed",
    phase: "terminal",
    isTerminal: true,
    description: "Terminal success state with finalized fiscal receipt.",
  },
};

/**
 * Validates whether transitioning from current to next is permitted.
 */
export function canOrderTransition(from: OrderStatus, to: OrderStatus): boolean {
  if (from === to) {
    return true; // Idempotent no-op
  }
  const permitted = CANONICAL_ORDER_GRAPH[from];
  return Boolean(permitted && permitted.includes(to));
}

/**
 * Returns allowed next statuses for a given status.
 */
export function getNextOrderStatuses(from: OrderStatus): readonly OrderStatus[] {
  return CANONICAL_ORDER_GRAPH[from] ?? [];
}

/**
 * Checks if status is terminal (zero outbound transitions).
 */
export function isTerminalOrderStatus(status: OrderStatus): boolean {
  const next = CANONICAL_ORDER_GRAPH[status];
  return Boolean(next && next.length === 0);
}
