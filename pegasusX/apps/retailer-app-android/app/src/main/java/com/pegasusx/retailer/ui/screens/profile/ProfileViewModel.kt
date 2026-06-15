package com.pegasusx.retailer.ui.screens.profile

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.model.UpdateGlobalSettingsRequest
import dagger.hilt.android.lifecycle.HiltViewModel
import java.io.IOException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.boolean
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive
import retrofit2.HttpException
import javax.inject.Inject

enum class ProfileLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

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
    val isLoading: Boolean = false,
    val showHistoryDialog: Boolean = false,
    val error: String? = null,
    val loadIssue: ProfileLoadIssue? = null,
    val pricingRulesSummary: String? = null,
    val clientPolicyMessage: String? = null,
) {
    val syncMessage: String?
        get() = when (loadIssue) {
            ProfileLoadIssue.RESTRICTED -> "Profile access is restricted for this account"
            ProfileLoadIssue.OFFLINE -> "Offline mode active. Showing latest cached profile"
            ProfileLoadIssue.ERROR -> "Profile sync degraded. Retry is available"
            null -> null
        }
}

@HiltViewModel
class ProfileViewModel @Inject constructor(
    private val api: PegasusApi,
    private val tokenManager: com.pegasusx.retailer.data.local.TokenManager,
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
        refresh()
    }

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }

            var nextIssue: ProfileLoadIssue? = null
            var nextError: String? = null

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
            } catch (e: Exception) {
                nextIssue = resolveLoadIssue(e)
                nextError = resolveErrorMessage(e, nextIssue)
            }

            val rid = tokenManager.getUserId()
            if (rid.isNullOrBlank()) {
                if (nextIssue == null) {
                    nextIssue = ProfileLoadIssue.ERROR
                    nextError = "Retailer identity unavailable for stats sync"
                }
            } else {
                try {
                    val orders = api.getOrders(rid)
                    _uiState.update {
                        it.copy(
                            orderCount = orders.size,
                            totalSpent = orders.sumOf { o -> o.totalAmount.toLong() },
                        )
                    }
                } catch (e: Exception) {
                    if (nextIssue == null) {
                        nextIssue = resolveLoadIssue(e)
                        nextError = resolveErrorMessage(e, nextIssue)
                    }
                }
            }

            try {
                val rules = api.getPricingRules()
                val summary = rules.jsonObject["summary"]?.jsonPrimitive?.content
                    ?: rules.jsonObject["status"]?.jsonPrimitive?.content
                if (!summary.isNullOrBlank()) {
                    _uiState.update { it.copy(pricingRulesSummary = summary) }
                }
            } catch (_: Exception) {
                // Pricing rules are read-only and optional on partial stacks.
            }

            try {
                val policy = api.getClientPolicy(
                    platform = "android",
                    version = com.pegasusx.retailer.BuildConfig.VERSION_NAME,
                )
                val outdated = policy.jsonObject["outdated"]?.jsonPrimitive?.boolean == true
                val force = policy.jsonObject["force_update"]?.jsonPrimitive?.boolean == true
                val minimum = policy.jsonObject["minimum_version"]?.jsonPrimitive?.content
                if (outdated || force) {
                    _uiState.update {
                        it.copy(
                            clientPolicyMessage = buildString {
                                append(if (force) "Update required" else "Update available")
                                if (!minimum.isNullOrBlank()) append(" — minimum version $minimum")
                            },
                        )
                    }
                }
            } catch (_: Exception) {
                // Client policy is best-effort on app launch.
            }

            _uiState.update {
                it.copy(
                    isLoading = false,
                    loadIssue = nextIssue,
                    error = nextError,
                )
            }
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
                _uiState.update { it.copy(isUpdatingSettings = false, error = null, loadIssue = null) }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        globalAutoOrderEnabled = previous,
                        isUpdatingSettings = false,
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun confirmEnableGlobal(useHistory: Boolean) {
        _uiState.update { it.copy(showHistoryDialog = false, globalAutoOrderEnabled = true, isUpdatingSettings = true) }
        viewModelScope.launch {
            try {
                api.updateGlobalAutoOrder(UpdateGlobalSettingsRequest(globalAutoOrderEnabled = true, useHistory = useHistory))
                _uiState.update { it.copy(isUpdatingSettings = false, error = null, loadIssue = null) }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        globalAutoOrderEnabled = false,
                        isUpdatingSettings = false,
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    fun dismissHistoryDialog() {
        _uiState.update { it.copy(showHistoryDialog = false) }
    }

    fun saveProfile(name: String, phone: String, company: String) {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }
            try {
                val payload = buildMap {
                    val trimmedName = name.trim()
                    val trimmedPhone = phone.trim()
                    val trimmedCompany = company.trim()
                    if (trimmedName.isNotEmpty()) put("name", trimmedName)
                    if (trimmedPhone.isNotEmpty()) put("phone", trimmedPhone)
                    if (trimmedCompany.isNotEmpty()) put("company", trimmedCompany)
                }
                if (payload.isEmpty()) {
                    _uiState.update { it.copy(isLoading = false, error = "No profile changes to save") }
                    return@launch
                }
                api.updateRetailerProfile(payload)
                _uiState.update {
                    it.copy(
                        retailerName = name.trim().ifEmpty { it.retailerName },
                        phone = phone.trim().ifEmpty { it.phone },
                        company = company.trim().ifEmpty { it.company },
                        isLoading = false,
                        loadIssue = null,
                        error = null,
                    )
                }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        error = resolveErrorMessage(e, issue),
                        loadIssue = issue,
                    )
                }
            }
        }
    }

    private fun resolveLoadIssue(error: Exception): ProfileLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> ProfileLoadIssue.RESTRICTED
            error is IOException -> ProfileLoadIssue.OFFLINE
            else -> ProfileLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: ProfileLoadIssue): String {
        return when (issue) {
            ProfileLoadIssue.RESTRICTED -> "Profile access is restricted for this account"
            ProfileLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            ProfileLoadIssue.ERROR -> error.message ?: "Profile request failed"
        }
    }
}
