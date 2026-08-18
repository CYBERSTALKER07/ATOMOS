// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse the JSON, add this file to your project and do:
//
//   let pegasusWSEventEnvelope = try PegasusWSEventEnvelope(json)

import Foundation

// MARK: - PegasusWSEventEnvelope
struct PegasusWSEventEnvelope {
    let type: TypeEnum
    let aggregateID, aggregateType: String?
    let baseEvent: Any?
    let decidedBy, decision, note, recommendationID: String?
    let status, supplierID, action, actorID: String?
    let actorRole, attemptID, confirmationStatus, currency: String?
    let driverID, escalatedTo, fromDriverID, fromRouteID: String?
    let gpsLat, gpsLng: Double?
    let h3Cell: String?
    let lat: Double?
    let licensePlate: String?
    let lineItems: Any?
    let lng: Double?
    let manifestID, negotiationID, orderID, orderSource: String?
    let paymentMethod, previousStatus, proposalID: String?
    let proposedPriceMinor: Int?
    let reason, receivingWindowClose, receivingWindowOpen, requestedDeliveryDate: String?
    let resolution, response, retailerID, routeID: String?
    let toDriverID, toRouteID: String?
    let totalMinor: Int?
    let vehicleID: String?
    let version: Version?
    let warehouseID, deadline, ehfID, accountHolder: String?
    let assignedWarehouseID, bankName, contactName, email: String?
    let legalName: String?
    let selectedGateways: [String]?
    let supplierRole, userID: String?
    let expectedMinor, overageMinor, receivedMinor, shortfallMinor: Int?
    let traceID: String?
    let amountMinor: Int?
    let chargebackID, claimID, claimType: String?
    let photoUrls: [String]?
    let resolutionNote, settlementMode, source, commandID: String?
    let photoProofURL, signatureURL, sessionID: String?
    let baselineQty: Int?
    let baselineSource, blockedReason: String?
    let confidence: Double?
    let confidencePct: Int?
    let factoryID: String?
    let highUnits: Int?
    let insightID: String?
    let lowUnits, networkNodes: Int?
    let overrideID, polygonGeojson, productID, publishedBy: String?
    let scenarioID, signalID, simulationID: String?
    let transferRecommendations, ttlSeconds: Int?
    let countryCode, name, phone: String?
    let available: Bool?
    let homeNodeID, homeNodeType: String?
    let onShift: Bool?
    let returnID: String?
    let itemCount: Int?
    let requestID, errorCode, errorMessage, fiscalQr: String?
    let fiscalReceiptID, provider, lockName, systemID: String?
    let gcsPath: String?
    let suggestedMappings, attemptCount, depth: Int?
    let escalated: Bool?
    let fromManifestID, fromVehicleID, manifestDomain: String?
    let orderCount: Int?
    let state: String?
    let stopCount: Int?
    let toManifestID, toVehicleID: String?
    let totalVolumeVu: TotalVolumeVu?
    let transferCount: Int?
    let transferID: String?
    let driverIDS, manifestIDS, orderIDS: [String]?
    let sharedRouteID, splitGroupID: String?
    let truckCount: Int?
    let conditionType: String?
    let gcsPaths: [String]?
    let notes: String?
    let quantity: Int?
    let reportID, reporterID, reporterRole, sku: String?
    let reasonCode, executionAction, executionMode, gateway: String?
    let policySource, providerReference, transactionID, preOrderID: String?
    let handlingClass: String?
    let isHazardous, isPerishable, requiresColdChain: Bool?
    let storageTempMaxC, storageTempMinC: Double?
    let promotionID: String?
    let retailerIDS: [String]?
    let retailerScope: String?
    let creditLimitMinor, currentBalance, requestedAmount: Int?
    let delinquent: Bool?
    let profileID, riskTier: String?
    let priceMinor: Int?
    let setBy, setByRole, assignedFactoryID: String?
    let categories: [String]?
    let country: String?
    let isConfigured, isRegistered: Bool?
    let committedUnits, coverageDays: Int?
    let coverageStartDate, linkedTransferID, lockID: String?
    let pendingConfirmationUnits, projectedUnits: Int?
    let requestedBy: String?
    let requestedUnits: Int?
    let transferMode, fromWarehouse, toWarehouse: String?
    let isActive: Bool?
    let unavailableNote, unavailableReason: String?
}

enum TotalVolumeVu {
    case double(Double)
    case integer(Int)
}

