import { warehouseApi } from "@/lib/warehouse-api";

/** Typed warehouse mutation helpers for portal order + transfer action panels. */
export const warehouseOps = {
  delayOrder: (orderId: string, reason?: string) =>
    warehouseApi.postWarehouseOrderDelay(orderId, reason ? { reason } : {}),
  rejectOrder: (orderId: string, reason: string) =>
    warehouseApi.postWarehouseOrderReject(orderId, { reason }),
  overflowOrder: (orderId: string, reason?: string) =>
    warehouseApi.postWarehouseOrderOverflow(orderId, reason ? { reason } : {}),
  emergencyTransfer: (totalVolumeVu: number, notes?: string) =>
    warehouseApi.postWarehouseEmergencyTransfer({ total_volume_vu: totalVolumeVu, notes }),
  forceReceive: (totalVolumeVu: number, notes?: string, factoryId?: string) =>
    warehouseApi.postWarehouseForceReceive({
      total_volume_vu: totalVolumeVu,
      notes,
      factory_id: factoryId,
    }),
  receiveTransfer: (transferId: string) =>
    warehouseApi.postWarehouseReceiveTransfer(transferId),
};
