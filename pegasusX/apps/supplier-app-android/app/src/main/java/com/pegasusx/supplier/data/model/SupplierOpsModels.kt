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
data class SupplierDispatchPreview(
    @SerialName("undispatched_orders") val undispatchedOrders: List<JsonElement> = emptyList(),
    @SerialName("available_drivers") val availableDrivers: List<JsonElement> = emptyList(),
    @SerialName("unavailable_drivers") val unavailableDrivers: List<JsonElement> = emptyList(),
    @SerialName("pending_count") val pendingCount: Int = 0,
    @SerialName("available_driver_count") val availableDriverCount: Int = 0,
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
data class SupplierTopologyWarehouse(
    @SerialName("warehouse_id") val warehouseId: String,
    val name: String = "",
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
    val lat: Double = 0.0,
    val lng: Double = 0.0,
    @SerialName("is_active") val isActive: Boolean = true,
)

@Serializable
data class SupplierTopologyWarehouseInput(
    @SerialName("warehouse_id") val warehouseId: String? = null,
    val name: String,
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
    @SerialName("created_at") val createdAt: String = "",
    @SerialName("updated_at") val updatedAt: String? = null,
)

@Serializable
data class ShopClosedActiveResponse(
    val data: List<ShopClosedAttemptRow> = emptyList(),
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
data class PaymentLedgerEntry(
    @SerialName("ledger_entry_id") val ledgerEntryId: String,
    @SerialName("order_id") val orderId: String? = null,
    @SerialName("supplier_id") val supplierId: String? = null,
    @SerialName("retailer_id") val retailerId: String? = null,
    val gateway: String = "",
    @SerialName("entry_type") val entryType: String = "",
    @SerialName("amount_minor") val amountMinor: Long = 0,
    val currency: String = "",
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
