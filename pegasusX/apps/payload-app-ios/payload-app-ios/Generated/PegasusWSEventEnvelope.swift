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
        case productID = "product_id"
        case recallReason = "recall_reason"
        case severity
        case photoProofURL = "photo_proof_url"
        case signatureURL = "signature_url"
        case sessionID = "session_id"
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
        case publishedBy = "published_by"
        case scenarioID = "scenario_id"
        case signalID = "signal_id"
        case simulationID = "simulation_id"
        case transferRecommendations = "transfer_recommendations"
        case ttlSeconds = "ttl_seconds"
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
        case driverIDS = "driver_ids"
        case manifestIDS = "manifest_ids"
        case sharedRouteID = "shared_route_id"
        case splitGroupID = "split_group_id"
        case truckCount = "truck_count"
        case conditionType = "condition_type"
        case gcsPaths = "gcs_paths"
        case notes, quantity
        case reportID = "report_id"
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
        case requiresColdChain = "requires_cold_chain"
        case stockQuantity = "stock_quantity"
        case storageTempMaxC = "storage_temp_max_c"
        case storageTempMinC = "storage_temp_min_c"
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
        productID: String?? = nil,
        recallReason: String?? = nil,
        severity: String?? = nil,
        photoProofURL: String?? = nil,
        signatureURL: String?? = nil,
        sessionID: String?? = nil,
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
        publishedBy: String?? = nil,
        scenarioID: String?? = nil,
        signalID: String?? = nil,
        simulationID: String?? = nil,
        transferRecommendations: Int?? = nil,
        ttlSeconds: Int?? = nil,
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
        driverIDS: [String]?? = nil,
        manifestIDS: [String]?? = nil,
        sharedRouteID: String?? = nil,
        splitGroupID: String?? = nil,
        truckCount: Int?? = nil,
        conditionType: String?? = nil,
        gcsPaths: [String]?? = nil,
        notes: String?? = nil,
        quantity: Int?? = nil,
        reportID: String?? = nil,
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
        requiresColdChain: Bool?? = nil,
        stockQuantity: Int?? = nil,
        storageTempMaxC: Double?? = nil,
        storageTempMinC: Double?? = nil,
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
        unavailableReason: String?? = nil
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
            productID: productID ?? self.productID,
            recallReason: recallReason ?? self.recallReason,
            severity: severity ?? self.severity,
            photoProofURL: photoProofURL ?? self.photoProofURL,
            signatureURL: signatureURL ?? self.signatureURL,
            sessionID: sessionID ?? self.sessionID,
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
            publishedBy: publishedBy ?? self.publishedBy,
            scenarioID: scenarioID ?? self.scenarioID,
            signalID: signalID ?? self.signalID,
            simulationID: simulationID ?? self.simulationID,
            transferRecommendations: transferRecommendations ?? self.transferRecommendations,
            ttlSeconds: ttlSeconds ?? self.ttlSeconds,
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
            driverIDS: driverIDS ?? self.driverIDS,
            manifestIDS: manifestIDS ?? self.manifestIDS,
            sharedRouteID: sharedRouteID ?? self.sharedRouteID,
            splitGroupID: splitGroupID ?? self.splitGroupID,
            truckCount: truckCount ?? self.truckCount,
            conditionType: conditionType ?? self.conditionType,
            gcsPaths: gcsPaths ?? self.gcsPaths,
            notes: notes ?? self.notes,
            quantity: quantity ?? self.quantity,
            reportID: reportID ?? self.reportID,
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
            requiresColdChain: requiresColdChain ?? self.requiresColdChain,
            stockQuantity: stockQuantity ?? self.stockQuantity,
            storageTempMaxC: storageTempMaxC ?? self.storageTempMaxC,
            storageTempMinC: storageTempMinC ?? self.storageTempMinC,
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
            unavailableReason: unavailableReason ?? self.unavailableReason
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
                    let key = JSONCodingKey(stringValue: key)!
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
