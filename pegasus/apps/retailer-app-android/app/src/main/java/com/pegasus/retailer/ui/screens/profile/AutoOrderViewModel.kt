package com.pegasus.retailer.ui.screens.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.PegasusApi
import com.pegasus.retailer.data.model.AutoOrderSettings
import com.pegasus.retailer.data.model.UpdateGlobalSettingsRequest
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class AutoOrderUiState(
    val isLoading: Boolean = true,
    val settings: AutoOrderSettings? = null,
    val error: String? = null
)

@HiltViewModel
class AutoOrderViewModel @Inject constructor(
    private val api: PegasusApi
) : ViewModel() {

    private val _uiState = MutableStateFlow(AutoOrderUiState())
    val uiState: StateFlow<AutoOrderUiState> = _uiState.asStateFlow()

    init {
        loadSettings()
    }

    fun loadSettings() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            try {
                val settings = api.getAutoOrderSettings()
                _uiState.update { it.copy(isLoading = false, settings = settings) }
            } catch (e: Exception) {
                _uiState.update { it.copy(isLoading = false, error = e.message) }
            }
        }
    }

    fun toggleGlobalEnabled(enabled: Boolean) {
        viewModelScope.launch {
            try {
                val request = UpdateGlobalSettingsRequest(globalAutoOrderEnabled = enabled, useHistory = true)
                api.updateGlobalAutoOrder(request)
                
                // Optimistically update
                _uiState.update { state -> 
                    state.copy(settings = state.settings?.copy(globalEnabled = enabled))
                }
            } catch (e: Exception) {
                _uiState.update { it.copy(error = e.message) }
                loadSettings() // rollback on error
            }
        }
    }
}
