import {
  warehouseOrderProposeDeliveryKey,
  warehouseOrderRejectKey,
} from '@pegasusx/api-core';
import { createSupplierApi } from '@/lib/api';

const api = createSupplierApi();

/** Supplier ADMIN warehouse-compat order mutations (scoped by order warehouse_id). */
export const supplierWarehouseOps = {
  proposeOrderDelivery: (
    orderId: string,
    warehouseId: string,
    proposedDeliveryDate: string,
    reason: string,
  ) =>
    api.postWarehouseOrderProposeDelivery(
      orderId,
      { proposed_delivery_date: proposedDeliveryDate, reason },
      { warehouse_id: warehouseId },
      warehouseOrderProposeDeliveryKey(orderId, proposedDeliveryDate, reason),
    ),
  rejectOrder: (orderId: string, warehouseId: string, reason: string) =>
    api.postWarehouseOrderReject(
      orderId,
      { reason },
      { warehouse_id: warehouseId },
      warehouseOrderRejectKey(orderId, reason),
    ),
  getOrderDetail: (orderId: string, warehouseId?: string) =>
    api.getWarehouseOrder(orderId, warehouseId ? { warehouse_id: warehouseId } : {}),
};
