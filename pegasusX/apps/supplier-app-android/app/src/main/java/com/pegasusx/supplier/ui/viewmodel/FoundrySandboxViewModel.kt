package com.pegasusx.supplier.ui.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.mobilekit.foundry.ActionExecutionRequest
import com.pegasusx.mobilekit.foundry.LotRecallRequest
import com.pegasusx.mobilekit.foundry.RecallBlastRadiusReport
import com.pegasusx.mobilekit.foundry.ScenarioCommitResult
import com.pegasusx.mobilekit.foundry.ScenarioPatchRequest
import com.pegasusx.mobilekit.foundry.ScenarioWatermarkConflict
import com.pegasusx.supplier.data.remote.SupplierApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonPrimitive
import java.util.UUID
import javax.inject.Inject

data class ScenarioPatchSummary(
    val objectType: String,
    val objectId: String,
    val key: String,
    val value: String,
)

data class FoundrySandboxUiState(
    val supplierId: String = "supplier-tashkent-1",
    val cellId: String = "cell-uz",
    val scenarioId: String = "scenario-surge-plan-01",
    val objectType: String = "SKU",
    val objectId: String = "sku-uz-cola-1l",
    val patchKey: String = "price_minor",
    val patchValue: String = "1250000",
    val appliedPatches: List<ScenarioPatchSummary> = emptyList(),
    val isCommitting: Boolean = false,
    val commitResult: ScenarioCommitResult? = null,
    val hasConflict: Boolean = false,
    val conflicts: List<ScenarioWatermarkConflict> = emptyList(),
    val lotBarcode: String = "LOT-SUGAR-2026-X8",
    val isAnalyzingRecall: Boolean = false,
    val recallReport: RecallBlastRadiusReport? = null,
    val isQuarantining: Boolean = false,
    val quarantineSuccess: Boolean = false,
    val statusMessage: String? = null,
    val errorMessage: String? = null,
)

