// This file was generated from JSON Schema using quicktype, do not modify it directly.
// To parse the JSON, add this file to your project and do:
//
//   let pegasusWSEventEnvelope = try PegasusWSEventEnvelope(json)

import Foundation

// MARK: - PegasusWSEventEnvelope
struct PegasusWSEventEnvelope: Codable {
    let type: TypeEnum
    let aggregateID, aggregateType: String?
    let baseEvent: JSONAny
    let decidedBy, decision, note, recommendationID: String?
    let status, supplierID, action, actorID: String?
    let actorRole, attemptID, confirmationStatus, currency: String?
    let driverID, escalatedTo, fromDriverID, fromRouteID: String?
    let gpsLat, gpsLng: Double?
    let h3Cell: String?
    let lat: Double?
    let licensePlate: String?
    let lineItems: JSONAny?
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
    let warehouseID, accountHolder, assignedWarehouseID, bankName: String?
    let contactName, email, legalName: String?
    let selectedGateways: [String]?
    let supplierRole, userID: String?
    let expectedMinor, overageMinor, receivedMinor, shortfallMinor: Int?
    let traceID: String?
    let amountMinor: Int?
    let chargebackID, claimID, claimType: String?
    let photoUrls: [String]?
    let resolutionNote, settlementMode, source, commandID: String?
    let sessionID: String?
    let baselineQty: Int?
    let baselineSource, blockedReason: String?
    let confidence: Double?
    let confidencePct: Int?
    let factoryID: String?
    let highUnits: Int?
    let insightID: String?
    let lowUnits, networkNodes: Int?
    let overrideID, polygonGeojson, productID, signalID: String?
    let simulationID: String?
    let transferRecommendations, ttlSeconds: Int?
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
    let fromManifestID, fromVehicleID: String?
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
    let setBy, setByRole, countryCode, name: String?
    let phone, assignedFactoryID: String?
    let categories: [String]?
    let country: String?
    let isConfigured, isRegistered: Bool?
    let fromWarehouse, toWarehouse: String?
    let isActive: Bool?
    let unavailableNote, unavailableReason: String?
    let committedUnits, coverageDays: Int?
    let coverageStartDate, linkedTransferID, lockID: String?
    let pendingConfirmationUnits, projectedUnits: Int?
    let requestedBy: String?
    let requestedUnits: Int?
    let transferMode: String?

