// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse the JSON, add this file to your project and do:
//
//   let pegasusWSEventEnvelope = try PegasusWSEventEnvelope(json)

import Foundation

// MARK: - PegasusWSEventEnvelope
struct PegasusWSEventEnvelope: Codable {
    let aggregateID, aggregateType: String?
    let baseEvent: JSONAny
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
    let lineItems: JSONAny?
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
    let releasedLots: [JSONAny]?
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

    enum CodingKeys: String, CodingKey {
        case aggregateID = "aggregate_id"
        case aggregateType = "aggregate_type"
        case baseEvent = "base_event"
        case decidedBy = "decided_by"
        case decision, note
        case recommendationID = "recommendation_id"
        case status
        case supplierID = "supplier_id"
        case action
        case actorID = "actor_id"
        case actorRole = "actor_role"
        case attemptID = "attempt_id"
        case confirmationStatus = "confirmation_status"
        case currency
        case customerProgressPercent = "customer_progress_percent"
        case customerStage = "customer_stage"
        case driverID = "driver_id"
        case escalatedTo = "escalated_to"
        case fromDriverID = "from_driver_id"
        case fromRouteID = "from_route_id"
        case gpsLat = "gps_lat"
        case gpsLng = "gps_lng"
        case h3Cell = "h3_cell"
        case lat
        case licensePlate = "license_plate"
        case lineItems = "line_items"
        case lng
        case manifestID = "manifest_id"
        case message
        case negotiationID = "negotiation_id"
        case orderID = "order_id"
        case orderSource = "order_source"
        case paymentMethod = "payment_method"
        case previousStatus = "previous_status"
        case proposalID = "proposal_id"
        case proposedPriceMinor = "proposed_price_minor"
        case reason
        case receivingWindowClose = "receiving_window_close"
        case receivingWindowOpen = "receiving_window_open"
        case requestedDeliveryDate = "requested_delivery_date"
        case resolution, response
        case retailerID = "retailer_id"
        case routeID = "route_id"
        case toDriverID = "to_driver_id"
        case toRouteID = "to_route_id"
        case totalMinor = "total_minor"
        case vehicleID = "vehicle_id"
        case version
        case warehouseID = "warehouse_id"
        case baselineQty = "baseline_qty"
        case baselineSource = "baseline_source"
        case blockedReason = "blocked_reason"
        case confidence
        case confidencePct = "confidence_pct"
        case factoryID = "factory_id"
        case highUnits = "high_units"
        case insightID = "insight_id"
        case lowUnits = "low_units"
        case networkNodes = "network_nodes"
        case overrideID = "override_id"
        case polygonGeojson = "polygon_geojson"
        case productID = "product_id"
        case publishedBy = "published_by"
        case scenarioID = "scenario_id"
        case signalID = "signal_id"
        case simulationID = "simulation_id"
        case transferRecommendations = "transfer_recommendations"
        case ttlSeconds = "ttl_seconds"
        case agingBucket = "aging_bucket"
        case amountMinor = "amount_minor"
        case balanceMinor = "balance_minor"
        case dueAt = "due_at"
        case dunningStep = "dunning_step"
        case invoiceID = "invoice_id"
        case lastDunnedAt = "last_dunned_at"
        case principalMinor = "principal_minor"
        case deadline
        case ehfID = "ehf_id"
        case accountHolder = "account_holder"
        case assignedWarehouseID = "assigned_warehouse_id"
        case bankName = "bank_name"
        case contactName = "contact_name"
        case email
        case legalName = "legal_name"
        case selectedGateways = "selected_gateways"
        case supplierRole = "supplier_role"
        case userID = "user_id"
        case expectedMinor = "expected_minor"
        case overageMinor = "overage_minor"
        case receivedMinor = "received_minor"
        case shortfallMinor = "shortfall_minor"
        case traceID = "trace_id"
        case categoryID = "category_id"
        case iconKey = "icon_key"
        case name
        case parentCategoryID = "parent_category_id"
        case sortOrder = "sort_order"
        case chargebackID = "chargeback_id"
        case claimID = "claim_id"
        case claimType = "claim_type"
        case photoUrls = "photo_urls"
        case resolutionNote = "resolution_note"
        case settlementMode = "settlement_mode"
        case source
        case commandID = "command_id"
        case actor
        case campaignID = "campaign_id"
        case impactedLotCount = "impacted_lot_count"
        case impactedOrderCount = "impacted_order_count"
        case impactedUnitsCount = "impacted_units_count"
        case lotCode = "lot_code"
        case lotID = "lot_id"
        case recallReason = "recall_reason"
        case severity
        case photoProofURL = "photo_proof_url"
        case signatureURL = "signature_url"
        case sessionID = "session_id"
        case countryCode = "country_code"
        case phone
        case orderCount = "order_count"
        case orderIDS = "order_ids"
        case available
        case homeNodeID = "home_node_id"
        case homeNodeType = "home_node_type"
        case onShift = "on_shift"
        case returnID = "return_id"
        case itemCount = "item_count"
        case requestID = "request_id"
        case errorCode = "error_code"
        case errorMessage = "error_message"
        case fiscalQr = "fiscal_qr"
        case fiscalReceiptID = "fiscal_receipt_id"
        case provider
        case lockName = "lock_name"
        case systemID = "system_id"
        case gcsPath = "gcs_path"
        case suggestedMappings = "suggested_mappings"
        case committedUnits = "committed_units"
        case coverageDays = "coverage_days"
        case coverageStartDate = "coverage_start_date"
        case linkedTransferID = "linked_transfer_id"
        case lockID = "lock_id"
        case pendingConfirmationUnits = "pending_confirmation_units"
        case projectedUnits = "projected_units"
        case requestedBy = "requested_by"
        case requestedUnits = "requested_units"
        case state
        case transferID = "transfer_id"
        case transferMode = "transfer_mode"
        case decisionReason = "decision_reason"
        case normalizedInn = "normalized_inn"
        case requesterName = "requester_name"
        case requesterPhone = "requester_phone"
        case roleSought = "role_sought"
        case targetTenantID = "target_tenant_id"
        case targetTenantType = "target_tenant_type"
        case attemptCount = "attempt_count"
        case depth, escalated
        case fromManifestID = "from_manifest_id"
        case fromVehicleID = "from_vehicle_id"
        case manifestDomain = "manifest_domain"
        case stopCount = "stop_count"
        case toManifestID = "to_manifest_id"
        case toVehicleID = "to_vehicle_id"
        case totalVolumeVu = "total_volume_vu"
        case transferCount = "transfer_count"
        case bundleID = "bundle_id"
        case domain
        case importedCount = "imported_count"
        case processedRows = "processed_rows"
        case projectID = "project_id"
        case quarantineCount = "quarantine_count"
        case reportID = "report_id"
        case totalRows = "total_rows"
        case driverIDS = "driver_ids"
        case manifestIDS = "manifest_ids"
        case sharedRouteID = "shared_route_id"
        case splitGroupID = "split_group_id"
        case truckCount = "truck_count"
        case conditionType = "condition_type"
        case gcsPaths = "gcs_paths"
        case notes, quantity
        case reporterID = "reporter_id"
        case reporterRole = "reporter_role"
        case sku
        case deliveredAt = "delivered_at"
        case disputeNote = "dispute_note"
        case proofPhotoURL = "proof_photo_url"
        case proofSignature = "proof_signature"
        case reasonCode = "reason_code"
        case cardMinor = "card_minor"
        case cashMinor = "cash_minor"
        case executionAction = "execution_action"
        case executionMode = "execution_mode"
        case gateway
        case policySource = "policy_source"
        case providerReference = "provider_reference"
        case transactionID = "transaction_id"
        case batchID = "batch_id"
        case netPayoutMinor = "net_payout_minor"
        case railReference = "rail_reference"
        case preOrderID = "pre_order_id"
        case barcode
        case handlingClass = "handling_class"
        case isActive = "is_active"
        case isHazardous = "is_hazardous"
        case isPerishable = "is_perishable"
        case previousPriceMinor = "previous_price_minor"
        case priceMinor = "price_minor"
        case stockQuantity = "stock_quantity"
        case unit
        case promotionID = "promotion_id"
        case retailerIDS = "retailer_ids"
        case retailerScope = "retailer_scope"
        case creditLimitMinor = "credit_limit_minor"
        case currentBalance = "current_balance"
        case requestedAmount = "requested_amount"
        case delinquent
        case profileID = "profile_id"
        case riskTier = "risk_tier"
        case setBy = "set_by"
        case setByRole = "set_by_role"
        case assignedFactoryID = "assigned_factory_id"
        case categories, country
        case isConfigured = "is_configured"
        case isRegistered = "is_registered"
        case breachReason = "breach_reason"
        case breachedAt = "breached_at"
        case cutoffAppliedAt = "cutoff_applied_at"
        case fillRateTargetBps = "fill_rate_target_bps"
        case guaranteedDeliveryDate = "guaranteed_delivery_date"
        case minOrderMinor = "min_order_minor"
        case promiseType = "promise_type"
        case slaHours = "sla_hours"
        case fromWarehouse = "from_warehouse"
        case toWarehouse = "to_warehouse"
        case unavailableNote = "unavailable_note"
        case unavailableReason = "unavailable_reason"
        case assignedTo = "assigned_to"
        case caseID = "case_id"
        case category
        case detectedAt = "detected_at"
        case exposureMinor = "exposure_minor"
        case objectID = "object_id"
        case objectType = "object_type"
        case occurredAt = "occurred_at"
        case playbookAction = "playbook_action"
        case resolvedAt = "resolved_at"
        case resolvedBy = "resolved_by"
        case ruleCode = "rule_code"
        case scoreBps = "score_bps"
        case tenantID = "tenant_id"
        case approverID = "approver_id"
        case breakGlass = "break_glass"
        case comment
        case decisionID = "decision_id"
        case delegatedFrom = "delegated_from"
        case mfaVerified = "mfa_verified"
        case stepIndex = "step_index"
        case actorType = "actor_type"
        case expiresAt = "expires_at"
        case payloadHash = "payload_hash"
        case policyID = "policy_id"
        case policyVersion = "policy_version"
        case requiredApprovers = "required_approvers"
        case anchoredAt = "anchored_at"
        case anchoredBy = "anchored_by"
        case dayBucket = "day_bucket"
        case eventCount = "event_count"
        case headEventID = "head_event_id"
        case rootHash = "root_hash"
        case signerKeyID = "signer_key_id"
        case wormURI = "worm_uri"
        case affectedLotIDS = "affected_lot_ids"
        case complianceRateBps = "compliance_rate_bps"
        case excursionCount = "excursion_count"
        case maxMilliC = "max_milli_c"
        case minMilliC = "min_milli_c"
        case openBreaches = "open_breaches"
        case readingID = "reading_id"
        case sensorID = "sensor_id"
        case tempMilliC = "temp_milli_c"
        case totalReadings = "total_readings"
        case accountLabel = "account_label"
        case approvalID = "approval_id"
        case completedAt = "completed_at"
        case direction
        case externalRef = "external_ref"
        case recordsProcessed = "records_processed"
        case runID = "run_id"
        case streamName = "stream_name"
        case triggeredBy = "triggered_by"
        case balanceDeltaCount = "balance_delta_count"
        case balanceDeltaMinor = "balance_delta_minor"
        case mismatchCount = "mismatch_count"
        case missingExternalCount = "missing_external_count"
        case missingInternalCount = "missing_internal_count"
        case stockDeltaCount = "stock_delta_count"
        case summaryJSON = "summary_json"
        case startedAt = "started_at"
        case prepayOnlyLocked = "prepay_only_locked"
        case score
        case newLimitMinor = "new_limit_minor"
        case previousLimitMinor = "previous_limit_minor"
        case reasonCodesJSON = "reason_codes_json"
        case computedAt = "computed_at"
        case consentVerified = "consent_verified"
        case featuresHash = "features_hash"
        case modelVersion = "model_version"
        case pdBps = "pd_bps"
        case scope
        case scoreID = "score_id"
        case scorerSource = "scorer_source"
        case suggestedLimitMinor = "suggested_limit_minor"
        case createdBy = "created_by"
        case creditJournalID = "credit_journal_id"
        case debitJournalID = "debit_journal_id"
        case fromEntityID = "from_entity_id"
        case periodID = "period_id"
        case referenceID = "reference_id"
        case referenceType = "reference_type"
        case toEntityID = "to_entity_id"
        case transferType = "transfer_type"
        case counterpartyTin = "counterparty_tin"
        case entityID = "entity_id"
        case esfID = "esf_id"
        case pegasusWSEventEnvelopeOperator = "operator"
        case operatorRef = "operator_ref"
        case poID = "po_id"
        case vatMinor = "vat_minor"
        case vatRateBps = "vat_rate_bps"
        case createdAt = "created_at"
        case format
        case fromDate = "from_date"
        case packID = "pack_id"
        case packType = "pack_type"
        case rowCount = "row_count"
        case sha256Hex = "sha256_hex"
        case toDate = "to_date"
        case challengerModel = "challenger_model"
        case challengerWapeBps = "challenger_wape_bps"
        case championModel = "champion_model"
        case championWapeBps = "champion_wape_bps"
        case horizonDays = "horizon_days"
        case locationCount = "location_count"
        case locationID = "location_id"
        case skuCount = "sku_count"
        case wapeImprovementBps = "wape_improvement_bps"
        case winnerModel = "winner_model"
        case baseCurrency = "base_currency"
        case journalEntryID = "journal_entry_id"
        case ratesJSON = "rates_json"
        case roundingDeltaMinor = "rounding_delta_minor"
        case unrealizedGainLossMinor = "unrealized_gain_loss_minor"
        case entryID = "entry_id"
        case hash
        case reversesEntryID = "reverses_entry_id"
        case sourceID = "source_id"
        case sourceType = "source_type"
        case totalCreditMinor = "total_credit_minor"
        case totalDebitMinor = "total_debit_minor"
        case grantedBy = "granted_by"
        case roleName = "role_name"
        case assignedBy = "assigned_by"
        case active
        case displayName = "display_name"
        case externalID = "external_id"
        case resourceID = "resource_id"
        case resourceType = "resource_type"
        case rolesSynced = "roles_synced"
        case userName = "user_name"
        case daysRemaining = "days_remaining"
        case daysToExpiry = "days_to_expiry"
        case expiryDate = "expiry_date"
        case layerID = "layer_id"
        case qtyRemaining = "qty_remaining"
        case unitCostMinor = "unit_cost_minor"
        case valueAtRiskMinor = "value_at_risk_minor"
        case glEntryID = "gl_entry_id"
        case movementID = "movement_id"
        case totalCostMinor = "total_cost_minor"
        case releasedLots = "released_lots"
        case actionMode = "action_mode"
        case approvalRequestID = "approval_request_id"
        case confidenceBps = "confidence_bps"
        case qty
        case sourceWarehouseID = "source_warehouse_id"
        case targetWarehouseID = "target_warehouse_id"
        case cellID = "cell_id"
        case changeKind = "change_kind"
        case securityMarkings = "security_markings"
        case sourceEventID = "source_event_id"
        case sourceEventType = "source_event_type"
        case updatedAt = "updated_at"
        case esfDocID = "esf_doc_id"
        case lineCount = "line_count"
        case matchStatus = "match_status"
        case receiptID = "receipt_id"
        case subtotalMinor = "subtotal_minor"
        case totalReceivedMinor = "total_received_minor"
        case varianceMinor = "variance_minor"
        case varianceReason = "variance_reason"
        case vendorID = "vendor_id"
        case vendorInvoiceID = "vendor_invoice_id"
        case snapshotHash = "snapshot_hash"
        case snapshotURI = "snapshot_uri"
        case timezone
        case agentID = "agent_id"
        case expectedImpactMinor = "expected_impact_minor"
        case proposalType = "proposal_type"
        case roiEventID = "roi_event_id"
        case targetObjectID = "target_object_id"
        case targetObjectType = "target_object_type"
        case autonomyLevel = "autonomy_level"
        case estimatedImpactMinor = "estimated_impact_minor"
        case rollbackRateBps = "rollback_rate_bps"
        case soakDays = "soak_days"
        case soakExecutedCount = "soak_executed_count"
        case actualMinor = "actual_minor"
        case breakID = "break_id"
        case controlTowerExceptionID = "control_tower_exception_id"
        case deltaMinor = "delta_minor"
        case dimension
        case ownerID = "owner_id"
        case slaDueAt = "sla_due_at"
        case breakCount = "break_count"
        case mode
        case eventID = "event_id"
        case evidenceRefID = "evidence_ref_id"
        case evidenceRefType = "evidence_ref_type"
        case holdoutControl = "holdout_control"
        case isEstimate = "is_estimate"
        case methodology
        case savingsMinor = "savings_minor"
        case makerID = "maker_id"
        case routePath = "route_path"
        case violationID = "violation_id"
        case creditExposureDeltaMinor = "credit_exposure_delta_minor"
        case dispatchCostDeltaMinor = "dispatch_cost_delta_minor"
        case forecastDeltaMinor = "forecast_delta_minor"
        case netPnlDeltaMinor = "net_pnl_delta_minor"
        case replenishmentCostMinor = "replenishment_cost_minor"
    }
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
        aggregateID: String?? = nil,
        aggregateType: String?? = nil,
        baseEvent: JSONAny? = nil,
        decidedBy: String?? = nil,
        decision: String?? = nil,
        note: String?? = nil,
        recommendationID: String?? = nil,
        status: String?? = nil,
        supplierID: String?? = nil,
        action: String?? = nil,
        actorID: String?? = nil,
        actorRole: String?? = nil,
        attemptID: String?? = nil,
        confirmationStatus: String?? = nil,
        currency: String?? = nil,
        customerProgressPercent: Int?? = nil,
        customerStage: String?? = nil,
        driverID: String?? = nil,
        escalatedTo: String?? = nil,
        fromDriverID: String?? = nil,
        fromRouteID: String?? = nil,
        gpsLat: Double?? = nil,
        gpsLng: Double?? = nil,
        h3Cell: String?? = nil,
        lat: Double?? = nil,
        licensePlate: String?? = nil,
        lineItems: JSONAny?? = nil,
        lng: Double?? = nil,
        manifestID: String?? = nil,
        message: String?? = nil,
        negotiationID: String?? = nil,
        orderID: String?? = nil,
        orderSource: String?? = nil,
        paymentMethod: String?? = nil,
        previousStatus: String?? = nil,
        proposalID: String?? = nil,
        proposedPriceMinor: Int?? = nil,
        reason: String?? = nil,
        receivingWindowClose: String?? = nil,
        receivingWindowOpen: String?? = nil,
        requestedDeliveryDate: String?? = nil,
        resolution: String?? = nil,
        response: String?? = nil,
        retailerID: String?? = nil,
        routeID: String?? = nil,
        toDriverID: String?? = nil,
        toRouteID: String?? = nil,
        totalMinor: Int?? = nil,
        vehicleID: String?? = nil,
        version: Version?? = nil,
        warehouseID: String?? = nil,
        baselineQty: Int?? = nil,
        baselineSource: String?? = nil,
        blockedReason: String?? = nil,
        confidence: Double?? = nil,
        confidencePct: Int?? = nil,
        factoryID: String?? = nil,
        highUnits: Int?? = nil,
        insightID: String?? = nil,
        lowUnits: Int?? = nil,
        networkNodes: Int?? = nil,
        overrideID: String?? = nil,
        polygonGeojson: String?? = nil,
        productID: String?? = nil,
        publishedBy: String?? = nil,
        scenarioID: String?? = nil,
        signalID: String?? = nil,
        simulationID: String?? = nil,
        transferRecommendations: Int?? = nil,
        ttlSeconds: Int?? = nil,
        agingBucket: String?? = nil,
        amountMinor: Int?? = nil,
        balanceMinor: Int?? = nil,
        dueAt: String?? = nil,
        dunningStep: Int?? = nil,
        invoiceID: String?? = nil,
        lastDunnedAt: String?? = nil,
        principalMinor: Int?? = nil,
        deadline: String?? = nil,
        ehfID: String?? = nil,
        accountHolder: String?? = nil,
        assignedWarehouseID: String?? = nil,
        bankName: String?? = nil,
        contactName: String?? = nil,
        email: String?? = nil,
        legalName: String?? = nil,
        selectedGateways: [String]?? = nil,
        supplierRole: String?? = nil,
        userID: String?? = nil,
        expectedMinor: Int?? = nil,
        overageMinor: Int?? = nil,
        receivedMinor: Int?? = nil,
        shortfallMinor: Int?? = nil,
        traceID: String?? = nil,
        categoryID: String?? = nil,
        iconKey: String?? = nil,
        name: String?? = nil,
        parentCategoryID: String?? = nil,
        sortOrder: Int?? = nil,
        chargebackID: String?? = nil,
        claimID: String?? = nil,
        claimType: String?? = nil,
        photoUrls: [String]?? = nil,
        resolutionNote: String?? = nil,
        settlementMode: String?? = nil,
        source: String?? = nil,
        commandID: String?? = nil,
        actor: String?? = nil,
        campaignID: String?? = nil,
        impactedLotCount: Int?? = nil,
        impactedOrderCount: Int?? = nil,
        impactedUnitsCount: Int?? = nil,
        lotCode: String?? = nil,
        lotID: String?? = nil,
        recallReason: String?? = nil,
        severity: String?? = nil,
        photoProofURL: String?? = nil,
        signatureURL: String?? = nil,
        sessionID: String?? = nil,
        countryCode: String?? = nil,
        phone: String?? = nil,
        orderCount: Int?? = nil,
        orderIDS: [String]?? = nil,
        available: Bool?? = nil,
        homeNodeID: String?? = nil,
        homeNodeType: String?? = nil,
        onShift: Bool?? = nil,
        returnID: String?? = nil,
        itemCount: Int?? = nil,
        requestID: String?? = nil,
        errorCode: String?? = nil,
        errorMessage: String?? = nil,
        fiscalQr: String?? = nil,
        fiscalReceiptID: String?? = nil,
        provider: String?? = nil,
        lockName: String?? = nil,
        systemID: String?? = nil,
        gcsPath: String?? = nil,
        suggestedMappings: Int?? = nil,
        committedUnits: Int?? = nil,
        coverageDays: Int?? = nil,
        coverageStartDate: String?? = nil,
        linkedTransferID: String?? = nil,
        lockID: String?? = nil,
        pendingConfirmationUnits: Int?? = nil,
        projectedUnits: Int?? = nil,
        requestedBy: String?? = nil,
        requestedUnits: Int?? = nil,
        state: String?? = nil,
        transferID: String?? = nil,
        transferMode: String?? = nil,
        decisionReason: String?? = nil,
        normalizedInn: String?? = nil,
        requesterName: String?? = nil,
        requesterPhone: String?? = nil,
        roleSought: String?? = nil,
        targetTenantID: String?? = nil,
        targetTenantType: String?? = nil,
        attemptCount: Int?? = nil,
        depth: Int?? = nil,
        escalated: Bool?? = nil,
        fromManifestID: String?? = nil,
        fromVehicleID: String?? = nil,
        manifestDomain: String?? = nil,
        stopCount: Int?? = nil,
        toManifestID: String?? = nil,
        toVehicleID: String?? = nil,
        totalVolumeVu: TotalVolumeVu?? = nil,
        transferCount: Int?? = nil,
        bundleID: String?? = nil,
        domain: String?? = nil,
        importedCount: Int?? = nil,
        processedRows: Int?? = nil,
        projectID: String?? = nil,
        quarantineCount: Int?? = nil,
        reportID: String?? = nil,
        totalRows: Int?? = nil,
        driverIDS: [String]?? = nil,
        manifestIDS: [String]?? = nil,
        sharedRouteID: String?? = nil,
        splitGroupID: String?? = nil,
        truckCount: Int?? = nil,
        conditionType: String?? = nil,
        gcsPaths: [String]?? = nil,
        notes: String?? = nil,
        quantity: Int?? = nil,
        reporterID: String?? = nil,
        reporterRole: String?? = nil,
        sku: String?? = nil,
        deliveredAt: String?? = nil,
        disputeNote: String?? = nil,
        proofPhotoURL: String?? = nil,
        proofSignature: String?? = nil,
        reasonCode: String?? = nil,
        cardMinor: Int?? = nil,
        cashMinor: Int?? = nil,
        executionAction: String?? = nil,
        executionMode: String?? = nil,
        gateway: String?? = nil,
        policySource: String?? = nil,
        providerReference: String?? = nil,
        transactionID: String?? = nil,
        batchID: String?? = nil,
        netPayoutMinor: Int?? = nil,
        railReference: String?? = nil,
        preOrderID: String?? = nil,
        barcode: String?? = nil,
        handlingClass: String?? = nil,
        isActive: Bool?? = nil,
        isHazardous: Bool?? = nil,
        isPerishable: Bool?? = nil,
        previousPriceMinor: Int?? = nil,
        priceMinor: Int?? = nil,
        stockQuantity: Int?? = nil,
        unit: String?? = nil,
        promotionID: String?? = nil,
        retailerIDS: [String]?? = nil,
        retailerScope: String?? = nil,
        creditLimitMinor: Int?? = nil,
        currentBalance: Int?? = nil,
        requestedAmount: Int?? = nil,
        delinquent: Bool?? = nil,
        profileID: String?? = nil,
        riskTier: String?? = nil,
        setBy: String?? = nil,
        setByRole: String?? = nil,
        assignedFactoryID: String?? = nil,
        categories: [String]?? = nil,
        country: String?? = nil,
        isConfigured: Bool?? = nil,
        isRegistered: Bool?? = nil,
        breachReason: String?? = nil,
        breachedAt: String?? = nil,
        cutoffAppliedAt: String?? = nil,
        fillRateTargetBps: Int?? = nil,
        guaranteedDeliveryDate: String?? = nil,
        minOrderMinor: Int?? = nil,
        promiseType: String?? = nil,
        slaHours: Int?? = nil,
        fromWarehouse: String?? = nil,
        toWarehouse: String?? = nil,
        unavailableNote: String?? = nil,
        unavailableReason: String?? = nil,
        assignedTo: String?? = nil,
        caseID: String?? = nil,
        category: String?? = nil,
        detectedAt: String?? = nil,
        exposureMinor: Int?? = nil,
        objectID: String?? = nil,
        objectType: String?? = nil,
        occurredAt: String?? = nil,
        playbookAction: String?? = nil,
        resolvedAt: String?? = nil,
        resolvedBy: String?? = nil,
        ruleCode: String?? = nil,
        scoreBps: Int?? = nil,
        tenantID: String?? = nil,
        approverID: String?? = nil,
        breakGlass: Bool?? = nil,
        comment: String?? = nil,
        decisionID: String?? = nil,
        delegatedFrom: String?? = nil,
        mfaVerified: Bool?? = nil,
        stepIndex: Int?? = nil,
        actorType: String?? = nil,
        expiresAt: String?? = nil,
        payloadHash: String?? = nil,
        policyID: String?? = nil,
        policyVersion: Int?? = nil,
        requiredApprovers: Int?? = nil,
        anchoredAt: String?? = nil,
        anchoredBy: String?? = nil,
        dayBucket: String?? = nil,
        eventCount: Int?? = nil,
        headEventID: String?? = nil,
        rootHash: String?? = nil,
        signerKeyID: String?? = nil,
        wormURI: String?? = nil,
        affectedLotIDS: [String]?? = nil,
        complianceRateBps: Int?? = nil,
        excursionCount: Int?? = nil,
        maxMilliC: Int?? = nil,
        minMilliC: Int?? = nil,
        openBreaches: Int?? = nil,
        readingID: String?? = nil,
        sensorID: String?? = nil,
        tempMilliC: Int?? = nil,
        totalReadings: Int?? = nil,
        accountLabel: String?? = nil,
        approvalID: String?? = nil,
        completedAt: String?? = nil,
        direction: String?? = nil,
        externalRef: String?? = nil,
        recordsProcessed: Int?? = nil,
        runID: String?? = nil,
        streamName: String?? = nil,
        triggeredBy: String?? = nil,
        balanceDeltaCount: Int?? = nil,
        balanceDeltaMinor: Int?? = nil,
        mismatchCount: Int?? = nil,
        missingExternalCount: Int?? = nil,
        missingInternalCount: Int?? = nil,
        stockDeltaCount: Int?? = nil,
        summaryJSON: String?? = nil,
        startedAt: String?? = nil,
        prepayOnlyLocked: Bool?? = nil,
        score: Int?? = nil,
        newLimitMinor: Int?? = nil,
        previousLimitMinor: Int?? = nil,
        reasonCodesJSON: String?? = nil,
        computedAt: String?? = nil,
        consentVerified: Bool?? = nil,
        featuresHash: String?? = nil,
        modelVersion: String?? = nil,
        pdBps: Int?? = nil,
        scope: String?? = nil,
        scoreID: String?? = nil,
        scorerSource: String?? = nil,
        suggestedLimitMinor: Int?? = nil,
        createdBy: String?? = nil,
        creditJournalID: String?? = nil,
        debitJournalID: String?? = nil,
        fromEntityID: String?? = nil,
        periodID: String?? = nil,
        referenceID: String?? = nil,
        referenceType: String?? = nil,
        toEntityID: String?? = nil,
        transferType: String?? = nil,
        counterpartyTin: String?? = nil,
        entityID: String?? = nil,
        esfID: String?? = nil,
        pegasusWSEventEnvelopeOperator: String?? = nil,
        operatorRef: String?? = nil,
        poID: String?? = nil,
        vatMinor: Int?? = nil,
        vatRateBps: Int?? = nil,
        createdAt: String?? = nil,
        format: String?? = nil,
        fromDate: String?? = nil,
        packID: String?? = nil,
        packType: String?? = nil,
        rowCount: Int?? = nil,
        sha256Hex: String?? = nil,
        toDate: String?? = nil,
        challengerModel: String?? = nil,
        challengerWapeBps: Int?? = nil,
        championModel: String?? = nil,
        championWapeBps: Int?? = nil,
        horizonDays: Int?? = nil,
        locationCount: Int?? = nil,
        locationID: String?? = nil,
        skuCount: Int?? = nil,
        wapeImprovementBps: Int?? = nil,
        winnerModel: String?? = nil,
        baseCurrency: String?? = nil,
        journalEntryID: String?? = nil,
        ratesJSON: String?? = nil,
        roundingDeltaMinor: Int?? = nil,
        unrealizedGainLossMinor: Int?? = nil,
        entryID: String?? = nil,
        hash: String?? = nil,
        reversesEntryID: String?? = nil,
        sourceID: String?? = nil,
        sourceType: String?? = nil,
        totalCreditMinor: Int?? = nil,
        totalDebitMinor: Int?? = nil,
        grantedBy: String?? = nil,
        roleName: String?? = nil,
        assignedBy: String?? = nil,
        active: Bool?? = nil,
        displayName: String?? = nil,
        externalID: String?? = nil,
        resourceID: String?? = nil,
        resourceType: String?? = nil,
        rolesSynced: [String]?? = nil,
        userName: String?? = nil,
        daysRemaining: Int?? = nil,
        daysToExpiry: Int?? = nil,
        expiryDate: String?? = nil,
        layerID: String?? = nil,
        qtyRemaining: Int?? = nil,
        unitCostMinor: Int?? = nil,
        valueAtRiskMinor: Int?? = nil,
        glEntryID: String?? = nil,
        movementID: String?? = nil,
        totalCostMinor: Int?? = nil,
        releasedLots: [JSONAny]?? = nil,
        actionMode: String?? = nil,
        approvalRequestID: String?? = nil,
        confidenceBps: Int?? = nil,
        qty: Int?? = nil,
        sourceWarehouseID: String?? = nil,
        targetWarehouseID: String?? = nil,
        cellID: String?? = nil,
        changeKind: String?? = nil,
        securityMarkings: [String]?? = nil,
        sourceEventID: String?? = nil,
        sourceEventType: String?? = nil,
        updatedAt: String?? = nil,
        esfDocID: String?? = nil,
        lineCount: Int?? = nil,
        matchStatus: String?? = nil,
        receiptID: String?? = nil,
        subtotalMinor: Int?? = nil,
        totalReceivedMinor: Int?? = nil,
        varianceMinor: Int?? = nil,
        varianceReason: String?? = nil,
        vendorID: String?? = nil,
        vendorInvoiceID: String?? = nil,
        snapshotHash: String?? = nil,
        snapshotURI: String?? = nil,
        timezone: String?? = nil,
        agentID: String?? = nil,
        expectedImpactMinor: Int?? = nil,
        proposalType: String?? = nil,
        roiEventID: String?? = nil,
        targetObjectID: String?? = nil,
        targetObjectType: String?? = nil,
        autonomyLevel: String?? = nil,
        estimatedImpactMinor: Int?? = nil,
        rollbackRateBps: Int?? = nil,
        soakDays: Int?? = nil,
        soakExecutedCount: Int?? = nil,
        actualMinor: Int?? = nil,
        breakID: String?? = nil,
        controlTowerExceptionID: String?? = nil,
        deltaMinor: Int?? = nil,
        dimension: String?? = nil,
        ownerID: String?? = nil,
        slaDueAt: String?? = nil,
        breakCount: Int?? = nil,
        mode: String?? = nil,
        eventID: String?? = nil,
        evidenceRefID: String?? = nil,
        evidenceRefType: String?? = nil,
        holdoutControl: Bool?? = nil,
        isEstimate: Bool?? = nil,
        methodology: String?? = nil,
        savingsMinor: Int?? = nil,
        makerID: String?? = nil,
        routePath: String?? = nil,
        violationID: String?? = nil,
        creditExposureDeltaMinor: Int?? = nil,
        dispatchCostDeltaMinor: Int?? = nil,
        forecastDeltaMinor: Int?? = nil,
        netPnlDeltaMinor: Int?? = nil,
        replenishmentCostMinor: Int?? = nil
    ) -> PegasusWSEventEnvelope {
        return PegasusWSEventEnvelope(
            aggregateID: aggregateID ?? self.aggregateID,
            aggregateType: aggregateType ?? self.aggregateType,
            baseEvent: baseEvent ?? self.baseEvent,
            decidedBy: decidedBy ?? self.decidedBy,
            decision: decision ?? self.decision,
            note: note ?? self.note,
            recommendationID: recommendationID ?? self.recommendationID,
            status: status ?? self.status,
            supplierID: supplierID ?? self.supplierID,
            action: action ?? self.action,
            actorID: actorID ?? self.actorID,
            actorRole: actorRole ?? self.actorRole,
            attemptID: attemptID ?? self.attemptID,
            confirmationStatus: confirmationStatus ?? self.confirmationStatus,
            currency: currency ?? self.currency,
            customerProgressPercent: customerProgressPercent ?? self.customerProgressPercent,
            customerStage: customerStage ?? self.customerStage,
            driverID: driverID ?? self.driverID,
            escalatedTo: escalatedTo ?? self.escalatedTo,
            fromDriverID: fromDriverID ?? self.fromDriverID,
            fromRouteID: fromRouteID ?? self.fromRouteID,
            gpsLat: gpsLat ?? self.gpsLat,
            gpsLng: gpsLng ?? self.gpsLng,
            h3Cell: h3Cell ?? self.h3Cell,
            lat: lat ?? self.lat,
            licensePlate: licensePlate ?? self.licensePlate,
            lineItems: lineItems ?? self.lineItems,
            lng: lng ?? self.lng,
            manifestID: manifestID ?? self.manifestID,
            message: message ?? self.message,
            negotiationID: negotiationID ?? self.negotiationID,
            orderID: orderID ?? self.orderID,
            orderSource: orderSource ?? self.orderSource,
            paymentMethod: paymentMethod ?? self.paymentMethod,
            previousStatus: previousStatus ?? self.previousStatus,
            proposalID: proposalID ?? self.proposalID,
            proposedPriceMinor: proposedPriceMinor ?? self.proposedPriceMinor,
            reason: reason ?? self.reason,
            receivingWindowClose: receivingWindowClose ?? self.receivingWindowClose,
            receivingWindowOpen: receivingWindowOpen ?? self.receivingWindowOpen,
            requestedDeliveryDate: requestedDeliveryDate ?? self.requestedDeliveryDate,
            resolution: resolution ?? self.resolution,
            response: response ?? self.response,
            retailerID: retailerID ?? self.retailerID,
            routeID: routeID ?? self.routeID,
            toDriverID: toDriverID ?? self.toDriverID,
            toRouteID: toRouteID ?? self.toRouteID,
            totalMinor: totalMinor ?? self.totalMinor,
            vehicleID: vehicleID ?? self.vehicleID,
            version: version ?? self.version,
            warehouseID: warehouseID ?? self.warehouseID,
            baselineQty: baselineQty ?? self.baselineQty,
            baselineSource: baselineSource ?? self.baselineSource,
            blockedReason: blockedReason ?? self.blockedReason,
            confidence: confidence ?? self.confidence,
            confidencePct: confidencePct ?? self.confidencePct,
            factoryID: factoryID ?? self.factoryID,
            highUnits: highUnits ?? self.highUnits,
            insightID: insightID ?? self.insightID,
            lowUnits: lowUnits ?? self.lowUnits,
            networkNodes: networkNodes ?? self.networkNodes,
            overrideID: overrideID ?? self.overrideID,
            polygonGeojson: polygonGeojson ?? self.polygonGeojson,
            productID: productID ?? self.productID,
            publishedBy: publishedBy ?? self.publishedBy,
            scenarioID: scenarioID ?? self.scenarioID,
            signalID: signalID ?? self.signalID,
            simulationID: simulationID ?? self.simulationID,
            transferRecommendations: transferRecommendations ?? self.transferRecommendations,
            ttlSeconds: ttlSeconds ?? self.ttlSeconds,
            agingBucket: agingBucket ?? self.agingBucket,
            amountMinor: amountMinor ?? self.amountMinor,
            balanceMinor: balanceMinor ?? self.balanceMinor,
            dueAt: dueAt ?? self.dueAt,
            dunningStep: dunningStep ?? self.dunningStep,
            invoiceID: invoiceID ?? self.invoiceID,
            lastDunnedAt: lastDunnedAt ?? self.lastDunnedAt,
            principalMinor: principalMinor ?? self.principalMinor,
            deadline: deadline ?? self.deadline,
            ehfID: ehfID ?? self.ehfID,
            accountHolder: accountHolder ?? self.accountHolder,
            assignedWarehouseID: assignedWarehouseID ?? self.assignedWarehouseID,
            bankName: bankName ?? self.bankName,
            contactName: contactName ?? self.contactName,
            email: email ?? self.email,
            legalName: legalName ?? self.legalName,
            selectedGateways: selectedGateways ?? self.selectedGateways,
            supplierRole: supplierRole ?? self.supplierRole,
            userID: userID ?? self.userID,
            expectedMinor: expectedMinor ?? self.expectedMinor,
            overageMinor: overageMinor ?? self.overageMinor,
            receivedMinor: receivedMinor ?? self.receivedMinor,
            shortfallMinor: shortfallMinor ?? self.shortfallMinor,
            traceID: traceID ?? self.traceID,
            categoryID: categoryID ?? self.categoryID,
            iconKey: iconKey ?? self.iconKey,
            name: name ?? self.name,
            parentCategoryID: parentCategoryID ?? self.parentCategoryID,
            sortOrder: sortOrder ?? self.sortOrder,
            chargebackID: chargebackID ?? self.chargebackID,
            claimID: claimID ?? self.claimID,
            claimType: claimType ?? self.claimType,
            photoUrls: photoUrls ?? self.photoUrls,
            resolutionNote: resolutionNote ?? self.resolutionNote,
            settlementMode: settlementMode ?? self.settlementMode,
            source: source ?? self.source,
            commandID: commandID ?? self.commandID,
            actor: actor ?? self.actor,
            campaignID: campaignID ?? self.campaignID,
            impactedLotCount: impactedLotCount ?? self.impactedLotCount,
            impactedOrderCount: impactedOrderCount ?? self.impactedOrderCount,
            impactedUnitsCount: impactedUnitsCount ?? self.impactedUnitsCount,
            lotCode: lotCode ?? self.lotCode,
            lotID: lotID ?? self.lotID,
            recallReason: recallReason ?? self.recallReason,
            severity: severity ?? self.severity,
            photoProofURL: photoProofURL ?? self.photoProofURL,
            signatureURL: signatureURL ?? self.signatureURL,
            sessionID: sessionID ?? self.sessionID,
            countryCode: countryCode ?? self.countryCode,
            phone: phone ?? self.phone,
            orderCount: orderCount ?? self.orderCount,
            orderIDS: orderIDS ?? self.orderIDS,
            available: available ?? self.available,
            homeNodeID: homeNodeID ?? self.homeNodeID,
            homeNodeType: homeNodeType ?? self.homeNodeType,
            onShift: onShift ?? self.onShift,
            returnID: returnID ?? self.returnID,
            itemCount: itemCount ?? self.itemCount,
            requestID: requestID ?? self.requestID,
            errorCode: errorCode ?? self.errorCode,
            errorMessage: errorMessage ?? self.errorMessage,
            fiscalQr: fiscalQr ?? self.fiscalQr,
            fiscalReceiptID: fiscalReceiptID ?? self.fiscalReceiptID,
            provider: provider ?? self.provider,
            lockName: lockName ?? self.lockName,
            systemID: systemID ?? self.systemID,
            gcsPath: gcsPath ?? self.gcsPath,
            suggestedMappings: suggestedMappings ?? self.suggestedMappings,
            committedUnits: committedUnits ?? self.committedUnits,
            coverageDays: coverageDays ?? self.coverageDays,
            coverageStartDate: coverageStartDate ?? self.coverageStartDate,
            linkedTransferID: linkedTransferID ?? self.linkedTransferID,
            lockID: lockID ?? self.lockID,
            pendingConfirmationUnits: pendingConfirmationUnits ?? self.pendingConfirmationUnits,
            projectedUnits: projectedUnits ?? self.projectedUnits,
            requestedBy: requestedBy ?? self.requestedBy,
            requestedUnits: requestedUnits ?? self.requestedUnits,
            state: state ?? self.state,
            transferID: transferID ?? self.transferID,
            transferMode: transferMode ?? self.transferMode,
            decisionReason: decisionReason ?? self.decisionReason,
            normalizedInn: normalizedInn ?? self.normalizedInn,
            requesterName: requesterName ?? self.requesterName,
            requesterPhone: requesterPhone ?? self.requesterPhone,
            roleSought: roleSought ?? self.roleSought,
            targetTenantID: targetTenantID ?? self.targetTenantID,
            targetTenantType: targetTenantType ?? self.targetTenantType,
            attemptCount: attemptCount ?? self.attemptCount,
            depth: depth ?? self.depth,
            escalated: escalated ?? self.escalated,
            fromManifestID: fromManifestID ?? self.fromManifestID,
            fromVehicleID: fromVehicleID ?? self.fromVehicleID,
            manifestDomain: manifestDomain ?? self.manifestDomain,
            stopCount: stopCount ?? self.stopCount,
            toManifestID: toManifestID ?? self.toManifestID,
            toVehicleID: toVehicleID ?? self.toVehicleID,
            totalVolumeVu: totalVolumeVu ?? self.totalVolumeVu,
            transferCount: transferCount ?? self.transferCount,
            bundleID: bundleID ?? self.bundleID,
            domain: domain ?? self.domain,
            importedCount: importedCount ?? self.importedCount,
            processedRows: processedRows ?? self.processedRows,
            projectID: projectID ?? self.projectID,
            quarantineCount: quarantineCount ?? self.quarantineCount,
            reportID: reportID ?? self.reportID,
            totalRows: totalRows ?? self.totalRows,
            driverIDS: driverIDS ?? self.driverIDS,
            manifestIDS: manifestIDS ?? self.manifestIDS,
            sharedRouteID: sharedRouteID ?? self.sharedRouteID,
            splitGroupID: splitGroupID ?? self.splitGroupID,
            truckCount: truckCount ?? self.truckCount,
            conditionType: conditionType ?? self.conditionType,
            gcsPaths: gcsPaths ?? self.gcsPaths,
            notes: notes ?? self.notes,
            quantity: quantity ?? self.quantity,
            reporterID: reporterID ?? self.reporterID,
            reporterRole: reporterRole ?? self.reporterRole,
            sku: sku ?? self.sku,
            deliveredAt: deliveredAt ?? self.deliveredAt,
            disputeNote: disputeNote ?? self.disputeNote,
            proofPhotoURL: proofPhotoURL ?? self.proofPhotoURL,
            proofSignature: proofSignature ?? self.proofSignature,
            reasonCode: reasonCode ?? self.reasonCode,
            cardMinor: cardMinor ?? self.cardMinor,
            cashMinor: cashMinor ?? self.cashMinor,
            executionAction: executionAction ?? self.executionAction,
            executionMode: executionMode ?? self.executionMode,
            gateway: gateway ?? self.gateway,
            policySource: policySource ?? self.policySource,
            providerReference: providerReference ?? self.providerReference,
            transactionID: transactionID ?? self.transactionID,
            batchID: batchID ?? self.batchID,
            netPayoutMinor: netPayoutMinor ?? self.netPayoutMinor,
            railReference: railReference ?? self.railReference,
            preOrderID: preOrderID ?? self.preOrderID,
            barcode: barcode ?? self.barcode,
            handlingClass: handlingClass ?? self.handlingClass,
            isActive: isActive ?? self.isActive,
            isHazardous: isHazardous ?? self.isHazardous,
            isPerishable: isPerishable ?? self.isPerishable,
            previousPriceMinor: previousPriceMinor ?? self.previousPriceMinor,
            priceMinor: priceMinor ?? self.priceMinor,
            stockQuantity: stockQuantity ?? self.stockQuantity,
            unit: unit ?? self.unit,
            promotionID: promotionID ?? self.promotionID,
            retailerIDS: retailerIDS ?? self.retailerIDS,
            retailerScope: retailerScope ?? self.retailerScope,
            creditLimitMinor: creditLimitMinor ?? self.creditLimitMinor,
            currentBalance: currentBalance ?? self.currentBalance,
            requestedAmount: requestedAmount ?? self.requestedAmount,
            delinquent: delinquent ?? self.delinquent,
            profileID: profileID ?? self.profileID,
            riskTier: riskTier ?? self.riskTier,
            setBy: setBy ?? self.setBy,
            setByRole: setByRole ?? self.setByRole,
            assignedFactoryID: assignedFactoryID ?? self.assignedFactoryID,
            categories: categories ?? self.categories,
            country: country ?? self.country,
            isConfigured: isConfigured ?? self.isConfigured,
            isRegistered: isRegistered ?? self.isRegistered,
            breachReason: breachReason ?? self.breachReason,
            breachedAt: breachedAt ?? self.breachedAt,
            cutoffAppliedAt: cutoffAppliedAt ?? self.cutoffAppliedAt,
            fillRateTargetBps: fillRateTargetBps ?? self.fillRateTargetBps,
            guaranteedDeliveryDate: guaranteedDeliveryDate ?? self.guaranteedDeliveryDate,
            minOrderMinor: minOrderMinor ?? self.minOrderMinor,
            promiseType: promiseType ?? self.promiseType,
            slaHours: slaHours ?? self.slaHours,
            fromWarehouse: fromWarehouse ?? self.fromWarehouse,
            toWarehouse: toWarehouse ?? self.toWarehouse,
            unavailableNote: unavailableNote ?? self.unavailableNote,
            unavailableReason: unavailableReason ?? self.unavailableReason,
            assignedTo: assignedTo ?? self.assignedTo,
            caseID: caseID ?? self.caseID,
            category: category ?? self.category,
            detectedAt: detectedAt ?? self.detectedAt,
            exposureMinor: exposureMinor ?? self.exposureMinor,
            objectID: objectID ?? self.objectID,
            objectType: objectType ?? self.objectType,
            occurredAt: occurredAt ?? self.occurredAt,
            playbookAction: playbookAction ?? self.playbookAction,
            resolvedAt: resolvedAt ?? self.resolvedAt,
            resolvedBy: resolvedBy ?? self.resolvedBy,
            ruleCode: ruleCode ?? self.ruleCode,
            scoreBps: scoreBps ?? self.scoreBps,
            tenantID: tenantID ?? self.tenantID,
            approverID: approverID ?? self.approverID,
            breakGlass: breakGlass ?? self.breakGlass,
            comment: comment ?? self.comment,
            decisionID: decisionID ?? self.decisionID,
            delegatedFrom: delegatedFrom ?? self.delegatedFrom,
            mfaVerified: mfaVerified ?? self.mfaVerified,
            stepIndex: stepIndex ?? self.stepIndex,
            actorType: actorType ?? self.actorType,
            expiresAt: expiresAt ?? self.expiresAt,
            payloadHash: payloadHash ?? self.payloadHash,
            policyID: policyID ?? self.policyID,
            policyVersion: policyVersion ?? self.policyVersion,
            requiredApprovers: requiredApprovers ?? self.requiredApprovers,
            anchoredAt: anchoredAt ?? self.anchoredAt,
            anchoredBy: anchoredBy ?? self.anchoredBy,
            dayBucket: dayBucket ?? self.dayBucket,
            eventCount: eventCount ?? self.eventCount,
            headEventID: headEventID ?? self.headEventID,
            rootHash: rootHash ?? self.rootHash,
            signerKeyID: signerKeyID ?? self.signerKeyID,
            wormURI: wormURI ?? self.wormURI,
            affectedLotIDS: affectedLotIDS ?? self.affectedLotIDS,
            complianceRateBps: complianceRateBps ?? self.complianceRateBps,
            excursionCount: excursionCount ?? self.excursionCount,
            maxMilliC: maxMilliC ?? self.maxMilliC,
            minMilliC: minMilliC ?? self.minMilliC,
            openBreaches: openBreaches ?? self.openBreaches,
            readingID: readingID ?? self.readingID,
            sensorID: sensorID ?? self.sensorID,
            tempMilliC: tempMilliC ?? self.tempMilliC,
            totalReadings: totalReadings ?? self.totalReadings,
            accountLabel: accountLabel ?? self.accountLabel,
            approvalID: approvalID ?? self.approvalID,
            completedAt: completedAt ?? self.completedAt,
            direction: direction ?? self.direction,
            externalRef: externalRef ?? self.externalRef,
            recordsProcessed: recordsProcessed ?? self.recordsProcessed,
            runID: runID ?? self.runID,
            streamName: streamName ?? self.streamName,
            triggeredBy: triggeredBy ?? self.triggeredBy,
            balanceDeltaCount: balanceDeltaCount ?? self.balanceDeltaCount,
            balanceDeltaMinor: balanceDeltaMinor ?? self.balanceDeltaMinor,
            mismatchCount: mismatchCount ?? self.mismatchCount,
            missingExternalCount: missingExternalCount ?? self.missingExternalCount,
            missingInternalCount: missingInternalCount ?? self.missingInternalCount,
            stockDeltaCount: stockDeltaCount ?? self.stockDeltaCount,
            summaryJSON: summaryJSON ?? self.summaryJSON,
            startedAt: startedAt ?? self.startedAt,
            prepayOnlyLocked: prepayOnlyLocked ?? self.prepayOnlyLocked,
            score: score ?? self.score,
            newLimitMinor: newLimitMinor ?? self.newLimitMinor,
            previousLimitMinor: previousLimitMinor ?? self.previousLimitMinor,
            reasonCodesJSON: reasonCodesJSON ?? self.reasonCodesJSON,
            computedAt: computedAt ?? self.computedAt,
            consentVerified: consentVerified ?? self.consentVerified,
            featuresHash: featuresHash ?? self.featuresHash,
            modelVersion: modelVersion ?? self.modelVersion,
            pdBps: pdBps ?? self.pdBps,
            scope: scope ?? self.scope,
            scoreID: scoreID ?? self.scoreID,
            scorerSource: scorerSource ?? self.scorerSource,
            suggestedLimitMinor: suggestedLimitMinor ?? self.suggestedLimitMinor,
            createdBy: createdBy ?? self.createdBy,
            creditJournalID: creditJournalID ?? self.creditJournalID,
            debitJournalID: debitJournalID ?? self.debitJournalID,
            fromEntityID: fromEntityID ?? self.fromEntityID,
            periodID: periodID ?? self.periodID,
            referenceID: referenceID ?? self.referenceID,
            referenceType: referenceType ?? self.referenceType,
            toEntityID: toEntityID ?? self.toEntityID,
            transferType: transferType ?? self.transferType,
            counterpartyTin: counterpartyTin ?? self.counterpartyTin,
            entityID: entityID ?? self.entityID,
            esfID: esfID ?? self.esfID,
            pegasusWSEventEnvelopeOperator: pegasusWSEventEnvelopeOperator ?? self.pegasusWSEventEnvelopeOperator,
            operatorRef: operatorRef ?? self.operatorRef,
            poID: poID ?? self.poID,
            vatMinor: vatMinor ?? self.vatMinor,
            vatRateBps: vatRateBps ?? self.vatRateBps,
            createdAt: createdAt ?? self.createdAt,
            format: format ?? self.format,
            fromDate: fromDate ?? self.fromDate,
            packID: packID ?? self.packID,
            packType: packType ?? self.packType,
            rowCount: rowCount ?? self.rowCount,
            sha256Hex: sha256Hex ?? self.sha256Hex,
            toDate: toDate ?? self.toDate,
            challengerModel: challengerModel ?? self.challengerModel,
            challengerWapeBps: challengerWapeBps ?? self.challengerWapeBps,
            championModel: championModel ?? self.championModel,
            championWapeBps: championWapeBps ?? self.championWapeBps,
            horizonDays: horizonDays ?? self.horizonDays,
            locationCount: locationCount ?? self.locationCount,
            locationID: locationID ?? self.locationID,
            skuCount: skuCount ?? self.skuCount,
            wapeImprovementBps: wapeImprovementBps ?? self.wapeImprovementBps,
            winnerModel: winnerModel ?? self.winnerModel,
            baseCurrency: baseCurrency ?? self.baseCurrency,
            journalEntryID: journalEntryID ?? self.journalEntryID,
            ratesJSON: ratesJSON ?? self.ratesJSON,
            roundingDeltaMinor: roundingDeltaMinor ?? self.roundingDeltaMinor,
            unrealizedGainLossMinor: unrealizedGainLossMinor ?? self.unrealizedGainLossMinor,
            entryID: entryID ?? self.entryID,
            hash: hash ?? self.hash,
            reversesEntryID: reversesEntryID ?? self.reversesEntryID,
            sourceID: sourceID ?? self.sourceID,
            sourceType: sourceType ?? self.sourceType,
            totalCreditMinor: totalCreditMinor ?? self.totalCreditMinor,
            totalDebitMinor: totalDebitMinor ?? self.totalDebitMinor,
            grantedBy: grantedBy ?? self.grantedBy,
            roleName: roleName ?? self.roleName,
            assignedBy: assignedBy ?? self.assignedBy,
            active: active ?? self.active,
            displayName: displayName ?? self.displayName,
            externalID: externalID ?? self.externalID,
            resourceID: resourceID ?? self.resourceID,
            resourceType: resourceType ?? self.resourceType,
            rolesSynced: rolesSynced ?? self.rolesSynced,
            userName: userName ?? self.userName,
            daysRemaining: daysRemaining ?? self.daysRemaining,
            daysToExpiry: daysToExpiry ?? self.daysToExpiry,
            expiryDate: expiryDate ?? self.expiryDate,
            layerID: layerID ?? self.layerID,
            qtyRemaining: qtyRemaining ?? self.qtyRemaining,
            unitCostMinor: unitCostMinor ?? self.unitCostMinor,
            valueAtRiskMinor: valueAtRiskMinor ?? self.valueAtRiskMinor,
            glEntryID: glEntryID ?? self.glEntryID,
            movementID: movementID ?? self.movementID,
            totalCostMinor: totalCostMinor ?? self.totalCostMinor,
            releasedLots: releasedLots ?? self.releasedLots,
            actionMode: actionMode ?? self.actionMode,
            approvalRequestID: approvalRequestID ?? self.approvalRequestID,
            confidenceBps: confidenceBps ?? self.confidenceBps,
            qty: qty ?? self.qty,
            sourceWarehouseID: sourceWarehouseID ?? self.sourceWarehouseID,
            targetWarehouseID: targetWarehouseID ?? self.targetWarehouseID,
            cellID: cellID ?? self.cellID,
            changeKind: changeKind ?? self.changeKind,
            securityMarkings: securityMarkings ?? self.securityMarkings,
            sourceEventID: sourceEventID ?? self.sourceEventID,
            sourceEventType: sourceEventType ?? self.sourceEventType,
            updatedAt: updatedAt ?? self.updatedAt,
            esfDocID: esfDocID ?? self.esfDocID,
            lineCount: lineCount ?? self.lineCount,
            matchStatus: matchStatus ?? self.matchStatus,
            receiptID: receiptID ?? self.receiptID,
            subtotalMinor: subtotalMinor ?? self.subtotalMinor,
            totalReceivedMinor: totalReceivedMinor ?? self.totalReceivedMinor,
            varianceMinor: varianceMinor ?? self.varianceMinor,
            varianceReason: varianceReason ?? self.varianceReason,
            vendorID: vendorID ?? self.vendorID,
            vendorInvoiceID: vendorInvoiceID ?? self.vendorInvoiceID,
            snapshotHash: snapshotHash ?? self.snapshotHash,
            snapshotURI: snapshotURI ?? self.snapshotURI,
            timezone: timezone ?? self.timezone,
            agentID: agentID ?? self.agentID,
            expectedImpactMinor: expectedImpactMinor ?? self.expectedImpactMinor,
            proposalType: proposalType ?? self.proposalType,
            roiEventID: roiEventID ?? self.roiEventID,
            targetObjectID: targetObjectID ?? self.targetObjectID,
            targetObjectType: targetObjectType ?? self.targetObjectType,
            autonomyLevel: autonomyLevel ?? self.autonomyLevel,
            estimatedImpactMinor: estimatedImpactMinor ?? self.estimatedImpactMinor,
            rollbackRateBps: rollbackRateBps ?? self.rollbackRateBps,
            soakDays: soakDays ?? self.soakDays,
            soakExecutedCount: soakExecutedCount ?? self.soakExecutedCount,
            actualMinor: actualMinor ?? self.actualMinor,
            breakID: breakID ?? self.breakID,
            controlTowerExceptionID: controlTowerExceptionID ?? self.controlTowerExceptionID,
            deltaMinor: deltaMinor ?? self.deltaMinor,
            dimension: dimension ?? self.dimension,
            ownerID: ownerID ?? self.ownerID,
            slaDueAt: slaDueAt ?? self.slaDueAt,
            breakCount: breakCount ?? self.breakCount,
            mode: mode ?? self.mode,
            eventID: eventID ?? self.eventID,
            evidenceRefID: evidenceRefID ?? self.evidenceRefID,
            evidenceRefType: evidenceRefType ?? self.evidenceRefType,
            holdoutControl: holdoutControl ?? self.holdoutControl,
            isEstimate: isEstimate ?? self.isEstimate,
            methodology: methodology ?? self.methodology,
            savingsMinor: savingsMinor ?? self.savingsMinor,
            makerID: makerID ?? self.makerID,
            routePath: routePath ?? self.routePath,
            violationID: violationID ?? self.violationID,
            creditExposureDeltaMinor: creditExposureDeltaMinor ?? self.creditExposureDeltaMinor,
            dispatchCostDeltaMinor: dispatchCostDeltaMinor ?? self.dispatchCostDeltaMinor,
            forecastDeltaMinor: forecastDeltaMinor ?? self.forecastDeltaMinor,
            netPnlDeltaMinor: netPnlDeltaMinor ?? self.netPnlDeltaMinor,
            replenishmentCostMinor: replenishmentCostMinor ?? self.replenishmentCostMinor
        )
    }

    func jsonData() throws -> Data {
        return try newJSONEncoder().encode(self)
    }

    func jsonString(encoding: String.Encoding = .utf8) throws -> String? {
        return String(data: try self.jsonData(), encoding: encoding)
    }
}

