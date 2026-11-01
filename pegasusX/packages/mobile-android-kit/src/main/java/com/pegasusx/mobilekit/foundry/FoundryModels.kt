package com.pegasusx.mobilekit.foundry

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.JsonElement

// ── Role 7: Cashier & AML Vault Models ──

@Serializable
data class StartCashierSessionRequest(
    @SerialName("cell_id") val cellId: String = "cell-uz",
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("depot_id") val depotId: String,
    @SerialName("driver_id") val driverId: String,
    @SerialName("route_id") val routeId: String,
    @SerialName("bag_seal_barcode") val bagSealBarcode: String,
    @SerialName("expected_cash_minor") val expectedCashMinor: Long,
)

@Serializable
data class CashierSessionView(
    @SerialName("session_id") val sessionId: String,
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("depot_id") val depotId: String,
    @SerialName("driver_id") val driverId: String,
    @SerialName("route_id") val routeId: String,
    @SerialName("bag_seal_barcode") val bagSealBarcode: String,
    @SerialName("attempt_number") val attemptNumber: Long,
    @SerialName("counted_cash_minor") val countedCashMinor: Long,
    @SerialName("expected_cash_minor") val expectedCashMinor: Long? = null,
    @SerialName("variance_minor") val varianceMinor: Long? = null,
    @SerialName("status") val status: String,
    @SerialName("clearance_token_hash") val clearanceTokenHash: String? = null,
    @SerialName("micro_skimming_flagged") val microSkimmingFlagged: Boolean = false,
    @SerialName("created_at") val createdAt: String,
    @SerialName("resolved_at") val resolvedAt: String? = null,
)

@Serializable
data class BlindCountRequest(
    @SerialName("denominations_uzs") val denominationsUzs: Map<String, Long>,
    @SerialName("tolerance_threshold_minor") val toleranceThresholdMinor: Long = 500000L,
)

@Serializable
data class RRNTenderReclassificationRequest(
    @SerialName("session_id") val sessionId: String,
    @SerialName("order_id") val orderId: String,
    @SerialName("disputed_cash_minor") val disputedCashMinor: Long,
    @SerialName("gateway_rrn") val gatewayRrn: String,
    @SerialName("terminal_id") val terminalId: String,
    @SerialName("card_last_four") val cardLastFour: String,
    @SerialName("auth_code") val authCode: String,
    @SerialName("payment_network") val paymentNetwork: String,
    @SerialName("rrn_captured_at") val rrnCapturedAt: String,
    @SerialName("pod_geofence_timestamp") val podGeofenceTimestamp: String,
)

@Serializable
data class CITHandoverRequest(
    @SerialName("depot_id") val depotId: String,
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("session_ids") val sessionIds: List<String>,
    @SerialName("courier_badge_id") val courierBadgeId: String,
    @SerialName("cit_seal_barcode") val citSealBarcode: String,
    @SerialName("cash_minor") val cashMinor: Long,
)

@Serializable
data class DepotVaultLedgerEntry(
    @SerialName("vault_entry_id") val vaultEntryId: String,
    @SerialName("depot_id") val depotId: String,
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("session_id") val sessionId: String? = null,
    @SerialName("cit_manifest_id") val citManifestId: String? = null,
    @SerialName("entry_type") val entryType: String,
    @SerialName("amount_minor") val amountMinor: Long,
    @SerialName("running_balance_minor") val runningBalanceMinor: Long,
    @SerialName("courier_badge_id") val courierBadgeId: String? = null,
    @SerialName("seal_barcode") val sealBarcode: String? = null,
    @SerialName("created_at") val createdAt: String,
)

@Serializable
data class AMLCheckRequest(
    @SerialName("legal_entity_tin") val legalEntityTin: String,
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("calendar_date") val calendarDate: String,
    @SerialName("cash_minor") val cashMinor: Long,
)

@Serializable
data class AMLCheckResponse(
    @SerialName("legal_entity_tin") val legalEntityTin: String,
    @SerialName("accumulated_cash_minor") val accumulatedCashMinor: Long,
    @SerialName("ceiling_minor") val ceilingMinor: Long,
)

// ── Role 9: Fleet, Yard & Maintenance Models ──

@Serializable
data class DVIRChecklist8Point(
    @SerialName("steering_and_horn_ok") val steeringAndHornOk: Boolean = true,
    @SerialName("service_and_parking_brakes_ok") val serviceAndParkingBrakesOk: Boolean = true,
    @SerialName("tires_tread_and_pressure_ok") val tiresTreadAndPressureOk: Boolean = true,
    @SerialName("lights_and_hazards_ok") val lightsAndHazardsOk: Boolean = true,
    @SerialName("wipers_and_fluids_ok") val wipersAndFluidsOk: Boolean = true,
    @SerialName("emergency_kit_and_extinguisher_ok") val emergencyKitAndExtinguisherOk: Boolean = true,
    @SerialName("cargo_box_sanitation_ok") val cargoBoxSanitationOk: Boolean = true,
    @SerialName("reefer_temp_calibrated_ok") val reeferTempCalibratedOk: Boolean = true,
)

