package com.pegasusx.retailer.ui.screens.setup

import androidx.lifecycle.ViewModel
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.local.RetailerUserIdProvider
import com.pegasusx.retailer.data.model.RetailerInnLookupRequest
import com.pegasusx.retailer.data.model.RetailerInnLookupResponse
import com.pegasusx.retailer.data.model.RetailerJoinRequestConflict
import com.pegasusx.retailer.util.RetailerIdempotencyKeys
import com.pegasusx.retailer.util.RetailerW3ContractHelper
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import javax.inject.Inject

data class SetupWizardUiState(
    val taxId: String = "",
    val isInformalShop: Boolean = false,
    val innChecking: Boolean = false,
    val innLookup: RetailerInnLookupResponse? = null,
    val innError: String? = null,
    val joinRequestConflict: RetailerJoinRequestConflict? = null,
    val joinRequestSent: Boolean = false,
    val billingAddress: String = "",
    val shippingAddress: String = "",
    val city: String = "",
    val postalCode: String = "",
    val isSubmitting: Boolean = false,
    val submitError: String? = null,
    val setupCompleted: Boolean = false,
)

@HiltViewModel
class SetupWizardViewModel @Inject constructor(
    private val api: PegasusApi,
    private val userIdProvider: RetailerUserIdProvider,
) : ViewModel() {

    private val _uiState = MutableStateFlow(SetupWizardUiState())
    val uiState: StateFlow<SetupWizardUiState> = _uiState.asStateFlow()

    fun setTaxId(value: String) {
        val cleaned = RetailerW3ContractHelper.normalizedInnOrNull(value)
        val existing = _uiState.value.innLookup
        val keepLookup = if (existing != null && existing.normalizedInn == cleaned) existing else null
        _uiState.value = _uiState.value.copy(
            taxId = value,
            innLookup = keepLookup,
            innError = null,
            submitError = null,
        )
    }

    fun setInformalShop(informal: Boolean) {
        _uiState.value = _uiState.value.copy(
            isInformalShop = informal,
            innLookup = if (informal) null else _uiState.value.innLookup,
            innError = null,
            submitError = null,
        )
    }

    fun setBillingAddress(value: String) {
        _uiState.value = _uiState.value.copy(billingAddress = value, submitError = null)
    }

    fun setShippingAddress(value: String) {
        _uiState.value = _uiState.value.copy(shippingAddress = value, submitError = null)
    }

    fun setCity(value: String) {
        _uiState.value = _uiState.value.copy(city = value, submitError = null)
    }

    fun setPostalCode(value: String) {
        _uiState.value = _uiState.value.copy(postalCode = value, submitError = null)
    }

    fun clearJoinRequestState() {
        _uiState.value = _uiState.value.copy(
            joinRequestSent = false,
            joinRequestConflict = null,
            submitError = null,
        )
    }

    suspend fun verifyInn(): Boolean {
        val state = _uiState.value
        val cleanInn = RetailerW3ContractHelper.normalizedInnOrNull(state.taxId)
        if (cleanInn == null) {
            _uiState.value = state.copy(
                innError = "INN must be a valid 9-digit Uzbekistan tax identifier.",
                innLookup = null,
            )
            return false
        }

        _uiState.value = state.copy(
            innChecking = true,
            innError = null,
            submitError = null,
            joinRequestSent = false,
            joinRequestConflict = null,
        )

        return try {
            val response = api.lookupInn(RetailerInnLookupRequest(inn = cleanInn, tenantType = "RETAILER"))
            val err = when {
                response.isUnavailable -> "Tax registry lookup is currently unavailable. Please retry in a few moments."
                response.isInvalidOrMissing -> "INN must be a valid 9-digit Uzbekistan tax identifier."
                !response.isVerifiedFound -> "Tax registry lookup is currently unavailable. Please retry in a few moments."
                else -> null
            }
            _uiState.value = _uiState.value.copy(
                innChecking = false,
                innLookup = response,
                innError = err,
            )
            response.isVerifiedFound
        } catch (_: Throwable) {
            _uiState.value = _uiState.value.copy(
                innChecking = false,
                innLookup = RetailerInnLookupResponse(status = "UNAVAILABLE", normalizedInn = cleanInn),
                innError = "Tax registry lookup is currently unavailable. Please retry in a few moments.",
            )
            false
        }
    }

    suspend fun canAdvanceFromTax(): Boolean {
        val state = _uiState.value
        if (state.isInformalShop) {
            _uiState.value = state.copy(innError = null, submitError = null)
            return true
        }
        val cleanInn = RetailerW3ContractHelper.normalizedInnOrNull(state.taxId)
        if (cleanInn == null) {
            _uiState.value = state.copy(innError = "INN must be a valid 9-digit Uzbekistan tax identifier.")
            return false
        }
        val existing = state.innLookup
        if (existing != null && existing.normalizedInn == cleanInn && existing.isVerifiedFound) {
            return true
        }
        return verifyInn()
    }

    suspend fun submitSetup(): Boolean {
        val current = _uiState.value
        if (current.billingAddress.trim().length < 5) {
            _uiState.value = current.copy(submitError = "Billing address is required")
            return false
        }
        if (current.shippingAddress.trim().length < 5) {
            _uiState.value = current.copy(submitError = "Shipping address is required")
            return false
        }
        if (current.city.trim().length < 2) {
            _uiState.value = current.copy(submitError = "City is required")
            return false
        }

        val cleanInn: String
        if (current.isInformalShop) {
            cleanInn = ""
        } else {
            val parsed = RetailerW3ContractHelper.normalizedInnOrNull(current.taxId)
            if (parsed == null) {
                _uiState.value = current.copy(
                    innError = "INN must be a valid 9-digit Uzbekistan tax identifier.",
                    submitError = "INN must be a valid 9-digit Uzbekistan tax identifier.",
                )
                return false
            }
            val verifiedNow = (current.innLookup?.normalizedInn == parsed && current.innLookup.isVerifiedFound) || verifyInn()
            if (!verifiedNow) {
                _uiState.value = _uiState.value.copy(
                    submitError = _uiState.value.innError
                        ?: "Tax registry lookup is currently unavailable. Please retry in a few moments.",
                )
                return false
            }
            cleanInn = parsed
        }

        val stateAfterInn = _uiState.value
        _uiState.value = stateAfterInn.copy(
            isSubmitting = true,
            submitError = null,
            joinRequestConflict = null,
            joinRequestSent = false,
        )

        val retailerId = userIdProvider.getUserId().orEmpty().ifBlank { "default" }
        val payload = buildMap<String, Any> {
            put("tax_id", cleanInn)
            put("informal_shop", stateAfterInn.isInformalShop)
            put("legal_name", stateAfterInn.innLookup?.legalName.orEmpty())
            put("billing_address", stateAfterInn.billingAddress.trim())
            put("shipping_address", stateAfterInn.shippingAddress.trim())
            put("city", stateAfterInn.city.trim())
            put("postal_code", stateAfterInn.postalCode.trim())
        }

        return try {
            api.setupRetailer(
                body = payload,
                idempotencyKey = RetailerIdempotencyKeys.setup(retailerId),
            )
            _uiState.value = _uiState.value.copy(
                isSubmitting = false,
                setupCompleted = true,
            )
            true
        } catch (t: Throwable) {
            val conflict = RetailerW3ContractHelper.parseJoinRequestConflict(t)
            if (conflict != null) {
                _uiState.value = _uiState.value.copy(
                    isSubmitting = false,
                    joinRequestConflict = conflict,
                    joinRequestSent = true,
                    submitError = null,
                    setupCompleted = false,
                )
            } else {
                _uiState.value = _uiState.value.copy(
                    isSubmitting = false,
                    submitError = RetailerW3ContractHelper.parseArErrorMessage(t, "Setup failed. Please try again."),
                )
            }
            false
        }
    }
}