enum TotalVolumeVu: Codable {
    case double(Double)
    case integer(Int)

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if let x = try? container.decode(Int.self) {
            self = .integer(x)
            return
        }
        if let x = try? container.decode(Double.self) {
            self = .double(x)
            return
        }
        throw DecodingError.typeMismatch(TotalVolumeVu.self, DecodingError.Context(codingPath: decoder.codingPath, debugDescription: "Wrong type for TotalVolumeVu"))
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch self {
        case .double(let x):
            try container.encode(x)
        case .integer(let x):
            try container.encode(x)
        }
    }
}

enum Version: Codable {
    case integer(Int)
    case string(String)

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        if let x = try? container.decode(Int.self) {
            self = .integer(x)
            return
        }
        if let x = try? container.decode(String.self) {
            self = .string(x)
            return
        }
        throw DecodingError.typeMismatch(Version.self, DecodingError.Context(codingPath: decoder.codingPath, debugDescription: "Wrong type for Version"))
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()
        switch self {
        case .integer(let x):
            try container.encode(x)
        case .string(let x):
            try container.encode(x)
        }
    }
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

// MARK: - Encode/decode helpers

class JSONNull: Codable, Hashable {

    public static func == (lhs: JSONNull, rhs: JSONNull) -> Bool {
            return true
    }

