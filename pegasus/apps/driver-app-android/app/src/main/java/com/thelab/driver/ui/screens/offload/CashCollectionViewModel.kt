package com.thelab.driver.ui.screens.offload

import android.annotation.SuppressLint
import android.app.Application
import androidx.lifecycle.SavedStateHandle
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.google.android.gms.tasks.CancellationTokenSource
import com.thelab.driver.data.model.CollectCashRequest
import com.thelab.driver.data.remote.DriverApi
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await
import java.util.UUID
import javax.inject.Inject

data class CashCollectionUiState(
    val orderId: String = "",
    val amount: Long = 0,
    val isCompleting: Boolean = false,
    val completed: Boolean = false,
    val error: String? = null,
    val distanceM: Double? = null,
    val locationAvailable: Boolean = true
)

@HiltViewModel
class CashCollectionViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val api: DriverApi,
    private val app: Application
) : ViewModel() {

    private val orderId: String = savedStateHandle["orderId"] ?: ""
    private val amount: Long = savedStateHandle.get<Long>("amount") ?: 0L

    private val _state = MutableStateFlow(CashCollectionUiState(orderId = orderId, amount = amount))
    val state: StateFlow<CashCollectionUiState> = _state.asStateFlow()

    private val fusedClient = LocationServices.getFusedLocationProviderClient(app)

    @SuppressLint("MissingPermission")
    fun collectCash() {
        viewModelScope.launch {
            _state.update { it.copy(isCompleting = true, error = null) }
            try {
                val cts = CancellationTokenSource()
                val location = fusedClient.getCurrentLocation(
                    Priority.PRIORITY_HIGH_ACCURACY, cts.token
                ).await()

                if (location == null) {
                    _state.update {
                        it.copy(
                            isCompleting = false,
                            locationAvailable = false,
                            error = "Unable to get GPS location. Move to an open area and try again."
                        )
                    }
                    return@launch
                }

                val resp = api.collectCash(
                    request = CollectCashRequest(
                        orderId = orderId,
                        latitude = location.latitude,
                        longitude = location.longitude
                    ),
                    idempotencyKey = UUID.randomUUID().toString()
                )
                _state.update {
                    it.copy(isCompleting = false, completed = true, distanceM = resp.distanceM)
                }
            } catch (e: Exception) {
                val msg = e.message ?: "Failed to collect cash"
                _state.update { it.copy(isCompleting = false, error = msg) }
            }
        }
    }
}
