package com.pegasusx.driver.ui.screens.fleet

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.driver.data.remote.DriverApi
import com.pegasusx.mobilekit.foundry.DVIRChecklist8Point
import com.pegasusx.mobilekit.foundry.FleetDVIRInspection
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class DVIRUiState(
    val supplierId: String = "supplier-tashkent-1",
    val vehicleId: String = "vh-isuzu-01",
    val driverId: String = "driver-uz-09",
    val odometerKm: Long = 125400L,
    val requiresColdChain: Boolean = false,
    val reeferTempCelsius: Double = 4.0,
    val checklist: DVIRChecklist8Point = DVIRChecklist8Point(),
    val isSubmitting: Boolean = false,
    val isRedTagged: Boolean = false,
    val redTagWorkOrderId: String? = null,
    val redTagFailedPoints: List<String> = emptyList(),
    val dispatchAuthorized: Boolean? = null,
    val statusMessage: String? = null,
    val errorMessage: String? = null,
)

@HiltViewModel
class DVIRViewModel @Inject constructor(
    private val driverApi: DriverApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(DVIRUiState())
    val uiState: StateFlow<DVIRUiState> = _uiState.asStateFlow()

    fun updateVehicleId(id: String) {
        _uiState.update { it.copy(vehicleId = id, dispatchAuthorized = null, isRedTagged = false, errorMessage = null) }
    }

    fun updateDriverId(id: String) {
        _uiState.update { it.copy(driverId = id) }
    }

    fun updateOdometerKm(km: Long) {
        _uiState.update { it.copy(odometerKm = km) }
    }

    fun updateRequiresColdChain(required: Boolean) {
        _uiState.update { it.copy(requiresColdChain = required) }
    }

    fun updateReeferTempCelsius(temp: Double) {
        _uiState.update { it.copy(reeferTempCelsius = temp) }
    }

    fun updateChecklist(transform: (DVIRChecklist8Point) -> DVIRChecklist8Point) {
        _uiState.update { it.copy(checklist = transform(it.checklist)) }
    }

    fun submitDVIR() {
        val state = _uiState.value
        _uiState.update { it.copy(isSubmitting = true, errorMessage = null, statusMessage = null) }

        val failedPoints = mutableListOf<String>()
        val cl = state.checklist
        if (!cl.steeringAndHornOk) failedPoints.add("steering_and_horn")
        if (!cl.serviceAndParkingBrakesOk) failedPoints.add("service_and_parking_brakes")
        if (!cl.tiresTreadAndPressureOk) failedPoints.add("tires_tread_and_pressure")
        if (!cl.lightsAndHazardsOk) failedPoints.add("lights_and_hazards")
        if (!cl.wipersAndFluidsOk) failedPoints.add("wipers_and_fluids")
        if (!cl.emergencyKitAndExtinguisherOk) failedPoints.add("emergency_kit_and_extinguisher")
        if (!cl.cargoBoxSanitationOk) failedPoints.add("cargo_box_sanitation")
        if (state.requiresColdChain && !cl.reeferTempCalibratedOk) failedPoints.add("reefer_temp_calibrated")

        val criticalDefect = failedPoints.isNotEmpty()
        val inspection = FleetDVIRInspection(
            supplierId = state.supplierId,
            vehicleId = state.vehicleId,
            driverId = state.driverId,
            requiresColdChain = state.requiresColdChain,
            checklist = state.checklist,
            failedPoints = failedPoints,
            criticalDefectFound = criticalDefect,
            redTagApplied = criticalDefect,
            odometerKm = state.odometerKm,
            reeferTempCelsius = state.reeferTempCelsius,
        )

        viewModelScope.launch {
            try {
                val resp = driverApi.submitDVIR(inspection)
                if (resp.isSuccessful) {
                    val body = resp.body()
                    if (body != null && body.redTagApplied) {
                        _uiState.update {
                            it.copy(
                                isSubmitting = false,
                                isRedTagged = true,
                                dispatchAuthorized = false,
                                redTagWorkOrderId = body.maintenanceWorkOrderId ?: "wo-maint-${System.currentTimeMillis()}",
                                redTagFailedPoints = body.failedPoints ?: failedPoints,
                                statusMessage = "Vehicle Grounded: OUT_OF_SERVICE (RED_TAG_QUARANTINE)",
                            )
                        }
                    } else {
                        _uiState.update {
                            it.copy(
                                isSubmitting = false,
                                isRedTagged = false,
                                dispatchAuthorized = true,
                                redTagWorkOrderId = null,
                                redTagFailedPoints = emptyList(),
                                statusMessage = "DVIR Passed: Vehicle Authorized for Dispatch",
                            )
                        }
                    }
                } else if (resp.code() == 409) {
                    val workOrderId = "wo-${state.vehicleId}-${System.currentTimeMillis() % 100000}"
                    _uiState.update {
                        it.copy(
                            isSubmitting = false,
                            isRedTagged = true,
                            dispatchAuthorized = false,
                            redTagWorkOrderId = workOrderId,
                            redTagFailedPoints = failedPoints,
                            statusMessage = "Vehicle Grounded: OUT_OF_SERVICE (RED_TAG_QUARANTINE)",
                        )
                    }
                } else {
                    _uiState.update {
                        it.copy(
                            isSubmitting = false,
                            errorMessage = "DVIR submission failed (HTTP ${resp.code()})",
                        )
                    }
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        isSubmitting = false,
                        errorMessage = e.message ?: "Network error during DVIR submission",
                    )
                }
            }
        }
    }

    fun checkDispatchAuthorization() {
        val state = _uiState.value
        _uiState.update { it.copy(isSubmitting = true, errorMessage = null) }
        viewModelScope.launch {
            try {
                val resp = driverApi.checkVehicleDispatch(state.vehicleId)
                if (resp.isSuccessful && resp.body() != null) {
                    val isAuth = resp.body()?.dispatchAuthorized == true
                    _uiState.update {
                        it.copy(
                            isSubmitting = false,
                            dispatchAuthorized = isAuth,
                            isRedTagged = !isAuth,
                            statusMessage = if (isAuth) "Dispatch Authorized" else "Dispatch Blocked: Inspection Required or Red-Tagged",
                        )
                    }
                } else if (resp.code() == 409) {
                    _uiState.update {
                        it.copy(
                            isSubmitting = false,
                            dispatchAuthorized = false,
                            isRedTagged = true,
                            statusMessage = "Dispatch Blocked (409 Conflict): Vehicle Red-Tagged or Hold Active",
                        )
                    }
                } else {
                    _uiState.update {
                        it.copy(
                            isSubmitting = false,
                            errorMessage = "Dispatch check failed (HTTP ${resp.code()})",
                        )
                    }
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        isSubmitting = false,
                        errorMessage = e.message ?: "Failed to verify vehicle dispatch status",
                    )
                }
            }
        }
    }
}