    public var hashValue: Int {
            return 0
    }

    public init() {}

    public required init(from decoder: Decoder) throws {
            let container = try decoder.singleValueContainer()
            if !container.decodeNil() {
                    throw DecodingError.typeMismatch(JSONNull.self, DecodingError.Context(codingPath: decoder.codingPath, debugDescription: "Wrong type for JSONNull"))
            }
    }

    public func encode(to encoder: Encoder) throws {
            var container = encoder.singleValueContainer()
            try container.encodeNil()
    }
}

class JSONCodingKey: CodingKey {
    let key: String

    required init?(intValue: Int) {
            return nil
    }

    required init?(stringValue: String) {
            key = stringValue
    }

    var intValue: Int? {
            return nil
    }

    var stringValue: String {
            return key
    }
}

class JSONAny: Codable {

    let value: Any

    static func decodingError(forCodingPath codingPath: [CodingKey]) -> DecodingError {
            let context = DecodingError.Context(codingPath: codingPath, debugDescription: "Cannot decode JSONAny")
            return DecodingError.typeMismatch(JSONAny.self, context)
    }

    static func encodingError(forValue value: Any, codingPath: [CodingKey]) -> EncodingError {
            let context = EncodingError.Context(codingPath: codingPath, debugDescription: "Cannot encode JSONAny")
            return EncodingError.invalidValue(value, context)
    }

