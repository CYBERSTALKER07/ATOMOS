package com.pegasusx.retailer.ui.screens.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.model.CreditProfile
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import retrofit2.HttpException

data class CreditProfileUiState(
    val isLoading: Boolean = true,
    val profile: CreditProfile? = null,
    val missing: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class CreditProfileViewModel @Inject constructor(
    private val api: PegasusApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(CreditProfileUiState())
    val uiState: StateFlow<CreditProfileUiState> = _uiState.asStateFlow()

    init {
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update {
                it.copy(isLoading = true, error = null, missing = false)
            }
            try {
                val profile = api.getCreditProfile()
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        profile = profile,
                        missing = false,
                        error = null,
                    )
                }
            } catch (e: HttpException) {
                if (e.code() == 404) {
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            profile = null,
                            missing = true,
                            error = null,
                        )
                    }
                } else {
                    _uiState.update {
                        it.copy(
                            isLoading = false,
                            profile = null,
                            missing = false,
                            error = "Credit unavailable",
                        )
                    }
                }
            } catch (_: Exception) {
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        profile = null,
                        missing = false,
                        error = "Credit unavailable",
                    )
                }
            }
        }
    }
}
