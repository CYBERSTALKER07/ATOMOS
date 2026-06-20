import {
  warehouseOrderDelayKey,
  warehouseOrderRejectKey,
} from '@pegasusx/api-client';
import { createSupplierApi } from '@/lib/api';

const api = createSupplierApi();

/** Supplier ADMIN warehouse-compat order mutations (scoped by order warehouse_id). */
export const supplierWarehouseOps = {
  delayOrder: (orderId: string, warehouseId: string, reason?: string) =>
    api.postWarehouseOrderDelay(
      orderId,
      reason ? { reason } : {},
      { warehouse_id: warehouseId },
      warehouseOrderDelayKey(orderId),
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