    static func decode(from container: SingleValueDecodingContainer) throws -> Any {
            if let value = try? container.decode(Bool.self) {
                    return value
            }
            if let value = try? container.decode(Int64.self) {
                    return value
            }
            if let value = try? container.decode(Double.self) {
                    return value
            }
            if let value = try? container.decode(String.self) {
                    return value
            }
            if container.decodeNil() {
                    return JSONNull()
            }
            throw decodingError(forCodingPath: container.codingPath)
    }

    static func decode(from container: inout UnkeyedDecodingContainer) throws -> Any {
            if let value = try? container.decode(Bool.self) {
                    return value
            }
            if let value = try? container.decode(Int64.self) {
                    return value
            }
            if let value = try? container.decode(Double.self) {
                    return value
            }
            if let value = try? container.decode(String.self) {
                    return value
            }
            if let value = try? container.decodeNil() {
                    if value {
                            return JSONNull()
                    }
            }
            if var container = try? container.nestedUnkeyedContainer() {
                    return try decodeArray(from: &container)
            }
            if var container = try? container.nestedContainer(keyedBy: JSONCodingKey.self) {
                    return try decodeDictionary(from: &container)
            }
            throw decodingError(forCodingPath: container.codingPath)
    }

    static func decode(from container: inout KeyedDecodingContainer<JSONCodingKey>, forKey key: JSONCodingKey) throws -> Any {
            if let value = try? container.decode(Bool.self, forKey: key) {
                    return value
            }
            if let value = try? container.decode(Int64.self, forKey: key) {
                    return value
            }
            if let value = try? container.decode(Double.self, forKey: key) {
                    return value
            }
            if let value = try? container.decode(String.self, forKey: key) {
                    return value
            }
            if let value = try? container.decodeNil(forKey: key) {
                    if value {
                            return JSONNull()
                    }
            }
            if var container = try? container.nestedUnkeyedContainer(forKey: key) {
                    return try decodeArray(from: &container)
            }
            if var container = try? container.nestedContainer(keyedBy: JSONCodingKey.self, forKey: key) {
                    return try decodeDictionary(from: &container)
            }
            throw decodingError(forCodingPath: container.codingPath)
    }

