package com.pegasusx.supplier.ui.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.supplier.data.push.DeviceTokenRegistrar
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.TokenHolder
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.JsonPrimitive
import kotlinx.serialization.json.buildJsonObject
import kotlinx.serialization.json.contentOrNull
import kotlinx.serialization.json.jsonObject
import kotlinx.serialization.json.jsonPrimitive

data class RegisterUiState(
    val step: Int = 0,
    val phone: String = "",
    val otpCode: String = "",
    val legalName: String = "",
    val contactName: String = "",
    val email: String = "",
    val password: String = "",
    val countryCode: String = "UZ",
    val loading: Boolean = false,
    val error: String? = null,
)

data class BusinessSetupUiState(
    val taxId: String = "",
    val registrationNumber: String = "",
    val headquartersAddress: String = "",
    val city: String = "",
    val postalCode: String = "",
    val loading: Boolean = false,
    val error: String? = null,
)

@HiltViewModel
class OnboardingViewModel @Inject constructor(
    private val api: SupplierApi,
) : ViewModel() {
    private val _registerState = MutableStateFlow(RegisterUiState())
    val registerState: StateFlow<RegisterUiState> = _registerState.asStateFlow()

    private val _businessState = MutableStateFlow(BusinessSetupUiState())
    val businessState: StateFlow<BusinessSetupUiState> = _businessState.asStateFlow()

    fun updateRegister(block: RegisterUiState.() -> RegisterUiState) {
        _registerState.update(block)
    }

    fun updateBusiness(block: BusinessSetupUiState.() -> BusinessSetupUiState) {
        _businessState.update(block)
    }

    fun nextRegisterStep() {
        _registerState.update { it.copy(step = (it.step + 1).coerceAtMost(2), error = null) }
    }

    fun prevRegisterStep() {
        _registerState.update { it.copy(step = (it.step - 1).coerceAtLeast(0), error = null) }
    }

    fun submitRegister(onSuccess: () -> Unit) {
        val state = _registerState.value
        viewModelScope.launch {
            _registerState.update { it.copy(loading = true, error = null) }
            try {
                val payload = buildJsonObject {
                    put("phone", JsonPrimitive(state.phone.trim()))
                    put(
                        "account",
                        buildJsonObject {
                            put("legalName", JsonPrimitive(state.legalName.trim()))
                            put("contactName", JsonPrimitive(state.contactName.trim()))
                            put("email", JsonPrimitive(state.email.trim()))
                            put("country", JsonPrimitive(state.countryCode))
                            put("phone", JsonPrimitive(state.phone.trim()))
                            put("password", JsonPrimitive(state.password))
                        },
                    )
                    put("id_token", JsonPrimitive(state.otpCode.ifBlank { "otp-placeholder" }))
                }
                val resp = api.register(payload)
                if (resp.isSuccessful) {
                    val body = resp.body()
                    val token = body?.let { el ->
                        el.jsonObject["token"]?.let { t -> t.jsonPrimitive.contentOrNull }
                    }
                    if (!token.isNullOrBlank()) {
                        TokenHolder.token = token
                        TokenHolder.supplierId = body.jsonObject["supplier_id"]?.jsonPrimitive?.content
                        TokenHolder.isConfigured = false
                        DeviceTokenRegistrar.uploadBestEffort(api)
                    }
                    onSuccess()
                } else {
                    _registerState.update { it.copy(error = "Registration failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _registerState.update { it.copy(error = e.message ?: "Network error") }
            } finally {
                _registerState.update { it.copy(loading = false) }
            }
        }
    }

    fun submitBusinessSetup(onSuccess: () -> Unit) {
        val state = _businessState.value
        viewModelScope.launch {
            _businessState.update { it.copy(loading = true, error = null) }
            try {
                val payload = buildJsonObject {
                    put("taxId", JsonPrimitive(state.taxId.trim()))
                    put("registrationNumber", JsonPrimitive(state.registrationNumber.trim()))
                    put("headquartersAddress", JsonPrimitive(state.headquartersAddress.trim()))
                    put("city", JsonPrimitive(state.city.trim()))
                    put("postalCode", JsonPrimitive(state.postalCode.trim()))
                }
                val resp = api.setupBusiness(payload)
                if (resp.isSuccessful) {
                    onSuccess()
                } else {
                    _businessState.update { it.copy(error = "Setup failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _businessState.update { it.copy(error = e.message ?: "Network error") }
            } finally {
                _businessState.update { it.copy(loading = false) }
            }
        }
    }
}