    enum CodingKeys: String, CodingKey {
        case type
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
        case amountMinor = "amount_minor"
        case chargebackID = "chargeback_id"
        case claimID = "claim_id"
        case claimType = "claim_type"
        case photoUrls = "photo_urls"
        case resolutionNote = "resolution_note"
        case settlementMode = "settlement_mode"
        case source
        case commandID = "command_id"
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
        case productID = "product_id"
        case signalID = "signal_id"
        case simulationID = "simulation_id"
        case transferRecommendations = "transfer_recommendations"
        case ttlSeconds = "ttl_seconds"
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
        case attemptCount = "attempt_count"
        case depth, escalated
        case fromManifestID = "from_manifest_id"
        case fromVehicleID = "from_vehicle_id"
        case orderCount = "order_count"
        case state
        case stopCount = "stop_count"
        case toManifestID = "to_manifest_id"
        case toVehicleID = "to_vehicle_id"
        case totalVolumeVu = "total_volume_vu"
        case transferCount = "transfer_count"
        case transferID = "transfer_id"
        case driverIDS = "driver_ids"
        case manifestIDS = "manifest_ids"
        case orderIDS = "order_ids"
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
        case reasonCode = "reason_code"
        case executionAction = "execution_action"
        case executionMode = "execution_mode"
        case gateway
        case policySource = "policy_source"
        case providerReference = "provider_reference"
        case transactionID = "transaction_id"
        case preOrderID = "pre_order_id"
        case handlingClass = "handling_class"
        case isHazardous = "is_hazardous"
        case isPerishable = "is_perishable"
        case requiresColdChain = "requires_cold_chain"
        case storageTempMaxC = "storage_temp_max_c"
        case storageTempMinC = "storage_temp_min_c"
        case promotionID = "promotion_id"
        case retailerIDS = "retailer_ids"
        case retailerScope = "retailer_scope"
        case creditLimitMinor = "credit_limit_minor"
        case currentBalance = "current_balance"
        case requestedAmount = "requested_amount"
        case delinquent
        case profileID = "profile_id"
        case riskTier = "risk_tier"
        case priceMinor = "price_minor"
        case setBy = "set_by"
        case setByRole = "set_by_role"
        case countryCode = "country_code"
        case name, phone
        case assignedFactoryID = "assigned_factory_id"
        case categories, country
        case isConfigured = "is_configured"
        case isRegistered = "is_registered"
        case fromWarehouse = "from_warehouse"
        case toWarehouse = "to_warehouse"
        case isActive = "is_active"
        case unavailableNote = "unavailable_note"
        case unavailableReason = "unavailable_reason"
        case committedUnits = "committed_units"
        case coverageDays = "coverage_days"
        case coverageStartDate = "coverage_start_date"
        case linkedTransferID = "linked_transfer_id"
        case lockID = "lock_id"
        case pendingConfirmationUnits = "pending_confirmation_units"
        case projectedUnits = "projected_units"
        case requestedBy = "requested_by"
        case requestedUnits = "requested_units"
        case transferMode = "transfer_mode"
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
        type: TypeEnum? = nil,
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
        amountMinor: Int?? = nil,
        chargebackID: String?? = nil,
        claimID: String?? = nil,
        claimType: String?? = nil,
        photoUrls: [String]?? = nil,
        resolutionNote: String?? = nil,
        settlementMode: String?? = nil,
        source: String?? = nil,
        commandID: String?? = nil,
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
        productID: String?? = nil,
        signalID: String?? = nil,
        simulationID: String?? = nil,
        transferRecommendations: Int?? = nil,
        ttlSeconds: Int?? = nil,
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
        attemptCount: Int?? = nil,
        depth: Int?? = nil,
        escalated: Bool?? = nil,
        fromManifestID: String?? = nil,
        fromVehicleID: String?? = nil,
        orderCount: Int?? = nil,
        state: String?? = nil,
        stopCount: Int?? = nil,
        toManifestID: String?? = nil,
        toVehicleID: String?? = nil,
        totalVolumeVu: TotalVolumeVu?? = nil,
        transferCount: Int?? = nil,
        transferID: String?? = nil,
        driverIDS: [String]?? = nil,
        manifestIDS: [String]?? = nil,
        orderIDS: [String]?? = nil,
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
        reasonCode: String?? = nil,
        executionAction: String?? = nil,
        executionMode: String?? = nil,
        gateway: String?? = nil,
        policySource: String?? = nil,
        providerReference: String?? = nil,
        transactionID: String?? = nil,
        preOrderID: String?? = nil,
        handlingClass: String?? = nil,
        isHazardous: Bool?? = nil,
        isPerishable: Bool?? = nil,
        requiresColdChain: Bool?? = nil,
        storageTempMaxC: Double?? = nil,
        storageTempMinC: Double?? = nil,
        promotionID: String?? = nil,
        retailerIDS: [String]?? = nil,
        retailerScope: String?? = nil,
        creditLimitMinor: Int?? = nil,
        currentBalance: Int?? = nil,
        requestedAmount: Int?? = nil,
        delinquent: Bool?? = nil,
        profileID: String?? = nil,
        riskTier: String?? = nil,
        priceMinor: Int?? = nil,
        setBy: String?? = nil,
        setByRole: String?? = nil,
        countryCode: String?? = nil,
        name: String?? = nil,
        phone: String?? = nil,
        assignedFactoryID: String?? = nil,
        categories: [String]?? = nil,
        country: String?? = nil,
        isConfigured: Bool?? = nil,
        isRegistered: Bool?? = nil,
        fromWarehouse: String?? = nil,
        toWarehouse: String?? = nil,
        isActive: Bool?? = nil,
        unavailableNote: String?? = nil,
        unavailableReason: String?? = nil,
        committedUnits: Int?? = nil,
        coverageDays: Int?? = nil,
        coverageStartDate: String?? = nil,
        linkedTransferID: String?? = nil,
        lockID: String?? = nil,
        pendingConfirmationUnits: Int?? = nil,
        projectedUnits: Int?? = nil,
        requestedBy: String?? = nil,
        requestedUnits: Int?? = nil,
        transferMode: String?? = nil
    ) -> PegasusWSEventEnvelope {
        return PegasusWSEventEnvelope(
            type: type ?? self.type,
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
            amountMinor: amountMinor ?? self.amountMinor,
            chargebackID: chargebackID ?? self.chargebackID,
            claimID: claimID ?? self.claimID,
            claimType: claimType ?? self.claimType,
            photoUrls: photoUrls ?? self.photoUrls,
            resolutionNote: resolutionNote ?? self.resolutionNote,
            settlementMode: settlementMode ?? self.settlementMode,
            source: source ?? self.source,
            commandID: commandID ?? self.commandID,
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
            productID: productID ?? self.productID,
            signalID: signalID ?? self.signalID,
            simulationID: simulationID ?? self.simulationID,
            transferRecommendations: transferRecommendations ?? self.transferRecommendations,
            ttlSeconds: ttlSeconds ?? self.ttlSeconds,
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
            attemptCount: attemptCount ?? self.attemptCount,
            depth: depth ?? self.depth,
            escalated: escalated ?? self.escalated,
            fromManifestID: fromManifestID ?? self.fromManifestID,
            fromVehicleID: fromVehicleID ?? self.fromVehicleID,
            orderCount: orderCount ?? self.orderCount,
            state: state ?? self.state,
            stopCount: stopCount ?? self.stopCount,
            toManifestID: toManifestID ?? self.toManifestID,
            toVehicleID: toVehicleID ?? self.toVehicleID,
            totalVolumeVu: totalVolumeVu ?? self.totalVolumeVu,
            transferCount: transferCount ?? self.transferCount,
            transferID: transferID ?? self.transferID,
            driverIDS: driverIDS ?? self.driverIDS,
            manifestIDS: manifestIDS ?? self.manifestIDS,
            orderIDS: orderIDS ?? self.orderIDS,
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
            reasonCode: reasonCode ?? self.reasonCode,
            executionAction: executionAction ?? self.executionAction,
            executionMode: executionMode ?? self.executionMode,
            gateway: gateway ?? self.gateway,
            policySource: policySource ?? self.policySource,
            providerReference: providerReference ?? self.providerReference,
            transactionID: transactionID ?? self.transactionID,
            preOrderID: preOrderID ?? self.preOrderID,
            handlingClass: handlingClass ?? self.handlingClass,
            isHazardous: isHazardous ?? self.isHazardous,
            isPerishable: isPerishable ?? self.isPerishable,
            requiresColdChain: requiresColdChain ?? self.requiresColdChain,
            storageTempMaxC: storageTempMaxC ?? self.storageTempMaxC,
            storageTempMinC: storageTempMinC ?? self.storageTempMinC,
            promotionID: promotionID ?? self.promotionID,
            retailerIDS: retailerIDS ?? self.retailerIDS,
            retailerScope: retailerScope ?? self.retailerScope,
            creditLimitMinor: creditLimitMinor ?? self.creditLimitMinor,
            currentBalance: currentBalance ?? self.currentBalance,
            requestedAmount: requestedAmount ?? self.requestedAmount,
            delinquent: delinquent ?? self.delinquent,
            profileID: profileID ?? self.profileID,
            riskTier: riskTier ?? self.riskTier,
            priceMinor: priceMinor ?? self.priceMinor,
            setBy: setBy ?? self.setBy,
            setByRole: setByRole ?? self.setByRole,
            countryCode: countryCode ?? self.countryCode,
            name: name ?? self.name,
            phone: phone ?? self.phone,
            assignedFactoryID: assignedFactoryID ?? self.assignedFactoryID,
            categories: categories ?? self.categories,
            country: country ?? self.country,
            isConfigured: isConfigured ?? self.isConfigured,
            isRegistered: isRegistered ?? self.isRegistered,
            fromWarehouse: fromWarehouse ?? self.fromWarehouse,
            toWarehouse: toWarehouse ?? self.toWarehouse,
            isActive: isActive ?? self.isActive,
            unavailableNote: unavailableNote ?? self.unavailableNote,
            unavailableReason: unavailableReason ?? self.unavailableReason,
            committedUnits: committedUnits ?? self.committedUnits,
            coverageDays: coverageDays ?? self.coverageDays,
            coverageStartDate: coverageStartDate ?? self.coverageStartDate,
            linkedTransferID: linkedTransferID ?? self.linkedTransferID,
            lockID: lockID ?? self.lockID,
            pendingConfirmationUnits: pendingConfirmationUnits ?? self.pendingConfirmationUnits,
            projectedUnits: projectedUnits ?? self.projectedUnits,
            requestedBy: requestedBy ?? self.requestedBy,
            requestedUnits: requestedUnits ?? self.requestedUnits,
            transferMode: transferMode ?? self.transferMode
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

enum TypeEnum: String, Codable {
    case aiRecommendationCreated = "AI_RECOMMENDATION_CREATED"
    case aiRecommendationDecided = "AI_RECOMMENDATION_DECIDED"
    case allocationFairShareApplied = "ALLOCATION_FAIR_SHARE_APPLIED"
    case allocationPolicyApplied = "ALLOCATION_POLICY_APPLIED"
    case cartSyncUpdated = "CART_SYNC_UPDATED"
    case cashOverage = "CASH_OVERAGE"
    case cashShortfall = "CASH_SHORTFALL"
    case claimFiled = "CLAIM_FILED"
    case claimResolved = "CLAIM_RESOLVED"
    case commandDispatched = "COMMAND_DISPATCHED"
    case commandReceived = "COMMAND_RECEIVED"
    case commandSettled = "COMMAND_SETTLED"
    case creditDeliveryMarked = "CREDIT_DELIVERY_MARKED"
    case creditDeliveryResolved = "CREDIT_DELIVERY_RESOLVED"
    case creditLeave = "CREDIT_LEAVE"
    case deliveryDisputed = "DELIVERY_DISPUTED"
    case deliverySessionUpdated = "DELIVERY_SESSION_UPDATED"
    case demandBaselineUpdated = "DEMAND_BASELINE_UPDATED"
    case dispatchZoneOverride = "DISPATCH_ZONE_OVERRIDE"
    case driverAvailabilityChanged = "DRIVER_AVAILABILITY_CHANGED"
    case driverCreated = "DRIVER_CREATED"
    case driverLocationUpdated = "DRIVER_LOCATION_UPDATED"
    case driverReturnApproaching = "DRIVER_RETURN_APPROACHING"
    case factoryCreated = "FACTORY_CREATED"
    case factoryLocationUpdated = "FACTORY_LOCATION_UPDATED"
    case factorySupplyRequestUpdate = "FACTORY_SUPPLY_REQUEST_UPDATE"
    case fiscalReceiptFailed = "FISCAL_RECEIPT_FAILED"
    case fiscalReceiptRequested = "FISCAL_RECEIPT_REQUESTED"
    case fiscalReceiptSucceeded = "FISCAL_RECEIPT_SUCCEEDED"
    case freezeLockAcquired = "FREEZE_LOCK_ACQUIRED"
    case freezeLockReleased = "FREEZE_LOCK_RELEASED"
    case inventoryImportStatusUpdate = "INVENTORY_IMPORT_STATUS_UPDATE"
    case inventoryImportUploaded = "INVENTORY_IMPORT_UPLOADED"
    case inventorySyncComplete = "INVENTORY_SYNC_COMPLETE"
    case logisticsExceptionReported = "LOGISTICS_EXCEPTION_REPORTED"
    case logisticsTelemetry = "LOGISTICS_TELEMETRY"
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
    case orderAllocated = "ORDER_ALLOCATED"
    case orderAmended = "ORDER_AMENDED"
    case orderAssigned = "ORDER_ASSIGNED"
    case orderCapacityOverflow = "ORDER_CAPACITY_OVERFLOW"
    case orderConditionReported = "ORDER_CONDITION_REPORTED"
    case orderCreated = "ORDER_CREATED"
    case orderFinalized = "ORDER_FINALIZED"
    case orderForceCompleted = "ORDER_FORCE_COMPLETED"
    case orderReassigned = "ORDER_REASSIGNED"
    case orderStatusChanged = "ORDER_STATUS_CHANGED"
    case orderValidationFailed = "ORDER_VALIDATION_FAILED"
    case partialOffload = "PARTIAL_OFFLOAD"
    case paymentCleared = "PAYMENT_CLEARED"
    case paymentFailed = "PAYMENT_FAILED"
    case paymentRequired = "PAYMENT_REQUIRED"
    case planningAgentBroadcast = "PLANNING_AGENT_BROADCAST"
    case planningConfidenceDowngraded = "PLANNING_CONFIDENCE_DOWNGRADED"
    case planningForecastUpdated = "PLANNING_FORECAST_UPDATED"
    case planningMeioRecommendationV1 = "planning.meio.recommendation.v1"
    case planningPromoSimulationReady = "PLANNING_PROMO_SIMULATION_READY"
    case planningSignalIngestV1 = "planning.signal.ingest.v1"
    case preOrderAutoAccepted = "PRE_ORDER_AUTO_ACCEPTED"
    case preOrderCancelled = "PRE_ORDER_CANCELLED"
    case preOrderConfirmation = "PRE_ORDER_CONFIRMATION"
    case preOrderConfirmed = "PRE_ORDER_CONFIRMED"
    case preOrderDateAccepted = "PRE_ORDER_DATE_ACCEPTED"
    case preOrderDateProposed = "PRE_ORDER_DATE_PROPOSED"
    case preOrderDateRejected = "PRE_ORDER_DATE_REJECTED"
    case preOrderEdited = "PRE_ORDER_EDITED"
    case preOrderNotified = "PRE_ORDER_NOTIFIED"
    case preOrderNudge = "PRE_ORDER_NUDGE"
    case productHandlingUpdated = "PRODUCT_HANDLING_UPDATED"
    case promotionChanged = "PROMOTION_CHANGED"
    case proximityUnlocked = "PROXIMITY_UNLOCKED"
    case replenishmentAutoApproved = "REPLENISHMENT_AUTO_APPROVED"
    case replenishmentInsightCreated = "REPLENISHMENT_INSIGHT_CREATED"
    case retailerCreditLimitBreached = "RETAILER_CREDIT_LIMIT_BREACHED"
    case retailerCreditProfileChanged = "RETAILER_CREDIT_PROFILE_CHANGED"
    case retailerPriceOverride = "RETAILER_PRICE_OVERRIDE"
    case retailerRegistered = "RETAILER_REGISTERED"
    case retailerSegmentUpdated = "RETAILER_SEGMENT_UPDATED"
    case returnReceivedAtWarehouse = "RETURN_RECEIVED_AT_WAREHOUSE"
    case reverseLogisticsRequired = "REVERSE_LOGISTICS_REQUIRED"
    case routeCreated = "ROUTE_CREATED"
    case routeReordered = "ROUTE_REORDERED"
    case servicePolicyUpdated = "SERVICE_POLICY_UPDATED"
    case settlementRequired = "SETTLEMENT_REQUIRED"
    case shopClosed = "SHOP_CLOSED"
    case shopClosedBypassOffload = "SHOP_CLOSED_BYPASS_OFFLOAD"
    case shopClosedEscalated = "SHOP_CLOSED_ESCALATED"
    case shopClosedResolved = "SHOP_CLOSED_RESOLVED"
    case shopClosedResponse = "SHOP_CLOSED_RESPONSE"
    case shopClosedTimeout = "SHOP_CLOSED_TIMEOUT"
    case skuClassUpdated = "SKU_CLASS_UPDATED"
    case splitPaymentCreated = "SPLIT_PAYMENT_CREATED"
    case splitShipmentCreated = "SPLIT_SHIPMENT_CREATED"
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
    case warehouseLocationUpdated = "WAREHOUSE_LOCATION_UPDATED"
    case warehouseSupplyRequestOpened = "WAREHOUSE_SUPPLY_REQUEST_OPENED"
    case warehouseTransferCreated = "WAREHOUSE_TRANSFER_CREATED"
    case warehouseTransferReceived = "WAREHOUSE_TRANSFER_RECEIVED"
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
