// To parse the JSON, install kotlin's serialization plugin and do:
//
// val json                   = Json { allowStructuredMapKeys = true }
// val pegasusWSEventEnvelope = json.parse(PegasusWSEventEnvelope.serializer(), jsonString)

package com.pegasus.payload.generated.contracts

import kotlinx.serialization.*
import kotlinx.serialization.json.*
import kotlinx.serialization.descriptors.*
import kotlinx.serialization.encoding.*

@Serializable
data class PegasusWSEventEnvelope (
    @SerialName("aggregate_id")
    val aggregateID: String? = null,

    @SerialName("aggregate_type")
    val aggregateType: String? = null,

    @SerialName("base_event")
    val baseEvent: JsonElement?,

    @SerialName("decided_by")
    val decidedBy: String? = null,

    val decision: String? = null,
    val note: String? = null,

    @SerialName("recommendation_id")
    val recommendationID: String? = null,

    val status: String? = null,

    @SerialName("supplier_id")
    val supplierID: String? = null,

    val action: String? = null,

    @SerialName("actor_id")
    val actorID: String? = null,

    @SerialName("actor_role")
    val actorRole: String? = null,

    @SerialName("attempt_id")
    val attemptID: String? = null,

    @SerialName("confirmation_status")
    val confirmationStatus: String? = null,

    val currency: String? = null,

    @SerialName("customer_progress_percent")
    val customerProgressPercent: Long? = null,

    @SerialName("customer_stage")
    val customerStage: String? = null,

    @SerialName("driver_id")
    val driverID: String? = null,

    @SerialName("escalated_to")
    val escalatedTo: String? = null,

    @SerialName("from_driver_id")
    val fromDriverID: String? = null,

    @SerialName("from_route_id")
    val fromRouteID: String? = null,

    @SerialName("gps_lat")
    val gpsLat: Double? = null,

    @SerialName("gps_lng")
    val gpsLng: Double? = null,

    @SerialName("h3_cell")
    val h3Cell: String? = null,

    val lat: Double? = null,

    @SerialName("license_plate")
    val licensePlate: String? = null,

    @SerialName("line_items")
    val lineItems: JsonElement? = null,

    val lng: Double? = null,

    @SerialName("manifest_id")
    val manifestID: String? = null,

    val message: String? = null,

    @SerialName("negotiation_id")
    val negotiationID: String? = null,

    @SerialName("order_id")
    val orderID: String? = null,

    @SerialName("order_source")
    val orderSource: String? = null,

    @SerialName("payment_method")
    val paymentMethod: String? = null,

    @SerialName("previous_status")
    val previousStatus: String? = null,

    @SerialName("proposal_id")
    val proposalID: String? = null,

    @SerialName("proposed_price_minor")
    val proposedPriceMinor: Long? = null,

    val reason: String? = null,

    @SerialName("receiving_window_close")
    val receivingWindowClose: String? = null,

    @SerialName("receiving_window_open")
    val receivingWindowOpen: String? = null,

    @SerialName("requested_delivery_date")
    val requestedDeliveryDate: String? = null,

    val resolution: String? = null,
    val response: String? = null,

    @SerialName("retailer_id")
    val retailerID: String? = null,

    @SerialName("route_id")
    val routeID: String? = null,

    @SerialName("to_driver_id")
    val toDriverID: String? = null,

    @SerialName("to_route_id")
    val toRouteID: String? = null,

    @SerialName("total_minor")
    val totalMinor: Long? = null,

    @SerialName("vehicle_id")
    val vehicleID: String? = null,

    val version: Version? = null,

    @SerialName("warehouse_id")
    val warehouseID: String? = null,

    @SerialName("baseline_qty")
    val baselineQty: Long? = null,

    @SerialName("baseline_source")
    val baselineSource: String? = null,

    @SerialName("blocked_reason")
    val blockedReason: String? = null,

    val confidence: Double? = null,

    @SerialName("confidence_pct")
    val confidencePct: Long? = null,

    @SerialName("factory_id")
    val factoryID: String? = null,

    @SerialName("high_units")
    val highUnits: Long? = null,

    @SerialName("insight_id")
    val insightID: String? = null,

    @SerialName("low_units")
    val lowUnits: Long? = null,

    @SerialName("network_nodes")
    val networkNodes: Long? = null,

    @SerialName("override_id")
    val overrideID: String? = null,

    @SerialName("polygon_geojson")
    val polygonGeojson: String? = null,

    @SerialName("product_id")
    val productID: String? = null,

    @SerialName("published_by")
    val publishedBy: String? = null,

    @SerialName("scenario_id")
    val scenarioID: String? = null,

    @SerialName("signal_id")
    val signalID: String? = null,

    @SerialName("simulation_id")
    val simulationID: String? = null,

    @SerialName("transfer_recommendations")
    val transferRecommendations: Long? = null,

    @SerialName("ttl_seconds")
    val ttlSeconds: Long? = null,

    @SerialName("aging_bucket")
    val agingBucket: String? = null,

    @SerialName("amount_minor")
    val amountMinor: Long? = null,

    @SerialName("balance_minor")
    val balanceMinor: Long? = null,

    @SerialName("due_at")
    val dueAt: String? = null,

    @SerialName("dunning_step")
    val dunningStep: Long? = null,

    @SerialName("invoice_id")
    val invoiceID: String? = null,

    @SerialName("last_dunned_at")
    val lastDunnedAt: String? = null,

    @SerialName("principal_minor")
    val principalMinor: Long? = null,

    val deadline: String? = null,

    @SerialName("ehf_id")
    val ehfID: String? = null,

    @SerialName("account_holder")
    val accountHolder: String? = null,

    @SerialName("assigned_warehouse_id")
    val assignedWarehouseID: String? = null,

    @SerialName("bank_name")
    val bankName: String? = null,

    @SerialName("contact_name")
    val contactName: String? = null,

    val email: String? = null,

    @SerialName("legal_name")
    val legalName: String? = null,

    @SerialName("selected_gateways")
    val selectedGateways: List<String>? = null,

    @SerialName("supplier_role")
    val supplierRole: String? = null,

    @SerialName("user_id")
    val userID: String? = null,

    @SerialName("expected_minor")
    val expectedMinor: Long? = null,

    @SerialName("overage_minor")
    val overageMinor: Long? = null,

    @SerialName("received_minor")
    val receivedMinor: Long? = null,

    @SerialName("shortfall_minor")
    val shortfallMinor: Long? = null,

    @SerialName("trace_id")
    val traceID: String? = null,

    @SerialName("category_id")
    val categoryID: String? = null,

    @SerialName("icon_key")
    val iconKey: String? = null,

    val name: String? = null,

    @SerialName("parent_category_id")
    val parentCategoryID: String? = null,

    @SerialName("sort_order")
    val sortOrder: Long? = null,

    @SerialName("chargeback_id")
    val chargebackID: String? = null,

    @SerialName("claim_id")
    val claimID: String? = null,

    @SerialName("claim_type")
    val claimType: String? = null,

    @SerialName("photo_urls")
    val photoUrls: List<String>? = null,

    @SerialName("resolution_note")
    val resolutionNote: String? = null,

    @SerialName("settlement_mode")
    val settlementMode: String? = null,

    val source: String? = null,

    @SerialName("command_id")
    val commandID: String? = null,

    val actor: String? = null,

    @SerialName("campaign_id")
    val campaignID: String? = null,

    @SerialName("impacted_lot_count")
    val impactedLotCount: Long? = null,

    @SerialName("impacted_order_count")
    val impactedOrderCount: Long? = null,

    @SerialName("impacted_units_count")
    val impactedUnitsCount: Long? = null,

    @SerialName("lot_code")
    val lotCode: String? = null,

    @SerialName("lot_id")
    val lotID: String? = null,

    @SerialName("recall_reason")
    val recallReason: String? = null,

    val severity: String? = null,

    @SerialName("photo_proof_url")
    val photoProofURL: String? = null,

    @SerialName("signature_url")
    val signatureURL: String? = null,

    @SerialName("session_id")
    val sessionID: String? = null,

    @SerialName("country_code")
    val countryCode: String? = null,

    val phone: String? = null,

    @SerialName("order_count")
    val orderCount: Long? = null,

    @SerialName("order_ids")
    val orderIDS: List<String>? = null,

    val available: Boolean? = null,

    @SerialName("home_node_id")
    val homeNodeID: String? = null,

    @SerialName("home_node_type")
    val homeNodeType: String? = null,

    @SerialName("on_shift")
    val onShift: Boolean? = null,

    @SerialName("return_id")
    val returnID: String? = null,

    @SerialName("item_count")
    val itemCount: Long? = null,

    @SerialName("request_id")
    val requestID: String? = null,

    @SerialName("error_code")
    val errorCode: String? = null,

    @SerialName("error_message")
    val errorMessage: String? = null,

    @SerialName("fiscal_qr")
    val fiscalQr: String? = null,

    @SerialName("fiscal_receipt_id")
    val fiscalReceiptID: String? = null,

    val provider: String? = null,

    @SerialName("lock_name")
    val lockName: String? = null,

    @SerialName("system_id")
    val systemID: String? = null,

    @SerialName("gcs_path")
    val gcsPath: String? = null,

    @SerialName("suggested_mappings")
    val suggestedMappings: Long? = null,

    @SerialName("committed_units")
    val committedUnits: Long? = null,

    @SerialName("coverage_days")
    val coverageDays: Long? = null,

    @SerialName("coverage_start_date")
    val coverageStartDate: String? = null,

    @SerialName("linked_transfer_id")
    val linkedTransferID: String? = null,

    @SerialName("lock_id")
    val lockID: String? = null,

    @SerialName("pending_confirmation_units")
    val pendingConfirmationUnits: Long? = null,

    @SerialName("projected_units")
    val projectedUnits: Long? = null,

    @SerialName("requested_by")
    val requestedBy: String? = null,

    @SerialName("requested_units")
    val requestedUnits: Long? = null,

    val state: String? = null,

    @SerialName("transfer_id")
    val transferID: String? = null,

    @SerialName("transfer_mode")
    val transferMode: String? = null,

    @SerialName("decision_reason")
    val decisionReason: String? = null,

    @SerialName("normalized_inn")
    val normalizedInn: String? = null,

    @SerialName("requester_name")
    val requesterName: String? = null,

    @SerialName("requester_phone")
    val requesterPhone: String? = null,

    @SerialName("role_sought")
    val roleSought: String? = null,

    @SerialName("target_tenant_id")
    val targetTenantID: String? = null,

    @SerialName("target_tenant_type")
    val targetTenantType: String? = null,

    @SerialName("attempt_count")
    val attemptCount: Long? = null,

    val depth: Long? = null,
    val escalated: Boolean? = null,

    @SerialName("from_manifest_id")
    val fromManifestID: String? = null,

    @SerialName("from_vehicle_id")
    val fromVehicleID: String? = null,

    @SerialName("manifest_domain")
    val manifestDomain: String? = null,

    @SerialName("stop_count")
    val stopCount: Long? = null,

    @SerialName("to_manifest_id")
    val toManifestID: String? = null,

    @SerialName("to_vehicle_id")
    val toVehicleID: String? = null,

    @SerialName("total_volume_vu")
    val totalVolumeVu: TotalVolumeVu? = null,

    @SerialName("transfer_count")
    val transferCount: Long? = null,

    @SerialName("bundle_id")
    val bundleID: String? = null,

    val domain: String? = null,

    @SerialName("imported_count")
    val importedCount: Long? = null,

    @SerialName("processed_rows")
    val processedRows: Long? = null,

    @SerialName("project_id")
    val projectID: String? = null,

    @SerialName("quarantine_count")
    val quarantineCount: Long? = null,

    @SerialName("report_id")
    val reportID: String? = null,

    @SerialName("total_rows")
    val totalRows: Long? = null,

    @SerialName("driver_ids")
    val driverIDS: List<String>? = null,

    @SerialName("manifest_ids")
    val manifestIDS: List<String>? = null,

    @SerialName("shared_route_id")
    val sharedRouteID: String? = null,

    @SerialName("split_group_id")
    val splitGroupID: String? = null,

    @SerialName("truck_count")
    val truckCount: Long? = null,

    @SerialName("condition_type")
    val conditionType: String? = null,

    @SerialName("gcs_paths")
    val gcsPaths: List<String>? = null,

    val notes: String? = null,
    val quantity: Long? = null,

    @SerialName("reporter_id")
    val reporterID: String? = null,

    @SerialName("reporter_role")
    val reporterRole: String? = null,

    val sku: String? = null,

    @SerialName("delivered_at")
    val deliveredAt: String? = null,

    @SerialName("dispute_note")
    val disputeNote: String? = null,

    @SerialName("proof_photo_url")
    val proofPhotoURL: String? = null,

    @SerialName("proof_signature")
    val proofSignature: String? = null,

    @SerialName("reason_code")
    val reasonCode: String? = null,

    @SerialName("card_minor")
    val cardMinor: Long? = null,

    @SerialName("cash_minor")
    val cashMinor: Long? = null,

    @SerialName("execution_action")
    val executionAction: String? = null,

    @SerialName("execution_mode")
    val executionMode: String? = null,

    val gateway: String? = null,

    @SerialName("policy_source")
    val policySource: String? = null,

    @SerialName("provider_reference")
    val providerReference: String? = null,

    @SerialName("transaction_id")
    val transactionID: String? = null,

    @SerialName("batch_id")
    val batchID: String? = null,

    @SerialName("net_payout_minor")
    val netPayoutMinor: Long? = null,

    @SerialName("rail_reference")
    val railReference: String? = null,

    @SerialName("pre_order_id")
    val preOrderID: String? = null,

    val barcode: String? = null,

    @SerialName("handling_class")
    val handlingClass: String? = null,

    @SerialName("is_active")
    val isActive: Boolean? = null,

    @SerialName("is_hazardous")
    val isHazardous: Boolean? = null,

    @SerialName("is_perishable")
    val isPerishable: Boolean? = null,

    @SerialName("previous_price_minor")
    val previousPriceMinor: Long? = null,

    @SerialName("price_minor")
    val priceMinor: Long? = null,

    @SerialName("stock_quantity")
    val stockQuantity: Long? = null,

    val unit: String? = null,

    @SerialName("promotion_id")
    val promotionID: String? = null,

    @SerialName("retailer_ids")
    val retailerIDS: List<String>? = null,

    @SerialName("retailer_scope")
    val retailerScope: String? = null,

    @SerialName("credit_limit_minor")
    val creditLimitMinor: Long? = null,

    @SerialName("current_balance")
    val currentBalance: Long? = null,

    @SerialName("requested_amount")
    val requestedAmount: Long? = null,

    val delinquent: Boolean? = null,

    @SerialName("profile_id")
    val profileID: String? = null,

    @SerialName("risk_tier")
    val riskTier: String? = null,

    @SerialName("set_by")
    val setBy: String? = null,

    @SerialName("set_by_role")
    val setByRole: String? = null,

    @SerialName("assigned_factory_id")
    val assignedFactoryID: String? = null,

    val categories: List<String>? = null,
    val country: String? = null,

    @SerialName("is_configured")
    val isConfigured: Boolean? = null,

    @SerialName("is_registered")
    val isRegistered: Boolean? = null,

    @SerialName("breach_reason")
    val breachReason: String? = null,

    @SerialName("breached_at")
    val breachedAt: String? = null,

    @SerialName("cutoff_applied_at")
    val cutoffAppliedAt: String? = null,

    @SerialName("fill_rate_target_bps")
    val fillRateTargetBps: Long? = null,

    @SerialName("guaranteed_delivery_date")
    val guaranteedDeliveryDate: String? = null,

    @SerialName("min_order_minor")
    val minOrderMinor: Long? = null,

    @SerialName("promise_type")
    val promiseType: String? = null,

    @SerialName("sla_hours")
    val slaHours: Long? = null,

    @SerialName("from_warehouse")
    val fromWarehouse: String? = null,

    @SerialName("to_warehouse")
    val toWarehouse: String? = null,

    @SerialName("unavailable_note")
    val unavailableNote: String? = null,

    @SerialName("unavailable_reason")
    val unavailableReason: String? = null
)

@Serializable
sealed class TotalVolumeVu {
    class DoubleValue(val value: Double) : TotalVolumeVu()
    class IntegerValue(val value: Long)  : TotalVolumeVu()
}

@Serializable
sealed class Version {
    class IntegerValue(val value: Long)  : Version()
    class StringValue(val value: String) : Version()
}
