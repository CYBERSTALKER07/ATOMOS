package com.pegasusx.supplier.data.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

@Serializable
data class SupplierExceptionRow(
    @SerialName("order_id") val orderId: String,
    val kind: String = "",
    val status: String = "",
    @SerialName("retailer_id") val retailerId: String? = null,
    val note: String? = null,
    @SerialName("manifest_id") val manifestId: String? = null,
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class SupplierExceptionsResponse(
    val exceptions: List<SupplierExceptionRow> = emptyList(),
)

@Serializable
data class SupplierManifestRow(
    @SerialName("manifest_id") val manifestId: String,
    val status: String = "",
    val state: String = "",
    @SerialName("orders_count") val ordersCount: Int = 0,
    @SerialName("driver_id") val driverId: String? = null,
    @SerialName("driver_name") val driverName: String = "",
    @SerialName("vehicle_id") val vehicleId: String? = null,
    @SerialName("vehicle_plate") val vehiclePlate: String? = null,
    @SerialName("truck_id") val truckId: String? = null,
    @SerialName("total_vu") val totalVu: Long = 0,
    @SerialName("total_volume_vu") val totalVolumeVu: Double = 0.0,
    @SerialName("max_volume_vu") val maxVolumeVu: Double = 0.0,
    @SerialName("stop_count") val stopCount: Int = 0,
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class SupplierManifestsResponse(
    val manifests: List<SupplierManifestRow> = emptyList(),
)

@Serializable
data class SupplierManifestOrderWire(
    @SerialName("order_id") val orderId: String,
    @SerialName("retailer_id") val retailerId: String? = null,
    val amount: Long = 0,
    val state: String = "",
    val status: String = "",
)

@Serializable
data class SupplierManifestDetail(
    @SerialName("manifest_id") val manifestId: String,
    val status: String = "",
    val state: String = "",
    @SerialName("orders_count") val ordersCount: Int = 0,
    @SerialName("driver_id") val driverId: String? = null,
    @SerialName("driver_name") val driverName: String = "",
    @SerialName("vehicle_plate") val vehiclePlate: String? = null,
    @SerialName("total_vu") val totalVu: Long = 0,
    @SerialName("total_volume_vu") val totalVolumeVu: Double = 0.0,
    @SerialName("max_volume_vu") val maxVolumeVu: Double = 0.0,
    @SerialName("updated_at") val updatedAt: String = "",
    val orders: List<SupplierManifestOrderWire> = emptyList(),
)

@Serializable
data class SupplierManifestExceptionRow(
    @SerialName("exception_id") val exceptionId: String,
    @SerialName("manifest_id") val manifestId: String,
    @SerialName("order_id") val orderId: String,
    val reason: String = "",
    val metadata: String? = null,
    @SerialName("attempt_count") val attemptCount: Long = 0,
    val escalated: Boolean = false,
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class SupplierManifestExceptionsResponse(
    val exceptions: List<SupplierManifestExceptionRow> = emptyList(),
)

@Serializable
data class SupplierManifestInjectOrderRequest(
    @SerialName("order_id") val orderId: String,
    @SerialName("volume_vu") val volumeVu: Int? = null,
)

@Serializable
data class SupplierDispatchOrderRow(
    @SerialName("order_id") val orderId: String,
    @SerialName("volume_vu") val volumeVu: Double = 0.0,
    @SerialName("retailer_id") val retailerId: String? = null,
)

@Serializable
data class SupplierDispatchDriverRow(
    @SerialName("driver_id") val driverId: String,
    val name: String = "",
    @SerialName("max_volume_vu") val maxVolumeVu: Double? = null,
)

@Serializable
data class SupplierDispatchManualRoute(
    @SerialName("driver_id") val driverId: String,
    @SerialName("order_ids") val orderIds: List<String>,
)

@Serializable
data class SupplierDispatchPreview(
    @SerialName("undispatched_orders") val undispatchedOrders: List<SupplierDispatchOrderRow> = emptyList(),
    @SerialName("available_drivers") val availableDrivers: List<SupplierDispatchDriverRow> = emptyList(),
    @SerialName("unavailable_drivers") val unavailableDrivers: List<JsonElement> = emptyList(),
    @SerialName("pending_count") val pendingCount: Int = 0,
    @SerialName("available_driver_count") val availableDriverCount: Int = 0,
    @SerialName("proposed_routes") val proposedRoutes: List<DispatchProposedRoute> = emptyList(),
    @SerialName("optimizer_source") val optimizerSource: String? = null,
    @SerialName("optimizer_warnings") val optimizerWarnings: List<String> = emptyList(),
    @SerialName("window_constrained_count") val windowConstrainedCount: Int = 0,
    @SerialName("plan_fingerprint") val planFingerprint: String? = null,
    @SerialName("warehouse_plan_fingerprint") val warehousePlanFingerprint: String? = null,
    @SerialName("plan_fingerprint_mismatch") val planFingerprintMismatch: Boolean = false,
)

@Serializable
data class SupplierPricingRule(
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("base_markup_bps") val baseMarkupBps: Long = 0,
    @SerialName("retailer_discount_bps") val retailerDiscountBps: Long = 0,
    @SerialName("min_margin_bps") val minMarginBps: Long = 0,
    val currency: String = "",
    @SerialName("rule_version") val ruleVersion: Long = 0,
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class RetailerPriceOverride(
    @SerialName("override_id") val overrideId: String = "",
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("retailer_id") val retailerId: String = "",
    @SerialName("product_id") val productId: String = "",
    val price: Long = 0,
    @SerialName("set_by") val setBy: String = "",
    @SerialName("set_by_role") val setByRole: String = "",
    @SerialName("is_active") val isActive: Boolean = true,
    val notes: String? = null,
    @SerialName("expires_at") val expiresAt: String? = null,
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class RetailerPriceOverridesResponse(
    val overrides: List<RetailerPriceOverride> = emptyList(),
    val total: Int = 0,
)

@Serializable
data class CreateRetailerPriceOverrideRequest(
    @SerialName("retailer_id") val retailerId: String,
    @SerialName("product_id") val productId: String,
    val price: Long,
    val notes: String? = null,
    @SerialName("expires_at") val expiresAt: String? = null,
)

@Serializable
data class CreateRetailerPriceOverrideResponse(
    val status: String = "",
    @SerialName("override_id") val overrideId: String = "",
    @SerialName("retailer_id") val retailerId: String = "",
    @SerialName("product_id") val productId: String = "",
    val price: Long = 0,
)

@Serializable
data class SupplierTopologyWarehouse(
    @SerialName("warehouse_id") val warehouseId: String,
    val name: String = "",
    val address: String = "",
    @SerialName("place_id") val placeId: String? = null,
    val lat: Double = 0.0,
    val lng: Double = 0.0,
    @SerialName("coverage_radius_km") val coverageRadiusKm: Double = 50.0,
    @SerialName("is_active") val isActive: Boolean = true,
    @SerialName("is_on_shift") val isOnShift: Boolean = true,
    @SerialName("transfer_mode") val transferMode: String = "TRUCK",
    @SerialName("co_locate_with_factory_id") val coLocateWithFactoryId: String? = null,
)

@Serializable
data class SupplierTopologyFactory(
    @SerialName("factory_id") val factoryId: String,
    val name: String = "",
    val address: String = "",
    @SerialName("place_id") val placeId: String? = null,
    val lat: Double = 0.0,
    val lng: Double = 0.0,
    @SerialName("is_active") val isActive: Boolean = true,
)

@Serializable
data class SupplierTopologyWarehouseInput(
    @SerialName("warehouse_id") val warehouseId: String? = null,
    val name: String,
    val address: String? = null,
    @SerialName("place_id") val placeId: String? = null,
    val lat: Double,
    val lng: Double,
    @SerialName("coverage_radius_km") val coverageRadiusKm: Double? = null,
    @SerialName("is_active") val isActive: Boolean? = null,
    @SerialName("is_on_shift") val isOnShift: Boolean? = null,
    @SerialName("transfer_mode") val transferMode: String? = null,
    @SerialName("co_locate_with_factory_id") val coLocateWithFactoryId: String? = null,
)

@Serializable
data class SupplierTopologyFactoryInput(
    @SerialName("factory_id") val factoryId: String? = null,
    val name: String,
    val address: String? = null,
    @SerialName("place_id") val placeId: String? = null,
    val lat: Double,
    val lng: Double,
    @SerialName("is_active") val isActive: Boolean? = null,
)

@Serializable
data class SupplierTopologyUpdateRequest(
    val warehouses: List<SupplierTopologyWarehouseInput>,
    val factories: List<SupplierTopologyFactoryInput>,
)

@Serializable
data class SupplierTopologyResponse(
    @SerialName("supplier_id") val supplierId: String = "",
    val warehouses: List<SupplierTopologyWarehouse> = emptyList(),
    val factories: List<SupplierTopologyFactory> = emptyList(),
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class SupplierSupplyLaneRow(
    @SerialName("lane_id") val laneId: String,
    val name: String = "",
    @SerialName("warehouse_id") val warehouseId: String = "",
    @SerialName("h3_cells") val h3Cells: Int = 0,
    val drivers: Int = 0,
    @SerialName("orders_today") val ordersToday: Int = 0,
    val capacity: Int = 0,
    @SerialName("utilization_pct") val utilizationPct: Double = 0.0,
)

@Serializable
data class SupplierSupplyLanesResponse(
    val lanes: List<SupplierSupplyLaneRow> = emptyList(),
)

@Serializable
data class SupplierWsSessionResponse(
    val token: String = "",
    @SerialName("expires_at") val expiresAt: String = "",
)

@Serializable
data class SupplierFleetOrderRow(
    val id: String = "",
    @SerialName("order_id") val orderId: String,
    @SerialName("retailer_id") val retailerId: String? = null,
    @SerialName("driver_id") val driverId: String? = null,
    val status: String = "",
    val state: String? = null,
    @SerialName("route_id") val routeId: String? = null,
    @SerialName("total_minor") val totalMinor: Long? = null,
    val currency: String? = null,
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class RouteCoordinateWire(
    val lat: Double,
    val lng: Double,
)

@Serializable
data class RouteGeometryWire(
    @SerialName("route_id") val routeId: String? = null,
    @SerialName("encoded_polyline") val encodedPolyline: String? = null,
    val coordinates: List<RouteCoordinateWire> = emptyList(),
    val source: String = "",
    @SerialName("stop_count") val stopCount: Int? = null,
)

@Serializable
data class DispatchProposedRoute(
    @SerialName("driver_id") val driverId: String? = null,
    @SerialName("driver_name") val driverName: String? = null,
    @SerialName("vehicle_id") val vehicleId: String? = null,
    @SerialName("order_ids") val orderIds: List<String> = emptyList(),
    @SerialName("volume_vu") val volumeVu: Double? = null,
    @SerialName("max_volume_vu") val maxVolumeVu: Double? = null,
    @SerialName("stop_count") val stopCount: Int? = null,
    @SerialName("route_geometry") val routeGeometry: RouteGeometryWire? = null,
)

@Serializable
data class SupplierDriverLocationWire(
    @SerialName("driver_id") val driverId: String,
    @SerialName("supplier_id") val supplierId: String? = null,
    val lat: Double,
    val lng: Double,
    val latitude: Double,
    val longitude: Double,
    @SerialName("reported_at") val reportedAt: String = "",
    @SerialName("received_at") val receivedAt: String = "",
    @SerialName("stale_after_seconds") val staleAfterSeconds: Int = 0,
)

@Serializable
data class SupplierFleetLiveRoute(
    @SerialName("manifest_id") val manifestId: String,
    @SerialName("route_id") val routeId: String,
    @SerialName("driver_id") val driverId: String,
    @SerialName("driver_name") val driverName: String? = null,
    @SerialName("manifest_state") val manifestState: String,
    @SerialName("route_geometry") val routeGeometry: RouteGeometryWire? = null,
    @SerialName("driver_location") val driverLocation: SupplierDriverLocationWire? = null,
    @SerialName("live_location_available") val liveLocationAvailable: Boolean = false,
    @SerialName("location_stale") val locationStale: Boolean? = null,
)

@Serializable
data class SupplierFleetLiveMapResponse(
    val routes: List<SupplierFleetLiveRoute> = emptyList(),
    @SerialName("fetched_at") val fetchedAt: String = "",
)

@Serializable
data class ShopClosedAttemptRow(
    @SerialName("attempt_id") val attemptId: String,
    @SerialName("order_id") val orderId: String,
    @SerialName("driver_id") val driverId: String = "",
    @SerialName("retailer_id") val retailerId: String = "",
    val resolution: String = "",
    @SerialName("shop_closed_reason") val shopClosedReason: String? = null,
    @SerialName("shop_closed_resolution") val shopClosedResolution: String? = null,
    @SerialName("grace_ends_at") val graceEndsAt: String? = null,
    @SerialName("shop_closed_at") val shopClosedAt: String? = null,
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String? = null,
)

@Serializable
data class ShopClosedActiveResponse(
    val data: List<ShopClosedAttemptRow> = emptyList(),
)

@Serializable
data class OrderReceiptMeta(
    @SerialName("receipt_id") val receiptId: String = "",
    @SerialName("html_url") val htmlUrl: String = "",
    @SerialName("pdf_url") val pdfUrl: String = "",
    @SerialName("qr_url") val qrUrl: String = "",
    @SerialName("party_copy") val partyCopy: String = "",
    @SerialName("legal_class") val legalClass: String = "",
    @SerialName("tax_ofd") val taxOfd: Boolean = false,
)

@Serializable
data class ComplianceSummary(
    @SerialName("open_fiscal_count") val openFiscalCount: Int = 0,
    @SerialName("force_complete_count") val forceCompleteCount: Int = 0,
    @SerialName("claim_mismatch_count") val claimMismatchCount: Int = 0,
    @SerialName("credit_freeze_count") val creditFreezeCount: Int = 0,
    @SerialName("generated_at") val generatedAt: String = "",
)

@Serializable
data class ComplianceFiscalOpenRow(
    @SerialName("order_id") val orderId: String,
    @SerialName("retailer_id") val retailerId: String = "",
    @SerialName("driver_id") val driverId: String = "",
    val status: String = "",
    @SerialName("fiscal_status") val fiscalStatus: String = "",
    @SerialName("total_minor") val totalMinor: Long = 0,
    val currency: String = "UZS",
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class ComplianceForceCompleteRow(
    @SerialName("order_id") val orderId: String,
    @SerialName("reason_code") val reasonCode: String = "",
    @SerialName("actor_id") val actorId: String = "",
    @SerialName("total_minor") val totalMinor: Long = 0,
    val currency: String = "UZS",
    @SerialName("completed_at") val completedAt: String = "",
)

@Serializable
data class ComplianceClaimMismatchRow(
    @SerialName("claim_id") val claimId: String,
    @SerialName("order_id") val orderId: String,
    @SerialName("claim_amount_minor") val claimAmountMinor: Long = 0,
    @SerialName("order_total_minor") val orderTotalMinor: Long = 0,
    @SerialName("mismatch_reason") val mismatchReason: String = "",
    val currency: String = "UZS",
)

@Serializable
data class ComplianceCreditFreezeRow(
    @SerialName("retailer_id") val retailerId: String,
    val status: String = "",
    @SerialName("credit_limit_minor") val creditLimitMinor: Long = 0,
    @SerialName("current_balance_minor") val currentBalanceMinor: Long = 0,
    @SerialName("available_credit_minor") val availableCreditMinor: Long = 0,
)

@Serializable
data class ComplianceDashboardResponse(
    val summary: ComplianceSummary = ComplianceSummary(),
    @SerialName("open_fiscal") val openFiscal: List<ComplianceFiscalOpenRow> = emptyList(),
    @SerialName("force_completes") val forceCompletes: List<ComplianceForceCompleteRow> = emptyList(),
    @SerialName("claim_mismatches") val claimMismatches: List<ComplianceClaimMismatchRow> = emptyList(),
    @SerialName("credit_freezes") val creditFreezes: List<ComplianceCreditFreezeRow> = emptyList(),
)

@Serializable
data class NegotiationProposalItem(
    @SerialName("sku_id") val skuId: String,
    @SerialName("original_qty") val originalQty: Int = 0,
    @SerialName("proposed_qty") val proposedQty: Int = 0,
)

@Serializable
data class NegotiationProposalRow(
    @SerialName("proposal_id") val proposalId: String,
    @SerialName("order_id") val orderId: String,
    @SerialName("driver_id") val driverId: String = "",
    val items: List<NegotiationProposalItem> = emptyList(),
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class NegotiationPendingResponse(
    val data: List<NegotiationProposalRow> = emptyList(),
)

@Serializable
data class ShopClosedResolveRequest(
    @SerialName("attempt_id") val attemptId: String,
    val action: String,
)

@Serializable
data class NegotiationResolveRequest(
    @SerialName("proposal_id") val proposalId: String,
    val action: String,
    val resolution: String? = null,
)

@Serializable
data class NegotiationResolveResponse(
    val status: String = "",
    @SerialName("proposal_id") val proposalId: String = "",
    @SerialName("order_id") val orderId: String = "",
)

@Serializable
data class SupplierReplenishmentTriggerResponse(
    val status: String = "",
    @SerialName("request_id") val requestId: String = "",
    @SerialName("warehouse_id") val warehouseId: String = "",
)

@Serializable
data class SupplierAnalyticsVelocityPoint(
    val date: String = "",
    @SerialName("orders_created") val ordersCreated: Int = 0,
    @SerialName("orders_completed") val ordersCompleted: Int = 0,
)

@Serializable
data class SupplierAnalyticsVelocityResponse(
    @SerialName("period_days") val periodDays: Int = 0,
    val points: List<SupplierAnalyticsVelocityPoint> = emptyList(),
    @SerialName("generated_at") val generatedAt: String = "",
)

@Serializable
data class SupplierAnalyticsRevenuePoint(
    val date: String = "",
    @SerialName("revenue_minor") val revenueMinor: Long = 0,
)

@Serializable
data class SupplierAnalyticsRevenueResponse(
    val currency: String = "",
    @SerialName("total_minor") val totalMinor: Long = 0,
    val series: List<SupplierAnalyticsRevenuePoint> = emptyList(),
    @SerialName("generated_at") val generatedAt: String = "",
)

@Serializable
data class SupplierDemandSummaryResponse(
    @SerialName("total_retailers") val totalRetailers: Int = 0,
    @SerialName("total_pallets") val totalPallets: Int = 0,
    @SerialName("total_value") val totalValue: Long = 0,
    @SerialName("prediction_count") val predictionCount: Int = 0,
    @SerialName("generated_at") val generatedAt: String = "",
    @SerialName("baseline_source") val baselineSource: String? = null,
    @SerialName("granularity") val granularity: String? = null,
    val confidence: ForecastConfidence? = null,
)

@Serializable
data class SupplierEmpathyAdoption(
    @SerialName("total_predictions") val totalPredictions: Long = 0,
    @SerialName("predictions_dormant") val predictionsDormant: Long = 0,
    @SerialName("predictions_waiting") val predictionsWaiting: Long = 0,
    @SerialName("predictions_fired") val predictionsFired: Long = 0,
    @SerialName("predictions_rejected") val predictionsRejected: Long = 0,
)

@Serializable
data class SupplierBroadcastRequest(
    val title: String,
    val body: String,
    val role: String = "ALL",
)

@Serializable
data class SupplierBroadcastResponse(
    val status: String = "",
    @SerialName("supplier_id") val supplierId: String = "",
)

@Serializable
data class ExceptionMapCell(
    @SerialName("h3_cell") val h3Cell: String = "",
    val lat: Double = 0.0,
    val lng: Double = 0.0,
    val severity: String = "low",
    val counts: Map<String, Int> = emptyMap(),
    @SerialName("sample_order_ids") val sampleOrderIds: List<String> = emptyList(),
    @SerialName("deep_link") val deepLink: String = "",
)

@Serializable
data class ExceptionMapResponse(
    val cells: List<ExceptionMapCell> = emptyList(),
    @SerialName("window_hours") val windowHours: Int = 24,
)

@Serializable
data class RetailerOverridePreview(
    @SerialName("retailers_on_sku_count") val retailersOnSkuCount: Int = 0,
    @SerialName("active_override_count") val activeOverrideCount: Int = 0,
    @SerialName("catalog_list_price") val catalogListPrice: Long = 0,
    @SerialName("margin_delta_per_unit") val marginDeltaPerUnit: Long = 0,
    @SerialName("margin_estimate_label") val marginEstimateLabel: String = "",
    @SerialName("affected_retailer_ids") val affectedRetailerIds: List<String> = emptyList(),
)

@Serializable
data class RetailerOverridePreviewRequest(
    @SerialName("retailer_id") val retailerId: String? = null,
    @SerialName("product_id") val productId: String? = null,
    @SerialName("sku_id") val skuId: String? = null,
    @SerialName("proposed_price") val proposedPrice: Long,
)

data class SupplierBroadcastTemplate(
    val id: String,
    val title: String,
    val body: String,
    val defaultRole: String,
)

val SUPPLIER_BROADCAST_TEMPLATES = listOf(
    SupplierBroadcastTemplate("storm_delay", "Delivery delay notice", "Due to weather conditions, deliveries may be delayed on {date}. We will update routes as conditions improve.", "RETAILER"),
    SupplierBroadcastTemplate("holiday_hours", "Holiday receiving hours", "Our network will operate on reduced hours on {date}. Please confirm your receiving window in the app.", "RETAILER"),
    SupplierBroadcastTemplate("fee_notice", "Service fee update", "A service fee adjustment takes effect on {date}. Review your latest invoices for details.", "RETAILER"),
    SupplierBroadcastTemplate("yard_hold", "Yard congestion advisory", "Loading bay congestion reported. Drivers: expect queue delays at warehouse check-in.", "DRIVER"),
)

@Serializable
data class PaymentBypassRequest(
    @SerialName("order_id") val orderId: String,
    val reason: String = "",
)

@Serializable
data class PaymentBypassResponse(
    val status: String = "",
    @SerialName("bypass_token") val bypassToken: String = "",
    @SerialName("order_id") val orderId: String = "",
)

@Serializable
data class PaymentLedgerEntry(
    @SerialName("ledger_entry_id") val ledgerEntryId: String = "",
    @SerialName("order_id") val orderId: String? = null,
    @SerialName("supplier_id") val supplierId: String? = null,
    @SerialName("retailer_id") val retailerId: String? = null,
    val gateway: String = "",
    @SerialName("entry_type") val entryType: String = "",
    @SerialName("amount_minor") val amountMinor: Long = 0,
    val currency: String = "",
    @SerialName("reference_id") val referenceId: String? = null,
    val source: String? = null,
    @SerialName("occurred_at") val occurredAt: String = "",
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class PaymentLedgerResponse(
    val items: List<PaymentLedgerEntry> = emptyList(),
    val count: Int = 0,
    val limit: Int = 0,
    @SerialName("supplier_id") val supplierId: String = "",
)

// Logistics claims (GET /v1/supplier/claims, approve/reject, claim-chargebacks)

@Serializable
data class SupplierClaimLine(
    val sku: String = "",
    val quantity: Long = 0,
    val reason: String? = null,
    @SerialName("unit_price_minor") val unitPriceMinor: Long? = null,
    @SerialName("amount_minor") val amountMinor: Long? = null,
)

@Serializable
data class SupplierClaimEvidence(
    @SerialName("evidence_type") val evidenceType: String = "",
    val uri: String = "",
)

@Serializable
data class SupplierClaim(
    @SerialName("claim_id") val claimId: String = "",
    @SerialName("order_id") val orderId: String = "",
    @SerialName("retailer_id") val retailerId: String = "",
    @SerialName("claim_type") val claimType: String = "",
    val status: String = "",
    @SerialName("amount_minor") val amountMinor: Long? = null,
    val currency: String? = null,
    val description: String? = null,
    @SerialName("line_items") val lineItems: List<SupplierClaimLine> = emptyList(),
    val evidences: List<SupplierClaimEvidence> = emptyList(),
    @SerialName("created_at") val createdAt: String = "",
)

@Serializable
data class SupplierClaimsListResponse(
    val claims: List<SupplierClaim> = emptyList(),
)

@Serializable
data class ApproveClaimRequest(
    @SerialName("resolution_note") val resolutionNote: String = "",
    @SerialName("settlement_mode") val settlementMode: String = "LEDGER_ONLY",
    @SerialName("skip_gateway_refund") val skipGatewayRefund: Boolean = true,
)

@Serializable
data class RejectClaimRequest(
    @SerialName("resolution_note") val resolutionNote: String = "",
)

@Serializable
data class ClaimSettlementResult(
    @SerialName("chargeback_id") val chargebackId: String? = null,
    @SerialName("amount_minor") val amountMinor: Long = 0,
    val mode: String = "",
    @SerialName("gateway_refunded") val gatewayRefunded: Boolean = false,
)

@Serializable
data class ApproveClaimResponse(
    val claim: SupplierClaim? = null,
    val settlement: ClaimSettlementResult? = null,
)

@Serializable
data class ClaimChargebacksResponse(
    val items: List<PaymentLedgerEntry> = emptyList(),
    val count: Int = 0,
    val limit: Int = 0,
    @SerialName("supplier_id") val supplierId: String = "",
)

@Serializable
data class FleetDriverCreateRequest(
    val name: String,
    val phone: String,
    val pin: String,
    @SerialName("home_node_type") val homeNodeType: String,
    @SerialName("home_node_id") val homeNodeId: String,
    @SerialName("vehicle_id") val vehicleId: String? = null,
    @SerialName("is_active") val isActive: Boolean? = null,
)

@Serializable
data class FleetVehicleCreateRequest(
    val label: String? = null,
    @SerialName("license_plate") val licensePlate: String,
    @SerialName("home_node_type") val homeNodeType: String,
    @SerialName("home_node_id") val homeNodeId: String,
    @SerialName("is_active") val isActive: Boolean? = null,
)

@Serializable
data class SupplierActivityEvent(
    val id: String = "",
    val type: String = "",
    val timestamp: String = "",
    val description: String = "",
    @SerialName("order_id") val orderId: String? = null,
    @SerialName("manifest_id") val manifestId: String? = null,
)

@Serializable
data class SupplierActivityResponse(
    val events: List<SupplierActivityEvent> = emptyList(),
)

@Serializable
data class SettlementCurrencyTotal(
    val currency: String = "",
    @SerialName("entry_count") val entryCount: Int = 0,
    @SerialName("amount_minor_total") val amountMinorTotal: Long = 0,
)

@Serializable
data class SettlementAuthorityRow(
    val gateway: String = "",
    @SerialName("entry_type") val entryType: String = "",
    val currency: String = "",
    @SerialName("entry_count") val entryCount: Long = 0,
    @SerialName("amount_minor_total") val amountMinorTotal: Long = 0,
    @SerialName("first_occurred_at") val firstOccurredAt: String = "",
    @SerialName("last_occurred_at") val lastOccurredAt: String = "",
)

@Serializable
data class SettlementAuthorityResponse(
    val items: List<SettlementAuthorityRow> = emptyList(),
    val count: Int = 0,
    @SerialName("group_limit") val groupLimit: Int = 0,
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("entry_count_total") val entryCountTotal: Int = 0,
    @SerialName("totals_by_currency") val totalsByCurrency: List<SettlementCurrencyTotal> = emptyList(),
)

@Serializable
data class ReconciliationMismatchRow(
    val gateway: String = "",
    val currency: String = "",
    @SerialName("net_amount_minor") val netAmountMinor: Long = 0,
    @SerialName("credit_amount_minor_total") val creditAmountMinorTotal: Long = 0,
    @SerialName("debit_amount_minor_total") val debitAmountMinorTotal: Long = 0,
    @SerialName("entry_count_total") val entryCountTotal: Long = 0,
    @SerialName("last_occurred_at") val lastOccurredAt: String = "",
)

@Serializable
data class SupplierAIRecommendationEvidence(
    val label: String = "",
    val value: String = "",
    val href: String? = null,
)

@Serializable
data class SupplierAIRecommendation(
    @SerialName("recommendation_id") val recommendationId: String = "",
    @SerialName("aggregate_id") val aggregateId: String = "",
    @SerialName("aggregate_type") val aggregateType: String = "",
    val action: String = "",
    val status: String = "",
    val score: Double = 0.0,
    val confidence: Double = 0.0,
    val source: String = "",
    val explanation: String = "",
    @SerialName("reason_codes") val reasonCodes: List<String> = emptyList(),
    val evidence: List<SupplierAIRecommendationEvidence> = emptyList(),
    val decision: String? = null,
    @SerialName("decision_note") val decisionNote: String? = null,
    @SerialName("decided_by") val decidedBy: String? = null,
    @SerialName("decided_at") val decidedAt: String? = null,
    @SerialName("generated_at") val generatedAt: String = "",
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class SupplierAIRecommendationsResponse(
    val items: List<SupplierAIRecommendation> = emptyList(),
    val count: Int = 0,
    val limit: Int = 0,
    val status: String? = null,
    @SerialName("updated_at") val updatedAt: String = "",
)

@Serializable
data class SupplierAIRecommendationDecisionRequest(
    @SerialName("recommendation_id") val recommendationId: String,
    val decision: String,
    val note: String? = null,
)

@Serializable
data class SupplierAIRecommendationDecisionResponse(
    val recommendation: SupplierAIRecommendation,
)

@Serializable
data class ReconciliationMismatchResponse(
    val items: List<ReconciliationMismatchRow> = emptyList(),
    val count: Int = 0,
    @SerialName("mismatch_threshold_minor") val mismatchThresholdMinor: Long = 0,
)

@Serializable
data class SupplierOrgMember(
    @SerialName("user_id") val userId: String,
    val name: String = "",
    val phone: String = "",
    @SerialName("supplier_role") val supplierRole: String = "",
    @SerialName("assigned_warehouse_id") val assignedWarehouseId: String? = null,
    @SerialName("assigned_factory_id") val assignedFactoryId: String? = null,
    @SerialName("is_active") val isActive: Boolean = true,
)

@Serializable
data class SupplierOrgMembersResponse(
    @SerialName("supplier_id") val supplierId: String = "",
    val items: List<SupplierOrgMember> = emptyList(),
)

@Serializable
data class SupplierOrgMemberCreateRequest(
    val name: String,
    val email: String? = null,
    val phone: String,
    val password: String,
    @SerialName("supplier_role") val supplierRole: String,
    @SerialName("assigned_warehouse_id") val assignedWarehouseId: String? = null,
    @SerialName("assigned_factory_id") val assignedFactoryId: String? = null,
)

@Serializable
data class SupplierOrgMemberUpdateRequest(
    val name: String? = null,
    @SerialName("supplier_role") val supplierRole: String? = null,
    @SerialName("assigned_warehouse_id") val assignedWarehouseId: String? = null,
    @SerialName("assigned_factory_id") val assignedFactoryId: String? = null,
    @SerialName("is_active") val isActive: Boolean? = null,
)

@Serializable
data class ApproveEarlyCompleteRequest(
    @SerialName("driver_id") val driverId: String,
)

@Serializable
data class DemandHistoryPoint(
    val date: String = "",
    val predicted: Long = 0,
    val actual: Long = 0,
    @SerialName("predicted_qty") val predictedQty: Long = 0,
    @SerialName("actual_qty") val actualQty: Long = 0,
)

@Serializable
data class DemandUpcomingRow(
    val date: String = "",
    @SerialName("retailer_name") val retailerName: String = "",
    @SerialName("sku_id") val skuId: String = "",
    @SerialName("product_name") val productName: String = "",
    @SerialName("predicted_qty") val predictedQty: Long = 0,
)

@Serializable
data class DemandHistoryResponse(
    @SerialName("time_series") val timeSeries: List<DemandHistoryPoint> = emptyList(),
    val upcoming: List<DemandUpcomingRow> = emptyList(),
)

@Serializable
data class ImportSessionCreateRequest(
    @SerialName("file_name") val fileName: String,
    @SerialName("file_size_bytes") val fileSizeBytes: Int,
)

@Serializable
data class ImportSessionCreateResponse(
    @SerialName("session_id") val sessionId: String = "",
    val status: String = "",
)

@Serializable
data class SupplierMEIONetworkSummary(
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("warehouses_scanned") val warehousesScanned: Int = 0,
    @SerialName("skus_analyzed") val skusAnalyzed: Int = 0,
    @SerialName("insights_generated") val insightsGenerated: Int = 0,
    @SerialName("transfer_recommendations") val transferRecommendations: Int = 0,
    @SerialName("generated_at") val generatedAt: String = "",
)

@Serializable
data class ControlTowerZoneOverride(
    @SerialName("override_id") val overrideId: String = "",
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("warehouse_id") val warehouseId: String? = null,
    val action: String = "",
    @SerialName("ttl_expires_at") val ttlExpiresAt: String = "",
    @SerialName("is_active") val isActive: Boolean = true,
)

@Serializable
data class ControlTowerZoneOverridesResponse(
    val overrides: List<ControlTowerZoneOverride> = emptyList(),
)

@Serializable
data class PlanningSAndOPSnapshot(
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("horizon_days") val horizonDays: Int = 7,
    @SerialName("factory_capacity_units") val factoryCapacityUnits: Long = 0,
    @SerialName("warehouse_inbound_cap_units") val warehouseInboundCapUnits: Long = 0,
    @SerialName("warehouse_outbound_cap_units") val warehouseOutboundCapUnits: Long = 0,
    @SerialName("utilization_pct") val utilizationPct: Double = 0.0,
    @SerialName("capacity_alert") val capacityAlert: Boolean = false,
)

@Serializable
data class PlanningScenarioInput(
    @SerialName("factory_downtime_hours") val factoryDowntimeHours: Int = 0,
    @SerialName("demand_delta_pct") val demandDeltaPct: Double = 0.0,
    @SerialName("horizon_days") val horizonDays: Int = 7,
)

@Serializable
data class PlanningScenarioResult(
    @SerialName("scenario_id") val scenarioId: String = "",
    @SerialName("sla_risk_pct") val slaRiskPct: Double = 0.0,
    @SerialName("fleet_volume_orders") val fleetVolumeOrders: Long = 0,
    @SerialName("stockout_skus") val stockoutSkus: List<String> = emptyList(),
    @SerialName("capacity_breach") val capacityBreach: Boolean = false,
)

@Serializable
data class PromoSimulateInput(
    @SerialName("promotion_id") val promotionId: String? = null,
    @SerialName("discount_pct") val discountPct: Double? = null,
    @SerialName("expected_units") val expectedUnits: Long? = null,
    @SerialName("avg_unit_margin_minor") val avgUnitMarginMinor: Long? = null,
)

@Serializable
data class PromoSimulateResult(
    @SerialName("simulation_id") val simulationId: String = "",
    @SerialName("promotion_id") val promotionId: String? = null,
    @SerialName("projected_volume") val projectedVolume: Long = 0,
    @SerialName("projected_revenue_minor") val projectedRevenueMinor: Long = 0,
    @SerialName("projected_margin_minor") val projectedMarginMinor: Long = 0,
    @SerialName("margin_delta_pct") val marginDeltaPct: Double = 0.0,
    @SerialName("sandbox_only") val sandboxOnly: Boolean = true,
)

@Serializable
data class ForecastConfidence(
    @SerialName("low_units") val lowUnits: Long? = null,
    @SerialName("high_units") val highUnits: Long? = null,
    @SerialName("confidence_pct") val confidencePct: Int? = null,
    @SerialName("baseline_source") val baselineSource: String? = null,
    @SerialName("blocked_reason") val blockedReason: String? = null,
    val label: String? = null,
)

@Serializable
data class SeasonalOverrideInput(
    @SerialName("template_id") val templateId: String? = null,
    @SerialName("start_date") val startDate: String,
    @SerialName("end_date") val endDate: String,
    val name: String? = null,
)

@Serializable
data class SeasonalOverrideRow(
    @SerialName("override_id") val overrideId: String = "",
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("template_id") val templateId: String = "",
    val name: String? = null,
    @SerialName("start_date") val startDate: String = "",
    @SerialName("end_date") val endDate: String = "",
    @SerialName("is_active") val isActive: Boolean = true,
)

@Serializable
data class SeasonalBuiltinTemplate(
    val id: String = "",
    val name: String = "",
)

@Serializable
data class SeasonalTemplatesResponse(
    @SerialName("builtin_templates") val builtinTemplates: List<SeasonalBuiltinTemplate> = emptyList(),
    val overrides: List<SeasonalOverrideRow> = emptyList(),
)

/** GET/PUT /v1/supplier/return-policy */
@Serializable
data class SupplierReturnPolicy(
    @SerialName("default_window_hours") val defaultWindowHours: Long = 48,
    @SerialName("concealed_damage_window_hours") val concealedDamageWindowHours: Long? = null,
    @SerialName("require_photo") val requirePhoto: Boolean = true,
    @SerialName("allow_expired_claims") val allowExpiredClaims: Boolean = false,
    @SerialName("policy_source_hint") val policySourceHint: String? = null,
)

@Serializable
data class KnowledgeGraphNode(
    val id: String = "",
    val type: String = "",
    val name: String? = null,
)

@Serializable
data class KnowledgeGraphEdge(
    val from: String = "",
    val to: String = "",
    val relation: String = "",
)

@Serializable
data class SupplierKnowledgeGraph(
    @SerialName("supplier_id") val supplierId: String = "",
    val nodes: List<KnowledgeGraphNode> = emptyList(),
    val edges: List<KnowledgeGraphEdge> = emptyList(),
)

@Serializable
data class SupplierReplenishmentPolicy(
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("auto_approve_stable") val autoApproveStable: Boolean = false,
    @SerialName("auto_approve_predictive_push") val autoApprovePredictivePush: Boolean = false,
    @SerialName("max_daily_transfer_units") val maxDailyTransferUnits: Long = 0,
    @SerialName("min_confidence_score") val minConfidenceScore: Double = 0.0,
)

@Serializable
data class GeoJSONPolygonPayload(
    val type: String = "Polygon",
    val coordinates: List<List<List<Double>>>,
)

@Serializable
data class ControlTowerZoneOverrideCreateRequest(
    val action: String,
    @SerialName("ttl_seconds") val ttlSeconds: Int = 1800,
    @SerialName("polygon_geojson") val polygonGeojson: GeoJSONPolygonPayload,
)

@Serializable
data class CashReconciliationRow(
    @SerialName("reconciliation_id") val reconciliationId: String = "",
    @SerialName("driver_id") val driverId: String = "",
    val status: String = "",
    @SerialName("difference_minor") val differenceMinor: Long = 0,
)

@Serializable
data class CashReconciliationsResponse(
    val reconciliations: List<CashReconciliationRow> = emptyList(),
)

@Serializable
data class CreditNoteRow(
    @SerialName("credit_note_id") val creditNoteId: String = "",
    @SerialName("order_id") val orderId: String = "",
    val status: String = "",
    @SerialName("total_gross_minor") val totalGrossMinor: Long = 0,
)

@Serializable
data class CreditNotesResponse(
    @SerialName("credit_notes") val creditNotes: List<CreditNoteRow> = emptyList(),
)

@Serializable
data class CreditProfileRow(
    @SerialName("retailer_id") val retailerId: String = "",
    @SerialName("supplier_id") val supplierId: String = "",
    @SerialName("credit_limit_minor") val creditLimitMinor: Long = 0,
    @SerialName("current_balance_minor") val currentBalanceMinor: Long = 0,
    @SerialName("available_credit_minor") val availableCreditMinor: Long = 0,
    @SerialName("risk_score") val riskScore: Long = 0,
    @SerialName("risk_tier") val riskTier: String = "",
    @SerialName("delinquency_count") val delinquencyCount: Long = 0,
    val status: String = "",
    @SerialName("utilization_bps") val utilizationBps: Long? = null,
    @SerialName("needs_attention") val needsAttention: Boolean = false,
)

@Serializable
data class CreditProfilesResponse(
    val profiles: List<CreditProfileRow> = emptyList(),
    val count: Int = 0,
)

@Serializable
data class RetailerCreditProfilePatchRequest(
    @SerialName("retailer_id") val retailerId: String,
    @SerialName("credit_limit_minor") val creditLimitMinor: Long,
    val status: String,
    val reason: String = "collections_desk",
)

@Serializable
data class RoutePerformanceRow(
    @SerialName("route_id") val routeId: String = "",
    @SerialName("driver_id") val driverId: String = "",
    @SerialName("orders_completed") val ordersCompleted: Int = 0,
)

@Serializable
data class RoutePerformanceResponse(
    val routes: List<RoutePerformanceRow> = emptyList(),
)

@Serializable
data class NotificationPreferenceRow(
    @SerialName("event_type") val eventType: String = "",
    val channel: String = "",
    val enabled: Boolean = true,
    @SerialName("quiet_from") val quietFrom: String? = null,
    @SerialName("quiet_to") val quietTo: String? = null,
)

@Serializable
data class NotificationPreferencesResponse(
    val preferences: List<NotificationPreferenceRow> = emptyList(),
)

@Serializable
data class NotificationPreferencesPatchRequest(
    val preferences: List<NotificationPreferenceRow> = emptyList(),
)

@Serializable
data class StatusResponse(
    val status: String = "",
)
