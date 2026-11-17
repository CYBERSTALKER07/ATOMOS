package com.pegasusx.supplier.ui.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.supplier.data.model.INNLookupRequest
import com.pegasusx.supplier.data.model.INNLookupResponse
import com.pegasusx.supplier.data.model.JoinRequestConflictPayload
import com.pegasusx.supplier.data.push.DeviceTokenRegistrar
import com.pegasusx.supplier.data.remote.SupplierApi
import com.pegasusx.supplier.data.remote.TokenHolder
import com.pegasusx.supplier.util.SupplierIdempotencyKeys
import dagger.hilt.android.lifecycle.HiltViewModel
import javax.inject.Inject
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.serialization.json.Json
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
    val innChecking: Boolean = false,
    val innLookup: INNLookupResponse? = null,
    val innError: String? = null,
    val joinRequestConflict: JoinRequestConflictPayload? = null,
    val joinRequestSent: Boolean = false,
) {
    val normalizedInn: String get() = taxId.filter { it.isDigit() }
}

@HiltViewModel
class OnboardingViewModel @Inject constructor(
    private val api: SupplierApi,
    private val json: Json = Json { ignoreUnknownKeys = true },
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

    fun verifyINN() {
        val digits = _businessState.value.normalizedInn
        if (digits.length != 9) {
            _businessState.update {
                it.copy(
                    innLookup = null,
                    innError = "INN must be 9 digits",
                )
            }
            return
        }
        viewModelScope.launch {
            performVerifyINN(digits)
        }
    }

    private suspend fun performVerifyINN(digits: String): INNLookupResponse? {
        _businessState.update {
            it.copy(innChecking = true, innError = null, joinRequestConflict = null, joinRequestSent = false)
        }
        return try {
            val resp = api.lookupINN(INNLookupRequest(inn = digits, tenantType = "SUPPLIER"))
            val body = resp.body()
            if (!resp.isSuccessful || body == null) {
                _businessState.update {
                    it.copy(innChecking = false, innError = "Tax registry verification unavailable (${resp.code()})")
                }
                return null
            }
            val status = body.status.uppercase()
            val errMsg = when (status) {
                "UNAVAILABLE" -> "Tax registry verification is temporarily unavailable. Registration cannot proceed."
                "INVALID", "NOT_FOUND" -> "INN is invalid or not found in official tax registry."
                else -> null
            }
            _businessState.update { cur ->
                val filledAddr = if (cur.headquartersAddress.isBlank() && !body.address.isNullOrBlank()) {
                    body.address
                } else {
                    cur.headquartersAddress
                }
                cur.copy(
                    innChecking = false,
                    innLookup = body,
                    innError = errMsg,
                    headquartersAddress = filledAddr,
                )
            }
            body
        } catch (e: Exception) {
            _businessState.update {
                it.copy(innChecking = false, innError = e.message ?: "Tax registry verification unavailable")
            }
            null
        }
    }

    fun submitBusinessSetup(onSuccess: () -> Unit) {
        val initial = _businessState.value
        viewModelScope.launch {
            _businessState.update { it.copy(loading = true, error = null) }
            try {
                val digits = initial.normalizedInn
                if (digits.length == 9) {
                    val existing = _businessState.value.innLookup
                    val verified = if (existing == null || existing.normalizedInn != digits) {
                        performVerifyINN(digits)
                    } else {
                        existing
                    }
                    if (verified == null) {
                        _businessState.update {
                            it.copy(loading = false, error = it.innError ?: "INN verification unavailable")
                        }
                        return@launch
                    }
                    when (verified.status.uppercase()) {
                        "UNAVAILABLE", "INVALID", "NOT_FOUND" -> {
                            _businessState.update {
                                it.copy(loading = false, error = it.innError ?: "INN verification failed")
                            }
                            return@launch
                        }
                    }
                }

                val current = _businessState.value
                val payload = buildJsonObject {
                    put("taxId", JsonPrimitive(current.taxId.trim()))
                    put("registrationNumber", JsonPrimitive(current.registrationNumber.trim()))
                    put("headquartersAddress", JsonPrimitive(current.headquartersAddress.trim()))
                    put("city", JsonPrimitive(current.city.trim()))
                    put("postalCode", JsonPrimitive(current.postalCode.trim()))
                }
                val scopeId = SupplierIdempotencyKeys.supplierScopeId()
                val key = SupplierIdempotencyKeys.businessSetup(scopeId, payload.toString())
                val resp = api.setupBusiness(payload, key)
                if (resp.isSuccessful) {
                    _businessState.update { it.copy(loading = false) }
                    onSuccess()
                } else if (resp.code() == 409) {
                    val errText = resp.errorBody()?.string()
                    val conflict = errText?.let { raw ->
                        runCatching { json.decodeFromString<JoinRequestConflictPayload>(raw) }.getOrNull()
                    }
                    _businessState.update {
                        it.copy(
                            loading = false,
                            joinRequestConflict = conflict,
                            joinRequestSent = true,
                            error = null,
                        )
                    }
                } else {
                    _businessState.update { it.copy(loading = false, error = "Setup failed (${resp.code()})") }
                }
            } catch (e: Exception) {
                _businessState.update { it.copy(loading = false, error = e.message ?: "Network error") }
            }
        }
    }
}
