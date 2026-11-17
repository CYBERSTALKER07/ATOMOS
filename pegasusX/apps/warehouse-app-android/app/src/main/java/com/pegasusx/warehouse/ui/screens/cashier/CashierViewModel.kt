package com.pegasusx.warehouse.ui.screens.cashier

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.warehouse.data.foundry.AMLCheckRequest
import com.pegasusx.warehouse.data.foundry.AMLCheckResponse
import com.pegasusx.warehouse.data.foundry.BlindCountRequest
import com.pegasusx.warehouse.data.foundry.CITHandoverRequest
import com.pegasusx.warehouse.data.foundry.CashierSessionView
import com.pegasusx.warehouse.data.foundry.DepotVaultLedgerEntry
import com.pegasusx.warehouse.data.foundry.RRNTenderReclassificationRequest
import com.pegasusx.warehouse.data.foundry.StartCashierSessionRequest
import com.pegasusx.warehouse.data.model.Driver
import com.pegasusx.warehouse.data.model.Manifest
import com.pegasusx.warehouse.data.model.WarehouseFleetLiveRoute
import com.pegasusx.warehouse.data.remote.TokenHolder
import com.pegasusx.warehouse.data.remote.WarehouseApi
import com.pegasusx.warehouse.ui.design.MarketPackStore
import com.pegasusx.warehouse.util.JwtPayload
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.async
import kotlinx.coroutines.coroutineScope
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import java.text.DecimalFormat
import java.text.DecimalFormatSymbols
import java.util.Locale
import javax.inject.Inject

data class DenominationEntry(
    val noteSom: Long,
    val count: Long = 0L,
) {
    val totalMinor: Long get() = noteSom * count * 100L
    val totalSom: Long get() = noteSom * count
}

data class CashierDriverOption(
    val driverId: String,
    val label: String,
    val routeId: String? = null,
)

data class CashierRouteOption(
    val routeId: String,
    val label: String,
    val driverId: String? = null,
    val manifestId: String? = null,
)

data class CashierUiState(
    val cellId: String = "",
    val depotId: String = "",
    val supplierId: String = "",
    val driverId: String = "",
    val routeId: String = "",
    val bagSealBarcode: String = "",
    val driverOptions: List<CashierDriverOption> = emptyList(),
    val routeOptions: List<CashierRouteOption> = emptyList(),
    val selectorsLoading: Boolean = false,
    val selectorsError: String? = null,
    val manualEntryMode: Boolean = false,
    val session: CashierSessionView? = null,
    val attemptNumber: Long = 1L,
    val denominations: Map<Long, Long> = mapOf(
        200000L to 0L,
        100000L to 0L,
        50000L to 0L,
        20000L to 0L,
        10000L to 0L,
        5000L to 0L,
        2000L to 0L,
        1000L to 0L,
    ),
    val countedCashMinor: Long = 0L,
    val isBlindCountMasked: Boolean = true,
    val isRecountRequired: Boolean = false,
    val isBalanced: Boolean = false,
    val isQuarantined: Boolean = false,
    val clearanceTokenHash: String? = null,
    val microSkimmingFlagged: Boolean = false,
    val isLoading: Boolean = false,
    val statusMessage: String? = null,
    val errorMessage: String? = null,
    val amlExposureMinor: Long = 0L,
    val amlCeilingMinor: Long = 0L, // Default uncapped
    val amlExceeded: Boolean = false,
    val lastVaultEntry: DepotVaultLedgerEntry? = null,
)

