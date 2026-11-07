package com.pegasusx.warehouse.ui.screens.cashier

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.mobilekit.foundry.AMLCheckRequest
import com.pegasusx.mobilekit.foundry.AMLCheckResponse
import com.pegasusx.mobilekit.foundry.BlindCountRequest
import com.pegasusx.mobilekit.foundry.CITHandoverRequest
import com.pegasusx.mobilekit.foundry.CashierSessionView
import com.pegasusx.mobilekit.foundry.DepotVaultLedgerEntry
import com.pegasusx.mobilekit.foundry.RRNTenderReclassificationRequest
import com.pegasusx.mobilekit.foundry.StartCashierSessionRequest
import com.pegasusx.warehouse.data.remote.WarehouseApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class DenominationEntry(
    val noteUzs: Long,
    val count: Long = 0L,
) {
    val totalTiyin: Long get() = noteUzs * count * 100L
    val totalUzs: Long get() = noteUzs * count
}

data class CashierUiState(
    val depotId: String = "depot-tashkent-south",
    val supplierId: String = "supplier-tashkent-1",
    val driverId: String = "driver-uz-09",
    val routeId: String = "route-tashkent-042",
    val bagSealBarcode: String = "BAG-UZ-99120",
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

    fun updateBagSealBarcode(barcode: String) {
        _uiState.update { it.copy(bagSealBarcode = barcode, errorMessage = null) }
    }

    fun updateDriverId(id: String) {
        _uiState.update { it.copy(driverId = id) }
    }

    fun updateRouteId(id: String) {
        _uiState.update { it.copy(routeId = id) }
    }

    fun setDenominationCount(noteUzs: Long, count: Long) {
        val safeCount = if (count < 0L) 0L else count
        _uiState.update { current ->
            val updated = current.denominations.toMutableMap()
            updated[noteUzs] = safeCount
            val totalMinor = updated.entries.sumOf { (note, cnt) -> note * cnt * 100L }
            current.copy(
                denominations = updated,
                countedCashMinor = totalMinor,
            )
        }
    }

    fun adjustDenominationCount(noteUzs: Long, delta: Long) {
        val currentCount = _uiState.value.denominations[noteUzs] ?: 0L
        setDenominationCount(noteUzs, currentCount + delta)
    }

    fun clearCounts() {
        _uiState.update { current ->
            val cleared = current.denominations.keys.associateWith { 0L }
            current.copy(denominations = cleared, countedCashMinor = 0L)
        }
    }

    fun startSession(expectedCashMinor: Long = 1425000000L) {
        val state = _uiState.value
        _uiState.update { it.copy(isLoading = true, errorMessage = null, statusMessage = null) }
        viewModelScope.launch {
            try {
                val req = StartCashierSessionRequest(
                    cellId = "cell-uz",
                    supplierId = state.supplierId,
                    depotId = state.depotId,
                    driverId = state.driverId,
                    routeId = state.routeId,
                    bagSealBarcode = state.bagSealBarcode,
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
        terminalId: String = "POS-TASH-01",
        cardLastFour: String = "8821",
        paymentNetwork: String = "HUMO",
    ) {
        val sessionId = _uiState.value.session?.sessionId
        if (sessionId.isNullOrBlank()) {
            _uiState.update { it.copy(errorMessage = "Active session required for reclassification") }
            return
        }

        _uiState.update { it.copy(isLoading = true, errorMessage = null) }
        viewModelScope.launch {
            try {
                val nowIso = java.time.Instant.now().toString()
                val req = RRNTenderReclassificationRequest(
                    sessionId = sessionId,
                    orderId = orderId,
                    disputedCashMinor = disputedCashMinor,
                    gatewayRrn = gatewayRrn,
                    terminalId = terminalId,
                    cardLastFour = cardLastFour,
                    authCode = "AUTH-${(100000..999999).random()}",
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
    ) {
        val state = _uiState.value
        val sessionId = state.session?.sessionId ?: "sess-batch-01"
        _uiState.update { it.copy(isLoading = true, errorMessage = null) }

        viewModelScope.launch {
            try {
                val req = CITHandoverRequest(
                    depotId = state.depotId,
                    supplierId = state.supplierId,
                    sessionIds = listOf(sessionId),
                    courierBadgeId = courierBadgeId,
                    citSealBarcode = citSealBarcode,
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
        _uiState.update { it.copy(isLoading = true, errorMessage = null) }
        viewModelScope.launch {
            try {
                val today = java.time.LocalDate.now().toString()
                val req = AMLCheckRequest(
                    legalEntityTin = tin,
                    supplierId = state.supplierId,
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
                            statusMessage = if (ceil > 0) "AML Exposure OK: ${acc / 100} / ${ceil / 100} UZS" else "AML Exposure OK: ${acc / 100} UZS (Uncapped)",
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
}