enum TypeEnum: String {
    case aiRecommendationCreated
    case aiRecommendationDecided
    case allocationFairShareApplied
    case allocationPolicyApplied
    case arInvoiceAgingUpdated
    case arInvoiceDunned
    case arInvoiceOpened
    case arInvoicePayment
    case arInvoiceSettled
    case buyerAcceptanceAccepted
    case buyerAcceptanceExpired
    case buyerAcceptancePending
    case buyerAcceptanceRejected
    case cartSyncUpdated
    case cashOverage
    case cashShortfall
    case claimFiled
    case claimResolved
    case claimUnderReview
    case commandDispatched
    case commandReceived
    case commandSettled
    case controlTowerPlaybookChanged
    case controlTowerRunCreated
    case controlTowerRunUpdated
    case creditDeliveryMarked
    case creditDeliveryResolved
    case creditLeave
    case deliveryDisputed
    case deliverySessionUpdated
    case demandBaselineUpdated
    case demandSignal
    case dispatchZoneOverride
    case driverAvailabilityChanged
    case driverCreated
    case driverLocationUpdated
    case driverReturnApproaching
    case factoryCreated
    case factoryLocationUpdated
    case factorySlaBreach
    case factoryStaffCreated
    case factoryStaffPasswordSet
    case factorySupplyRequestUpdate
    case fiscalCorrectiveRequested
    case fiscalReceiptFailed
    case fiscalReceiptRequested
    case fiscalReceiptSucceeded
    case freezeLockAcquired
    case freezeLockReleased
    case inventoryImportStatusUpdate
    case inventoryImportUploaded
    case inventoryPolicyUpdated
    case inventoryQuantityUpdated
    case inventorySyncComplete
    case logisticsExceptionReported
    case logisticsTelemetry
    case lookAheadCompleted
    case loyaltyPointsEarned
    case manifestCancelled
    case manifestCompleted
    case manifestDispatched
    case manifestDlqEscalation
    case manifestDraftCreated
    case manifestExceptionResolved
    case manifestLoadingStarted
    case manifestOrderException
    case manifestOrderInjected
    case manifestRebalanced
    case manifestSealed
    case missingItemsReported
    case negotiationProposed
    case negotiationResolved
    case networkModeChanged
    case orderAllocated
    case orderAmended
    case orderAssigned
    case orderCapacityOverflow
    case orderConditionReported
    case orderCreated
    case orderFinalized
    case orderForceCompleted
    case orderReassigned
    case orderStatusChanged
    case orderValidationFailed
    case parentOrderCreated
    case parentOrderUpdated
    case partialOffload
    case paymentCleared
    case paymentFailed
    case paymentRequired
    case payoutBatchDispatched
    case payoutBatchExported
    case payoutBatchGenerated
    case payoutBatchPaid
    case payoutPolicyUpdated
    case planningAgentBroadcast
    case planningConfidenceDowngraded
    case planningForecastUpdated
    case planningMeioRecommendationV1
    case planningPromoSimulationReady
    case planningScenarioPublishedV1
    case planningSignalIngestV1
    case posSaleCompleted
    case posSaleVoided
    case posSessionClosed
    case posSessionOpened
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
    case proximityUnlocked
    case pullMatrixCompleted
    case refundFailed
    case refundRequested
    case refundSucceeded
    case replenishmentAutoApproved
    case replenishmentInsightCreated
    case replenishmentPolicyUpdated
    case retailerAssistSlaBreached
    case retailerAssistTicketCancelled
    case retailerAssistTicketClaimed
    case retailerAssistTicketCompleted
    case retailerAssistTicketOpened
    case retailerAutoOrderUpdated
    case retailerCapabilityPackChanged
    case retailerClockIn
    case retailerClockOut
    case retailerCreditLimitBreached
    case retailerCreditProfileChanged
    case retailerLocationCreated
    case retailerLocationUpdated
    case retailerPriceOverride
    case retailerRegistered
    case retailerSectionCreated
    case retailerSectionSkuMapped
    case retailerSectionUpdated
    case retailerSegmentUpdated
    case retailerSellThroughUpdated
    case retailerShiftCashVariance
    case retailerShiftClosed
    case retailerShiftOpened
    case retailerStaffCreated
    case retailerStaffSectionAssigned
    case returnReceivedAtWarehouse
    case returnScanReceived
    case reverseLogisticsRequired
    case routeCreated
    case routeReordered
    case servicePolicyUpdated
    case settlementRequired
    case shopClosed
    case shopClosedBypassOffload
    case shopClosedEscalated
    case shopClosedResolved
    case shopClosedResponse
    case shopClosedTimeout
    case skuClassUpdated
    case splitPaymentCreated
    case splitShipmentCreated
    case storeStockAdjusted
    case storeStockClaimHold
    case storeStockCounted
    case storeStockReceived
    case storeStockTransferred
    case supplierBillingConfigured
    case supplierBillingUpdated
    case supplierBroadcast
    case supplierCreated
    case supplierCreditProgramChanged
    case supplierCreditTermsChanged
    case supplierMemberAdded
    case supplierProfileUpdated
    case supplierReturnCreated
    case supplierReturnResolved
    case supplierUpdated
    case supplyRequestAccepted
    case supplyRequestUpdate
    case supplyTransferApproaching
    case supplyTransferArrived
    case systemAppOutdated
    case transferCreated
    case vehicleAvailabilityChanged
    case vehicleCreated
    case warehouseBroadcast
    case warehouseCreated
    case warehouseDispatchLockChanged
    case warehouseLocationUpdated
    case warehouseSupplyRequestOpened
    case warehouseTransferCreated
    case warehouseTransferReceived
    case wmsCycleApproved
    case wmsPickConfirmed
    case wmsPutaway
    case wmsTemperatureBreach
}

enum Version {
    case integer(Int)
    case string(String)
}
