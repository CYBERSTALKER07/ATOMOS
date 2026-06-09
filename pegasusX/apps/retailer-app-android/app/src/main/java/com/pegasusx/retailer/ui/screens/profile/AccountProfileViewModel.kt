package com.pegasusx.retailer.ui.screens.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import retrofit2.HttpException

data class AccountProfileUiState(
    val name: String = "",
    val company: String = "",
    val phone: String = "",
    val regionId: String = "",
    val receivingWindowOpen: String = "",
    val receivingWindowClose: String = "",
    val isLoading: Boolean = false,
    val isSaving: Boolean = false,
    val error: String? = null,
    val saveMessage: String? = null,
    val openWindowError: String? = null,
    val closeWindowError: String? = null,
)

@HiltViewModel
class AccountProfileViewModel @Inject constructor(
    private val api: PegasusApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(AccountProfileUiState())
    val uiState: StateFlow<AccountProfileUiState> = _uiState.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null, saveMessage = null) }
            try {
                val profile = api.getRetailerProfile()
                _uiState.update {
                    it.copy(
                        name = profile["name"].orEmpty(),
                        company = profile["company"].orEmpty(),
                        phone = profile["phone"].orEmpty(),
                        regionId = profile["region_id"].orEmpty(),
                        receivingWindowOpen = profile["receiving_window_open"].orEmpty(),
                        receivingWindowClose = profile["receiving_window_close"].orEmpty(),
                        isLoading = false,
                        error = null,
                    )
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        error = resolveErrorMessage(e),
                    )
                }
            }
        }
    }

    fun onNameChanged(value: String) {
        _uiState.update { it.copy(name = value) }
    }

    fun onCompanyChanged(value: String) {
        _uiState.update { it.copy(company = value) }
    }

    fun onRegionIdChanged(value: String) {
        _uiState.update { it.copy(regionId = value) }
    }

    fun onReceivingWindowOpenChanged(value: String) {
        _uiState.update { it.copy(receivingWindowOpen = value, openWindowError = null) }
    }

    fun onReceivingWindowCloseChanged(value: String) {
        _uiState.update { it.copy(receivingWindowClose = value, closeWindowError = null) }
    }

    fun save() {
        val state = _uiState.value
        val openError = ReceivingWindowValidator.validate(state.receivingWindowOpen)
        val closeError = ReceivingWindowValidator.validate(state.receivingWindowClose)
        if (openError != null || closeError != null) {
            _uiState.update {
                it.copy(openWindowError = openError, closeWindowError = closeError)
            }
            return
        }

        viewModelScope.launch {
            _uiState.update {
                it.copy(
                    isSaving = true,
                    error = null,
                    saveMessage = null,
                    openWindowError = null,
                    closeWindowError = null,
                )
            }
            try {
                val payload = buildMap {
                    val trimmedName = state.name.trim()
                    val trimmedCompany = state.company.trim()
                    val trimmedRegionId = state.regionId.trim()
                    val open = ReceivingWindowValidator.normalize(state.receivingWindowOpen)
                    val close = ReceivingWindowValidator.normalize(state.receivingWindowClose)
                    if (trimmedName.isNotEmpty()) put("name", trimmedName)
                    if (trimmedCompany.isNotEmpty()) put("company", trimmedCompany)
                    if (trimmedRegionId.isNotEmpty()) put("region_id", trimmedRegionId)
                    put("receiving_window_open", open)
                    put("receiving_window_close", close)
                }
                api.updateRetailerProfile(payload)
                _uiState.update {
                    it.copy(
                        isSaving = false,
                        saveMessage = "Profile saved",
                        receivingWindowOpen = ReceivingWindowValidator.normalize(state.receivingWindowOpen),
                        receivingWindowClose = ReceivingWindowValidator.normalize(state.receivingWindowClose),
                    )
                }
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        isSaving = false,
                        error = resolveErrorMessage(e),
                    )
                }
            }
        }
    }

    private fun resolveErrorMessage(error: Exception): String {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) ->
                "Profile access is restricted for this account"
            error is IOException -> "Offline mode active. Reconnect and retry"
            else -> error.message ?: "Profile request failed"
        }
    }
}