    static func decodeArray(from container: inout UnkeyedDecodingContainer) throws -> [Any] {
            var arr: [Any] = []
            while !container.isAtEnd {
                    let value = try decode(from: &container)
                    arr.append(value)
            }
            return arr
    }

    static func decodeDictionary(from container: inout KeyedDecodingContainer<JSONCodingKey>) throws -> [String: Any] {
            var dict = [String: Any]()
            for key in container.allKeys {
                    let value = try decode(from: &container, forKey: key)
                    dict[key.stringValue] = value
            }
            return dict
    }

    static func encode(to container: inout UnkeyedEncodingContainer, array: [Any]) throws {
            for value in array {
                    if let value = value as? Bool {
                            try container.encode(value)
                    } else if let value = value as? Int64 {
                            try container.encode(value)
                    } else if let value = value as? Double {
                            try container.encode(value)
                    } else if let value = value as? String {
                            try container.encode(value)
                    } else if value is JSONNull {
                            try container.encodeNil()
                    } else if let value = value as? [Any] {
                            var container = container.nestedUnkeyedContainer()
                            try encode(to: &container, array: value)
                    } else if let value = value as? [String: Any] {
                            var container = container.nestedContainer(keyedBy: JSONCodingKey.self)
                            try encode(to: &container, dictionary: value)
                    } else {
                            throw encodingError(forValue: value, codingPath: container.codingPath)
                    }
            }
    }

