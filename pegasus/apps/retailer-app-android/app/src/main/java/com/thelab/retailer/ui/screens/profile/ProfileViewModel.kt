package com.pegasus.retailer.ui.screens.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.LabApi
import com.pegasus.retailer.data.model.UpdateGlobalSettingsRequest
import com.pegasus.retailer.data.model.UpdateSettingsRequest
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class ProfileUiState(
    val retailerName: String = "",
    val retailerId: String = "",
    val phone: String = "",
    val company: String = "",
    val location: String = "",
    val orderCount: Int = 0,
    val totalSpent: Long = 0,
    val globalAutoOrderEnabled: Boolean = false,
    val isUpdatingSettings: Boolean = false,
    val showHistoryDialog: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class ProfileViewModel @Inject constructor(
    private val api: LabApi,
    private val tokenManager: com.pegasus.retailer.data.local.TokenManager,
) : ViewModel() {

    private val _uiState = MutableStateFlow(ProfileUiState())
    val uiState: StateFlow<ProfileUiState> = _uiState.asStateFlow()

    init {
        _uiState.update {
            it.copy(
                retailerName = tokenManager.getUserName() ?: "",
                retailerId = tokenManager.getUserId() ?: "",
            )
        }
        loadProfile()
        loadStats()
    }

    private fun loadProfile() {
        viewModelScope.launch {
            try {
                val profile = api.getRetailerProfile()
                _uiState.update {
                    it.copy(
                        retailerName = profile["name"] ?: it.retailerName,
                        phone = profile["phone"] ?: "",
                        company = profile["company"] ?: "",
                        location = profile["location"] ?: "",
                    )
                }
            } catch (_: Exception) {}
        }
    }

    private fun loadStats() {
        viewModelScope.launch {
            try {
                val rid = tokenManager.getUserId() ?: return@launch
                val orders = api.getOrders(rid)
                _uiState.update {
                    it.copy(
                        orderCount = orders.size,
                        totalSpent = orders.sumOf { o -> o.totalAmount.toLong() },
                    )
                }
            } catch (_: Exception) {}
        }
    }

    /**
     * When enabling, show the history/fresh dialog first.
     * When disabling, fire immediately.
     */
    fun toggleGlobalAutoOrder(enabled: Boolean) {
        if (enabled) {
            _uiState.update { it.copy(showHistoryDialog = true) }
            return
        }
        val previous = _uiState.value.globalAutoOrderEnabled
        _uiState.update { it.copy(globalAutoOrderEnabled = false, isUpdatingSettings = true) }
        viewModelScope.launch {
            try {
                api.updateGlobalAutoOrder(UpdateGlobalSettingsRequest(globalAutoOrderEnabled = false))
                _uiState.update { it.copy(isUpdatingSettings = false) }
            } catch (e: Exception) {
                _uiState.update { it.copy(globalAutoOrderEnabled = previous, isUpdatingSettings = false, error = e.message) }
            }
        }
    }

    fun confirmEnableGlobal(useHistory: Boolean) {
        _uiState.update { it.copy(showHistoryDialog = false, globalAutoOrderEnabled = true, isUpdatingSettings = true) }
        viewModelScope.launch {
            try {
                api.updateGlobalAutoOrder(UpdateGlobalSettingsRequest(globalAutoOrderEnabled = true, useHistory = useHistory))
                _uiState.update { it.copy(isUpdatingSettings = false) }
            } catch (e: Exception) {
                _uiState.update { it.copy(globalAutoOrderEnabled = false, isUpdatingSettings = false, error = e.message) }
            }
        }
    }

    fun dismissHistoryDialog() {
        _uiState.update { it.copy(showHistoryDialog = false) }
    }
}
