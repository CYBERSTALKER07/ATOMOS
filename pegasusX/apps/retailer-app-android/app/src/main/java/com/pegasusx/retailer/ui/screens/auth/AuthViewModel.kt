package com.pegasusx.retailer.ui.screens.auth

import android.content.Context
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.pegasusx.retailer.data.api.PegasusApi
import com.pegasusx.retailer.data.local.TokenManager
import com.pegasusx.retailer.data.model.LoginRequest
import com.pegasusx.retailer.data.model.RegisterRequest
import com.pegasusx.retailer.util.RetailerIdempotencyKeys
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import java.io.IOException
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import retrofit2.HttpException
import javax.inject.Inject

enum class AuthLoadIssue {
    RESTRICTED,
    OFFLINE,
    ERROR,
}

@HiltViewModel
class AuthViewModel @Inject constructor(
    private val api: PegasusApi,
    private val tokenManager: TokenManager,
    @ApplicationContext private val context: Context,
) : ViewModel() {

    private val _uiState = MutableStateFlow(AuthUiState())
    val uiState = _uiState.asStateFlow()

    val isAuthenticated: Boolean get() = tokenManager.getToken() != null

    // ── Login ──

    fun login(phone: String, password: String) {
        val formatted = formatUzPhone(phone)
        if (formatted == null) {
            _uiState.value = _uiState.value.copy(error = "Invalid number. Use +998 XX XXX XX XX.", loadIssue = AuthLoadIssue.ERROR)
            return
        }
        if (password.length < 4) {
            _uiState.value = _uiState.value.copy(error = "Password too short.", loadIssue = AuthLoadIssue.ERROR)
            return
        }
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(isLoading = true, error = null, loadIssue = null)
            try {
                val response = api.login(LoginRequest(phoneNumber = formatted, password = password))
                completeAuth(response)
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.value = _uiState.value.copy(
                    isLoading = false,
                    error = resolveErrorMessage(e, issue, fallback = "Login failed"),
                    loadIssue = issue,
                )
            }
        }
    }

    fun loginWithOtp(idToken: String) {
        if (idToken.isBlank()) {
            _uiState.value = _uiState.value.copy(error = "Verification required.", loadIssue = AuthLoadIssue.ERROR)
            return
        }
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(isLoading = true, error = null, loadIssue = null)
            try {
                val response = api.login(LoginRequest(idToken = idToken))
                completeAuth(response)
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.value = _uiState.value.copy(
                    isLoading = false,
                    error = resolveErrorMessage(e, issue, fallback = "OTP login failed"),
                    loadIssue = issue,
                )
            }
        }
    }

    private suspend fun completeAuth(
        response: com.pegasusx.retailer.data.model.AuthResponse,
        clearScoped: Boolean = false,
    ) {
        if (response.isPendingOrgSelect) {
            tokenManager.saveToken(response.token)
            _uiState.value = _uiState.value.copy(
                isLoading = false,
                isAuthenticated = false,
                needsOrgSelect = true,
                pendingMemberships = response.memberships.filter { it.isActive },
                error = null,
                loadIssue = null,
            )
            return
        }
        if (clearScoped) {
            clearOrgScopedState()
        }
        tokenManager.saveToken(response.token)
        val user = response.user
        if (user != null) {
            tokenManager.saveUserId(user.id)
            tokenManager.saveUserName(user.name)
        }
        if (response.firebaseToken.isNotBlank()) {
            val fbIdToken = com.pegasusx.retailer.data.auth.FirebaseAuthHelper.exchangeCustomToken(response.firebaseToken)
            if (fbIdToken != null) tokenManager.saveFirebaseIdToken(fbIdToken)
        }
        _uiState.value = _uiState.value.copy(
            isLoading = false,
            isAuthenticated = true,
            needsOrgSelect = false,
            pendingMemberships = emptyList(),
            error = null,
            loadIssue = null,
        )
    }

    /** C1.3 hard contract: cart, POS session, offline count drafts, assist context. */
    fun clearOrgScopedState() {
        val prefs = context.getSharedPreferences("retailer_org_scoped", Context.MODE_PRIVATE)
        prefs.edit().clear().apply()
        val keys = listOf(
            "retailer_cart",
            "retailer_pos_parked_cart_v1",
            "retailer_pending_pos_sales_v1",
            "retailer_stock_count_draft_v1",
            "retailer_assist_context_v1",
            "retailer_pos_session_v1",
        )
        val appPrefs = context.getSharedPreferences("retailer_prefs", Context.MODE_PRIVATE)
        appPrefs.edit().apply {
            keys.forEach { remove(it) }
            apply()
        }
    }

    fun selectOrg(retailerId: String) {
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(isLoading = true, error = null, loadIssue = null)
            try {
                val response = api.selectOrg(com.pegasusx.retailer.data.model.SelectOrgRequest(retailerId))
                completeAuth(response, clearScoped = true)
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.value = _uiState.value.copy(
                    isLoading = false,
                    error = resolveErrorMessage(e, issue, fallback = "Select organization failed"),
                    loadIssue = issue,
                )
            }
        }
    }

    fun switchOrg(retailerId: String) {
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(isLoading = true, error = null, loadIssue = null)
            try {
                val response = api.switchOrg(com.pegasusx.retailer.data.model.SelectOrgRequest(retailerId))
                completeAuth(response, clearScoped = true)
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.value = _uiState.value.copy(
                    isLoading = false,
                    error = resolveErrorMessage(e, issue, fallback = "Switch organization failed"),
                    loadIssue = issue,
                )
            }
        }
    }

    suspend fun resolveAddress(lat: Double, lng: Double): String? =
        runCatching { api.reverseGeocode(lat, lng).address.takeIf { it.isNotBlank() } }.getOrNull()

    // ── Register ──

    fun register(
        phone: String,
        password: String,
        storeName: String,
        ownerName: String,
        addressText: String,
        taxId: String?,
        latitude: Double = 0.0,
        longitude: Double = 0.0,
        receivingWindowOpen: String? = null,
        receivingWindowClose: String? = null,
        accessType: String? = null,
        storageCeilingHeightCM: Double? = null,
    ) {
        val formatted = formatUzPhone(phone)
        if (formatted == null) {
            _uiState.value = _uiState.value.copy(error = "Invalid number. Use +998 XX XXX XX XX.", loadIssue = AuthLoadIssue.ERROR)
            return
        }
        if (password.length < 4) {
            _uiState.value = _uiState.value.copy(error = "Password too short.", loadIssue = AuthLoadIssue.ERROR)
            return
        }
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(isLoading = true, error = null, loadIssue = null)
            try {
                val response = api.register(RegisterRequest(
                    phoneNumber = formatted, password = password,
                    storeName = storeName, ownerName = ownerName,
                    addressText = addressText,
                    deliveryAddress = addressText.takeIf { it.isNotBlank() },
                    latitude = latitude, longitude = longitude,
                    taxId = taxId?.takeIf { it.isNotBlank() },
                    receivingWindowOpen = receivingWindowOpen?.takeIf { it.isNotBlank() },
                    receivingWindowClose = receivingWindowClose?.takeIf { it.isNotBlank() },
                    accessType = accessType?.takeIf { it.isNotBlank() },
                    storageCeilingHeightCM = storageCeilingHeightCM,
                ))
                completeAuth(response, clearScoped = false)
                val userId = response.user?.id ?: response.retailerId
                try {
                    if (userId.isNotBlank()) {
                        api.setupRetailer(
                            body = mapOf(
                                "store_name" to storeName,
                                "owner_name" to ownerName,
                                "address_text" to addressText,
                                "latitude" to latitude,
                                "longitude" to longitude,
                            ),
                            idempotencyKey = RetailerIdempotencyKeys.setup(userId),
                        )
                    }
                } catch (_: Exception) {
                    // Registration already captured core profile; setup is additive best-effort.
                }
            } catch (e: Exception) {
                val issue = resolveLoadIssue(e)
                _uiState.value = _uiState.value.copy(
                    isLoading = false,
                    error = resolveErrorMessage(e, issue, fallback = "Registration failed"),
                    loadIssue = issue,
                )
            }
        }
    }

    fun logout() {
        tokenManager.clearToken()
        com.pegasusx.retailer.data.auth.FirebaseAuthHelper.signOut()
        _uiState.value = AuthUiState()
    }

    fun clearError() {
        _uiState.value = _uiState.value.copy(error = null, loadIssue = null)
    }

    private fun resolveLoadIssue(error: Exception): AuthLoadIssue {
        return when {
            error is HttpException && (error.code() == 401 || error.code() == 403) -> AuthLoadIssue.RESTRICTED
            error is IOException -> AuthLoadIssue.OFFLINE
            else -> AuthLoadIssue.ERROR
        }
    }

    private fun resolveErrorMessage(error: Exception, issue: AuthLoadIssue, fallback: String): String {
        return when (issue) {
            AuthLoadIssue.RESTRICTED -> "Access is restricted for this account"
            AuthLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            AuthLoadIssue.ERROR -> error.message ?: fallback
        }
    }

    private fun formatUzPhone(raw: String): String? {
        val digits = raw.replace("[^0-9]".toRegex(), "")
        val phone = when {
            digits.startsWith("998") && digits.length == 12 -> "+$digits"
            digits.length == 9 -> "+998$digits"
            raw.startsWith("+998") && raw.length == 13 -> raw
            else -> null
        }
        return phone?.takeIf { it.matches(Regex("^\\+998\\d{9}$")) }
    }
}

data class AuthUiState(
    val needsOrgSelect: Boolean = false,
    val pendingMemberships: List<com.pegasusx.retailer.data.model.RetailerMembershipDTO> = emptyList(),
    val isLoading: Boolean = false,
    val isAuthenticated: Boolean = false,
    val error: String? = null,
    val loadIssue: AuthLoadIssue? = null,
) {
    val syncMessage: String?
        get() = when (loadIssue) {
            AuthLoadIssue.RESTRICTED -> "Access is restricted for this account"
            AuthLoadIssue.OFFLINE -> "Offline mode active. Reconnect and retry"
            AuthLoadIssue.ERROR -> "Authentication request failed"
            null -> null
        }
}
