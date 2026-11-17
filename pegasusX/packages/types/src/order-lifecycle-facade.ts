import { OrderStatus } from "./primitives";

/**
 * High-level simplified customer stages presented to Retailers and Mobile Apps.
 * Bridges the gap between the 17-state backend invariant machine and clean 4-stage UX.
 */
export type OrderCustomerStage =
  | "PLACED"
  | "PROCESSING"
  | "OUT_FOR_DELIVERY"
  | "DELIVERED"
  | "CANCELLED"
  | "EXCEPTION_ACTION_REQUIRED";

export interface CustomerStageMeta {
  readonly stage: OrderCustomerStage;
  readonly label: string;
  readonly progressPercent: number;
  readonly description: string;
  readonly isTerminal?: boolean;
  readonly requiresCustomerAction?: boolean;
}

export const CUSTOMER_STAGE_METADATA: Record<OrderCustomerStage, CustomerStageMeta> = {
  PLACED: {
    stage: "PLACED",
    label: "Order Placed",
    progressPercent: 20,
    description: "Order is confirmed and queued for fulfillment scheduling.",
  },
  PROCESSING: {
    stage: "PROCESSING",
    label: "Preparing Order",
    progressPercent: 50,
    description: "Warehouse is picking, verifying lots, and staging items for transport.",
  },
  OUT_FOR_DELIVERY: {
    stage: "OUT_FOR_DELIVERY",
    label: "Out for Delivery",
    progressPercent: 80,
    description: "Driver is en route or at your doorstep for receipt verification.",
  },
  DELIVERED: {
    stage: "DELIVERED",
    label: "Delivered",
    progressPercent: 100,
    description: "Delivery completed with verified tax stamp and digital receipt.",
    isTerminal: true,
  },
  CANCELLED: {
    stage: "CANCELLED",
    label: "Cancelled",
    progressPercent: 0,
    description: "Order was cancelled before fulfillment.",
    isTerminal: true,
  },
  EXCEPTION_ACTION_REQUIRED: {
    stage: "EXCEPTION_ACTION_REQUIRED",
    label: "Action Required",
    progressPercent: 75,
    description: "An operational prompt requires your input (e.g. shop closed or delivery resolution).",
    requiresCustomerAction: true,
  },
};

/**
 * Projects any of the 17 canonical backend order statuses into the simplified customer stage.
 */
export function toOrderCustomerStage(rawStatus: string | OrderStatus): OrderCustomerStage {
  const status = String(rawStatus || "").trim().toUpperCase();

  switch (status) {
    case "SCHEDULED":
    case "AUTO_ACCEPTED":
    case "BACKORDERED":
    case "PENDING_REVIEW":
    case "DRAFT":
    case "PENDING_APPROVAL":
    case "CONFIRMED":
      return "PLACED";

    case "PENDING":
    case "DELAYED":
    case "PICKING":
    case "PACKED":
    case "LOADED":
    case "DISPATCHED":
      return "PROCESSING";

    case "IN_TRANSIT":
    case "EN_ROUTE":
    case "ARRIVING":
    case "ARRIVED":
    case "AWAITING_PAYMENT":
    case "PENDING_CASH_COLLECTION":
      return "OUT_FOR_DELIVERY";

    case "DELIVERED_ON_CREDIT":
    case "FISCALIZING":
    case "COMPLETED":
    case "DELIVERED":
      return "DELIVERED";

    case "CANCELLED":
    case "NO_CAPACITY":
      return "CANCELLED";

    case "SHOP_CLOSED_PENDING":
    case "ARRIVED_SHOP_CLOSED":
    case "CANCEL_REQUESTED":
    case "FISCAL_FAILED":
    case "RECONCILIATION_REQUIRED":
    case "QUARANTINE":
    case "SYNC_CONFLICT":
    case "DISPUTED":
      return "EXCEPTION_ACTION_REQUIRED";

    default:
      return "PROCESSING";
  }
}

/**
 * Returns progress percentage (0 - 100) for UI progress bars.
 */
export function getCustomerStageProgressPercent(stage: OrderCustomerStage): number {
  return CUSTOMER_STAGE_METADATA[stage]?.progressPercent ?? 0;
}

/**
 * Checks whether the current stage demands customer attention.
 */
export function isCustomerStageActionRequired(stage: OrderCustomerStage): boolean {
  return Boolean(CUSTOMER_STAGE_METADATA[stage]?.requiresCustomerAction);
}
