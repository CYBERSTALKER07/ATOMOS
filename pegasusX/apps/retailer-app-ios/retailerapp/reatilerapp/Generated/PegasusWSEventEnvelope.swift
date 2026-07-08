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
    case creditDeliveryMarked
    case creditDeliveryResolved
    case deliveryDisputed
    case deliverySessionUpdated
    case demandBaselineUpdated
    case dispatchZoneOverride
    case driverAvailabilityChanged
    case driverCreated
    case driverLocationUpdated
    case driverReturnApproaching
    case factoryCreated
    case factoryLocationUpdated
    case factorySupplyRequestUpdate
    case freezeLockAcquired
    case freezeLockReleased
    case inventoryImportStatusUpdate
    case inventoryImportUploaded
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
    case orderAmended
    case orderAssigned
    case orderConditionReported
    case orderCreated
    case orderFinalized
    case orderReassigned
    case orderStatusChanged
    case orderValidationFailed
    case paymentCleared
    case paymentFailed
    case paymentRequired
    case planningAgentBroadcast
    case planningConfidenceDowngraded
    case planningForecastUpdated
    case planningMeioRecommendationV1
    case planningPromoSimulationReady
    case planningSignalIngestV1
    case preOrderAutoAccepted
    case preOrderCancelled
    case preOrderConfirmation
    case preOrderConfirmed
    case preOrderDateAccepted
    case preOrderDateProposed
    case preOrderDateRejected
    case preOrderEdited
    case preOrderNotified
    case preOrderNudge
    case productHandlingUpdated
    case promotionChanged
    case replenishmentAutoApproved
    case replenishmentInsightCreated
    case retailerCreditLimitBreached
    case retailerCreditProfileChanged
    case retailerPriceOverride
    case retailerRegistered
    case returnReceivedAtWarehouse
    case routeCreated
    case routeReordered
    case settlementRequired
    case shopClosed
    case shopClosedBypassOffload
    case shopClosedEscalated
    case shopClosedResolved
    case shopClosedResponse
    case splitPaymentCreated
    case supplierBillingConfigured
    case supplierBillingUpdated
    case supplierCreated
    case supplierMemberAdded
    case supplierProfileUpdated
    case supplierReturnCreated
    case supplierReturnResolved
    case supplierUpdated
    case supplyRequestAccepted
    case supplyRequestUpdate
    case supplyTransferApproaching
    case systemAppOutdated
    case vehicleAvailabilityChanged
    case vehicleCreated
    case warehouseCreated
    case warehouseDispatchLockChanged
    case warehouseLocationUpdated
    case warehouseSupplyRequestOpened
    case warehouseTransferCreated
    case warehouseTransferReceived
}