@Serializable
data class FleetDVIRInspection(
    @SerialName("inspection_id") val inspectionId: String? = null,
    @SerialName("cell_id") val cellId: String = "cell-uz",
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("vehicle_id") val vehicleId: String,
    @SerialName("driver_id") val driverId: String,
    @SerialName("inspection_type") val inspectionType: String = "PRE_TRIP",
    @SerialName("requires_cold_chain") val requiresColdChain: Boolean = false,
    @SerialName("checklist") val checklist: DVIRChecklist8Point = DVIRChecklist8Point(),
    @SerialName("failed_points") val failedPoints: List<String> = emptyList(),
    @SerialName("critical_defect_found") val criticalDefectFound: Boolean = false,
    @SerialName("red_tag_applied") val redTagApplied: Boolean = false,
    @SerialName("odometer_km") val odometerKm: Long = 0L,
    @SerialName("reefer_temp_celsius") val reeferTempCelsius: Double = 0.0,
    @SerialName("status") val status: String? = null,
    @SerialName("maintenance_work_order_id") val maintenanceWorkOrderId: String? = null,
    @SerialName("preventative_service_due") val preventativeServiceDue: Boolean = false,
    @SerialName("created_at") val createdAt: String? = null,
)

@Serializable
data class YardGateCheckEvent(
    @SerialName("gate_event_id") val gateEventId: String? = null,
    @SerialName("depot_id") val depotId: String,
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("vehicle_id") val vehicleId: String,
    @SerialName("license_plate") val licensePlate: String,
    @SerialName("direction") val direction: String,
    @SerialName("vehicle_purpose") val vehiclePurpose: String,
    @SerialName("bolt_seal_barcode") val boltSealBarcode: String? = null,
    @SerialName("seal_intact") val sealIntact: Boolean = true,
    @SerialName("assigned_bay_slot") val assignedBaySlot: String? = null,
    @SerialName("created_at") val createdAt: String? = null,
)

@Serializable
data class PredictiveReeferRequest(
    @SerialName("vehicle_id") val vehicleId: String,
    @SerialName("cooling_rate_deg_per_min") val coolingRateDegPerMin: Double,
    @SerialName("min_required_rate_deg_per_min") val minRequiredRateDegPerMin: Double = 0.25,
)

@Serializable
data class PredictiveReeferResponse(
    @SerialName("vehicle_id") val vehicleId: String,
    @SerialName("pre_failure_maintenance_hold") val preFailureMaintenanceHold: Boolean,
)

@Serializable
data class DispatchCheckResponse(
    @SerialName("vehicle_id") val vehicleId: String,
    @SerialName("dispatch_authorized") val dispatchAuthorized: Boolean,
)

@Serializable
data class RescueToteItem(
    @SerialName("tote_id") val toteId: String,
    @SerialName("stop_sequence") val stopSequence: Long,
    @SerialName("is_cold_chain") val isColdChain: Boolean = false,
    @SerialName("expected_cash_minor") val expectedCashMinor: Long = 0L,
    @SerialName("thermal_excursion_minutes") val thermalExcursionMinutes: Long = 0L,
    @SerialName("max_allowed_excursion_mins") val maxAllowedExcursionMins: Long = 30L,
)

@Serializable
data class MidRouteRescueRequest(
    @SerialName("cell_id") val cellId: String = "cell-uz",
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("original_route_id") val originalRouteId: String,
    @SerialName("broken_vehicle_id") val brokenVehicleId: String,
    @SerialName("original_driver_id") val originalDriverId: String,
    @SerialName("rescue_vehicle_id") val rescueVehicleId: String,
    @SerialName("rescue_driver_id") val rescueDriverId: String,
    @SerialName("last_completed_stop_seq") val lastCompletedStopSeq: Long,
    @SerialName("new_rescue_bolt_seal") val newRescueBoltSeal: String,
    @SerialName("totes") val totes: List<RescueToteItem>,
)

@Serializable
data class MidRouteRescueResult(
    @SerialName("rescue_route_id") val rescueRouteId: String,
    @SerialName("original_driver_liability_minor") val originalDriverLiabilityMinor: Long,
    @SerialName("rescue_driver_liability_minor") val rescueDriverLiabilityMinor: Long,
    @SerialName("transferred_tote_ids") val transferredToteIds: List<String>,
    @SerialName("quarantined_spoiled_tote_ids") val quarantinedSpoiledToteIds: List<String>,
    @SerialName("new_rescue_bolt_seal") val newRescueBoltSeal: String,
)

