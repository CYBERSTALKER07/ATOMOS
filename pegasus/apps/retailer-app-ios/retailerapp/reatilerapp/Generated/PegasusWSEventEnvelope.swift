// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse the JSON, add this file to your project and do:
//
//   let pegasusWSEventEnvelope = try PegasusWSEventEnvelope(json)

import Foundation

// MARK: - PegasusWSEventEnvelope
struct PegasusWSEventEnvelope {
    let type: TypeEnum
    let commandID, commandState, eventType, targetID: String?
    let targetRole: String?
    let timestamp: Date?
    let traceID, ackByRole, ackByUserID: String?
    let adjustedAmount: Int?
    let currency, deliverySessionID: String?
    let deltaAmount: Int?
    let gateway, orderID: String?
    let originalAmount: Int?
    let providerRefundID, refundID, retailerID, sessionID: String?
    let status, supplierID, disputedBy, driverID: String?
    let reason: String?
    let feeAmount, feeBasisPoints: Int?
    let feeCapApplied: Bool?
    let feePolicyVersion: String?
    let netPayoutAmount: Int?
    let selectedTierKey, state: String?
    let skuCount: Int?
    let warehouseID: String?
    let available: Bool?
    let note, truckID, createdBy, driverType: String?
    let homeNodeID, homeNodeType, name, phone: String?
    let orderIDS: [String]?
    let routeID, factoryID, h3Index: String?
    let lat: Double?
    let leadTimeDays: Int?
    let lng: Double?
    let productTypes: [String]?
    let productionCapacityVu: Double?
    let regionCode: String?
    let warehousesLinked: Int?
    let escalationLevel, replacementTransferID: String?
    let slaBreachMinutes: Int?
    let transferID: String?
    let globalOrderCount, milestoneIndex, milestoneOrderCount, newFeeBasisPoints: Int?
    let previousFeeBasisPoints: Int?
    let triggerOrderID, geoZone, manifestID: String?
    let count24H, quota: Int?
    let sealedBy: String?
    let itemsCount: Int?
    let receivedBy, confirmedBy: String?
    let volumeVu: Double?
    let suggestedMappings: Int?
    let gcsPath: String?
    let durationMS, horizonDays: Int?
    let runID, source, cancelledBy: String?
    let releasedIDS: [String]?
    let releasedKind: String?
    let attemptCount: Int?
    let escalated: Bool?
    let exceptionID, metadata, injectedBy: String?
    let newTotalVolumeVu: Double?
    let newDriverID, oldDriverID, reassignedBy, sourceManifestID: String?
    let targetManifestID, rebalancedBy: String?
    let transferIDS: [String]?
    let proposalID, action, changedBy, newMode: String?
    let oldMode: String?
    let amount: Int?
    let paymentMethod, jobID: String?
    let matrixSize: Int?
    let producedAt, solverType: String?
    let timedOut: Bool?
    let warnings: [String]?
    let fiscalSign, invoiceID: String?
    let items: [[String: Any?]]?
    let receiptType, terminalID, tin: String?
    let total: Int?
    let warehouseName, amendmentID: String?
    let newAmount, refunded: Int?
    let newRouteID, oldRouteID: String?
    let distanceKM, newLoadPercent: Double?
    let newWarehouseID: String?
    let originalLoadPercent: Double?
    let originalWarehouseID: String?
    let retailerLat, retailerLng: Double?
    let newState, oldState: String?
    let shortfallMap: [String: Int]?
    let deliveryToken, deliveryDate, editedBy, newDate: String?
    let skusProcessed, transfersGenerated: Int?
    let overrideID: String?
    let price: Int?
    let setBy, setByRole, skuID, h3Cell: String?
    let ownerName, phoneNumber, shopName: String?
    let stopCount: Int?
    let routeJSON, paymentSessionID: String?
    let grossAmount: Int?
    let originalSliceID, payoutOwnerID, payoutOwnerType, revisionReason: String?
    let revisionSliceID, settlementTarget, attemptID: String?
    let gpsLat, gpsLng: Double?
    let escalatedTo, resolution, resolvedBy, response: String?
    let backorderID, backorderOrderID: String?
    let currentStock: Int?
    let productID: String?
    let safetyLevel: Int?
    let operatingCategories: [String]?
    let taxID, companyName, email, laneID: String?
    let newDampenedHours, oldDampenedHours, rawTransitHours: Double?
    let unassignedBy: String?
    let orderCount: Int?
    let label, licensePlate: String?
    let maxVolumeVu: Double?
    let vehicleClass, vehicleID: String?
    let coverageRadiusKM: Double?
    let h3Count, newH3Count, oldH3Count: Int?
    let field: String?
    let newValue, oldValue: Bool?
}

