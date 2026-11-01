// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse the JSON, add this file to your project and do:
//
//   let pegasusWSEventEnvelope = try PegasusWSEventEnvelope(json)

import Foundation

// MARK: - PegasusWSEventEnvelope
struct PegasusWSEventEnvelope {
    let aggregateID, aggregateType: String?
    let baseEvent: Any?
    let decidedBy, decision, note, recommendationID: String?
    let status, supplierID, action, actorID: String?
    let actorRole, attemptID, confirmationStatus, currency: String?
    let customerProgressPercent: Int?
    let customerStage, driverID, escalatedTo, fromDriverID: String?
    let fromRouteID: String?
    let gpsLat, gpsLng: Double?
    let h3Cell: String?
    let lat: Double?
    let licensePlate: String?
    let lineItems: Any?
    let lng: Double?
    let manifestID, message, negotiationID, orderID: String?
    let orderSource, paymentMethod, previousStatus, proposalID: String?
    let proposedPriceMinor: Int?
    let reason, receivingWindowClose, receivingWindowOpen, requestedDeliveryDate: String?
    let resolution, response, retailerID, routeID: String?
    let toDriverID, toRouteID: String?
    let totalMinor: Int?
    let vehicleID: String?
    let version: Version?
    let warehouseID, agingBucket: String?
    let amountMinor, balanceMinor: Int?
    let dueAt: String?
    let dunningStep: Int?
    let invoiceID, lastDunnedAt: String?
    let principalMinor: Int?
    let deadline, ehfID, accountHolder, assignedWarehouseID: String?
    let bankName, contactName, email, legalName: String?
    let selectedGateways: [String]?
    let supplierRole, userID: String?
    let expectedMinor, overageMinor, receivedMinor, shortfallMinor: Int?
    let traceID, categoryID, iconKey, name: String?
    let parentCategoryID: String?
    let sortOrder: Int?
    let chargebackID, claimID, claimType: String?
    let photoUrls: [String]?
    let resolutionNote, settlementMode, source, commandID: String?
    let actor, campaignID: String?
    let impactedLotCount, impactedOrderCount, impactedUnitsCount: Int?
    let lotCode, lotID, productID, recallReason: String?
    let severity, photoProofURL, signatureURL, sessionID: String?
    let baselineQty: Int?
    let baselineSource, blockedReason: String?
    let confidence: Double?
    let confidencePct: Int?
    let factoryID: String?
    let highUnits: Int?
    let insightID: String?
    let lowUnits, networkNodes: Int?
    let overrideID, polygonGeojson, publishedBy, scenarioID: String?
    let signalID, simulationID: String?
    let transferRecommendations, ttlSeconds: Int?
    let countryCode, phone: String?
    let orderCount: Int?
    let orderIDS: [String]?
    let available: Bool?
    let homeNodeID, homeNodeType: String?
    let onShift: Bool?
    let returnID: String?
    let itemCount: Int?
    let requestID, errorCode, errorMessage, fiscalQr: String?
    let fiscalReceiptID, provider, lockName, systemID: String?
    let gcsPath: String?
    let suggestedMappings, committedUnits, coverageDays: Int?
    let coverageStartDate, linkedTransferID, lockID: String?
    let pendingConfirmationUnits, projectedUnits: Int?
    let requestedBy: String?
    let requestedUnits: Int?
    let state, transferID, transferMode: String?
    let attemptCount, depth: Int?
    let escalated: Bool?
    let fromManifestID, fromVehicleID, manifestDomain: String?
    let stopCount: Int?
    let toManifestID, toVehicleID: String?
    let totalVolumeVu: TotalVolumeVu?
    let transferCount: Int?
    let driverIDS, manifestIDS: [String]?
    let sharedRouteID, splitGroupID: String?
    let truckCount: Int?
    let conditionType: String?
    let gcsPaths: [String]?
    let notes: String?
    let quantity: Int?
    let reportID, reporterID, reporterRole, sku: String?
    let deliveredAt, disputeNote, proofPhotoURL, proofSignature: String?
    let reasonCode: String?
    let cardMinor, cashMinor: Int?
    let executionAction, executionMode, gateway, policySource: String?
    let providerReference, transactionID, batchID: String?
    let netPayoutMinor: Int?
    let railReference, preOrderID, barcode, handlingClass: String?
    let isActive, isHazardous, isPerishable: Bool?
    let previousPriceMinor, priceMinor: Int?
    let requiresColdChain: Bool?
    let stockQuantity: Int?
    let storageTempMaxC, storageTempMinC: Double?
    let unit, promotionID: String?
    let retailerIDS: [String]?
    let retailerScope: String?
    let creditLimitMinor, currentBalance, requestedAmount: Int?
    let delinquent: Bool?
    let profileID, riskTier, setBy, setByRole: String?
    let assignedFactoryID: String?
    let categories: [String]?
    let country: String?
    let isConfigured, isRegistered: Bool?
    let breachReason, breachedAt, cutoffAppliedAt: String?
    let fillRateTargetBps: Int?
    let guaranteedDeliveryDate: String?
    let minOrderMinor: Int?
    let promiseType: String?
    let slaHours: Int?
    let fromWarehouse, toWarehouse, unavailableNote, unavailableReason: String?
}

enum TotalVolumeVu {
    case double(Double)
    case integer(Int)
}

enum Version {
    case integer(Int)
    case string(String)
}
