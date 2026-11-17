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
    let warehouseID: String?
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
    let agingBucket: String?
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
    let lotCode, lotID, recallReason, severity: String?
    let photoProofURL, signatureURL, sessionID, countryCode: String?
    let phone: String?
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
    let state, transferID, transferMode, decisionReason: String?
    let normalizedInn, requesterName, requesterPhone, roleSought: String?
    let targetTenantID, targetTenantType: String?
    let attemptCount, depth: Int?
    let escalated: Bool?
    let fromManifestID, fromVehicleID, manifestDomain: String?
    let stopCount: Int?
    let toManifestID, toVehicleID: String?
    let totalVolumeVu: TotalVolumeVu?
    let transferCount: Int?
    let bundleID, domain: String?
    let importedCount, processedRows: Int?
    let projectID: String?
    let quarantineCount: Int?
    let reportID: String?
    let totalRows: Int?
    let driverIDS, manifestIDS: [String]?
    let sharedRouteID, splitGroupID: String?
    let truckCount: Int?
    let conditionType: String?
    let gcsPaths: [String]?
    let notes: String?
    let quantity: Int?
    let reporterID, reporterRole, sku, deliveredAt: String?
    let disputeNote, proofPhotoURL, proofSignature, reasonCode: String?
    let cardMinor, cashMinor: Int?
    let executionAction, executionMode, gateway, policySource: String?
    let providerReference, transactionID, batchID: String?
    let netPayoutMinor: Int?
    let railReference, preOrderID, barcode, handlingClass: String?
    let isActive, isHazardous, isPerishable: Bool?
    let previousPriceMinor, priceMinor, stockQuantity: Int?
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
    let assignedTo, caseID, category, detectedAt: String?
    let exposureMinor: Int?
    let objectID, objectType, occurredAt, playbookAction: String?
    let resolvedAt, resolvedBy, ruleCode: String?
    let scoreBps: Int?
    let tenantID, approverID: String?
    let breakGlass: Bool?
    let comment, decisionID, delegatedFrom: String?
    let mfaVerified: Bool?
    let stepIndex: Int?
    let actorType, expiresAt, payloadHash, policyID: String?
    let policyVersion, requiredApprovers: Int?
    let anchoredAt, anchoredBy, dayBucket: String?
    let eventCount: Int?
    let headEventID, rootHash, signerKeyID, wormURI: String?
    let affectedLotIDS: [String]?
    let complianceRateBps, excursionCount, maxMilliC, minMilliC: Int?
    let openBreaches: Int?
    let readingID, sensorID: String?
    let tempMilliC, totalReadings: Int?
    let accountLabel, approvalID, completedAt, direction: String?
    let externalRef: String?
    let recordsProcessed: Int?
    let runID, streamName, triggeredBy: String?
    let balanceDeltaCount, balanceDeltaMinor, mismatchCount, missingExternalCount: Int?
    let missingInternalCount, stockDeltaCount: Int?
    let summaryJSON, startedAt: String?
    let prepayOnlyLocked: Bool?
    let score, newLimitMinor, previousLimitMinor: Int?
    let reasonCodesJSON, computedAt: String?
    let consentVerified: Bool?
    let featuresHash, modelVersion: String?
    let pdBps: Int?
    let scope, scoreID, scorerSource: String?
    let suggestedLimitMinor: Int?
    let createdBy, creditJournalID, debitJournalID, fromEntityID: String?
    let periodID, referenceID, referenceType, toEntityID: String?
    let transferType, counterpartyTin, entityID, esfID: String?
    let pegasusWSEventEnvelopeOperator, operatorRef, poID: String?
    let vatMinor, vatRateBps: Int?
    let createdAt, format, fromDate, packID: String?
    let packType: String?
    let rowCount: Int?
    let sha256Hex, toDate, challengerModel: String?
    let challengerWapeBps: Int?
    let championModel: String?
    let championWapeBps, horizonDays, locationCount: Int?
    let locationID: String?
    let skuCount, wapeImprovementBps: Int?
    let winnerModel, baseCurrency, journalEntryID, ratesJSON: String?
    let roundingDeltaMinor, unrealizedGainLossMinor: Int?
    let entryID, hash, reversesEntryID, sourceID: String?
    let sourceType: String?
    let totalCreditMinor, totalDebitMinor: Int?
    let grantedBy, roleName, assignedBy: String?
    let active: Bool?
    let displayName, externalID, resourceID, resourceType: String?
    let rolesSynced: [String]?
    let userName: String?
    let daysRemaining, daysToExpiry: Int?
    let expiryDate, layerID: String?
    let qtyRemaining, unitCostMinor, valueAtRiskMinor: Int?
    let glEntryID, movementID: String?
    let totalCostMinor: Int?
    let releasedLots: [Any?]?
    let actionMode, approvalRequestID: String?
    let confidenceBps, qty: Int?
    let sourceWarehouseID, targetWarehouseID, cellID, changeKind: String?
    let securityMarkings: [String]?
    let sourceEventID, sourceEventType, updatedAt, esfDocID: String?
    let lineCount: Int?
    let matchStatus, receiptID: String?
    let subtotalMinor, totalReceivedMinor, varianceMinor: Int?
    let varianceReason, vendorID, vendorInvoiceID, snapshotHash: String?
    let snapshotURI, timezone, agentID: String?
    let expectedImpactMinor: Int?
    let proposalType, roiEventID, targetObjectID, targetObjectType: String?
    let autonomyLevel: String?
    let estimatedImpactMinor, rollbackRateBps, soakDays, soakExecutedCount: Int?
    let actualMinor: Int?
    let breakID, controlTowerExceptionID: String?
    let deltaMinor: Int?
    let dimension, ownerID, slaDueAt: String?
    let breakCount: Int?
    let mode, eventID, evidenceRefID, evidenceRefType: String?
    let holdoutControl, isEstimate: Bool?
    let methodology: String?
    let savingsMinor: Int?
    let makerID, routePath, violationID: String?
    let creditExposureDeltaMinor, dispatchCostDeltaMinor, forecastDeltaMinor, netPnlDeltaMinor: Int?
    let replenishmentCostMinor: Int?
}

enum TotalVolumeVu {
    case double(Double)
    case integer(Int)
}

enum Version {
    case integer(Int)
    case string(String)
}
