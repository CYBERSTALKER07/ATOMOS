package com.pegasus.retailer.ui.screens.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasus.retailer.data.api.PegasusApi
import com.pegasus.retailer.data.model.AutoOrderSettings
import com.pegasus.retailer.data.model.UpdateGlobalSettingsRequest
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import retrofit2.HttpException
import javax.inject.Inject

enum class ProfileAutoOrderLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

data class AutoOrderUiState(
    val isLoading: Boolean = true,
    val settings: AutoOrderSettings? = null,
    val error: String? = null,
    val loadIssue: ProfileAutoOrderLoadIssue? = null,
) {
    val syncMessage: String?
        get() = when (loadIssue) {
            ProfileAutoOrderLoadIssue.RESTRICTED -> "Auto-order settings access is restricted for this account"
            ProfileAutoOrderLoadIssue.OFFLINE -> "Offline mode active. Showing latest auto-order settings"
            ProfileAutoOrderLoadIssue.ERROR -> "Auto-order settings sync degraded. Retry is available"
            null -> null
        }
}

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
            _uiState.update { it.copy(isLoading = true, error = null, loadIssue = null) }
            try {
                val settings = api.getAutoOrderSettings()
                _uiState.update { it.copy(isLoading = false, settings = settings, error = null, loadIssue = null) }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        error = resolveErrorMessage(e, issue, "Could not load auto-order settings"),
                        loadIssue = issue,
                    )
                }
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
                    state.copy(settings = state.settings?.copy(globalEnabled = enabled), error = null, loadIssue = null)
                }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update { it.copy(error = resolveErrorMessage(e, issue, "Could not update auto-order settings"), loadIssue = issue) }
                loadSettings() // rollback on error
            }
        }
    }

    private fun resolveLoadIssue(error: Exception): ProfileAutoOrderLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> ProfileAutoOrderLoadIssue.RESTRICTED
            error is IOException -> ProfileAutoOrderLoadIssue.OFFLINE
            else -> ProfileAutoOrderLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: ProfileAutoOrderLoadIssue, fallback: String): String {
        return when (issue) {
            ProfileAutoOrderLoadIssue.RESTRICTED -> "Auto-order settings access is restricted for this account"
            ProfileAutoOrderLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            ProfileAutoOrderLoadIssue.ERROR -> error.message ?: fallback
        }
    }
}
