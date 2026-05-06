export interface FactorySupplyRequestUpdateEvent {
  type: 'FACTORY_SUPPLY_REQUEST_UPDATE';
  factory_id: string;
  supplier_id: string;
  warehouse_id: string;
  request_id: string;
  state: string;
  action: string;
  trace_id: string;
  timestamp: string;
}

export interface FactoryTransferUpdateEvent {
  type: 'FACTORY_TRANSFER_UPDATE';
  factory_id: string;
  supplier_id: string;
  transfer_id: string;
  warehouse_id: string;
  manifest_id: string;
  from_state: string;
  to_state: string;
  action: string;
  trace_id: string;
  timestamp: string;
}

export interface FactoryManifestUpdateEvent {
  type: 'FACTORY_MANIFEST_UPDATE';
  factory_id: string;
  supplier_id: string;
  manifest_id: string;
  state: string;
  action: string;
  reason: string;
  transfer_ids: string[];
  trace_id: string;
  timestamp: string;
}

export interface FactoryOutboxFailureEvent {
  type: 'FACTORY_OUTBOX_FAILED';
  factory_id: string;
  supplier_id: string;
  event_id: string;
  aggregate_id: string;
  topic: string;
  reason: string;
  trace_id: string;
  timestamp: string;
}

export type FactoryLiveEvent =
  | FactorySupplyRequestUpdateEvent
  | FactoryTransferUpdateEvent
  | FactoryManifestUpdateEvent
  | FactoryOutboxFailureEvent;
