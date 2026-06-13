// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse the JSON, add this file to your project and do:
//
//   let pegasusWSEventEnvelope = try PegasusWSEventEnvelope(json)

import Foundation

// MARK: - PegasusWSEventEnvelope
struct PegasusWSEventEnvelope {
    let type: TypeEnum
}

enum TypeEnum: String {
    case aiRecommendationCreated
    case aiRecommendationDecided
    case cartSyncUpdated
    case commandDispatched
    case commandReceived
    case commandSettled
    case deliveryDisputed
    case deliverySessionUpdated
    case driverAvailabilityChanged
    case driverCreated
    case driverLocationUpdated
    case factoryCreated
    case freezeLockAcquired
    case freezeLockReleased
    case inventorySyncComplete
    case manifestCancelled
    case manifestCompleted
    case manifestDispatched
    case manifestDlqEscalation
    case manifestDraftCreated
    case manifestLoadingStarted
    case manifestOrderException
    case manifestOrderInjected
    case manifestRebalanced
    case manifestSealed
    case missingItemsReported
    case negotiationProposed
    case negotiationResolved
    case orderAssigned
    case orderCreated
    case orderFinalized
    case orderReassigned
    case orderStatusChanged
    case orderValidationFailed
    case paymentCleared
    case paymentRequired
    case promotionChanged
    case retailerRegistered
    case routeCreated
    case routeReordered
    case settlementRequired
    case shopClosed
    case shopClosedEscalated
    case shopClosedResolved
    case shopClosedResponse
    case splitPaymentCreated
    case supplierBillingConfigured
    case supplierBillingUpdated
    case supplierCreated
    case supplierMemberAdded
    case supplierProfileUpdated
    case supplierUpdated
    case systemAppOutdated
    case vehicleCreated
    case warehouseCreated
    case warehouseDispatchLockChanged
    case warehouseSupplyRequestOpened
    case warehouseTransferCreated
    case warehouseTransferReceived
}