// ── Foundry Triad Models ──

@Serializable
data class SearchAroundRequest(
    @SerialName("cell_id") val cellId: String = "cell-uz",
    @SerialName("start_object_type") val startObjectType: String,
    @SerialName("start_object_ids") val startObjectIds: List<String>,
    @SerialName("allowed_link_types") val allowedLinkTypes: List<String> = emptyList(),
    @SerialName("max_hops") val maxHops: Int = 3,
)

@Serializable
data class SearchAroundResponse(
    @SerialName("objects") val objects: List<OntologyObjectDto>,
)

@Serializable
data class OntologyObjectDto(
    @SerialName("cell_id") val cellId: String,
    @SerialName("object_type") val objectType: String,
    @SerialName("object_id") val objectId: String,
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("version") val version: Long,
    @SerialName("properties") val properties: Map<String, JsonElement> = emptyMap(),
    @SerialName("security_markings") val securityMarkings: List<String> = emptyList(),
    @SerialName("valid_from") val validFrom: String? = null,
    @SerialName("updated_at") val updatedAt: String? = null,
)

@Serializable
data class LotRecallRequest(
    @SerialName("cell_id") val cellId: String = "cell-uz",
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("raw_material_lot_id") val rawMaterialLotId: String,
)

@Serializable
data class RecallBlastRadiusReport(
    @SerialName("cell_id") val cellId: String,
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("contaminated_raw_material_lot") val contaminatedRawMaterialLot: String,
    @SerialName("affected_work_orders") val affectedWorkOrders: List<String> = emptyList(),
    @SerialName("affected_finished_lots") val affectedFinishedLots: List<String> = emptyList(),
    @SerialName("quarantined_sscc_pallets") val quarantinedSsccPallets: List<String> = emptyList(),
    @SerialName("quarantined_warehouse_totes") val quarantinedWarehouseTotes: List<String> = emptyList(),
    @SerialName("in_transit_driver_drop_blocks") val inTransitDriverDropBlocks: List<String> = emptyList(),
    @SerialName("retailer_reverse_pickup_orders") val retailerReversePickupOrders: List<String> = emptyList(),
    @SerialName("impacted_soliq_facturas") val impactedSoliqFacturas: List<String> = emptyList(),
)

@Serializable
data class ActionExecutionRequest(
    @SerialName("action_name") val actionName: String,
    @SerialName("action_type") val actionType: String,
    @SerialName("scenario_id") val scenarioId: String? = null,
    @SerialName("actor_id") val actorId: String,
    @SerialName("actor_role") val actorRole: String,
    @SerialName("target_object_type") val targetObjectType: String,
    @SerialName("target_object_id") val targetObjectId: String,
    @SerialName("payload") val payload: Map<String, JsonElement> = emptyMap(),
    @SerialName("idempotency_key") val idempotencyKey: String,
    @SerialName("cell_id") val cellId: String = "cell-uz",
    @SerialName("supplier_id") val supplierId: String,
)

@Serializable
data class ActionExecutionRecord(
    @SerialName("execution_id") val executionId: String,
    @SerialName("action_name") val actionName: String,
    @SerialName("action_type") val actionType: String,
    @SerialName("scenario_id") val scenarioId: String? = null,
    @SerialName("actor_id") val actorId: String,
    @SerialName("actor_role") val actorRole: String,
    @SerialName("target_object_type") val targetObjectType: String,
    @SerialName("target_object_id") val targetObjectId: String,
    @SerialName("idempotency_key") val idempotencyKey: String,
    @SerialName("executed_at") val executedAt: String,
    @SerialName("status") val status: String,
    @SerialName("cell_id") val cellId: String,
    @SerialName("supplier_id") val supplierId: String,
)

@Serializable
data class ScenarioPatchRequest(
    @SerialName("cell_id") val cellId: String = "cell-uz",
    @SerialName("supplier_id") val supplierId: String,
    @SerialName("object_type") val objectType: String,
    @SerialName("object_id") val objectId: String,
    @SerialName("operation") val operation: String,
    @SerialName("patch_properties") val patchProperties: Map<String, JsonElement> = emptyMap(),
)

@Serializable
data class ScenarioCommitResult(
    @SerialName("scenario_id") val scenarioId: String,
    @SerialName("committed_objects") val committedObjects: Int,
    @SerialName("conflicts") val conflicts: List<ScenarioWatermarkConflict> = emptyList(),
)

@Serializable
data class ScenarioWatermarkConflict(
    @SerialName("cell_id") val cellId: String,
    @SerialName("object_type") val objectType: String,
    @SerialName("object_id") val objectId: String,
    @SerialName("base_version") val baseVersion: Long,
    @SerialName("live_version") val liveVersion: Long,
)