    static func encode(to container: inout KeyedEncodingContainer<JSONCodingKey>, dictionary: [String: Any]) throws {
            for (key, value) in dictionary {
                    guard let key = JSONCodingKey(stringValue: key) else { continue }
                    if let value = value as? Bool {
                            try container.encode(value, forKey: key)
                    } else if let value = value as? Int64 {
                            try container.encode(value, forKey: key)
                    } else if let value = value as? Double {
                            try container.encode(value, forKey: key)
                    } else if let value = value as? String {
                            try container.encode(value, forKey: key)
                    } else if value is JSONNull {
                            try container.encodeNil(forKey: key)
                    } else if let value = value as? [Any] {
                            var container = container.nestedUnkeyedContainer(forKey: key)
                            try encode(to: &container, array: value)
                    } else if let value = value as? [String: Any] {
                            var container = container.nestedContainer(keyedBy: JSONCodingKey.self, forKey: key)
                            try encode(to: &container, dictionary: value)
                    } else {
                            throw encodingError(forValue: value, codingPath: container.codingPath)
                    }
            }
    }

    static func encode(to container: inout SingleValueEncodingContainer, value: Any) throws {
            if let value = value as? Bool {
                    try container.encode(value)
            } else if let value = value as? Int64 {
                    try container.encode(value)
            } else if let value = value as? Double {
                    try container.encode(value)
            } else if let value = value as? String {
                    try container.encode(value)
            } else if value is JSONNull {
                    try container.encodeNil()
            } else {
                    throw encodingError(forValue: value, codingPath: container.codingPath)
            }
    }

    public required init(from decoder: Decoder) throws {
            if var arrayContainer = try? decoder.unkeyedContainer() {
                    self.value = try JSONAny.decodeArray(from: &arrayContainer)
            } else if var container = try? decoder.container(keyedBy: JSONCodingKey.self) {
                    self.value = try JSONAny.decodeDictionary(from: &container)
            } else {
                    let container = try decoder.singleValueContainer()
                    self.value = try JSONAny.decode(from: container)
            }
    }

    public func encode(to encoder: Encoder) throws {
            if let arr = self.value as? [Any] {
                    var container = encoder.unkeyedContainer()
                    try JSONAny.encode(to: &container, array: arr)
            } else if let dict = self.value as? [String: Any] {
                    var container = encoder.container(keyedBy: JSONCodingKey.self)
                    try JSONAny.encode(to: &container, dictionary: dict)
            } else {
                    var container = encoder.singleValueContainer()
                    try JSONAny.encode(to: &container, value: self.value)
            }
    }
}