enum TypeEnum: String {
    case aiOrderConfirmed
    case aiOrderRejected
    case aiPlanDateShift
    case aiPlanSkuModified
    case aiPrediction
    case aiPredictionCorrected
    case bypassTokenIssued
    case cancelApproved
    case cancelRequested
    case cartSyncUpdated
    case cashCollectionRequired
    case commandDispatched
    case commandReceived
    case commandSettled
    case creditDeliveryMarked
    case creditDeliveryResolved
    case deliveryDeltaRefunded
    case deliveryDisputed
    case deliverySessionUpdated
    case demandForecastReady
    case dispatchLockAcquired
    case dispatchLockChange
    case dispatchLockReleased
    case driverApproaching
    case driverArrived
    case driverAvailabilityChanged
    case driverCreated
    case earlyCompleteApproved
    case earlyCompleteRequested
    case etaUpdated
    case factoryCreated
    case factoryManifestCreated
    case factoryManifestUpdate
    case factoryOutboxFailed
    case factorySlaBreach
    case factorySupplyRequestUpdate
    case factoryTransferUpdate
    case feeRateAdjusted
    case fleetDispatched
    case forceSealAlert
    case freezeLockAcquired
    case freezeLockReleased
    case fulfillmentPaid
    case fulfillmentPaymentCompleted
    case inboundFreightUnannounced
    case insightApprovedTransferCreated
    case internalLoadConfirmed
    case inventoryImportStatusUpdate
    case inventoryImportUploaded
    case inventorySyncComplete
    case lookAheadCompleted
    case manifestCancelled
    case manifestCompleted
    case manifestDispatched
    case manifestDlqEscalation
    case manifestDraftCreated
    case manifestForceSealed
    case manifestLoadingStarted
    case manifestOrderException
    case manifestOrderInjected
    case manifestOrderReassigned
    case manifestRebalanced
    case manifestSealed
    case manifestSettled
    case missingItemsReported
    case negotiationProposed
    case negotiationResolved
    case networkModeChanged
    case offloadConfirmed
    case optimizationJobQueued
    case optimizationSolved
    case orderAmended
    case orderAssigned
    case orderCancelLocked
    case orderCancelled
    case orderCancelledByOrigin
    case orderCompleted
    case orderCreated
    case orderDelayed
    case orderDispatched
    case orderFinalized
    case orderModified
    case orderReassigned
    case orderRejectedBySupplier
    case orderRerouted
    case orderStateChanged
    case orderStatusChanged
    case orderSync
    case orderValidationFailed
    case outOfStock
    case outboxFailed
    case payloadOverflow
    case payloadReadyToSeal
    case payloadSealed
    case payloadSync
    case paymentBypassCompleted
    case paymentBypassIssued
    case paymentCleared
    case paymentExpired
    case paymentFailed
    case paymentGatewayDegraded
    case paymentIntentCreated
    case paymentRefunded
    case paymentRequired
    case paymentSettled
    case powerOutageReported
    case preOrderAutoAccepted
    case preOrderCancelled
    case preOrderConfirmation
    case preOrderConfirmed
    case preOrderEdited
    case preOrderNotified
    case preOrderNudge
    case pullMatrixCompleted
    case replenishmentLockAcquired
    case replenishmentLockReleased
    case replenishmentTransferCreated
    case retailerPriceOverride
    case retailerRegistered
    case returnResolved
    case routeCreated
    case routeFinalized
    case settlementRequired
    case settlementRevised
    case shopClosed
    case shopClosedAlert
    case shopClosedEscalated
    case shopClosedResolved
    case shopClosedResponse
    case smsQuickComplete
    case splitPaymentCreated
    case stockBackordered
    case stockThresholdBreach
    case supplierConfigured
    case supplierRegistered
    case supplyLaneTransitUpdated
    case supplyRequestAcknowledged
    case supplyRequestCancelled
    case supplyRequestFulfilled
    case supplyRequestReady
    case supplyRequestSubmitted
    case supplyRequestUpdate
    case systemAppOutdated
    case systemBroadcast
    case tokenRefreshNeeded
    case transferApproved
    case transferReceived
    case transferStateChanged
    case transferUnassigned
    case unifiedCheckoutCompleted
    case vehicleCreated
    case warehouseCreated
    case warehouseSpatialUpdated
    case warehouseStatusChanged
}
