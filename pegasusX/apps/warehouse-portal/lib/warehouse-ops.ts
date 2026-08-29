import {
  warehouseDispatchLockAcquireKey,
  warehouseDispatchLockReleaseKey,
  warehouseEmergencyTransferKey,
  warehouseForceReceiveKey,
  warehouseOrderDelayKey,
  warehouseOrderOverflowKey,
  warehouseOrderProposeDeliveryKey,
  warehouseOrderRejectKey,
  warehouseReceiveTransferKey,
} from '@pegasusx/api-core';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseHomeNodeId, warehouseScopeQuery } from '@/lib/warehouse-scope';

/** Typed warehouse mutation helpers for portal order + transfer action panels. */
export const warehouseOps = {
  delayOrder: (orderId: string, reason?: string) =>
    warehouseApi.postWarehouseOrderDelay(
      orderId,
      reason ? { reason } : {},
      {},
      warehouseOrderDelayKey(orderId),
    ),
  rejectOrder: (orderId: string, reason: string) =>
    warehouseApi.postWarehouseOrderReject(
      orderId,
      { reason },
      {},
      warehouseOrderRejectKey(orderId, reason),
    ),
  proposeOrderDelivery: (orderId: string, proposedDeliveryDate: string, reason: string) =>
    warehouseApi.postWarehouseOrderProposeDelivery(
      orderId,
      { proposed_delivery_date: proposedDeliveryDate, reason },
      {},
      warehouseOrderProposeDeliveryKey(orderId, proposedDeliveryDate, reason),
    ),
  proposePreorderDelivery: (orderId: string, proposedDeliveryDate: string, reason: string) =>
    warehouseApi.postWarehouseOrderProposeDelivery(
      orderId,
      { proposed_delivery_date: proposedDeliveryDate, reason },
      {},
      warehouseOrderProposeDeliveryKey(orderId, proposedDeliveryDate, reason),
    ),
  rejectPreorder: (orderId: string, reason: string) =>
    warehouseApi.postWarehousePreorderReject(
      orderId,
      { reason },
      {},
      warehouseOrderRejectKey(orderId, reason),
    ),
  overflowOrder: (orderId: string, reason?: string) =>
    warehouseApi.postWarehouseOrderOverflow(
      orderId,
      reason ? { reason } : {},
      {},
      warehouseOrderOverflowKey(orderId),
    ),
  emergencyTransfer: (totalVolumeVu: number, notes?: string) => {
    const warehouseId = warehouseHomeNodeId() || 'warehouse';
    return warehouseApi.postWarehouseEmergencyTransfer(
      { total_volume_vu: totalVolumeVu, notes },
      warehouseScopeQuery(),
      warehouseEmergencyTransferKey(warehouseId, totalVolumeVu, notes),
    );
  },
  forceReceive: (totalVolumeVu: number, notes?: string, factoryId?: string) => {
    const warehouseId = warehouseHomeNodeId() || 'warehouse';
    return warehouseApi.postWarehouseForceReceive(
      {
        total_volume_vu: totalVolumeVu,
        notes,
        factory_id: factoryId,
      },
      warehouseScopeQuery(),
      warehouseForceReceiveKey(warehouseId, totalVolumeVu, notes, factoryId),
    );
  },
  receiveTransfer: (transferId: string) =>
    warehouseApi.postWarehouseReceiveTransfer(
      transferId,
      warehouseScopeQuery(),
      warehouseReceiveTransferKey(transferId),
    ),
  acquireDispatchLock: (warehouseId: string, entityType: string, entityId: string, reason: string) =>
    warehouseApi.acquireWarehouseDispatchLock(
      { entity_type: entityType, entity_id: entityId, reason },
      {},
      warehouseDispatchLockAcquireKey(warehouseId, entityType, entityId),
    ),
  releaseDispatchLock: (lockId: string) =>
    warehouseApi.releaseWarehouseDispatchLock(
      { lock_id: lockId },
      warehouseDispatchLockReleaseKey(lockId),
    ),
};
