// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse the JSON, add this file to your project and do:
//
//   let pegasusWSEventEnvelope = try PegasusWSEventEnvelope(json)

import Foundation

// MARK: - PegasusWSEventEnvelope
struct PegasusWSEventEnvelope: Codable {
    let type: TypeEnum
}

// MARK: PegasusWSEventEnvelope convenience initializers and mutators

extension PegasusWSEventEnvelope {
    init(data: Data) throws {
        self = try newJSONDecoder().decode(PegasusWSEventEnvelope.self, from: data)
    }

    init(_ json: String, using encoding: String.Encoding = .utf8) throws {
        guard let data = json.data(using: encoding) else {
            throw NSError(domain: "JSONDecoding", code: 0, userInfo: nil)
        }
        try self.init(data: data)
    }

    init(fromURL url: URL) throws {
        try self.init(data: try Data(contentsOf: url))
    }

    func with(
        type: TypeEnum? = nil
    ) -> PegasusWSEventEnvelope {
        return PegasusWSEventEnvelope(
            type: type ?? self.type
        )
    }

    func jsonData() throws -> Data {
        return try newJSONEncoder().encode(self)
    }

    func jsonString(encoding: String.Encoding = .utf8) throws -> String? {
        return String(data: try self.jsonData(), encoding: encoding)
    }
}

enum TypeEnum: String, Codable {
    case aiRecommendationCreated = "AI_RECOMMENDATION_CREATED"
    case aiRecommendationDecided = "AI_RECOMMENDATION_DECIDED"
    case cartSyncUpdated = "CART_SYNC_UPDATED"
    case commandDispatched = "COMMAND_DISPATCHED"
    case commandReceived = "COMMAND_RECEIVED"
    case commandSettled = "COMMAND_SETTLED"
    case deliveryDisputed = "DELIVERY_DISPUTED"
    case deliverySessionUpdated = "DELIVERY_SESSION_UPDATED"
    case driverAvailabilityChanged = "DRIVER_AVAILABILITY_CHANGED"
    case driverCreated = "DRIVER_CREATED"
    case driverLocationUpdated = "DRIVER_LOCATION_UPDATED"
    case driverReturnApproaching = "DRIVER_RETURN_APPROACHING"
    case factoryCreated = "FACTORY_CREATED"
    case factorySupplyRequestUpdate = "FACTORY_SUPPLY_REQUEST_UPDATE"
    case freezeLockAcquired = "FREEZE_LOCK_ACQUIRED"
    case freezeLockReleased = "FREEZE_LOCK_RELEASED"
    case inventoryImportStatusUpdate = "INVENTORY_IMPORT_STATUS_UPDATE"
    case inventoryImportUploaded = "INVENTORY_IMPORT_UPLOADED"
    case inventorySyncComplete = "INVENTORY_SYNC_COMPLETE"
    case manifestCancelled = "MANIFEST_CANCELLED"
    case manifestCompleted = "MANIFEST_COMPLETED"
    case manifestDispatched = "MANIFEST_DISPATCHED"
    case manifestDlqEscalation = "MANIFEST_DLQ_ESCALATION"
    case manifestDraftCreated = "MANIFEST_DRAFT_CREATED"
    case manifestLoadingStarted = "MANIFEST_LOADING_STARTED"
    case manifestOrderException = "MANIFEST_ORDER_EXCEPTION"
    case manifestOrderInjected = "MANIFEST_ORDER_INJECTED"
    case manifestRebalanced = "MANIFEST_REBALANCED"
    case manifestSealed = "MANIFEST_SEALED"
    case missingItemsReported = "MISSING_ITEMS_REPORTED"
    case negotiationProposed = "NEGOTIATION_PROPOSED"
    case negotiationResolved = "NEGOTIATION_RESOLVED"
    case orderAmended = "ORDER_AMENDED"
    case orderAssigned = "ORDER_ASSIGNED"
    case orderCreated = "ORDER_CREATED"
    case orderFinalized = "ORDER_FINALIZED"
    case orderReassigned = "ORDER_REASSIGNED"
    case orderStatusChanged = "ORDER_STATUS_CHANGED"
    case orderValidationFailed = "ORDER_VALIDATION_FAILED"
    case paymentCleared = "PAYMENT_CLEARED"
    case paymentRequired = "PAYMENT_REQUIRED"
    case promotionChanged = "PROMOTION_CHANGED"
    case retailerPriceOverride = "RETAILER_PRICE_OVERRIDE"
    case retailerRegistered = "RETAILER_REGISTERED"
    case returnReceivedAtWarehouse = "RETURN_RECEIVED_AT_WAREHOUSE"
    case routeCreated = "ROUTE_CREATED"
    case routeReordered = "ROUTE_REORDERED"
    case settlementRequired = "SETTLEMENT_REQUIRED"
    case shopClosed = "SHOP_CLOSED"
    case shopClosedEscalated = "SHOP_CLOSED_ESCALATED"
    case shopClosedResolved = "SHOP_CLOSED_RESOLVED"
    case shopClosedResponse = "SHOP_CLOSED_RESPONSE"
    case splitPaymentCreated = "SPLIT_PAYMENT_CREATED"
    case supplierBillingConfigured = "SUPPLIER_BILLING_CONFIGURED"
    case supplierBillingUpdated = "SUPPLIER_BILLING_UPDATED"
    case supplierCreated = "SUPPLIER_CREATED"
    case supplierMemberAdded = "SUPPLIER_MEMBER_ADDED"
    case supplierProfileUpdated = "SUPPLIER_PROFILE_UPDATED"
    case supplierReturnCreated = "SUPPLIER_RETURN_CREATED"
    case supplierReturnResolved = "SUPPLIER_RETURN_RESOLVED"
    case supplierUpdated = "SUPPLIER_UPDATED"
    case supplyRequestAccepted = "SUPPLY_REQUEST_ACCEPTED"
    case supplyRequestUpdate = "SUPPLY_REQUEST_UPDATE"
    case supplyTransferApproaching = "SUPPLY_TRANSFER_APPROACHING"
    case systemAppOutdated = "SYSTEM_APP_OUTDATED"
    case vehicleAvailabilityChanged = "VEHICLE_AVAILABILITY_CHANGED"
    case vehicleCreated = "VEHICLE_CREATED"
    case warehouseCreated = "WAREHOUSE_CREATED"
    case warehouseDispatchLockChanged = "WAREHOUSE_DISPATCH_LOCK_CHANGED"
    case warehouseSupplyRequestOpened = "WAREHOUSE_SUPPLY_REQUEST_OPENED"
    case warehouseTransferCreated = "WAREHOUSE_TRANSFER_CREATED"
    case warehouseTransferReceived = "WAREHOUSE_TRANSFER_RECEIVED"
}

// MARK: - Helper functions for creating encoders and decoders

func newJSONDecoder() -> JSONDecoder {
    let decoder = JSONDecoder()
    if #available(iOS 10.0, OSX 10.12, tvOS 10.0, watchOS 3.0, *) {
        decoder.dateDecodingStrategy = .iso8601
    }
    return decoder
}

func newJSONEncoder() -> JSONEncoder {
    let encoder = JSONEncoder()
    if #available(iOS 10.0, OSX 10.12, tvOS 10.0, watchOS 3.0, *) {
        encoder.dateEncodingStrategy = .iso8601
    }
    return encoder
}