@HiltViewModel
class CashierViewModel @Inject constructor(
    private val api: WarehouseApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(CashierUiState())
    val uiState: StateFlow<CashierUiState> = _uiState.asStateFlow()

    init {
        val token = runCatching { TokenHolder.token }.getOrNull()
        val resolvedDepot = runCatching { TokenHolder.warehouseId }.getOrNull()
            ?.takeIf { it.isNotBlank() }
            ?: JwtPayload.homeNodeId(token).orEmpty()
        val resolvedCell = JwtPayload.homeCell(token)
            ?: runCatching { MarketPackStore.current?.homeCell }.getOrNull()?.takeIf { it.isNotBlank() }
            ?: ""
        val resolvedSupplier = JwtPayload.supplierId(token).orEmpty()

        _uiState.update {
            it.copy(
                cellId = resolvedCell,
                depotId = resolvedDepot,
                supplierId = resolvedSupplier,
            )
        }
    }

    fun applyScope(cellId: String, depotId: String, supplierId: String) {
        _uiState.update {
            it.copy(
                cellId = cellId,
                depotId = depotId,
                supplierId = supplierId,
                errorMessage = null,
            )
        }
    }

    fun updateDepotId(id: String) {
        _uiState.update { it.copy(depotId = id, errorMessage = null) }
    }

    fun updateSupplierId(id: String) {
        _uiState.update { it.copy(supplierId = id, errorMessage = null) }
    }

    fun updateBagSealBarcode(barcode: String) {
        _uiState.update { it.copy(bagSealBarcode = barcode, errorMessage = null) }
    }

    fun updateDriverId(id: String) {
        _uiState.update { it.copy(driverId = id, errorMessage = null) }
    }

    fun updateRouteId(id: String) {
        _uiState.update { it.copy(routeId = id, errorMessage = null) }
    }

    fun selectDriver(driverId: String) {
        _uiState.update { current ->
            val linkedRoute = current.driverOptions.firstOrNull { it.driverId == driverId }?.routeId
            val nextRoute = if (current.routeId.isBlank() && !linkedRoute.isNullOrBlank()) {
                linkedRoute
            } else {
                current.routeId
            }
            current.copy(driverId = driverId, routeId = nextRoute, errorMessage = null)
        }
    }

    fun selectRoute(routeId: String) {
        _uiState.update { current ->
            val linkedDriver = current.routeOptions.firstOrNull { it.routeId == routeId }?.driverId
            val nextDriver = if (current.driverId.isBlank() && !linkedDriver.isNullOrBlank()) {
                linkedDriver
            } else {
                current.driverId
            }
            current.copy(routeId = routeId, driverId = nextDriver, errorMessage = null)
        }
    }

    fun toggleManualEntryMode() {
        _uiState.update { it.copy(manualEntryMode = !it.manualEntryMode) }
    }

    fun loadSelectors() {
        val currentDepot = _uiState.value.depotId.trim()
        _uiState.update { it.copy(selectorsLoading = true, selectorsError = null) }
        viewModelScope.launch {
            var anyFailed = false
            val (drivers, manifests, fleetRoutes) = coroutineScope {
                val driversDef = async {
                    runCatching { api.getDrivers() }
                        .onFailure { anyFailed = true }
                        .getOrNull()
                        ?.let { resp ->
                            if (resp.isSuccessful) resp.body()?.drivers.orEmpty() else {
                                anyFailed = true
                                emptyList()
                            }
                        } ?: emptyList()
                }
                val manifestsDef = async {
                    runCatching { api.getManifests() }
                        .onFailure { anyFailed = true }
                        .getOrNull()
                        ?.let { resp ->
                            if (resp.isSuccessful) resp.body()?.manifests.orEmpty() else {
                                anyFailed = true
                                emptyList()
                            }
                        } ?: emptyList()
                }
                val fleetDef = async {
                    runCatching { api.getFleetLiveMap(warehouseId = currentDepot.ifBlank { null }) }
                        .onFailure { anyFailed = true }
                        .getOrNull()
                        ?.let { resp ->
                            if (resp.isSuccessful) resp.body()?.routes.orEmpty() else {
                                anyFailed = true
                                emptyList()
                            }
                        } ?: emptyList()
                }
                Triple(driversDef.await(), manifestsDef.await(), fleetDef.await())
            }

            val (driverOpts, routeOpts) = buildSelectorOptions(drivers, manifests, fleetRoutes)
            _uiState.update { current ->
                current.copy(
                    driverOptions = driverOpts,
                    routeOptions = routeOpts,
                    selectorsLoading = false,
                    manualEntryMode = current.manualEntryMode || (driverOpts.isEmpty() && routeOpts.isEmpty()),
                    selectorsError = if (anyFailed && driverOpts.isEmpty() && routeOpts.isEmpty()) {
                        "Live driver and route lists are unavailable; manual ID entry enabled."
                    } else {
                        null
                    },
                )
            }
        }
    }

    fun setDenominationCount(noteSom: Long, count: Long) {
        val safeCount = if (count < 0L) 0L else count
        _uiState.update { current ->
            val updated = current.denominations.toMutableMap()
            updated[noteSom] = safeCount
            val totalMinor = updated.entries.sumOf { (note, cnt) -> note * cnt * 100L }
            current.copy(
                denominations = updated,
                countedCashMinor = totalMinor,
            )
        }
    }

    fun adjustDenominationCount(noteSom: Long, delta: Long) {
        val currentCount = _uiState.value.denominations[noteSom] ?: 0L
        setDenominationCount(noteSom, currentCount + delta)
    }

    fun clearCounts() {
        _uiState.update { current ->
            val cleared = current.denominations.keys.associateWith { 0L }
            current.copy(denominations = cleared, countedCashMinor = 0L)
        }
    }

    fun startSession(expectedCashMinor: Long = 0L) {
        val state = _uiState.value
        if (state.bagSealBarcode.isBlank()) {
            _uiState.update { it.copy(errorMessage = "Bag seal barcode is required") }
            return
        }
        if (state.driverId.isBlank()) {
            _uiState.update { it.copy(errorMessage = "Driver selection or ID is required") }
            return
        }
        if (state.routeId.isBlank()) {
            _uiState.update { it.copy(errorMessage = "Route selection or ID is required") }
            return
        }
        if (state.depotId.isBlank() || state.supplierId.isBlank()) {
            _uiState.update { it.copy(errorMessage = "Depot and supplier scope are required") }
            return
        }

        _uiState.update { it.copy(isLoading = true, errorMessage = null, statusMessage = null) }
        viewModelScope.launch {
            try {
                val req = StartCashierSessionRequest(
                    cellId = state.cellId.trim(),
                    supplierId = state.supplierId.trim(),
                    depotId = state.depotId.trim(),
                    driverId = state.driverId.trim(),
                    routeId = state.routeId.trim(),
                    bagSealBarcode = state.bagSealBarcode.trim(),
                    expectedCashMinor = expectedCashMinor,
                )
                val resp = api.startCashierSession(req)
                if (resp.isSuccessful && resp.body() != null) {
                    val sess = resp.body()
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            session = sess,
                            attemptNumber = sess?.attemptNumber ?: 1L,
                            isBlindCountMasked = true,
                            isRecountRequired = false,
                            statusMessage = "Deposit Bag Registered. Please perform blind physical count.",
                        )
                    }
                } else if (resp.code() == 409) {
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            errorMessage = "Bag Seal Conflict (409): Duplicate deposit bag seal barcode already logged.",
                        )
                    }
                } else {
                    _uiState.update {
                        it.copy(isLoading = false, errorMessage = "Failed to start session (${resp.code()})")
                    }
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(isLoading = false, errorMessage = e.message ?: "Network error starting cashier session")
                }
            }
        }
    }

    fun submitBlindCount(toleranceThresholdMinor: Long = 500000L) {
        val state = _uiState.value
        val sessionId = state.session?.sessionId
        if (sessionId.isNullOrBlank()) {
            _uiState.update { it.copy(errorMessage = "Please start or select a session first") }
            return
        }

        _uiState.update { it.copy(isLoading = true, errorMessage = null, statusMessage = null) }
        viewModelScope.launch {
            try {
                val denomMap = state.denominations.mapKeys { it.key.toString() }
                val req = BlindCountRequest(
                    denominationsUzs = denomMap,
                    toleranceThresholdMinor = toleranceThresholdMinor,
                )
                val resp = api.submitBlindCount(sessionId, req)
                if (resp.isSuccessful && resp.body() != null) {
                    val updated = resp.body()
                    val status = updated?.status ?: ""
                    val isBal = status == "BALANCED" || status == "TOLERANCE_VARIANCE"
                    val isQuar = status == "DISCREPANT_QUARANTINE"

                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            session = updated,
                            attemptNumber = updated?.attemptNumber ?: 1L,
                            isBalanced = isBal,
                            isQuarantined = isQuar,
                            isBlindCountMasked = !isBal && !isQuar,
                            isRecountRequired = false,
                            clearanceTokenHash = updated?.clearanceTokenHash,
                            microSkimmingFlagged = updated?.microSkimmingFlagged ?: false,
                            statusMessage = if (isBal) "Count Balanced! Clearance Token Issued." else "Session quarantined for supervisor review.",
                        )
                    }
                } else if (resp.code() == 409) {
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            isRecountRequired = true,
                            isBlindCountMasked = true, // Strictly masked on recount #1!
                            attemptNumber = 2L,
                            statusMessage = "MANDATORY BLIND RECOUNT #2 REQUIRED: Physical count does not match route records. Expected amount remains masked.",
                        )
                    }
                } else {
                    _uiState.update {
                        it.copy(isLoading = false, errorMessage = "Count submission failed (${resp.code()})")
                    }
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(isLoading = false, errorMessage = e.message ?: "Network error submitting blind count")
                }
            }
        }
    }

    fun reclassifyTenderRRN(
        orderId: String,
        gatewayRrn: String,
        disputedCashMinor: Long,
        terminalId: String = "",
        cardLastFour: String = "",
        paymentNetwork: String = "HUMO",
        authCode: String = "",
    ) {
        val sessionId = _uiState.value.session?.sessionId
        if (sessionId.isNullOrBlank()) {
            _uiState.update { it.copy(errorMessage = "Active session required for reclassification") }
            return
        }
        if (orderId.isBlank() || gatewayRrn.isBlank() || disputedCashMinor <= 0L) {
            _uiState.update { it.copy(errorMessage = "Order ID, Gateway RRN, and disputed so'm amount are required") }
            return
        }

        _uiState.update { it.copy(isLoading = true, errorMessage = null) }
        viewModelScope.launch {
            try {
                val nowIso = java.time.Instant.now().toString()
                val req = RRNTenderReclassificationRequest(
                    sessionId = sessionId,
                    orderId = orderId.trim(),
                    disputedCashMinor = disputedCashMinor,
                    gatewayRrn = gatewayRrn.trim(),
                    terminalId = terminalId.trim(),
                    cardLastFour = cardLastFour.trim(),
                    authCode = authCode.trim(),
                    paymentNetwork = paymentNetwork,
                    rrnCapturedAt = nowIso,
                    podGeofenceTimestamp = nowIso,
                )
                val resp = api.reclassifyTender(req)
                if (resp.isSuccessful && resp.body() != null) {
                    val updated = resp.body()
                    val isBal = updated?.status == "BALANCED"
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            session = updated,
                            isBalanced = isBal,
                            clearanceTokenHash = updated?.clearanceTokenHash,
                            statusMessage = "RRN Payment Reclassified! Compensating ledger applied.",
                        )
                    }
                } else {
                    _uiState.update {
                        it.copy(isLoading = false, errorMessage = "RRN reclassification failed (${resp.code()})")
                    }
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(isLoading = false, errorMessage = e.message ?: "Failed to reclassify tender")
                }
            }
        }
    }

    fun executeCITHandover(
        courierBadgeId: String,
        citSealBarcode: String,
        amountMinor: Long,
        sessionIds: List<String> = emptyList(),
    ) {
        val state = _uiState.value
        val resolvedSessionIds = sessionIds
            .map { it.trim() }
            .filter { it.isNotEmpty() }
            .ifEmpty { listOfNotNull(state.session?.sessionId?.takeIf { it.isNotBlank() }) }

        if (resolvedSessionIds.isEmpty()) {
            _uiState.update { it.copy(errorMessage = "At least one cashier session ID is required for CIT handover") }
            return
        }
        if (courierBadgeId.isBlank() || citSealBarcode.isBlank() || amountMinor <= 0L) {
            _uiState.update { it.copy(errorMessage = "Courier badge, seal barcode, and positive so'm amount are required") }
            return
        }
        if (state.depotId.isBlank() || state.supplierId.isBlank()) {
            _uiState.update { it.copy(errorMessage = "Depot and supplier scope are required for CIT handover") }
            return
        }

        _uiState.update { it.copy(isLoading = true, errorMessage = null) }
        viewModelScope.launch {
            try {
                val req = CITHandoverRequest(
                    depotId = state.depotId.trim(),
                    supplierId = state.supplierId.trim(),
                    sessionIds = resolvedSessionIds,
                    courierBadgeId = courierBadgeId.trim(),
                    citSealBarcode = citSealBarcode.trim(),
                    cashMinor = amountMinor,
                )
                val resp = api.executeCITHandover(req)
                if (resp.isSuccessful && resp.body() != null) {
                    val entry = resp.body()
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            lastVaultEntry = entry,
                            statusMessage = "CIT Armored Handover Complete: Manifest ${entry?.citManifestId ?: entry?.vaultEntryId}",
                        )
                    }
                } else {
                    _uiState.update {
                        it.copy(isLoading = false, errorMessage = "CIT Handover failed (${resp.code()})")
                    }
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(isLoading = false, errorMessage = e.message ?: "Failed to execute CIT handover")
                }
            }
        }
    }

    fun checkAMLExposure(tin: String, cashMinor: Long) {
        val state = _uiState.value
        if (tin.isBlank()) {
            _uiState.update { it.copy(errorMessage = "Legal entity TIN is required for AML check") }
            return
        }
        if (state.supplierId.isBlank()) {
            _uiState.update { it.copy(errorMessage = "Supplier scope is required for AML check") }
            return
        }

        _uiState.update { it.copy(isLoading = true, errorMessage = null) }
        viewModelScope.launch {
            try {
                val today = java.time.LocalDate.now().toString()
                val req = AMLCheckRequest(
                    legalEntityTin = tin.trim(),
                    supplierId = state.supplierId.trim(),
                    calendarDate = today,
                    cashMinor = cashMinor,
                )
                val resp = api.checkAML(req)
                if (resp.isSuccessful && resp.body() != null) {
                    val body = resp.body()
                    val acc = body?.accumulatedCashMinor ?: 0L
                    val ceil = body?.ceilingMinor ?: 0L
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            amlExposureMinor = acc,
                            amlCeilingMinor = ceil,
                            amlExceeded = ceil > 0 && acc > ceil,
                            statusMessage = if (ceil > 0) {
                                "AML Exposure OK: ${formatWholeSom(acc)} / ${formatWholeSom(ceil)}"
                            } else {
                                "AML Exposure OK: ${formatWholeSom(acc)} (Uncapped)"
                            },
                        )
                    }
                } else if (resp.code() == 409) {
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            amlExceeded = true,
                            errorMessage = "AML Ceiling Exceeded (409): Daily settlement exposure limit reached for TIN $tin.",
                        )
                    }
                } else {
                    _uiState.update {
                        it.copy(isLoading = false, errorMessage = "AML check failed (${resp.code()})")
                    }
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(isLoading = false, errorMessage = e.message ?: "Failed to check AML exposure")
                }
            }
        }
    }

    companion object {
        fun formatWholeSom(minor: Long): String = formatWholeSomMajor(minor / 100L)

        fun formatWholeSomMajor(wholeSom: Long): String {
            val symbols = DecimalFormatSymbols(Locale.US).apply {
                groupingSeparator = ' '
            }
            val formatter = DecimalFormat("#,##0", symbols)
            return "${formatter.format(wholeSom)} so'm"
        }

        fun buildSelectorOptions(
            drivers: List<Driver>,
            manifests: List<Manifest>,
            fleetRoutes: List<WarehouseFleetLiveRoute>,
        ): Pair<List<CashierDriverOption>, List<CashierRouteOption>> {
            val driverMap = LinkedHashMap<String, CashierDriverOption>()
            val routeMap = LinkedHashMap<String, CashierRouteOption>()
            val seenManifestIds = HashSet<String>()

            for (driver in drivers) {
                val id = driver.driverId.trim()
                if (id.isEmpty()) continue
                val name = driver.name.trim()
                driverMap[id] = CashierDriverOption(
                    driverId = id,
                    label = if (name.isNotEmpty()) "$name ($id)" else id,
                )
            }

            for (route in fleetRoutes) {
                val rId = route.routeId.trim()
                val dId = route.driverId.trim().takeIf { it.isNotEmpty() }
                val mId = route.manifestId.trim().takeIf { it.isNotEmpty() }
                if (mId != null) seenManifestIds.add(mId)

                if (rId.isNotEmpty()) {
                    val suffix = route.driverName?.trim()?.takeIf { it.isNotEmpty() } ?: dId.orEmpty()
                    routeMap[rId] = CashierRouteOption(
                        routeId = rId,
                        label = if (suffix.isNotEmpty()) "$rId · $suffix" else rId,
                        driverId = dId,
                        manifestId = mId,
                    )
                }

                if (dId != null) {
                    val existing = driverMap[dId]
                    if (existing == null) {
                        val name = route.driverName?.trim().orEmpty()
                        driverMap[dId] = CashierDriverOption(
                            driverId = dId,
                            label = if (name.isNotEmpty()) "$name ($dId)" else dId,
                            routeId = rId.takeIf { it.isNotEmpty() },
                        )
                    } else if (existing.routeId == null && rId.isNotEmpty()) {
                        driverMap[dId] = existing.copy(routeId = rId)
                    }
                }
            }

            for (manifest in manifests) {
                val mId = manifest.manifestId.trim()
                if (mId.isEmpty() || seenManifestIds.contains(mId) || routeMap.containsKey(mId)) {
                    continue
                }
                val parts = mutableListOf(mId)
                manifest.driverName.trim().takeIf { it.isNotEmpty() }?.let { parts.add(it) }
                manifest.vehicleLabel.trim().takeIf { it.isNotEmpty() }?.let { parts.add(it) }
                routeMap[mId] = CashierRouteOption(
                    routeId = mId,
                    label = parts.joinToString(" · "),
                    manifestId = mId,
                )
            }

            return driverMap.values.toList() to routeMap.values.toList()
        }
    }
}
