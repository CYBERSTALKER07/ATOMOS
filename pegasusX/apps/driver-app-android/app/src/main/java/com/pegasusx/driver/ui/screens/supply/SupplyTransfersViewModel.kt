package com.pegasusx.driver.ui.screens.supply

import android.annotation.SuppressLint
import android.app.Application
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.google.android.gms.tasks.CancellationTokenSource
import com.pegasusx.driver.data.model.ArriveSupplyTransferRequest
import com.pegasusx.driver.data.model.SupplyTransferRow
import com.pegasusx.driver.data.remote.DriverApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await
import javax.inject.Inject

data class SupplyTransfersUiState(
    val transfers: List<SupplyTransferRow> = emptyList(),
    val isLoading: Boolean = true,
    val isArriving: String? = null,
    val error: String? = null,
    val successMessage: String? = null,
) {
    val activeCount: Int get() = transfers.count { it.state.uppercase() != "ARRIVED" }
}

@HiltViewModel
class SupplyTransfersViewModel @Inject constructor(
    private val app: Application,
    private val api: DriverApi,
) : ViewModel() {

    private val _state = MutableStateFlow(SupplyTransfersUiState())
    val state: StateFlow<SupplyTransfersUiState> = _state.asStateFlow()

    private val fusedClient = LocationServices.getFusedLocationProviderClient(app)

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _state.update { it.copy(isLoading = true, error = null) }
            try {
                val response = api.getSupplyTransfers()
                _state.update { it.copy(isLoading = false, transfers = response.transfers) }
            } catch (e: Exception) {
                _state.update {
                    it.copy(isLoading = false, error = e.message ?: "Failed to load supply transfers")
                }
            }
        }
    }

    @SuppressLint("MissingPermission")
    fun markArrived(transferId: String) {
        viewModelScope.launch {
            _state.update { it.copy(isArriving = transferId, error = null, successMessage = null) }
            try {
                val location = fusedClient.getCurrentLocation(
                    Priority.PRIORITY_HIGH_ACCURACY,
                    CancellationTokenSource().token,
                ).await()
                val response = api.arriveSupplyTransfer(
                    transferId = transferId,
                    idempotencyKey = DriverIdempotencyKeys.supplyTransferArrive(transferId),
                    body = ArriveSupplyTransferRequest(
                        latitude = location?.latitude ?: 0.0,
                        longitude = location?.longitude ?: 0.0,
                    ),
                )
                _state.update {
                    it.copy(
                        isArriving = null,
                        successMessage = "Transfer ${response.transferId.takeLast(6)} marked arrived",
                    )
                }
                refresh()
            } catch (e: Exception) {
                _state.update {
                    it.copy(
                        isArriving = null,
                        error = e.message ?: "Arrive failed",
                    )
                }
            }
        }
    }

    fun clearMessages() {
        _state.update { it.copy(error = null, successMessage = null) }
    }
}