@HiltViewModel
class FoundrySandboxViewModel @Inject constructor(
    private val api: SupplierApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(FoundrySandboxUiState())
    val uiState: StateFlow<FoundrySandboxUiState> = _uiState.asStateFlow()

    fun updateScenarioId(id: String) {
        _uiState.update { it.copy(scenarioId = id, errorMessage = null) }
    }

    fun updateObjectType(type: String) {
        _uiState.update { it.copy(objectType = type) }
    }

    fun updateObjectId(id: String) {
        _uiState.update { it.copy(objectId = id) }
    }

    fun updatePatchKey(key: String) {
        _uiState.update { it.copy(patchKey = key) }
    }

    fun updatePatchValue(value: String) {
        _uiState.update { it.copy(patchValue = value) }
    }

    fun updateLotBarcode(barcode: String) {
        _uiState.update { it.copy(lotBarcode = barcode, errorMessage = null) }
    }

    fun applyCowPatch() {
        val state = _uiState.value
        val key = state.patchKey.trim()
        val value = state.patchValue.trim()
        if (key.isBlank() || value.isBlank()) {
            _uiState.update { it.copy(errorMessage = "Patch key and value must not be blank") }
            return
        }

        _uiState.update { it.copy(errorMessage = null, statusMessage = null) }
        viewModelScope.launch {
            try {
                val patchProps = mapOf(key to JsonPrimitive(value))
                val req = ScenarioPatchRequest(
                    cellId = state.cellId,
                    supplierId = state.supplierId,
                    objectType = state.objectType,
                    objectId = state.objectId,
                    operation = "SET",
                    patchProperties = patchProps,
                )
                val resp = api.applyScenarioPatch(state.scenarioId, req)
                if (resp.isSuccessful) {
                    val newSummary = ScenarioPatchSummary(
                        objectType = state.objectType,
                        objectId = state.objectId,
                        key = key,
                        value = value,
                    )
                    _uiState.update {
                        it.copy(
                            appliedPatches = it.appliedPatches + newSummary,
                            statusMessage = "CoW Patch applied to ${state.objectType}:${state.objectId} [$key = $value]",
                        )
                    }
                } else {
                    _uiState.update {
                        it.copy(errorMessage = "Failed to apply scenario patch (${resp.code()})")
                    }
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(errorMessage = e.message ?: "Network error applying scenario patch")
                }
            }
        }
    }

    fun commitScenario() {
        val state = _uiState.value
        _uiState.update { it.copy(isCommitting = true, errorMessage = null, statusMessage = null) }

        viewModelScope.launch {
            try {
                val resp = api.commitScenario(state.scenarioId)
                if (resp.isSuccessful && resp.body() != null) {
                    val result = resp.body()
                    val hasConf = (result?.conflicts?.isNotEmpty()) ?: false
                    _uiState.update {
                        it.copy(
                            isCommitting = false,
                            commitResult = result,
                            hasConflict = hasConf,
                            conflicts = result?.conflicts.orEmpty(),
                            statusMessage = if (hasConf) "Commit Conflict (409): Baseline objects modified since scenario branched." else "Scenario successfully committed (${result?.committedObjects ?: 0} objects).",
                        )
                    }
                } else if (resp.code() == 409) {
                    _uiState.update {
                        it.copy(
                            isCommitting = false,
                            hasConflict = true,
                            errorMessage = "Scenario Optimistic Concurrency Conflict (409): Base layer mutated by concurrent transaction.",
                        )
                    }
                } else {
                    _uiState.update {
                        it.copy(isCommitting = false, errorMessage = "Commit failed (${resp.code()})")
                    }
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(isCommitting = false, errorMessage = e.message ?: "Failed to commit scenario")
                }
            }
        }
    }

    fun analyzeLotRecall() {
        val state = _uiState.value
        val rawLot = state.lotBarcode.trim()
        if (rawLot.isBlank()) {
            _uiState.update { it.copy(errorMessage = "Raw material lot barcode required") }
            return
        }

        _uiState.update { it.copy(isAnalyzingRecall = true, errorMessage = null, statusMessage = null) }
        viewModelScope.launch {
            try {
                val req = LotRecallRequest(
                    cellId = state.cellId,
                    supplierId = state.supplierId,
                    rawMaterialLotId = rawLot,
                )
                val resp = api.triggerLotRecall(req)
                if (resp.isSuccessful && resp.body() != null) {
                    val report = resp.body()
                    _uiState.update {
                        it.copy(
                            isAnalyzingRecall = false,
                            recallReport = report,
                            quarantineSuccess = false,
                            statusMessage = "Recall Blast-Radius Evaluated: ${report?.quarantinedSsccPallets?.size ?: 0} pallets & ${report?.inTransitDriverDropBlocks?.size ?: 0} transit drops affected.",
                        )
                    }
                } else {
                    _uiState.update {
                        it.copy(isAnalyzingRecall = false, errorMessage = "Lot recall analysis failed (${resp.code()})")
                    }
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(isAnalyzingRecall = false, errorMessage = e.message ?: "Failed to evaluate lot recall")
                }
            }
        }
    }

    fun executeQuarantineBlastRadius() {
        val state = _uiState.value
        val report = state.recallReport
        if (report == null) {
            _uiState.update { it.copy(errorMessage = "No evaluated recall report to quarantine") }
            return
        }

        _uiState.update { it.copy(isQuarantining = true, errorMessage = null) }
        viewModelScope.launch {
            try {
                val req = ActionExecutionRequest(
                    actionName = "EXECUTE_LOT_QUARANTINE_BLAST_RADIUS",
                    actionType = "MUTATION",
                    actorId = state.supplierId,
                    actorRole = "SUPPLIER_QUALITY_DIRECTOR",
                    targetObjectType = "RAW_MATERIAL_LOT",
                    targetObjectId = report.contaminatedRawMaterialLot,
                    payload = emptyMap(),
                    idempotencyKey = UUID.randomUUID().toString(),
                    cellId = state.cellId,
                    supplierId = state.supplierId,
                )
                val resp = api.executeAction(req)
                if (resp.isSuccessful) {
                    _uiState.update {
                        it.copy(
                            isQuarantining = false,
                            quarantineSuccess = true,
                            statusMessage = "EMERGENCY QUARANTINE APPLIED: All downstream pallets, totes, and in-transit driver deliveries locked.",
                        )
                    }
                } else {
                    _uiState.update {
                        it.copy(isQuarantining = false, errorMessage = "Quarantine action failed (${resp.code()})")
                    }
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(isQuarantining = false, errorMessage = e.message ?: "Failed to execute quarantine action")
                }
            }
        }
    }
}
