import { WsEventEnvelope } from "./envelope";
import { ARInvoiceEventPayload, BuyerAcceptanceEventPayload, CommandLifecycle, DeliverySessionUpdated, DriverAvailabilityChangedEvent, DriverCreated, DriverLocationUpdatedEvent, FactoryCreated, FactoryStaffCreated, FactoryTransferCreated, ManifestCancelled, ManifestExceptionResolved, ManifestOrderException, ManifestOrderInjected, ManifestRebalanced, ManifestSealed, OrderAssigned, OrderCreated, OrderFinalized, OrderReassigned, OrderStatusChanged, OrderValidationFailed, PaymentCleared, PaymentRequired, PayoutBatchEventPayload, RetailerRegistered, SettlementRequired, SplitPaymentCreated, SupplierCreated, VehicleCreated, WarehouseCreated, WarehouseDispatchLockChanged, WarehouseSupplyRequestOpened } from "./event-payloads";

// ── Discriminated WS event union ────────────────────────────────────────────
export type WsEvent =
  | WsEventEnvelope<"SUPPLIER_CREATED", SupplierCreated>
  | WsEventEnvelope<"RETAILER_REGISTERED", RetailerRegistered>
  | WsEventEnvelope<"DRIVER_CREATED", DriverCreated>
  | WsEventEnvelope<"VEHICLE_CREATED", VehicleCreated>
  | WsEventEnvelope<"WAREHOUSE_CREATED", WarehouseCreated>
  | WsEventEnvelope<"WAREHOUSE_SUPPLY_REQUEST_OPENED", WarehouseSupplyRequestOpened>
  | WsEventEnvelope<"WAREHOUSE_DISPATCH_LOCK_CHANGED", WarehouseDispatchLockChanged>
  | WsEventEnvelope<"FACTORY_CREATED", FactoryCreated>
  | WsEventEnvelope<"FACTORY_STAFF_CREATED", FactoryStaffCreated>
  | WsEventEnvelope<"TRANSFER_CREATED", FactoryTransferCreated>
  | WsEventEnvelope<"MANIFEST_EXCEPTION_RESOLVED", ManifestExceptionResolved>
  | WsEventEnvelope<"ORDER_CREATED", OrderCreated>
  | WsEventEnvelope<"ORDER_STATUS_CHANGED", OrderStatusChanged>
  | WsEventEnvelope<"ORDER_VALIDATION_FAILED", OrderValidationFailed>
  | WsEventEnvelope<"ORDER_ASSIGNED", OrderAssigned>
  | WsEventEnvelope<"ORDER_REASSIGNED", OrderReassigned>
  | WsEventEnvelope<"ORDER_FINALIZED", OrderFinalized>
  | WsEventEnvelope<"SPLIT_PAYMENT_CREATED", SplitPaymentCreated>
  | WsEventEnvelope<"PAYMENT_REQUIRED", PaymentRequired>
  | WsEventEnvelope<"PAYMENT_CLEARED", PaymentCleared>
  | WsEventEnvelope<"SETTLEMENT_REQUIRED", SettlementRequired>
  | WsEventEnvelope<"AR_INVOICE_OPENED", ARInvoiceEventPayload>
  | WsEventEnvelope<"AR_INVOICE_PAYMENT", ARInvoiceEventPayload>
  | WsEventEnvelope<"AR_INVOICE_DUNNED", ARInvoiceEventPayload>
  | WsEventEnvelope<"AR_INVOICE_SETTLED", ARInvoiceEventPayload>
  | WsEventEnvelope<"PAYOUT_BATCH_GENERATED", PayoutBatchEventPayload>
  | WsEventEnvelope<"PAYOUT_BATCH_EXPORTED", PayoutBatchEventPayload>
  | WsEventEnvelope<"PAYOUT_BATCH_DISPATCHED", PayoutBatchEventPayload>
  | WsEventEnvelope<"PAYOUT_BATCH_PAID", PayoutBatchEventPayload>
  | WsEventEnvelope<"BUYER_ACCEPTANCE_PENDING", BuyerAcceptanceEventPayload>
  | WsEventEnvelope<"BUYER_ACCEPTANCE_ACCEPTED", BuyerAcceptanceEventPayload>
  | WsEventEnvelope<"BUYER_ACCEPTANCE_REJECTED", BuyerAcceptanceEventPayload>
  | WsEventEnvelope<"BUYER_ACCEPTANCE_EXPIRED", BuyerAcceptanceEventPayload>
  | WsEventEnvelope<"DELIVERY_SESSION_UPDATED", DeliverySessionUpdated>
  | DriverAvailabilityChangedEvent
  | DriverLocationUpdatedEvent
  | WsEventEnvelope<"MANIFEST_ORDER_INJECTED", ManifestOrderInjected>
  | WsEventEnvelope<"MANIFEST_ORDER_EXCEPTION", ManifestOrderException>
  | WsEventEnvelope<"MANIFEST_DLQ_ESCALATION", ManifestOrderException>
  | WsEventEnvelope<"MANIFEST_REBALANCED", ManifestRebalanced>
  | WsEventEnvelope<"MANIFEST_CANCELLED", ManifestCancelled>
  | WsEventEnvelope<"MANIFEST_SEALED", ManifestSealed>
  | WsEventEnvelope<"COMMAND_DISPATCHED", CommandLifecycle>
  | WsEventEnvelope<"COMMAND_RECEIVED", CommandLifecycle>
  | WsEventEnvelope<"COMMAND_SETTLED", CommandLifecycle>
  | WsEventEnvelope<"SYSTEM_APP_OUTDATED", { minimum_version: string }>;

